import re
from datetime import datetime, timezone

import mongomock
import pytest
from bson.objectid import ObjectId
from flask.sessions import SecureCookieSessionInterface

import main


@pytest.fixture
def db(monkeypatch):
    database = mongomock.MongoClient()["mrssanta"]
    monkeypatch.setattr(main, "db", database)
    return database


@pytest.fixture
def client(db):
    main.app.config.update(TESTING=True, SECRET_KEY="test-secret")
    # Keep sessions in signed cookies so tests don't write session files.
    main.app.session_interface = SecureCookieSessionInterface()
    with main.app.test_client() as test_client:
        yield test_client


@pytest.fixture
def logged_in(client):
    with client.session_transaction() as sess:
        sess["user"] = {"id": "u1", "name": "Admin", "email": "admin@example.com", "profile_pic": None}
    return client


@pytest.fixture
def csrf_token(logged_in):
    page = logged_in.get("/admin/settings")
    assert page.status_code == 200
    match = re.search(r'<meta name="csrf-token" content="([^"]+)"', page.get_data(as_text=True))
    assert match, "base template must expose the CSRF token"
    return match.group(1)


def post_json(client, url, payload, token):
    return client.post(url, json=payload, headers={"X-CSRFToken": token})


class TestCSRF:
    def test_post_without_token_is_rejected(self, logged_in, db):
        response = logged_in.post("/admin/save_rule", json={
            "identifier": "abc", "evaluation": "BINARY", "action": "ALLOWLIST", "scope": "global",
        })

        assert response.status_code == 400
        assert db.rules.count_documents({}) == 0

    def test_post_with_wrong_token_is_rejected(self, logged_in, csrf_token):
        response = post_json(logged_in, "/admin/settings", {}, "not-the-token")
        assert response.status_code == 400

    def test_delete_without_token_is_rejected(self, logged_in, db):
        db.events.insert_one({"_id": "abc", "file_sha256": "abc"})

        response = logged_in.delete("/admin/events/abc/delete")

        assert response.status_code == 400
        assert db.events.count_documents({}) == 1


class TestAuth:
    @pytest.mark.parametrize("url", ["/admin/settings", "/admin/file_access", "/admin/standalone"])
    def test_pages_redirect_to_login(self, client, url):
        response = client.get(url)
        assert response.status_code == 302
        assert "/admin/login" in response.headers["Location"]

    def test_json_endpoints_require_login(self, client):
        assert client.get("/admin/load_file_access").status_code == 401
        assert client.get("/admin/load_standalone").status_code == 401


class TestPagesRender:
    @pytest.mark.parametrize("url", [
        "/admin/rules", "/admin/devices", "/admin/events",
        "/admin/settings", "/admin/file_access", "/admin/standalone",
    ])
    def test_page_renders(self, logged_in, url):
        response = logged_in.get(url)
        assert response.status_code == 200
        assert 'name="csrf-token"' in response.get_data(as_text=True)

    def test_rules_page_offers_new_policies(self, logged_in):
        html = logged_in.get("/admin/rules").get_data(as_text=True)
        for policy in ["CEL", "SILENT_GUI_BLOCKLIST", "SILENT_TTY_BLOCKLIST"]:
            assert f'value="{policy}"' in html

    def test_events_page_offers_new_decisions(self, logged_in):
        html = logged_in.get("/admin/events").get_data(as_text=True)
        assert 'value="ALLOW_PLATFORM"' in html
        assert 'value="BLOCK_BINARY_MISMATCH"' in html

    def test_devices_page_offers_standalone(self, logged_in):
        html = logged_in.get("/admin/devices").get_data(as_text=True)
        assert ">STANDALONE<" in html
        assert 'value="CLEAN_STANDALONE"' in html


