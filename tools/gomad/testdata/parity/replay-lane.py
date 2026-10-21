#!/usr/bin/env python3
"""Mandatory fresh semantic, authenticated replay and golden lane."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
p = argparse.ArgumentParser()
for name in ('binary', 'baseline', 'source', 'go-compiler', 'work',
             'stress-binary', 'determinism-binary', 'release-binary', 'preparation-root',
             'cluster-baseline', 'cluster-source'):
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
    return result.stdout


run('fresh-parity', [sys.executable, corpus / 'replay-reference-lane.py', '--baseline', args.baseline,
    '--source', args.source, '--go-compiler', args.go_compiler, '--rust-binary', args.binary,
    '--output', args.work / 'fresh-parity', '--preparation-root', args.preparation_root])
for name, driver in [('consumer', 'replay-check.py'), ('opener', 'replay-fault-check.py'),
                     ('cluster', 'replay-cluster-check.py'), ('cluster-failures', 'replay-cluster-failure-check.py'),
                     ('golden', 'replay-golden-check.py')]:
    run(name, [sys.executable, corpus / driver, '--binary', args.binary, '--work', args.work / name])
run('runtime-golden', [sys.executable, corpus / 'replay-golden-check.py', '--binary', args.binary,
    '--work', args.work / 'runtime-golden', '--input', args.work / 'fresh-parity/replay-input.json'])
storage = args.work / 'fresh-parity/go/storage-origin-input.json'
for name, driver in [('storage', 'replay-storage-check.py'), ('storage-prefix', 'replay-storage-prefix-check.py')]:
    run(name, [sys.executable, corpus / driver, '--binary', args.binary, '--reference', storage,
        '--work', args.work / name])
registered = json.loads(run('registered-fixture', [sys.executable, corpus / 'replay-inspection-fixture.py',
    '--binary', args.binary, '--baseline', args.baseline, '--source', args.source,
    '--compiler', args.go_compiler, '--work', args.work / 'registered']))
for name, binary, role in [('ordinary', args.binary, None), ('stress', args.stress_binary, 'stress-jitter'),
                           ('determinism', args.determinism_binary, 'determinism-fixture')]:
    artifact = registered['artifact']
    if role:
        produced = json.loads(run(name + '-registered-produce', [binary, 'fixture-replay-input',
            '--root', args.work / (name + '-registered'), '--input', registered['input'], '--role', role]))
        artifact = str(args.work / (name + '-registered') / produced['records'][0]['artifact'])
    run(name + '-inspection', [sys.executable, corpus / 'replay-inspection-check.py', '--binary', binary,
        '--artifact', artifact, '--go-binary', registered['go_binary'], '--go-reference', registered['go_reference'],
        '--work', args.work / (name + '-inspection'), '--duration-edges', *(['--role', role] if role else [])])
run('roles', [sys.executable, corpus / 'replay-role-check.py', '--ordinary-binary', args.binary,
    '--stress-binary', args.stress_binary, '--determinism-binary', args.determinism_binary,
    '--release-binary', args.release_binary, '--output', args.work / 'roles'])
cluster_arguments = ['--baseline', args.cluster_baseline, '--source', args.cluster_source, '--compiler', args.go_compiler]
run('public-cluster-reference', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--work', args.work / 'public-cluster'])
public_reference = args.work / 'public-cluster/cluster-reference.json'
run('public-cluster-import', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--binary', args.binary, '--from-reference', public_reference, '--work', args.work / 'public-cluster-import'])
run('cluster-api-reference', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--from-reference', public_reference, '--api-registered', registered['go_reference'],
    '--work', args.work / 'cluster-api'])
run('cluster-api-import', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--binary', args.binary, '--from-reference', args.work / 'cluster-api/api/cluster-reference.json',
    '--work', args.work / 'cluster-api-import'])
run('cluster-alternate-reference', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--from-reference', public_reference, '--api-registered', registered['go_reference'], '--api-concurrent',
    '--work', args.work / 'cluster-alternate'])
run('cluster-alternate-import', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--binary', args.binary, '--from-reference', args.work / 'cluster-alternate/api/cluster-reference.json',
    '--work', args.work / 'cluster-alternate-import'])
successful = json.loads(run('successful-registered-fixture', [sys.executable, corpus / 'replay-inspection-fixture.py',
    '--binary', args.binary, *cluster_arguments, '--success', '--work', args.work / 'successful-registered']))
run('cluster-success-reference', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--from-reference', public_reference, '--api-registered', successful['go_reference'], '--api-success',
    '--work', args.work / 'cluster-success'])
run('cluster-success-import', [sys.executable, corpus / 'replay-cluster-reference.py', *cluster_arguments,
    '--binary', args.binary, '--from-reference', args.work / 'cluster-success/api/cluster-reference.json',
    '--work', args.work / 'cluster-success-import'])
print('replay lane: fresh Go/Rust semantics, retained replay, opener faults, cluster and exact goldens passed')
