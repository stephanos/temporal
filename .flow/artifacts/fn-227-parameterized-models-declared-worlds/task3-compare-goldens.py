"""Compare this task's exact baseline snapshot/expected/diagnostic inventory."""
import hashlib
import json
from pathlib import Path
import subprocess

BASE = "376fda53b344e5503a69e628195b87c71837a4c5"
ROOT = Path(__file__).resolve().parents[3]
OWNER = "model/temporal/feature/activity/expected/"
RELATED_EXPANSION = "model/temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization.snap"
paths = subprocess.check_output(
    ["git", "ls-tree", "-r", "--name-only", BASE, "model"], cwd=ROOT
).decode().splitlines()
records = []
for path in paths:
    if not ("/expected/" in path or ".snap" in Path(path).name or path.endswith(".stderr")):
        continue
    before = subprocess.check_output(["git", "show", BASE + ":" + path], cwd=ROOT)
    current = ROOT / path
    after = current.read_bytes() if current.is_file() else b""
    records.append({
        "path": path,
        "before_sha256": hashlib.sha256(before).hexdigest(),
        "after_sha256": hashlib.sha256(after).hexdigest(),
        "changed": before != after,
    })
changed = [record for record in records if record["changed"]]
assert all(record["path"].startswith(OWNER) or record["path"] == RELATED_EXPANSION for record in changed), changed
folder = Path(__file__).resolve().parent
(folder / "task3-golden-inventory.json").write_text(json.dumps({
    "base": BASE,
    "selection": "tracked at BASE under model/; path contains /expected/, filename contains .snap, or path ends .stderr",
    "records": records,
}, indent=2) + "\n")
(folder / "task3-all-golden-comparison.json").write_text(json.dumps({
    "base": BASE,
    "inventory": "task3-golden-inventory.json",
    "script": "task3-compare-goldens.py",
    "total": len(records),
    "unchanged": len(records) - len(changed),
    "changed": changed,
    "all_unrelated_exact": True,
    "related_expansion": RELATED_EXPANSION,
}, indent=2) + "\n")
print(f"{len(records)} baseline files; {len(records)-len(changed)} exact; {len(changed)} related changes")
