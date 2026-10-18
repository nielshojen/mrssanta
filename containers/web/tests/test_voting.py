from datetime import datetime, timezone

import mongomock
import pytest

import voting

NOW = datetime(2026, 10, 18, 12, 0, tzinfo=timezone.utc)
ALICE = {"id": "user-alice", "upn": "alice@example.com", "email": "alice@example.com"}
BOB = {"id": "user-bob", "upn": "bob@example.com", "email": "bob@example.com"}


@pytest.fixture
def db():
    return mongomock.MongoClient()["mrssanta"]


def device(owner="alice", machine_id="mac-alice"):
    return {"_id": machine_id, "primary_user": owner}


def binary(**overrides):
    doc = {
        "_id": "sha-tool",
        "file_sha256": "sha-tool",
        "file_name": "tool",
        "signing_id": "TEAM123456:com.example.tool",
        "team_id": "TEAM123456",
        "signingStatus": "SIGNING_STATUS_PRODUCTION",
    }
    doc.update(overrides)
    return doc


class TestUsername:
    def test_uses_upn_local_part(self):
        assert voting.username_from_session_user({"upn": "Alice@Example.com", "email": "a.smith@example.com"}) == "alice"

    def test_falls_back_to_email(self):
        assert voting.username_from_session_user({"email": "alice@example.com"}) == "alice"

    def test_empty(self):
        assert voting.username_from_session_user(None) == ""
        assert voting.username_from_session_user({}) == ""


class TestRuleTarget:
    def test_signed_binary_uses_signing_id(self):
        assert voting.rule_target(binary()) == ("SIGNINGID", "TEAM123456:com.example.tool")

    @pytest.mark.parametrize("status", ["SIGNING_STATUS_ADHOC", "SIGNING_STATUS_UNSIGNED", "SIGNING_STATUS_INVALID"])
    def test_unusable_signature_uses_hash(self, status):
        assert voting.rule_target(binary(signingStatus=status)) == ("BINARY", "sha-tool")

    def test_no_signing_id_uses_hash(self):
        assert voting.rule_target(binary(signing_id="")) == ("BINARY", "sha-tool")

    def test_nothing_usable(self):
        assert voting.rule_target({}) is None


class TestAllowDecision:
    def test_owner_may_request_new_rule(self):
        decision = voting.allow_decision(device(), binary(), None, ALICE)
        assert decision == {"allowed": True, "reason": "", "action": "new", "machine_id": "mac-alice"}

    def test_owner_match_ignores_case(self):
        assert voting.allow_decision(device(owner="Alice"), binary(), None, ALICE)["allowed"]

    def test_other_users_mac_is_refused(self):
        decision = voting.allow_decision(device(owner="bob"), binary(), None, ALICE)
        assert not decision["allowed"]
        assert "your own Mac" in decision["reason"]

    def test_mac_without_owner_is_refused(self):
        decision = voting.allow_decision(device(owner=""), binary(), None, ALICE)
        assert not decision["allowed"]
        assert "no registered owner" in decision["reason"]

    def test_malicious_binary_is_refused(self):
        decision = voting.allow_decision(device(), binary(virustotal_result=2), None, ALICE)
        assert not decision["allowed"]
        assert "malicious" in decision["reason"]

    @pytest.mark.parametrize("rule", [
        {"_id": "TEAM123456", "scope": "global", "policy": "BLOCKLIST"},
        {"_id": "TEAM123456", "scope": "global", "policy": "ALLOWLIST"},
        {"_id": "TEAM123456", "scope": "managedapp", "policy": "ALLOWLIST"},
        {"_id": "TEAM123456", "scope": "machine", "policy": "BLOCKLIST"},
    ])
    def test_existing_non_votable_rule_is_refused(self, rule):
        decision = voting.allow_decision(device(), binary(), rule, ALICE)
        assert not decision["allowed"]
        assert "already exists" in decision["reason"]

    def test_existing_machine_allow_rule_is_a_vote(self):
        rule = {"_id": "TEAM123456:com.example.tool", "scope": "machine", "policy": "ALLOWLIST"}
        assert voting.allow_decision(device(), binary(), rule, ALICE)["action"] == "vote"

    def test_missing_device_binary_or_identity(self):
        assert not voting.allow_decision(None, binary(), None, ALICE)["allowed"]
        assert not voting.allow_decision(device(), None, None, ALICE)["allowed"]
        assert not voting.allow_decision(device(), binary(), None, {"upn": "alice@example.com"})["allowed"]


class TestFindMatchingRule:
    def test_prefers_team_id_and_ignores_tombstones(self, db):
        db.rules.insert_many([
            {"_id": "TEAM123456", "scope": "global", "policy": "ALLOWLIST", "deleted": True},
            {"_id": "TEAM123456:com.example.tool", "scope": "machine", "policy": "ALLOWLIST"},
        ])
        assert voting.find_matching_rule(db, binary())["_id"] == "TEAM123456:com.example.tool"

    def test_no_rule(self, db):
        assert voting.find_matching_rule(db, binary()) is None


