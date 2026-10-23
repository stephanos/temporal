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
            row['stored_bytes'] = '$AUTHENTICATED_STORED_BYTES'
    return rows


def selected_seeds(text):
    if not isinstance(text, str) or len(text) > 256:
        raise ValueError('bounded seed selection required')
    selected = []
    for term in text.split(','):
        endpoints = term.split('-')
        if len(endpoints) not in (1, 2) or any(not value.isascii() or not value.isdecimal() for value in endpoints):
            raise ValueError('invalid seed selection')
        first, last = int(endpoints[0]), int(endpoints[-1])
        if not 0 <= first <= last < 1 << 64 or last - first >= 64 or len(selected) + last - first + 1 > 64:
            raise ValueError('focused corpus selection exceeds 64 seeds')
        selected.extend(range(first, last + 1))
    if len(set(selected)) != len(selected):
        raise ValueError('duplicate seed selection')
    return selected


def validate_cases(inputs, declaration):
    equal(inputs['schema'], 'gomad.interception-corpus-inputs/v1', 'inputs.schema')
    equal(sorted(case['name'] for case in inputs['cases']), sorted(declaration['required_cases']), 'complete_case_set')
    for case in inputs['cases']:
        name = case['name']
        selected = selected_seeds(case['seeds'])
        if type(case['json']) is not bool or type(case['replayable']) is not bool or type(case['status']) is not int:
            raise ValueError(name + ': explicit reporting/replay/status types required')
        expected = case['outcomes']
        if len(expected) > len(selected):
            raise ValueError(name + ': more outcomes than selected seeds')
        failed = False
        for ordinal, row in enumerate(expected):
            equal((row['ordinal'], row['seed']), (ordinal, selected[ordinal]), name + '.expected_prefix')
            if failed:
                raise ValueError(name + ': executions declared after first failure')
            failed = row['status'] != 0
            if row['class'] == 'success':
                equal((row['termination'], row['status']), ('exit', 0), name + '.expected_success')
            elif not failed:
                raise ValueError(name + ': non-success outcome cannot have success status')
        equal(case['status'], expected[-1]['status'] if expected else 2, name + '.expected_terminal_status')
        if name == 'unprepared-module':
            equal((case['status'], expected, case['json'], case['replayable']), (2, [], False, False), name + '.refusal')
            if 'receipt' in case or 'receipt_sha256' in case:
                raise ValueError(name + ': unprepared case must not supply preparation metadata')
        else:
            if not expected:
                raise ValueError(name + ': real attempted outcome required')
            digest = case['receipt_sha256']
            if len(digest) != 64 or any(byte not in '0123456789abcdef' for byte in digest):
                raise ValueError(name + ': exact receipt digest required')
            if not Path(case['receipt']).is_absolute():
                raise ValueError(name + ': absolute receipt path required')
        if not Path(case['module']).is_absolute():
            raise ValueError(name + ': absolute module path required')
        if name in ('success', 'standalone-json'):
            equal((case['status'], len(expected)), (0, len(selected)), name + '.all_selected_pass')
            equal(case['json'], name == 'standalone-json', name + '.report_mode')
            if name == 'standalone-json' and len(expected) < 2:
                raise ValueError(name + ': two genuine outcome rows required')
        if name == 'first-failure':
            equal((len(expected), expected[0]['class'], case['json']), (1, 'target_failure', False), name + '.first_seed_fails')
            if len(selected) < 2 or case.get('later_would_pass_seed') not in selected[1:]:
                raise ValueError(name + ': a later selected would-pass seed is mandatory')
            equal(case.get('later_outcome'), dict(**{'class':'success'}, termination='exit'), name + '.mandatory_pass_probe')
        if name == 'failure-with-replay':
            equal((expected[-1]['class'], case['replayable'], case['json']), ('target_failure', True, False), name + '.replayable_target_failure')
        if name == 'infrastructure':
            equal((case['status'], len(expected), case['json']), (3, 1, False), name + '.real_infrastructure')
            if expected[0]['class'] not in ('runner_failure', 'watchdog') or expected[0]['termination'] not in ('capacity', 'fuel', 'infrastructure', 'watchdog_timeout', 'cancelled'):
                raise ValueError(name + ': actual infrastructure outcome required')
            if len(selected) < 2:
                raise ValueError(name + ': retain an unexecuted later seed')


