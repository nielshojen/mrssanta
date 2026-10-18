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

# Start timer
start_time = time.time()


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)
logger = logging.getLogger(__name__)


MRSSANTA_API_URL = os.getenv("MRSSANTA_API_URL")
MRSSANTA_API_KEY = os.getenv("MRSSANTA_API_KEY")


def run(cmd):
    return subprocess.check_output(cmd, text=True, stderr=subprocess.DEVNULL).strip()


def get_santactl_fileinfo(path):
    try:
        raw = subprocess.check_output(
            ["santactl", "fileinfo", "--json", path],
            text=True,
            stderr=subprocess.DEVNULL,
        )
        data = json.loads(raw)

        # santactl fileinfo --json returns a list
        if isinstance(data, list) and data:
            if 'Type' in data[0] and data[0]['Type'] != 'Script':
                if 'Signing ID' in data[0]:
                    if 'Bundle Version' in data[0]:
                        return 'SigningID', data[0]['Bundle Version'], data[0]['Signing ID']
                    else:
                        return 'SIGNINGID', None, data[0]['Signing ID']
                else:
                    if 'Bundle Version' in data[0]:
                        return 'BINARY', data[0]['Bundle Version'], data[0]['SHA-256']
                    else:
                        return 'BINARY', None, data[0]['SHA-256']
            else:
                return None, None, None

        return None, None, None
    except (subprocess.CalledProcessError, json.JSONDecodeError, IndexError, TypeError):
        return None, None, None


def get_brew_prefix(pkg):
    try:
        output = subprocess.check_output(["brew", "--prefix", pkg], text=True).strip()
        return output
    except subprocess.CalledProcessError:
        logger.info(f"Error: Could not find prefix for package '{pkg}'.", file=sys.stderr)
        sys.exit(1)


