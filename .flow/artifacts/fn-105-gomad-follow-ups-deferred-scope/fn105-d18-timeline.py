#!/usr/bin/env python3
"""Reduce a `go test -v` log of TestDispatchCancelOnWorkflowTimeout to the server
log lines that trace workflow timeout -> cancel command -> control-queue poll.
Usage: timeline.py <stdout-file> [<stderr-file>]"""
import re, sys

KEYS = ["component", "operation", "queue-task-id", "queue-task-type", "timestamp",
        "wf-scheduled-event-id", "wf-poll-context-timeout", "control_queue", "error"]
WANT = re.compile(r"DispatchCancelOnWorkflowTimeout|PollNexusTaskQueue|too short|"
                  r"No worker polling|WorkerCommands|Skipping worker cancel")
DROP = re.compile(r"ping check|Frontend method invoked|ContextMetadataInterceptor")
LINE = re.compile(r"(\d{4}-\d\d-\d\dT[\d:.]+(?:Z|[+-]\d{4}))\t(\w+)\t([^\t]*)\t(\{.*)")
VALUE = re.compile(r'"value": "(\w+)\{WorkflowKey: \S+?, (.*?)\}"')

def fields(rest):
    out = []
    m = VALUE.search(rest)
    if m:
        out.append("task=%s{%s}" % (m.group(1), m.group(2)))
    for k in KEYS:
        mm = re.search(r'"%s": ("[^"]*"|[^,}]*)' % re.escape(k), rest)
        if mm:
            v = mm.group(1).strip('"')
            if k == "control_queue":
                v = v.split("/")[-1]
            out.append("%s=%s" % (k, v))
    return " ".join(out)

for path in sys.argv[1:]:
    for raw in open(path, errors="replace"):
        if not WANT.search(raw) or DROP.search(raw):
            continue
        m = LINE.search(raw)
        if not m:
            continue
        ts, lvl, msg, rest = m.groups()
        if "DispatchCancelOnWorkflowTimeout" not in raw and "PollNexusTaskQueue" not in raw \
                and "too short" not in raw:
            continue
        print("%s %-5s %s | %s" % (ts[11:], lvl, msg, fields(rest)))
