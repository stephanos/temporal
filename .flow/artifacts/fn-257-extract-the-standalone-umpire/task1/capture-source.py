"""Capture immutable Git source and observational candidate state for extraction."""

import argparse
import datetime
import hashlib
import io
import json
import subprocess
from pathlib import Path


def git(root, *args):
    return subprocess.check_output(["git", "-C", str(root), *args])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    root = Path(git(Path.cwd(), "rev-parse", "--show-toplevel").decode().strip())
    source = git(root, "rev-parse", args.source + "^{commit}").decode().strip()
    records = []
    for record in git(root, "ls-tree", "-rz", source, "--", "model/umpire/", "LICENSE").split(b"\0"):
        if not record:
            continue
        metadata, path = record.split(b"\t", 1)
        mode, kind, oid = metadata.split()
        if kind != b"blob":
            raise ValueError(f"unsupported object kind {kind!r} at {path!r}")
        source_path = path.decode()
        target_path = source_path.removeprefix("model/umpire/")
        records.append({
            "source": source_path,
            "destination": target_path,
            "mode": mode.decode(),
            "blob": oid.decode(),
        })
    destinations = [record["destination"] for record in records]
    if len(destinations) != len(set(destinations)):
        raise ValueError("duplicate destination in source map")
    oids = sorted({record["blob"] for record in records})
    output = subprocess.check_output(
        ["git", "-C", str(root), "cat-file", "--batch"],
        input="".join(oid + "\n" for oid in oids).encode(),
    )
    blobs = {}
    stream = io.BytesIO(output)
    for expected in oids:
        oid, kind, size = stream.readline().split()
        if oid.decode() != expected or kind != b"blob":
            raise ValueError("unexpected cat-file object")
        payload = stream.read(int(size))
        if len(payload) != int(size) or stream.read(1) != b"\n":
            raise ValueError("truncated cat-file object")
        blobs[expected] = payload
    if stream.read():
        raise ValueError("unexpected cat-file output suffix")
    for record in records:
        payload = blobs[record["blob"]]
        record["size"] = len(payload)
        record["sha256"] = hashlib.sha256(payload).hexdigest()
        if record["mode"] == "120000":
            record["symlink_target_hex"] = payload.hex()
        elif record["mode"] not in {"100644", "100755"}:
            raise ValueError(f"unsupported mode {record['mode']}")
    manifest = {
        "schema": 1,
        "source_commit": source,
        "framework_tree": git(root, "rev-parse", source + ":model/umpire").decode().strip(),
        "archive_tree": git(root, "rev-parse", source + ":model/archived").decode().strip(),
        "framework_files": sum(record["source"].startswith("model/umpire/") for record in records),
        "supplemental_files": ["LICENSE"],
        "records": records,
    }
    worktrees = []
    for block in git(root, "worktree", "list", "--porcelain").decode().strip().split("\n\n"):
        item = dict(line.split(" ", 1) if " " in line else (line, True) for line in block.splitlines())
        checkout = Path(item["worktree"])
        if checkout.is_dir():
            item["status_porcelain_v1_z_hex"] = git(checkout, "status", "--porcelain=v1", "-z").hex()
            item["head_is_source_ancestor"] = subprocess.run(
                ["git", "-C", str(root), "merge-base", "--is-ancestor", item["HEAD"], source],
                check=False,
            ).returncode == 0
        worktrees.append(item)
    args.output.mkdir(parents=True, exist_ok=True)
    (args.output / "source-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    observation = {
        "observed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "source_commit": source,
        "current_head": git(root, "rev-parse", "HEAD").decode().strip(),
        "worktrees": worktrees,
        "note": "Observational only; dirty state is not backed up or transferred. Preserve every unresolved checkout/ref.",
    }
    (args.output / "candidate-observation.json").write_text(json.dumps(observation, indent=2) + "\n")
    print(f"Captured {manifest['framework_files']} framework files, LICENSE and {len(worktrees)} worktrees at {source}")


if __name__ == "__main__":
    main()
