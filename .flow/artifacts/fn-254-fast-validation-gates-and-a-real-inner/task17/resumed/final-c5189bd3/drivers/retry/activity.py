"""Prepared ordinary Activity quick renewal, gated on the accepted final standalone proof."""
from pathlib import Path
import argparse
import collections
import hashlib
import json
import re
import signal
import subprocess
from proof import ROOT, Run, interrupt, sha, snapshot, write

PACKAGES = ['umpire-temporal', 'umpire-temporal-api', 'umpire-temporal-guest',
            'umpire-temporal-realization-macros', 'umpire-temporal-tests']
REFERENCE = '.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/' \
            'semantic-renewal/activity-equivalence-final.json'


def cache_state(cache):
    return {'root': str(cache), 'exists': cache.exists(), 'files': {
        str(path.relative_to(cache)): sha(path) for path in sorted(cache.rglob('*'))
        if path.is_file() and not path.is_symlink()}}


def validate(run, name):
    stdout = (run.out/(name+'.stdout.log')).read_text()
    stderr = (run.out/(name+'.stderr.log')).read_text()
    selection = json.loads(next(line for line in stdout.splitlines() if line.startswith('{"base":')))
    assert [package for workspace in selection['workspaces'] for package in workspace['packages']] == PACKAGES
    assert any(str(path).endswith('/model/temporal/feature/activity/model.rs') for path in selection['changed'])
    expected = {tuple(row) for row in json.loads((run.root/REFERENCE).read_text())['actual_identity_set']}
    assert len(expected) == 148 and not any(row[-1] for row in expected)
    test_output = stderr.split('quick: stage=doctest')[0]
    names = collections.Counter(re.findall(r'^\s*test (.+) \.\.\. ok$', test_output, re.M))
    wanted = collections.Counter(row[4] for row in expected)
    assert not wanted-names, 'selected test identity missing from actual successful output'
    extra = names-wanted
    assert sum(extra.values()) == 25
    assert sum(path.startswith('tests/ui/') for path in extra) == 23
    assert sum(path.startswith('tests/ui-pass/') for path in extra) == 2
    assert re.search(r'Summary.*148 tests run: 148 passed.*40 skipped', stderr)
    docs_output = stderr.split('quick: stage=doctest', 1)[1]
    docs = re.findall(r'test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored;',
                      stdout+'\n'+docs_output)
    assert len(docs) == 3 and all(tuple(map(int, row)) == (0, 0, 0) for row in docs)
    hits = [line.strip() for line in stderr.splitlines() if 'TLC cache hit Activity ' in line]
    misses = [line.strip() for line in stderr.splitlines() if 'TLC cache miss Activity ' in line]
    execution = [line.strip() for line in stderr.splitlines() if 'TLC execute Activity ' in line]
    result = {'selection': selection, 'selected_tests': 148, 'selected_ignored': 0,
        'filtered_out_tests': 40, 'nested_negative_controls': 23, 'nested_positive_controls': 2,
        'doctest_targets': 3, 'doctest_passed': 0, 'doctest_ignored': 0,
        'cache_hits': hits, 'cache_misses': misses, 'model_execution_labels': execution,
        'stages': re.findall(r'quick: stage=(\S+).*?elapsed=([\d.]+)s exit=(\S+)', stderr),
        'reference_path': REFERENCE, 'reference_sha256': sha(run.root/REFERENCE),
        'identity_set': sorted(expected), 'nested_ui_successes': dict(extra),
        'native_interpreter_activity': [line.strip() for line in stderr.splitlines()
                                        if 'quick native/interpreter agreement' in line]}
    assert result['native_interpreter_activity'], 'native/interpreter agreement evidence missing'
    write(run.out/(name+'-validation.json'), result)
    return result


