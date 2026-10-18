"""Self-service allow requests ("votes") for blocked binaries.

Everything that decides what a vote may do is derived here from the stored
device, binary and rules. The client only says which device and binary it
means, so a request can never pick its own scope, rule type or identifier.

Rules created here are always machine rules for the requesting user's own
Mac. A rule is promoted to global once enough distinct users have voted for
it.
"""

MALICIOUS = 2

# Signing statuses whose signing ID can't be used for a SIGNINGID rule.
UNUSABLE_SIGNING_STATUSES = {"SIGNING_STATUS_ADHOC", "SIGNING_STATUS_UNSIGNED", "SIGNING_STATUS_INVALID"}


def username_from_session_user(user):
    """The local username for a signed-in user, matching how devices store
    primary_user (the part of the UPN before the @)."""
    if not user:
        return ""
    principal = user.get("upn") or user.get("email") or ""
    return principal.split("@")[0].strip().lower()


def find_matching_rule(db, binary):
    """The existing rule that applies to the binary, checked from the
    broadest identifier (team ID) to the narrowest (CDHash). Tombstoned
    rules are ignored."""
    for field in ("team_id", "signing_id", "file_sha256", "cdhash"):
        value = binary.get(field)
        if not value:
            continue
        rule = db["rules"].find_one({"_id": value})
        if rule and not rule.get("deleted"):
            return rule
    return None


def rule_target(binary):
    """The (rule_type, identifier) a new allow rule for the binary uses:
    its signing ID when it is properly signed, otherwise its SHA-256."""
    signing_id = binary.get("signing_id")
    if signing_id and binary.get("signingStatus") not in UNUSABLE_SIGNING_STATUSES:
        return "SIGNINGID", signing_id
    if binary.get("file_sha256"):
        return "BINARY", binary["file_sha256"]
    return None


def is_votable(rule):
    """Only machine allow rules collect votes. Global, managed app and
    block rules are left alone."""
    return rule.get("scope") == "machine" and rule.get("policy") == "ALLOWLIST"


def bundle_members(db, binary):
    """Other binaries from the same app bundle. Membership uses the bundle
    hash Santa computes from the bundle's contents, plus the same team ID, so
    an uploaded event can't claim to belong to someone else's app."""
    bundle_hash = binary.get("file_bundle_hash")
    if not bundle_hash:
        return []

    query = {
        "file_bundle_hash": bundle_hash,
        "team_id": binary.get("team_id") or {"$in": [None, ""]},
        "_id": {"$ne": binary.get("_id")},
    }
    return list(db["events"].find(query))


def _refuse(reason):
    return {"allowed": False, "reason": reason}


def allow_decision(device, binary, rule, user):
    """Decide whether the signed-in user may request that the binary be
    allowed on the device. Returns {"allowed": bool, "reason": str} and, when
    allowed, the "action" ("new" or "vote") and the "machine_id"."""
    if not device:
        return _refuse("This Mac is not known to the sync server.")
    if not binary:
        return _refuse("This application is not known to the sync server.")

    username = username_from_session_user(user)
    if not username or not (user or {}).get("id"):
        return _refuse("Sign in again to request access.")

    owner = (device.get("primary_user") or "").strip().lower()
    if not owner:
        return _refuse("This Mac has no registered owner, so it can't be used to request access.")
    if owner != username:
        return _refuse("You can only request access for your own Mac.")

    name = binary.get("file_name") or "This application"
    if binary.get("virustotal_result") == MALICIOUS:
        return _refuse(f"{name} cannot be allowed because it was reported as malicious.")

    machine_id = device.get("_id")
    if rule:
        if not is_votable(rule):
            return _refuse(f"A matching {rule.get('scope', '')} {rule.get('policy', '')} rule already exists for this application.")
        return {"allowed": True, "reason": "", "action": "vote", "machine_id": machine_id}

    if not rule_target(binary):
        return _refuse(f"{name} has no identifier a rule can be created for.")

    return {"allowed": True, "reason": "", "action": "new", "machine_id": machine_id}


def record_vote(db, rule_type, identifier, machine_id, user_id, threshold, custom_msg, now):
    """Allow the identifier on machine_id and count user_id's vote.

    A missing (or deleted) rule becomes a new machine rule. An existing
    machine allow rule gains the machine and the voter. Any other existing
    rule is left untouched. The rule is promoted to global once the number of
    distinct voters reaches threshold. Returns "created", "voted", "promoted"
    or "skipped".
    """
    rules = db["rules"]
    existing = rules.find_one({"_id": identifier})

    if existing and not existing.get("deleted"):
        if not is_votable(existing):
            return "skipped"
        rules.update_one(
            {"_id": identifier, "scope": "machine", "policy": "ALLOWLIST"},
            {
                "$addToSet": {"assigned": machine_id, "voters": user_id},
                "$pull": {"removed_from": machine_id},
                "$set": {"last_updated": now},
            },
        )
        outcome = "voted"
    elif existing:
        # Revive a tombstone as a fresh machine rule.
        rules.update_one(
            {"_id": identifier},
            {
                "$set": {
                    "identifier": identifier,
                    "rule_type": rule_type,
                    "policy": "ALLOWLIST",
                    "scope": "machine",
                    "assigned": [machine_id],
                    "voters": [user_id],
                    "custom_msg": custom_msg,
                    "last_updated": now,
                },
                "$unset": {"deleted": "", "deleted_at": "", "converted": "", "reviewed": ""},
            },
        )
        outcome = "created"
    else:
        rules.update_one(
            {"_id": identifier},
            {
                "$setOnInsert": {
                    "identifier": identifier,
                    "rule_type": rule_type,
                    "policy": "ALLOWLIST",
                    "scope": "machine",
                    "custom_msg": custom_msg,
                    "creation_time": now,
                },
                "$addToSet": {"assigned": machine_id, "voters": user_id},
                "$set": {"last_updated": now},
            },
            upsert=True,
        )
        outcome = "created"

    rule = rules.find_one({"_id": identifier}) or {}
    if rule.get("scope") == "machine" and len(set(rule.get("voters") or [])) >= threshold:
        rules.update_one(
            {"_id": identifier, "scope": "machine"},
            {"$set": {"scope": "global", "converted": True, "last_updated": now}, "$unset": {"assigned": ""}},
        )
        return "promoted"

    return outcome


def custom_message(rule_type, identifier, binary):
    if binary.get("file_name"):
        return f"Allow {rule_type} with identifier: {identifier} for file: {binary['file_name']}"
    return f"Allow {rule_type} with identifier: {identifier}"


def apply_allow_request(db, decision, binary, rule, user, threshold, now):
    """Record the votes for an allowed request: the binary itself, plus any
    other binaries from the same bundle. Returns the outcome per identifier."""
    machine_id = decision["machine_id"]
    user_id = user["id"]

    if decision["action"] == "vote":
        targets = [(rule.get("rule_type"), rule["_id"], binary)]
    else:
        rule_type, identifier = rule_target(binary)
        targets = [(rule_type, identifier, binary)]

    for member in bundle_members(db, binary):
        if member.get("virustotal_result") == MALICIOUS:
            continue
        target = rule_target(member)
        if target:
            targets.append((target[0], target[1], member))

    outcomes = {}
    for rule_type, identifier, source in targets:
        if identifier in outcomes:
            continue
        outcomes[identifier] = record_vote(
            db, rule_type, identifier, machine_id, user_id, threshold,
            custom_message(rule_type, identifier, source), now,
        )
    return outcomes
