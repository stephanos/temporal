#!/usr/bin/env python3
"""Require every selected source test root to pass; reject absent or skipped roots."""
from pathlib import Path
import re
import sys

package, selection, logfile, repeats = sys.argv[1:]
pattern = re.compile(selection)
names = sorted({name for source in Path(package).glob("*_test.go")
                for name in re.findall(r"^func (Test\w+)\(", source.read_text(), re.M)
                if pattern.search(name)})
assert names, f"empty source selection: {package} {selection}"
log = Path(logfile).read_text()
for name in names:
    if name in {"TestCompressedWorldOwnerMeasuredCardinality", "TestWorldOwnerCodecMeasuredCardinality"}:
        assert len(re.findall(r"^--- SKIP: " + re.escape(name) + r" ", log, re.M)) == 1, f"opt-in probe unexpectedly executed: {name}"
        continue
    passes = re.findall(r"^--- PASS: " + re.escape(name) + r" ", log, re.M)
    assert len(passes) == int(repeats), f"{name}: expected {repeats} PASS roots, observed {len(passes)}"
    assert not re.search(r"^--- SKIP: " + re.escape(name) + r" ", log, re.M), f"skipped: {name}"
print(f"required roots: {len(names)}, repetitions: {repeats}")
