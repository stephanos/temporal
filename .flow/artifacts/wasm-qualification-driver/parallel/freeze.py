#!/usr/bin/env python3
"""Freeze a clean candidate worktree for run-validation.sh (q1 freeze format).

  freeze.py REPO FREEZE_ROOT STAGED_RELATIVE [--manifest-only]

Writes FREEZE_ROOT/{source.sha256,source-symlinks.json,preflight.json}:
every git-tracked regular file at HEAD outside .flow/ (absolute paths, git
index order), symlinks bound by target text, gitlinks excluded. Unless
--manifest-only, it also stages the Temporal recovery/broker fixture sources
into REPO/STAGED_RELATIVE (must be git-ignored) and records their digests.
Prints the config lines for config/<candidate>.env.
"""
import argparse
import hashlib
import json
import os
import shutil
import subprocess
from pathlib import Path

FIXTURES = {"recovery_test.go": "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go",
            "replication_test.go": "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go"}

parser = argparse.ArgumentParser()
parser.add_argument("repo", type=Path)
parser.add_argument("root", type=Path)
parser.add_argument("staged_relative")
parser.add_argument("--manifest-only", action="store_true")
args = parser.parse_args()
repo = args.repo.resolve()
env = dict(os.environ, GIT_DISCOVERY_ACROSS_FILESYSTEM="1")
git = lambda *a: subprocess.check_output(["git", "-C", str(repo), *a], env=env)
digest = lambda p: hashlib.file_digest(open(p, "rb"), "sha256").hexdigest()
head = git("rev-parse", "HEAD").decode().strip()
assert git("status", "--porcelain", "--untracked-files=all") == b"", "worktree not clean"
rows, links, gitlinks, total = [], {}, [], 0
for entry in git("ls-files", "-s", "-z").split(b"\0"):
    if not entry:
        continue
    meta, name = entry.decode().split("\t", 1)
    mode = meta.split()[0]
    if name.startswith(".flow/"):
        continue
    path = repo / name
    if mode == "160000":
        gitlinks.append(name)
    elif mode == "120000":
        assert path.is_symlink(), name
        links[str(path)] = os.readlink(path)
    else:
        assert mode in ("100644", "100755") and path.is_file() and not path.is_symlink(), name
        rows.append(f"{digest(path)}  {path}")
        total += path.stat().st_size
args.root.mkdir(parents=True, exist_ok=False)
manifest = args.root / "source.sha256"
manifest.write_text("\n".join(rows) + "\n")
(args.root / "source-symlinks.json").write_text(json.dumps(links, indent=1) + "\n")
staged = {}
if not args.manifest_only:
    target = repo / args.staged_relative
    assert subprocess.run(["git", "-C", str(repo), "check-ignore", "-q", args.staged_relative + "/probe"], env=env).returncode == 0
    target.mkdir(parents=True, exist_ok=False)
    for name, source in FIXTURES.items():
        shutil.copyfile(repo / source, target / name)
        staged[name] = digest(target / name)
        assert staged[name] == digest(repo / source)
    assert git("status", "--porcelain", "--untracked-files=all") == b"", "staging dirtied the worktree"
preflight = {"head": head, "repo": str(repo), "source_files": len(rows), "source_bytes": total,
             "source_manifest_sha256": digest(manifest),
             "selection": "every git-tracked regular file at HEAD outside .flow/; symlinks bound by target text "
                          "(regular targets included); gitlink submodule entries (unpopulated) excluded",
             "gitlinks_excluded": gitlinks, "staged_temporal_source": args.staged_relative, "symlinks": len(links),
             "staged_sha256": staged}
(args.root / "preflight.json").write_text(json.dumps(preflight, indent=2) + "\n")
print(f"QUAL_REPO={repo}\nFREEZE_ROOT={args.root.resolve()}\nEXPECTED_HEAD={head}\n"
      f"EXPECTED_SOURCE_SHA256={preflight['source_manifest_sha256']}\nSTAGED_FIXTURE_RELATIVE={args.staged_relative}")
