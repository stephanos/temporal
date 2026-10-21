#!/usr/bin/env python3
"""Record every actual within-OS and cross-OS byte comparison."""
import hashlib
import itertools
import json
from pathlib import Path
import subprocess
import sys

repository = Path(sys.argv[1]).resolve()
proof = repository / ".flow/tmp/fn236-snapshot-proof"
result = dict(source_manifest_sha256=hashlib.sha256((proof / "frozen-source.json").read_bytes()).hexdigest(),
              artifacts={}, targets={}, comparisons=[])
for platform in ["linux", "macos"]:
    for lane in ["a", "b"]:
        target = proof / platform / ("target-" + lane)
        stat = target.stat()
        result["targets"][platform + "-" + lane] = dict(path=str(target), device=stat.st_dev, inode=stat.st_ino)
        for guest in ["stock", "temporal"]:
            artifact = proof / platform / (guest + "-" + lane + ".wasm")
            data = artifact.read_bytes()
            result["artifacts"][str(artifact.relative_to(proof))] = dict(bytes=len(data), sha256=hashlib.sha256(data).hexdigest())
assert len({(item["device"], item["inode"]) for item in result["targets"].values()}) == 4
for guest in ["stock", "temporal"]:
    artifacts = [proof / platform / (guest + "-" + lane + ".wasm")
                 for platform in ["linux", "macos"] for lane in ["a", "b"]]
    for first, second in itertools.combinations(artifacts, 2):
        comparison = subprocess.run(["cmp", str(first), str(second)], capture_output=True, text=True, check=False)
        result["comparisons"].append(dict(first=str(first.relative_to(proof)), second=str(second.relative_to(proof)),
                                          exit_code=comparison.returncode, output=comparison.stdout + comparison.stderr))
print(json.dumps(result, indent=2))
sys.exit(any(item["exit_code"] for item in result["comparisons"]))
