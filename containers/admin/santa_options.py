"""Allowed values and input validation for the admin UI.

These mirror the Santa sync protocol (santa.sync.v1) and the settings the
preflight service understands. Validation functions raise ValueError with a
message that is safe to show to the admin.
"""

POLICIES = [
    "ALLOWLIST",
    "ALLOWLIST_COMPILER",
    "BLOCKLIST",
    "SILENT_BLOCKLIST",
    "SILENT_GUI_BLOCKLIST",
    "SILENT_TTY_BLOCKLIST",
    "CEL",
    "REMOVE",
]

RULE_TYPES = ["BINARY", "CERTIFICATE", "TEAMID", "SIGNINGID", "CDHASH"]

SCOPES = ["global", "managedapp", "machine"]

DECISIONS = [
    "ALLOW_BINARY",
    "ALLOW_CERTIFICATE",
    "ALLOW_SCOPE",
    "ALLOW_TEAMID",
    "ALLOW_SIGNINGID",
    "ALLOW_CDHASH",
    "ALLOW_PLATFORM",
    "ALLOW_UNKNOWN",
    "BLOCK_BINARY",
    "BLOCK_CERTIFICATE",
    "BLOCK_SCOPE",
    "BLOCK_TEAMID",
    "BLOCK_SIGNINGID",
    "BLOCK_CDHASH",
    "BLOCK_BINARY_MISMATCH",
    "BLOCK_UNKNOWN",
    "BUNDLE_BINARY",
]

CLIENT_MODES = {1: "MONITOR", 2: "LOCKDOWN", 3: "STANDALONE"}

# Clean syncs an admin can request for a device.
REQUESTED_SYNC_TYPES = {
    "CLEAN_RULES": "Replace execution rules (keeps compiler-created rules)",
    "CLEAN_ALL": "Replace all rules, including compiler-created rules",
    "CLEAN_STANDALONE": "Remove rules users created in Standalone mode",
}

FILE_ACCESS_ACTIONS = ["NONE", "AUDIT_ONLY", "DISABLE"]

MEDIA_ACTIONS = ["allow", "block", "remount"]

# Rule fields that are optional and removed when left empty.
OPTIONAL_RULE_FIELDS = [
    "custom_url",
    "notification_app_name",
    "event_detail_button_label",
]

BOOL_SETTINGS = [
    "enable_bundles",
    "enable_transitive_rules",
    "enable_all_event_upload",
    "disable_unknown_event_upload",
]

STRING_SETTINGS = [
    "allowed_path_regex",
    "blocked_path_regex",
    "event_detail_url",
    "event_detail_text",
]

MEDIA_SETTINGS = ["removable_media_policy", "encrypted_removable_media_policy"]

SETTING_KEYS = (
    ["default_client_mode", "event_batch_size", "override_file_access_action"]
    + BOOL_SETTINGS
    + STRING_SETTINGS
    + MEDIA_SETTINGS
)


def _text(data, key):
    value = data.get(key)
    if value is None:
        return ""
    return str(value).strip()


def normalize_rule(data):
    """Validate a rule submitted from the admin UI.

    Returns (set_fields, unset_fields) for the rule document. The rule type
    comes in as "evaluation" and the policy as "action", matching the forms.
    """
    identifier = _text(data, "identifier")
    if not identifier:
        raise ValueError("Identifier is required")

    rule_type = _text(data, "evaluation").upper()
    if rule_type not in RULE_TYPES:
        raise ValueError(f"Unknown rule type: {rule_type or '(empty)'}")

    policy = _text(data, "action").upper()
    if policy not in POLICIES:
        raise ValueError(f"Unknown policy: {policy or '(empty)'}")

    scope = _text(data, "scope") or "global"
    if scope not in SCOPES:
        raise ValueError(f"Unknown scope: {scope}")

    set_fields = {
        "identifier": identifier,
        "rule_type": rule_type,
        "policy": policy,
        "scope": scope,
        "custom_msg": _text(data, "custom_message"),
    }
    unset_fields = ["deleted", "deleted_at"]

    cel_expr = _text(data, "cel_expr")
    if policy == "CEL":
        if not cel_expr:
            raise ValueError("A CEL expression is required for CEL rules")
        set_fields["cel_expr"] = cel_expr
    else:
        unset_fields.append("cel_expr")

    for field in OPTIONAL_RULE_FIELDS:
        value = _text(data, field)
        if value:
            set_fields[field] = value
        else:
            unset_fields.append(field)

    return set_fields, unset_fields


