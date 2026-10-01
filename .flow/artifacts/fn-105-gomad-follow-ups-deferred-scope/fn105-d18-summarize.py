#!/usr/bin/env python3
"""Tabulate per-seed outcomes from the timelines run.sh leaves under out/<name>/."""
import glob, os, re, sys
for d in sys.argv[1:]:
    print("## %s" % os.path.basename(d.rstrip("/")))
    print("seed result virtual_s first_timer_task worker_commands_tasks activity_timeout_tasks run_expiration activity_scheduled")
    rows = []
    for tl in glob.glob(os.path.join(d, "seed*-timeline.txt")):
        seed = int(re.search(r"seed(\d+)-", tl).group(1))
        t = open(tl).read()
        res = open(tl.replace("-timeline", "-results")).read()
        m = re.search(r"--- (PASS|FAIL): \S+TestDispatchCancelOnWorkflowTimeout \(([\d.]+)s\)", res)
        first = re.search(r"task=(WorkflowRunTimeoutTask|ActivityTimeoutTask)", t)
        run = re.search(r"queue-task-type=WorkflowRunTimeout timestamp=\S+T(\S+)", t)
        act = re.search(r"queue-task-type=TransferActivityTask timestamp=\S+T(\S+)", t)
        rows.append((seed, m.group(1) if m else "?", m.group(2) if m else "?",
                     first.group(1) if first else "-",
                     len(re.findall(r"Assigning new task key \| .*queue-task-type=WorkerCommands", t)),
                     len(re.findall(r"Assigning new task key \| .*queue-task-type=ActivityTimeout", t)),
                     run.group(1) if run else "-", act.group(1) if act else "-"))
    for r in sorted(rows):
        print(" ".join(str(x) for x in r))
    n = len(rows); f = sum(1 for r in rows if r[1] == "FAIL")
    print("# seeds=%d pass=%d fail=%d\n" % (n, n - f, f))
