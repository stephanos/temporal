#!/usr/bin/env python3
"""Exercise public seed campaigns and independently authenticate retained evidence."""
import argparse
import copy
import fcntl
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import os
from pathlib import Path
import shutil
import shlex
import signal
import subprocess
import sys
import termios
import threading
import time

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
loader = SourceFileLoader('campaign_parity', str(CORPUS.parents[1] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f'{label}: got {actual!r}, expected {expected!r}')


def canonical_file(path):
    raw = path.read_bytes()
    value = parity.decode(raw)
    equal(parity.canonical(value), raw, str(path) + '.canonical')
    return value


def authenticate_artifact(root):
    equal(canonical_file(root / 'storage.json'), {'format': 'gomad.artifact-storage/v2'}, 'artifact.lineage')
    manifest = canonical_file(root / 'manifest.json')
    equal(manifest['schema'], 'gomad.artifact-manifest/v2', 'artifact.schema')
    record = manifest['record']
    parity.record_contract(record, str(root))
    encoded = parity.canonical(record)
    equal(parity.digest(encoded), manifest['record_sha256'], 'artifact.record_sha256')
    listed = {'storage.json', 'manifest.json', *manifest['files']}
    for child in root.rglob('*'):
        if child.is_symlink() or child.is_file() and str(child.relative_to(root)) not in listed:
            raise ValueError(f'unlisted or symlink artifact payload: {child}')
    for name, entry in manifest['files'].items():
        raw = parity.contained(root, name).read_bytes()
        equal((len(raw), parity.digest(raw)), (entry['bytes'], entry['sha256']), name + '.payload')
        equal(entry['module'], name == 'target/module.wasm', name + '.kind')
        if entry['module']:
            equal(entry['sha256'], record['identity']['target'], 'artifact.target')
    equal((root / 'record.json').read_bytes(), encoded, 'artifact.record_bytes')
    for name, stream in record['streams'].items():
        entry = manifest['files'][name]
        equal((stream['bytes'], stream['sha256']), (entry['bytes'], entry['sha256']), name + '.stream')
    return record


def seeds(selection):
    result = []
    for term in selection.split(','):
        endpoints = list(map(int, term.split('-')))
        result.extend(range(endpoints[0], endpoints[-1] + 1))
    equal(len(set(result)), len(result), 'selection.unique')
    return result


def authenticate_campaign(root, complete=True):
    storage = canonical_file(root / 'storage.json')
    equal(storage['format'], 'gomad.campaign-storage/v2', 'campaign.lineage')
    plan = canonical_file(root / 'plan.json')
    equal(storage['plan_sha256'], parity.digest((root / 'plan.json').read_bytes()), 'plan.sha256')
    if plan['schema'] not in ('gomad.seed-campaign-plan/v2', 'gomad.merged-seed-campaign-plan/v2'):
        raise ValueError('unsupported campaign plan schema')
    equal(storage['max_bytes'], plan['journal_bytes'], 'plan.journal_bound')
    selected = seeds(plan['selection'])
    equal(len(selected), plan['selection_count'], 'plan.selection_count')
    prepared = authenticate_artifact(root / plan['prepared_artifact'])
    equal(prepared['identity']['target'], plan['module_sha256'], 'plan.module')
    equal(prepared['identity']['toolchain'], plan['toolchain'], 'plan.toolchain')
    equal(prepared['identity']['configuration'], parity.digest(parity.canonical(plan['configuration'])), 'plan.configuration')
    raw = (root / 'journal.jsonl').read_bytes()
    if len(raw) > storage['max_bytes']:
        raise ValueError('journal exceeds authenticated bound')
    equal(raw.endswith(b'\n') or not raw, True, 'journal.complete_lines')
    live_receipts, live_outcomes = {}, set()
    for line in raw.splitlines():
        event = parity.decode(line)['event']
        if event['kind'] == 'receipt':
            execution = event['execution']
            live_receipts[execution['ordinal']] = execution['artifact']
        elif event['kind'] == 'outcome':
            for execution in event['executions']:
                live_receipts.pop(execution['ordinal'], None)
                if execution['artifact'] is not None:
                    live_outcomes.add(execution['artifact'])
    live_artifacts = live_outcomes | {reference for reference in live_receipts.values() if reference is not None}
    previous = storage['plan_sha256']
    pending, receipts, executions = {}, {}, []
    shard = plan['shard'] or {'index': 0, 'count': 1}
    next_ordinal = shard['index']
    missing = set()
    for interval in plan.get('merge_missing', []):
        start, end = interval['start'], interval['end']
        if not 0 <= start <= end < len(selected) or any(index in missing for index in range(start, end + 1)):
            raise ValueError('invalid merged missing ordinal ranges')
        missing.update(range(start, end + 1))
    if plan['schema'] == 'gomad.merged-seed-campaign-plan/v2':
        parity.sha(plan['plan_sha256'], 'merge.source_plan_sha256')
    stopped = False
    completed = False
    rows = []

    def admit(jobs):
        nonlocal next_ordinal
        if jobs and stopped:
            raise ValueError('admission after failure stop')
        for job in jobs:
            while next_ordinal in missing:
                next_ordinal += shard['count']
            ordinal, seed = job['ordinal'], job['seed']
            equal(ordinal, next_ordinal, 'admission.order')
            equal(seed, selected[ordinal], 'admission.seed')
            if ordinal in pending:
                raise ValueError('duplicate admission')
            pending[ordinal] = seed
            next_ordinal += shard['count']
        if len(pending) > plan['parallel']:
            raise ValueError('admission exceeds slot bound')

    def execution_contract(execution):
        ordinal = execution['ordinal']
        equal(execution['seed'], pending[ordinal], 'execution.seed')
        record = execution['record']
        parity.record_contract(record, 'journal.execution')
        equal(record['seed'], execution['seed'], 'record.seed')
        if plan['scenario'] is None:
            equal(record['identity'], prepared['identity'], 'record.immutable_identity')
            equal(record['limits'], plan['configuration']['limits'], 'record.bounds')
        else:
            equal({key: value for key, value in record['identity'].items() if key != 'configuration'},
                  {key: value for key, value in prepared['identity'].items() if key != 'configuration'},
                  'scenario.record.immutable_identity')
        if execution['artifact'] is not None:
            artifact = root / execution['artifact']
            if not artifact.resolve().is_relative_to(root.resolve()):
                raise ValueError('execution artifact escapes campaign')
            if execution['artifact'] in live_artifacts:
                retained = authenticate_artifact(artifact)
                equal(parity.digest(parity.canonical(dict(retained, seed=0))),
                      parity.digest(parity.canonical(dict(record, seed=0))), 'retained.failure_signature')
                equal(sum(child.stat().st_size for child in artifact.rglob('*') if child.is_file()),
                      execution['stored_bytes'], 'retained.logical_byte_accounting')
                if plan['scenario'] is not None:
                    inputs = canonical_file(artifact / 'evidence/cluster-replay.json')
                    equal(parity.digest(parity.canonical(inputs['configuration'])), record['identity']['configuration'],
                          'scenario.configuration_sha256')
                    equal(inputs['configuration']['limits'], record['limits'], 'scenario.record.bounds')
                    equal(inputs['configuration']['broker']['seed'], execution['seed'], 'scenario.seed')
                    for incarnation in inputs['configuration']['incarnations']:
                        equal(incarnation['module']['sha256'], record['identity']['target'], 'scenario.module_sha256')
                        equal(incarnation['environment']['GOMADSEED'], str(execution['seed']), 'scenario.incarnation_seed')
        return record

    for ordinal, line in enumerate(raw.splitlines()):
        row = parity.decode(line)
        equal(parity.canonical(row), line, 'journal.canonical')
        equal(set(row), {'ordinal', 'previous_sha256', 'sha256', 'event'}, 'journal.fields')
        equal(row['ordinal'], ordinal, 'journal.ordinal')
        equal(row['previous_sha256'], previous, 'journal.previous')
        equal(row['sha256'], parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'})), 'journal.hash')
        previous = row['sha256']
        if completed:
            raise ValueError('event after complete')
        event = row['event']
        kind = event['kind']
        if kind == 'admit':
            admit(event['jobs'])
        elif kind == 'receipt':
            execution = event['execution']
            execution_contract(execution)
            if execution['ordinal'] in receipts:
                raise ValueError('duplicate receipt')
            receipts[execution['ordinal']] = execution
        elif kind == 'outcome':
            for execution in event['executions']:
                execution_contract(execution)
                index = execution['ordinal']
                equal(index, min(pending), 'outcome.order')
                receipt = receipts.pop(index)
                equal(receipt['record'], execution['record'], 'outcome.receipt')
                del pending[index]
                executions.append(execution)
            stopped |= event['stop_reason'] is not None
            admit(event['admitted'])
        elif kind == 'complete':
            equal((pending, receipts), ({}, {}), 'completion.pending')
            completed = True
        else:
            raise ValueError(f'unknown journal event {kind}')
        rows.append(row)
    equal(completed, complete, 'campaign.complete')
    if completed and plan['schema'] == 'gomad.merged-seed-campaign-plan/v2':
        equal({item['ordinal'] for item in executions}, set(range(len(selected))) - missing, 'merge.complete_selection')
    return dict(plan=plan, executions=executions, journal=rows)


def authenticate_report_advertisements(result, root, validated):
    expected = {}
    for execution in validated['executions']:
        if execution['artifact'] is not None:
            path = root / execution['artifact']
            record = authenticate_artifact(path)
            equal(record, execution['record'], 'report.advertised_record')
            expected[str(path)] = parity.digest(parity.canonical(record))
    rows = result.get('artifacts')
    if not isinstance(rows, list):
        raise ValueError('missing advertised retained replay invocations')
    equal((len(rows), result['retained_evidence']), (len(expected), len(expected)), 'report.advertised_count')
    observed = set()
    for row in rows:
        parity.fields(row, dict(path=str, record_hash=str, replay_argv=list, replay_command=str), 'report.artifact')
        path = row['path']
        if path not in expected or path in observed or not Path(path).is_absolute():
            raise ValueError('unexpected/duplicate advertised artifact')
        equal(row['record_hash'], expected[path], 'report.advertised_hash')
        equal(row['replay_argv'], ['gomad', 'replay', path], 'report.advertised_argv')
        equal(shlex.split(row['replay_command']), row['replay_argv'], 'report.advertised_command')
        observed.add(path)
    equal(observed, set(expected), 'report.advertised_complete_set')


def execution_view(validated):
    return [dict(ordinal=item['ordinal'], seed=item['seed'], record=item['record'],
                 record_sha256=parity.digest(parity.canonical(item['record'])),
                 failure_signature=parity.digest(parity.canonical(dict(item['record'], seed=0))))
            for item in validated['executions']]


def rechain(root, change):
    plan = canonical_file(root / 'plan.json')
    change(plan)
    (root / 'plan.json').write_bytes(parity.canonical(plan))
    storage = canonical_file(root / 'storage.json')
    previous = parity.digest(parity.canonical(plan))
    storage['plan_sha256'] = previous
    (root / 'storage.json').write_bytes(parity.canonical(storage))
    rows = []
    for line in (root / 'journal.jsonl').read_bytes().splitlines():
        row = parity.decode(line)
        row['previous_sha256'] = previous
        row['sha256'] = parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'}))
        previous = row['sha256']
        rows.append(parity.canonical(row) + b'\n')
    (root / 'journal.jsonl').write_bytes(b''.join(rows))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-compiler', 'work', 'input'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--preparation-root', type=Path)
    args = parser.parse_args()
    args.binary, args.go_compiler, args.work, args.input = [path.resolve() for path in (args.binary, args.go_compiler, args.work, args.input)]
    args.work.mkdir(parents=True, mode=0o700, exist_ok=False)
    preparation = (args.preparation_root or args.work.parent / 'campaign-preparation').resolve()
    module = CORPUS.parents[1]
    declaration = parity.read(CORPUS / 'campaign-corpus.json')
    equal(declaration['schema'], 'gomad.seed-campaign-corpus/v1', 'corpus.schema')
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    commands = []

    def run(name, command, status=0, extra_environment=None):
        result = subprocess.run(list(map(str, command)), capture_output=True, timeout=600,
                                env=dict(environment, **(extra_environment or {})))
        commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                             expected=status, stdout_sha256=parity.digest(result.stdout), stderr_sha256=parity.digest(result.stderr)))
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        (args.work / 'commands.json').write_bytes(parity.canonical(commands) + b'\n')
        equal(result.returncode, status, name + '.status: ' + result.stderr.decode(errors='replace'))
        return result.stdout

    fixture_root = args.work / 'identity'
    bundle = parity.decode(run('identity', [args.binary, 'fixture-replay-input', '--root', fixture_root, '--input', args.input]))
    parity.authenticate(fixture_root, bundle)
    artifacts = [fixture_root / row['artifact'] for row in bundle['records']]
    artifact = next(path for path in artifacts if parity.read(path / 'manifest.json')['record']['profile'] == 'gomad.wasi-cooperative/v1')
    inputs = canonical_file(artifact / 'evidence/replay.json')
    configuration = inputs['configuration']
    installation = dict(schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
        compiler_sha256=parity.digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
        runtime_root=str(module / 'internal/model/runtimesource'), cache_root=str(preparation),
        profile='gomad.wasi-cooperative/v1', fuel=configuration['limits']['fuel'], memory_bytes=configuration['limits']['memory_bytes'],
        helper_sha256=inputs['helper_sha256'], host_sha256=inputs['host_sha256'],
        runtime_implementation_sha256=inputs['runtime_implementation_sha256'], model=configuration['model'],
        namespace=configuration['namespace'], storage=configuration.get('storage'), network=None, build_tags=[])
    manifest = args.work / 'installation.json'
    manifest.write_bytes(parity.canonical(installation))
    binary = [args.binary, '--installation', manifest]

    def explore_command(case, destination):
        return [*binary, 'explore', '--json', '--choices', '--on-failure=all', '--parallel=1',
            '--seeds=' + declaration['seeds'], '--keep-successes=all', '--success-limit=8', '--success-bytes=256MiB',
            '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', destination,
            '--working-dir', module / case['module'],
            *['--build-tag=' + tag for tag in case['tags']], case['kind'], case['source'], '--', *case['arguments']]

    def campaign_root(destination):
        roots = sorted((destination / 'campaigns').glob('campaign-*'))
        equal(len(roots), 1, 'campaign.roots')
        return roots[0]

    def report_view(raw, root):
        reports = [parity.decode(line) for line in raw.splitlines()]
        results = [row for row in reports if row.get('type') == 'result']
        equal(len(results), 1, 'report.results')
        result = copy.deepcopy(results[0])
        authenticate_report_advertisements(result, root, authenticate_campaign(root))
        del result['artifacts']
        equal(result['campaign_path'], str(root), 'report.campaign_path')
        result['campaign_path'] = '<campaign>'
        for retained in result.get('success_artifacts', []):
            retained['path'] = '<campaign>/' + str(Path(retained['path']).relative_to(root))
        return result

    views = []
    roots = {}
    for repetition in range(2):
        view = {}
        for case in declaration['cases']:
            name = case['name']
            destination = args.work / f'{name}-{repetition}'
            raw = run(f'{name}-{repetition}', explore_command(case, destination), case['status'])
            root = campaign_root(destination)
            validated = authenticate_campaign(root)
            equal([item['seed'] for item in validated['executions']], seeds(declaration['seeds']), name + '.seed_order')
            view[name] = dict(plan=validated['plan'], journal=validated['journal'], records=execution_view(validated), report=report_view(raw, root), status=case['status'])
            roots[name] = root
        views.append(view)
    equal(views[0], views[1], 'fresh repeated campaign projection')

    runtime = authenticate_campaign(roots['runtime'])
    first_features, second_features = [set(row['choice_features']) for row in runtime['executions']]
    if not first_features & second_features or not second_features - first_features:
        raise ValueError('runtime same-target coverage must overlap and add distinct features')
    module_paths = [roots['runtime'] / runtime['plan']['prepared_artifact'] / 'target/module.wasm']
    module_paths += [roots['runtime'] / row['artifact'] / 'target/module.wasm' for row in runtime['executions']]
    equal(len({(path.stat().st_dev, path.stat().st_ino) for path in module_paths}), 1,
          'shared_pool.prepared_and_execution_module_inode')
    if any(path.stat().st_nlink < len(module_paths) for path in module_paths):
        raise ValueError('shared module pool lost retained hardlink ownership')
    novel_destination = args.work / 'novel'
    novel_command = explore_command(declaration['cases'][0], novel_destination)
    novel_command[novel_command.index('--keep-successes=all')] = '--keep-successes=novel'
    novel_command.insert(novel_command.index('--keep-successes=novel'), '--coverage=choice')
    run('novel-overlap', novel_command)
    novel = authenticate_campaign(campaign_root(novel_destination))
    equal([row['seed'] for row in novel['executions'] if row['artifact'] is not None], seeds(declaration['seeds']),
          'novel.same_target_overlapping_distinct_successes')
    budget_destination = args.work / 'retention-count-bound'
    budget_command = explore_command(declaration['cases'][0], budget_destination)
    budget_command[budget_command.index('--success-limit=8')] = '--success-limit=1'
    run('retention-count-bound', budget_command, 3)
    budget = authenticate_campaign(campaign_root(budget_destination), complete=False)
    equal(len(budget['executions']), 1, 'retention_count_bound.committed_prefix')

    failure_case = next(case for case in declaration['cases'] if case['name'] == 'failure')
    for policy, expected_count in [('first', 1), ('budget', 1), ('all', 2)]:
        destination = args.work / ('policy-' + policy)
        command = explore_command(failure_case, destination)
        command[command.index('--on-failure=all')] = '--on-failure=' + policy
        command.insert(command.index('--on-failure=' + policy), '--failure-budget=1')
        run('policy-' + policy, command, 1)
        observed = authenticate_campaign(campaign_root(destination))
        equal(len(observed['executions']), expected_count, 'policy.' + policy + '.ordered_attempts')
    destination = args.work / 'parallel-first'
    command = explore_command(failure_case, destination)
    command[command.index('--on-failure=all')] = '--on-failure=first'
    command[command.index('--parallel=1')] = '--parallel=2'
    run('parallel-first', command, 1)
    observed = authenticate_campaign(campaign_root(destination))
    equal([row['ordinal'] for row in observed['executions']], [0, 1], 'parallel_first.ordered_drain')
    equal(observed['executions'][0]['record']['outcome']['class'], 'target_failure', 'parallel_first.failure')
    destination = args.work / 'watchdog'
    command = explore_command(declaration['cases'][0], destination)
    command[command.index('--execution-timeout=30s')] = '--execution-timeout=1ns'
    command.insert(command.index('--execution-timeout=1ns'), '--terminate-grace=0s')
    run('watchdog', command, 1)
    observed = authenticate_campaign(campaign_root(destination))
    equal([row['record']['outcome']['class'] for row in observed['executions']], ['watchdog', 'watchdog'],
          'watchdog.exact_classification')
    destination = args.work / 'slot-panic'
    run('slot-panic', explore_command(declaration['cases'][0], destination), 3,
        {'GOMAD_FIXTURE_CAMPAIGN_PANIC_SEED': '8'})
    observed = authenticate_campaign(campaign_root(destination), complete=False)
    equal(observed['executions'], [], 'panic.no_fabricated_completion')

    case = declaration['cases'][0]
    interrupted_destination = args.work / 'interrupted'
    run('interrupt', explore_command(case, interrupted_destination), 86,
        {'GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER': '2'})
    interrupted = campaign_root(interrupted_destination)
    prefix = authenticate_campaign(interrupted, complete=False)
    equal(len(prefix['executions']), 1, 'interrupted.complete_prefix')
    # Identity validation must be reached on a resumable prefix, not masked by the
    # completed-campaign refusal; a rechained but unchanged prefix is the control.
    for name, mutate, status, message in [
            ('identity', lambda plan: plan.__setitem__('host_sha256', '0' * 64), 2,
             'campaign implementation identity mismatch'),
            ('identity-control', lambda plan: None, case['status'], None)]:
        root = args.work / ('changed-' + name)
        shutil.copytree(interrupted, root)
        rechain(root, mutate)
        result = run('refuse-' + name if status == 2 else 'resume-' + name, [*binary, 'resume', '--json', root], status)
        if message:
            equal(parity.decode(result)['message'], message, name + '.refusal')
        else:
            equal(execution_view(authenticate_campaign(root)), views[0]['runtime']['records'], name + '.records')
    # fn-232 R1 process-wide deadline backstop, observed externally: a fixture
    # stall inside the second execution's armed window (compile or teardown)
    # must end the whole CLI with the infrastructure status within the wall
    # bound plus bounded cleanup, leave no process behind, keep the committed
    # prefix authenticated, and recover through an ordinary resume. The
    # full-stderr case holds stderr as a full pipe that is never drained, so
    # termination cannot depend on delivering the backstop diagnostic.
    for label, phase, full_stderr in (('compile', 'compile', False), ('teardown', 'teardown', False),
                                      ('teardown-full-stderr', 'teardown', True)):
        destination = args.work / ('backstop-' + label)
        marker = args.work / ('backstop-' + label + '.stalled')
        command = explore_command(case, destination)
        command[command.index('--execution-timeout=30s')] = '--execution-timeout=10s'
        held = None
        if full_stderr:
            held, stderr_end = os.pipe()
            # Fill through a separate open file description so the child's
            # inherited stderr stays a blocking descriptor.
            filler = os.open('/proc/self/fd/' + str(stderr_end), os.O_WRONLY | os.O_NONBLOCK)
            try:
                while True:
                    os.write(filler, b'\0' * 4096)
            except BlockingIOError:
                pass
            os.close(filler)
            equal(os.get_blocking(stderr_end), True, 'backstop.' + label + '.blocking_stderr')
            capacity = fcntl.fcntl(held, 1032)  # F_GETPIPE_SZ
            queued = int.from_bytes(fcntl.ioctl(held, termios.FIONREAD, b'\0' * 4), sys.byteorder)
            equal(queued, capacity, 'backstop.' + label + '.stderr_full')
        child = subprocess.Popen(list(map(str, command)), stdout=subprocess.PIPE,
            stderr=stderr_end if full_stderr else subprocess.PIPE,
            env=dict(environment, GOMAD_FIXTURE_BACKSTOP_STALL=phase, GOMAD_FIXTURE_BACKSTOP_STALL_WINDOW='1',
                     GOMAD_FIXTURE_BACKSTOP_STALL_FILE=str(marker)), start_new_session=True)
        if full_stderr:
            os.close(stderr_end)
        # Elapsed time is measured on this process's monotonic clock from the
        # first observation of the stall marker to the observed exit; marker
        # file timestamps come from another clock (the artifact filesystem) and
        # are not comparable with it.
        drained = {}
        drains = [threading.Thread(target=lambda name=name, stream=stream: drained.__setitem__(name, stream.read()))
                  for name, stream in (('stdout', child.stdout), ('stderr', child.stderr)) if stream is not None]
        for drain in drains:
            drain.start()
        poll = 0.02
        limit = time.monotonic() + 120
        stall_seen = None
        while child.poll() is None and time.monotonic() < limit:
            if stall_seen is None and marker.exists():
                stall_seen = time.monotonic()
            time.sleep(poll)
        if child.poll() is None:
            os.killpg(child.pid, signal.SIGKILL)
            child.wait()
            raise ValueError('backstop.' + label + '.bounded_exit: no exit within 120 s; process group killed')
        exited = time.monotonic()
        for drain in drains:
            drain.join()
        stdout, stderr = drained['stdout'], drained.get('stderr', b'')
        equal(stall_seen is not None, True, 'backstop.' + label + '.stall_observed_before_exit')
        if full_stderr:
            # Still never drained: the pipe stays full until the reader closes.
            queued = int.from_bytes(fcntl.ioctl(held, termios.FIONREAD, b'\0' * 4), sys.byteorder)
            equal(queued, capacity, 'backstop.' + label + '.stderr_undrained')
            os.close(held)
            stderr = b''
        (args.work / ('backstop-' + label + '.stdout')).write_bytes(stdout)
        (args.work / ('backstop-' + label + '.stderr')).write_bytes(stderr)
        commands.append(dict(name='backstop-' + label, argv=list(map(str, command)), status=child.returncode, expected=3,
            stdout_sha256=parity.digest(stdout), stderr_sha256=parity.digest(stderr)))
        (args.work / 'commands.json').write_bytes(parity.canonical(commands) + b'\n')
        equal(child.returncode, 3, 'backstop.' + label + '.status: ' + stderr.decode(errors='replace'))
        equal(marker.read_text(), phase + ' 1\n', 'backstop.' + label + '.stalled_window')
        if not full_stderr:
            equal(b'deadline backstop ended the process' in stderr, True, 'backstop.' + label + '.message')
        # Armed for the 10 s wall plus 5 s cleanup grace; the stall itself never
        # returns, so the exit follows the stall by at most that bound.
        # Both observations may lag their events by one poll interval.
        stalled = exited - stall_seen
        commands[-1]['seconds_after_stall'] = round(stalled, 3)
        (args.work / 'commands.json').write_bytes(parity.canonical(commands) + b'\n')
        equal(0 <= stalled < 15 + 5, True, 'backstop.' + label + '.bounded_exit: ' + str(stalled))
        if phase == 'compile':
            equal(stalled >= 15 - 1, True, 'backstop.compile.waited_for_cleanup_grace: ' + str(stalled))
        try:
            os.killpg(child.pid, 0)
            survived = True
        except ProcessLookupError:
            survived = False
        equal(survived, False, 'backstop.' + label + '.no_surviving_process')
        stopped = campaign_root(destination)
        committed = authenticate_campaign(stopped, complete=False)
        equal([row['seed'] for row in committed['executions']], [views[0]['runtime']['records'][0]['seed']],
              'backstop.' + label + '.committed_prefix')
        run('backstop-' + label + '-resume', [*binary, 'resume', '--json', stopped], case['status'])
        # The plan keeps its 2 s bound, so compare selection and outcomes.
        outcomes = lambda rows: [(row['seed'], row['record']['outcome']) for row in rows]
        equal(outcomes(execution_view(authenticate_campaign(stopped))), outcomes(views[0]['runtime']['records']),
              'backstop.' + label + '.resumed_outcomes')
    malformed_tail = args.work / 'noncanonical-tail'
    shutil.copytree(interrupted, malformed_tail)
    with (malformed_tail / 'journal.jsonl').open('ab') as stream:
        stream.write(b'{"ordinal":')
    malformed_before = (malformed_tail / 'journal.jsonl').read_bytes()
    refusal = run('refuse-noncanonical-tail', [*binary, 'resume', '--json', malformed_tail], 2)
    equal(parity.decode(refusal)['message'], 'campaign journal has a malformed final record', 'noncanonical_tail.refusal')
    equal((malformed_tail / 'journal.jsonl').read_bytes(), malformed_before, 'noncanonical_tail.unchanged')
    with (interrupted / 'journal.jsonl').open('ab') as stream:
        stream.write(b'{"event":')
    raw = run('resume', [*binary, 'resume', '--json', interrupted], case['status'])
    resumed = authenticate_campaign(interrupted)
    equal(execution_view(resumed), views[0]['runtime']['records'], 'resume.records')
    equal(report_view(raw, interrupted), views[0]['runtime']['report'], 'resume.report')
    before = (interrupted / 'journal.jsonl').read_bytes()
    run('resume-again', [*binary, 'resume', '--json', interrupted], 2)
    equal((interrupted / 'journal.jsonl').read_bytes(), before, 'resume.idempotent')

    retiring_destination = args.work / 'retiring-interruption'
    retiring_command = explore_command(case, retiring_destination)
    retiring_command[retiring_command.index('--seeds=' + declaration['seeds'])] = '--seeds=1-5'
    retiring_command[retiring_command.index('--keep-successes=all')] = '--keep-successes=novel'
    retiring_command.insert(retiring_command.index('--keep-successes=novel'), '--coverage=choice')
    run('retire-interrupt', retiring_command, 86, {'GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER': '10'})
    retiring = campaign_root(retiring_destination)
    retired_prefix = authenticate_campaign(retiring, complete=False)
    references = [row['artifact'] for row in retired_prefix['executions']]
    equal(len(references), 5, 'retirement.committed_successes')
    equal(sum(reference is not None for reference in references), 4, 'retirement.retained_novel_successes')
    equal(references[-1], None, 'retirement.durable_non_novel_discard')
    equal(len(list((retiring / 'evidence').glob('*/manifest.json'))), 5, 'retirement.crash_gap_candidate')
    run('retire-resume', [*binary, 'resume', '--json', retiring])
    retired = authenticate_campaign(retiring)
    equal([row['artifact'] for row in retired['executions']], references, 'retirement.completed_references')
    equal(len(list((retiring / 'evidence').glob('*/manifest.json'))), 4, 'retirement.reconciled_candidates')
    equal(list((retiring / 'evidence').glob('.retire-*')), [], 'retirement.no_partial_tombstone')
    prepared_module = retiring / retired['plan']['prepared_artifact'] / 'target/module.wasm'
    equal(parity.digest(prepared_module.read_bytes()), retired['plan']['module_sha256'], 'retirement.prepared_module_preserved')

    source = roots['runtime']
    shards = []
    shard_records = []
    for index in range(2):
        destination = args.work / f'shard-{index}'
        run(f'shard-{index}', [*binary, 'execute-shard', '--json', '--shard', f'{index}/2', '--artifacts', destination, source / 'plan.json'])
        root = campaign_root(destination)
        shards.append(root)
        validated = authenticate_campaign(root)
        equal(validated['plan']['shard'], {'index': index, 'count': 2}, 'shard.assignment')
        shard_records.extend(execution_view(validated))
    equal(sorted(shard_records, key=lambda item: item['ordinal']), views[0]['runtime']['records'], 'shards.unsharded_records')
    merged_root = args.work / 'merged'
    run('merge', [*binary, 'merge', '--json', '--output', merged_root, source / 'plan.json', *shards])
    merged = authenticate_campaign(merged_root)
    equal(merged['plan']['plan_sha256'], parity.digest((source / 'plan.json').read_bytes()), 'merge.original_plan')
    equal(execution_view(merged), views[0]['runtime']['records'], 'merge.unsharded_records')
    run('merge-missing', [*binary, 'merge', '--json', '--output', args.work / 'missing', source / 'plan.json', shards[0]], 2)
    partial_root = args.work / 'partial'
    run('merge-partial', [*binary, 'merge', '--json', '--partial', '--output', partial_root, source / 'plan.json', shards[0]])
    partial = authenticate_campaign(partial_root)
    equal([item['ordinal'] for item in partial['executions']], [0], 'merge.partial_ordinals')
    run('merge-duplicate', [*binary, 'merge', '--json', '--output', args.work / 'duplicate', source / 'plan.json', shards[0], shards[0]], 2)

    root = args.work / 'changed-bounds'
    shutil.copytree(source, root)
    rechain(root, lambda plan: plan['configuration']['limits'].__setitem__('fuel', plan['configuration']['limits']['fuel'] + 1))
    refusal = run('refuse-bounds', [*binary, 'resume', '--json', root], 2)
    equal(parity.decode(refusal)['message'], 'campaign record differs from its prepared plan', 'bounds.refusal')
    corrupt = args.work / 'corrupt-journal'
    shutil.copytree(source, corrupt)
    data = (corrupt / 'journal.jsonl').read_bytes()
    first, remainder = data.split(b'\n', 1)
    row = parity.decode(first)
    row['sha256'] = '0' * 64
    (corrupt / 'journal.jsonl').write_bytes(parity.canonical(row) + b'\n' + remainder)
    run('refuse-journal', [*binary, 'resume', '--json', corrupt], 2)
    legacy = args.work / 'go-lineage'
    legacy.mkdir()
    (legacy / 'storage.json').write_bytes(parity.canonical({'format': 'gomad.campaign-storage/v1'}))
    run('refuse-go-lineage', [*binary, 'resume', '--json', legacy], 2)
    invalid = [[], ['--count=0'], ['--seeds=7', '--count=2'], ['--parallel=0'],
        ['--choice-bytes=1'], ['--coverage=semantic'], ['--guide'], ['--toolchain-root=/tmp'],
        ['--env=GOMADSEED=7'], ['--env=LD_PRELOAD=x'], ['--env=A=1', '--env=A=2'], ['--env=9BAD=x'],
        ['--strategy=simulation-exploration'], ['--max-executions=2'], ['--build-tag=test_dep', 'go-test', './io_failure', '-test.v']]
    for index, flags in enumerate(invalid):
        target = [] if not flags or 'go-test' in flags else ['go-run', './internal/backends/wasm/testdata/runtime_probes']
        run(f'grammar-{index}', [*binary, 'explore', '--json', *flags, *target], 2)
    view = dict(schema='gomad.seed-campaign-golden/v1', cases=views[0])
    golden = CORPUS / declaration['golden']
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    parity.golden(golden, view, regenerate)
    first = golden.read_bytes()
    parity.golden(golden, dict(schema=view['schema'], cases=views[1]), regenerate)
    equal(golden.read_bytes(), first, 'second regeneration bytes')
    print('campaign: public preparation, repeated authenticated records, resume, shards, merge and refusals passed')


if __name__ == '__main__':
    main()
