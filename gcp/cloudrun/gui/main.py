import os
import requests
import base64
import uuid
from datetime import datetime, timezone

from flask import Flask, render_template, send_from_directory, abort, jsonify, request, redirect, url_for, session
from flask_session import Session
import msal
from google.cloud import firestore
from pymongo import MongoClient

FIRESTORE_DATABASE = os.getenv("FIRESTORE_DATABASE")
firestore_client = firestore.Client(database=FIRESTORE_DATABASE)

class FirestoreSessionInterface:
    def __init__(self):
        self.collection = firestore_client.collection("sessions")  # Firestore collection for sessions

    def open_session(self, request):
        session_id = request.cookies.get("session_id")
        if not session_id:
            session_id = str(uuid.uuid4())  # Generate new session ID

        doc = self.collection.document(session_id).get()
        return doc.to_dict() if doc.exists else {}

    def save_session(self, response):
        session_id = request.cookies.get("session_id", str(uuid.uuid4()))
        self.collection.document(session_id).set(dict(session))
        response.set_cookie("session_id", session_id, httponly=True, secure=True)

app = Flask(__name__)

app.config["SESSION_TYPE"] = "filesystem"
app.session_interface = FirestoreSessionInterface()
app.secret_key = os.getenv("FLASK_SECRET_KEY")

Session(app)

CLIENT_ID = os.getenv("MSAL_CLIENT_ID")
CLIENT_SECRET = os.getenv("MSAL_CLIENT_SECRET")
TENANT_ID = os.getenv("MSAL_TENANT_ID")
AUTHORITY = f"https://login.microsoftonline.com/{TENANT_ID}"
REDIRECT_URI = os.getenv("MSAL_REDIRECT_URI")
SCOPES = ["User.Read"]

msal_app = msal.ConfidentialClientApplication(
    CLIENT_ID, authority=AUTHORITY, client_credential=CLIENT_SECRET
)

VT_API_KEY = os.environ.get('VT_API_KEY')

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
    print(result)
    return data

def save_rule_db(identifier, data):
    collection = db["rules"]
    data["last_updated"] = datetime.now(timezone.utc)

    result = collection.update_one(
        {"_id": identifier},
        {"$set": data},
        upsert=True
    )
    print(result)
    return data

def get_mongodb_count(collection_name):
    collection = db[collection_name]
    return collection.count_documents({})

@app.route('/site.webmanifest')
def manifest():
    return send_from_directory('static/manifest', 'site.webmanifest', mimetype='application/manifest+json')


@app.route("/login")
def login():
    auth_url = msal_app.get_authorization_request_url(SCOPES, redirect_uri=REDIRECT_URI)
    return redirect(auth_url)

@app.route("/token")
def token():
    """Handles login callback, gets user profile and profile picture."""
    if "error" in request.args:
        return f"Error: {request.args['error']} - {request.args['error_description']}"

    code = request.args.get("code")
    if not code:
        return "No authorization code received"

    result = msal_app.acquire_token_by_authorization_code(code, scopes=SCOPES, redirect_uri=REDIRECT_URI)

    if "access_token" in result:
        headers = {"Authorization": f"Bearer {result['access_token']}"}
        
        # Fetch User Profile
        user_info = requests.get("https://graph.microsoft.com/v1.0/me", headers=headers).json()
        
        # Fetch Profile Picture
        profile_pic_url = "https://graph.microsoft.com/v1.0/me/photo/$value"
        profile_pic = None

        profile_pic_response = requests.get(profile_pic_url, headers=headers)
        if profile_pic_response.status_code == 200:
            profile_pic = base64.b64encode(profile_pic_response.content).decode('utf-8')

        # Store user info in session
        session["user"] = {
            "name": user_info.get("displayName", "User"),
            "email": user_info.get("mail", user_info.get("userPrincipalName")),
            "profile_pic": profile_pic,  # Store Base64-encoded image
        }
        session["access_token"] = result.get("access_token")

        return redirect(url_for("index"))

    return f"Authentication failed: {result.get('error_description')}"


@app.route("/logout")
def logout():
    session.clear()
    return redirect(url_for("index"))

@app.route("/")
def index():
    if "user" not in session:
        return redirect(url_for("login"))
    
    collections_to_count = {
        "devices": {"collection": "devices"},
        "rules": {"collection": "rules"},
        "events": {"collection": "events"},
    }

    counts = {
        db_key: get_mongodb_count(db_info["collection"])
        for db_key, db_info in collections_to_count.items()
    }

    return render_template("index.html", counts=counts)

# Devices
@app.route("/devices")
def devices():
    if "user" not in session:
        return redirect(url_for("login"))
    
    """Render the devices page with search functionality."""
    return render_template("devices.html")

