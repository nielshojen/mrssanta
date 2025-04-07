import functions_framework
from flask import render_template, jsonify, send_from_directory
import os
import requests
from urllib.parse import unquote

from datetime import datetime, timezone
from pymongo import MongoClient

organization = os.environ.get('ORGANIZATION', 'this Organization')
vt_api_key = os.environ.get('VT_API_KEY')
vote_threshold = int(os.environ.get('VOTE_THRESHOLD'))

# MongoDB connection
MONGO_URI = os.environ.get("MONGO_URI")
client = MongoClient(MONGO_URI)

# Select database
db = client[os.environ.get("MONGO_DB")]

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
    """Fetch a device by its identifier from MongoDB."""
    collection = db["devices"]

    device = collection.find_one({"_id": identifier})

    return device if device else None

def get_binary(identifier):
    """Fetch a binary by its identifier from MongoDB and update VirusTotalResult if missing."""
    collection = db["events"]

    binary = collection.find_one({"_id": identifier})

    if binary:
        if "virustotal_result" not in binary or binary["virustotal_result"] == 0:
            virustotalresult = get_vt_result(binary.get("file_sha256"))
            if virustotalresult is not None:
                binary["virustotal_result"] = virustotalresult
                save_binary(identifier, binary)
        return binary
    else:
        return None

def save_binary(identifier, data):
    """Saves or updates a binary document in MongoDB."""
    collection = db["events"]

    data["last_updated"] = datetime.now(timezone.utc)

    collection.update_one(
        {"_id": identifier},
        {"$set": data},
        upsert=True
    )

    return data

def get_rule(identifier):
    """Fetch a rule by its identifier from MongoDB."""
    collection = db["rules"]

    rule = collection.find_one({"_id": identifier})

    return rule if rule else None

def save_rule(identifier, data):
    """Saves or updates a rule document in MongoDB."""
    collection = db["rules"]

    data["last_updated"] = datetime.now(timezone.utc)

    update_ops = {"$set": data}

    if "assigned" not in data:
        update_ops["$unset"] = {"assigned": ""}

    collection.update_one(
        {"_id": identifier},
        update_ops,
        upsert=True
    )

    return data

@functions_framework.http
def blockables(request):

    request_args = request.args

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
            if rule == None:
                rule = {}
                assigned = []
                assigned.append(identifier)
                rule['identifier'] = ruleid
                rule['scope'] = scope
                rule['rule_type'] = ruletype
                rule['policy'] = 'ALLOWLIST'
                rule['assigned'] = assigned
                save_rule(ruleid, rule)
            else:
                if 'assigned' in rule:
                    assigned = rule['assigned']
                    if not identifier in assigned:
                        assigned.append(identifier)
                        rule['assigned'] = assigned
                        save_rule(ruleid, rule)
                else:
                    assigned = []
                    assigned.append(identifier)
                    rule['assigned'] = assigned
                    save_rule(ruleid, rule)
            return jsonify({"success": True, "message": "Rule added successfully! Rule will be applied within a minute"}), 200
        elif data['action'] == "machine":
            identifier = data['identifier']
            scope = data['scope']
            ruletype = data['ruletype']
            ruleid = data['ruleid']
            rule = get_rule(ruleid)
            if rule == None:
                rule = {}
                assigned = []
                assigned.append(identifier)
                rule['identifier'] = ruleid
                rule['scope'] = scope
                rule['rule_type'] = ruletype
                rule['policy'] = 'ALLOWLIST'
                rule['assigned'] = assigned
                save_rule(ruleid, rule)
            else:
                if 'assigned' in rule:
                    assigned = rule['assigned']
                    if not identifier in assigned:
                        assigned.append(identifier)
                        if len(assigned) >= vote_threshold:
                            rule['scope'] = 'global'
                            rule['custom_msg'] = 'Converted to global rule'
                            rule['voted'] = True
                            rule.pop("assigned")
                        else:
                            rule['assigned'] = assigned
                        save_rule(ruleid, rule)
                else:
                    assigned = []
                    assigned.append(identifier)
                    rule['assigned'] = assigned
                    save_rule(ruleid, rule)
            return jsonify({"success": True, "message": "Rule added successfully! Rule will be applied within a minute"}), 200
        elif data['action'] == "managedapp":
            identifier = data['identifier']
            scope = data['scope']
            ruletype = data['ruletype']
            ruleid = data['ruleid']
            rule = get_rule(ruleid)
            if rule == None:
                rule = {}
                assigned = []
                assigned.append(identifier)
                rule['identifier'] = ruleid
                rule['scope'] = scope
                rule['rule_type'] = ruletype
                rule['policy'] = 'ALLOWLIST'
                rule['assigned'] = assigned
                save_rule(ruleid, rule)
            else:
                if 'assigned' in rule:
                    assigned = rule['assigned']
                    if not identifier in assigned:
                        assigned.append(identifier)
                        rule['assigned'] = assigned
                        save_rule(ruleid, rule)
                else:
                    assigned = []
                    assigned.append(identifier)
                    rule['assigned'] = assigned
                    save_rule(ruleid, rule)
            return jsonify({"success": True, "message": "Rule added successfully!"}), 200
        else:
            return jsonify({"success": False, "message": "Rule not added"}), 400


    if request.method == "GET":

        path = request.path.strip("/")

        path = unquote(path)

        if path.startswith("static/"):
            filename = path[len("static/"):]
            return send_from_directory(STATIC_FOLDER, filename)

        if path.startswith("favico.ico"):
            filename = "favico.ico"
            return send_from_directory(STATIC_FOLDER, filename)
    
        response = {}
        
        response['organization'] = organization

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
                rule = None
                response['binary'] = binary

                if 'TeamID' in binary and binary['team_id']:
                    rule = get_rule(binary['team_id'])

                if not rule and 'signing_id' in binary and binary['signing_id']:
                    rule = get_rule(binary['signing_id'])

                if not rule and 'file_sha256' in binary and binary['file_sha256']:
                    rule = get_rule(binary['file_sha256'])

                if not rule and 'cdhash' in binary and binary['cdhash']:
                    rule = get_rule(binary['cdhash'])

                if rule:
                    response['ruleexists'] = True
                    response['rule'] = rule
                else:
                    response['ruleexists'] = False

                    rule = {}

                    if 'signing_id' in binary and binary['signing_id']:
                        rule['identifier'] = binary['signing_id']
                        rule['rule_type'] = 'SIGNINGID'

                    elif 'team_id' in binary and binary['team_id']:
                        rule['identifier'] = binary['team_id']
                        rule['rule_type'] = 'TEAMID'

                    elif 'file_sha256' in binary and binary['file_sha256']:
                        rule['identifier'] = binary['file_sha256']
                        rule['rule_type'] = 'BINARY'
                    
                    response['rule'] = rule
            else:
                return render_template('error.html')

        return render_template('index.html', response=response)