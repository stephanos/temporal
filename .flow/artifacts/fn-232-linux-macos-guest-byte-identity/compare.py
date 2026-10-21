import hashlib
import itertools
import json
from pathlib import Path
import subprocess
import sys

repository = Path(sys.argv[1]).resolve()
pin = "8d429e97409b1537ecb347c359129bffafa63f78"
artifacts = {}
sources = {}
comparisons = []
for system, lane in itertools.product(("linux", "macos"), ("a", "b")):
    root = repository / ".flow/tmp" / f"fn232-certification-{system}"
    source = root / f"source-{lane}"
    revision = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip()
    status = subprocess.check_output(["git", "-C", str(source), "status", "--porcelain", "--untracked-files=all"], text=True)
    assert revision == pin, (source, revision)
    assert not status, (source, status)
    sources[f"{system}-{lane}"] = {"path": str(source), "revision": revision, "status": status}
    for guest in ("stock", "temporal"):
        path = root / f"{guest}-{lane}.wasm"
        data = path.read_bytes()
        artifacts[f"{system}-{guest}-{lane}"] = {
            "path": str(path), "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest(),
        }
for guest in ("stock", "temporal"):
    names = [name for name in artifacts if f"-{guest}-" in name]
    for left, right in itertools.combinations(names, 2):
        a, b = [Path(artifacts[name]["path"]) for name in (left, right)]
        result = subprocess.run(["cmp", str(a), str(b)], capture_output=True, text=True, check=False)
        assert result.returncode in (0, 1), result
        first_difference = None
        if result.returncode:
            av, bv = a.read_bytes(), b.read_bytes()
            first_difference = next((i for i, (x, y) in enumerate(zip(av, bv)) if x != y), min(len(av), len(bv)))
        comparisons.append({
            "left": left, "right": right, "command": result.args,
            "exit_code": result.returncode, "stdout": result.stdout, "stderr": result.stderr,
            "first_difference_zero_based": first_difference,
        })
print(json.dumps({"pin": pin, "sources": sources, "artifacts": artifacts, "comparisons": comparisons}, indent=2))
sys.exit(1 if any(comparison["exit_code"] for comparison in comparisons) else 0)
