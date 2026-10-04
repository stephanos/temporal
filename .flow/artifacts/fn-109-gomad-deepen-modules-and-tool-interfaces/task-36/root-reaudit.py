#!/usr/bin/env python3
"""Verify archived task-36 command bindings without changing any artifact."""
import hashlib
import json
from pathlib import Path
from datetime import datetime

ROOT = Path(__file__).resolve().parents[4]
HERE = Path(__file__).resolve().parent


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


evidence = json.loads((HERE / "evidence.json").read_text())
admission = json.loads((HERE / "root-admission.json").read_text())
for name, digest in admission["protected_files"].items():
    assert sha(ROOT / name) == digest, name
for name, digest in evidence["source_after_sha256"].items():
    assert sha(ROOT / name) == digest, name
receipts = []
for entry in evidence["receipts"]:
    path = ROOT / entry["path"]
    assert sha(path) == entry["sha256"], path
    receipt = json.loads(path.read_text())
    assert sha(ROOT / receipt["log"]) == receipt["log_sha256"], path
    assert receipt["source_before"] == receipt["source_after"], path
    assert receipt["stability"] is True, path
    assert receipt["argv"] and receipt["cwd"] and receipt["environment"], path
    assert datetime.fromisoformat(receipt["end"]) >= datetime.fromisoformat(receipt["start"]), path
    snap = receipt["source_before"]
    assert snap["admission_sha256"] == sha(HERE / "root-admission.json"), path
    assert snap["protected_count"] == 1044 and snap["protected_unchanged"] is True, path
    for name, digest in snap["tools"].items():
        assert sha(Path(name)) == digest, name
    if entry["status"] == "passed":
        assert receipt["exit_code"] == 0, path
    elif entry["status"] == "expected_analyzer_red":
        assert receipt["exit_code"] == 1, path
    elif entry["status"] == "raw_archive_warning":
        assert receipt["exit_code"] == 3, path
    else:
        assert entry["status"] == "inconclusive" and path.name == "diff-product.receipt.json", path
    receipts.append({"path": entry["path"], "status": entry["status"], "exit_code": receipt["exit_code"]})
audit = HERE / "source_audit.py"
source = audit.read_text()
write = "(PROOF / 'source-preservation.json').write_text(json.dumps(report, indent=2) + '\\n')"
assert source.count(write) == 1
namespace = {"__file__": str(audit), "__name__": "__task36_readonly_audit__"}
exec(compile(source.replace(write, "assert report == json.loads((PROOF / 'source-preservation.json').read_text())", 1), str(audit), "exec"), namespace)
print(json.dumps({"result": "PASS", "writes": 0, "worker_receipts": receipts, "protected_count": 1044, "source_after_sha256": evidence["source_after_sha256"], "audit_sha256": sha(audit)}))
