#!/usr/bin/env python3

import subprocess
import json
import os
import sys
import hashlib
import requests

import time
import atexit
import logging


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)
logger = logging.getLogger(__name__)


MRSSANTA_API_URL = os.getenv("MRSSANTA_API_URL")
MRSSANTA_API_KEY = os.getenv("MRSSANTA_API_KEY")


def get_mrssanta_rules(url, key):
    url = f"{url}/api?endpoint=rules"
    logger.info(f"Fetching rules from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    result = requests.get(url, headers=headers)
    if result.status_code != 200:
        logger.info(f"Error fetching rules from {url}: {result.status_code} {result.text}")
        sys.exit(1)
    return result.json()


def get_mrssanta_events(url, key,):
    url = f"{url}/api?endpoint=events"
    logger.info(f"GETing events from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    result = requests.get(url, headers=headers, timeout=60)
    if result.status_code != 200:
        logger.info(f"Error fetching events from {url}: {result.status_code} {result.text}")
        sys.exit(1)
    return result.json()


def get_mrssanta_event(url, key, id):
    url = f"{url}/api/events/{id}"
    logger.info(f"GETing event from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    result = requests.get(url, headers=headers, timeout=30)
    if result.status_code != 200:
        logger.info(f"Error fetching event from {url}: {result.status_code} {result.text}")
        sys.exit(1)
    return result.json()


def post_mrssanta_rule(url, key, rule):
    url = f"{url}/api?endpoint=rules"
    logger.info(f"POSTing to {url}")
    headers = {
        "X-API-Key": f"{key}",
        "Content-Type": "application/json"
    }
    result = requests.post(url, headers=headers, data=json.dumps(rule))
    if result.status_code != 200:
        logger.info(f"Error posting rules to {url}: {result.status_code} {result.text}")
        sys.exit(1)



def main():
    mrssanta_events = get_mrssanta_events(MRSSANTA_API_URL, MRSSANTA_API_KEY)
    mrssanta_rules = get_mrssanta_rules(MRSSANTA_API_URL, MRSSANTA_API_KEY)

    print(f"Found {len(mrssanta_events)} events and {len(mrssanta_rules)} rules")

    for rule in mrssanta_rules:
        rules = []
        if rule['policy'] == 'ALLOWLIST' and 'custom_msg' not in rule and rule['rule_type'] == 'BINARY':
            event = get_mrssanta_event(MRSSANTA_API_URL, MRSSANTA_API_KEY, rule['identifier'])
            event = next(
                (e for e in mrssanta_events if e.get("file_sha256") == rule['identifier']),
                None
            )

            if event and 'file_path' in event:
                if 'Homebrew' in event['file_path']:
                    print(f"Need to dig out the description from {event['file_path']}")
                elif 'file_path' in event:
                    custom_msg = f"Allow {rule['rule_type']} with identifier: {rule['identifier']} for file: {event['file_name']}"
                    rule['custom_msg'] = custom_msg
                    rules.append(rule)
                    post_mrssanta_rule(MRSSANTA_API_URL, MRSSANTA_API_KEY,rules)
                else:
                    print(f">>>>. Don't know what this is: {event}")
            print("______________________")
        elif rule['policy'] == 'ALLOWLIST' and 'custom_msg' not in rule and rule['rule_type'] == 'SIGNINGID':
            print(rule)
            event = next(
                (e for e in mrssanta_events if e.get("signing_id") == rule['identifier']),
                None
            )

            if event:
                if 'file_bundle_name' in event:
                    if 'file_bundle_version' in event:
                        custom_msg = f"Allow {rule['rule_type']} with identifier: {rule['identifier']} for file bundle: {event['file_bundle_name']} {event['file_bundle_version']}"
                        rule['custom_msg'] = custom_msg
                    else:
                        custom_msg = f"Allow {rule['rule_type']} with identifier: {rule['identifier']} for file bundle: {event['file_bundle_name']}"
                        rule['custom_msg'] = custom_msg
                elif 'file_name' in event:
                    custom_msg = f"Allow {rule['rule_type']} with identifier: {rule['identifier']} for file: {event['file_name']}"
                    rule['custom_msg'] = custom_msg
                elif 'file_name' in event:
                    custom_msg = f"Allow {rule['rule_type']} with identifier: {rule['identifier']} for file: {event['file_name']}"
                    rule['custom_msg'] = custom_msg
                else:
                    print(f">>>>>>>>>> {event}")

                rules.append(rule)
                post_mrssanta_rule(MRSSANTA_API_URL, MRSSANTA_API_KEY,rules)



            print("______________________")
if __name__ == "__main__":
    main()