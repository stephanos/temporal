#!/usr/bin/env python3
"""Mandatory fresh inspect parity plus authenticated consumer and golden gates."""
import argparse
import json
from pathlib import Path
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--baseline', type=Path, required=True)
p.add_argument('--source', type=Path, required=True)
p.add_argument('--go-compiler', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
args = p.parse_args()
corpus = Path(__file__).resolve().parent
declaration = json.loads((corpus / 'inspect-corpus.json').read_bytes())
assert set(declaration) == {'schema', 'verb', 'fresh_execution_corpus', 'runtime_choice_target',
                            'runtime_overflow_target', 'consumer_details', 'golden', 'projection'}
assert declaration['schema'] == 'gomad.inspect-corpus/v1' and declaration['verb'] == 'inspect'
assert set(declaration['projection']) == {'path', 'record', 'files', 'choices', 'world',
                                         'simulation', 'minimization', 'stdout', 'stderr'}
for name in ('fresh_execution_corpus', 'runtime_choice_target', 'runtime_overflow_target', 'consumer_details'):
    assert (corpus / declaration[name]).is_file(), declaration[name]
for path in (args.binary, args.baseline / 'inputs.sha256', args.go_compiler, args.source / 'tools/gomad/go.mod'):
    assert path.is_file(), f'required inspect input missing: {path}'
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)


def run(name, command):
    result = subprocess.run(list(map(str, command)), capture_output=True)
    (args.work / (name + '.json')).write_text(json.dumps(dict(argv=list(map(str, command)),
        status=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode()), indent=2) + '\n')
    assert result.returncode == 0, (name, result.returncode, result.stderr)


run('fresh-parity', ['python3', corpus / 'inspect-reference-check.py', '--baseline', args.baseline,
    '--source', args.source, '--go-compiler', args.go_compiler, '--rust-binary', args.binary,
    '--output', args.work / 'fresh-parity'])
run('consumer', ['python3', corpus / 'inspect-check.py', '--binary', args.binary,
    '--work', args.work / 'consumer', '--details-input', corpus / declaration['consumer_details']])
run('store', ['python3', corpus / 'foundation-store-check.py', args.binary, args.work / 'store', '--inspect'])
run('golden', ['python3', corpus / 'inspect-golden-check.py', '--binary', args.binary,
    '--work', args.work / 'golden', '--details-input', corpus / declaration['consumer_details']])
print('inspect lane: fresh Go/Rust semantics, consumer faults, shared opener and exact goldens passed')
