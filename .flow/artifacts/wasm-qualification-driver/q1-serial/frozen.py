#!/usr/bin/env python3
"""Guard the frozen q1 candidate: HEAD, clean status, every manifest input, symlinks, staged fixtures."""
import hashlib, json, os, subprocess, sys
from pathlib import Path
repo, root = map(Path, sys.argv[1:])
digest = lambda p: hashlib.file_digest(open(p, "rb"), "sha256").hexdigest()
pre = json.loads((root / "preflight.json").read_text())
manifest = root / "source.sha256"
assert digest(manifest) == pre["source_manifest_sha256"] == "503ccf73ae1b9c651ad2544175b854464febf296409d2da6192f0ecad21a2641", "manifest changed"
env = dict(os.environ, GIT_DISCOVERY_ACROSS_FILESYSTEM="1")
head = subprocess.check_output(["git", "-C", str(repo), "rev-parse", "HEAD"], text=True, env=env).strip()
assert head == pre["head"] == "e50c6476fba40d3b31ddb20d1030abf08807412c", "HEAD changed"
status = subprocess.check_output(["git", "-C", str(repo), "status", "--porcelain", "--untracked-files=all"], env=env)
assert status == b"", "worktree not clean:\n" + status.decode()[:4000]
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
