import os
import requests

from flask import Flask, render_template, send_from_directory, abort, jsonify, request
from google.cloud import firestore

app = Flask(__name__)

VT_API_KEY = os.environ.get('VT_API_KEY')
DATABASE_NAME = os.getenv("FIRESTORE_DATABASE", "default")

db = firestore.Client(database=DATABASE_NAME)

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

def save_binary(identifier, data):
    doc_ref = db.collection(f"{DATABASE_NAME}_events").document(identifier)
    data['LastUpdated'] = firestore.SERVER_TIMESTAMP
    doc_ref.set(data, merge=True)
    return data

def get_firestore_count(collection_name):
    count_query = db.collection(collection_name).count()
    count_result = list(count_query.get())
    return count_result[0][0].value if count_result else 0

@app.route('/site.webmanifest')
def manifest():
    return send_from_directory('static/manifest', 'site.webmanifest', mimetype='application/manifest+json')

@app.route("/")
def index():
    collections_to_count = {
        "devices": {"collection": f"{DATABASE_NAME}_devices"},
        "rules": {"collection": f"{DATABASE_NAME}_rules"},
        "events": {"collection": f"{DATABASE_NAME}_events"},
    }

    counts = {
        db_key: get_firestore_count(db_info["collection"])
        for db_key, db_info in collections_to_count.items()
    }

    return render_template("index.html", counts=counts)

# Devices
@app.route("/devices")
def devices():
    docs = db.collection(f"{DATABASE_NAME}_devices").stream()
    data = [doc.to_dict() for doc in docs]

    return render_template("devices.html", data=data)

# Events
@app.route("/events")
def events():
    """Render the template for infinite scroll."""
    return render_template("events.html")

@app.route("/load_events", methods=["GET"])
def load_events():
    """Fetch paginated events for infinite scrolling with optional filtering and search."""
    page_size = 25
    cursor = request.args.get("cursor", None)  # Get cursor for pagination
    decision_filter = request.args.get("decision", None)  # Get Decision filter
    virus_total_filter = request.args.get("virustotal", None)  # Get VirusTotalResult filter
    file_sha256_query = request.args.get("fileSha256", None)  # Get FileSha256 search query
    file_name_query = request.args.get("fileName", None)  # Get FileName search query

    print(f"Fetching events with cursor: {cursor}, Decision Filter: {decision_filter}, VirusTotal Filter: {virus_total_filter}, FileSha256: {file_sha256_query}, FileName: {file_name_query}")  # Debugging

    collection_ref = db.collection(f"{DATABASE_NAME}_events").limit(page_size)

    if decision_filter:
        collection_ref = collection_ref.where("Decision", "==", decision_filter)  # Apply Decision filter

    if virus_total_filter:
        collection_ref = collection_ref.where("VirusTotalResult", "==", int(virus_total_filter))  # Apply VirusTotalResult filter

    if file_sha256_query:
        collection_ref = collection_ref.where("FileSha256", "==", file_sha256_query)  # Apply FileSha256 search

    if file_name_query:
        collection_ref = collection_ref.where("FileName", "==", file_name_query)  # Apply FileName search

    if cursor:
        last_doc_ref = db.collection(f"{DATABASE_NAME}_events").document(cursor)
        last_doc = last_doc_ref.get()
        if last_doc.exists:
            collection_ref = collection_ref.start_after(last_doc)

    docs = list(collection_ref.stream())  
    data = [{"id": doc.id, **doc.to_dict()} for doc in docs]

    next_cursor = data[-1]["id"] if data else None  

    return jsonify({"data": data, "next_cursor": next_cursor})

@app.route('/events/<filesha256>')
def event_details(filesha256):
    events = db.collection(f"{DATABASE_NAME}_events")
    query = events.where('FileSha256', '==', filesha256).limit(1).stream()

    event = None
    for doc in query:
        event = doc.to_dict()
        break

    if not event:
        abort(404, description="Event not found")
    
    if 'VirusTotalResult' not in event or event['VirusTotalResult'] == 0:
        virustotalresult = get_vt_result(filesha256)
        if virustotalresult is not None:
            event['VirusTotalResult'] = virustotalresult
            save_binary(filesha256, event)

    return render_template('event_details.html', data=event)

@app.route('/events/<filesha256>/delete', methods=['DELETE'])
def delete_event(filesha256):
    doc_ref = db.collection(f"{DATABASE_NAME}_events").document(filesha256)
    
    if not doc_ref.get().exists:
        abort(404, description="Document not found")
    
    doc_ref.delete()
    return jsonify({"success": True, "message": f"Event {filesha256} deleted successfully."}), 200

# Rules
@app.route("/rules")
def rules():
    docs = db.collection(f"{DATABASE_NAME}_rules").stream()
    data = [doc.to_dict() for doc in docs]

    return render_template("rules.html", data=data)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)