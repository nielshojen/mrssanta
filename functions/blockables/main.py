import functions_framework
from flask import render_template
import os
import json

from google.cloud import firestore

# Initialize Firestore client with a specific database ID
db = firestore.Client()

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
        return binary.to_dict()
    else:
        return None

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
            signing_chain = binary['signing_chain']
            binary['signed_by'] = signing_chain[0]['org']
            response.update(binary)
            if 'signing_id' in binary:
                rule = get_rule(binary['signing_id'])
                if rule:
                    response['scope'] = rule['scope']
                    response['policy'] = rule['policy']
                    response['custom_msg'] = rule['custom_msg']
        else:
            response['file_sha256'] = file_identifier

    return render_template('index.html', response=response)