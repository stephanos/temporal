#!/usr/bin/env python3
"""Compare frozen source inputs, excluding ignored build outputs."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

root = Path(sys.argv[1])
files = subprocess.check_output(["git", "-C", str(root), "ls-files", "-z"]).decode().split("\0")
files.append("model/scripts/guest-rustc.py")
result = {}
for name in sorted(set(files) - {""}):
    path = root / name
    if path.is_dir():
        result[name] = dict(gitlink=subprocess.check_output(
            ["git", "-C", str(root), "rev-parse", "HEAD:" + name]).decode().strip())
        continue
    result[name] = dict(sha256=hashlib.sha256(path.read_bytes()).hexdigest(),
                       executable=bool(path.stat().st_mode & 0o111))
print(json.dumps(result, sort_keys=True, indent=2))
