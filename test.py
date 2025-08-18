#!/usr/local/munki/munki-python

import objc
from Foundation import CFPreferencesCopyAppValue, NSBundle, NSString
from SystemConfiguration import SCDynamicStoreCopyConsoleUser
import requests
import json
import sys

rule_id = "6JSW4SJWN9:md.obsidian"

BUNDLE_ID = 'com.northpolesec.santa'

SyncBaseURL = CFPreferencesCopyAppValue('SyncBaseURL', BUNDLE_ID)
SyncExtraHeaders = CFPreferencesCopyAppValue('SyncExtraHeaders', BUNDLE_ID)

IOKit_bundle = NSBundle.bundleWithIdentifier_('com.apple.framework.IOKit')

functions = [("IOServiceGetMatchingService", b"II@"),
             ("IOServiceMatching", b"@*"),
             ("IORegistryEntryCreateCFProperty", b"@I@@I"),
            ]

objc.loadBundleFunctions(IOKit_bundle, globals(), functions)

data = {}

def io_key(keyname):
    return IORegistryEntryCreateCFProperty(IOServiceGetMatchingService(0, IOServiceMatching("IOPlatformExpertDevice".encode("utf-8"))), NSString.stringWithString_(keyname), None, 0)

def get_hardware_uuid():
    uuid = io_key("IOPlatformUUID")
    data["identifier"] = uuid


def strip_last_segment(url):
    if url.endswith('/'):
        url = url.rstrip('/') 
    return url.rsplit('/', 1)[0]

get_hardware_uuid()

url = strip_last_segment(SyncBaseURL) + '/api/managedapp/' + rule_id

headers = dict(SyncExtraHeaders)
headers["Content-Type"] = "application/json"
headers["Accept"] = "application/json"

response = requests.post(url, json=data, headers=headers)

if response.status_code != 200:
    print("Error:", response.status_code, response.text)

sys.exit(0)