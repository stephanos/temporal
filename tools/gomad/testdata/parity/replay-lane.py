#!/usr/bin/env python3
"""Mandatory fresh semantic, authenticated replay and golden lane."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
p = argparse.ArgumentParser()
for name in ('binary', 'baseline', 'source', 'go-compiler', 'work'):
    p.add_argument('--' + name, type=Path, required=True)
args = p.parse_args()
corpus = Path(__file__).resolve().parent
declaration = json.loads((corpus / 'replay-corpus.json').read_bytes())
assert set(declaration) == {'schema', 'verb', 'targets', 'seeds', 'profiles', 'goldens', 'projection'}
assert declaration['schema'] == 'gomad.replay-corpus/v1' and declaration['verb'] == 'replay'
assert set(declaration['projection']) == {'record', 'manifest', 'ledger', 'status', 'replay', 'verify', 'observed'}
for target in declaration['targets']:
    assert (corpus / target).is_file(), target
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)


def run(name, command):
    result = subprocess.run(list(map(str, command)), capture_output=True, timeout=1200)
    (args.work / (name + '.json')).write_text(json.dumps(dict(argv=list(map(str, command)),
        status=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode()), indent=2) + '\n')
    assert result.returncode == 0, (name, result.returncode, result.stderr)


run('fresh-parity', [sys.executable, corpus / 'replay-reference-lane.py', '--baseline', args.baseline,
    '--source', args.source, '--go-compiler', args.go_compiler, '--rust-binary', args.binary,
    '--output', args.work / 'fresh-parity'])
for name, driver in [('consumer', 'replay-check.py'), ('opener', 'replay-fault-check.py'),
                     ('cluster', 'replay-cluster-check.py'), ('golden', 'replay-golden-check.py')]:
    run(name, [sys.executable, corpus / driver, '--binary', args.binary, '--work', args.work / name])
run('runtime-golden', [sys.executable, corpus / 'replay-golden-check.py', '--binary', args.binary,
    '--work', args.work / 'runtime-golden', '--input', args.work / 'fresh-parity/replay-input.json'])
print('replay lane: fresh Go/Rust semantics, retained replay, opener faults, cluster and exact goldens passed')
