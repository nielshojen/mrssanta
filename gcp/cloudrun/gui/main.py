import os
import requests
from datetime import datetime, timezone

from flask import Flask, render_template, send_from_directory, abort, jsonify, request
#from google.cloud import firestore
from pymongo import MongoClient
from bson import ObjectId

app = Flask(__name__)

VT_API_KEY = os.environ.get('VT_API_KEY')
#DATABASE_NAME = os.getenv("FIRESTORE_DATABASE", "default")

# MongoDB connection URI (Update this with your connection details)
MONGO_URI = os.getenv("MONGO_URI")

# Connect to MongoDB
client = MongoClient(MONGO_URI)
db = client["mrssanta"]

#db = firestore.Client(database=DATABASE_NAME)

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

@app.route("/")
def index():
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
    """Render the devices page with search functionality."""
    return render_template("devices.html")

@app.route("/load_devices", methods=["GET"])
def load_devices():
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
    """Render the template for infinite scroll."""
    return render_template("events.html")

@app.route("/load_events", methods=["GET"])
def load_events():
    """Fetch paginated events for infinite scrolling with optional filtering and partial search."""
    collection = db["events"]
    page_size = 25
    cursor = request.args.get("cursor", None)  # For pagination
    decision_filter = request.args.get("decision", None)
    virustotalresult_filter = request.args.get("virustotalresult", None)
    file_sha256_query = request.args.get("file_sha256", None)
    file_name_query = request.args.get("file_name", None)

    print(f"Fetching events with cursor: {cursor}, decision Filter: {decision_filter}, virustotalresult Filter: {virustotalresult_filter}, file_sha256: {file_sha256_query}, file_name: {file_name_query}")

    query = {}

    # Apply filters
    if decision_filter:
        query["decision"] = decision_filter  # Exact match

    if virustotalresult_filter:
        try:
            query["virustotalresult"] = int(virustotalresult_filter)  # Convert to int
        except ValueError:
            return jsonify({"error": "Invalid virustotalresult value"}), 400

    if file_sha256_query:
        query["file_sha256"] = {"$regex": f"^{file_sha256_query}", "$options": "i"}  # Partial match

    if file_name_query:
        query["file_name"] = {"$regex": f"^{file_name_query}", "$options": "i"}  # Case-insensitive search

    # Pagination: If a cursor is provided, fetch events **after** the given `_id`
    if cursor:
        try:
            query["_id"] = {"$gt": ObjectId(cursor)}  # Convert cursor to ObjectId
        except:
            return jsonify({"error": "Invalid cursor"}), 400

    # Fetch events
    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    # Convert MongoDB documents to JSON-serializable format
    data = []
    for doc in docs:
        doc["_id"] = str(doc["_id"])  # Convert `_id` to string
        data.append(doc)

    # Determine the next cursor
    next_cursor = data[-1]["_id"] if data else None

    print(f"Returning {len(data)} events, Next cursor: {next_cursor}")

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route('/events/<filesha256>')
def event_details(filesha256):
    """
    Fetches details of an event based on FileSha256.
    If VirusTotalResult is missing, it fetches and updates it.
    """
    collection = db["events"]
    # Find the event where 'FileSha256' matches (limit 1)
    event = collection.find_one({"file_sha256": filesha256})

    if not event:
        abort(404, description="Event not found")

    # Convert MongoDB `_id` to string for rendering
    event["id"] = str(event["_id"])

    # Fetch VirusTotal result if missing or 0
    if 'virustotalresult' not in event or event['virustotalresult'] == 0:
        virustotalresult = get_vt_result(filesha256)
        if virustotalresult is not None:
            event['virustotalresult'] = virustotalresult
            save_binary_db(filesha256, event)  # Save back to MongoDB

    return render_template('event_details.html', data=event)

@app.route('/events/<filesha256>/delete', methods=['DELETE'])
def delete_event(filesha256):
    """
    Deletes an event from MongoDB based on FileSha256.
    """
    collection = db["events"]
    # Check if the document exists
    existing_doc = collection.find_one({"FileSha256": filesha256})
    
    if not existing_doc:
        abort(404, description="Document not found")

    # Delete the document
    collection.delete_one({"FileSha256": filesha256})

    return jsonify({"success": True, "message": f"Event {filesha256} deleted successfully."}), 200

# Rules
@app.route("/rules")
def rules():
    return render_template("rules.html")  # Renders the template with filters

@app.route("/load_rules", methods=["GET"])
def load_rules():
    """Fetch paginated rules with optional search and filters."""
    collection = db["rules"]
    page_size = 25
    cursor = request.args.get("cursor", None)  # For pagination
    identifier_filter = request.args.get("identifier", None)  # Search by Identifier
    rule_type_filter = request.args.get("rule_type", None)  # Filter by RuleType
    policy_filter = request.args.get("policy", None)  # Filter by Policy
    scope_filter = request.args.get("scope", None)  # Filter by Scope

    print(f"Fetching rules with cursor: {cursor}, Filters: identifier={identifier_filter}, rule_type={rule_type_filter}, policy={policy_filter}, scope={scope_filter}")

    query = {}

    # Apply filters
    if identifier_filter:
        query["identifier"] = {"$regex": f"^{identifier_filter}", "$options": "i"}  # Case-insensitive partial match

    if rule_type_filter:
        query["rule_type"] = rule_type_filter  # Exact match

    if policy_filter:
        query["policy"] = policy_filter  # Exact match

    if scope_filter:
        query["scope"] = scope_filter  # Exact match

    # Pagination: If a cursor is provided, fetch rules **after** the given `_id`
    if cursor:
        query["_id"] = {"$gt": cursor}  # Fetch next rules **greater than** the last `_id`

    # Fetch rules
    docs = list(collection.find(query).sort("_id", 1).limit(page_size))

    # Convert MongoDB documents to JSON-serializable format
    data = [{"id": str(doc["_id"]), **doc} for doc in docs]

    # Determine the next cursor
    next_cursor = data[-1]["id"] if data else None

    print(f"Returning {len(data)} rules, Next cursor: {next_cursor}")

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route("/save_rule", methods=["POST"])
def save_rule():
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
            "RuleType": evaluation,
            "Identifier": identifier,
            "Policy": action,
            "Scope": scope,
            "CustomMessage": custom_message
        }

        save_rule_db(identifier, rule_data)

        return jsonify({"success": True, "message": "Rule saved", "doc_id": identifier}), 200

    except Exception as e:
        return jsonify({"success": False, "message": str(e)}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)