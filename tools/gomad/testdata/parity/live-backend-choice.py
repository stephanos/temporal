#!/usr/bin/env python3
"""Execute transferred roots 8-11 on the original branching Go guest."""
from pathlib import Path
import json
import runpy

common = runpy.run_path(str(Path(__file__).with_name('live-backend-common.py')))
Lane, equal = common['Lane'], common['equal']


def events(lane, root):
    return lane.authenticate(root, choice=True)['events']


def ordinals(lane, root):
    rows = [row['ordinal'] for event in events(lane, root) for row in event['executions']]
    equal(rows, list(range(len(rows))), 'logical_ordinals.exactly_once')
    return rows


def main():
    lane = Lane(__doc__)
    installation = lane.installation('choice')
    source = lane.guest('branching', 'explorationGuest')
    target = ['go-run', '.']

    def explore(name, executions=16, depth=4, capacity=8 << 20, extra=None, status=(0, 1)):
        return lane.explore(name, target, source, [
            '--build-tag=test_dep', '--max-executions=' + str(executions),
            '--max-choice-depth=' + str(depth), '--max-exploration-bytes=' + str(capacity)],
            strategy='choice-exploration', parallel=2, overall='300s', choices=True, keep='all',
            installation=installation, extra=extra, status=status)

    baseline, output = explore('canonical-first')
    first = lane.report(output)
    checked = lane.authenticate(baseline, choice=True)
    first_events = checked['events']
    executions = [row for event in first_events for row in event['executions']]
    equal(any(row['record']['outcome']['class'] == 'success' for row in executions), True, 'success_discovered')
    equal(any(row['record']['outcome']['class'] == 'target_failure' for row in executions), True, 'failure_discovered')
    equal(all(row['divergence'] is None for row in executions), True, 'no_divergence')
    equal(all(row['artifact'] is not None for row in executions), True, 'all_successes_retained')
    equal(first['envelope']['deepest_prefix'] >= 2, True, 'branching_depth')
    repeated, output = explore('canonical-second')
    equal(lane.report(output)['envelope'], first['envelope'], 'canonical_envelope')
    equal(events(lane, repeated), first_events, 'canonical_rounds')
    ordinals(lane, baseline)
    for name, executions_bound, depth, capacity, reason in [
        ('execution-bound', 1, 4, 8 << 20, 'max_executions'),
        ('depth-bound', 16, 1, 8 << 20, 'choice_depth_complete'),
        ('byte-bound', 16, 4, len(common['encode'](dict(sha256=first_events[0]['segment']['results'][0]['candidate_sha256'], forced_depth=0))) + 1, 'exploration_capacity')]:
        root, output = explore(name, executions_bound, depth, capacity)
        report = lane.report(output)
        envelope = report['envelope']
        equal(envelope['stop_reason'], reason, name + '.stop_reason')
        equal(envelope['omitted_by_' + {'execution-bound': 'execution_bound', 'depth-bound': 'depth',
                                       'byte-bound': 'capacity'}[name]] > 0, True, name + '.omissions')
        if name == 'byte-bound':
            expected = dict(max_exploration_bytes=capacity, pending=0, pending_bytes=0, seen_prefixes=1,
                            logical_executions=1, committed_rounds=1, deepest_prefix=0, bounded_complete=False)
            equal({key: envelope[key] for key in expected}, expected, 'capacity.closed_envelope')
            equal(events(lane, root)[0]['segment']['results'], first_events[0]['segment']['results'], 'capacity.canonical_root')
            equal(ordinals(lane, root), [0], 'capacity.one_execution')
        elif name == 'execution-bound':
            equal(report['attempted'], 1, 'execution_bound.one')
        else:
            equal(envelope['deepest_prefix'], 1, 'depth_bound.one')
    source.rename(source.with_name(source.name + '-unavailable'))
    # The retained executor must not touch preparation on replay.
    unavailable = json.loads(installation.read_bytes())
    unavailable['compiler_path'] = str(lane.args.work / 'unavailable-go')
    unavailable['runtime_root'] = str(lane.args.work / 'unavailable-runtime')
    unavailable['cache_root'] = str(lane.args.work / 'unavailable-cache')
    detached = lane.args.work / 'detached-installation.json'
    detached.write_bytes(common['encode'](unavailable))
    detached.chmod(0o600)
    lane.args.installation = detached
    for index, row in enumerate(executions):
        lane.replay('detached-' + str(index), baseline / row['artifact'],
                    failure=row['record']['outcome']['class'] != 'success')
    # A mismatched profile must refuse before any minimizer control runs.
    import shutil
    reseal = runpy.run_path(str(Path(__file__).with_name('live-backend-observed.py')))['reseal']
    incompatible = lane.args.work / 'incompatible-profile'
    parent = next(baseline / row['artifact'] for row in executions if row['record']['outcome']['class'] == 'target_failure')
    shutil.copytree(parent, incompatible)
    inputs = json.loads((incompatible / 'evidence/replay.json').read_bytes())
    inputs['configuration']['profile'] = 'gomad.wasi-stock-exploratory/v1'
    reseal(incompatible, 'evidence/replay.json', common['encode'](inputs))
    controls = lane.args.work / 'incompatible-controls'
    refusal = lane.run('incompatible-minimize', ['minimize', '--json', '--attempt-budget=8',
        '--artifacts', lane.args.work / 'incompatible-minimized', incompatible], (2,),
        extra={'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(controls)})
    equal(b'identity mismatch' in refusal.stderr or b'profile mismatch' in refusal.stderr,
          True, 'minimize.incompatible_profile')
    equal(controls.exists(), False, 'minimize.refused_before_controls')
    lane.roots.append(dict(root=8, original='TestWASMChoiceCampaignCanonicalRoundsAndDetachedReplay'))
    # Re-enable the original source solely for the independent interruption cell.
    source.with_name(source.name + '-unavailable').rename(source)
    recovery_log = lane.args.work / 'recovery-executions.jsonl'
    partial, output = explore('inside-second-round', extra={'GOMAD_FIXTURE_BACKEND_CANCEL': 'choice-second',
        'GOMAD_FIXTURE_BACKEND_EXECUTIONS': str(recovery_log)}, status=(3,))
    equal(json.loads(output.stdout.splitlines()[-1])['classification'], 'cancelled', 'interruption.classification')
    equal(len(ordinals(lane, partial)), 1, 'interruption.one_committed')
    lane.run('recover', ['recover', partial])
    resumed = lane.run('resume', ['resume', '--json', partial], (0, 1),
        extra={'GOMAD_FIXTURE_BACKEND_EXECUTIONS': str(recovery_log)})
    equal(events(lane, partial), first_events, 'recovery.canonical_rounds')
    equal(ordinals(lane, partial), list(range(len(executions))), 'recovery.logical_executions')
    equal([int(line) for line in recovery_log.read_text().splitlines()].count(1) >= 2, True,
          'recovery.nonzero_reexecuted_uncommitted_work')
    before = common['rounds']['snapshot'](partial)
    started = lane.args.work / 'completed-resume-guest-started'
    refusal = lane.run('completed-resume-refused', ['resume', '--json', partial], (2,),
        extra={'GOMAD_FIXTURE_GUEST_START_FILE': str(started)})
    equal(json.loads(refusal.stdout)['classification'], 'invalid_input', 'completed_resume.classification')
    equal('complete' in json.loads(refusal.stdout)['message'], True, 'completed_resume.reason')
    equal(started.exists(), False, 'completed_resume.zero_guest_starts')
    equal(common['rounds']['snapshot'](partial), before, 'completed_resume.immutable')
    lane.roots.append(dict(root=9, original='TestWASMChoiceCampaignInterruptedRoundRecoveryNoDuplicates'))

    cpu = lane.guest('fuel', 'explorationGuest', (
        'if a == 1 && b == 1 { os.Exit(17) }',
        'if os.Getenv("CPU_LOOP_EXIT") == "yes" { os.Exit(0) }; for {}'))
    fuel = lane.installation('fuel', fuel=100000000)
    root, output = lane.explore('fuel-capacity', target, cpu,
        ['--build-tag=test_dep', '--max-executions=16', '--max-choice-depth=4', '--max-exploration-bytes=8388608'],
        strategy='choice-exploration', parallel=2, overall='300s', choices=True, keep='all',
        installation=fuel, status=(3,))
    error = json.loads(output.stdout.splitlines()[-1])
    equal('fuel' in error['message'].lower(), True, 'fuel.reason')
    equal('nonzero_exit' in error['message'], False, 'fuel.not_application_failure')
    equal(events(lane, root), [], 'fuel.no_committed_rounds')
    equal(list(root.glob('evidence/*/manifest.json')), [], 'fuel.no_artifacts')
    lane.roots.append(dict(root=10, original='TestWASMChoiceCampaignFuelCapacityIsNotApplicationFailure'))

    environment = lane.guest('environment', 'explorationGuest', (
        'if a == 1 && b == 1 { os.Exit(17) }',
        'if os.Getenv("GOMAD_ENV_PROBE") != "set" { os.Exit(3) }'))
    root, output = lane.explore('explicit-environment', target, environment,
        ['--build-tag=test_dep', '--env=GOMAD_ENV_PROBE=set'], installation=installation,
        choices=True, keep='all', overall='300s', status=(0,))
    equal((lane.report(output)['attempted'], lane.report(output)['succeeded']), (1, 1), 'explicit_environment.success')
    lane.authenticate(root)
    lane.roots.append(dict(root=11, original='TestWASMCooperativeCampaignWithExplicitEnvironment'))
    lane.finish()


if __name__ == '__main__':
    main()
