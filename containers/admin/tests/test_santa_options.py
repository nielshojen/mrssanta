import pytest

from santa_options import mongo_update, normalize_rule, parse_malwarebazaar_key, parse_settings


def rule_form(**overrides):
    data = {
        "identifier": "EQHXZ8M8AV",
        "evaluation": "TEAMID",
        "action": "BLOCKLIST",
        "scope": "global",
        "custom_message": "Not allowed",
    }
    data.update(overrides)
    return data


class TestNormalizeRule:
    def test_basic_rule(self):
        set_fields, unset_fields = normalize_rule(rule_form())

        assert set_fields == {
            "identifier": "EQHXZ8M8AV",
            "rule_type": "TEAMID",
            "policy": "BLOCKLIST",
            "scope": "global",
            "custom_msg": "Not allowed",
        }
        assert "deleted" in unset_fields and "deleted_at" in unset_fields
        assert "cel_expr" in unset_fields

    def test_values_are_normalised(self):
        set_fields, _ = normalize_rule(rule_form(evaluation="signingid", action="silent_tty_blocklist", identifier=" x "))

        assert set_fields["rule_type"] == "SIGNINGID"
        assert set_fields["policy"] == "SILENT_TTY_BLOCKLIST"
        assert set_fields["identifier"] == "x"

    def test_scope_defaults_to_global(self):
        set_fields, _ = normalize_rule(rule_form(scope=""))
        assert set_fields["scope"] == "global"

    def test_cel_rule_requires_expression(self):
        with pytest.raises(ValueError, match="CEL expression"):
            normalize_rule(rule_form(action="CEL", cel_expr="  "))

        set_fields, unset_fields = normalize_rule(rule_form(action="CEL", cel_expr="target.signing_time > 0"))
        assert set_fields["cel_expr"] == "target.signing_time > 0"
        assert "cel_expr" not in unset_fields

    def test_cel_expr_dropped_for_other_policies(self):
        set_fields, unset_fields = normalize_rule(rule_form(cel_expr="true"))
        assert "cel_expr" not in set_fields
        assert "cel_expr" in unset_fields

    def test_optional_fields(self):
        set_fields, unset_fields = normalize_rule(rule_form(
            custom_url="https://example.com",
            notification_app_name="Example",
            event_detail_button_label="",
        ))

        assert set_fields["custom_url"] == "https://example.com"
        assert set_fields["notification_app_name"] == "Example"
        assert "event_detail_button_label" in unset_fields

    @pytest.mark.parametrize("overrides, message", [
        ({"identifier": ""}, "Identifier"),
        ({"evaluation": "PATH"}, "rule type"),
        ({"action": "MAYBE"}, "policy"),
        ({"scope": "everyone"}, "scope"),
    ])
    def test_invalid_rules(self, overrides, message):
        with pytest.raises(ValueError, match=message):
            normalize_rule(rule_form(**overrides))


class TestParseSettings:
    def test_empty_form_unsets_everything(self):
        set_fields, unset_fields = parse_settings({})

        assert set_fields == {}
        assert "default_client_mode" in unset_fields
        assert "event_batch_size" in unset_fields
        assert "removable_media_policy" in unset_fields

    def test_full_form(self):
        set_fields, unset_fields = parse_settings({
            "default_client_mode": "monitor",
            "event_batch_size": "100",
            "enable_bundles": "false",
            "enable_all_event_upload": "true",
            "allowed_path_regex": "^/opt/",
            "event_detail_url": "https://santa.example.com/%file_sha%",
            "override_file_access_action": "audit_only",
            "removable_media_policy": {"action": "remount", "remount_flags": "rdonly, noexec"},
            "encrypted_removable_media_policy": {"action": "allow"},
        })

        assert set_fields == {
            "default_client_mode": "MONITOR",
            "event_batch_size": 100,
            "enable_bundles": False,
            "enable_all_event_upload": True,
            "allowed_path_regex": "^/opt/",
            "event_detail_url": "https://santa.example.com/%file_sha%",
            "override_file_access_action": "AUDIT_ONLY",
            "removable_media_policy": {"action": "remount", "remount_flags": ["rdonly", "noexec"]},
            "encrypted_removable_media_policy": {"action": "allow"},
        }
        assert "enable_transitive_rules" in unset_fields
        assert "blocked_path_regex" in unset_fields

    def test_false_is_kept_not_unset(self):
        set_fields, unset_fields = parse_settings({"disable_unknown_event_upload": "false"})
        assert set_fields["disable_unknown_event_upload"] is False
        assert "disable_unknown_event_upload" not in unset_fields

    def test_device_settings_cannot_set_default_client_mode(self):
        with pytest.raises(ValueError, match="globally"):
            parse_settings({"default_client_mode": "MONITOR"}, allow_default_client_mode=False)

        _, unset_fields = parse_settings({}, allow_default_client_mode=False)
        assert "default_client_mode" not in unset_fields

    @pytest.mark.parametrize("data, message", [
        ({"default_client_mode": "PARANOID"}, "client mode"),
        ({"event_batch_size": "0"}, "greater than 0"),
        ({"event_batch_size": "lots"}, "whole number"),
        ({"enable_bundles": "maybe"}, "true or false"),
        ({"override_file_access_action": "SOMETIMES"}, "file access action"),
        ({"removable_media_policy": {"action": "explode"}}, "unknown action"),
        ({"removable_media_policy": {"action": "remount", "remount_flags": ""}}, "mount flag"),
        ({"removable_media_policy": "block"}, "object"),
    ])
    def test_invalid_settings(self, data, message):
        with pytest.raises(ValueError, match=message):
            parse_settings(data)


def test_mongo_update_with_prefix():
    update = mongo_update({"event_batch_size": 10}, ["enable_bundles"], prefix="settings.")
    assert update == {
        "$set": {"settings.event_batch_size": 10},
        "$unset": {"settings.enable_bundles": ""},
    }


def test_mongo_update_omits_empty_operators():
    assert mongo_update({}, []) == {}
    assert mongo_update({"a": 1}, []) == {"$set": {"a": 1}}


class TestParseMalwareBazaarKey:
    def test_key_is_trimmed(self):
        assert parse_malwarebazaar_key({"api_key": "  abc123  "}) == "abc123"

    @pytest.mark.parametrize("data", [{}, {"api_key": ""}, {"api_key": "   "}, {"api_key": None}])
    def test_empty_key_turns_feed_off(self, data):
        assert parse_malwarebazaar_key(data) is None

    @pytest.mark.parametrize("key", ["two words", "a" * 201, "line\nbreak", "kéy"])
    def test_invalid_keys(self, key):
        with pytest.raises(ValueError, match="single word"):
            parse_malwarebazaar_key({"api_key": key})