class TestDashboard:
    @pytest.mark.parametrize("value,expected", [
        (None, 30), ("", 30), ("7", 7), ("0", 30), ("-3", 30), ("abc", 30), ("1.5", 30), ("70000", 30),
    ])
    def test_parse_clean_sync_max_age_days(self, value, expected):
        assert main.parse_clean_sync_max_age_days(value) == expected

    def test_stale_devices_query(self, db, monkeypatch):
        monkeypatch.setattr(main, "CLEAN_SYNC_MAX_AGE_DAYS", 30)
        now = datetime(2026, 10, 18, tzinfo=timezone.utc)
        db.devices.insert_many([
            {"_id": "fresh", "last_updated": datetime(2026, 10, 10, tzinfo=timezone.utc)},
            {"_id": "edge", "last_updated": datetime(2026, 9, 18, 0, 1, tzinfo=timezone.utc)},
            {"_id": "stale", "last_updated": datetime(2026, 9, 1, tzinfo=timezone.utc)},
            {"_id": "never"},
        ])

        stale = sorted(doc["_id"] for doc in db.devices.find(main.stale_devices_query(now)))

        assert stale == ["never", "stale"]

    def test_dashboard_shows_stale_device_card(self, logged_in, db, monkeypatch):
        monkeypatch.setattr(main, "CLEAN_SYNC_MAX_AGE_DAYS", 14)
        db.devices.insert_many([
            {"_id": "fresh", "last_updated": datetime.now(timezone.utc)},
            {"_id": "stale", "last_updated": datetime(2020, 1, 1, tzinfo=timezone.utc)},
        ])

        html = logged_in.get("/admin/").get_data(as_text=True)

        assert re.search(r"devices not seen in 14 days</div>\s*<div class=\"card-body text-center\">\s*"
                         r"<h3 class=\"card-title display-4\">1</h3>", html)


class TestLayout:
    @pytest.mark.parametrize("url,title", [
        ("/admin/", "Dashboard"), ("/admin/rules", "Rules"), ("/admin/review", "Review"),
        ("/admin/devices", "Devices"), ("/admin/events", "Events"), ("/admin/settings", "Settings"),
        ("/admin/file_access", "File Access"), ("/admin/standalone", "Standalone Approvals"),
    ])
    def test_page_has_title(self, logged_in, url, title):
        html = logged_in.get(url).get_data(as_text=True)
        assert f'<h1 class="page-title">{title}</h1>' in html

    @pytest.mark.parametrize("url,label", [
        ("/admin/", "Home"), ("/admin/rules", "Rules"), ("/admin/events", "Events"),
        ("/admin/events/abc", "Events"), ("/admin/settings", "Settings"),
    ])
    def test_current_page_is_active_in_nav(self, logged_in, db, url, label):
        db.events.insert_one({"_id": "abc", "file_sha256": "abc"})
        html = logged_in.get(url).get_data(as_text=True)

        active = re.findall(r'class="nav-link[^"]* active"[^>]*>\s*(\w[\w ]*?)\s*<', html)
        assert active == [label]

    def test_theme_is_applied_before_first_paint(self, logged_in):
        html = logged_in.get("/admin/").get_data(as_text=True)
        head = html.split("</head>")[0]
        assert 'localStorage.getItem("theme")' in head
        assert "prefers-color-scheme: dark" in head
        assert 'systemDark.addEventListener("change", applyTheme)' in head
        assert '<meta name="theme-color" content="#ff5050">' in head

    def test_theme_menu_offers_system_light_and_dark(self, logged_in):
        html = logged_in.get("/admin/").get_data(as_text=True)
        choices = re.findall(r'data-theme-choice="(\w+)"', html)
        assert choices == ["system", "light", "dark"]
        assert 'localStorage.removeItem("theme")' in html

    def test_header_shows_logo_and_user(self, logged_in):
        html = logged_in.get("/admin/").get_data(as_text=True)
        assert 'src="/admin/static/images/android-chrome-192x192.png" class="brand-mark"' in html
        assert 'id="themeToggle"' in html
        assert ">Admin<" in html

    def test_review_count_shown_in_nav(self, logged_in, db):
        db.rules.insert_many([
            {"_id": f"r{i}", "identifier": f"r{i}", "policy": "ALLOWLIST", "scope": "machine"} for i in range(3)
        ])
        html = logged_in.get("/admin/").get_data(as_text=True)
        assert '<span class="badge rounded-pill nav-count">3</span>' in html


