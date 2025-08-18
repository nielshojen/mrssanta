#!/usr/bin/env python3
import subprocess, json, os, sys, hashlib

def sha256sum(path):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for chunk in iter(lambda: f.read(8192), b''):
            h.update(chunk)
    return h.hexdigest()

def is_installed(cask):
    return subprocess.call(["brew", "list", "--cask", cask],
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL) == 0

def is_outdated(cask):
    out = subprocess.check_output(["brew", "outdated", "--cask", cask], text=True).strip()
    return any(line.strip() == cask for line in out.splitlines())

def ensure_local_latest(cask):
    subprocess.call(["brew", "update"])
    if not is_installed(cask):
        print(f"➡️ Installing cask: {cask}")
        subprocess.check_call(["brew", "install", "--cask", cask])
    elif is_outdated(cask):
        print(f"⬆️ Upgrading cask: {cask}")
        subprocess.check_call(["brew", "upgrade", "--cask", cask])
    else:
        print(f"✅ {cask} is already installed and up-to-date")

def get_metadata(cask):
    raw = subprocess.check_output(["brew", "info", "--cask", cask, "--json=v2"], text=True)
    return json.loads(raw)["casks"][0]

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

def main():
    if len(sys.argv) != 2:
        print("Usage: script.py <cask>", file=sys.stderr)
        sys.exit(1)
    cask = sys.argv[1]
    prefix = subprocess.check_output(["brew", "--prefix"], text=True).strip()

    ensure_local_latest(cask)

    meta = get_metadata(cask)
    bins, version = resolve_binaries(cask, meta, prefix)
    print(f"ℹ️ Found {len(bins)} binary paths via JSON (version={version})") if bins else None

    if not bins:
        print("🔄 Falling back to scanning local Caskroom")
        bins, version = fallback_scan(cask, prefix)
        print(f"ℹ️ Fallback scan found {len(bins)} executables")

    if not bins:
        print("❌ No executable files found!", file=sys.stderr)
        sys.exit(1)

    results = []
    for fp in sorted(set(bins)):
        if os.path.isfile(fp) and os.access(fp, os.X_OK):
            results.append({
                "package": cask,
                "version": version,
                "path": fp,
                "filename": os.path.basename(fp),
                "sha256": sha256sum(fp)
            })

    print(json.dumps(results, indent=2))

if __name__ == "__main__":
    main()