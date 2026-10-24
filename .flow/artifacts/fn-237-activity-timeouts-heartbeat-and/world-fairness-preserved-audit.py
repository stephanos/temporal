"""Read-only preserved-artifact audit; writes task-owned evidence only."""
import datetime
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

root = Path(__file__).resolve().parents[3]
if Path.cwd().resolve() != root:
    raise SystemExit("Run from the temporal repository root")
suffix = sys.argv[1] if len(sys.argv) > 1 else "recheck"
if not re.fullmatch(r"[A-Za-z0-9_-]+", suffix):
    raise SystemExit("Invalid evidence suffix")
manifest_path = Path(__file__).with_name("world-fairness-preserved-manifest.json")
manifest = json.loads(manifest_path.read_text())
head = subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip()
proc = subprocess.Popen(["git", "cat-file", "--batch"], stdin=subprocess.PIPE, stdout=subprocess.PIPE)

def blob(ref, path):
    proc.stdin.write(f"{ref}:{path}\n".encode())
    proc.stdin.flush()
    header = proc.stdout.readline().decode().strip()
    if header.endswith(" missing"):
        return None
    _, kind, size = header.split()
    if kind != "blob":
        raise ValueError(header)
    data = proc.stdout.read(int(size))
    if proc.stdout.read(1) != b"\n":
        raise ValueError("Invalid git blob boundary")
    return data

def digest(data):
    return None if data is None else {"sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data)}

rows = []
for path, expected in manifest["files"].items():
    file = root / path
    working = digest(file.read_bytes()) if file.is_file() else None
    base = digest(blob(manifest["base_commit"], path))
    committed = digest(blob(head, path))
    rows.append({"path": path, "expected": expected, "working": working,
                 "manifest_commit": base, "head_commit": committed,
                 "working_matches": working == expected,
                 "manifest_commit_matches": base == expected,
                 "head_commit_matches": committed == expected})
compat_path = root / "model/temporal/feature/activity/compatibility/manifest.json"
compat = json.loads(compat_path.read_text())
authority = root / ".flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256"
authoritative = {line.split(None, 1)[1]: line.split(None, 1)[0] for line in authority.read_text().splitlines()}
compat_rows = []
for entry in compat["files"]:
    path = entry["retained_path"]
    expected = entry["sha256"]
    values = {"working": (root / path).read_bytes(), "head_retained": blob(head, path),
              "pinned_original": blob(compat["commit"], entry["original_path"])}
    checks = {key + "_matches": value is not None and hashlib.sha256(value).hexdigest() == expected
              for key, value in values.items()}
    compat_rows.append({**entry, **checks, "authority_matches": expected == authoritative[entry["original_path"]],
                        "in_preserved_manifest": path in manifest["files"]})
proc.stdin.close()
if proc.wait() != 0:
    raise SystemExit("git cat-file failed")
failed = [row["path"] for row in rows if not all(row[key] for key in ("working_matches", "manifest_commit_matches", "head_commit_matches"))]
compat_failed = [row["retained_path"] for row in compat_rows if not all(row[key] for key in ("working_matches", "head_retained_matches", "pinned_original_matches", "authority_matches"))]
summary = {"manifest_entries": len(rows), "manifest_all_match": not failed,
           "manifest_failures": failed, "compat_entries": len(compat_rows),
           "compat_all_match": not compat_failed, "compat_failures": compat_failed}
result = {"observed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
          "manifest_base_commit": manifest["base_commit"], "head_commit": head,
          "summary": summary, "files": rows, "frozen_compatibility": compat_rows}
output = root / f".flow/tmp/fn2376-preserved-{suffix}.json"
output.write_text(json.dumps(result, indent=2) + "\n")
print(json.dumps(summary, indent=2))
print(output)
raise SystemExit(bool(failed or compat_failed))
