#!/usr/bin/env python3
"""Run foundation fixtures twice before checking or regenerating exact goldens."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument('--binary', type=Path, required=True)
parser.add_argument('--work', type=Path, required=True)
args = parser.parse_args()
corpus = Path(__file__).resolve().parent
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
prompt = subprocess.run([str(args.binary), 'fixture-watchdog-cancel', '--corpus', str(corpus)], capture_output=True, timeout=5)
(args.work / 'watchdog-cancellation.stdout').write_bytes(prompt.stdout)
(args.work / 'watchdog-cancellation.stderr').write_bytes(prompt.stderr)
assert prompt.returncode == 0, prompt.stderr

def command(name, argv):
    result = subprocess.run(argv, capture_output=True, timeout=120)
    (args.work / (name + '.stdout')).write_bytes(result.stdout)
    (args.work / (name + '.stderr')).write_bytes(result.stderr)
    (args.work / (name + '.status')).write_text(str(result.returncode) + '\n')
    assert result.returncode == 0, (name, result.returncode, result.stderr.decode())
    return result.stdout

def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()

views = []
records = []
executions = []
execution_ledgers = []
for repetition in range(2):
    suffix = str(repetition)
    matrix = json.loads(command('store-' + suffix, ['python3', str(corpus / 'foundation-store-check.py'), str(args.binary), str(args.work / ('store-' + suffix))]))
    assert set(matrix) == {'count', 'passed'} and matrix['count'] == len(matrix['passed'])
    views.append(canonical(matrix))
    command('record-' + suffix, ['env', '-u', 'BASH_ENV', '-u', 'SANDBOX_START_DIR', '/bin/bash', str(corpus / 'foundation-record-check.sh'), str(args.binary), str(args.work / ('record-' + suffix))])
    records.append((args.work / ('record-' + suffix) / 'one.json').read_bytes())
    command('bundle-' + suffix, [str(args.binary), 'fixture-bundle', '--root', str(args.work / ('bundle-' + suffix))])
    command('authenticate-' + suffix, ['python3', str(corpus.parents[1] / 'scripts/behavioral-parity'), '--corpus', str(corpus), '--foundation-root', str(args.work / ('bundle-' + suffix)), '--foundation-result', str(args.work / ('bundle-' + suffix) / 'foundation-result.json')])
    execution_root = args.work / ('executions-' + suffix)
    report = json.loads(command('executions-' + suffix, [str(args.binary), 'fixture-execution-bundle', '--root', str(execution_root), '--corpus', str(corpus)]))
    command('execution-authenticate-' + suffix, ['python3', str(corpus.parents[1] / 'scripts/behavioral-parity'), '--corpus', str(corpus), '--foundation-root', str(execution_root), '--foundation-result', str(execution_root / 'foundation-result.json')])
    executions.append([(execution_root / item['record_file']).read_bytes() for item in report['records']])
    execution_ledgers.append((execution_root / report['ledger_file']).read_bytes())
assert views[0] == views[1], 'fault-matrix changed across repetitions'
assert records[0] == records[1], 'canonical record bytes changed across repetitions'
assert executions[0] == executions[1], 'fresh engine/store execution records changed across repetitions'
assert execution_ledgers[0] == execution_ledgers[1], 'authenticated raw execution ledger changed across repetitions'
goldens = corpus / 'goldens-foundation'
expected = {'store-matrix.json': views[0] + b'\n', 'execution-ledger.jsonl': execution_ledgers[0]}
index = json.loads((corpus / 'foundation-execution.json').read_bytes())
names = [(case['name'], seed) for case in index['cases'] for seed in index['seeds']]
assert len(names) == len(executions[0])
hashes = []
for (name, seed), raw in zip(names, executions[0]):
    record = json.loads(raw)
    expected[f'records/{name}-{seed}.json'] = raw
    hashes.append({'case': name, 'seed': seed, 'record_sha256': hashlib.sha256(raw).hexdigest(), 'failure_signature': hashlib.sha256(canonical(dict(record, seed=0))).hexdigest(), 'execution_identity': record['execution_identity']})
expected['record-sha256.json'] = canonical(hashes) + b'\n'
for filename, value in expected.items():
    path = goldens / filename
    if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(value)
    else:
        assert path.is_file() and path.read_bytes() == value, str(path) + ': golden mismatch'
print('foundation raw integrity and repeated goldens match')
