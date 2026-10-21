#!/usr/bin/env python3
"""Exercise public minimization and authenticate the immutable predicate record."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--artifact', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
a = p.parse_args()
a.work.mkdir(mode=0o700, parents=True, exist_ok=False)
receipts = []


def run(name, *flags):
    result = subprocess.run([str(a.binary), 'minimize', '--json', '--artifacts', str(a.work),
                             *map(str, flags), str(a.artifact)], capture_output=True)
    receipts.append(dict(name=name, status=result.returncode, stdout=result.stdout.decode(),
                         stderr=result.stderr.decode()))
    (a.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    return result


first = run('first')
assert first.returncode == 0, (first.returncode, first.stderr)
row = json.loads(first.stdout)
again = run('resume', '--resume')
assert again.returncode == 0 and json.loads(again.stdout) == row, again.stderr
assert run('changed-bounds', '--resume', '--attempt-budget', '65').returncode == 3
assert run('already-exists').returncode == 3
if row['changed']:
    artifact = Path(row['artifact_path'])
    record = (artifact / 'record.json').read_bytes()
    assert hashlib.sha256(record).hexdigest() == row['record_hash']
    details = json.loads((artifact / 'evidence/inspection.json').read_bytes())
    metadata = details['minimization']
    original = json.loads((artifact / 'minimization/original-record.json').read_bytes())
    original['seed'] = 0
    raw = json.dumps(original, sort_keys=True, separators=(',', ':')).encode()
    assert hashlib.sha256(raw).hexdigest() == metadata['predicate']['failure_signature']
    replay = subprocess.run([str(a.binary), 'replay', str(artifact)], capture_output=True)
    assert replay.returncode == 1 and b'reproduced=true' in replay.stdout, replay.stderr
print('public minimize: checkpoint resume, changed bounds, immutable predicate and exact replay checked')
