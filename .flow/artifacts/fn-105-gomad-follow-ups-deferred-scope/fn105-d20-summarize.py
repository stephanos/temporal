#!/usr/bin/env python3
"""Summarizes the per-seed outputs fn105-d20-run.sh keeps: for every run directory, the Gomad
classification line and each seed's result for TestWorkflowTaskHeartbeatingWithEmptyResult plus
the suite's pass/fail counts.
usage: fn105-d20-summarize.py <D20_OUT>/out [name...]
"""
import glob, gzip, os, re, sys
root, names = sys.argv[1], sys.argv[2:]
for name in names or sorted(os.listdir(root)):
    d = os.path.join(root, name)
    cls = [l.split(" artifact=")[0].strip() for l in open(os.path.join(d, "gomad.log"), errors="replace") if "classification=" in l]
    print("## %s\n   %s" % (name, cls[-1] if cls else "no classification line"))
    rows = []
    for f in glob.glob(os.path.join(d, "seed*-stdout.gz")):
        seed = int(re.search(r"seed(\d+)-", f)[1])
        text = gzip.open(f, "rt", errors="replace").read()
        leaf = re.findall(r"--- (PASS|FAIL): TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult \(([\d.]+s)\)", text)
        sub = re.findall(r"^    --- (PASS|FAIL)", text, re.M)
        count = re.search(r"D20 loop-end elapsed=\S+ hbTimeout=(\d+)", text)
        rows.append((seed, leaf[0] if leaf else ("ABSENT", "-"), sub.count("PASS"), sub.count("FAIL"), count[1] if count else "not instrumented"))
    for seed, leaf, p, fl, c in sorted(rows):
        print("   seed %2d: %s (%s virtual); subtests pass=%d fail=%d; heartbeat rejections counted by the test: %s" % (seed, leaf[0], leaf[1], p, fl, c))
