"""Observe source refs, indexes and dirty candidate bytes without changing them."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess


def git(root, *arguments):
    return subprocess.check_output(
        ["git", "--no-optional-locks", "--no-replace-objects", "-C", str(root), *arguments],
        env={**os.environ, "GIT_OPTIONAL_LOCKS": "0"},
    )


def fingerprint(path):
    try:
        info = os.lstat(path)
    except FileNotFoundError:
        return {"kind": "missing"}
    if stat.S_ISLNK(info.st_mode):
        payload = os.readlink(path)
        mode = "120000"
    elif stat.S_ISREG(info.st_mode):
        with open(path, "rb") as stream:
            payload = stream.read()
        mode = "100755" if info.st_mode & 0o111 else "100644"
    else:
        return {"kind": "non-file", "type": stat.S_IFMT(info.st_mode)}
    return {"mode": mode, "size": len(payload), "sha256": hashlib.sha256(payload).hexdigest()}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-repo", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = args.source_repo.resolve()
    excluded = [":(exclude).flow", ":(exclude)MILESTONES.md", ":(exclude)MIGRATE.md"]
    rows = []
    registry = git(root, "worktree", "list", "--porcelain", "-z")
    for field in registry.split(b"\0"):
        if not field.startswith(b"worktree "):
            continue
        checkout = Path(os.fsdecode(field.removeprefix(b"worktree "))).resolve()
        arguments = ["status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all"]
        if checkout == root:
            arguments += ["--", ".", *excluded]
        status = git(checkout, *arguments)
        dirty = []
        for entry in status.split(b"\0"):
            if entry:
                name = entry[3:]
                dirty.append({
                    "status": entry[:2].decode("ascii"), "path": os.fsdecode(name),
                    **fingerprint(os.path.join(os.fsencode(checkout), name)),
                })
        index_arguments = ["ls-files", "--stage", "-z"]
        if checkout == root:
            index_arguments += ["--", ".", *excluded]
        rows.append({
            "path": str(checkout),
            "head": git(checkout, "rev-parse", "HEAD").decode().strip(),
            "index_entries_sha256": hashlib.sha256(git(checkout, *index_arguments)).hexdigest(),
            "status_hex": status.hex(), "dirty_entries": dirty,
        })
    shallow = Path(os.fsdecode(git(root, "rev-parse", "--path-format=absolute", "--git-path", "shallow").strip()))
    result = {
        "scope": "Refs, worktree registry, index entries and every dirty/nonignored untracked path; clean files use Git status/index identity. Ignored cache bytes are not inventoried or modified.",
        "root_allowed_changes": [".flow/**", "MILESTONES.md", "MIGRATE.md"],
        "head": git(root, "rev-parse", "HEAD").decode().strip(),
        "refs_sha256": hashlib.sha256(git(root, "for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
        "registry_sha256": hashlib.sha256(registry).hexdigest(),
        "shallow_file_sha256": hashlib.sha256(shallow.read_bytes()).hexdigest() if shallow.exists() else None,
        "worktrees": rows,
    }
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(f"Observed {len(rows)} worktrees and {sum(len(row['dirty_entries']) for row in rows)} dirty paths")


if __name__ == "__main__":
    main()
