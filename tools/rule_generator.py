#!/usr/bin/env python3

import subprocess, json, os, sys, hashlib, requests

MRSSANTA_API_URL = "https://mrssanta.bestseller.com"
MRSSANTA_API_KEY = "njMfW2R3MkbfhWjvxNlor8MIpMVliFgTbQ0QDr1bF4ILddGSmyb6lS9xYISxVuWf"

def run(cmd):
    return subprocess.check_output(cmd, text=True, stderr=subprocess.DEVNULL).strip()

def sha256sum(path):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for chunk in iter(lambda: f.read(8192), b''):
            h.update(chunk)
    return h.hexdigest()

def get_brew_prefix(pkg):
    try:
        output = subprocess.check_output(["brew", "--prefix", pkg], text=True).strip()
        return output
    except subprocess.CalledProcessError:
        print(f"Error: Could not find prefix for package '{pkg}'.", file=sys.stderr)
        sys.exit(1)

def is_installed(name, kind):
    flag = "--formula" if kind=="formula" else "--cask"
    return subprocess.call(["brew", "list", flag, name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL) == 0

# def is_outdated(name, kind):
#     flag = "--formula" if kind=="formula" else "--cask"
#     out = run(["brew", "outdated", flag, name])
#     return name in out.split()

def ensure_latest(name, kind):
    subprocess.call(["brew", "update"])
    if not is_installed(name, kind):
        cmd = ["brew", "install", name] if kind=="formula" else ["brew", "install", "--cask", name]
        subprocess.check_call(cmd)
    # elif is_outdated(name, kind):
    #     cmd = ["brew", "upgrade", name] if kind=="formula" else ["brew", "upgrade", "--cask", name]
    #     subprocess.check_call(cmd)

def get_metadata(cask):
    raw = subprocess.check_output(["brew", "info", "--cask", cask, "--json=v2"], text=True)
    return json.loads(raw)["casks"][0]


def is_executable(path):
    return os.path.isfile(path) and os.access(path, os.X_OK)

def find_executables(base_path):
    executables = []
    for root, dirs, files in os.walk(base_path):
        for name in files:
            full_path = os.path.join(root, name)
            try:
                if is_executable(full_path):
                    executables.append(full_path)
            except Exception as e:
                print(f"Warning: Failed to check {full_path}: {e}", file=sys.stderr)
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
        for dp, _, files in os.walk(rt):
            for fn in files:
                fp = os.path.join(dp, fn)
                if os.path.isfile(fp) and os.access(fp, os.X_OK):
                    bins.append(fp)
    return bins, None

def get_cask(c):
    prefix = subprocess.check_output(["brew", "--prefix"], text=True).strip()

    ensure_latest(c, "cask")

    meta = get_metadata(c)
    bins, version = resolve_binaries(c, meta, prefix)

    if not bins:
        bins, version = fallback_scan(c, prefix)

    results = []
    for fp in sorted(set(bins)):
        if os.path.isfile(fp) and os.access(fp, os.X_OK):
            results.append({
                "source": "homebrew",
                "package": c,
                "version": version,
                "file_path": fp,
                "file_name": os.path.basename(fp),
                "identifier": sha256sum(fp)
            })

    return results

def get_formula(p):
    raw = subprocess.check_output(["brew", "info", "--json=v2", p], text=True)
    j = json.loads(raw)

    info = j["formulae"][0]
    installed = info.get("installed", [])
    version = installed[0]["version"] if isinstance(installed, list) and installed else None

    prefix = get_brew_prefix(p)

    ensure_latest(p, "formula")

    executables = find_executables(os.path.join(prefix, "Cellar", p, version))
    if not executables:
        return []

    results = []
    for exe in executables:
        results.append({
            "source": "homebrew",
            "package": p,
            "version": version,
            "file_path": exe,
            "file_name": os.path.basename(exe),
            "identifier": sha256sum(exe)
        })
    return results

def get_mrssanta_rules(url, key):
    url = f"{url}/api/rules"
    print(f"Fetching rules from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    print(f"Using headers: {headers}")
    result = requests.get(url, headers=headers)
    if result.status_code != 200:
        print(f"Error fetching rules from {url}: {result.status_code} {result.text}", file=sys.stderr)
        sys.exit(1)
    return result.json()

def post_mrssanta_rule(url, key, rule):
    url = f"{url}/api/rules"
    print(f"Posting rule to {url}")
    headers = {
        "X-API-Key": f"{key}",
        "Content-Type": "application/json"
    }
    print(f"Using headers: {headers}")
    result = requests.post(url, headers=headers, data=json.dumps(rule))
    if result.status_code != 200:
        print(f"Error posting rule to {url}: {result.status_code} {result.text}", file=sys.stderr)
        sys.exit(1)

candidate_shas = []

def main():
    with open("tools/rule_generator.json", encoding='utf-8') as f:
        rules = json.load(f)
    
    for rule in rules:
        if rule.get("package_type") == "homebrew" and rule.get("type") == "cask":
            shas = get_cask(rule.get("package"))
            if shas:
                candidate_shas.extend(shas)
        elif rule.get("package_type") == "homebrew" and rule.get("type") == "formula":
            shas = get_formula(rule.get("package"))
            if shas:
                candidate_shas.extend(shas)
        else:
            print(f"Skipping unsupported rule: {rule}")

    if len(candidate_shas) > 0:
        new_rules = []
        mrssanta_rules = get_mrssanta_rules(MRSSANTA_API_URL, MRSSANTA_API_KEY)
        for sha in candidate_shas:
            match = next((item for item in mrssanta_rules if item['identifier'] == sha['identifier']), None)
            if match:
                print(f"Found existing rule for {sha['file_name']} ({sha['identifier']})")
            else:
                new_rule = {
                    "custom_msg": f"Allow {sha['source']} {sha['file_name']} {sha['version']}",
                    "identifier": sha['identifier'],
                    "policy": "ALLOWLIST",
                    "rule_type": "BINARY",
                    "scope": "global"
                }
                new_rules.append(new_rule)
        
        print(json.dumps(new_rules, indent=2))
        if len(new_rules) > 0:
            post_mrssanta_rule(MRSSANTA_API_URL, MRSSANTA_API_KEY, new_rules)

if __name__ == "__main__":
    main()