class TestRules:
    def test_save_rule_with_new_fields(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/save_rule", {
            "identifier": "EQHXZ8M8AV",
            "evaluation": "TEAMID",
            "action": "CEL",
            "scope": "global",
            "custom_message": "",
            "cel_expr": "target.signing_time > timestamp('2025-01-01T00:00:00Z')",
            "notification_app_name": "Example",
        }, csrf_token)

        assert response.status_code == 200
        rule = db.rules.find_one({"_id": "EQHXZ8M8AV"})
        assert rule["policy"] == "CEL"
        assert rule["cel_expr"].startswith("target.signing_time")
        assert rule["notification_app_name"] == "Example"
        assert "last_updated" in rule

    def test_save_rule_validates(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/save_rule", {
            "identifier": "abc", "evaluation": "TEAMID", "action": "CEL", "scope": "global",
        }, csrf_token)

        assert response.status_code == 400
        assert "CEL expression" in response.get_json()["message"]
        assert db.rules.count_documents({}) == 0

    def test_saving_a_deleted_rule_revives_it(self, logged_in, csrf_token, db):
        db.rules.insert_one({"_id": "abc", "identifier": "abc", "deleted": True, "deleted_at": datetime.now(timezone.utc)})

        post_json(logged_in, "/admin/save_rule", {
            "identifier": "abc", "evaluation": "BINARY", "action": "ALLOWLIST", "scope": "global",
        }, csrf_token)

        rule = db.rules.find_one({"_id": "abc"})
        assert "deleted" not in rule
        assert "deleted_at" not in rule

    def test_delete_rule_leaves_tombstone(self, logged_in, csrf_token, db):
        db.rules.insert_one({"_id": "abc", "identifier": "abc", "policy": "ALLOWLIST", "scope": "global"})

        response = post_json(logged_in, "/admin/rules/abc/delete", {}, csrf_token)

        assert response.status_code == 200
        rule = db.rules.find_one({"_id": "abc"})
        assert rule["deleted"] is True
        assert "deleted_at" in rule and "last_updated" in rule

    def test_delete_unknown_rule(self, logged_in, csrf_token):
        assert post_json(logged_in, "/admin/rules/nope/delete", {}, csrf_token).status_code == 404

    def test_load_rules_hides_tombstones(self, logged_in, db):
        db.rules.insert_many([
            {"_id": "a", "identifier": "a", "policy": "ALLOWLIST"},
            {"_id": "b", "identifier": "b", "policy": "ALLOWLIST", "deleted": True},
        ])

        data = logged_in.get("/admin/load_rules").get_json()["data"]

        assert [rule["identifier"] for rule in data] == ["a"]

    def test_load_rules_escapes_regex_input(self, logged_in, db):
        db.rules.insert_one({"_id": "a", "identifier": "a"})
        response = logged_in.get("/admin/load_rules?identifier=(")
        assert response.status_code == 200


