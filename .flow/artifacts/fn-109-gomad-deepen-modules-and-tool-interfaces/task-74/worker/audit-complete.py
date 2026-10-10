import sys
sys.dont_write_bytecode = True
import run as r

historical = r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
research = r.ROOT/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retention-helper-next-slice.md'
prior = r.ROOT.parent/'fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence'
complexity = r.ROOT.parent/'fn-109-23-lint-complexity-candidate/.flow/tmp/fn10923-complexity'
inputs = [historical,research,prior/'after-aggregate-lint.log',prior/'runner-lint.log',complexity/'final-fast-lint.log',complexity/'final-gomad-lint.log',r.PRIMARY/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/deadline-observation-admission-20261010.md']
inputs += sorted(r.OUTPUT.glob('*.log'))+sorted(r.OUTPUT.glob('*.json'))+sorted(r.OUTPUT.glob('*.trace'))
assert all(path.is_file() for path in inputs)
assert r.run('evidence-audit-complete',['/usr/bin/python3',str(r.PACKET/'check-progress-complete.py')],r.ROOT,inputs=inputs) == 0
