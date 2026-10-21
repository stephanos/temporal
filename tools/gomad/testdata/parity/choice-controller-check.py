#!/usr/bin/env python3
"""Fresh Go/Rust bounded choice controller and authenticated raw-chain corpus."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import runpy
import shutil
import subprocess
import tempfile

helpers = runpy.run_path(str(Path(__file__).with_name('simulation-controller-check.py')))
encoded, digest, domain, first_difference = [helpers[name] for name in
                                           ['encoded', 'digest', 'domain', 'first_difference']]


def request(case, traces):
    identity = dict(TargetSHA256=list(hashlib.sha256(b'target').digest()),
                    ToolchainBuildKey='a' * 64, GOOS='darwin', GOARCH='arm64',
                    ImplementationSHA256=list(hashlib.sha256(b'controller').digest()))
    config = dict(execution=identity,
                  controller_sha256=domain('gomad-choice-exploration-controller/v1',
                                          b'deterministic-rounds/breadth-first-rank-prefix/v3'),
                  base_seed=7, parallel=case.get('parallel', 2),
                  max_executions=case.get('max_executions', 16),
                  max_choice_depth=case.get('max_choice_depth', 2),
                  start_ordinal=case.get('start_ordinal', 0),
                  max_exploration_bytes=case.get('max_exploration_bytes', 1 << 20),
                  failure_policy=case.get('failure_policy', 'all'),
                  failure_budget=case.get('failure_budget', 1))
    return dict(config=config, traces=traces, failed_executions=case.get('failed_executions', []))


def authenticate(output):
    if len(output['states']) != len(output['segments']) + 1:
        raise ValueError('choice state/segment count mismatch')
    for index, segment in enumerate(output['segments']):
        if segment['index'] != index:
            raise ValueError('choice segment order mismatch')
        for field, state in [('before_sha256', output['states'][index]),
                             ('after_sha256', output['states'][index + 1])]:
            if segment[field] != domain('gomad-choice-exploration-state/v2', encoded(state)):
                raise ValueError(f'segment[{index}].{field}: raw state identity mismatch')
        raw = {**segment, 'sha256': ''}
        if segment['sha256'] != domain('gomad-choice-exploration-round-segment/v3', encoded(raw)):
            raise ValueError(f'segment[{index}]: raw segment identity mismatch')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--go', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--corpus', type=Path, default=Path(__file__).with_name('choice-controller-corpus.json'))
    parser.add_argument('--golden', type=Path, default=Path(__file__).with_name('goldens-choice-controller.json'))
    args = parser.parse_args()
    gomad = Path(__file__).resolve().parents[2]
    corpus = json.loads(args.corpus.read_bytes())
    if corpus['schema'] != 'gomad.choice-controller-inputs/v1':
        raise ValueError('unknown choice corpus schema')
    args.output.mkdir(parents=True, exist_ok=False)
    golden = {'schema': 'gomad.choice-controller-golden/v1', 'cases': {}}
    with tempfile.TemporaryDirectory(prefix='choice-oracle-', dir=gomad / 'internal/runner') as scratch:
        scratch = Path(scratch)
        shutil.copyfile(Path(__file__).with_name('reference') / 'choice-controller.go.txt', scratch / 'main.go')
        executable = scratch / 'oracle'
        environment = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOWORK': 'off', 'GOENV': 'off'}
        environment.pop('GOROOT', None)
        subprocess.run([str(args.go), 'build', '-trimpath', '-o', str(executable), './' + str(scratch.relative_to(gomad))], cwd=gomad, env=environment, check=True, timeout=600)
        raw_traces = subprocess.run([str(executable), '--traces'], capture_output=True, check=True, timeout=120).stdout
        (args.output / 'canonical-controller-traces.json').write_bytes(raw_traces)
        traces = json.loads(raw_traces)
        for case in corpus['cases']:
            data = encoded(request(case, traces))
            path = args.output / (case['name'] + '.input.json')
            path.write_bytes(data)
            command = [str(args.binary), 'fixture-choice-controller', '--input', str(path.resolve())]
            rust = subprocess.run(command, capture_output=True, timeout=120)
            reference = subprocess.run([str(executable)], input=data, capture_output=True, timeout=120)
            for name, result in [('rust', rust), ('go', reference)]:
                for stream in ['stdout', 'stderr']:
                    (args.output / (case['name'] + '.' + name + '.' + stream)).write_bytes(getattr(result, stream))
            if rust.returncode or reference.returncode:
                raise ValueError(case['name'] + ': nonzero exit; see retained stderr')
            actual, expected = json.loads(rust.stdout), json.loads(reference.stdout)
            authenticate(actual)
            for row in expected['view']:
                row['summary'].setdefault('start_ordinal', 0)
                row['summary'].setdefault('stop_reason', '')
            first_difference(actual['view'], expected['view'], case['name'])
            repeated = subprocess.run(command, capture_output=True, check=True, timeout=120)
            repeated_go = subprocess.run([str(executable)], input=data, capture_output=True, check=True, timeout=120)
            if repeated.stdout != rust.stdout or repeated_go.stdout != reference.stdout:
                raise ValueError(case['name'] + ': repeated raw bytes differ')
            stop = actual['view'][-1]['summary']['stop_reason']
            if stop != case['stop_reason']:
                raise ValueError(f"{case['name']}: stop {stop!r} != {case['stop_reason']!r}")
            golden['cases'][case['name']] = dict(view=actual['view'],
                state_sha256=[digest(encoded(state)) for state in actual['states']],
                segment_sha256=[segment['sha256'] for segment in actual['segments']])
    if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
        args.golden.write_bytes(encoded(golden) + b'\n')
    else:
        first_difference(golden, json.loads(args.golden.read_bytes()), 'golden')
    print(f"choice controller: {len(corpus['cases'])} fresh Go/Rust cases agree")


if __name__ == '__main__':
    main()