def is_installed(name, kind):
    logger.info(f"Checking if {name} is installed as {kind}...")
    flag = "--formula" if kind=="formula" else "--cask"
    return subprocess.call(["brew", "list", flag, name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL) == 0


def ensure_updated():
    try:
        logger.info("Running: brew update")
        subprocess.run(["brew", "update"], check=True, stdout=sys.stdout, stderr=sys.stderr)

        logger.info("Checking for outdated formulae or casks...")
        outdated = subprocess.check_output(["brew", "outdated"], text=True).strip().splitlines()

        if not outdated:
            logger.info("Everything is already up to date.")
            return True

        logger.info(f"⬆Upgrading {len(outdated)} item(s): {', '.join(outdated)}")
        subprocess.run(["brew", "upgrade"], check=True, stdout=sys.stdout, stderr=sys.stderr)

        logger.info("Upgrade completed.")
        return True
    except subprocess.CalledProcessError as e:
        logger.info(f"Brew command failed: {e}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        logger.info("Interrupted by user.", file=sys.stderr)
        sys.exit(1)


def ensure_installed(name, kind):
    if not is_installed(name, kind):
        cmd = ["brew", "install", name] if kind=="formula" else ["brew", "install", "--cask", name]
        subprocess.check_call(cmd)

def get_metadata(cask):
    raw = subprocess.check_output(["brew", "info", "--cask", cask, "--json=v2"], text=True)
    return json.loads(raw)["casks"][0]

def install_tap(t):
    subprocess.check_call(["brew", "tap", t])
    

def is_executable(path):
    return os.path.isfile(path) and os.access(path, os.X_OK)

def find_executables(base_path):
    executables = []
    for root, path, files in os.walk(base_path):
        for name in files:
            full_path = os.path.join(root, name)
            try:
                if is_executable(full_path):
                    executables.append(full_path)
            except Exception as e:
                logger.info(f"Warning: Failed to check {full_path}: {e}", file=sys.stderr)
    return executables

def resolve_binaries(cask, meta, prefix):
    inst = meta.get("installed")
    version = (inst if isinstance(inst, str)
               else (inst[0].get("version") if isinstance(inst, list) and inst else None))
    rels = []
    for art in meta.get("artifacts", []):
        if isinstance(art, dict) and isinstance(art.get("binary"), list):
            for r in art["binary"]:
                if isinstance(r, str):
                    rels.append(r)
    root = os.path.join(prefix, "Caskroom", cask, version) if version else None
    if root:
        return [os.path.join(root, r) for r in rels], version
    return [], version


def fallback_scan(cask, prefix):
    out = subprocess.check_output(["brew", "list", "--cask", "--verbose", cask], text=True)
    roots = set()
    for ln in out.splitlines():
        if "/Caskroom/" in ln:
            parts = ln.split("/")
            idx = parts.index("Caskroom")
            roots.add(os.path.join("/", *parts[:idx+3]))
    bins = []
    for rt in roots:
        for dp, path, files in os.walk(rt):
            for fn in files:
                fp = os.path.join(dp, fn)
                if os.path.isfile(fp) and os.access(fp, os.X_OK):
                    bins.append(fp)
    return bins, None


def get_cask(c):
    prefix = subprocess.check_output(["brew", "--prefix"], text=True).strip()

    ensure_installed(c, "cask")

    meta = get_metadata(c)
    bins, version = resolve_binaries(c, meta, prefix)

    if not bins:
        bins, version = fallback_scan(c, prefix)

    logger.info(f"Found {len(bins)} binaries for cask {c} v{version}")

    results = []
    for fp in sorted(set(bins)):
        if os.path.isfile(fp) and os.access(fp, os.X_OK):
            # print(f"Found executable: {fp}")
            identifier_type, bundle_version, identifier = get_santactl_fileinfo(fp)

            # results.append({
            #     "source": "homebrew",
            #     "package": c,
            #     "version": version,
            #     "file_path": fp,
            #     "file_name": os.path.basename(fp),
            #     "identifier": sha256sum(fp)
            # })


            # if bundle_version != None:
            #     version = bundle_version

            if identifier != None:
                results.append({
                    "source": "homebrew",
                    "package": c,
                    "version": version,
                    "file_path": fp,
                    "file_name": os.path.basename(fp),
                    "identifier_type": identifier_type,
                    "identifier": identifier
                })

    return results


def get_formula(p):
    raw = subprocess.check_output(["brew", "info", "--json=v2", p], text=True)
    j = json.loads(raw)

    info = j["formulae"][0]
    installed = info.get("installed", [])
    version = installed[0]["version"] if isinstance(installed, list) and installed else None

    prefix = subprocess.check_output(["brew", "--prefix"], text=True).strip()

    ensure_installed(p, "formula")

    if version:
        executables = find_executables(os.path.join(prefix, "Cellar", p, version))
    else:
        return []
    
    if not executables:
        logger.info(f"No executables found for formula {p} v{version}")
        return []

    logger.info(f"Found {len(executables)} executables for formula {p} v{version}")

    results = []
    for exe in executables:
        # print(f"Found executable: {exe}")
        identifier_type, bundle_version, identifier = get_santactl_fileinfo(exe)

        # results.append({
        #     "source": "homebrew",
        #     "package": p,
        #     "version": version,
        #     "file_path": exe,
        #     "file_name": os.path.basename(exe),
        #     "identifier": sha256sum(exe)
        # })

        # if bundle_version != None:
        #     version = bundle_version

        if identifier != None:
            results.append({
                "source": "homebrew",
                "package": p,
                "version": version,
                "file_path": exe,
                "file_name": os.path.basename(exe),
                "identifier_type": identifier_type,
                "identifier": identifier
            })
    return results


def get_vendor_binaries(vendor_path="/opt/homebrew/Library/Homebrew/vendor"):
    logger.info(f"Scanning vendor path: {vendor_path}")
    if not os.path.isdir(vendor_path):
        logger.info(f"Vendor path not found: {vendor_path}")
        return []

    results = []

    for root, path, files in os.walk(vendor_path):
        for name in files:
            full_path = os.path.join(root, name)
            try:
                if os.path.islink(full_path):
                    continue
                if not os.path.isfile(full_path) or not os.access(full_path, os.X_OK):
                    continue
                parts = full_path.split(os.sep)
                try:
                    idx = parts.index("vendor")
                    package = parts[idx + 1] if len(parts) > idx + 1 else "unknown"
                    version = parts[idx + 2] if len(parts) > idx + 2 else "unknown"
                except ValueError:
                    package = "unknown"
                    version = "unknown"

                # print(f"Found vendor binary: {full_path} (package: {package}, version: {version})")
                identifier_type, bundle_version, identifier = get_santactl_fileinfo(full_path)

                # if bundle_version != None:
                #     version = bundle_version

                if identifier != None:
                    results.append({
                        "source": "homebrew",
                        "package": package,
                        "version": version,
                        "file_path": full_path,
                        "file_name": name,
                        "identifier_type": identifier_type,
                        "identifier": identifier
                    })

            except Exception as e:
                logger.warning(f"Failed to check {full_path}: {e}")
    return results


def get_mrssanta_rules(url, key):
    url = f"{url}/api/rules"
    logger.info(f"Fetching rules from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    result = requests.get(url, headers=headers)
    if result.status_code != 200:
        logger.info(f"Error fetching rules from {url}: {result.status_code} {result.text}")
        sys.exit(1)
    return result.json()


def post_mrssanta_rule(url, key, rule):
    url = f"{url}/api/rules"
    logger.info(f"Posting to {url}")
    headers = {
        "X-API-Key": f"{key}",
        "Content-Type": "application/json"
    }
    result = requests.post(url, headers=headers, data=json.dumps(rule))
    if result.status_code != 200:
        logger.info(f"Error posting rule to {url}: {result.status_code} {result.text}")
        sys.exit(1)


def report_runtime():
    end_time = time.time()
    logger.info(f"Total runtime: {end_time - start_time:.2f} seconds")


candidate_identifiers = []


def main():
    result = ensure_updated()

    logger.info(f"Update successful: {result}")

    with open("tools/rule_generator.json", encoding='utf-8') as f:
        rules = json.load(f)

    logger.info(f"Loaded {len(rules)} packages to generate rules based on ...")
    
    for rule in rules:
        try:
            logger.info(f"Processing {rule['package_type']} {rule['type']} package: {rule['package']}")
            if rule.get("package_type") == "homebrew" and rule.get("type") == "tap":
                install_tap(rule.get("package"))
            elif rule.get("package_type") == "homebrew" and rule.get("type") == "cask":
                shas = get_cask(rule.get("package"))
                if shas:
                    candidate_identifiers.extend(shas)
            elif rule.get("package_type") == "homebrew" and rule.get("type") == "formula":
                shas = get_formula(rule.get("package"))
                if shas:
                    candidate_identifiers.extend(shas)
            else:
                logger.info(f"Skipping unsupported rule: {rule}")
        except subprocess.CalledProcessError as e:
            command = e.cmd
            if isinstance(command, (list, tuple)):
                command = " ".join(str(part) for part in command)
            logger.warning(
                "Skipping %s %s package %s because command failed with exit code %s: %s",
                rule.get("package_type"),
                rule.get("type"),
                rule.get("package"),
                e.returncode,
                command,
            )
            continue

    vendor_binaries = get_vendor_binaries()

    if vendor_binaries:
        candidate_identifiers.extend(vendor_binaries)

    if len(candidate_identifiers) > 0:
        new_rules = []
        mrssanta_rules = get_mrssanta_rules(MRSSANTA_API_URL, MRSSANTA_API_KEY)
        for identifier in candidate_identifiers:
            match = next((item for item in mrssanta_rules if item['identifier'] == identifier['identifier']), None)
            if match:
                logger.info(f"Found existing rule for {identifier['file_name']} ({identifier['identifier']})")
            else:
                new_rule = {
                    "custom_msg": f"Allow {identifier['source']} {identifier['package']} {identifier['file_name']} v{identifier['version']}",
                    "identifier": identifier['identifier'],
                    "policy": "ALLOWLIST",
                    "rule_type": f"{identifier['identifier_type']}",
                    "scope": "global"
                }
                new_rules.append(new_rule)

        if len(new_rules) > 0:
            logger.info(f"Posting {len(new_rules)} new rules ...")
            post_mrssanta_rule(MRSSANTA_API_URL, MRSSANTA_API_KEY, new_rules)
        
    atexit.register(report_runtime)

if __name__ == "__main__":
    main()
