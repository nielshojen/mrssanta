#!/usr/bin/env python3

import json
import csv
import subprocess

def santactl_fileinfo(file):
    cmd = ["santactl fileinfo --json %s" % file]
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    result = result.stdout
    try:
        result = json.loads(result)
    except:
        print(result)
        return None
    if len(result) != 0 and type(result) == list:
        return result[0]
    else:
        return None

cmd = ["/usr/bin/mdfind kind:application 2>/dev/null"]
result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
result = result.stdout.splitlines()

fileinfo = []

if result:
    for app in result:
        print('Getting fileinfo for: %s' % app)
        info = santactl_fileinfo(app)
        if info is None:
            print('No fileinfo for %s' % app)
        else:
            if 'Bundle Name' in info:
                print('%s: %s' % ('Bundle Name', info['Bundle Name']))
            if 'Bundle Version' in info:
               print('%s: %s' % ('Bundle Version', info['Bundle Version']))
            if 'Bundle Version Str' in info:
                print('%s: %s' % ('Bundle Version Str', info['Bundle Version Str']))
            print('%s: %s' % ('Path', info['Path']))
            print('%s: %s' % ('SHA-256', info['SHA-256']))
            if 'Signing ID' in info:
                print('%s: %s' % ('Signing ID', info['Signing ID']))
            if 'Team ID' in info:
                print('%s: %s' % ('Team ID', info['Team ID']))
            print('%s: %s' % ('Code-signed', info['Code-signed']))
            if 'Signing Chain' in info:
                print('%s: %s' % ('Signing Chain', info['Signing Chain']))
            if 'Rule' in info:
                print('%s: %s' % ('Rule', info['Rule']))
            fileinfo.append(info)
        print('-------------------------------------------')

with open("fileinfo.json", "w") as f:
    json.dump(fileinfo, f)