import os
import re
import requests
import base64
import msal
from datetime import datetime, timedelta, timezone

from flask import Flask, render_template, send_from_directory, abort, jsonify, request, redirect, url_for, session
from flask_session import Session
from markupsafe import escape
from flask_wtf.csrf import CSRFProtect
from pymongo import MongoClient
from bson.objectid import ObjectId

from log_config import configure_logging
import santa_options
from santa_options import mongo_update, normalize_rule, parse_malwarebazaar_key, parse_settings


logger = configure_logging("admin", "mrssanta-admin")

app = Flask(
    __name__,
    static_folder="static",
    static_url_path="/admin/static",
)
app.logger.handlers.clear()
app.logger.propagate = True

app.config["SESSION_TYPE"] = "filesystem"
app.config["SESSION_FILE_DIR"] = os.getenv("SESSION_FILE_DIR") or os.path.join(app.root_path, "flask_session")
app.secret_key = os.getenv("FLASK_SECRET_KEY")

Session(app)

# Every state-changing request (POST/PUT/PATCH/DELETE) must carry the CSRF
# token, sent by the pages in the X-CSRFToken header (see base.html).
csrf = CSRFProtect(app)

CLIENT_ID = os.getenv("MSAL_CLIENT_ID")
CLIENT_SECRET = os.getenv("MSAL_CLIENT_SECRET")
TENANT_ID = os.getenv("MSAL_TENANT_ID")
AUTHORITY = f"https://login.microsoftonline.com/{TENANT_ID}"
REDIRECT_URI = os.getenv("MSAL_REDIRECT_URI")
SCOPES = ["User.Read"]
VT_API_KEY = os.environ.get('VT_API_KEY')

DEFAULT_CLEAN_SYNC_MAX_AGE_DAYS = 30


def parse_clean_sync_max_age_days(value):
    """Parse CLEAN_SYNC_MAX_AGE_DAYS the same way the preflight service does:
    a whole number of days greater than 0, else the default."""
    if not value:
        return DEFAULT_CLEAN_SYNC_MAX_AGE_DAYS
    try:
        days = int(value)
    except ValueError:
        days = 0
    if days <= 0 or days > 65535:
        logger.warning(f"Invalid value for CLEAN_SYNC_MAX_AGE_DAYS ({value}) - keeping default {DEFAULT_CLEAN_SYNC_MAX_AGE_DAYS} days")
        return DEFAULT_CLEAN_SYNC_MAX_AGE_DAYS
    return days


CLEAN_SYNC_MAX_AGE_DAYS = parse_clean_sync_max_age_days(os.getenv("CLEAN_SYNC_MAX_AGE_DAYS"))

MONGO_URI = os.getenv("MONGO_URI")
client = MongoClient(MONGO_URI)
db = client["mrssanta"]

def get_vt_result(file_hash):
    check_failed = 0

    url = "https://www.virustotal.com/api/v3/files/%s" % file_hash

    headers = {"accept": "application/json", "x-apikey": VT_API_KEY}

    response = requests.get(url, headers=headers)

    result = response.json()

    if response.status_code == 200 and len(result['data']) > 0:
        try:
            malicious = result['data']['attributes']['last_analysis_stats']['malicious']
        except:
            malicious = 0
            check_failed = 1
    else:
        malicious = 0
        check_failed = 1

    if check_failed == 0:
        if malicious == 0:
            return 1
        else:
            return 2
    else:
        return 0

def save_binary_db(identifier, data):
    collection = db["events"]
    data["last_updated"] = datetime.now(timezone.utc)

    result = collection.update_one(
        {"_id": identifier},
        {"$set": data},
        upsert=True
    )
    logger.info("Saved binary", extra={"fields": {
        "identifier": identifier,
        "matched_count": result.matched_count,
        "modified_count": result.modified_count,
        "upserted_id": result.upserted_id,
    }})
    return data

