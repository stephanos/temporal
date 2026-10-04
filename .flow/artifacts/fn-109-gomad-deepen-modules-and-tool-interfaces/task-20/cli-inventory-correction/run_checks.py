#!/usr/bin/env python3
"""Capture one foreground run of each admitted post-correction Quick gate."""
import datetime
import hashlib
import json
import os
import pathlib
import subprocess
import time

AREA = pathlib.Path(__file__).resolve().parent
ROOT = pathlib.Path(__file__).resolve().parents[5]
GO = "/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go"
ENV = {"GOWORK": "off", "GOTOOLCHAIN": "local", "GOENV": "off", "GOFLAGS": "",
       "PATH": str(pathlib.Path(GO).parent) + ":/usr/local/bin:/usr/bin:/bin"}


def source_digest():
    paths = json.loads((AREA / "before.json").read_text())["source"]
    inventory = {p: hashlib.sha256((ROOT / "tools/gomad3" / p).read_bytes()).hexdigest() for p in paths}
    return hashlib.sha256(json.dumps(inventory, sort_keys=True).encode()).hexdigest()


receipts = []
for name, argv in [
    ("final-focused", [GO, "test", "-v", "-count=1", "-tags", "test_dep", ".", "-run", "TestCurrentVocabularyHasNoLegacyCampaignBoundary|TestMakeTargetsMatchTheirOwnership"]),
    ("final-validate", ["make", "validate"]),
]:
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock_start = time.monotonic()
    before = source_digest()
    with (AREA / (name + ".log")).open("wb") as log:
        result = subprocess.run(argv, cwd=ROOT / "tools/gomad3", env={**os.environ, **ENV}, stdout=log, stderr=subprocess.STDOUT, timeout=600)
    receipt = {"name": name, "argv": argv, "cwd": str(ROOT / "tools/gomad3"), "environment_overrides": ENV,
               "started_at": start, "finished_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
               "elapsed_seconds": time.monotonic() - clock_start, "exit_code": result.returncode,
               "source_before": before, "source_after": source_digest(), "log": name + ".log",
               "log_sha256": hashlib.sha256((AREA / (name + ".log")).read_bytes()).hexdigest()}
    receipts.append(receipt)
    (AREA / "final-quick-receipts.json").write_text(json.dumps(receipts, indent=2) + "\n")
    print(json.dumps(receipt), flush=True)
    if result.returncode or receipt["source_before"] != receipt["source_after"]:
        raise SystemExit(result.returncode or 1)
