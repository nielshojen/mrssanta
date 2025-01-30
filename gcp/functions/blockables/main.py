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

    if request.method == "GET":
    
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
                response.update(binary)
                if 'SigningChain' in binary and binary['SigningChain']:
                    signing_chain = binary['SigningChain']
                    binary['SignedBy'] = signing_chain[0]['Org']
                if 'TeamID' in binary and binary['TeamID']:
                    rule = get_rule(binary['TeamID'])
                    if rule:
                        response['scope'] = rule['Scope']
                        response['policy'] = rule['Policy']
                        response['custom_msg'] = rule['CustomMessage']
                elif 'SigningID' in binary and binary['SigningID']:
                    rule = get_rule(binary['SigningID'])
                    if rule:
                        response['scope'] = rule['Scope']
                        response['policy'] = rule['Policy']
                        response['custom_msg'] = rule['CustomMessage']
                elif 'FileSha256' in binary and binary['FileSha256']:
                    rule = get_rule(binary['FileSha256'])
                    if rule:
                        response['scope'] = rule['Scope']
                        response['policy'] = rule['Policy']
                        response['custom_msg'] = rule['CustomMessage']
                elif 'CDHash' in binary and binary['CDHash']:
                    rule = get_rule(binary['CDHash'])
                    if rule:
                        response['scope'] = rule['Scope']
                        response['policy'] = rule['Policy']
                        response['custom_msg'] = rule['CustomMessage']
            else:
                response['file_sha256'] = file_identifier
            
            # print('response: %s' % response)

        return render_template('index.html', response=response)

    if request.method == "POST":
        return 'POST request not implemented yet'