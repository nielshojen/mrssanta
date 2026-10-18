import re

import mongomock
import pytest
from flask.sessions import SecureCookieSessionInterface

import main

ALICE = {"id": "user-alice", "name": "Alice", "email": "alice@example.com", "upn": "alice@example.com", "profile_pic": None}


@pytest.fixture
def db(monkeypatch):
    database = mongomock.MongoClient()["mrssanta"]
    monkeypatch.setattr(main, "db", database)
    # No VirusTotal lookups in tests.
    monkeypatch.setattr(main, "get_vt_result", lambda file_hash: None)
    database.devices.insert_many([
        {"_id": "mac-alice", "identifier": "mac-alice", "hostname": "alice-mac.example.com", "primary_user": "alice"},
        {"_id": "mac-bob", "identifier": "mac-bob", "hostname": "bob-mac.example.com", "primary_user": "bob"},
        {"_id": "mac-shared", "identifier": "mac-shared", "hostname": "shared.example.com"},
    ])
    database.events.insert_one({
        "_id": "sha-tool", "file_sha256": "sha-tool", "file_name": "tool",
        "signing_id": "TEAM123456:com.example.tool", "team_id": "TEAM123456",
        "signingStatus": "SIGNING_STATUS_PRODUCTION", "virustotal_result": 1,
    })
    return database


@pytest.fixture
def client(db):
    main.app.config.update(TESTING=True, SECRET_KEY="test-secret")
    main.app.session_interface = SecureCookieSessionInterface()
    with main.app.test_client() as test_client:
        with test_client.session_transaction() as sess:
            sess["user"] = ALICE
        yield test_client


def post_rule(client, payload):
    return client.post("/santa/blockables/rule", json=payload)


class TestPost:
    def test_owner_gets_machine_rule(self, client, db):
        response = post_rule(client, {"machine_id": "mac-alice", "file_sha256": "sha-tool"})

        assert response.status_code == 200
        rule = db.rules.find_one({"_id": "TEAM123456:com.example.tool"})
        assert rule["scope"] == "machine"
        assert rule["policy"] == "ALLOWLIST"
        assert rule["assigned"] == ["mac-alice"]
        assert rule["voters"] == ["user-alice"]

    def test_forged_fields_are_ignored(self, client, db):
        response = post_rule(client, {
            "machine_id": "mac-alice", "file_sha256": "sha-tool",
            "action": "new", "scope": "global", "ruletype": "TEAMID", "ruleid": "EVIL000000",
            "identifier": "someone-else", "bundle_binaries": [{"signing_id": "EVIL000000:com.evil"}],
        })

        assert response.status_code == 200
        assert db.rules.find_one({"_id": "EVIL000000"}) is None
        assert db.rules.find_one({"_id": "EVIL000000:com.evil"}) is None
        assert db.rules.count_documents({"scope": "global"}) == 0
        assert db.rules.find_one({"_id": "TEAM123456:com.example.tool"})["assigned"] == ["mac-alice"]

    def test_other_users_mac_is_refused(self, client, db):
        response = post_rule(client, {"machine_id": "mac-bob", "file_sha256": "sha-tool"})

        assert response.status_code == 403
        assert db.rules.count_documents({}) == 0

    def test_mac_without_owner_is_refused(self, client, db):
        assert post_rule(client, {"machine_id": "mac-shared", "file_sha256": "sha-tool"}).status_code == 403
        assert db.rules.count_documents({}) == 0

    def test_malicious_binary_is_refused(self, client, db):
        db.events.update_one({"_id": "sha-tool"}, {"$set": {"virustotal_result": 2}})

        assert post_rule(client, {"machine_id": "mac-alice", "file_sha256": "sha-tool"}).status_code == 403
        assert db.rules.count_documents({}) == 0

    def test_existing_global_block_is_left_alone(self, client, db):
        db.rules.insert_one({"_id": "TEAM123456", "scope": "global", "policy": "BLOCKLIST"})

        assert post_rule(client, {"machine_id": "mac-alice", "file_sha256": "sha-tool"}).status_code == 403
        assert db.rules.find_one({"_id": "TEAM123456"}) == {"_id": "TEAM123456", "scope": "global", "policy": "BLOCKLIST"}

    @pytest.mark.parametrize("payload", [
        {},
        {"machine_id": "mac-alice"},
        {"machine_id": {"$ne": ""}, "file_sha256": "sha-tool"},
        {"machine_id": "mac-alice", "file_sha256": ["sha-tool"]},
    ])
    def test_invalid_payloads(self, client, payload):
        assert post_rule(client, payload).status_code == 400

    def test_non_json_body_is_rejected(self, client):
        response = client.post("/santa/blockables/rule", data="machine_id=mac-alice", content_type="text/plain")
        assert response.status_code == 400

    def test_requires_login(self, db):
        main.app.session_interface = SecureCookieSessionInterface()
        with main.app.test_client() as anonymous:
            response = anonymous.post("/santa/blockables/rule", json={"machine_id": "mac-alice", "file_sha256": "sha-tool"})
        assert response.status_code == 302
        assert db.rules.count_documents({}) == 0


