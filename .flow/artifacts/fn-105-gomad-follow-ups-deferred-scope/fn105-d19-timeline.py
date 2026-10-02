#!/usr/bin/env python3
"""Reduce a `go test -v` log of Test_Activity_Basic (FairnessSuite) to a transfer/poll/dispatch
timeline in log order, and recompute the test's unfairness metric.

Usage: fn105-d19-timeline.py [--suite NAME] [--events] <log> [<log>...]   (files may be .gz)

Server log lines used (all debug level, already emitted by the test cluster):
  T  "Process task as active." component=transfer-queue-processor, value=<ns-id>/wfN/<run>
     -> history starts executing one TransferActivityTask of workflow N (it then calls matching)
  K  "Assigning new task key" queue-task-type=TransferActivityTask -> the transfer task is created
  P  "Frontend method invoked." operation=PollActivityTaskQueue    -> an activity poll reached frontend
  C  "routing key extraction" .../RespondActivityTaskCompleted wf-id=wfN -> a dispatched task is completed
Test log lines used:
  D  "priority_fairness_test.go:NNN: activity fkey F wfidx N" -> dispatch order as the test records it
Everything in one Gomad strict-tick busy stretch carries the same virtual timestamp, so the order of
log lines is the timeline; the script prints the timestamp range next to it.
"""
import gzip, re, sys
from collections import Counter

args = sys.argv[1:]
suite = "TestFairnessSuite"
events = False
while args and args[0].startswith("--"):
    if args[0] == "--suite":
        suite = args[1]; args = args[2:]
    elif args[0] == "--events":
        events = True; args = args[1:]
nsname = suite + "-Test_Activity_Basic-"

def lines(p):
    op = gzip.open if p.endswith(".gz") else open
    with op(p, "rt", errors="replace") as f:
        yield from f

TS = re.compile(r"(\d{4}-\d\d-\d\dT[\d:.]+)(?:Z|[+-]\d{4})?\t")
nsid = None
ev = []          # (kind, wf, ts) in server-log order
disp = []        # (fkey, wfidx) in test-log order
unf_logged = None
in_test = False
for p in args:
    for raw in lines(p):
        m = re.search(r"=== (?:NAME|CONT|RUN)\s+(\S+)", raw)
        if m:
            in_test = m.group(1) == suite + "/Test_Activity_Basic"
        m = re.search(r"activity fkey (\d+) wfidx (\d+)", raw)
        if m and in_test:
            disp.append((int(m.group(1)), int(m.group(2)))); continue
        m = re.search(r"unfairness: ([\d.]+)", raw)
        if m and in_test:
            unf_logged = float(m.group(1)); continue
        t = TS.search(raw)
        ts = t.group(1)[11:] if t else "?"
        if nsid is None and "Register namespace succeeded" in raw and nsname in raw:
            nsid = re.search(r'"wf-namespace-id": "([^"]+)"', raw).group(1); continue
        if nsid is None:
            continue
        if "Assigning new task key" in raw and '"TransferActivityTask"' in raw and nsid in raw:
            m = re.search(r'"wf-id": "wf(\d+)"', raw)   # skips the auto-enable "trigger" workflow
            if m:
                ev.append(("K", int(m.group(1)), ts))
        elif "Process task as active" in raw and "transfer-queue-processor" in raw:
            m = re.search(r'"value": "%s/wf(\d+)/' % re.escape(nsid), raw)
            if m:
                ev.append(("T", int(m.group(1)), ts))
        elif "Frontend method invoked" in raw and '"PollActivityTaskQueue"' in raw and nsname in raw:
            ev.append(("P", -1, ts))
        elif "routing key extraction" in raw and "RespondActivityTaskCompleted" in raw:
            m = re.search(r'"wf-id": "wf(\d+)"', raw)
            if m:
                ev.append(("C", int(m.group(1)), ts))

def unfairness(vs):
    firsts = {}
    for i, v in enumerate(vs):
        firsts.setdefault(v, i)
    return sum(firsts.values()) / float(len(firsts) ** 2), firsts

