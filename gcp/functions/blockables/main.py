import functions_framework
from flask import render_template, jsonify, send_from_directory
import os
import json
import requests

from google.cloud import firestore

vt_api_key = os.environ.get('VT_API_KEY')
vote_threshold = os.environ.get('VOTE_THRESHOLD')

# Initialize Firestore client with a specific database ID
db = firestore.Client(database=os.environ.get('FIRESTORE_DATABASE'))

STATIC_FOLDER = os.path.join(os.getcwd(), "static")

def get_vt_result(file_hash):
    check_failed = 0

    url = "https://www.virustotal.com/api/v3/files/%s" % file_hash

    headers = {"accept": "application/json", "x-apikey": vt_api_key}

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

def get_device(identifier):

    # Reference to the device document
    device_ref = db.collection('%s_devices' % os.environ.get('DB_PREFIX')).document(identifier)

    # Get the device document
    device = device_ref.get()

    if device.exists:
        return device.to_dict()
    else:
        return None

def get_binary(identifier):

    # Reference to the binary document
    binary_ref = db.collection('%s_events' % os.environ.get('DB_PREFIX')).document(identifier)

    # Get the binary document
    binary = binary_ref.get()

    if binary.exists:
        data =  binary.to_dict()
        if 'VirusTotalResult' not in data or data['VirusTotalResult'] == 0:
            virustotalresult = get_vt_result(data['FileSha256'])
            if virustotalresult is not None:
                data['VirusTotalResult'] = virustotalresult
                save_binary(identifier, data)
        return data
    else:
        return None

def save_binary(identifier, data):

    # Define Firestore document path
    doc_ref = db.collection('%s_events' % os.environ.get('DB_PREFIX')).document(identifier)

    # Add Timestamp
    data['LastUpdated'] = firestore.SERVER_TIMESTAMP

    # Set data in Firestore with merge=True
    doc_ref.set(data, merge=True)

    return data

def get_rule(identifier):

    # Reference to the binary document
    rule_ref = db.collection('%s_rules' % os.environ.get('DB_PREFIX')).document(identifier)

    # Get the binary document
    rule = rule_ref.get()

    if rule.exists:
        return rule.to_dict()
    else:
        return None

def save_rule(identifier, data):

    # Define Firestore document path
    doc_ref = db.collection('%s_rules' % os.environ.get('DB_PREFIX')).document(identifier)

    # Add Timestamp
    data['LastUpdated'] = firestore.SERVER_TIMESTAMP

    # Set data in Firestore with merge=True
    doc_ref.set(data, merge=True)

    return data