class TestDevices:
    @pytest.fixture(autouse=True)
    def device(self, db):
        db.devices.insert_one({"_id": "m1", "identifier": "m1", "hostname": "mac1.example.com", "client_mode": 2})

    def test_save_device_accepts_standalone(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/save_device", {"identifier": "m1", "client_mode": 3}, csrf_token)

        assert response.status_code == 200
        assert db.devices.find_one({"_id": "m1"})["client_mode"] == 3

    def test_save_device_rejects_unknown_mode(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/save_device", {"identifier": "m1", "client_mode": 7}, csrf_token)

        assert response.status_code == 400
        assert db.devices.find_one({"_id": "m1"})["client_mode"] == 2

    def test_request_and_cancel_clean_sync(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/devices/m1/clean_sync", {"sync_type": "clean_rules"}, csrf_token)
        assert response.status_code == 200
        assert db.devices.find_one({"_id": "m1"})["requested_sync_type"] == "CLEAN_RULES"

        response = post_json(logged_in, "/admin/devices/m1/clean_sync", {"sync_type": ""}, csrf_token)
        assert response.status_code == 200
        assert "requested_sync_type" not in db.devices.find_one({"_id": "m1"})

    def test_request_clean_sync_validates(self, logged_in, csrf_token):
        assert post_json(logged_in, "/admin/devices/m1/clean_sync", {"sync_type": "NORMAL"}, csrf_token).status_code == 400
        assert post_json(logged_in, "/admin/devices/nope/clean_sync", {"sync_type": "CLEAN_ALL"}, csrf_token).status_code == 404

    def test_device_settings_are_stored_as_overrides(self, logged_in, csrf_token, db):
        db.devices.update_one({"_id": "m1"}, {"$set": {"settings": {"enable_bundles": True}}})

        response = post_json(logged_in, "/admin/devices/m1/settings", {
            "event_batch_size": "25",
            "enable_bundles": "",
            "removable_media_policy": {"action": "block"},
        }, csrf_token)

        assert response.status_code == 200
        settings = db.devices.find_one({"_id": "m1"})["settings"]
        assert settings == {"event_batch_size": 25, "removable_media_policy": {"action": "block"}}

    def test_device_settings_cannot_set_default_client_mode(self, logged_in, csrf_token):
        response = post_json(logged_in, "/admin/devices/m1/settings", {"default_client_mode": "MONITOR"}, csrf_token)
        assert response.status_code == 400


class TestGlobalSettings:
    def test_save_global_settings(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/settings", {
            "default_client_mode": "MONITOR",
            "event_detail_url": "https://santa.example.com/%file_sha%",
            "enable_all_event_upload": "true",
        }, csrf_token)

        assert response.status_code == 200
        settings = db.settings.find_one({"_id": "global"})
        assert settings["default_client_mode"] == "MONITOR"
        assert settings["event_detail_url"] == "https://santa.example.com/%file_sha%"
        assert settings["enable_all_event_upload"] is True

    def test_invalid_settings_are_rejected(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/settings", {"event_batch_size": "-1"}, csrf_token)

        assert response.status_code == 400
        assert db.settings.count_documents({}) == 0

    @pytest.mark.parametrize("url", ["/admin/settings", "/admin/devices"])
    def test_path_regex_fields_get_builder(self, logged_in, db, url):
        html = logged_in.get(url).get_data(as_text=True)

        assert 'data-setting="allowed_path_regex" data-regex-builder="allow"' in html
        assert 'data-setting="blocked_path_regex" data-regex-builder="block"' in html
        assert "js/path_regex.js" in html

    def test_settings_page_shows_saved_values(self, logged_in, db):
        db.settings.insert_one({"_id": "global", "event_detail_text": "Ask IT"})
        html = logged_in.get("/admin/settings").get_data(as_text=True)
        assert "Ask IT" in html


FLAGGED_SHA = "a" * 64


def flagged_event(**match):
    return {
        "_id": FLAGGED_SHA,
        "file_sha256": FLAGGED_SHA,
        "file_name": "evil",
        "decision": "BLOCK_UNKNOWN",
        "malwarebazaar": {
            "signature": "AMOS",
            "flagged_at": datetime(2026, 10, 18, 12, 0, tzinfo=timezone.utc),
            **match,
        },
    }


class TestMalwareBazaar:
    def test_saving_key_turns_feed_on(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/settings/malwarebazaar", {"api_key": " key123 "}, csrf_token)

        assert response.status_code == 200
        assert db.settings.find_one({"_id": "malwarebazaar"})["api_key"] == "key123"

    def test_saving_key_clears_previous_error(self, logged_in, csrf_token, db):
        db.settings.insert_one({"_id": "malwarebazaar", "api_key": "old", "last_error": "HTTP 401"})

        post_json(logged_in, "/admin/settings/malwarebazaar", {"api_key": "new"}, csrf_token)

        doc = db.settings.find_one({"_id": "malwarebazaar"})
        assert doc["api_key"] == "new"
        assert "last_error" not in doc

    def test_empty_key_turns_feed_off_and_drops_list(self, logged_in, csrf_token, db):
        db.settings.insert_one({"_id": "malwarebazaar", "api_key": "key123"})
        db.malwarebazaar_samples.insert_one({"_id": FLAGGED_SHA})
        db.events.insert_one(flagged_event())

        response = post_json(logged_in, "/admin/settings/malwarebazaar", {"api_key": ""}, csrf_token)

        assert response.status_code == 200
        assert db.settings.find_one({"_id": "malwarebazaar"}) is None
        assert db.malwarebazaar_samples.count_documents({}) == 0
        assert "malwarebazaar" in db.events.find_one({"_id": FLAGGED_SHA}), "existing flags stay"

    def test_invalid_key_is_rejected(self, logged_in, csrf_token, db):
        response = post_json(logged_in, "/admin/settings/malwarebazaar", {"api_key": "two words"}, csrf_token)

        assert response.status_code == 400
        assert db.settings.count_documents({}) == 0

    def test_saving_key_requires_csrf_token(self, logged_in, db):
        response = logged_in.post("/admin/settings/malwarebazaar", json={"api_key": "key123"})

        assert response.status_code == 400
        assert db.settings.count_documents({}) == 0

    def test_saving_key_requires_login(self, client, db):
        response = client.post("/admin/settings/malwarebazaar", json={"api_key": "key123"})

        assert response.status_code in (400, 401)
        assert db.settings.count_documents({}) == 0

    def test_settings_page_never_shows_key(self, logged_in, db):
        db.settings.insert_one({
            "_id": "malwarebazaar", "api_key": "super-secret-key",
            "list_refreshed_at": datetime(2026, 10, 18, 12, 0), "list_size": 12345,
        })

        html = logged_in.get("/admin/settings").get_data(as_text=True)

        assert "super-secret-key" not in html
        assert "12,345 macOS samples, list updated 2026-10-18 12:00 UTC" in html
        assert 'id="malwarebazaarOff"' in html

    def test_settings_page_shows_last_error(self, logged_in, db):
        db.settings.insert_one({"_id": "malwarebazaar", "api_key": "k", "last_error": "HTTP 401: unknown_auth_key"})

        html = logged_in.get("/admin/settings").get_data(as_text=True)

        assert "Last check failed: HTTP 401: unknown_auth_key" in html

    def test_settings_page_when_off(self, logged_in, db):
        html = logged_in.get("/admin/settings").get_data(as_text=True)

        assert "Turn on" in html
        assert 'id="malwarebazaarOff"' not in html

    def test_settings_page_before_first_download(self, logged_in, db):
        db.settings.insert_one({"_id": "malwarebazaar", "api_key": "k"})

        html = logged_in.get("/admin/settings").get_data(as_text=True)

        assert "Waiting for the first download" in html

    def test_flagged_events_listed_on_review_and_counted(self, logged_in, db):
        db.events.insert_many([
            flagged_event(),
            {**flagged_event(reviewed=True), "_id": "b" * 64, "file_name": "already-reviewed"},
            {"_id": "c" * 64, "file_name": "clean"},
        ])

        html = logged_in.get("/admin/review").get_data(as_text=True)

        assert "Known malware" in html
        assert FLAGGED_SHA in html
        assert "AMOS" in html
        assert "already-reviewed" not in html
        assert '<span class="badge rounded-pill nav-count">1</span>' in html

    def test_review_page_without_flagged_events(self, logged_in, db):
        html = logged_in.get("/admin/review").get_data(as_text=True)

        assert "Known malware" not in html

    def test_block_creates_global_rule_and_marks_reviewed(self, logged_in, csrf_token, db):
        db.events.insert_one(flagged_event())

        response = post_json(logged_in, f"/admin/malwarebazaar/{FLAGGED_SHA}/review", {"block": True}, csrf_token)

        assert response.status_code == 200
        rule = db.rules.find_one({"_id": FLAGGED_SHA})
        assert rule["policy"] == "BLOCKLIST"
        assert rule["rule_type"] == "BINARY"
        assert rule["scope"] == "global"
        assert "AMOS" in rule["custom_msg"]
        match = db.events.find_one({"_id": FLAGGED_SHA})["malwarebazaar"]
        assert match["reviewed"] is True
        assert match["blocked"] is True
        assert match["reviewed_by"] == "admin@example.com"
        assert match["signature"] == "AMOS"

    def test_block_replaces_existing_allow_rule(self, logged_in, csrf_token, db):
        db.events.insert_one(flagged_event())
        db.rules.insert_one({"_id": FLAGGED_SHA, "identifier": FLAGGED_SHA, "policy": "ALLOWLIST", "scope": "machine",
                             "deleted": True})

        post_json(logged_in, f"/admin/malwarebazaar/{FLAGGED_SHA}/review", {"block": True}, csrf_token)

        rule = db.rules.find_one({"_id": FLAGGED_SHA})
        assert rule["policy"] == "BLOCKLIST"
        assert rule["scope"] == "global"
        assert "deleted" not in rule

    def test_dismiss_marks_reviewed_without_rule(self, logged_in, csrf_token, db):
        db.events.insert_one(flagged_event())

        response = post_json(logged_in, f"/admin/malwarebazaar/{FLAGGED_SHA}/review", {"block": False}, csrf_token)

        assert response.status_code == 200
        assert db.rules.count_documents({}) == 0
        match = db.events.find_one({"_id": FLAGGED_SHA})["malwarebazaar"]
        assert match["reviewed"] is True
        assert match["blocked"] is False

    def test_review_of_unflagged_event_is_rejected(self, logged_in, csrf_token, db):
        db.events.insert_one({"_id": FLAGGED_SHA, "file_name": "clean"})

        response = post_json(logged_in, f"/admin/malwarebazaar/{FLAGGED_SHA}/review", {"block": True}, csrf_token)

        assert response.status_code == 404
        assert db.rules.count_documents({}) == 0

    def test_review_requires_csrf_token(self, logged_in, db):
        db.events.insert_one(flagged_event())

        response = logged_in.post(f"/admin/malwarebazaar/{FLAGGED_SHA}/review", json={"block": True})

        assert response.status_code == 400
        assert db.rules.count_documents({}) == 0

    def test_event_details_shows_match(self, logged_in, db, monkeypatch):
        monkeypatch.setattr(main, "get_vt_result", lambda sha: None)
        db.events.insert_one(flagged_event())

        html = logged_in.get(f"/admin/events/{FLAGGED_SHA}").get_data(as_text=True)

        assert 'id="malwarebazaarMatch"' in html
        assert "AMOS" in html
        assert "Review it" in html


class TestEventLogs:
    def test_file_access_events_include_hostname_and_paginate(self, logged_in, db):
        db.devices.insert_one({"_id": "m1", "hostname": "mac1.example.com"})
        db.file_access_events.insert_many([
            {"_id": ObjectId(), "machine_id": "m1", "rule_name": "ProtectSSH", "target": f"/t{i}"}
            for i in range(30)
        ])

        first = logged_in.get("/admin/load_file_access").get_json()
        assert len(first["data"]) == 25
        assert first["data"][0]["hostname"] == "mac1.example.com"
        assert first["data"][0]["target"] == "/t29", "newest first"

        second = logged_in.get(f"/admin/load_file_access?cursor={first['next_cursor']}").get_json()
        assert len(second["data"]) == 5
        assert second["next_cursor"] is None

    def test_file_access_filters(self, logged_in, db):
        db.file_access_events.insert_many([
            {"machine_id": "m1", "rule_name": "A"},
            {"machine_id": "m2", "rule_name": "B"},
        ])

        data = logged_in.get("/admin/load_file_access?rule_name=B").get_json()["data"]
        assert [event["machine_id"] for event in data] == ["m2"]

    def test_bad_cursor_returns_empty_page(self, logged_in):
        assert logged_in.get("/admin/load_file_access?cursor=nope").get_json() == {"data": [], "next_cursor": None}

    def test_standalone_approvals(self, logged_in, db):
        db.standalone_approvals.insert_one({"machine_id": "m1", "identifier": "abc", "decision": "ALLOW_BINARY"})

        data = logged_in.get("/admin/load_standalone?machine_id=m1").get_json()["data"]

        assert data[0]["identifier"] == "abc"
        assert data[0]["hostname"] == ""


class TestLoginCallbackEscaping:
    PAYLOAD = "<script>alert(1)</script>"

    def test_error_parameters_are_escaped(self, client):
        response = client.get("/admin/token", query_string={"error": self.PAYLOAD, "error_description": self.PAYLOAD})

        html = response.get_data(as_text=True)
        assert self.PAYLOAD not in html
        assert "&lt;script&gt;alert(1)&lt;/script&gt;" in html

    def test_missing_error_description(self, client):
        response = client.get("/admin/token", query_string={"error": "access_denied"})

        assert response.status_code == 200
        assert "access_denied" in response.get_data(as_text=True)

    def test_authentication_failure_is_escaped(self, client, monkeypatch):
        class FakeApp:
            def __init__(self, *args, **kwargs):
                pass

            def acquire_token_by_authorization_code(self, *args, **kwargs):
                return {"error": "invalid_grant", "error_description": TestLoginCallbackEscaping.PAYLOAD}

        monkeypatch.setattr(main.msal, "ConfidentialClientApplication", FakeApp)

        html = client.get("/admin/token", query_string={"code": "abc"}).get_data(as_text=True)

        assert self.PAYLOAD not in html
        assert "&lt;script&gt;" in html
