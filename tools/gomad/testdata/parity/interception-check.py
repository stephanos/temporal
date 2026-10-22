#!/usr/bin/env python3
"""Run prepared public launcher cases twice and authenticate their exact evidence.

Inputs are actual retained preparation receipts and modules, supplied after the
registered Go proof. This runner never builds a replacement module or installation.
No commands run at import time. See interception-proof-plan.md for the input schema.
"""
import argparse
import copy
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import shlex
import signal
import subprocess
import sys

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent


def load(name, path):
    loader = SourceFileLoader(name, str(path))
    module = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(module)
    return module


campaign = load('interception_campaign', HERE / 'campaign-check.py')
qualification = load('interception_qualification', HERE / 'qualify-lane.py')
parity = campaign.parity


def equal(actual, expected, label):
    campaign.equal(actual, expected, label)


def document(path):
    raw = path.read_bytes()
    if len(raw) > 8 << 20:
        raise ValueError('interception input exceeds 8MiB')
    return parity.decode(raw)


def normalize(value, paths):
    if isinstance(value, dict):
        return {key: normalize(item, paths) for key, item in value.items()}
    if isinstance(value, list):
        return [normalize(item, paths) for item in value]
    if isinstance(value, str):
        for original, replacement in sorted(paths.items(), key=lambda pair: -len(pair[0])):
            value = value.replace(original, replacement)
    return value


def mapped_status(outcome):
    if outcome['class'] == 'success':
        return 0
    if outcome['class'] == 'target_failure':
        code = outcome['exit_code']
        return code if outcome['termination'] == 'exit' and code is not None and 1 <= code <= 255 else 1
    return 3


def stable_rows(rows, paths):
    rows = normalize(copy.deepcopy(rows), paths)
    for row in rows:
        if row.get('type') == 'outcome':
            # Each launch has a distinct authenticated origin. These exact
            # commitments were checked against raw retained bytes above.
            row['record_sha256'] = '$RECORD/' + str(row['ordinal'])
            row['origin_sha256'] = '$ORIGIN'
    return rows