class TestPage:
    def page(self, client, machine_id):
        return client.get(f"/santa/blockables/{machine_id}/sha-tool").get_data(as_text=True)

    def test_page_has_no_api_key_and_minimal_payload(self, client, monkeypatch):
        monkeypatch.setenv("API_KEY", "super-secret-sync-key")
        html = self.page(client, "mac-alice")

        assert "super-secret-sync-key" not in html
        assert "X-API-Key" not in html
        payload = re.search(r'<script type="application/json" id="allow-rule-payload">\s*(.*?)\s*</script>', html, re.S).group(1)
        assert '"machine_id": "mac-alice"' in payload
        assert '"file_sha256": "sha-tool"' in payload
        assert "scope" not in payload

    def test_other_users_mac_shows_reason_without_button(self, client):
        html = self.page(client, "mac-bob")

        assert "You can only request access for your own Mac." in html
        assert "data-allow-rule" not in html
        assert "allow-rule-payload" not in html


class TestLoginCallbackEscaping:
    PAYLOAD = "<script>alert(1)</script>"

    def test_error_parameters_are_escaped(self, client):
        response = client.get("/santa/token", query_string={"error": self.PAYLOAD, "error_description": self.PAYLOAD})

        html = response.get_data(as_text=True)
        assert self.PAYLOAD not in html
        assert "&lt;script&gt;alert(1)&lt;/script&gt;" in html

    def test_missing_error_description(self, client):
        response = client.get("/santa/token", query_string={"error": "access_denied"})

        assert response.status_code == 200
        assert "access_denied" in response.get_data(as_text=True)

    def test_authentication_failure_is_escaped(self, client, monkeypatch):
        class FakeApp:
            def __init__(self, *args, **kwargs):
                pass

            def acquire_token_by_authorization_code(self, *args, **kwargs):
                return {"error": "invalid_grant", "error_description": TestLoginCallbackEscaping.PAYLOAD}

        monkeypatch.setattr(main.msal, "ConfidentialClientApplication", FakeApp)

        html = client.get("/santa/token", query_string={"code": "abc"}).get_data(as_text=True)

        assert self.PAYLOAD not in html
        assert "&lt;script&gt;" in html


class TestStaticFiles:
    def test_serves_icons(self, client):
        response = client.get("/santa/static/favicon-32x32.png")

        assert response.status_code == 200
        assert response.mimetype == "image/png"

    @pytest.mark.parametrize("name", ["popup.png", "results.png", "allow.png", "sync.png", "notallowed.png", "santa-block.gif"])
    def test_serves_help_images(self, client, name):
        assert client.get(f"/santa/static/{name}").status_code == 200

    def test_manifest_icons_are_relative(self, client):
        manifest = client.get("/santa/static/site.webmanifest").get_json(force=True)

        assert [icon["src"] for icon in manifest["icons"]] == ["android-chrome-192x192.png", "android-chrome-512x512.png"]

    @pytest.mark.parametrize("path", ["../main.py", "..%2Fmain.py", "%2E%2E/main.py"])
    def test_cannot_escape_static_folder(self, client, path):
        response = client.get(f"/santa/static/{path}")

        assert response.status_code == 404
        assert b"send_from_directory" not in response.get_data()

    def test_templates_no_longer_use_storage_bucket(self):
        import pathlib
        for template in ["templates/base.html", "templates/moreinfo.html"]:
            assert "storage.googleapis.com" not in pathlib.Path(template).read_text()
