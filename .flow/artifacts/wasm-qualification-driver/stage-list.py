#!/usr/bin/env python3
"""Print the expanded, sorted stage-label set of the original serial driver.

Labels come from every `stage <label>` and `named_test <label>` invocation in
run-qualification.sh. The three loop-generated label families are expanded
from the loop bounds the original states literally; any other variable label
fails, so a driver change cannot silently escape the equivalence check.
"""
import re
import sys

text = open(sys.argv[1]).read()
labels = []
for match in re.finditer(r'^\s*(?:stage|named_test) +("?)([^\s"]+)\1', text, re.M):
    label = match.group(2)
    if label == "$label":
        continue  # the named_test helper body
    if "$" not in label:
        labels.append(label)
    elif label == "runtime-$index":
        assert "for index in $(seq 0 31); do" in text
        labels += [f"runtime-{i}" for i in range(32)]
    elif label == "workload-$workload-$profile-$cell":
        entries = re.findall(r"'(\w[\w-]*) (\w+) Test\w+'", text)
        assert len(entries) == 5, entries
        assert "for cell in success canary; do" in text
        labels += [f"workload-{w}-{p}-{c}" for w, p in entries for c in ("success", "canary")]
    elif label == "sqlite-selection-$index":
        assert "for index in 0 1 2 3; do" in text
        labels += [f"sqlite-selection-{i}" for i in range(4)]
    else:
        sys.exit(f"unexpanded variable stage label {label}")
assert len(labels) == len(set(labels)), "duplicate stage label"
print("\n".join(sorted(labels)))
