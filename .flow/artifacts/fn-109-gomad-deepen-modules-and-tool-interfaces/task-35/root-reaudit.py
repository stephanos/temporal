#!/usr/bin/env python3
"""Recheck frozen worker evidence without updating archived outputs."""
from datetime import datetime
import hashlib
import json
from pathlib import Path
import re

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]
admission = json.loads((HERE / "root-admission.json").read_text())
evidence = json.loads((HERE / "evidence.json").read_text())
archived = json.loads((HERE / "source-check.json").read_text())
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
freeze = {}
for line in (HERE / "worker-freeze.sha256").read_text().splitlines():
    digest, name = line.split("  ", 1)
    assert sha(ROOT / name) == digest, name
    freeze[name] = digest
assert len(freeze) == 39
for name, digest in evidence["sources"].items():
    assert sha(ROOT / name) == digest, name
for name, digest in admission["protected_files"].items():
    assert sha(ROOT / name) == digest, name

def verify_json(path, contents):
    assert json.loads(contents) == json.loads(path.read_text()), str(path)

audit_path = HERE / "source_audit.py"
code = audit_path.read_text()
for name in ("lint-delta.json", "source-check.json"):
    old = f"(PROOF / '{name}').write_text("
    assert code.count(old) == 1
    code = code.replace(old, f"verify_json(PROOF / '{name}',")
old = "sorted(PROOF.glob('*.receipt.json'))"
assert code.count(old) == 1
names = [Path(item["path"]).name for item in archived["receipts"]]
code = code.replace(old, "[PROOF / name for name in " + repr(names) + "]")
exec(compile(code, str(audit_path), "exec"), {"__file__": str(audit_path), "verify_json": verify_json})

counts = {"baseline-package": 20, "baseline-focused": 10, "baseline-boundaries": 5,
          "controls-focused": 14, "preservation-focused": 14,
          "final-package": 24, "final-focused": 14, "final-boundaries": 5}
receipts = []
for name in evidence["receipts"]:
    receipt = json.loads((HERE / name).read_text())
    assert receipt["source_before"] == receipt["source_after"] and receipt["stability"]
    before = receipt["source_before"]
    assert before["admission_sha256"] == sha(HERE / "root-admission.json")
    assert before["protected_count"] == 1043 and before["protected_unchanged"]
    for tool, digest in before["tools"].items():
        assert sha(Path(tool)) == digest, tool
    phase = name.removesuffix(".receipt.json")
    expected = admission["source_before_sha256"] if phase.startswith("baseline-") else evidence["sources"]
    if phase in ("controls-focused", "preservation-focused"):
        assert before["sources"][admission["source_paths"][0]] == admission["source_before_sha256"][admission["source_paths"][0]]
    else:
        assert before["sources"] == expected, phase
    log = ROOT / receipt["log"]
    assert sha(log) == receipt["log_sha256"]
    assert receipt["exit_code"] == (1 if phase == "baseline-lint" else 0), phase
    if phase in counts:
        assert len(re.findall(r"^--- PASS:", log.read_text(), re.M)) == counts[phase], phase
        assert "-tags test_dep" in receipt["command"] and "-count=1" in receipt["command"]
    if phase.endswith(("errortype", "gofmt")):
        assert log.read_bytes() == b"", phase
    assert datetime.fromisoformat(receipt["end"]) >= datetime.fromisoformat(receipt["start"])
    receipts.append(receipt)
ordered = sorted(receipts, key=lambda item: item["start"])
assert all(datetime.fromisoformat(a["end"]) <= datetime.fromisoformat(b["start"]) for a, b in zip(ordered, ordered[1:]))
assert len(receipts) == 16
for name, digest in freeze.items():
    assert sha(ROOT / name) == digest, name
review = json.loads((HERE / "independent-source-review.json").read_text())
assert review["verdict"] == "PERMIT_SOURCE_PROGRESS_CHECKPOINT"
assert review["sources"] == evidence["sources"]
assert not review["formal_impl_review"] and not review["ship"] and not review["task_complete"]
assert all(not findings for findings in review["findings"].values())
review_path = HERE / "review-check.py"
review_code = review_path.read_text()
assert review_code.count("\nname = sys.argv[1]\n") == 1
namespace = {"__file__": str(review_path)}
exec(compile(review_code.split("\nname = sys.argv[1]\n", 1)[0], str(review_path), "exec"), namespace)
namespace["freeze"]()
namespace["audit"]()
review_counts = {"package": 24, "focused": 14, "boundaries": 5}
writer_environment = receipts[-1]["environment"]
for item in review["checks"]:
    receipt = json.loads((HERE / item["receipt"]).read_text())
    assert receipt["source_before"] == receipt["source_after"] and receipt["stability"]
    assert receipt["source_before"]["sources"] == evidence["sources"]
    assert receipt["environment"] == writer_environment
    assert receipt["exit_code"] == item["exit_code"] == 0
    log = ROOT / receipt["log"]
    assert sha(log) == receipt["log_sha256"] == item["raw_log_sha256"]
    if item["name"] in review_counts:
        count = len(re.findall(r"^--- PASS:", log.read_text(), re.M))
        assert count == receipt["top_level_pass_count"] == item["top_level_pass_count"] == review_counts[item["name"]]
    if item["name"] == "lint":
        assert log.read_text().strip() == "0 issues."
    if item["name"] in ("errortype", "gofmt"):
        assert not log.read_bytes()
    receipts.append({"start": receipt["start_utc"], "end": receipt["end_utc"]})
ordered = sorted(receipts, key=lambda item: item["start"])
assert all(datetime.fromisoformat(a["end"]) <= datetime.fromisoformat(b["start"]) for a, b in zip(ordered, ordered[1:]))
assert len(review["checks"]) == 7 and len(review["writes"]) == 17
assert all((ROOT / path).is_file() for path in review["writes"])
print(json.dumps({"result": "PASS", "source_sha256": evidence["sources"], "protected_inputs": 1043,
                  "worker_frozen_files": len(freeze), "writer_receipts": 16, "review_receipts": 7,
                  "archived_audit_receipts": len(names), "serial": True, "counts": counts,
                  "lint": "6 -> 0", "writes": 0,
                  "audit_adaptation": "Compare two generated JSON outputs and select original 15 audit receipts; audit-source receipt is checked separately."}))
