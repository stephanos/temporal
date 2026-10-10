import hashlib
import json
import pathlib
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-61')
OUT = pathlib.Path(__file__).resolve().parent
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
if hashlib.sha256((ROOT / 'tools/gomad3/toolchain/build_test.go').read_bytes()).hexdigest() != 'ba64e85358a6993e669a5a1925ab0cee8ece625fb43df987b9bdeca437b2cd15':
    raise SystemExit('Wrong frozen candidate')
inputs = json.loads((OUT / 'control-inputs-v2.json').read_text())
runner = OUT / 'run_final_gate.py'
gates = [('final-builds', 'tools/gomad3', "go test -tags test_dep -count=1 -timeout=90s -run '^TestBuild' -v ./toolchain", [], 0)]
for name in ['context-controls', 'muted-observer', 'waited-false', 'bypassed-wait-sequential']:
    overlay = OUT / (name + '-v2-overlay.json')
    test = 'TestBuildLockWaitContextControls' if name == 'context-controls' else 'TestBuildSerializesConcurrentSameKey'
    command = f"go test -tags test_dep -count=1 -timeout=30s -overlay={overlay} -run '^{test}$' -v ./toolchain"
    gates.append((name, 'tools/gomad3', command, list(inputs['controls'][name]), 0 if name == 'context-controls' else 1))
for name, directory, command, auxiliary, expected in gates:
    result = subprocess.run([sys.executable, str(runner), name, directory, command, json.dumps(auxiliary)], cwd=ROOT)
    print(name + ' terminal exit=' + str(result.returncode), flush=True)
    if result.returncode != expected:
        raise SystemExit(result.returncode or 1)
    receipt = json.loads((OUT / (name + '.json')).read_text())
    if not receipt['source_unchanged'] or receipt['timed_out'] or not receipt['terminal'] or receipt['auxiliary_before'] != receipt['auxiliary_after']:
        raise SystemExit('Unstable or inconclusive gate ' + name)
