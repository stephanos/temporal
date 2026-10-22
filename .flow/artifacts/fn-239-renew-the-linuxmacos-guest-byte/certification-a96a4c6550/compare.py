#!/usr/bin/env python3
"""Certify all native artifacts and clean source pins with actual cmp calls."""
import hashlib
import itertools
import json
from pathlib import Path
import subprocess
import sys

repository = Path(sys.argv[1]).resolve()
pin = "a96a4c6550b468c5d7c3011a7961d4e118dacd93"
roots = {platform: repository / (".flow/tmp/fn239-certified-" + platform + "-a96a4c6550")
         for platform in ["linux", "macos"]}
result = dict(pin=pin, sources={}, targets={}, artifacts={}, comparisons=[])
for platform, root in roots.items():
    for lane in ["a", "b"]:
        source = root / ("source-" + lane)
        revision = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"]).decode().strip()
        status = subprocess.check_output(["git", "-C", str(source), "status", "--porcelain", "--untracked-files=all"]).decode()
        tree = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD^{tree}"]).decode().strip()
        assert revision == pin and status == "", (source, revision, status)
        label = platform + "-" + lane
        result["sources"][label] = dict(path=str(source), revision=revision, tree=tree, status=status)
        target = root / ("target-" + lane)
        stat = target.stat()
        result["targets"][label] = dict(path=str(target), device=stat.st_dev, inode=stat.st_ino)
        for guest in ["stock", "temporal"]:
            artifact = root / (guest + "-" + lane + ".wasm")
            data = artifact.read_bytes()
            result["artifacts"][platform + "/" + artifact.name] = dict(bytes=len(data), sha256=hashlib.sha256(data).hexdigest())
assert len({(item["device"], item["inode"]) for item in result["targets"].values()}) == 4
assert len({item["tree"] for item in result["sources"].values()}) == 1
for guest in ["stock", "temporal"]:
    artifacts = [(platform, root / (guest + "-" + lane + ".wasm"))
                 for platform, root in roots.items() for lane in ["a", "b"]]
    for (first_platform, first), (second_platform, second) in itertools.combinations(artifacts, 2):
        comparison = subprocess.run(["cmp", str(first), str(second)], capture_output=True, text=True, check=False)
        result["comparisons"].append(dict(first=first_platform + "/" + first.name,
                                          second=second_platform + "/" + second.name,
                                          exit_code=comparison.returncode,
                                          output=comparison.stdout + comparison.stderr))
print(json.dumps(result, indent=2))
sys.exit(any(item["exit_code"] for item in result["comparisons"]))
