#!/usr/bin/env python3
"""Transferred roots 18, 27 and 28: real service and Go-test oracle evidence."""
import argparse
import base64
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path
import runpy
import subprocess
import sys

sys.dont_write_bytecode = True
common = runpy.run_path(str(Path(__file__).with_name('live-cluster-check.py')))
canonical = common['canonical']
handle = common['handle']
step = common['step']


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'input', 'service-module', 'softassert-module', 'capacity-module', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    args.work = args.work.resolve()
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    template = json.loads(args.input.read_bytes())
    receipts = []
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}

    def run(name, arguments, status=0, detached=False):
        env = dict(environment)
        if detached:
            env.update(PATH='/compiler/unavailable', GOMAD_STOCK_GO='/compiler/unavailable',
                       GOROOT='/runtime/unavailable', GOTOOLCHAIN='local')
        command = [str(args.binary.resolve()), *map(str, arguments)]
        result = subprocess.run(command, capture_output=True, env=env, timeout=600)
        receipts.append(dict(name=name, argv=command, status=result.returncode, expected=status,
            stdout=result.stdout.decode(errors='replace'), stderr=result.stderr.decode(errors='replace')))
        (args.work / 'commands.json').write_bytes(canonical(receipts))
        assert result.returncode == status, receipts[-1]
        return result

    def configuration(module, arguments, seed=7, test=False):
        config = common['configuration'](template, 'lifecycle')
        base = config['incarnations'][0]
        raw = module.read_bytes()
        if module != args.softassert_module:
            assert len(raw) <= 8 << 20, 'original S admission bound'
        base.update(arguments=arguments, workload_kind='go-test' if test else 'go-run', storage=None)
        base['model']['diagnostics'] = True
        base['module'] = dict(file='target/module.wasm', bytes=len(raw), sha256=digest(raw), limit=512 << 20)
        base['choice_identity']['TargetSHA256'] = list(bytes.fromhex(digest(raw)))
        base['limits']['wall_millis'] = 300000 if module == args.softassert_module else 60000
        config['limits'] = deepcopy(base['limits'])
        config['broker']['seed'] = seed
        config['incarnations'] = [deepcopy(base), deepcopy(base)]
        for node, item in zip(('a', 'b'), config['incarnations']):
            item['handle'] = handle(node)
        config['steps'] = [step(action, node) for node in ('a', 'b') for action in ('start', 'wait')]
        return config, raw

    def record(name, config, raw):
        root = args.work / name
        root.mkdir(mode=0o700)
        executable = root / 'original.wasm'
        executable.write_bytes(raw)
        request = dict(schema='gomad.cluster-fixture-input/v1', toolchain_sha256=template['toolchain_sha256'],
            configuration=config, modules={'target/module.wasm': str(executable)})
        input_path = root / 'input.json'
        input_path.write_bytes(canonical(request))
        destination = root / 'recording'
        run(name + '-record', ['fixture-live-cluster', '--root', destination, '--input', input_path])
        bundle = json.loads((destination / 'foundation-result.json').read_bytes())
        assert len(bundle['records']) == 1
        artifact = destination / bundle['records'][0]['artifact']
        report = json.loads((artifact / 'evidence/cluster_oracles').read_bytes())
        return root, executable, artifact, report

    def detached(name, executable, artifact):
        before = {str(p.relative_to(artifact)): digest(p.read_bytes()) for p in artifact.rglob('*') if p.is_file()}
        executable.unlink()
        result = run(name + '-detached-replay', ['replay', artifact], status=1 if name.endswith('canary') or name == 'softassert-7-False' else 0, detached=True)
        assert b'reproduced=true' in result.stdout, result.stdout
        assert before == {str(p.relative_to(artifact)): digest(p.read_bytes()) for p in artifact.rglob('*') if p.is_file()}

    for canary in (False, True):
        config, raw = configuration(args.service_module, ['service', 'observation'])
        if canary:
            config['incarnations'][1]['arguments'].append('canary')
        config['steps'] = [step('start', 'a'), step('start', 'b'), step('wait', 'a'), step('wait', 'b')]
        config['oracle_plan'] = dict(schema='gomad.wasm.history-observation/v1', prefix='GOMAD_HISTORY_OBSERVATION ', maximum_bytes=4096,
            histories=[dict(name='fixture.recovered', expected=handle('a'), actual=handle('b'), expected_events=['1', '2'])],
            convergence=[dict(name='fixture.converged', handles=[handle('a'), handle('b')])])
        name = 'history-canary' if canary else 'history-completed'
        root, executable, artifact, report = record(name, config, raw)
        assert report['outcome'] == ('oracle_failed' if canary else 'completed')
        assert len(report['oracles']) == 4
        assert [item['name'] for item in report['oracles']] == [
            'fixture.recovered.history', 'fixture.recovered.events', 'fixture.recovered.identity', 'fixture.converged']
        detached(name, executable, artifact)

    signature = None
    for report_only, seeds in ((False, (7, 19)), (True, (7,))):
        for seed in seeds:
            config, raw = configuration(args.softassert_module,
                ['soft-assertion', '-test.v', '-assertion-mode=late'], seed=seed, test=True)
            if report_only:
                config['oracle_plan'] = dict(schema='gomad.wasm.soft-assertion-observation/v1', maximum_bytes=4096,
                    soft_assertions=[dict(handle=handle(node), report_only=True) for node in ('a', 'b')])
            name = f'softassert-{seed}-{report_only}'
            root, executable, artifact, report = record(name, config, raw)
            assert report['outcome'] == ('completed' if report_only else 'oracle_failed') and len(report['oracles']) == 2
            for oracle in report['oracles']:
                assert oracle['soft_assertion'] and oracle['report_only'] == report_only and oracle['passed'] == report_only
                assert base64.b64decode(oracle['evidence'][1]['value']) == b'["invalid fixture state"]'
            if not report_only:
                assert report['reason'] == 'soft_assertion' and report['failure_identity']
                if signature is None:
                    signature = report['failure_identity']
                assert signature == report['failure_identity']
            summary = json.loads((artifact / 'evidence/cluster_summary').read_bytes())
            for item in summary['incarnations']:
                assert item['termination'] == 'exit' and item['exit_code'] == 0
                output = artifact / 'evidence/cluster' / item['handle']['node'] / '1/stdout'
                assert b'PASS' in output.read_bytes()
            if seed == 7:
                detached(name, executable, artifact)

    for report_only, maximum in ((False, 4096), (True, 4096), (True, 1)):
        config, raw = configuration(args.capacity_module, ['capacity', '-test.run=TestOutputCapacity'], test=True)
        config['nodes'] = ['a']
        config['incarnations'] = config['incarnations'][:1]
        config['steps'] = [step('start', 'a'), step('wait', 'a')]
        config['network']['addresses'].pop('b')
        config['network']['links'] = []
        config['limits']['output_bytes'] = 4
        config['incarnations'][0]['limits']['output_bytes'] = 4
        config['incarnations'][0]['namespace']['limits']['output_bytes'] = 4
        if report_only:
            config['oracle_plan'] = dict(schema='gomad.wasm.soft-assertion-observation/v1', maximum_bytes=maximum,
                soft_assertions=[dict(handle=handle('a'), report_only=True)])
        name = f'capacity-{report_only}-{maximum}'
        root = args.work / name
        root.mkdir(mode=0o700)
        request = dict(schema='gomad.live-cluster-observation/v1', configuration=config, modules={'target/module.wasm': str(args.capacity_module.resolve())})
        path = root / 'input.json'
        path.write_bytes(canonical(request))
        result = run(name, ['fixture-live-cluster', '--root', root / 'observed', '--input', path])
        observed = json.loads(result.stdout)
        assert len(observed['incarnations']) == 1 and observed['execution_returned']
        assert not list((root / 'observed').rglob('manifest.json')), 'capacity acquired retained evidence'
        guest = observed['incarnations'][0]
        assert guest['outcome']['termination'] == 'capacity'
        assert (root / 'observed/incarnations/a/1/stdout').read_bytes() == b'PASS'
        assert guest['outcome']['output_bytes'] == 4 and guest['choice_bytes'] == 0
        assessment = observed['oracle_assessment']
        failure = assessment['capacity_failure']
        assert failure['termination'] == 'capacity' and failure['message'] == 'output capacity'
        assert failure['soft_assertions']['inconclusive'] and failure['soft_assertions']['report_only'] == report_only
        expected = 'resource=oracle_evidence_bytes' if maximum == 1 else 'choice trace is not complete'
        assert expected in failure['detail'], failure
        if maximum == 1:
            assert expected in assessment['oracle_error'] and not assessment['oracles']
        else:
            assert len(assessment['oracles']) == 1
            oracle = assessment['oracles'][0]
            assert not oracle['passed'] and oracle['inconclusive'] and oracle['report_only'] == report_only
    (args.work / 'passed.json').write_bytes(canonical(dict(roots=[18, 27, 28], commands=len(receipts))))


if __name__ == '__main__':
    main()
