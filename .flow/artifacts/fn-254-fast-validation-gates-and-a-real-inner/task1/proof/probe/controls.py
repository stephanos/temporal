#!/usr/bin/env python3
"""Synthetic-only controls: no production executable, Cargo, Go, or Java."""
import json
import os
from pathlib import Path
import subprocess
import sys

from observer import write

root = Path(__file__).resolve().parent.parent / 'synthetic'
root.mkdir(parents=True, exist_ok=True)
probe = Path(__file__).resolve().parent
for mode in ['mise', 'java', 'cargo-runner', 'serial-runner']:
    destination = probe / mode
    if not destination.exists():
        destination.symlink_to('observer.py')
    destination.chmod(0o755)
fake = root / 'fake-harness'
fake.write_text('''#!/usr/bin/env python3
import sys,time
print('running 3 tests',flush=True)
print('test slow::case ... ',end='',flush=True)
time.sleep(0.09)
print('ok',flush=True)
print('test ignored ... ignored, intentional',flush=True)
print('test failed ... ',end='',flush=True)
time.sleep(0.06)
print('FAILED',flush=True)
print('test result: FAILED. 1 passed; 1 failed; 1 ignored',flush=True)
raise SystemExit(101)
''')
fake.chmod(0o755)
env = dict(os.environ, FN254_OUT=str(root), FN254_REAL_JAVA=str(fake), FN254_REAL_MISE='/usr/bin/true')
result = subprocess.run([str(probe/'serial-runner'), str(fake)], env=env)
assert result.returncode == 101, result
records = sorted((root/'binaries').glob('*.json'), key=lambda path: path.stat().st_mtime_ns)
record = json.loads(records[-1].read_text())
assert [test['name'] for test in record['tests']] == ['slow::case', 'ignored', 'failed'], record
assert [test['status'] for test in record['tests']] == ['ok','ignored','FAILED'], record
assert 0.07 < record['tests'][0]['seconds'] < 2, record
assert 0.04 < record['tests'][2]['seconds'] < 2, record
assert record['exit_code'] == 101
version = subprocess.run([str(probe/'mise'),'which','java'], env=env, capture_output=True, text=True)
assert version.returncode == 0 and version.stdout.strip() == str(probe/'java')
assert subprocess.call([str(probe/'mise'), 'anything'], env=env) == 0
(root/'Toy.tla').write_text('---- MODULE Toy ----\n====\n')
(root/'Run.cfg').write_text('SPECIFICATION Spec\n')
java = subprocess.run([str(probe/'java'),'tlc2.TLC','-tool','-config',str(root/'Run.cfg'),'Toy'], cwd=root, env=env, capture_output=True)
assert java.returncode == 101
events = list((root/'java').glob('*/event.json'))
event = json.loads(max(events,key=lambda path:path.stat().st_mtime_ns).read_text())
assert event['kind'] == 'tlc-model' and event['exit_code'] == 101 and event['seconds'] > 0.1
assert {entry['name'] for entry in event['files']} == {'Toy.tla','Run.cfg'}
write(root/'control-result.json', dict(exit_code=0, assertions=['partial-byte start timing','serial identity/status','nonzero exit preservation','mise exact interception/delegation','TLC input snapshots and elapsed/nonzero status']))
print('synthetic controls passed')
