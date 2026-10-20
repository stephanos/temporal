#!/usr/bin/env python3
"""Source-only comparison: never invokes a gate or imports candidate packages."""
import ast
import json
from pathlib import Path
import re
import sys

reference, current = map(Path, sys.argv[1:3])
def calls(path):
    text = path.read_text().replace("\\\n", " ")
    stages = {}
    for match in re.finditer(r'^\s*(stage|named_test|named_native_test) +("?)([^\s"]+)\2 +(.*)$', text, re.M):
        kind, _, label, command = match.groups()
        if label == "$label":
            continue
        expanded = [(label, command)]
        if label == "runtime-$index":
            assert "for index in $(seq 0 31); do" in text
            expanded = [(f"runtime-{i}", command) for i in range(32)]
        elif label == "sqlite-selection-$index":
            assert "for index in 0 1 2 3; do" in text
            expanded = [(f"sqlite-selection-{i}", command) for i in range(4)]
        elif label == "workload-$workload-$profile-$cell":
            entries = re.findall(r"'([\w-]+) (\w+) (Test\w+)'", text)
            assert len(entries) == 5
            assert "for cell in success canary; do" in text
            expanded = [(f"workload-{w}-{p}-{cell}", command.replace('"$gate"', test))
                        for w, p, test in entries for cell in ("success", "canary")]
        else:
            assert "$" not in label, label
        for name, call in expanded:
            assert name not in stages, f"duplicate label: {name}"
            stages[name] = (kind, call)
    return stages

old, new = calls(reference), calls(current)
assert len(old) == 89, len(old)
assert not old.keys() - new.keys(), sorted(old.keys() - new.keys())
for path in current.parent.glob("*.py"):
    ast.parse(path.read_text(), filename=str(path))
repo = Path("/Users/stephan/Workspace/skunkworks/gomad/temporal")
module = repo / "tools/gomad"
missing = []
named = []
for label, (kind, command) in new.items():
    if kind not in ("named_test", "named_native_test"):
        continue
    name, timeout, package = command.split()[:3]
    if package.startswith("go.temporal.io/server/tools/gomad/"):
        location = module / package.removeprefix("go.temporal.io/server/tools/gomad/")
    elif package.startswith("./internal/runner"):
        location = module / package.removeprefix("./")
    else:
        location = module / "internal/backends/wasm" / package.removeprefix("./")
    definition = re.compile(r"func\s+" + re.escape(name) + r"\s*\(")
    if not any(definition.search(file.read_text()) for file in location.glob("*_test.go")):
        missing.append({"label": label, "test": name, "package": package})
    named.append(name)
assert not missing, f"missing named tests: {missing}"
print(json.dumps({"reference_labels": len(old), "preserved": len(old.keys() & new.keys()),
                  "added": sorted(new.keys() - old.keys()), "total": len(new),
                  "named_test_invocations_resolved": len(named), "python_ast": "PASS",
                  "status": "SOURCE_ONLY_NO_GATES_EXECUTED"}, indent=2))