def save_rule_db(identifier, data):
    collection = db["rules"]
    data["last_updated"] = datetime.now(timezone.utc)

    result = collection.update_one(
        {"_id": identifier},
        {"$set": data},
        upsert=True
    )
    logger.info("Saved rule", extra={"fields": {
        "identifier": identifier,
        "matched_count": result.matched_count,
        "modified_count": result.modified_count,
        "upserted_id": result.upserted_id,
    }})
    return data

def save_device_db(identifier, data):
    collection = db["devices"]
    data["last_updated"] = datetime.now(timezone.utc)

    result = collection.update_one(
        {"_id": identifier},
        {"$set": data},
        upsert=True
    )
    logger.info("Saved device", extra={"fields": {
        "identifier": identifier,
        "matched_count": result.matched_count,
        "modified_count": result.modified_count,
        "upserted_id": result.upserted_id,
    }})
    return data

def build_msal_app(user_id):
    cache = load_cache(user_id)
    app = msal.ConfidentialClientApplication(
        client_id=CLIENT_ID,
        authority=AUTHORITY,
        client_credential=CLIENT_SECRET,
        token_cache=cache
    )
    return app, cache

def load_cache(user_id):
    cache = msal.SerializableTokenCache()
    record = db["sessions"].find_one({"user_id": user_id})
    if record and "token_cache" in record:
        cache.deserialize(record["token_cache"])
    return cache

def save_cache(user_id, cache):
    if cache.has_state_changed:
        db["sessions"].update_one(
            {"user_id": user_id},
            {"$set": {
                "token_cache": cache.serialize(),
                "updated_at": datetime.utcnow()
            }},
            upsert=True
        )

def stale_devices_query(now):
    """Devices that haven't synced within CLEAN_SYNC_MAX_AGE_DAYS, the deadline
    after which preflight forces a full sync. Devices that never recorded a sync
    count as stale too."""
    cutoff = now - timedelta(days=CLEAN_SYNC_MAX_AGE_DAYS)
    return {"$or": [
        {"last_updated": {"$lt": cutoff}},
        {"last_updated": {"$exists": False}},
    ]}


def get_mongodb_count(collection_name, query=None):
    collection = db[collection_name]
    return collection.count_documents(query or {})

@app.route('/admin/site.webmanifest')
def manifest():
    return send_from_directory('/admin/static/manifest', 'site.webmanifest', mimetype='application/manifest+json')


@app.route("/admin/login")
def login():
    # No user_id yet, so just create MSAL app without a token cache
    msal_app = msal.ConfidentialClientApplication(
        client_id=CLIENT_ID,
        authority=AUTHORITY,
        client_credential=CLIENT_SECRET,
    )
    auth_url = msal_app.get_authorization_request_url(SCOPES, redirect_uri=REDIRECT_URI)
    return redirect(auth_url)


