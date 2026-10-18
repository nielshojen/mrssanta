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

device_list = [
]

def get_mrssanta_devices(url, key):
    url = f"{url}/api/devices"
    logger.info(f"GETing events from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    result = requests.get(url, headers=headers, timeout=60)
    if result.status_code != 200:
        logger.info(f"Error fetching events from {url}: {result.status_code} {result.text}")
        sys.exit(1)
    return result.json()


def post_mrssanta_device(url, key, data):
    url = f"{url}/api/devices"
    logger.info(f"POSTing to {url}")
    headers = {
        "X-API-Key": f"{key}",
        "Content-Type": "application/json"
    }
    post_data = json.dumps(data)
    result = requests.post(url, headers=headers, data=post_data)
    if result.status_code != 200:
        logger.info(f"Error posting devices to {url}: {result.status_code} {result.text}")
        sys.exit(1)
    else:
        logger.info(f"Sucessfully posting devices to {url}: {result.status_code} {result.text}")



def main():
    mrssanta_devices = get_mrssanta_devices(MRSSANTA_API_URL, MRSSANTA_API_KEY)

    for device in mrssanta_devices:
        post_data = []
        devicename = device.get('hostname').split(".")[0]
        clientmode = device.get('client_mode')
        if devicename in device_list and clientmode != 2:
            print(device)
            device['client_mode'] = 2
            post_data.append(device)
            break
    
    if len(post_data) > 0:
        print(json.dumps(post_data, indent=2))
        post_mrssanta_device(MRSSANTA_API_URL, MRSSANTA_API_KEY, post_data)

if __name__ == "__main__":
    main()