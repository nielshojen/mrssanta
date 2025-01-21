import functions_framework
from flask import render_template
import os
import json
import requests

from google.cloud import firestore

vt_api_key = os.environ.get('VT_API_KEY')

# Initialize Firestore client with a specific database ID
db = firestore.Client(database=os.environ.get('FIRESTORE_DATABASE'))

def get_vt_result(file_hash):
    check_failed = 0

    parameters = {"resource": file_hash, "apikey": vt_api_key}

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
    binary_ref = db.collection('%s_binaries' % os.environ.get('DB_PREFIX')).document(identifier)

    # Get the binary document
    binary = binary_ref.get()

    if binary.exists:
        data =  binary.to_dict()
        if 'virustotalresult' not in data:
            virustotalresult = get_vt_result(data['FileSha256'])
            if virustotalresult is not None:
                data['virustotalresult'] = virustotalresult
                save_binary(data)
        return data
    else:
        return None

def save_binary(data):

    # Define Firestore document path
    doc_ref = db.collection('%s_binaries' % os.environ.get('DB_PREFIX')).document(data)

    # Add Timestamp
    data['last_updated'] = firestore.SERVER_TIMESTAMP

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

@functions_framework.http
def blockables(request):
    request_args = request.args

    print('request_args: %s' % request_args)
    
    response = {}

    if request_args and "machine_id" in request_args:
        machine_id = request_args.getlist('machine_id')[0]
        device = get_device(machine_id)
        if device:
            response.update(device)

    if request_args and "file_identifier" in request_args:
        file_identifier = request_args.getlist('file_identifier')[0]
        binary = get_binary(file_identifier)
        if binary:
            if 'SigningChain' in binary:
                signing_chain = binary['SigningChain']
                binary['signed_by'] = signing_chain[0]['Org']
                response.update(binary)
            if 'SigningID' in binary:
                rule = get_rule(binary['SigningID'])
                if rule:
                    response['scope'] = rule['scope']
                    response['policy'] = rule['policy']
                    response['custom_msg'] = rule['custom_msg']
        else:
            response['file_sha256'] = file_identifier
        
        print('response: %s' % response)

    return render_template('index.html', response=response)