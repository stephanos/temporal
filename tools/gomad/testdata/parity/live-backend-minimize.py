#!/usr/bin/env python3
"""Execute root 12: actual essential-prefix guest, detached reduction and recovery."""
import json
from pathlib import Path
import runpy

common = runpy.run_path(str(Path(__file__).with_name('live-backend-common.py')))
Lane, equal = common['Lane'], common['equal']


def main():
    lane = Lane(__doc__)
    installation = lane.installation('essential-prefix')
    source = lane.guest('essential-prefix', 'minimizationGuest')
    target = ['go-run', '.']
    root, output = lane.explore('unforced-seeds', target, source, ['--build-tag=test_dep'],
        seeds='0-7', choices=True, keep='all', installation=installation, overall='300s')
    executions = lane.authenticate(root)['executions']
    successes = [row for row in executions if row['record']['outcome']['class'] == 'success']
    equal(bool(successes), True, 'unforced.success_exists')
    seed = successes[0]['seed']
    campaign, output = lane.explore('essential-prefix-discovery', target, source,
        ['--build-tag=test_dep', '--max-executions=16', '--max-choice-depth=4', '--max-exploration-bytes=8388608'],
        seeds=str(seed), strategy='choice-exploration', choices=True, keep='all',
        installation=installation, overall='300s')
    checked = lane.authenticate(campaign, choice=True)
    rows = [row for event in checked['events'] for row in event['executions']]
    equal(any(row['record']['outcome']['class'] == 'success' for row in rows), True, 'discovery.success')
    failures = [row for row in rows if row['record']['outcome']['class'] == 'target_failure']
    equal(bool(failures), True, 'discovery.failure')
    artifact = campaign / failures[0]['artifact']
    original = common['seed']['authenticate_artifact'](artifact)
    details = json.loads((artifact / 'evidence/inspection.json').read_bytes())
    equal(details['choices']['records'] >= 3, True, 'redundant.trailing_choices')
    parent_snapshot = common['rounds']['snapshot'](artifact)
    source.rename(source.with_name(source.name + '-unavailable'))
    selected = json.loads(installation.read_bytes())
    cache = Path(selected['cache_root'])
    if cache.exists():
        cache.rename(cache.with_name(cache.name + '-unavailable'))
    selected.update(compiler_path=str(lane.args.work / 'unavailable-go'),
                    runtime_root=str(lane.args.work / 'unavailable-runtime'))
    detached = lane.args.work / 'detached-installation.json'
    detached.write_bytes(common['encode'](selected))
    detached.chmod(0o600)
    lane.args.installation = detached

    def minimize(name, root, extra=None, resume=False, statuses=(0,)):
        return lane.run(name, ['minimize', '--json', '--attempt-budget=64', '--artifacts', root,
            *(['--resume'] if resume else []), artifact], statuses, extra=extra)

    controls = lane.args.work / 'baseline-controls.jsonl'
    result = json.loads(minimize('baseline', lane.args.work / 'baseline',
        {'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(controls)}).stdout)
    equal(result['changed'], True, 'minimize.changed')
    equal(bool(result['accepted']), True, 'minimize.accepted')
    final = Path(result['artifact_path'])
    reduced = common['seed']['authenticate_artifact'](final)
    metadata = json.loads((final / 'evidence/inspection.json').read_bytes())['minimization']
    equal(0 < metadata['final_forced_decisions'] < metadata['original_forced_decisions'], True,
          'minimize.essential_prefix_preserved')
    equal(metadata['schema'], 'gomad.minimization-evidence/v1', 'minimize.schema')
    equal(metadata['predicate']['simulation_replay'], 'not_present', 'minimize.no_simulation')
    equal(metadata['predicate']['failure_signature'],
          common['digest'](common['encode'](dict(original, seed=0))), 'minimize.same_failure_signature')
    equal(all(row['kind'] == 'schedule_suffix' for row in result['accepted']), True,
          'minimize.only_suffix_reductions')
    original_behavior = json.loads((final / 'minimization/original-record.json').read_bytes())
    equal(original_behavior, original, 'minimize.failure_signature')
    for name in ('target/provenance.json', 'evidence/replay.json', 'evidence/transcript'):
        equal((final / name).read_bytes(), (artifact / name).read_bytes(), 'minimize.fixed_' + name)
    sequence = [int(line) for line in controls.read_text().splitlines()]
    equal(bool(sequence) and sequence[0] == 0, True, 'minimize.first_no_forcing_rejected')
    lane.replay('minimized-detached-replay', final)
    for mode in ('repeat', 'after accepted', 'after publication'):
        name = mode.replace(' ', '-')
        destination = lane.args.work / name
        control = lane.args.work / (name + '-controls.jsonl')
        extra = {'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(control)}
        if mode != 'repeat':
            extra['GOMAD_FIXTURE_MINIMIZE_INTERRUPT'] = mode
        trial = minimize(name, destination, extra, statuses=(0,) if mode == 'repeat' else (3,))
        if mode != 'repeat':
            equal(('interrupted ' + mode).encode() in trial.stderr, True, name + '.interruption_timing')
            trial = minimize(name + '-resume', destination,
                {'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(control)}, resume=True)
        repeated = json.loads(trial.stdout)
        equal((repeated['attempts'], repeated['record_hash'], repeated['accepted']),
              (result['attempts'], result['record_hash'], result['accepted']), name + '.identical_reduction')
        if mode != 'after publication':
            equal([int(line) for line in control.read_text().splitlines()], sequence,
                  name + '.no_repeated_committed_attempts')
    equal(common['rounds']['snapshot'](artifact), parent_snapshot, 'parent.immutable')
    lane.roots.append(dict(root=12, original='TestWASMMinimizeEssentialPrefixDetachedReplayRepeatAndResume',
                           variants=['repeat', 'after accepted', 'after publication']))
    lane.finish()


if __name__ == '__main__':
    main()
