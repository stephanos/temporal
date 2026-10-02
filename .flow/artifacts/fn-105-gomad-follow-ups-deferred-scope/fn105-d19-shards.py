#!/usr/bin/env python3
"""Per history shard: transfer-queue load before the first measured activity poll of
Test_Activity_Basic. Usage: fn105-d19-shards.py [--suite NAME] <log>...  (files may be .gz)

For every shard it prints the immediate transfer tasks created (all namespaces of the shared
cluster, "Assigning new task key" with a Transfer* type) and the transfer tasks whose execution
started ("Process task as active.", transfer-queue-processor) before that poll, and for this
test's workflows which had their 15 TransferActivityTasks started by then."""
import gzip, re, sys
from collections import Counter, defaultdict
args = sys.argv[1:]; suite = "TestFairnessSuite"
if args[0] == "--suite": suite = args[1]; args = args[2:]
ns = suite + "-Test_Activity_Basic-"
nsid = None; created = Counter(); started = Counter(); wfshard = {}; wfstarted = Counter(); seenK = False
done = False
for p in args:
    op = gzip.open if p.endswith(".gz") else open
    for raw in op(p, "rt", errors="replace"):
        if nsid is None and "Register namespace succeeded" in raw and ns in raw:
            nsid = re.search(r'"wf-namespace-id": "([^"]+)"', raw).group(1)
        sh = re.search(r'"shard-id": (\d+)', raw)
        if "Assigning new task key" in raw and sh:
            t = re.search(r'"queue-task-type": "(Transfer\w+)"', raw)
            if t:
                created[int(sh.group(1))] += 1
                m = re.search(r'"wf-id": "wf(\d+)"', raw)
                if m and nsid and nsid in raw and t.group(1) == "TransferActivityTask":
                    wfshard[int(m.group(1))] = int(sh.group(1)); seenK = True
        elif "Process task as active" in raw and "transfer-queue-processor" in raw and sh:
            started[int(sh.group(1))] += 1
            m = nsid and re.search(r'"value": "%s/wf(\d+)/' % re.escape(nsid), raw)
            if m: wfstarted[int(m.group(1))] += 1
        elif seenK and "Frontend method invoked" in raw and '"PollActivityTaskQueue"' in raw and ns in raw:
            done = True; break
    if done: break
print("before the first measured activity poll:")
for s in sorted(set(created) | set(started)):
    wfs = sorted(w for w, x in wfshard.items() if x == s)
    print("  shard %d: transfer tasks created=%3d started=%3d not-started=%3d | this test's workflows: %s"
          % (s, created[s], started[s], created[s] - started[s],
             " ".join("wf%d(%d/15)" % (w, wfstarted[w]) for w in wfs)))
