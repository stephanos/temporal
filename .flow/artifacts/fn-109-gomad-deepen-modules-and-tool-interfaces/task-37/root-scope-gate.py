#!/usr/bin/env python3
"""Read-only final gate for the qualification-storage source checkpoint."""
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]
FLOW = "/home/agent/.codex/scripts/flowctl"
ADMISSION = json.loads((HERE / "root-admission.json").read_bytes())
RECORD = json.loads((HERE / "root-verification.json").read_bytes())
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
require(RECORD["review_verdict"] == "SOURCE_PROGRESS_COMMIT_ONLY", "independent source review")
allowed = set(RECORD["allowed_paths"])
record_path = str((HERE / "root-verification.json").relative_to(ROOT))
require(set(RECORD["file_sha256"]) == allowed - {record_path}, "every admitted final input bound")
for path, digest in RECORD["file_sha256"].items():
    require(sha((ROOT / path).read_bytes()) == digest, "bound file: " + path)
expected = {p for p in allowed if p.startswith(str(HERE.relative_to(ROOT)) + "/")}
actual = {str(p.relative_to(ROOT)) for p in HERE.rglob("*") if p.is_file()}
require(actual == expected, "exact frozen artifact directory")
for path, edit in RECORD["document_insertions"].items():
    before = original(path).decode()
    require(before.count(edit["anchor"]) == 1, "unique document anchor: " + path)
    require((ROOT / path).read_text() == before.replace(edit["anchor"], edit["replacement"]), "exact additive document: " + path)
spec_json = ".flow/specs/" + RECORD["spec"] + ".json"
before = json.loads(original(spec_json))
after = json.loads((ROOT / spec_json).read_bytes())
for value in (before, after):
    value.pop("updated_at", None)
require(before == after, "parent metadata unchanged beyond timestamp")
task21_json = ".flow/tasks/" + RECORD["spec"] + ".21.json"
before = json.loads(original(task21_json))
after = json.loads((ROOT / task21_json).read_bytes())
before["depends_on"].append(ADMISSION["task"])
for value in (before, after):
    value.pop("updated_at", None)
require(before == after, "task21 only adds new repair dependency")
task_path = ".flow/tasks/" + ADMISSION["task"] + ".md"
task_md = (ROOT / task_path).read_text()
require(sha(task_md.split("## Done summary", 1)[0].encode()) == RECORD["original_task_prefix_sha256"], "original task requirements")
for stage in ("impl-review", "plan-sync"):
    require(task_md.count("stage: " + stage + " - ") == 1, "exact stage: " + stage)
require("TBD" not in task_md, "finished receipt text")
task_json_path = ".flow/tasks/" + ADMISSION["task"] + ".json"
task_metadata = json.loads((ROOT / task_json_path).read_bytes())
original_task_metadata = dict(RECORD["original_task_metadata"])
for value in (task_metadata, original_task_metadata):
    value.pop("updated_at", None)
require(task_metadata == original_task_metadata, "original task metadata unchanged beyond timestamp")
task = json.loads(command(FLOW, "show", ADMISSION["task"], "--json"))
task21 = json.loads(command(FLOW, "show", RECORD["spec"] + ".21", "--json"))
spec = json.loads(command(FLOW, "show", RECORD["spec"], "--json"))
require(task["status"] == "blocked", "repair acceptance remains open")
require(task["blocked_reason"] == RECORD["blocked_reason"], "repair blocked reason")
require(task21["status"] == "blocked" and task21["blocked_reason"] == RECORD["historical_task21_blocked_reason"], "historical qualification checkpoint")
require(spec["status"] == "open" and spec["completion_review_status"] == "unknown", "parent acceptance remains open")
require(len(spec["tasks"]) == 37 and sum(t["status"] == "done" for t in spec["tasks"]) == 2, "actual accepted counts")
for path, digest in ADMISSION["protected_sha256"].items():
    require(sha((ROOT / path).read_bytes()) == digest, "protected input: " + path)
