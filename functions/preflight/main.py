import functions_framework
import os
import json

from google.cloud import firestore

# Initialize Firestore client with a specific database ID
db = firestore.Client()

request_dict = {
  "machine_id" : None,
  "serial_num" : None,
  "hostname" : None,
  "os_version" : None,
  "os_build" : None,
  "model_identifier" : None,
  "santa_version" : None,
  "primary_user" : None,
  "binary_rule_count" : None,
  "certificate_rule_count" : None,
  "compiler_rule_count" : None,
  "transitive_rule_count" : None,
  "teamid_rule_count" : None,
  "signingid_rule_count" : None,
  "cdhash_rule_count" : None,
  "client_mode" : None,
  "request_clean_sync": None
}

def request_handler(d):
    data = request_dict
    for k, v in d.items():
        if v:
            data[k] = v
    return data

@functions_framework.http
def preflight(request):
    request_json = request.get_json(silent=True)
    request_args = request.args


    if request_json is not None:
        request_data = request_handler(request_json)
    else:
        request_data = request_dict

    if request_args and "machine_id" in request_args:
        request_data['machine_id'] = request_args.getlist('machine_id')[0]
        machine_id = request_args.getlist('machine_id')[0]
    else:
        machine_id = None

    print('Request: %s' % request_handler(request_data))

    if machine_id is None:
        return 'Error: Missing machine_id in request body', 400
    try:
        # Define Firestore document path
        doc_ref = db.collection('%s_devices' % os.environ.get('DB_PREFIX')).document(machine_id)

        # Add Timestamp
        request_data['last_updated'] = firestore.SERVER_TIMESTAMP

        # Set data in Firestore with merge=True
        doc_ref.set(request_data, merge=True)

        return_data = {}

        return_data['batch_size'] = 100
        return_data['full_sync_interval'] = 600
        return_data['client_mode'] = 'MONITOR'
        if 'request_clean_sync' in request_data and request_data['request_clean_sync'] is True:
            return_data['sync_type'] = 'clean_all'
        else:
            return_data['sync_type'] = 'normal'
        return_data['bundles_enabled'] = True
        return_data['enable_transitive_rules'] = False

        response = return_data
        print('Resonse: %s' % response)

        return json.dumps(response), 200, {'Content-Type': 'application/json'}
    except Exception as e:
        return f'Error storing data in Firestore: {str(e)}', 500