#!/usr/bin/env python3
"""Reduce a `go test -v` log of TestFairnessAutoEnableSuite/Test_Activity_Basic to the
triggerAutoEnable phase: matching queue lifecycle, user-data changes, and the trigger workflow's
tasks and polls. Usage: fn105-d19-trigger.py [--suite NAME] [--max N] <log>...  (files may be .gz)"""
import gzip, re, sys
args = sys.argv[1:]; suite = "TestFairnessAutoEnableSuite"; mx = 80
while args and args[0].startswith("--"):
    if args[0] == "--suite": suite = args[1]
    if args[0] == "--max": mx = int(args[1])
    args = args[2:]
ns = suite + "-Test_Activity_Basic-"
KEYS = ["operation", "backlog", "cause", "lifecycle", "queue-task-type", "wf-task-queue-type",
        "user-data-version", "user-data-update-source", "wf-id", "error", "long-poll"]
DROP = re.compile(r"ping check|ContextMetadataInterceptor|time skipping|Quota changed|Start workflow execution request|"
                  r"routing key extraction|Received |Workflow task updated|returning empty user data|DescribeNamespace|"
                  r"AddSearchAttributes|ListSearchAttributes|UpdateNamespace|Update namespace")
LINE = re.compile(r"(\d{4}-\d\d-\d\dT[\d:.]+)(?:Z|[+-]\d{4})?\t(\w+)\t([^\t]*)\t(\{.*)")
nsid = None; n = 0
for p in args:
    op = gzip.open if p.endswith(".gz") else open
    for raw in op(p, "rt", errors="replace"):
        if "Error Trace" in raw or "Error:  " in raw or "failed to poll" in raw or re.search(r"--- (PASS|FAIL): %s/Test_Activity_Basic" % suite, raw) or "unfairness:" in raw:
            print("TEST  " + raw.strip()[:230]); continue
        m = LINE.search(raw)
        if not m: continue
        ts, lvl, msg, rest = m.groups()
        if nsid is None and "Register namespace succeeded" in raw and ns in raw:
            nsid = re.search(r'"wf-namespace-id": "([^"]+)"', raw).group(1)
        if nsid is None or (ns not in raw and nsid not in raw) or DROP.search(raw): continue
        if re.search(r'"wf-id": "wf\d+"|/wf\d+/', raw): break   # the measured workload starts here
        f = []
        mm = re.search(r'"value": "(\w+)\{', rest)
        if mm: f.append("task=" + mm.group(1))
        for k in KEYS:
            mm = re.search(r'"%s": ("[^"]*"|[^,}]*)' % re.escape(k), rest)
            if mm: f.append("%s=%s" % (k, mm.group(1).strip('"')[:160]))
        n += 1
        if n <= mx: print("%s %-5s %s | %s" % (ts[11:], lvl, msg, " ".join(f)))