def _parse_bool(key, value):
    if value in (None, ""):
        return None
    if isinstance(value, bool):
        return value
    text = str(value).strip().lower()
    if text in ("true", "1", "yes", "on"):
        return True
    if text in ("false", "0", "no", "off"):
        return False
    raise ValueError(f"{key} must be true or false")


def _parse_media_policy(key, value):
    if value in (None, ""):
        return None
    if not isinstance(value, dict):
        raise ValueError(f"{key} must be an object")

    action = _text(value, "action").lower()
    if not action:
        return None
    if action not in MEDIA_ACTIONS:
        raise ValueError(f"{key}: unknown action {action}")

    policy = {"action": action}
    if action == "remount":
        flags = value.get("remount_flags") or []
        if isinstance(flags, str):
            flags = flags.split(",")
        flags = [str(flag).strip() for flag in flags if str(flag).strip()]
        if not flags:
            raise ValueError(f"{key}: remount needs at least one mount flag, e.g. rdonly")
        policy["remount_flags"] = flags
    return policy


def parse_settings(data, allow_default_client_mode=True):
    """Validate preflight settings submitted from the admin UI.

    Empty values mean "not set" (inherit), so they are returned in the unset
    list. Returns (set_fields, unset_fields) keyed by setting name.
    """
    set_fields = {}
    unset_fields = []

    def store(key, value):
        if value is None:
            unset_fields.append(key)
        else:
            set_fields[key] = value

    mode = _text(data, "default_client_mode").upper()
    if allow_default_client_mode:
        if mode and mode not in CLIENT_MODES.values():
            raise ValueError(f"Unknown client mode: {mode}")
        store("default_client_mode", mode or None)
    elif mode:
        raise ValueError("default_client_mode can only be set globally")

    batch = _text(data, "event_batch_size")
    if batch:
        try:
            batch_value = int(batch)
        except ValueError:
            raise ValueError("event_batch_size must be a whole number") from None
        if batch_value <= 0:
            raise ValueError("event_batch_size must be greater than 0")
        store("event_batch_size", batch_value)
    else:
        store("event_batch_size", None)

    for key in BOOL_SETTINGS:
        store(key, _parse_bool(key, data.get(key)))

    for key in STRING_SETTINGS:
        store(key, _text(data, key) or None)

    action = _text(data, "override_file_access_action").upper()
    if action and action not in FILE_ACCESS_ACTIONS:
        raise ValueError(f"Unknown file access action: {action}")
    store("override_file_access_action", action or None)

    for key in MEDIA_SETTINGS:
        store(key, _parse_media_policy(key, data.get(key)))

    if not allow_default_client_mode:
        unset_fields = [key for key in unset_fields if key != "default_client_mode"]

    return set_fields, unset_fields


MALWAREBAZAAR_KEY_MAX_LENGTH = 200


def parse_malwarebazaar_key(data):
    """Validate the MalwareBazaar Auth-Key from the settings page.

    An empty key turns the feed off and is returned as None.
    """
    key = _text(data, "api_key")
    if not key:
        return None
    if len(key) > MALWAREBAZAAR_KEY_MAX_LENGTH or not key.isascii() or not key.isprintable() or " " in key:
        raise ValueError("The API key must be a single word of printable characters")
    return key


def mongo_update(set_fields, unset_fields, prefix=""):
    """Build a MongoDB update document, optionally nesting under prefix."""
    update = {}
    if set_fields:
        update["$set"] = {f"{prefix}{key}": value for key, value in set_fields.items()}
    if unset_fields:
        update["$unset"] = {f"{prefix}{key}": "" for key in unset_fields}
    return update
