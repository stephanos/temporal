#!/usr/bin/env python3
"""Require exact replay to compare authenticated standalone network bytes."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--artifact', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
args = p.parse_args()
shutil.copytree(args.artifact, args.work)
manifest_path = args.work / 'manifest.json'
manifest = json.loads(manifest_path.read_bytes())
payload = args.work / 'evidence/network'
original = payload.read_bytes()
changed = original + b'\n'
assert json.loads(changed) == json.loads(original)
payload.write_bytes(changed)
entry = dict(bytes=len(changed), sha256=hashlib.sha256(changed).hexdigest())
manifest['files']['evidence/network'].update(entry)
manifest['record']['streams']['evidence/network'].update(entry)

def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()

record = canonical(manifest['record'])
(args.work / 'record.json').write_bytes(record)
manifest['record_sha256'] = hashlib.sha256(record).hexdigest()
manifest['files']['record.json'].update(bytes=len(record), sha256=manifest['record_sha256'])
manifest_path.write_bytes(canonical(manifest))
commands = []
for name, flags, status, message in [('verify', ['--verify-only'], 0, b'gomad: verified '),
                                     ('replay', [], 1, b'reproduced=false divergence=backend.evidence.sha256')]:
    argv = [str(args.binary), 'replay', *flags, str(args.work)]
    result = subprocess.run(argv, capture_output=True, timeout=60)
    commands.append(dict(name=name, argv=argv, status=result.returncode,
                         stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work.parent / (args.work.name + '-commands.json')).write_text(json.dumps(commands, indent=2) + '\n')
    assert result.returncode == status and message in result.stdout, commands[-1]
print('authenticated network evidence mismatch: exact replay reports first divergence')