@app.route("/load_devices", methods=["GET"])
def load_devices():
    if "user" not in session:
        return redirect(url_for("login"))

    """Fetch paginated devices with optional Hostname search."""
    collection = db["devices"]
    page_size = 25
    cursor = request.args.get("cursor", None)
    hostname_filter = request.args.get("hostname", None)

    print(f"Fetching devices with cursor: {cursor}, Hostname: {hostname_filter}")

    query = {}

    if hostname_filter:
        query["hostname"] = {"$regex": f"^{hostname_filter}", "$options": "i"}

    if cursor:
        query["_id"] = {"$gt": cursor}

    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    data = [{"id": str(doc["_id"]), **doc} for doc in docs]

    next_cursor = data[-1]["id"] if data else None

    print(f"Returning {len(data)} devices, Next cursor: {next_cursor}")

    return jsonify({"data": data, "next_cursor": next_cursor})

# Events
@app.route("/events")
def events():
    if "user" not in session:
        return redirect(url_for("login"))
    
    """Render the template for infinite scroll."""
    return render_template("events.html")

@app.route("/load_events", methods=["GET"])
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

    print(f"Fetching events with cursor: {cursor}, Filters: Decision={decision_filter}, VirusTotal={virustotal_filter}, FileSha256={file_sha256_query}, FileName={file_name_query}")

    query = {}

    if decision_filter:
        query["decision"] = decision_filter

    if virustotal_filter:
        query["virustotal_result"] = int(virustotal_filter)

    if file_sha256_query:
        query["file_sha256"] = {"$regex": f"^{file_sha256_query}", "$options": "i"}

    if file_name_query:
        query["file_name"] = {"$regex": f"^{file_name_query}", "$options": "i"}

    if cursor:
        query["_id"] = {"$gt": cursor}

    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    data = [{"id": str(doc["_id"]), **doc} for doc in docs]

    next_cursor = data[-1]["id"] if data else None

    print(f"Returning {len(data)} events, Next cursor: {next_cursor}")

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route('/events/<filesha256>')
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
        print("Getting Virus Total Scan Result for %s" % filesha256)
        virustotal_result = get_vt_result(filesha256)
        if virustotal_result is not None:
            event['virustotal_result'] = virustotal_result
            print(event)
            save_binary_db(filesha256, event)

    return render_template('event_details.html', data=event)

@app.route('/events/<filesha256>/delete', methods=['DELETE'])
def delete_event(filesha256):
    if "user" not in session:
        return redirect(url_for("login"))

    """
    Deletes an event from MongoDB based on FileSha256.
    """
    collection = db["events"]
    existing_doc = collection.find_one({"FileSha256": filesha256})
    
    if not existing_doc:
        abort(404, description="Document not found")

    collection.delete_one({"FileSha256": filesha256})

    return jsonify({"success": True, "message": f"Event {filesha256} deleted successfully."}), 200

# Rules
@app.route("/rules")
def rules():
    if "user" not in session:
        return redirect(url_for("login"))

    return render_template("rules.html")

@app.route("/load_rules", methods=["GET"])
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

    print(f"Fetching rules with cursor: {cursor}, Filters: identifier={identifier_filter}, rule_type={rule_type_filter}, policy={policy_filter}, scope={scope_filter}")

    query = {}

    if identifier_filter:
        query["identifier"] = {"$regex": f"^{identifier_filter}", "$options": "i"}

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

    print(f"Returning {len(data)} rules, Next cursor: {next_cursor}")

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route("/rules/<rule_id>", methods=["GET"])
def get_rule(rule_id):
    if "user" not in session:
        return redirect(url_for("login"))

    rule = db["rules"].find_one({"_id": rule_id})

    if rule:
        rule["_id"] = str(rule["_id"])
        return jsonify(rule)

    return jsonify({"error": "Rule not found"}), 404

@app.route("/save_rule", methods=["POST"])
def save_rule():
    if "user" not in session:
        return redirect(url_for("login"))

    try:
        data = request.json
        evaluation = data.get("evaluation")
        identifier = data.get("identifier")
        action = data.get("action")
        scope = data.get("scope")
        custom_message = data.get("custom_message", "")

        if not evaluation or not identifier:
            return jsonify({"success": False, "message": "Evaluation and Identifier are required"}), 400

        rule_data = {
            "rule_type": evaluation,
            "identifier": identifier,
            "policy": action,
            "scope": scope,
            "custom_msg": custom_message
        }

        save_rule_db(identifier, rule_data)

        return jsonify({"success": True, "message": "Rule saved", "doc_id": identifier}), 200

    except Exception as e:
        return jsonify({"success": False, "message": str(e)}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)