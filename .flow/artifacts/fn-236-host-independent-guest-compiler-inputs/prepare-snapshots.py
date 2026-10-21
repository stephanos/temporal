#!/usr/bin/env python3
"""Freeze the reviewed uncommitted fix without creating a commit or worktree."""
from pathlib import Path
import shutil
import subprocess
import sys

repository = Path(sys.argv[1]).resolve()
candidate = repository / ".flow/tmp/fn236-candidate"
proof = repository / ".flow/tmp/fn236-snapshot-proof"
proof.mkdir(exist_ok=True)
pin = "77d2e4e90eeea14711ebfd050f41f0ef3b1d0b1e"
overlays = ["model/scripts/guest.sh", "model/scripts/guest-rustc.py",
            "model/scripts/tests/test_guest.py", "model/scripts/tests/test_guest_freshness.py",
            "model/scripts/tests/test_guest_rustc.py", "model/GUEST_BUILDS.md",
            "model/temporal/hosts/go/umpire_temporal_guest.wasm"]
manifest = repository / ".flow/artifacts/fn-236-host-independent-guest-compiler-inputs/source-manifest.py"
for platform in ["linux", "macos"]:
    for lane in ["a", "b"]:
        source = proof / platform / ("source-" + lane)
        source.parent.mkdir(exist_ok=True)
        if not source.exists():
            subprocess.run(["git", "clone", "--shared", "--no-checkout", str(repository), str(source)], check=True)
            subprocess.run(["git", "-C", str(source), "checkout", "--detach", pin], check=True)
        assert subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"]).decode().strip() == pin
        for name in overlays:
            shutil.copy2(candidate / name, source / name)
        data = subprocess.check_output([sys.executable, str(manifest), str(source)])
        frozen = proof / "frozen-source.json"
        if frozen.exists():
            assert frozen.read_bytes() == data
        else:
            frozen.write_bytes(data)
shutil.copy2(proof / "frozen-source.json", manifest.parent / "frozen-source.json")
