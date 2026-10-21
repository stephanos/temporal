import hashlib
import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent

def sections(data):
    offset = 8
    result = {}
    while offset < len(data):
        tag = data[offset]
        offset += 1
        size = 0
        shift = 0
        while True:
            value = data[offset]
            offset += 1
            size |= (value & 127) << shift
            if value < 128:
                break
            shift += 7
        result[tag] = data[offset:offset + size]
        offset += size
    return result

names = ["stock-linux", "stock-macos", "stock-macos-replay", "stock-macos-replay-paths",
         "temporal-linux", "temporal-macos", "temporal-macos-replay-paths"]
artifacts = {}
for name in names:
    file = root / (name + ".wasm")
    data = file.read_bytes()
    artifacts[name] = {"path": str(file), "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
comparisons = []
for left, right in [("stock-linux", name) for name in names[1:4]] + [
    ("temporal-linux", "temporal-macos"), ("temporal-linux", "temporal-macos-replay-paths")]:
    a, b = [root / (name + ".wasm") for name in (left, right)]
    result = subprocess.run(["cmp", str(a), str(b)], capture_output=True, text=True, check=False)
    av, bv = a.read_bytes(), b.read_bytes()
    differences = [i for i, (x, y) in enumerate(zip(av, bv)) if x != y]
    sa, sb = sections(av), sections(bv)
    comparisons.append({"left": left, "right": right, "command": result.args,
                        "exit_code": result.returncode, "stdout": result.stdout, "stderr": result.stderr,
                        "first_difference_zero_based": differences[0] if differences else None,
                        "equal_sections": [key for key in sa if key in sb and sa[key] == sb[key]]})
    print(left, right, "cmp", result.returncode)
assert [c["exit_code"] for c in comparisons] == [1, 1, 0, 1, 0]
assert artifacts["stock-linux"]["sha256"] == "5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1"
assert artifacts["temporal-linux"]["sha256"] == "10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06"
assert 10 in comparisons[1]["equal_sections"]
(root / "results.json").write_text(json.dumps({"artifacts": artifacts, "comparisons": comparisons}, indent=2) + "\n")