def bind_module_input(record, manifest, module_sha256, module_bytes):
    entry = manifest['files']['target/module.wasm']
    equal(record['identity']['target'], module_sha256, 'supplied_module.target')
    equal((entry['sha256'], entry['bytes']), (module_sha256, module_bytes), 'supplied_module.bytes')


def authenticate_origin(artifact, record, plan_origin):
    name = 'target/intercepted-origin.json'
    raw = (artifact / name).read_bytes()
    if len(raw) > 32 << 20:
        raise ValueError('sealed origin exceeds existing origin bound')
    origin = campaign.canonical_file(artifact / name)
    parity.fields(origin, dict(schema=str, origin=str, invocation=dict, module_sha256=str,
                               preparation_sha256=str, launch=dict), 'sealed_origin')
    equal((origin['schema'], origin['origin']), ('gomad.intercepted-origin/v1', 'intercepted-go'), 'origin.schema')
    invocation, launch = origin['invocation'], origin['launch']
    parity.fields(invocation, dict(schema=str, wrapper_run_id=str, invocation_id=str, original_argv=list,
                                   driver_argv=list, compiler_sha256=str, installation_sha256=str,
                                   recipe_sha256=str, recipe_file=str), 'origin.invocation')
    launch_fields = dict(invocation_id=str, argv=list, process_id=int, parent_process_id=(int, type(None)))
    if 'preparation_receipt_sha256' in launch:
        launch_fields['preparation_receipt_sha256'] = str
        parity.sha(launch['preparation_receipt_sha256'], 'origin.launch.receipt_sha256')
    parity.fields(launch, launch_fields, 'origin.launch')
    equal(invocation['schema'], 'gomad.intercepted-invocation/v1', 'origin.invocation.schema')
    for value in [origin['module_sha256'], origin['preparation_sha256'], launch['invocation_id'],
                  *[invocation[key] for key in ('wrapper_run_id', 'invocation_id', 'compiler_sha256',
                                                'installation_sha256', 'recipe_sha256')]]:
        parity.sha(value, 'origin.digest')
    if launch['invocation_id'] in (invocation['wrapper_run_id'], invocation['invocation_id']):
        raise ValueError('launch identity aliases driver or wrapper')
    for pid in (launch['process_id'], launch['parent_process_id']):
        if pid is not None and not 0 < pid < 1 << 32:
            raise ValueError('invalid sealed launch PID')
    for argv in (invocation['original_argv'], invocation['driver_argv'], launch['argv']):
        if not 0 < len(argv) <= 65536 or any(type(value) is not str or len(value) > 2 << 20 or len(value) % 2
                                           or any(byte not in '0123456789abcdef' for byte in value) for value in argv):
            raise ValueError('invalid sealed origin argv bytes')
    if not invocation['recipe_file'] or invocation['recipe_file'] in ('.', '..') or any(value in invocation['recipe_file'] for value in ('/', '\\')):
        raise ValueError('invalid sealed recipe filename')
    manifest = campaign.canonical_file(artifact / 'manifest.json')
    equal(manifest['record'], record, 'origin.artifact_record')
    entry = manifest['files'][name]
    equal((entry['bytes'], entry['sha256'], entry['module']), (len(raw), parity.digest(raw), False), 'origin.manifest')
    equal(record['streams'].get(name), dict(bytes=len(raw), sha256=parity.digest(raw), records=1), 'origin.record_stream')
    equal(origin['module_sha256'], record['identity']['target'], 'origin.module')
    provenance = campaign.canonical_file(artifact / 'target/provenance.json')
    equal(provenance['origin'], origin, 'origin.provenance')
    equal(provenance['module_sha256'], origin['module_sha256'], 'origin.provenance_module')
    base = dict(provenance)
    del base['origin']
    equal(parity.digest(parity.canonical(base)), origin['preparation_sha256'], 'origin.preparation')
    equal(origin, plan_origin, 'origin.plan_origin')
    return origin


