#!/usr/bin/env python3
"""Read-only binding audit of the twelve original writer command receipts."""
from collections import Counter
from datetime import datetime
import hashlib
import json
from pathlib import Path
import re

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]
def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
admission = json.loads((HERE / "root-admission.json").read_text())
evidence = json.loads((HERE / "evidence.json").read_text())
source = ROOT / admission["source_path"]
assert sha(source) == evidence["source_sha256"]
for path, digest in admission["protected_files"].items():
    assert sha(ROOT / path) == digest, path
assert len(evidence["commands"]) == 12
previous_end = None
for entry in evidence["commands"]:
    receipt = json.loads((ROOT / entry["receipt"]).read_text())
    assert receipt["source_before"] == receipt["source_after"] and receipt["stability"]
    freeze = receipt["source_before"]
    expected = admission["source_before_sha256"] if entry["phase"] == "baseline" else evidence["source_sha256"]
    assert freeze["fixture_sha256"] == expected
    assert freeze["protected_count"] == 1044 and freeze["protected_unchanged"]
    assert freeze["admission_sha256"] == sha(HERE / "root-admission.json")
    for path, digest in freeze["tools"].items():
        assert sha(Path(path)) == digest
    assert receipt["signal"] is None
    assert receipt["exit_code"] == entry["exit_code"] == (1 if entry["check"] == "lint" else 0)
    assert receipt["elapsed_seconds"] == entry["elapsed_seconds"]
    start, end = map(datetime.fromisoformat, (receipt["start"], receipt["end"]))
    assert start <= end and (previous_end is None or previous_end <= start)
    previous_end = end
    log = ROOT / receipt["log"]
    assert sha(log) == receipt["log_sha256"]
    content = log.read_text()
    if entry["check"] in ("private-mode", "portable-cli", "boundaries"):
        count = len(re.findall(r"^--- PASS:", content, re.M))
        assert count == entry["top_level_passes"] == {"private-mode": 1, "portable-cli": 34, "boundaries": 5}[entry["check"]]
        assert not re.search(r"^--- (FAIL|SKIP):", content, re.M)
        assert "-count=1" in receipt["command"] and "-tags test_dep" in receipt["command"]
    if entry["check"] == "gofmt":
        assert not content
def diagnostics(name):
    lines = (HERE / name).read_text().splitlines(keepends=True)
    headers = [i for i, line in enumerate(lines) if re.match(r"tools/gomad3/.*\.go:\d+:\d+: ", line)]
    blocks = []
    for i in headers:
        block = lines[i]
        for line in lines[i + 1:]:
            if line[:1] not in ("\t", " "):
                break
            block += line
        blocks.append(block)
    return Counter(blocks)
before, after = diagnostics("baseline-lint.log"), diagnostics("final-lint.log")
resolved, introduced = before - after, after - before
assert sum(before.values()) == 54 and sum(after.values()) == 53
assert not introduced and sum(resolved.values()) == 1
removed = next(iter(resolved))
assert "characterization_test.go:92:15:" in removed and "`reader.Close`" in removed and "(errcheck)" in removed
assert sum(n for x, n in after.items() if "(errcheck)" in x) == 52
assert sum(n for x, n in after.items() if "(staticcheck)" in x) == 1
assert (HERE / "lint-delta.json").exists()
print(json.dumps({"result": "PASS", "writer_receipts": 12, "serial_commands": True, "protected_inputs": 1044, "source_sha256": sha(source), "baseline_findings": 54, "final_findings": 53, "resolved": 1, "introduced": 0, "remaining": {"errcheck": 52, "staticcheck": 1}, "writes": 0}))
