#!/usr/bin/env python3
"""Fresh Go/Rust simulation controller order, envelope and raw-chain corpus."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def encoded(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()


def digest(raw):
    return 'sha256:' + hashlib.sha256(raw).hexdigest()


def domain(name, raw):
    return digest(name.encode() + b'\0' + raw)


def first_difference(actual, expected, path='$'):
    if type(actual) is not type(expected):
        raise ValueError(f'{path}: type {type(actual).__name__} != {type(expected).__name__}')
    if isinstance(actual, dict):
        if actual.keys() != expected.keys():
            raise ValueError(f'{path}: keys {actual.keys()} != {expected.keys()}')
        for key in actual:
            first_difference(actual[key], expected[key], path + '.' + key)
    elif isinstance(actual, list):
        if len(actual) != len(expected):
            raise ValueError(f'{path}: length {len(actual)} != {len(expected)}')
        for index, (left, right) in enumerate(zip(actual, expected)):
            first_difference(left, right, f'{path}[{index}]')
    elif actual != expected:
        raise ValueError(f'{path}: {actual!r} != {expected!r}')


def request(case):
    dimensions = case.get('dimensions', ['runtime', 'scenario', 'network', 'storage', 'fault', 'crash'])
    controller = domain('gomad-simulation-exploration-controller/v1', b'deterministic-rounds/breadth-first-combined-rank-overrides/v2')
    config = dict(execution_sha256=digest(b'execution'), controller_sha256=controller,
                  base_seed=7, parallel=case.get('parallel', 8), max_executions=case.get('max_executions', 16),
                  max_forced_decisions=case.get('max_forced_decisions', 2),
                  max_exploration_bytes=case.get('max_exploration_bytes', 1 << 20),
                  max_result_bytes=case.get('max_result_bytes', 1 << 20), failure_budget=case.get('failure_budget', 8),
                  dimension_limits={name: case.get('dimension_limit', 8) for name in ['runtime', 'scenario', 'network', 'storage', 'fault', 'crash']})
    decisions = []
    for index, name in enumerate(dimensions):
        ordinal = case.get('ordinal', 0)
        alternatives = [digest(bytes([index, ordinal, rank + 1])) for rank in range(3 if name == 'scenario' else 2)]
        decisions.append(dict(dimension=name, ordinal=ordinal, site_sha256=digest(('site-' + name).encode()), alternatives=alternatives, selected=case.get('selected', 1 if name in ['scenario', 'fault', 'crash'] else 0)))
    result = dict(config=config, decisions=decisions)
    for key in ['failed_execution', 'diverged_execution']:
        if key in case:
            result[key] = case[key]
    return result


def authenticate(output):
    if len(output['states']) != len(output['segments']) + 1:
        raise ValueError('simulation state/segment count mismatch')
    for index, segment in enumerate(output['segments']):
        if segment['index'] != index:
            raise ValueError('simulation segment order mismatch')
        for key, state in [('before_sha256', output['states'][index]), ('after_sha256', output['states'][index + 1])]:
            if segment[key] != domain('gomad-simulation-exploration-state/v1', encoded(state)):
                raise ValueError(f'segment[{index}].{key}: raw state identity mismatch')
        raw = {key: value for key, value in segment.items() if key != 'sha256'}
        if segment['sha256'] != domain('gomad-simulation-exploration-round-segment/v1', encoded(raw)):
            raise ValueError(f'segment[{index}]: raw segment identity mismatch')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--go', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--corpus', type=Path, default=Path(__file__).with_name('simulation-controller-corpus.json'))
    parser.add_argument('--golden', type=Path, default=Path(__file__).with_name('goldens-simulation-controller.json'))
    args = parser.parse_args()
    gomad = Path(__file__).resolve().parents[2]
    corpus = json.loads(args.corpus.read_bytes())
    if corpus['schema'] != 'gomad.simulation-controller-inputs/v1':
        raise ValueError('unknown simulation corpus schema')
    args.output.mkdir(parents=True, exist_ok=False)
    source = gomad / 'internal/runner'
    golden = {'schema': 'gomad.simulation-controller-golden/v1', 'cases': {}}
    with tempfile.TemporaryDirectory(prefix='simulation-oracle-', dir=source) as scratch:
        scratch = Path(scratch)
        shutil.copyfile(Path(__file__).with_name('reference') / 'simulation-controller.go.txt', scratch / 'main.go')
        executable = scratch / 'oracle'
        environment = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOWORK': 'off', 'GOENV': 'off'}
        environment.pop('GOROOT', None)
        subprocess.run([str(args.go), 'build', '-trimpath', '-o', str(executable), './' + str(scratch.relative_to(gomad))], cwd=gomad, env=environment, check=True, timeout=600)
        for case in corpus['cases']:
            data = encoded(request(case))
            path = args.output / (case['name'] + '.input.json')
            path.write_bytes(data)
            rust = subprocess.run([str(args.binary), 'fixture-simulation-controller', '--input', str(path.resolve())], capture_output=True, timeout=120)
            reference = subprocess.run([str(executable)], input=data, capture_output=True, timeout=120)
            (args.output / (case['name'] + '.rust.stdout')).write_bytes(rust.stdout)
            (args.output / (case['name'] + '.rust.stderr')).write_bytes(rust.stderr)
            (args.output / (case['name'] + '.go.stdout')).write_bytes(reference.stdout)
            (args.output / (case['name'] + '.go.stderr')).write_bytes(reference.stderr)
            if 'error' in case:
                if rust.returncode == 0 or reference.returncode == 0 or case['error'].encode() not in rust.stderr or case['error'].encode() not in reference.stderr:
                    raise ValueError(case['name'] + ': missing explicit refusal')
                repeated = subprocess.run([str(args.binary), 'fixture-simulation-controller', '--input', str(path.resolve())], capture_output=True, timeout=120)
                if (repeated.returncode, repeated.stdout, repeated.stderr) != (rust.returncode, rust.stdout, rust.stderr):
                    raise ValueError(case['name'] + ': repeated refusal differs')
                golden['cases'][case['name']] = {'refused': True, 'status': rust.returncode}
                continue
            if rust.returncode != 0 or reference.returncode != 0:
                raise ValueError(case['name'] + ': nonzero exit')
            actual, expected = json.loads(rust.stdout), json.loads(reference.stdout)
            authenticate(actual)
            # Only absence of Go's optional zero stop reason is representation-specific.
            for row in expected['view']:
                row['summary'].setdefault('stop_reason', None)
            first_difference(actual['view'], expected['view'], case['name'])
            repeated = subprocess.run([str(args.binary), 'fixture-simulation-controller', '--input', str(path.resolve())], capture_output=True, timeout=120, check=True)
            if repeated.stdout != rust.stdout:
                raise ValueError(case['name'] + ': repeated raw bytes differ')
            golden['cases'][case['name']] = {
                'view': actual['view'],
                'state_sha256': [digest(encoded(state)) for state in actual['states']],
                'segment_sha256': [segment['sha256'] for segment in actual['segments']],
            }
    if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
        args.golden.write_bytes(encoded(golden) + b'\n')
    else:
        first_difference(golden, json.loads(args.golden.read_bytes()), 'golden')
    print(f"simulation controller: {len(corpus['cases'])} fresh Go/Rust cases agree")


if __name__ == '__main__':
    main()
