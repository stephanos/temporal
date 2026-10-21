#!/usr/bin/env python3
"""Exercise public retained replay; an unsupported verb must fail this gate."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
args = p.parse_args()
corpus = Path(__file__).resolve().parent
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
receipts = []


def run(name, argv):
    result = subprocess.run(list(map(str, argv)), capture_output=True)
    receipts.append(dict(name=name, argv=list(map(str, argv)), status=result.returncode,
                         stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    return result


result = run('produce', [args.binary, 'fixture-replay-execution', '--root', args.work / 'inputs',
                         '--corpus', corpus])
assert result.returncode == 0, ('fresh replay producer', result.returncode, result.stderr)
bundle = json.loads((args.work / 'inputs/foundation-result.json').read_bytes())
assert bundle['records'], 'missing replay corpus'
for index, row in enumerate(bundle['records']):
    artifact = args.work / 'inputs' / row['artifact']
    verified = run(f'verify-{index}', [args.binary, 'replay', '--verify-only', artifact])
    assert verified.returncode == 0, (index, verified.stderr)
    assert verified.stdout == f'gomad: verified {artifact}\n'.encode()
    result = run(f'replay-{index}', [args.binary, 'replay', artifact])
    assert result.returncode == (0 if row['outcome'] == 'success' else 1), (index, result.stderr)
    assert b'reproduced=true' in result.stdout, (index, result.stdout, result.stderr)
    observed = args.work / f'observed-{index}'
    result = run(f'observed-{index}', [args.binary, 'replay', '--observed', observed, artifact])
    assert result.returncode == (0 if row['outcome'] == 'success' else 1), (index, result.stderr)
    for stream in ('stdout', 'stderr'):
        expected = artifact / 'output' / stream
        assert (observed / stream).read_bytes() == (expected.read_bytes() if expected.exists() else b'')
    assert (observed / 'backend-evidence.bin').is_file()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def reseal(path, manifest):
    raw = canonical(manifest['record'])
    (path / 'record.json').write_bytes(raw)
    manifest['record_sha256'] = hashlib.sha256(raw).hexdigest()
    manifest['files']['record.json'].update(bytes=len(raw), sha256=manifest['record_sha256'])
    (path / 'manifest.json').write_bytes(canonical(manifest))


source = args.work / 'inputs' / bundle['records'][0]['artifact']
for name, mutate in [
    ('changed-fuel', lambda record: record['limits'].update(fuel=1)),
    ('changed-arguments', lambda record: record.update(arguments=['different'])),
    ('changed-environment', lambda record: record.update(environment={'PWD': '/different'})),
]:
    path = args.work / name
    shutil.copytree(source, path)
    manifest = json.loads((path / 'manifest.json').read_bytes())
    mutate(manifest['record'])
    reseal(path, manifest)
    result = run(name, [args.binary, 'replay', '--verify-only', path])
    assert result.returncode == 2 and not result.stdout, (name, result.returncode, result.stdout, result.stderr)
print('public replay: retained fresh exact and observed executions match')
