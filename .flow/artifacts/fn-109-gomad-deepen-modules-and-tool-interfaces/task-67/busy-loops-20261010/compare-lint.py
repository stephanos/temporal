import json
import re
import sys
from pathlib import Path


def blocks(path):
    lines = Path(path).read_text().splitlines()
    diagnostics = []
    for index, line in enumerate(lines):
        if re.match(r"^.*\.go:\d+:\d+: ", line):
            block = "\n".join(lines[index:index + 3])
            if len(lines[index:index + 3]) != 3 or "^" not in lines[index + 2]:
                raise ValueError(f"incomplete diagnostic block at {path}:{index + 1}")
            diagnostics.append(block)
    if not diagnostics or len(set(diagnostics)) != len(diagnostics):
        raise ValueError(f"empty or duplicate diagnostics in {path}")
    return diagnostics


before = blocks(sys.argv[1])
after = blocks(sys.argv[2])
removed = sorted(set(before) - set(after))
introduced = sorted(set(after) - set(before))
expected = [
    "tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go:310:5: SA5004:",
    "tools/gomad3/runner/internal/execution/process_test.go:1432:2: SA5002:",
]
matched = len(removed) == 2 and all(any(block.startswith(prefix) for block in removed) for prefix in expected)
print(json.dumps({
    "before": len(before), "after": len(after), "removed": removed,
    "introduced": introduced, "preserved_complete_blocks": len(set(before) & set(after)),
    "exact_admitted_removal": matched and not introduced,
}, indent=2))
sys.exit(0 if matched and not introduced else 1)
