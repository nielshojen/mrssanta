import functions_framework
import os
import json

from google.cloud import firestore

# Initialize Firestore client with a specific database ID
db = firestore.Client()

request_dict = {
  "machine_id" : None,
  "cursor" : None,
}

def request_handler(d):
    data = {}
    for k, v in d.items():
        if v:
            data[k] = v
    return data

# Function to get rules from Firestore
def get_rules():
    try:
        # Reference to the Firestore collection containing rules
        rules_collection_ref = db.collection('%s_rules' % os.environ.get('DB_PREFIX'))

        # Query Firestore for all documents in the 'rules' collection
        query = rules_collection_ref.get()

        # Initialize a list to store the rules
        rules = []

        # Iterate over the documents and extract rules
        for doc in query:
            # Get dictionary representation of document
            rule_data = doc.to_dict()

            # Filter out fields with None or empty values
            rule_data_filtered = {k: v for k, v in rule_data.items() if v is not None and v != ""}

            # Append filtered rule data to the list of rules
            rules.append(rule_data_filtered)

        return rules
    except Exception as e:
        # Handle exceptions or errors
        print(f"Error getting rules: {e}")
        return None

@functions_framework.http
def ruledownload(request):
    request_json = request.get_json(silent=True)
    request_args = request.args

    return_data = request_dict

    if request_args and "machine_id" in request_args:
        return_data['machine_id'] = request_args.getlist('machine_id')[0]

    print('Request: %s' % request_handler(return_data))
    print(request_json)

    return_data = {}

    if request_json and "cursor" in request_json:
        return_data['cursor'] = request_json['cursor']

    rules = []

    for rule in get_rules():
        print(rule)
        rules.append(request_handler(rule))

    return_data['rules'] = rules

    return json.dumps(return_data), 200, {'Content-Type': 'application/json'}