@functions_framework.http
def blockables(request):

    request_args = request.args

    path = request.path.strip("/")

    if path.startswith("static/"):
        filename = path[len("static/"):]
        return send_from_directory(STATIC_FOLDER, filename)

    if path.startswith("favico.ico"):
        filename = "favico.ico"
        return send_from_directory(STATIC_FOLDER, filename)

    if request.method == "POST":
    
        try:
            data = request.get_json()
            if not data:
                return jsonify({"success": False, "message": "Invalid JSON or empty body"}), 400
        except Exception as e:
            return jsonify({"success": False, "message": "Invalid request format"}), 400
        
        if data['filehash']:
            binary = get_binary(data['filehash'])
        else:
            return jsonify({"success": False, "message": "No binary data available"}), 400

        if data['action'] == "new":
            identifier = data['identifier']
            scope = data['scope']
            ruletype = data['ruletype']
            ruleid = data['ruleid']
            rule = get_rule(ruleid)
            print('rule: %s' % rule)
            if rule == None:
                rule = {}
                assigned = []
                assigned.append(identifier)
                rule['Identifier'] = ruleid
                rule['Scope'] = scope
                rule['RuleType'] = ruletype
                rule['Policy'] = 'ALLOWLIST'
                rule['Assigned'] = assigned
                print('New Rule: %s' % rule)
                save_rule(ruleid, rule)
            else:
                if 'Assigned' in rule:
                    assigned = rule['Assigned']
                    if not identifier in assigned:
                        print('Adding %s to assigned' % identifier)
                        assigned.append(identifier)
                        print(assigned)
                        rule['Assigned'] = assigned
                        save_rule(ruleid, rule)
                    else:
                        print('Already assigned')
                else:
                    assigned = []
                    assigned.append(identifier)
                    rule['Assigned'] = assigned
                    save_rule(ruleid, rule)
            return jsonify({"success": True, "message": "Rule added successfully!"}), 200
        elif data['action'] == "machine":
            identifier = data['identifier']
            scope = data['scope']
            ruletype = data['ruletype']
            ruleid = data['ruleid']
            rule = get_rule(ruleid)
            print('rule: %s' % rule)
            if rule == None:
                rule = {}
                assigned = []
                assigned.append(identifier)
                rule['Identifier'] = ruleid
                rule['Scope'] = scope
                rule['RuleType'] = ruletype
                rule['Policy'] = 'ALLOWLIST'
                rule['Assigned'] = assigned
                print('New Rule: %s' % rule)
                save_rule(ruleid, rule)
            else:
                if 'Assigned' in rule:
                    assigned = rule['Assigned']
                    if not identifier in assigned:
                        print('Adding %s to assigned' % identifier)
                        assigned.append(identifier)
                        if len(assigned) >= vote_threshold:
                            print("Converting rule to global")
                            rule['Scope'] = 'global'
                            rule.pop("Assigned")
                        else:
                            print('Assigned count has not yet reached %s' % vote_threshold)
                            rule['Assigned'] = assigned
                        save_rule(ruleid, rule)
                    else:
                        print('Already assigned')
                else:
                    assigned = []
                    assigned.append(identifier)
                    rule['Assigned'] = assigned
                    save_rule(ruleid, rule)
            return jsonify({"success": True, "message": "Rule added successfully!"}), 200
        elif data['action'] == "munki":
            identifier = data['identifier']
            scope = data['scope']
            ruletype = data['ruletype']
            ruleid = data['ruleid']
            rule = get_rule(ruleid)
            print('rule: %s' % rule)
            if rule == None:
                rule = {}
                assigned = []
                assigned.append(identifier)
                rule['Identifier'] = ruleid
                rule['Scope'] = scope
                rule['RuleType'] = ruletype
                rule['Policy'] = 'ALLOWLIST'
                rule['Assigned'] = assigned
                print('New Rule: %s' % rule)
                save_rule(ruleid, rule)
            else:
                if 'Assigned' in rule:
                    assigned = rule['Assigned']
                    if not identifier in assigned:
                        print('Adding %s to assigned' % identifier)
                        assigned.append(identifier)
                        print(assigned)
                        rule['Assigned'] = assigned
                        save_rule(ruleid, rule)
                    else:
                        print('Already assigned')
                else:
                    assigned = []
                    assigned.append(identifier)
                    rule['Assigned'] = assigned
                    save_rule(ruleid, rule)
            return jsonify({"success": True, "message": "Rule added successfully!"}), 200
        else:
            return jsonify({"success": False, "message": "Rule not added"}), 400


    if request.method == "GET":
    
        response = {}

        api_key = os.getenv("API_KEY")
        if api_key:
            response['api_key'] = api_key

        if request_args and "machine_id" in request_args:
            machine_id = request_args.getlist('machine_id')[0]
            device = get_device(machine_id)
            if device:
                response['device'] = device
            else:
                return render_template('error.html')

        if request_args and "file_identifier" in request_args:
            file_identifier = request_args.getlist('file_identifier')[0]
            binary = get_binary(file_identifier)
            if binary:
                response['binary'] = binary
                
                if 'SigningChain' in binary and binary['SigningChain']:
                    signing_chain = binary['SigningChain']
                    binary['SignedBy'] = signing_chain[0]['Org']

                if 'TeamID' in binary and binary['TeamID']:
                    print("Looking for rule based on TeamID")
                    rule = get_rule(binary['TeamID'])

                if not rule and 'SigningID' in binary and binary['SigningID']:
                    print("Looking for rule based on SigningID")
                    rule = get_rule(binary['SigningID'])

                if not rule and 'FileSha256' in binary and binary['FileSha256']:
                    print("Looking for rule based on FileSha256")
                    rule = get_rule(binary['FileSha256'])

                if not rule and 'CDHash' in binary and binary['CDHash']:
                    print("Looking for rule based on CDHash")
                    rule = get_rule(binary['CDHash'])

                if rule:
                    response['ruleexists'] = True
                    response['rule'] = rule
                else:
                    response['ruleexists'] = False

                    rule = {}

                    if 'SigningID' in binary and binary['SigningID']:
                        rule['Identifier'] = binary['SigningID']
                        rule['RuleType'] = 'SIGNINGID'

                    elif 'TeamID' in binary and binary['TeamID']:
                        rule['Identifier'] = binary['TeamID']
                        rule['RuleType'] = 'TEAMID'

                    elif 'FileSha256' in binary and binary['FileSha256']:
                        rule['Identifier'] = binary['FileSha256']
                        rule['RuleType'] = 'BINARY'
                    
                    response['rule'] = rule
            else:
                return render_template('error.html')

        print('response: %s' % response)

        return render_template('index.html', response=response)