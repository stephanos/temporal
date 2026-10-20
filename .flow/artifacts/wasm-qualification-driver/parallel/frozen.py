#!/usr/bin/env python3
"""Guard the frozen candidate: HEAD, clean status, every manifest input, symlinks, staged fixtures.

Same checks as q1-serial/frozen.py with the q1 constants passed as arguments.
Concurrent stages: --scratch-registry/--self allow `??` (untracked) entries
matching the declared in-tree scratch globs of *other* stages whose run
interval overlaps this check (see scheduler.py RealExecutor). Tracked
modifications, deletions and every manifest digest are never exempt; without
--scratch-registry (driver start, end, solo/exclusive stages) the check is the
strict serial one.
"""
import argparse
import fnmatch
import hashlib
import json
import os
import subprocess
import time
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("repo", type=Path)
parser.add_argument("root", type=Path)
parser.add_argument("expected_head")
parser.add_argument("expected_manifest_sha256")
parser.add_argument("--scratch-registry", type=Path)
parser.add_argument("--self", dest="self_label", default="")
args = parser.parse_args()
repo, root = args.repo, args.root
digest = lambda p: hashlib.file_digest(open(p, "rb"), "sha256").hexdigest()
pre = json.loads((root / "preflight.json").read_text())
manifest = root / "source.sha256"
assert digest(manifest) == pre["source_manifest_sha256"] == args.expected_manifest_sha256, "manifest changed"
env = dict(os.environ, GIT_DISCOVERY_ACROSS_FILESYSTEM="1", GIT_OPTIONAL_LOCKS="0")
head = subprocess.check_output(["git", "-C", str(repo), "rev-parse", "HEAD"], text=True, env=env).strip()
assert head == pre["head"] == args.expected_head, "HEAD changed"


def overlapping_scratch(before_ns, after_ns):
    globs = []
    if not args.scratch_registry or not args.scratch_registry.is_dir():
        return globs
    for entry in args.scratch_registry.glob("*.json"):
        try:
            record = json.loads(entry.read_text())
        except (OSError, ValueError):
            continue  # being replaced; the atomic rename completes before the stage process starts
        if record.get("label") == args.self_label or not record.get("scratch"):
            continue
        if record.get("start_ns", 0) <= after_ns and record.get("end_ns", after_ns) >= before_ns:
            globs.extend(record["scratch"])
    return globs


def exempt(path, globs):
    parts = path.rstrip("/").split("/")
    for pattern in globs:
        width = len(pattern.split("/"))
        if len(parts) > width and fnmatch.fnmatchcase("/".join(parts[:width]), pattern):
            return True
    return False


before_ns = time.time_ns()
status = subprocess.check_output(["git", "-C", str(repo), "status", "--porcelain", "--untracked-files=all"], env=env)
after_ns = time.time_ns()
lines = status.decode().splitlines()
if lines and args.scratch_registry:
    globs = overlapping_scratch(before_ns, after_ns)
    lines = [line for line in lines if not (line.startswith("?? ") and exempt(line[3:], globs))]
assert not lines, "worktree not clean:\n" + "\n".join(lines)[:4000]
rows = manifest.read_text().splitlines()
assert len(rows) == pre["source_files"]
for line in rows:
    wanted, name = line[:64], line[66:]
    p = Path(name)
    assert p.is_file() and not p.is_symlink(), name
    assert digest(p) == wanted, f"frozen input changed: {name}"
for name, target in json.loads((root / "source-symlinks.json").read_text()).items():
    assert Path(name).is_symlink() and os.readlink(name) == target, f"symlink changed: {name}"
for name, wanted in pre["staged_sha256"].items():
    assert digest(repo / pre["staged_temporal_source"] / name) == wanted, f"staged fixture changed: {name}"
print(f"frozen source verified: {len(rows)} inputs, HEAD, clean status, symlinks, staged fixtures")
