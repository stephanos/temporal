import sys
sys.dont_write_bytecode = True
import capture_v2 as r

code = r.run('lane-resume', ['/usr/bin/python3', str(r.PACKET/'inspect-lane.py')], r.ROOT, go_probe=False)
raise SystemExit(code if code >= 0 else 128-code)
