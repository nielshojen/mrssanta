#!/usr/bin/env python3

import os
import sys
import subprocess
import hashlib
import stat

def get_brew_prefix(pkg):
    try:
        output = subprocess.check_output(["brew", "--prefix", pkg], text=True).strip()
        return output
    except subprocess.CalledProcessError:
        print(f"Error: Could not find prefix for package '{pkg}'.", file=sys.stderr)
        sys.exit(1)

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


def is_executable(path):
    return os.path.isfile(path) and os.access(path, os.X_OK)

def sha256sum(file_path):
    h = hashlib.sha256()
    with open(file_path, 'rb') as f:
        for chunk in iter(lambda: f.read(8192), b''):
            h.update(chunk)
    return h.hexdigest()

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

def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <homebrew-package-name>")
        sys.exit(1)

    package = sys.argv[1]
    prefix = get_brew_prefix(package)
    print(f"🔍 Searching in: {prefix}")

    ensure_local_latest(package)

    executables = find_executables(prefix)
    if not executables:
        print("No executables found.")
        return

    print("\n📦 Executables and their SHA-256 hashes:\n")
    for exe in executables:
        checksum = sha256sum(exe)
        print(f"{checksum}  {exe}")

if __name__ == "__main__":
    main()