class Runner:
    def __init__(self, args):
        self.args = args
        self.commands = []
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith('GOMAD_')}

    def run(self, name, argv, status, extra=None):
        argv = list(map(str, argv))
        process = subprocess.Popen(argv, env=dict(self.environment, **(extra or {})),
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   start_new_session=True)
        try:
            stdout, stderr = process.communicate(timeout=self.args.timeout)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                stdout, stderr = process.communicate(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                stdout, stderr = process.communicate()
            self.save(name, argv, process.returncode, stdout, stderr, expired=True)
            raise ValueError(name + ': original overall deadline expired; not retried')
        self.save(name, argv, process.returncode, stdout, stderr)
        equal(process.returncode, status, name + ': ' + stderr.decode(errors='replace'))
        return stdout, stderr

    def save(self, name, argv, status, stdout, stderr, expired=False):
        (self.args.work / (name + '.stdout')).write_bytes(stdout)
        (self.args.work / (name + '.stderr')).write_bytes(stderr)
        self.commands.append(dict(name=name, argv=argv, status=status,
                                  overall_deadline_expired=expired,
                                  stdout_sha256=parity.digest(stdout),
                                  stderr_sha256=parity.digest(stderr)))
        (self.args.work / 'commands.json').write_bytes(parity.canonical(self.commands) + b'\n')


def qualification_refusal(runner, name, artifact, root):
    args = runner.args
    provenance = campaign.canonical_file(artifact / 'target/provenance.json')
    origin = campaign.canonical_file(artifact / 'target/intercepted-origin.json')
    equal(provenance['origin'], origin, name + '.sealed_origin')
    # First prove the unmodified Result reaches the accepting public reader.
    runner.run(name + '-report-baseline', [args.binary, 'report', '--result', args.qualification_result], 0)
    _, rewrite = qualification.tamperers(args.qualification_result, root / 'report-origin')
    refused = rewrite('sealed-intercepted-provenance',
                      lambda result: result.__setitem__('provenance', provenance))
    _, stderr = runner.run(name + '-report-origin-refusal',
                           [args.binary, 'report', '--result', refused], 3)
    equal(b'retained provenance does not bind the qualification identities' in stderr,
          True, name + '.actual_provenance_boundary')
    # Qualify itself has source-only grammar. This is recorded separately, not
    # mislabelled as either a fixture-role refusal or the provenance check above.
    spec = dict(Manifest=dict(Provisional=True, IntegratedSourceSHA256='0' * 64),
                Campaign=dict(Target=dict(Backend='wasm', Kind='exec', Source=str(artifact))),
                ArtifactRoot=str(root / 'qualification'))
    path = root / 'qualification-source-refusal.json'
    path.write_bytes(parity.canonical(spec))
    _, stderr = runner.run(name + '-qualify-source-refusal',
                           [args.release_binary, 'qualify', '--spec', path], 3)
    equal(b'unsupported WASM source preparation request' in stderr,
          True, name + '.public_source_boundary')
    return dict(report_origin_status=3, public_source_status=3)


def execute_case(runner, case, root, label):
    args = runner.args
    name = label + '-' + case['name']
    destination = root / case['name']
    destination.mkdir(mode=0o700)
    argv = [args.binary, '--installation', args.installation, 'run-guest',
            '--artifacts', destination, '--seeds', case['seeds']]
    if case.get('json', False):
        argv.append('--json')
    argv += [Path(case['module']), *case.get('args', [])]
    extra = {}
    if case['name'] != 'unprepared-module':
        extra = dict(GOMAD_PREPARATION_RECEIPT=case['receipt'],
                     GOMAD_PREPARATION_RECEIPT_SHA256=case['receipt_sha256'])
    stdout, stderr = runner.run(name, argv, case['status'], extra)
    roots = sorted((destination / 'campaigns').glob('campaign-*'))
    paths = {str(destination): '$ARTIFACTS'}
    observation = dict(status=case['status'])
    if case['name'] == 'unprepared-module':
        equal(roots, [], name + '.no_campaign')
        equal(stdout, b'', name + '.no_guest_stdout')
        equal(stderr.splitlines()[-1], b'gomad run-guest: no replayable artifact was retained', name + '.no_replay')
        inputs = list((destination / 'unaccepted-inputs').glob('*/input.json'))
        equal(len(inputs), 1, name + '.input_retained')
        metadata = document(inputs[0])
        equal((metadata['accepted'], metadata['replayable_artifact']), (False, False), name + '.unaccepted_label')
        equal(metadata['sha256'], parity.digest(Path(case['module']).read_bytes()), name + '.actual_input')
        return dict(status=case['status'], no_campaign=True, unaccepted_input=True, module_sha256=metadata['sha256'])
    equal(len(roots), 1, name + '.one_launcher_campaign')
    validated = campaign.authenticate_campaign(roots[0])
    equal(validated['plan']['origin']['origin'], 'intercepted-go', name + '.campaign_origin')
    executions = sorted(validated['executions'], key=lambda row: row['ordinal'])
    equal(len(executions), len(case['outcomes']), name + '.attempted_only')
    rows = []
    guest_stdout, guest_stderr = bytearray(), bytearray()
    for execution, expected in zip(executions, case['outcomes']):
        artifact = roots[0] / execution['artifact']
        record = campaign.authenticate_artifact(artifact)
        paths[str(artifact)] = '$ARTIFACT/' + str(execution['ordinal'])
        equal((execution['ordinal'], execution['seed']), (expected['ordinal'], expected['seed']), name + '.selection')
        equal((record['outcome']['class'], record['outcome']['termination']),
              (expected['class'], expected['termination']), name + '.outcome')
        manifest = campaign.canonical_file(artifact / 'manifest.json')
        equal(mapped_status(record['outcome']), expected['status'], name + '.status_mapping')
        row = dict(type='outcome', origin='intercepted-go', ordinal=execution['ordinal'], seed=execution['seed'],
                   status=expected['status'], **record['outcome'], artifact_path=str(artifact), stored_bytes=execution['stored_bytes'],
                   record_sha256=manifest['record_sha256'], execution_identity=record['execution_identity'],
                   module_sha256=record['identity']['target'], origin_sha256=record['streams']['target/intercepted-origin.json']['sha256'])
        for stream, output in [('stdout', guest_stdout), ('stderr', guest_stderr)]:
            path = artifact / 'output' / stream
            if path.exists():
                output.extend(path.read_bytes())
        rows.append(row)
        runner.run(name + '-inspect-' + str(execution['ordinal']), [args.binary, 'inspect', '--json', artifact], 0)
    if case.get('json', False):
        actual = [parity.decode(line) for line in stdout.splitlines()]
        equal(len(actual), len(rows) + 1, name + '.row_count')
        for reported, expected in zip(actual[:-1], rows):
            equal({key: reported[key] for key in expected}, expected, name + '.bound_row')
        equal(actual[-1], dict(type='result', origin='intercepted-go', campaign_path=str(roots[0]),
                              status=case['status'], attempted=len(rows),
                              unexecuted=validated['plan']['selection_count'] - len(rows)), name + '.summary')
    else:
        equal(stdout, bytes(guest_stdout), name + '.verbatim_guest_only_stdout')
    forwarded_stderr = b'' if case.get('json', False) else guest_stderr
    equal(stderr.startswith(forwarded_stderr), True, name + '.guest_stderr_prefix')
    if case['status']:
        if case['replayable']:
            artifact = roots[0] / executions[-1]['artifact']
            expected = ' '.join(shlex.quote(str(item)) for item in
                                [args.binary, '--installation', args.installation, 'replay', artifact])
            equal(shlex.split(stderr.decode().splitlines()[-1]), shlex.split(expected), name + '.exact_final_replay')
        else:
            equal(stderr.splitlines()[-1], b'gomad run-guest: no replayable artifact was retained', name + '.no_invented_replay')
    if case['replayable']:
        artifact = roots[0] / executions[-1]['artifact']
        replay, _ = runner.run(name + '-replay', [args.binary, '--installation', args.installation, 'replay', artifact],
                               int(executions[-1]['record']['outcome']['class'] != 'success'))
        equal(b'reproduced=true' in replay, True, name + '.exact_replay')
    if case.get('later_would_pass_seed') is not None:
        probe = dict(case, name=case['name'] + '-later-probe', seeds=str(case['later_would_pass_seed']),
                     status=0, json=True, outcomes=[dict(ordinal=0, seed=case['later_would_pass_seed'],
                                                       status=0, **case['later_outcome'])])
        probe.pop('later_would_pass_seed')
        execute_case(runner, probe, root, label)
    if case['name'] == 'failure-with-replay':
        artifact = roots[0] / executions[-1]['artifact']
        reduced, _ = runner.run(name + '-minimize', [args.binary, 'minimize', '--json', '--attempt-budget=1',
                                                    '--artifacts', destination / 'reductions', artifact], 0)
        reduced = parity.decode(reduced)
        minimized = Path(reduced['artifact_path'])
        campaign.authenticate_artifact(minimized)
        equal(campaign.canonical_file(minimized / 'target/intercepted-origin.json'),
              campaign.canonical_file(artifact / 'target/intercepted-origin.json'), name + '.minimized_origin')
        observation['qualification'] = qualification_refusal(runner, name, artifact, destination)
    paths[str(roots[0])] = '$CAMPAIGN'
    observation.update(rows=stable_rows(rows, paths), stdout_sha256=parity.digest(stdout if not case.get('json') else parity.canonical(stable_rows(actual, paths))),
                       guest_stdout_sha256=parity.digest(guest_stdout), guest_stderr_sha256=parity.digest(guest_stderr),
                       diagnostics=normalize(stderr.decode(), paths), unexecuted=validated['plan']['selection_count'] - len(rows))
    return observation


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'release-binary', 'installation', 'qualification-result', 'inputs', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--timeout', type=int, default=600)
    args = parser.parse_args()
    if not 1 <= args.timeout <= 600:
        parser.error('timeout must preserve the original overall bound, 1..600 seconds')
    for name in ('binary', 'release_binary', 'installation', 'qualification_result', 'inputs', 'work'):
        setattr(args, name, getattr(args, name).resolve())
    declaration = document(HERE / 'interception-corpus.json')
    inputs = document(args.inputs)
    equal(inputs['schema'], 'gomad.interception-corpus-inputs/v1', 'inputs.schema')
    equal(sorted(case['name'] for case in inputs['cases']), sorted(declaration['required_cases']), 'complete_case_set')
    for case in inputs['cases']:
        if len(case['outcomes']) > 64:
            raise ValueError('focused corpus case exceeds 64 expected attempts')
        for key in ('module', *(() if case['name'] == 'unprepared-module' else ('receipt',))):
            if not Path(case[key]).is_absolute():
                raise ValueError('prepared corpus input paths must be absolute: ' + key)
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    runner = Runner(args)
    views = []
    for repeat in range(2):
        root = args.work / str(repeat)
        root.mkdir(mode=0o700)
        views.append({case['name']: execute_case(runner, copy.deepcopy(case), root, str(repeat)) for case in inputs['cases']})
    equal(views[0], views[1], 'byte-identical_second_regeneration')
    encoded = parity.canonical(dict(schema='gomad.interception-golden/v1', cases=views[0])) + b'\n'
    (args.work / 'observed-golden.json').write_bytes(encoded)
    golden = HERE / declaration['golden']
    if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
        golden.parent.mkdir(exist_ok=True)
        golden.write_bytes(encoded)
    else:
        equal(golden.read_bytes(), encoded, 'reviewed_golden')
    print('interception public corpus passed twice')


if __name__ == '__main__':
    main()
