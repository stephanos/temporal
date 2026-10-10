import sys
sys.dont_write_bytecode = True
import capture_v2 as r

name = sys.argv[1]
assert name in ('validation-domain','analysis-after','analysis-final','final-terminal')
producer = 'terminal.py' if name == 'final-terminal' else 'domain.py' if name == 'validation-domain' else 'analyze.py'
inputs = sorted(r.OUTPUT.glob('*.log'))+sorted(r.OUTPUT.glob('*.json'))
if name == 'validation-domain':
    prior = r.ROOT.parent/'fn-109-74-retention-candidate/.flow/tmp/fn10974-evidence'
    inputs += [prior/'validate.json',prior/'validate-binding.json',prior/'validate.log',r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74']
code = r.run(name,['/usr/bin/python3',str(r.PACKET/producer)],r.ROOT,inputs=inputs,go_probe=False)
raise SystemExit(code if code >= 0 else 128-code)
