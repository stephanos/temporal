#!/usr/bin/env python3
"""Read-only scope gate for the task-36 source-progress checkpoint."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[4]
HERE = Path(__file__).resolve().parent
ADMISSION = json.loads((HERE / "root-admission.json").read_text())
CHECKS = json.loads((HERE / "root-source-checks.json").read_text())
FLOW = "/home/agent/.codex/scripts/flowctl"
SPEC = ADMISSION["task"].rsplit(".", 1)[0]


def command(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def check(condition, message):
    if not condition:
        raise AssertionError(message)


def diff_check(*args):
    result = subprocess.run(args, cwd=ROOT, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    archive = HERE / "audit-environment.log"
    name = str(archive.relative_to(ROOT))
    expected = name + ":10: new blank line at EOF.\n"
    check(sha(archive) == "ecc5d80c8f231082f3eec8d2705ed3d3ac9c9b6c7b813c310e6cc39b332d6b35", "immutable raw environment")
    check(result.returncode == 2 and result.stdout == expected and result.stderr == "", "only fixed archive EOF warning")
    paths = sorted(set(CHECKS["allowed_paths"]) - {name})
    check(command(*args, "--", *paths) == "", "all other scoped paths diff clean")


check(command("git", "rev-parse", "--show-toplevel").strip() == str(ROOT), "root")
check(CHECKS["metadata_review_pending"] is False, "metadata review unfinished")
for name in ADMISSION["source_paths"]:
    check(sha(ROOT / name) == CHECKS["source_after_sha256"][name], "source: " + name)
for name, digest in ADMISSION["protected_files"].items():
    check(sha(ROOT / name) == digest, "protected input: " + name)
for name, digest in CHECKS["immutable_proof_files"].items():
    check(sha(HERE / name) == digest, "proof: " + name)
for name, digest in CHECKS["document_sha256"].items():
    check(sha(ROOT / name) == digest, "document: " + name)
task_md = (ROOT / (".flow/tasks/" + ADMISSION["task"] + ".md")).read_text()
prefix = task_md.split("## Done summary", 1)[0]
check(hashlib.sha256(prefix.encode()).hexdigest() == CHECKS["original_task_prefix_sha256"], "original acceptance")
for stage in ("impl-review", "plan-sync"):
    check(task_md.count("stage: " + stage + " - ") == 1, "stage: " + stage)
check("TBD" not in task_md, "unfinished receipt")
task = json.loads(command(FLOW, "show", ADMISSION["task"], "--json"))
parent = json.loads(command(FLOW, "show", SPEC, "--json"))
consumer = json.loads(command(FLOW, "show", SPEC + ".21", "--json"))
check(task["status"] == "blocked" and consumer["status"] == "blocked", "acceptance remains open")
check(ADMISSION["task"] in consumer["depends_on"], "final verifier dependency")
check(parent["status"] == "open" and parent["completion_review_status"] == "unknown", "full qualification")
check(len(parent["tasks"]) == 36 and sum(t["status"] == "done" for t in parent["tasks"]) == 2, "completion count")
allowed = set(CHECKS["allowed_paths"])
changed = set(command("git", "diff", "--name-only", ADMISSION["base_commit"]).splitlines())
check(changed <= allowed, "out-of-scope changes: " + repr(sorted(changed - allowed)))
diff_check("git", "diff", "--check", ADMISSION["base_commit"])
staged = set(command("git", "diff", "--cached", "--name-only").splitlines())
check(staged <= allowed, "foreign staged paths")
if "--staged" in sys.argv:
    check(staged == allowed, "stage exact complete task scope")
    for name in staged:
        indexed = subprocess.check_output(("git", "show", ":" + name), cwd=ROOT)
        check(indexed == (ROOT / name).read_bytes(), "index bytes: " + name)
    diff_check("git", "diff", "--cached", "--check")
print(json.dumps({"result": "PASS", "task": ADMISSION["task"], "head": command("git", "rev-parse", "HEAD").strip(), "source_sha256": CHECKS["source_after_sha256"], "protected_files": len(ADMISSION["protected_files"]), "immutable_proofs": len(CHECKS["immutable_proof_files"]), "allowed_paths": len(allowed), "staged_paths": len(staged), "task_status": task["status"], "spec_done": 2, "spec_total": 36, "writes": 0}))