class TestRecordVote:
    def vote(self, db, user_id, machine_id, threshold=3, identifier="sig"):
        return voting.record_vote(db, "SIGNINGID", identifier, machine_id, user_id, threshold, "msg", NOW)

    def test_creates_machine_rule(self, db):
        assert self.vote(db, "u1", "m1") == "created"

        rule = db.rules.find_one({"_id": "sig"})
        assert rule["scope"] == "machine"
        assert rule["policy"] == "ALLOWLIST"
        assert rule["assigned"] == ["m1"]
        assert rule["voters"] == ["u1"]
        assert rule["last_updated"] is not None

    def test_promotes_after_threshold_distinct_users(self, db):
        assert self.vote(db, "u1", "m1") == "created"
        assert self.vote(db, "u2", "m2") == "voted"
        assert self.vote(db, "u3", "m3") == "promoted"

        rule = db.rules.find_one({"_id": "sig"})
        assert rule["scope"] == "global"
        assert rule["converted"] is True
        assert "assigned" not in rule

    def test_one_user_with_many_machines_is_one_vote(self, db):
        for machine in ["m1", "m2", "m3", "m4"]:
            self.vote(db, "u1", machine)

        rule = db.rules.find_one({"_id": "sig"})
        assert rule["scope"] == "machine"
        assert rule["voters"] == ["u1"]
        assert sorted(rule["assigned"]) == ["m1", "m2", "m3", "m4"]

    @pytest.mark.parametrize("existing", [
        {"scope": "global", "policy": "BLOCKLIST"},
        {"scope": "managedapp", "policy": "ALLOWLIST", "assigned": ["m9"]},
        {"scope": "machine", "policy": "BLOCKLIST", "assigned": ["m9"]},
    ])
    def test_leaves_other_rules_untouched(self, db, existing):
        db.rules.insert_one({"_id": "sig", **existing})

        assert self.vote(db, "u1", "m1") == "skipped"
        rule = db.rules.find_one({"_id": "sig"})
        rule.pop("_id")
        assert rule == existing

    def test_revives_tombstone_as_machine_rule(self, db):
        db.rules.insert_one({"_id": "sig", "scope": "global", "policy": "BLOCKLIST", "deleted": True, "deleted_at": NOW})

        assert self.vote(db, "u1", "m1") == "created"
        rule = db.rules.find_one({"_id": "sig"})
        assert rule["scope"] == "machine"
        assert rule["policy"] == "ALLOWLIST"
        assert rule["assigned"] == ["m1"]
        assert "deleted" not in rule

    def test_vote_clears_pending_removal(self, db):
        db.rules.insert_one({"_id": "sig", "scope": "machine", "policy": "ALLOWLIST", "assigned": [], "removed_from": ["m1"]})

        self.vote(db, "u1", "m1")
        assert db.rules.find_one({"_id": "sig"})["removed_from"] == []


class TestBundles:
    def test_members_match_bundle_hash_and_team(self, db):
        app = binary(file_bundle_hash="bundle-1")
        db.events.insert_many([
            app,
            {"_id": "sha-helper", "file_sha256": "sha-helper", "file_bundle_hash": "bundle-1", "team_id": "TEAM123456",
             "signing_id": "TEAM123456:com.example.helper", "signingStatus": "SIGNING_STATUS_PRODUCTION"},
            {"_id": "sha-other-team", "file_bundle_hash": "bundle-1", "team_id": "EVIL000000"},
            {"_id": "sha-same-parent", "parent_name": "tool", "team_id": "TEAM123456"},
        ])

        members = [doc["_id"] for doc in voting.bundle_members(db, app)]
        assert members == ["sha-helper"]

    def test_no_bundle_hash(self, db):
        assert voting.bundle_members(db, binary()) == []

    def test_apply_allow_request_covers_bundle(self, db):
        app = binary(file_bundle_hash="bundle-1")
        db.events.insert_many([
            app,
            {"_id": "sha-helper", "file_sha256": "sha-helper", "file_bundle_hash": "bundle-1", "team_id": "TEAM123456",
             "signingStatus": "SIGNING_STATUS_ADHOC", "signing_id": "helper"},
            {"_id": "sha-bad", "file_sha256": "sha-bad", "file_bundle_hash": "bundle-1", "team_id": "TEAM123456",
             "virustotal_result": 2},
        ])
        decision = voting.allow_decision(device(), app, None, ALICE)

        outcomes = voting.apply_allow_request(db, decision, app, None, ALICE, 3, NOW)

        assert outcomes == {"TEAM123456:com.example.tool": "created", "sha-helper": "created"}
        assert db.rules.find_one({"_id": "sha-helper"})["rule_type"] == "BINARY"
        assert db.rules.find_one({"_id": "sha-bad"}) is None