def campaign_roles(plans):
    executed, prepared = [], []
    for path, plan in plans:
        if 'launcher' in plan:
            if type(plan['launcher']) is not bool or not plan['launcher']:
                raise ValueError('invalid launcher campaign role')
            executed.append(path)
        else:
            prepared.append(path)
    equal((len(executed), len(prepared)), (1, 1), 'one_preparation_and_one_execution_campaign')
    return executed[0], prepared[0]


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
    spec = dict(Manifest=dict(Provisional=True, IntegratedSourceSHA256='sha256:' + '0' * 64),
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
    module_bytes = Path(case['module']).read_bytes()
    module_sha256 = parity.digest(module_bytes)
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
    execution_root, preparation_root = campaign_roles([
        (path, campaign.canonical_file(path / 'plan.json')) for path in roots])
    roots = [execution_root]
    validated = campaign.authenticate_campaign(roots[0])
    preparation = campaign.authenticate_campaign(preparation_root, complete=False)
    equal(preparation['executions'], [], name + '.zero_guest_preparation')
    equal(preparation['journal'], [], name + '.empty_preparation_journal')
    equal(preparation['plan']['origin'], validated['plan']['origin'], name + '.same_launch_association')
    prepared_artifact = preparation_root / preparation['plan']['prepared_artifact']
    prepared_record = campaign.authenticate_artifact(prepared_artifact)
    bind_module_input(prepared_record, campaign.canonical_file(prepared_artifact / 'manifest.json'),
                      module_sha256, len(module_bytes))
    sealed_origin = authenticate_origin(prepared_artifact, prepared_record, preparation['plan']['origin'])
    execution_prepared = roots[0] / validated['plan']['prepared_artifact']
    execution_prepared_record = campaign.authenticate_artifact(execution_prepared)
    bind_module_input(execution_prepared_record, campaign.canonical_file(execution_prepared / 'manifest.json'),
                      module_sha256, len(module_bytes))
    equal(authenticate_origin(execution_prepared, execution_prepared_record, validated['plan']['origin']),
          sealed_origin, name + '.both_prepared_origins')
    equal(validated['plan']['origin']['origin'], 'intercepted-go', name + '.campaign_origin')
    equal(campaign.seeds(validated['plan']['selection']), selected_seeds(case['seeds']), name + '.actual_selection')
    executions = sorted(validated['executions'], key=lambda row: row['ordinal'])
    equal(len(executions), len(case['outcomes']), name + '.attempted_only')
    rows = []
    guest_stdout, guest_stderr = bytearray(), bytearray()
    for execution, expected in zip(executions, case['outcomes']):
        artifact = roots[0] / execution['artifact']
        record = campaign.authenticate_artifact(artifact)
        equal(authenticate_origin(artifact, record, validated['plan']['origin']), sealed_origin, name + '.execution_origin')
        paths[str(artifact)] = '$ARTIFACT/' + str(execution['ordinal'])
        equal((execution['ordinal'], execution['seed']), (expected['ordinal'], expected['seed']), name + '.selection')
        equal((record['outcome']['class'], record['outcome']['termination']),
              (expected['class'], expected['termination']), name + '.outcome')
        manifest = campaign.canonical_file(artifact / 'manifest.json')
        bind_module_input(record, manifest, module_sha256, len(module_bytes))
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
    validate_cases(inputs, declaration)
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