changed = set(command("git", "diff", "--name-only", BASE).decode().splitlines())
require(changed <= allowed, "out-of-scope tracked changes: " + repr(sorted(changed - allowed)))
whitespace = subprocess.run(["git", "diff", "--check", BASE], cwd=ROOT, capture_output=True)
exception = RECORD.get("whitespace_exception")
if exception is None:
    require(whitespace.returncode == 0 and whitespace.stdout == whitespace.stderr == b"", "tracked whitespace clean")
else:
    require(sha((ROOT / exception["path"]).read_bytes()) == exception["sha256"], "immutable archive identity")
    require(whitespace.returncode == exception["exit_code"] and whitespace.stdout.decode() == exception["stdout"] and whitespace.stderr == b"", "only exact archive whitespace exception")
    remaining = sorted(changed - {exception["path"]})
    clean = subprocess.run(["git", "diff", "--check", BASE, "--", *remaining], cwd=ROOT, capture_output=True)
    require(clean.returncode == 0 and clean.stdout == clean.stderr == b"", "all other scoped paths whitespace clean")
protected = {}
for entry in command("git", "ls-tree", "-rz", "--full-tree", BASE).split(b"\0"):
    if not entry:
        continue
    metadata, path = entry.split(b"\t", 1)
    p = path.decode()
    if p in allowed:
        continue
    mode, kind, blob = metadata.decode().split()
    protected[p] = (mode, kind, blob)
for p, (mode, kind, blob) in protected.items():
    full = ROOT / p
    if mode == "160000":
        require(kind == "commit" and full.is_dir(), "protected Gitlink: " + p)
        if (full / ".git").exists():
            require(command("git", "-C", str(full), "rev-parse", "HEAD").decode().strip() == blob, "Gitlink recorded HEAD")
            require(command("git", "-C", str(full), "status", "--porcelain", "--untracked-files=all") == b"", "Gitlink clean")
        else:
            require(not any(full.iterdir()), "empty uninitialized Gitlink")
        continue
    require(kind == "blob", "protected blob: " + p)
    info = full.lstat()
    if mode == "120000":
        require(stat.S_ISLNK(info.st_mode), "protected symlink: " + p)
        content = os.fsencode(os.readlink(full))
    else:
        require(stat.S_ISREG(info.st_mode), "protected regular file: " + p)
        require(("100755" if info.st_mode & stat.S_IXUSR else "100644") == mode, "protected mode: " + p)
        content = full.read_bytes()
    observed = hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()
    require(observed == blob, "protected bytes: " + p)
proof = subprocess.run(RECORD["primary_verification_command"], cwd=ROOT, capture_output=True)
require(proof.returncode == 0, "fresh primary proof: " + proof.stderr.decode())
require(sha(proof.stdout) == RECORD["primary_verification_stdout_sha256"], "fresh proof output binding")
validation = json.loads(command(FLOW, "validate", "--spec", RECORD["spec"], "--json"))
require(validation["valid"] is True and not validation["errors"], "Flow validation")
staged = set(command("git", "diff", "--cached", "--name-only").decode().splitlines())
require(staged <= allowed, "foreign index paths")
if "--staged" in sys.argv:
    require(staged == allowed, "exact complete staged scope")
    for path in staged:
        require(command("git", "show", ":" + path) == (ROOT / path).read_bytes(), "index bytes: " + path)
    check = subprocess.run(["git", "diff", "--cached", "--check"], cwd=ROOT, capture_output=True)
    if exception is None:
        require(check.returncode == 0 and check.stdout == check.stderr == b"", "staged whitespace clean")
    else:
        require(check.returncode == exception["exit_code"] and check.stdout.decode() == exception["stdout"] and check.stderr == b"", "exact staged archive exception")
print(json.dumps({"result": "PASS", "writes": 0, "head": command("git", "rev-parse", "HEAD").decode().strip(), "task": task["id"], "task_status": task["status"], "allowed_paths": len(allowed), "staged_paths": len(staged), "protected_tracked_paths": len(protected), "spec_done": 2, "spec_total": 37}))
