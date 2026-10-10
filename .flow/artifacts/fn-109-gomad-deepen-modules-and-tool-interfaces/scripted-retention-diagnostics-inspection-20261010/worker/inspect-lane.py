import json
import os
import subprocess
from pathlib import Path

snapshot = subprocess.check_output(['/usr/bin/ps','-eo','pid=,ppid=,pgid=,etime=,args='],text=True)
active = []
for line in snapshot.splitlines():
    fields = line.split(None,4)
    if len(fields) != 5 or int(fields[0]) == os.getpid():
        continue
    argv = fields[4].split()
    executable = Path(argv[0]).name
    gate = executable in ('go','compile','link','errortype') or executable.startswith('golangci-lint') or executable.endswith('.test')
    gate |= executable == 'make' and any(any(term in arg for term in ('lint','validate','gomad','test','build')) for arg in argv[1:])
    gate |= executable.startswith('python') and any(Path(arg).name in ('run.py','gates.py') for arg in argv[1:])
    if gate:
        active.append(line)
print(json.dumps({'observer':'/usr/bin/ps -eo pid=,ppid=,pgid=,etime=,args=','active_gate_commands':active,'gate_lane_empty':not active},sort_keys=True,indent=2))
raise SystemExit(1 if active else 0)
