import sys
sys.dont_write_bytecode = True
import capture_v2 as r

name = sys.argv[1]
assert name in ('build-evidence', 'handover-seal')
producer = 'build-evidence.py' if name == 'build-evidence' else 'seal-packet.py'
inputs = sorted(r.OUTPUT.glob('*.log')) + sorted(r.OUTPUT.glob('*.json'))
if name == 'handover-seal':
    inputs += sorted(r.PACKET.glob('*.md')) + sorted(r.PACKET.glob('*.json'))
code = r.run(name, ['/usr/bin/python3', str(r.PACKET/producer)], r.ROOT, inputs=inputs, go_probe=False)
raise SystemExit(code if code >= 0 else 128-code)
