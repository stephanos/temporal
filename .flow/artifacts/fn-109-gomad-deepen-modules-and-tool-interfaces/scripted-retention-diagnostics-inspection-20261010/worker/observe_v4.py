import sys
sys.dont_write_bytecode = True
import capture_v2 as r

name = sys.argv[1]
assert name in ('validation-preflight', 'current-validation-domain')
producer = 'inspect-lane-v2.py' if name == 'validation-preflight' else 'current-domain.py'
inputs = []
if name == 'current-validation-domain':
    inputs = sorted(r.OUTPUT.glob('*.log')) + sorted(r.OUTPUT.glob('*.json'))
    inputs += [r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74']
code = r.run(name, ['/usr/bin/python3', str(r.PACKET/producer)], r.ROOT, inputs=inputs, go_probe=False)
raise SystemExit(code if code >= 0 else 128-code)
