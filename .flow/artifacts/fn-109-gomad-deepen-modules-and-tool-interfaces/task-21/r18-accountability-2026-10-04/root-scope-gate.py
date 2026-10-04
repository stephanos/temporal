#!/usr/bin/env python3
"""Read-only scope and preservation gate for the additive R18 checkpoint."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[4]
ADMISSION = json.loads((HERE / "root-admission.json").read_text())
RECORD = json.loads((HERE / "root-verification.json").read_text())
FLOW = "/home/agent/.codex/scripts/flowctl"
BASE = ADMISSION["base_commit"]


def command(*args):
    return subprocess.check_output(args, cwd=ROOT)


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def original(path):
    return command("git", "show", BASE + ":" + path)


require(command("git", "rev-parse", "--show-toplevel").decode().strip() == str(ROOT), "repository root")
require(command("git", "branch", "--show-current").decode().strip() == ADMISSION["branch"], "branch")
require(RECORD["review_verdict"] == "EVIDENCE_PROGRESS_COMMIT_ONLY", "independent review")
allowed = set(RECORD["allowed_paths"])
record_path = str((HERE / "root-verification.json").relative_to(ROOT))
require(set(RECORD["file_sha256"]) == allowed - {record_path}, "every admitted input is bound")
expected_artifacts = {p for p in allowed if p.startswith(str(HERE.relative_to(ROOT)) + "/")}
actual_artifacts = {str(p.relative_to(ROOT)) for p in HERE.rglob("*") if p.is_file()}
require(actual_artifacts == expected_artifacts, "exact artifact directory scope")
for path, digest in RECORD["file_sha256"].items():
    require(sha((ROOT / path).read_bytes()) == digest, "bound file: " + path)
for path, edit in RECORD["document_insertions"].items():
    before = original(path).decode()
    require(before.count(edit["anchor"]) == 1, "unique anchor: " + path)
    expected = before.replace(edit["anchor"], edit["replacement"])
    require((ROOT / path).read_text() == expected, "exact additive reconstruction: " + path)
for path in RECORD["metadata_paths"]:
    before = json.loads(original(path))
    after = json.loads((ROOT / path).read_bytes())
    before.pop("updated_at", None)
    after.pop("updated_at", None)
    require(before == after, "metadata changed beyond timestamp: " + path)
task_path = ".flow/tasks/" + ADMISSION["task"] + ".md"
task_md = (ROOT / task_path).read_text()
require(task_md.split("## Done summary", 1)[0] == original(task_path).decode().split("## Done summary", 1)[0], "original task description and acceptance")
for stage in ("impl-review", "plan-sync"):
    require(task_md.count("stage: " + stage + " - ") == 1, "stage: " + stage)
require("TBD" not in task_md, "unfinished receipt")
task = json.loads(command(FLOW, "show", ADMISSION["task"], "--json"))
spec = json.loads(command(FLOW, "show", task["spec"], "--json"))
require(task["status"] == "blocked", "task acceptance remains open")
require(task["blocked_reason"] == RECORD["historical_blocked_reason"], "historical blocked checkpoint")
require(spec["status"] == "open" and spec["completion_review_status"] == "unknown", "parent acceptance remains open")
require(len(spec["tasks"]) == 36 and sum(t["status"] == "done" for t in spec["tasks"]) == 2, "unchanged task acceptance count")
changed = set(command("git", "diff", "--name-only", BASE).decode().splitlines())
require(changed <= allowed, "out-of-scope tracked changes: " + repr(sorted(changed - allowed)))
require(command("git", "diff", "--check", BASE) == b"", "all scoped tracked paths whitespace clean")
verifier = subprocess.run([sys.executable, str(HERE / "verify_accountability.py")], cwd=ROOT, capture_output=True)
require(verifier.returncode == 0, "primary accountability verifier: " + verifier.stderr.decode())
verification = json.loads(verifier.stdout)
require(verification["success"] is True, "primary verification success")
require(sha(verifier.stdout) == RECORD["primary_verification_stdout_sha256"], "fresh primary verification matches record")
staged = set(command("git", "diff", "--cached", "--name-only").decode().splitlines())
require(staged <= allowed, "foreign staged paths")
if "--staged" in sys.argv:
    require(staged == allowed, "exact complete staged scope")
    for path in staged:
        require(command("git", "show", ":" + path) == (ROOT / path).read_bytes(), "index bytes: " + path)
    require(command("git", "diff", "--cached", "--check") == b"", "staged whitespace clean")
print(json.dumps({"result": "PASS", "writes": 0, "head": command("git", "rev-parse", "HEAD").decode().strip(), "task": task["id"], "task_status": task["status"], "allowed_paths": len(allowed), "staged_paths": len(staged), "spec_done": 2, "spec_total": 36}))
