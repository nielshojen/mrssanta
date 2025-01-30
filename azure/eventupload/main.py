import functions_framework
import os
import json

from google.cloud import firestore

# Initialize Firestore client with a specific database ID
db = firestore.Client()

response = {
 "event_upload_bundle_binaries": None,
}

def sanitize_event(data):

    if 'executing_user' in data:
        data.pop('executing_user')

    if 'execution_time' in data:
        data.pop('execution_time')

    if 'logged_in_users' in data:
        data.pop('logged_in_users')

    if 'current_sessions' in data:
        data.pop('current_sessions')

    if 'decision' in data:
        data.pop('decision')

    if 'file_bundle_path' in data:
        data.pop('file_bundle_path')

    if 'file_bundle_hash_millis' in data:
        data.pop('file_bundle_hash_millis')

    if 'pid' in data:
        data.pop('pid')

    if 'ppid' in data:
        data.pop('ppid')

    if 'parent_name' in data:
        data.pop('parent_name')

    if 'quarantine_timestamp' in data:
        data.pop('quarantine_timestamp')

    if 'quarantine_agent_bundle_id' in data:
        data.pop('quarantine_agent_bundle_id')
    
    return data

def save_event(data):

    file_sha256 = data['file_sha256']

    # Define Firestore document path
    doc_ref = db.collection('%s_events' % os.environ.get('DB_PREFIX')).document(file_sha256)

    # Add Timestamp
    data['last_updated'] = firestore.SERVER_TIMESTAMP

    # Set data in Firestore with merge=True
    doc_ref.set(data, merge=True)

    
    return data


@functions_framework.http
def eventupload(request):
    request_json = request.get_json(silent=True)
    request_args = request.args

    if request_args and "machine_id" in request_args:
        machine_id = request_args['machine_id']
        print('Request from: %s' % machine_id)

    return_data = response
    
    binaries = []

    if request_json and "events" in request_json:
        for event in request_json['events']:
            if 'decision' in event and event['decision'] != 'ALLOW_UNKNOWN':
                print(event)
            sanitized_event = sanitize_event(event)
            if sanitized_event:
                save_event(sanitized_event)
            if "file_sha256" in event:
                binaries.append(event['file_sha256'])

    return_data['event_upload_bundle_binaries'] = binaries

    return json.dumps(return_data), 200, {'Content-Type': 'application/json'}