"""Verify the immutable activity originals against the pinned SHA256 manifest."""
import hashlib
import json
from pathlib import Path

folder = Path(__file__).resolve().parent
manifest = json.loads((folder / "manifest.json").read_text())
assert manifest["commit"] == "af0ebafc571d33651fdf2c008d4415ad72467a0d"
assert len(manifest["files"]) == 15
authoritative = dict(line.split(None, 1)[::-1] for line in (folder.parents[4] / ".flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256").read_text().splitlines())
for entry in manifest["files"]:
    relative = Path(entry["original_path"]).relative_to("model/temporal/feature/activity/expected")
    assert entry["sha256"] == authoritative[entry["original_path"]]
    actual = hashlib.sha256((folder / "original" / relative).read_bytes()).hexdigest()
    assert actual == entry["sha256"], entry["original_path"]
print("15 original activity artifacts verified")
