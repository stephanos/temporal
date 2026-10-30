"""Inject an extra sibling parent without changing export snapshots or metadata."""

import argparse
import json
from pathlib import Path
import subprocess
import sys


def git(repo, *arguments, payload=None):
    return subprocess.check_output(["git", "-C", str(repo), *arguments], input=payload)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-repo", type=Path, required=True)
    parser.add_argument("--scratch", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    source = args.source_repo.resolve()
    task = source / ".flow/artifacts/fn-257-extract-the-standalone-umpire/task2"
    task1 = task.parent / "task1"
    args.scratch = args.scratch.resolve()
    args.scratch.mkdir()
    clone = args.scratch / "export"
    subprocess.run(["git", "clone", "--quiet", "--branch", "umpire",
                    str(task / "export-history.bundle"), str(clone)], check=True)
    history = json.loads((task1 / "history-map.json").read_text())
    rows = (task / "export-attempt1/commit-map").read_text().splitlines()
    mapping = dict(row.split() for row in rows[1:])
    canonical = mapping[history["boundaries"]["canonical_archive_move"]["commit"]]
    sibling = mapping[history["boundaries"]["sibling_archive_move"]["commit"]]
    if "0" * 40 in {canonical, sibling}:
        raise ValueError("archive siblings must be retained for this control")
    related = subprocess.run(["git", "-C", str(clone), "merge-base", "--is-ancestor",
                              canonical, sibling], check=False).returncode
    if related != 1:
        raise ValueError("adding the sibling parent would not be an independent extra edge")
    original_head = git(clone, "rev-parse", "HEAD").decode().strip()
    rewritten = {}
    for oid in git(clone, "rev-list", "--reverse", "--topo-order", "HEAD").decode().splitlines():
        original = git(clone, "cat-file", "commit", oid)
        header, message = original.split(b"\n\n", 1)
        lines = []
        for line in header.split(b"\n"):
            if line.startswith(b"parent "):
                old = line.removeprefix(b"parent ").decode()
                line = b"parent " + rewritten[old].encode()
            lines.append(line)
        if oid == canonical:
            position = next(index for index, line in enumerate(lines)
                            if not line.startswith((b"tree ", b"parent ")))
            lines.insert(position, b"parent " + rewritten.get(sibling, sibling).encode())
        payload = b"\n".join(lines) + b"\n\n" + message
        rewritten[oid] = oid if payload == original else git(
            clone, "hash-object", "-t", "commit", "-w", "--stdin", payload=payload
        ).decode().strip()
    checkpoint = rewritten[original_head]
    git(clone, "update-ref", "refs/heads/umpire", checkpoint, original_head)
    if git(clone, "rev-parse", "HEAD^{tree}") != git(clone, "rev-parse", original_head + "^{tree}"):
        raise ValueError("control changed the active snapshot")
    changed_map = args.scratch / "commit-map"
    changed_map.write_text(rows[0] + "\n" + "".join(
        old + " " + (rewritten[new] if new != "0" * 40 else new) + "\n"
        for old, new in mapping.items()
    ))
    legacy = args.scratch / "verifier-before-parent-fix.py"
    legacy.write_bytes(git(source, "show", "35efd6422e:" + str((task / "verify-export.py").relative_to(source))))
    common = ["--source-repo", str(source), "--export-repo", str(clone),
              "--checkpoint", checkpoint, "--commit-map", str(changed_map),
              "--manifest", str(task1 / "source-manifest.json"),
              "--history-map", str(task1 / "history-map.json"),
              "--root-head-baseline", git(source, "rev-parse", "HEAD").decode().strip()]
    results = {}
    for name, script in (("before", legacy), ("after", task / "verify-export.py")):
        report = args.scratch / (name + ".json")
        completed = subprocess.run([sys.executable, str(script), *common, "--output", str(report)],
                                   check=False, capture_output=True, text=True)
        value = json.loads(report.read_text())
        results[name] = {"exit": completed.returncode, "status": value["status"],
                         "failures": value["failures"]}
    if results["before"]["exit"] != 0 or results["before"]["status"] != "pass":
        raise ValueError("legacy verifier did not reproduce the accepted bad-ancestry gap")
    if results["after"]["exit"] != 1 or not any(
        item["category"] == "rewritten-parent-order" for item in results["after"]["failures"]["samples"]
    ):
        raise ValueError("new verifier did not reject the injected ancestry edge")
    args.output.write_text(json.dumps({
        "control": "extra sibling archive parent; all descendant hashes and commit-map rows recomputed",
        "original_checkpoint": original_head, "mutated_checkpoint": checkpoint,
        "canonical_archive_export": canonical, "unexpected_sibling_parent": sibling,
        "active_snapshot_unchanged": True, "results": results,
        "scope": "Only an independently cloned synthetic export graph was mutated; no Model or original repository changes.",
    }, indent=2) + "\n")
    print("Legacy verifier accepted the injected edge; updated verifier rejected it")


if __name__ == "__main__":
    main()
