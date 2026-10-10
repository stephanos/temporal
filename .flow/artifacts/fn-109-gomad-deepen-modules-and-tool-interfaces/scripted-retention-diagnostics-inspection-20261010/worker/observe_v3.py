import sys
sys.dont_write_bytecode = True
import capture_v2 as r

name = sys.argv[1]
assert name in ('analysis-ordinary', 'analysis-complete', 'final-terminal')
producer = 'terminal.py' if name == 'final-terminal' else 'analyze.py'
inputs = sorted(r.OUTPUT.glob('*.log')) + sorted(r.OUTPUT.glob('*.json'))
code = r.run(name, ['/usr/bin/python3', str(r.PACKET/producer)], r.ROOT, inputs=inputs, go_probe=False)
raise SystemExit(code if code >= 0 else 128-code)