kinds = Counter(k for k, _, _ in ev)
print("namespace-id=%s server events: created(K)=%d transfer-started(T)=%d polls(P)=%d completed(C)=%d; test dispatch lines(D)=%d"
      % (nsid, kinds["K"], kinds["T"], kinds["P"], kinds["C"], len(disp)))
if ev:
    print("timestamps: first=%s last=%s" % (ev[0][2], ev[-1][2]))
# server-side C order must equal the test-side D order (same workflow sequence)
cs = [w for k, w, _ in ev if k == "C"]
print("server completion order == test dispatch order (by workflow): %s" % (cs == [w for _, w in disp]))

# transfer progress at the first poll and at each dispatch
created = started = 0
wf_started = Counter()
first_poll = None
at_c = []   # (transfer-started count, distinct workflows started, created) when the k-th completion is logged
last_created_before_started = None
for i, (k, w, ts) in enumerate(ev):
    if k == "K":
        created += 1
    elif k == "T":
        started += 1; wf_started[w] += 1
    elif k == "P" and first_poll is None and created > 0:   # polls before the workload belong to triggerAutoEnable
        first_poll = (i, created, started, len(wf_started), dict(wf_started), ts)
    elif k == "C":
        at_c.append((started, len(wf_started), created))
if first_poll:
    i, c, s_, nw, per, ts = first_poll
    print("at the first activity poll (ts %s): transfer tasks created=%d/225, transfer started=%d/225 from %d/15 workflows %s"
          % (ts, c, s_, nw, sorted(per.items())))
for k in (1, 10, 20, 40, 66, 100, 150, 200, 225):
    if k <= len(at_c):
        s_, nw, c = at_c[k - 1]
        print("  when dispatch #%3d completes: transfer started=%3d/225 from %2d/15 workflows" % (k, s_, nw))
full = next((n + 1 for n, (s_, _, _) in enumerate(at_c) if s_ >= 225), None)
print("first dispatch completed after all 225 transfers had started: #%s" % full)

if disp and first_poll and first_poll[2] < 225:
    # fairness over what was eligible: the keys carried by the workflows whose transfers had started at the
    # first poll, against the keys of the first dispatches made before any further transfer started
    w0 = {w for w, n in first_poll[4].items()}
    k0 = {f for f, w in disp if w in w0}
    stall = next((n for n, (s_, _, _) in enumerate(at_c) if s_ > first_poll[2]), len(at_c))
    n = len(k0)
    print("partial backlog: %d keys present in the %d workflows transferred at the first poll; %d dispatches before further transfers started; "
          "distinct keys in the first %d dispatches: %d" % (n, len(w0), stall, n, len({f for f, _ in disp[:n]})))
if disp:
    keys = [f for f, _ in disp]
    u, firsts = unfairness(keys)
    print("unfairness recomputed=%.4g logged=%s (threshold: < 1.0); keys seen=%d" % (u, unf_logged, len(firsts)))
    print("first dispatch index per fairness key: %s" % sorted(firsts.items()))
    print("ideal (every key within the first %d dispatches) = %.4g" % (len(firsts), sum(range(len(firsts))) / float(len(firsts) ** 2)))
    for n in (10, 20, 40, 66):
        print("  first %2d dispatches: %2d distinct workflows, %2d distinct keys" % (n, len({w for _, w in disp[:n]}), len({f for f, _ in disp[:n]})))
    print("tasks per key: %s" % sorted(Counter(keys).items()))
    # per key: the first workflow (in dispatch order) that carries the key at all, and whether the key's
    # first dispatch waited for that workflow's transfer
    print("dispatch order (fkey@wf): " + " ".join("%d@%d" % d for d in disp[:80]) + " ...")
if events:
    out = []
    n = 0
    for k, w, ts in ev:
        if k == "C":
            n += 1
        if k in "TPC":
            out.append(k if k == "P" else "%s%d" % (k, w))
    print("event order (T<wf>=transfer started, P=poll, C<wf>=completed): " + " ".join(out))
