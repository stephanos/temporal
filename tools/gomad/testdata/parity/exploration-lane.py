#!/usr/bin/env python3
"""Require controller, public exploration, replay, divergence and model-owner evidence."""
import argparse
import ast
import hashlib
import json
import os
from pathlib import Path
import runpy
import subprocess
import sys
from types import SimpleNamespace
from unittest.mock import patch

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
MODULE = CORPUS.parents[1]
HARNESS = MODULE / 'scripts/behavioral-parity'
INPUTS = ('binary', 'baseline', 'source', 'go-compiler', 'input', 'preparation-root',
          'divergence-binary', 'conformance-helper', 'go-base', 'work')


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), allow_nan=False).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def selftest():
    for path in (Path(__file__), HARNESS):
        ast.parse(path.read_text(), filename=str(path))
    parity = runpy.run_path(str(HARNESS))
    flags = [item for name in INPUTS if name != 'work'
             for item in ('--exploration-' + name, '/test/' + name)]
    for verb in ('explore', 'resume'):
        calls = []
        with patch.object(sys, 'argv', [str(HARNESS), '--corpus', str(CORPUS), '--verb', verb,
                '--output-root', '/test/output', '--campaign-binary', '/test/campaign', *flags]), \
                patch.object(subprocess, 'run', side_effect=lambda command: (
                    calls.append(command) or SimpleNamespace(returncode=0))):
            parity['main']()
        assert len(calls) == 1 and Path(calls[0][1]).name == 'exploration-lane.py', calls
        assert calls[0][calls[0].index('--verb') + 1] == verb, calls
        assert calls[0][calls[0].index('--conformance-helper') + 1] == '/test/conformance-helper', calls
    with patch.object(sys, 'argv', [str(HARNESS), '--corpus', str(CORPUS), '--verb', 'explore',
            '--exploration-binary', '/test/binary']), patch.object(subprocess, 'run') as child:
        try:
            parity['main']()
        except parity['Refusal']:
            pass
        else:
            raise AssertionError('incomplete exploration inputs were accepted')
        child.assert_not_called()
    print('exploration lane syntax, explore/resume routing precedence and missing-input refusal passed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in INPUTS:
        parser.add_argument('--' + name, type=Path)
    parser.add_argument('--verb', choices=('explore', 'resume'), default='explore')
    parser.add_argument('--selftest', action='store_true')
    args = parser.parse_args()
    if args.selftest:
        selftest()
        return
    for name in INPUTS:
        key = name.replace('-', '_')
        value = getattr(args, key)
        if value is None:
            parser.error('--' + name + ' is required')
        setattr(args, key, value.resolve())
    for path in (args.binary, args.divergence_binary, args.conformance_helper, args.go_compiler,
                 args.input, args.go_base, args.baseline / 'inputs.sha256', args.baseline / 'bin/gomad'):
        assert path.is_file(), ('missing mandatory exploration input', path)
    assert (args.source / 'tools/gomad/go.mod').is_file(), args.source
    args.work.mkdir(parents=True, exist_ok=False)
    args.preparation_root.mkdir(parents=True, exist_ok=True)
    environment = {key: value for key, value in os.environ.items()
                   if not key.startswith('GOMAD_') and key != 'GOROOT'}
    environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', GOMAXPROCS='2')
    receipts = []

    def run(name, command, expected=0, cwd=None, extra=None):
        result = subprocess.run(list(map(str, command)), cwd=cwd,
            env=dict(environment, **(extra or {})), capture_output=True, timeout=2400)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), cwd=str(cwd) if cwd else None,
            environment_overrides=extra or {}, status=result.returncode, expected=expected,
            stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(encode(receipts))
        assert result.returncode == expected, (name, result.returncode,
            result.stdout.decode(errors='replace'), result.stderr.decode(errors='replace'))
        return result.stdout

    for kind, count in (('choice', 9), ('simulation', 12)):
        declaration = json.loads((CORPUS / (kind + '-controller-corpus.json')).read_bytes())
        assert len(declaration['cases']) == count, (kind, count)
        run(kind + '-controller', [sys.executable, CORPUS / (kind + '-controller-check.py'),
            '--binary', args.binary, '--go', args.go_compiler, '--output', args.work / (kind + '-controller')])

    common = ['--binary', args.binary, '--go-compiler', args.go_compiler, '--input', args.input]
    run('choice-public-go', [sys.executable, CORPUS / 'choice-live-check.py', *common,
        '--work', args.work / 'choice', '--go-baseline', args.baseline, '--go-source', args.source])
    simulation = []
    raw_records = []
    full_artifacts = []
    for index in range(2):
        work = args.work / ('simulation-' + str(index))
        run('simulation-live-' + str(index), [sys.executable, CORPUS / 'simulation-live-check.py',
            *common, '--work', work, '--preparation-root', args.preparation_root / 'simulation'])
        observed = json.loads((work / 'observations.json').read_bytes())
        assert set(observed['forced_dimensions']) >= {'runtime', 'scenario', 'network', 'storage', 'fault', 'crash'}
        records = sorted(digest(path.read_bytes()) for path in
                         (work / 'artifacts/campaigns').rglob('simulation/record.json'))
        replays = json.loads((work / 'replays.json').read_bytes())
        assert len(records) == len(replays) == 8, (index, len(records), len(replays))
        assert all(row['status'] == 0 for row in replays), replays
        simulation.append(observed)
        raw_records.append(records)
        full_artifacts.append((work / 'authenticated-artifacts.json').read_bytes())
    assert simulation[0] == simulation[1], 'simulation observations changed under the same preparation root'
    assert raw_records[0] == raw_records[1], 'raw simulation record SHA-256 multiset changed'
    assert full_artifacts[0] == full_artifacts[1], 'complete raw execution record or artifact payload identities changed'

    scenario = args.work / 'scenario'
    run('scenario-live', [sys.executable, CORPUS / 'simulation-live-check.py', *common,
        '--work', scenario, '--preparation-root', args.preparation_root / 'scenario', '--scenario-only'])
    campaigns = list((scenario / 'artifacts/campaigns').glob('campaign-*'))
    assert len(campaigns) == 1, campaigns
    run('scenario-go-api', [sys.executable, CORPUS / 'simulation-scenario-reference-check.py',
        '--source', args.source, '--compiler', args.go_compiler, '--baseline', args.baseline,
        '--base', args.go_base, '--campaign', campaigns[0], '--work', args.work / 'scenario-go'])

    original = json.loads((args.work / 'simulation-0/commands.json').read_bytes())
    combined = next(row['argv'] for row in original if row['name'] == 'explore').copy()
    combined[0], combined[2] = str(args.baseline / 'bin/gomad'), str(args.work / 'choice/go-installation.json')
    combined[combined.index('--artifacts') + 1] = str(args.work / 'go-combined-refusal')
    combined[combined.index('--working-dir') + 1] = str(args.source / 'tools/gomad')
    run('go-public-combined-refusal', combined, expected=2)
    run('typed-divergence-symbols', [sys.executable, CORPUS / 'choice-live-check.py',
        '--binary', args.divergence_binary, '--ordinary-binary', args.binary,
        '--go-compiler', args.go_compiler, '--input', args.input,
        '--work', args.work / 'divergence', '--divergence-only'])
    run('current-model-owner-conformance', [args.go_compiler, 'test', '-p=1', '-tags', 'test_dep',
        '-count=1', '-timeout=10m', '-run',
        '^(TestNamespaceConformance|TestVolumeConformance|TestNetworkAdapterCorpus|TestNetworkClusterCorpus|TestBrokerLifecycleCorpus)',
        './internal/model/wasi'], cwd=MODULE, extra={
            'GOMAD_WASM_CONFORMANCE_HELPER': str(args.conformance_helper), 'GOMAD_WASM_CONFORMANCE_REQUIRED': '1'})
    result = dict(schema='gomad.exploration-semantic-parity/v1', requested_verb=args.verb,
        binary_sha256=digest(args.binary.read_bytes()), divergence_binary_sha256=digest(args.divergence_binary.read_bytes()),
        conformance_helper_sha256=digest(args.conformance_helper.read_bytes()), go_base_sha256=digest(args.go_base.read_bytes()),
        driver_sha256=digest(Path(__file__).read_bytes()), controller_cases=dict(choice=9, simulation=12),
        simulation_observations=simulation[0], repeated_raw_simulation_record_sha256=raw_records[0],
        go_combined_public_cli='refused status 2; no positive combined CLI parity claimed',
        scenario_reference='fresh original Go model.RunWASM API through fixture adapter; not public CLI',
        model_conformance='separate current Go Namespace/Volume/network adapter/network cluster/broker owners; not combined CLI parity',
        divergence_scope='explicit helper fixture role; ordinary role refuses with no committed execution')
    (args.work / 'exploration-semantic-parity.json').write_bytes(encode(result))
    print('exploration lane: controllers, actual public guests, exact simulation repetition, replays, scenario API reference, typed divergence and separate model conformance passed; Go combined CLI refusal retained')


if __name__ == '__main__':
    main()
