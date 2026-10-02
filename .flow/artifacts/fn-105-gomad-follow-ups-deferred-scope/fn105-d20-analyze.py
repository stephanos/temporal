#!/usr/bin/env python3
"""Reduces `go test -v` logs of the D20-instrumented TestWorkflowTaskHeartbeatingWithEmptyResult
(fn105-d20-variant-i-instrumentation.diff.txt) to one block per test run: the heartbeat
iterations, and for every chain of heartbeat workflow tasks the time from the chain's first
WorkflowTaskScheduled event to each event that closed a workflow task of the chain.

usage: fn105-d20-analyze.py [--history] <log or log.gz>...
"""
import gzip, re, sys

TIMEOUT = 5.0  # WorkflowTaskHeartbeatTimeout override in tests/testcore/dynamic_config_overrides.go
UNITS = {"ns": 1e-9, "µs": 1e-6, "us": 1e-6, "ms": 1e-3, "s": 1.0, "m": 60.0, "h": 3600.0}

def dur(text):
    text = text.lstrip("+")
    sign = -1.0 if text.startswith("-") else 1.0
    total = 0.0
    for num, unit in re.findall(r"([0-9.]+)(ns|µs|us|ms|s|m|h)", text):
        total += float(num) * UNITS[unit]
    return sign * total

def clock(text):
    h, m, s = text.split(":")
    return int(h) * 3600 + int(m) * 60 + float(s)

ITER = re.compile(r"D20 iter=(\d+) sent=(\S+) \((\S+)\) returned=(\S+) err=(.*)")
POLL = re.compile(r"D20 iter=(\d+) recovery-poll returned=(\S+) attempt=(\d+)")
HIST = re.compile(r"D20 history \S+ +(\d+) (\w+) +(\S+)(.*)")
END = re.compile(r"D20 loop-end elapsed=(\S+) hbTimeout=(\d+)")

def runs(path):
    opener = gzip.open if path.endswith(".gz") else open
    cur = None
    out = []
    with opener(path, "rt", errors="replace") as f:
        for line in f:
            if "D20 loop-start" in line:
                cur = {"iters": [], "polls": [], "hist": [], "end": None, "result": "?"}
                out.append(cur)
            elif cur is None:
                continue
            elif m := ITER.search(line):
                cur["iters"].append((int(m[1]), dur(m[3]), dur(m[4]), m[5].strip()))
            elif m := POLL.search(line):
                cur["polls"].append((int(m[1]), dur(m[2])))
            elif m := HIST.search(line):
                cur["hist"].append((int(m[1]), m[2], clock(m[3]), m[4].strip()))
            elif m := END.search(line):
                cur["end"] = (dur(m[1]), int(m[2]))
            elif m := re.search(r"--- (PASS|FAIL): \S*TestWorkflowTaskHeartbeatingWithEmptyResult", line):
                cur["result"] = m[1]
                cur = None
    return out

def report(path, show_history):
    for n, run in enumerate(runs(path), 1):
        end = run["end"] or (float("nan"), -1)
        print("== %s run %d: %s, heartbeat timeouts=%d, loop elapsed=%.6fs" % (path.split("/")[-1], n, run["result"], end[1], end[0]))
        polls = dict(run["polls"])
        prev_returned = None
        for i, sent, returned, err in run["iters"]:
            # Under the forward tick `returned` comes from time.Since, which reads the timer clock
            # while `sent` is a ticked time.Now difference; only native and strict rows compare.
            gap = "" if prev_returned is None else " gap-since-previous-return=%.6fs" % (sent - prev_returned)
            print("  heartbeat %2d sent=+%.6fs rpc=%.6fs%s -> %s" % (i, sent, returned - sent, gap, "REJECTED (NotFound)" if "heartbeat timeout" in err else "accepted"))
            prev_returned = polls.get(i, returned)
            if i in polls:
                print("               recovery poll returned=+%.6fs" % polls[i])
        hist = run["hist"]
        chain = None
        started = {}
        for idx, (eid, etype, t, extra) in enumerate(hist):
            if show_history:
                print("  event %2d %-24s %.9f %s" % (eid, etype, t, extra))
            prev = hist[idx - 1][1] if idx else ""
            if etype == "WorkflowTaskScheduled" and prev in ("WorkflowExecutionStarted", "WorkflowTaskTimedOut"):
                chain = (eid, t)
                continue
            if etype == "WorkflowTaskStarted":
                started[eid] = t
            if chain and etype in ("WorkflowTaskCompleted", "WorkflowTaskTimedOut"):
                elapsed = t - chain[1]
                kind = "completed"
                if etype == "WorkflowTaskTimedOut":
                    since_started = t - hist[idx - 1][2]
                    kind = "TIMED OUT by start-to-close timer" if since_started > 2.9 else "TIMED OUT by heartbeat check"
                print("  chain from event %2d: event %2d %-34s at chain+%.9fs (deadline %+.9fs)" % (chain[0], eid, kind, elapsed, elapsed - TIMEOUT))

args = sys.argv[1:]
show = "--history" in args
for p in [a for a in args if a != "--history"]:
    report(p, show)