@app.route("/admin/token")
def token():
    """Handles login callback, gets user profile and profile picture."""

    # Error check
    if "error" in request.args:
        # Both values come from the query string, so anyone can put markup in
        # them with a crafted link: escape before returning HTML.
        return f"Error: {escape(request.args.get('error', ''))} - {escape(request.args.get('error_description', ''))}"

    # Code check
    code = request.args.get("code")
    if not code:
        return "No authorization code received"

    # Create fresh token cache
    cache = msal.SerializableTokenCache()

    # Build MSAL app with cache
    msal_app = msal.ConfidentialClientApplication(
        client_id=CLIENT_ID,
        authority=AUTHORITY,
        client_credential=CLIENT_SECRET,
        token_cache=cache
    )

    # Acquire token using auth code
    result = msal_app.acquire_token_by_authorization_code(
        code,
        scopes=SCOPES,
        redirect_uri=REDIRECT_URI
    )

    if "access_token" in result:
        access_token = result["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}

        # Fetch user profile
        user_info = requests.get("https://graph.microsoft.com/v1.0/me", headers=headers).json()
        user_id = user_info.get("id")

        # Fetch profile picture (optional)
        profile_pic = None
        pic_response = requests.get("https://graph.microsoft.com/v1.0/me/photo/$value", headers=headers)
        if pic_response.status_code == 200:
            profile_pic = base64.b64encode(pic_response.content).decode("utf-8")

        # Save MSAL token cache to MongoDB
        save_cache(user_id, cache)

        # Store user in session (for easy reference)
        session["user"] = {
            "id": user_id,
            "name": user_info.get("displayName", "User"),
            "email": user_info.get("mail", user_info.get("userPrincipalName")),
            "profile_pic": profile_pic,
        }

        return redirect(url_for("index"))

    # If something went wrong
    return f"Authentication failed: {escape(result.get('error_description') or result.get('error') or '')}"


@app.route("/admin/logout")
def logout():
    session.clear()
    return redirect(url_for("index"))

@app.route("/admin/")
def index():
    if "user" not in session:
        return redirect(url_for("login"))

    counts = {
        "devices": get_mongodb_count("devices"),
        "rules": get_mongodb_count("rules"),
        "events": get_mongodb_count("events"),
        "devices in lockdown": get_mongodb_count(
            "devices",
            {"client_mode": 2}
        ),
        f"devices not seen in {CLEAN_SYNC_MAX_AGE_DAYS} days": get_mongodb_count(
            "devices",
            stale_devices_query(datetime.now(timezone.utc)),
        ),
    }

    return render_template("index.html", counts=counts)

# Devices
@app.route("/admin/devices")
def devices():
    if "user" not in session:
        return redirect(url_for("login"))
    
    """Render the devices page with search functionality."""
    return render_template("devices.html", options=santa_options)

@app.route("/admin/load_devices", methods=["GET"])
def load_devices():
    if "user" not in session:
        return redirect(url_for("login"))

    """Fetch paginated devices with optional Hostname search."""
    collection = db["devices"]
    page_size = 25
    cursor = request.args.get("cursor", None)
    hostname_filter = request.args.get("hostname", None)

    logger.info("Fetching devices", extra={"fields": {
        "cursor": cursor,
        "hostname": hostname_filter,
    }})

    query = {}

    if hostname_filter:
        query["hostname"] = {"$regex": f"^{re.escape(hostname_filter)}", "$options": "i"}

    if cursor:
        query["_id"] = {"$gt": cursor}

    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    data = [{"id": str(doc["_id"]), **doc} for doc in docs]

    next_cursor = data[-1]["id"] if data else None

    logger.info("Returning devices", extra={"fields": {
        "count": len(data),
        "next_cursor": next_cursor,
    }})

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route('/admin/devices/<identifier>')
def device_details(identifier):
    logger.info("Fetching device details", extra={"fields": {"identifier": identifier}})
    if "user" not in session:
        return redirect(url_for("login"))

    collection = db["devices"]
    device = collection.find_one({"identifier": identifier})

    if not device:
        abort(404, description="Device not found")

    if device:
        device["identifier"] = str(device["_id"])
        return jsonify(device)

    return jsonify({"error": "Rule not found"}), 404

@app.route("/admin/save_device", methods=["POST"])
def save_device():
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    identifier = data.get("identifier")

    if not identifier:
        return jsonify({"success": False, "message": "Identifier is required"}), 400

    try:
        client_mode = int(data.get("client_mode"))
    except (TypeError, ValueError):
        client_mode = None
    if client_mode not in santa_options.CLIENT_MODES:
        return jsonify({"success": False, "message": "Unknown client mode"}), 400

    try:
        save_device_db(identifier, {"client_mode": client_mode})
        return jsonify({"success": True, "message": "Device updated", "doc_id": identifier}), 200

    except Exception as e:
        logger.exception("Failed to save device", extra={"fields": {"identifier": identifier}})
        return jsonify({"success": False, "message": str(e)}), 500

@app.route("/admin/devices/<identifier>/clean_sync", methods=["POST"])
def request_clean_sync(identifier):
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    sync_type = str(data.get("sync_type") or "").upper()

    if sync_type and sync_type not in santa_options.REQUESTED_SYNC_TYPES:
        return jsonify({"success": False, "message": "Unknown sync type"}), 400

    # Preflight hands this out on the next sync and postflight clears it once
    # a clean sync has completed. An empty value cancels a pending request.
    update = {"$set": {"requested_sync_type": sync_type}} if sync_type else {"$unset": {"requested_sync_type": ""}}
    result = db["devices"].update_one({"_id": identifier}, update)

    if result.matched_count == 0:
        return jsonify({"success": False, "message": "Device not found"}), 404

    message = f"{sync_type} requested" if sync_type else "Clean sync request cancelled"
    return jsonify({"success": True, "message": message}), 200

@app.route("/admin/devices/<identifier>/settings", methods=["POST"])
def save_device_settings(identifier):
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    try:
        set_fields, unset_fields = parse_settings(data, allow_default_client_mode=False)
    except ValueError as e:
        return jsonify({"success": False, "message": str(e)}), 400

    update = mongo_update(set_fields, unset_fields, prefix="settings.")
    result = db["devices"].update_one({"_id": identifier}, update)

    if result.matched_count == 0:
        return jsonify({"success": False, "message": "Device not found"}), 404

    return jsonify({"success": True, "message": "Device settings saved"}), 200

# Settings
@app.route("/admin/settings", methods=["GET"])
def settings_page():
    if "user" not in session:
        return redirect(url_for("login"))

    settings = db["settings"].find_one({"_id": "global"}) or {}
    return render_template(
        "settings.html",
        settings=settings,
        options=santa_options,
        malwarebazaar=malwarebazaar_status(),
    )

@app.route("/admin/settings", methods=["POST"])
def save_settings():
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    try:
        set_fields, unset_fields = parse_settings(data)
    except ValueError as e:
        return jsonify({"success": False, "message": str(e)}), 400

    update = mongo_update(set_fields, unset_fields)
    if update:
        db["settings"].update_one({"_id": "global"}, update, upsert=True)

    logger.info("Saved global settings", extra={"fields": {"set": sorted(set_fields), "unset": sorted(unset_fields)}})
    return jsonify({"success": True, "message": "Settings saved"}), 200

# MalwareBazaar
#
# While an API key is saved in the settings document below, the
# worker service keeps a local copy of MalwareBazaar's Mach-O samples
# and flags events for listed binaries by adding a "malwarebazaar" field. Flagged events wait on the Review page
# until an admin blocks or dismisses them.
MALWAREBAZAAR_SETTINGS_ID = "malwarebazaar"

FLAGGED_EVENTS_QUERY = {
    "malwarebazaar": {"$exists": True},
    "malwarebazaar.reviewed": {"$ne": True},
}


def malwarebazaar_status():
    """The feed's state for the settings page. Never includes the key."""
    doc = db["settings"].find_one({"_id": MALWAREBAZAAR_SETTINGS_ID}) or {}
    return {
        "enabled": bool(doc.get("api_key")),
        "last_run_at": doc.get("last_run_at"),
        "last_success_at": doc.get("last_success_at"),
        "last_error": doc.get("last_error"),
        "list_refreshed_at": doc.get("list_refreshed_at"),
        "list_size": doc.get("list_size"),
    }


@app.route("/admin/settings/malwarebazaar", methods=["POST"])
def save_malwarebazaar_settings():
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    try:
        api_key = parse_malwarebazaar_key(data)
    except ValueError as e:
        return jsonify({"success": False, "message": str(e)}), 400

    if api_key is None:
        # The saved list is downloaded again in full if the feed is turned
        # back on.
        db["settings"].delete_one({"_id": MALWAREBAZAAR_SETTINGS_ID})
        db["malwarebazaar_samples"].drop()
        logger.info("Turned off the MalwareBazaar feed")
        return jsonify({"success": True, "message": "MalwareBazaar feed turned off"}), 200

    db["settings"].update_one(
        {"_id": MALWAREBAZAAR_SETTINGS_ID},
        {"$set": {"api_key": api_key}, "$unset": {"last_error": ""}},
        upsert=True,
    )
    logger.info("Saved the MalwareBazaar API key")
    return jsonify({"success": True, "message": "MalwareBazaar feed turned on"}), 200


@app.route("/admin/malwarebazaar/<filesha256>/review", methods=["POST"])
def review_flagged_event(filesha256):
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    block = data.get("block") is True

    event = db["events"].find_one({"_id": filesha256, "malwarebazaar": {"$exists": True}})
    if not event:
        return jsonify({"success": False, "message": "Flagged event not found"}), 404

    now = datetime.now(timezone.utc)
    if block:
        signature = event["malwarebazaar"].get("signature") or "unknown family"
        set_fields, unset_fields = normalize_rule({
            "identifier": filesha256,
            "evaluation": "BINARY",
            "action": "BLOCKLIST",
            "scope": "global",
            "custom_message": f"Known malware ({signature}, reported to MalwareBazaar)",
        })
        set_fields["last_updated"] = now
        db["rules"].update_one({"_id": filesha256}, mongo_update(set_fields, unset_fields), upsert=True)

    db["events"].update_one(
        {"_id": filesha256},
        {"$set": {
            "malwarebazaar.reviewed": True,
            "malwarebazaar.reviewed_at": now,
            "malwarebazaar.reviewed_by": session["user"].get("email"),
            "malwarebazaar.blocked": block,
        }},
    )
    logger.info("Reviewed flagged binary", extra={"fields": {"file_sha256": filesha256, "blocked": block}})
    message = "Blocked on all devices" if block else "Dismissed"
    return jsonify({"success": True, "message": message}), 200


# Events
@app.route("/admin/events")
def events():
    if "user" not in session:
        return redirect(url_for("login"))
    
    """Render the template for infinite scroll."""
    return render_template("events.html", decisions=santa_options.DECISIONS)

@app.route("/admin/load_events", methods=["GET"])
def load_events():
    if "user" not in session:
        return redirect(url_for("login"))

    """Fetch paginated events with optional search and filters."""
    collection = db["events"]
    page_size = 25
    cursor = request.args.get("cursor", None)
    decision_filter = request.args.get("decision", None)
    virustotal_filter = request.args.get("virustotal_result", None)
    file_sha256_query = request.args.get("file_sha256", None)
    file_name_query = request.args.get("fileName", None)

    logger.info("Fetching events", extra={"fields": {
        "cursor": cursor,
        "decision": decision_filter,
        "virustotal_result": virustotal_filter,
        "file_sha256": file_sha256_query,
        "file_name": file_name_query,
    }})

    query = {}

    if decision_filter:
        query["decision"] = decision_filter

    if virustotal_filter:
        query["virustotal_result"] = int(virustotal_filter)

    if file_sha256_query:
        query["file_sha256"] = {"$regex": f"^{re.escape(file_sha256_query)}", "$options": "i"}

    if file_name_query:
        query["file_name"] = {"$regex": f"^{re.escape(file_name_query)}", "$options": "i"}

    if cursor:
        query["_id"] = {"$gt": cursor}

    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    data = [{"id": str(doc["_id"]), **doc} for doc in docs]

    next_cursor = data[-1]["id"] if data else None

    logger.info("Returning events", extra={"fields": {
        "count": len(data),
        "next_cursor": next_cursor,
    }})

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route('/admin/events/<filesha256>')
def event_details(filesha256):
    if "user" not in session:
        return redirect(url_for("login"))

    """
    Fetches details of an event based on FileSha256.
    If virustotal_result is missing, it fetches and updates it.
    """
    collection = db["events"]
    event = collection.find_one({"file_sha256": filesha256})

    if not event:
        abort(404, description="Event not found")

    event["id"] = str(event["_id"])

    if 'virustotal_result' not in event or event['virustotal_result'] == 0:
        logger.info("Getting VirusTotal scan result", extra={"fields": {"file_sha256": filesha256}})
        virustotal_result = get_vt_result(filesha256)
        if virustotal_result is not None:
            event['virustotal_result'] = virustotal_result
            logger.info("Updated VirusTotal scan result", extra={"fields": {
                "file_sha256": filesha256,
                "virustotal_result": virustotal_result,
            }})
            save_binary_db(filesha256, event)

    return render_template('event_details.html', data=event)

@app.route('/admin/events/<filesha256>/delete', methods=['DELETE'])
def delete_event(filesha256):
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    collection = db["events"]
    existing_doc = collection.find_one({"file_sha256": filesha256})

    if not existing_doc:
        return jsonify({"success": False, "message": "Document not found"}), 404

    collection.delete_one({"file_sha256": filesha256})

    return jsonify({"success": True, "message": f"Event {filesha256} deleted successfully."}), 200

# Rules
@app.route("/admin/rules")
def rules():
    if "user" not in session:
        return redirect(url_for("login"))

    return render_template("rules.html", options=santa_options)

@app.route("/admin/load_rules", methods=["GET"])
def load_rules():
    if "user" not in session:
        return redirect(url_for("login"))

    """Fetch paginated rules with optional search and filters."""
    collection = db["rules"]
    page_size = 25
    cursor = request.args.get("cursor", None)
    identifier_filter = request.args.get("identifier", None)
    rule_type_filter = request.args.get("rule_type", None)
    policy_filter = request.args.get("policy", None)
    scope_filter = request.args.get("scope", None)

    logger.info("Fetching rules", extra={"fields": {
        "cursor": cursor,
        "identifier": identifier_filter,
        "rule_type": rule_type_filter,
        "policy": policy_filter,
        "scope": scope_filter,
    }})

    query = {"deleted": {"$ne": True}}

    if identifier_filter:
        query["identifier"] = {"$regex": f"^{re.escape(identifier_filter)}", "$options": "i"}

    if rule_type_filter:
        query["rule_type"] = rule_type_filter

    if policy_filter:
        query["policy"] = policy_filter

    if scope_filter:
        query["scope"] = scope_filter

    if cursor:
        query["_id"] = {"$gt": cursor}

    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    data = [{"id": str(doc["_id"]), **doc} for doc in docs]

    next_cursor = data[-1]["id"] if data else None

    logger.info("Returning rules", extra={"fields": {
        "count": len(data),
        "next_cursor": next_cursor,
    }})

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route("/admin/rules/<rule_id>", methods=["GET"])
def get_rule(rule_id):
    if "user" not in session:
        return redirect(url_for("login"))

    rule = db["rules"].find_one({"_id": rule_id})

    if rule:
        rule["_id"] = str(rule["_id"])
        return jsonify(rule)

    return jsonify({"error": "Rule not found"}), 404

@app.route("/admin/save_rule", methods=["POST"])
def save_rule():
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    try:
        set_fields, unset_fields = normalize_rule(data)
    except ValueError as e:
        return jsonify({"success": False, "message": str(e)}), 400

    identifier = set_fields["identifier"]
    try:
        set_fields["last_updated"] = datetime.now(timezone.utc)
        db["rules"].update_one(
            {"_id": identifier},
            mongo_update(set_fields, unset_fields),
            upsert=True,
        )
        logger.info("Saved rule", extra={"fields": {"identifier": identifier}})
        return jsonify({"success": True, "message": "Rule saved", "doc_id": identifier}), 200

    except Exception as e:
        logger.exception("Failed to save rule", extra={"fields": {"identifier": identifier}})
        return jsonify({"success": False, "message": str(e)}), 500

@app.route("/admin/rules/<rule_id>/delete", methods=["POST"])
def delete_rule(rule_id):
    if "user" not in session:
        return jsonify({"success": False, "message": "Unauthorized"}), 401

    # Keep a tombstone so devices that already have the rule receive a REMOVE
    # on their next sync. Full syncs skip deleted rules.
    now = datetime.now(timezone.utc)
    result = db["rules"].update_one(
        {"_id": rule_id},
        {"$set": {"deleted": True, "deleted_at": now, "last_updated": now}},
    )

    if result.matched_count == 0:
        return jsonify({"success": False, "message": "Rule not found"}), 404

    logger.info("Deleted rule", extra={"fields": {"identifier": rule_id}})
    return jsonify({"success": True, "message": "Rule deleted"}), 200

# File access events
@app.route("/admin/file_access")
def file_access():
    if "user" not in session:
        return redirect(url_for("login"))

    return render_template("file_access.html")

@app.route("/admin/load_file_access", methods=["GET"])
def load_file_access():
    if "user" not in session:
        return jsonify({"message": "Unauthorized"}), 401

    query = {}
    rule_name = request.args.get("rule_name")
    machine_id = request.args.get("machine_id")
    if rule_name:
        query["rule_name"] = rule_name
    if machine_id:
        query["machine_id"] = machine_id

    return jsonify(load_page("file_access_events", query, request.args.get("cursor")))

# Standalone mode approvals
@app.route("/admin/standalone")
def standalone():
    if "user" not in session:
        return redirect(url_for("login"))

    return render_template("standalone.html")

@app.route("/admin/load_standalone", methods=["GET"])
def load_standalone():
    if "user" not in session:
        return jsonify({"message": "Unauthorized"}), 401

    query = {}
    machine_id = request.args.get("machine_id")
    if machine_id:
        query["machine_id"] = machine_id

    return jsonify(load_page("standalone_approvals", query, request.args.get("cursor")))

def load_page(collection_name, query, cursor, page_size=25):
    """Return one page of a log-style collection, newest first, with the
    device hostname attached to each entry."""
    if cursor:
        try:
            query["_id"] = {"$lt": ObjectId(cursor)}
        except Exception:
            return {"data": [], "next_cursor": None}

    docs = list(db[collection_name].find(query).sort("_id", -1).limit(page_size))

    machine_ids = {doc.get("machine_id") for doc in docs if doc.get("machine_id")}
    hostnames = {
        device["_id"]: device.get("hostname", "")
        for device in db["devices"].find({"_id": {"$in": list(machine_ids)}}, {"hostname": 1})
    }

    data = []
    for doc in docs:
        doc["id"] = str(doc.pop("_id"))
        doc["hostname"] = hostnames.get(doc.get("machine_id"), "")
        data.append(doc)

    next_cursor = data[-1]["id"] if len(data) == page_size else None
    return {"data": data, "next_cursor": next_cursor}

# Rules Review
@app.context_processor
def inject_review_count():
    if "user" not in session:
        return dict(review_count=0)

    count = db.rules.count_documents({
        "policy": "ALLOWLIST",
        "$or": [
            {"converted": True},
            {"scope": "machine"}
        ],
        "reviewed": {"$ne": True}
    })
    count += db.events.count_documents(FLAGGED_EVENTS_QUERY)

    return dict(review_count=count)


@app.route("/admin/review")
def review():
    if "user" not in session:
        return redirect(url_for("login"))

    rules_collection = db["rules"]

    rules = list(
        rules_collection.find({
            "policy": "ALLOWLIST",
            "$and": [
                {
                    "$or": [
                        {"converted": True},
                        {"scope": "machine"}
                    ]
                },
                {
                    "$or": [
                        {"reviewed": {"$ne": True}},
                        {"reviewed": {"$exists": False}}
                    ]
                }
            ]
        })
    )

    flagged_events = list(db["events"].find(FLAGGED_EVENTS_QUERY).sort("malwarebazaar.flagged_at", -1))

    return render_template(
        "review.html",
        rules=rules,
        flagged_events=flagged_events,
    )


@app.route("/admin/approve_rule", methods=["POST"])
def approve_rule():
    if "user" not in session:
        return jsonify({"message": "Unauthorized"}), 401

    data = request.get_json(silent=True) or {}
    rule_id = data.get("id")
    if not rule_id:
        return jsonify({"message": "Missing id"}), 400

    result = db.rules.update_one(
        {"_id": rule_id},
        {"$set": {"reviewed": True}}
    )

    if result.matched_count == 0:
        return jsonify({"message": "Rule not found"}), 404

    return jsonify({"message": "Rule marked as reviewed"})


# Health check endpoint
@app.route("/health")
def status():
    return {'status': 'ok'}, 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