def activity(options):
    accepted = json.loads(options.standalone_validation.read_text())
    assert accepted['accepted'] and accepted['head'] == options.expected_head
    assert accepted['cleanup']['all_observed_owned_pids_absent']
    source_after = options.standalone_validation.parent/'source-after.json'
    assert sha(source_after) == accepted['source_after_sha256']
    assert snapshot(options.source_root.resolve()) == json.loads(source_after.read_text())
    run = Run(options)
    run.env.update(CARGO_PROFILE_DEV_DEBUG='0', CARGO_PROFILE_TEST_DEBUG='0',
        UMPIRE_QUICK_CACHE='1', UMPIRE_CACHE_DISABLE='0', NEXTEST_PROFILE='required',
        NEXTEST_TEST_THREADS='4', NO_COLOR='1')
    assert not any(key.startswith('UMPIRE_GATE_') for key in run.env)
    write(run.out/'ordinary-quick-environment.json', {key:run.env.get(key) for key in [
        'UMPIRE_GATE_RECEIPTS', 'UMPIRE_GATE_SCOPE', 'UMPIRE_GATE_STAGE', 'UMPIRE_GATE_ORDER',
        'UMPIRE_COMPLETION_FD', 'UMPIRE_CANCELLATION_FD', 'UMPIRE_TLC_CONFIGURATION',
        'UMPIRE_QUICK_CACHE', 'UMPIRE_CACHE_DISABLE']})
    write(run.out/'baseline-git-status.json', {'porcelain': subprocess.check_output(
        ['git', 'status', '--porcelain'], cwd=run.root, text=True),
        'policy': 'Preserve root documentary deltas; source hashes are checked separately.'})
    source = run.root/'model/temporal/feature/activity/model.rs'
    original = source.read_bytes()
    edited = original.replace(b'//! Bounded activity retries, optional deadlines and cooperative cancellation.',
        b'//! Bounded activity retries, optional deadlines and cooperative cancellation (warm consumers).', 1)
    assert edited != original
    cache = run.target/'quick-cache'
    before = cache_state(cache)
    write(run.out/'quick-cache-before.json', before)
    compiler_warm = bool(list((run.target/'debug/deps').glob('libumpire_temporal-*.rlib')))
    setup_cold = not compiler_warm or not before['files']
    write(run.out/'initial-warmth.json', {'temporal_rlib_exists': compiler_warm,
        'cache_files': len(before['files']), 'setup_cold': setup_cold})
    try:
        run.run('native-system-bootstrap', ['mise','exec','--','cargo','build','--locked',
            '-p','temporal-xtask','--bin','temporal-xtask','--profile','dev'], run.root/'model', timeout=900)
        source.write_bytes(edited)
        run.overrides[str(source.relative_to(run.root))] = hashlib.sha256(edited).hexdigest()
        command = [str(run.target/'debug/temporal-xtask'), 'quick']
        first = run.run('activity-first', command, run.root/'model', timeout=1800, observe=True)
        first_checks = validate(run, 'activity-first')
        after_first = cache_state(cache)
        write(run.out/'quick-cache-after-first.json', after_first)
        warm = None
        if setup_cold:
            warm = run.run('activity-warm', command, run.root/'model', timeout=1800, observe=True)
            warm_checks = validate(run, 'activity-warm')
            assert not warm_checks['cache_misses'] and len(warm_checks['cache_hits']) == 14
            assert not warm_checks['model_execution_labels']
            assert cache_state(cache) == after_first, 'warm cache bytes changed unexpectedly'
        chosen = warm if warm is not None else first
        write(run.out/'summary.json', {'head': options.expected_head, 'setup_cold': setup_cold,
            'first_seconds': first['seconds'], 'warm_seconds': None if warm is None else warm['seconds'],
            'current_measurement_seconds': chosen['seconds'], 'soft_target_seconds': 60,
            'target_met': chosen['seconds'] <= 60, 'packages': PACKAGES, 'tests': 148,
            'negative_compiler_controls': 23, 'positive_compiler_controls': 2,
            'standalone_validation': str(options.standalone_validation),
            'standalone_validation_sha256': sha(options.standalone_validation),
            'historical_warm_seconds': 321.29908240299847,
            'comparison': 'Current final candidate; compile/cache state and host load disclosed. No isolated causal speedup claim.',
            'first_cache_hit_count': len(first_checks['cache_hits']),
            'ordinary_quick_has_no_gate_observation_scope': True, 'macos': 'owner-deferred'})
    finally:
        try:
            run.observer.finish()
        finally:
            source.write_bytes(original)
            run.overrides.clear()
            write(run.out/'model-restoration.json', {'path': str(source.relative_to(run.root)),
                'original_sha256': hashlib.sha256(original).hexdigest(),
                'edited_sha256': hashlib.sha256(edited).hexdigest(), 'restored_sha256': sha(source),
                'restored': source.read_bytes() == original})
            write(run.out/'quick-cache-after.json', cache_state(cache))
            write(run.out/'source-after.json', snapshot(run.root))
            run.source()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-root', type=Path, default=ROOT)
    parser.add_argument('--expected-head', required=True)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--standalone-validation', required=True, type=Path)
    parser.add_argument('--execute', action='store_true')
    options = parser.parse_args()
    assert re.fullmatch('[0-9a-f]{40}', options.expected_head)
    if not options.execute:
        print('Prepared only; no Activity process, build or Model mutation requested.')
        return
    signal.signal(signal.SIGTERM, interrupt)
    activity(options)


if __name__ == '__main__':
    main()
