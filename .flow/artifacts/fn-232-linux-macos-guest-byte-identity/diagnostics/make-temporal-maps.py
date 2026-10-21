import json
from pathlib import Path
from unit_key import unit_key

root = Path(__file__).resolve().parent
metadata = {}
paths = {}
for name in ("rustc-linux.jsonl", "rustc-linux-temporal.jsonl"):
    phase = {}
    for line in (root / name).read_text().splitlines():
        row = json.loads(line)
        result = unit_key(row["argv"], row["cwd"])
        if result is None:
            continue
        key, index = result
        value = row["argv"][index]
        assert key not in phase or phase[key] == value, key
        phase[key] = value
        if out := row.get("env", {}).get("OUT_DIR"):
            paths[key] = out.replace(str(root / "target-linux"), "/target")
    metadata.update(phase)
for line in (root / "rustc-macos-temporal.jsonl").read_text().splitlines():
    row = json.loads(line)
    result = unit_key(row["argv"], row["cwd"])
    if result:
        assert result[0] in metadata, result[0]
(root / "linux-temporal-metadata-map.json").write_text(json.dumps(metadata, indent=2) + "\n")
(root / "linux-temporal-out-dir-map.json").write_text(json.dumps(paths, indent=2) + "\n")
print("Matched target keys:", len(metadata))
