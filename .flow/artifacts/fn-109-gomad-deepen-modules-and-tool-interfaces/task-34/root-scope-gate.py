#!/usr/bin/env python3
"""Read-only scope audit for the task-34 source-progress checkpoint."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[4]
HERE = Path(__file__).resolve().parent
ADMISSION = json.loads((HERE / "root-admission.json").read_text())
CHECKS = json.loads((HERE / "root-source-checks.json").read_text())
SPEC = ADMISSION["task"].rsplit(".", 1)[0]
FLOW = "/home/agent/.codex/scripts/flowctl"
def command(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True)
def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def check(condition, message):
    if not condition:
        raise AssertionError(message)
check(command("git", "rev-parse", "--show-toplevel").strip() == str(ROOT), "root")
base = ADMISSION["base_commit"]
path = ADMISSION["source_path"]
old = command("git", "show", base + ":" + path)
before = "\t\tos.Stdin = original\n\t\treader.Close()\n"
after = "\t\tos.Stdin = original\n\t\tif err := reader.Close(); err != nil {\n\t\t\tt.Errorf(\"close coordinator request reader: %v\", err)\n\t\t}\n"
check(old.count(before) == 1, "unique admitted close")
check((ROOT / path).read_text() == old.replace(before, after, 1), "exact product edit")
check(sha(ROOT / path) == CHECKS["source_after_sha256"], "frozen source")
for name, digest in ADMISSION["protected_files"].items():
    check(sha(ROOT / name) == digest, "protected input: " + name)
for name, digest in CHECKS["immutable_proof_files"].items():
    check(sha(HERE / name) == digest, "immutable proof: " + name)
for name, digest in CHECKS["document_sha256"].items():
    check(sha(ROOT / name) == digest, "document freeze: " + name)
task_md = (ROOT / (".flow/tasks/" + ADMISSION["task"] + ".md")).read_text()
prefix = task_md.split("## Done summary", 1)[0]
check(hashlib.sha256(prefix.encode()).hexdigest() == CHECKS["original_task_prefix_sha256"], "original task")
for stage in ("impl-review", "plan-sync"):
    check(task_md.count("stage: " + stage + " - ") == 1, "stage outcome: " + stage)
check("TBD" not in task_md, "unfinished receipt placeholder")
task = json.loads(command(FLOW, "show", ADMISSION["task"], "--json"))
parent = json.loads(command(FLOW, "show", SPEC, "--json"))
consumer = json.loads(command(FLOW, "show", SPEC + ".21", "--json"))
check(task["status"] == "blocked", "qualification acceptance must remain blocked")
check(consumer["status"] == "blocked", "final consumer")
check(ADMISSION["task"] in consumer["depends_on"], "final evidence dependency")
check(parent["status"] == "open" and parent["completion_review_status"] == "unknown", "original full qualification")
check(len(parent["tasks"]) == 34 and sum(t["status"] == "done" for t in parent["tasks"]) == 2, "completion honesty")
allowed = set(CHECKS["allowed_paths"])
changed = set(command("git", "diff", "--name-only", base).splitlines())
check(changed <= allowed, "out-of-scope tracked changes: " + repr(sorted(changed - allowed)))
check(command("git", "diff", "--check", base) == "", "task scoped diff")
staged = set(command("git", "diff", "--cached", "--name-only").splitlines())
check(staged <= allowed, "foreign staged paths")
if "--staged" in sys.argv:
    check(staged == allowed, "stage exact complete task scope")
    for name in staged:
        indexed = subprocess.check_output(("git", "show", ":" + name), cwd=ROOT)
        check(indexed == (ROOT / name).read_bytes(), "index bytes: " + name)
    check(command("git", "diff", "--cached", "--check") == "", "staged diff")
print(json.dumps({"result": "PASS", "task": ADMISSION["task"], "head": command("git", "rev-parse", "HEAD").strip(), "source_sha256": CHECKS["source_after_sha256"], "protected_files": len(ADMISSION["protected_files"]), "immutable_proofs": len(CHECKS["immutable_proof_files"]), "allowed_paths": len(allowed), "staged_paths": len(staged), "task_status": task["status"], "spec_done": 2, "spec_total": 34, "writes": 0}))
