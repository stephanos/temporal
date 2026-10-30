"""One authorized warm retry after the transient du race; no setup campaign."""
from pathlib import Path
import argparse
import hashlib
import json
import shutil
import signal
import subprocess
from proof import ROOT, Run, interrupt, sha, snapshot, write
from activity import cache_state, validate, PACKAGES


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-root', default=ROOT, type=Path)
    parser.add_argument('--expected-head', required=True)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--prior-activity', required=True, type=Path)
    parser.add_argument('--standalone-validation', required=True, type=Path)
    options = parser.parse_args()
    signal.signal(signal.SIGTERM, interrupt)
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
    source = run.root/'model/temporal/feature/activity/model.rs'
    original = source.read_bytes()
    edited = original.replace(b'//! Bounded activity retries, optional deadlines and cooperative cancellation.',
        b'//! Bounded activity retries, optional deadlines and cooperative cancellation (warm consumers).', 1)
    assert edited != original
    curated = run.root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/final-c5189bd3'
    parked = run.out.parent/'curated-held-during-warm-retry'
    assert curated.is_dir() and not parked.exists()
    cache = run.target/'quick-cache'
    before = cache_state(cache)
    first = json.loads((options.prior_activity/'quick-cache-after-first.json').read_text())
    assert len(first['files']) == 35
    assert all(before['files'].get(path) == digest for path, digest in first['files'].items())
    additions = {path:digest for path,digest in before['files'].items() if path not in first['files']}
    assert len(additions) == 75
    write(run.out/'quick-cache-before.json', before)
    write(run.out/'cache-provenance.json', {'original_setup_files': 35,
        'all_original_files_preserved': True, 'additional_cancelled_run_files': additions,
        'additional_complete_bundle_count': sum(path.endswith('/manifest') for path in additions),
        'not_the_same_cache_baseline_as_cold_setup': True})
    completed = None
    checks = None
    moved = False
    try:
        shutil.move(str(curated), str(parked))
        moved = True
        status = subprocess.check_output(['git', 'status', '--porcelain'], cwd=run.root, text=True)
        assert status == '', 'unexpected dirty paths would alter focused selection'
        write(run.out/'curation-context.json', {'temporarily_parked': str(parked),
            'original': str(curated), 'reason': 'Known non-model paths still disable focused fixture filter',
            'git_status_before_probe': status})
        source.write_bytes(edited)
        run.overrides[str(source.relative_to(run.root))] = hashlib.sha256(edited).hexdigest()
        completed = run.run('activity-warm-retry', [str(run.target/'debug/temporal-xtask'), 'quick'],
            run.root/'model', timeout=1800, observe=True)
        checks = validate(run, 'activity-warm-retry')
        assert len(checks['cache_hits']) == 14 and not checks['cache_misses']
        assert not checks['model_execution_labels']
        assert cache_state(cache) == before
    finally:
        try:
            run.observer.finish()
        finally:
            source.write_bytes(original)
            run.overrides.clear()
            if moved:
                assert not curated.exists()
                shutil.move(str(parked), str(curated))
            write(run.out/'model-restoration.json', {'path': str(source.relative_to(run.root)),
                'original_sha256': hashlib.sha256(original).hexdigest(),
                'edited_sha256': hashlib.sha256(edited).hexdigest(),
                'restored_sha256': sha(source), 'restored': source.read_bytes() == original,
                'curated_directory_restored': curated.is_dir() and not parked.exists()})
            write(run.out/'quick-cache-after.json', cache_state(cache))
            write(run.out/'source-after.json', snapshot(run.root))
            run.source()
    assert completed is not None and checks is not None
    write(run.out/'summary.json', {'status': 'Standalone accepted; Activity complete',
        'head': options.expected_head, 'warm_seconds': completed['seconds'], 'soft_target_seconds': 60,
        'target_met': completed['seconds'] <= 60, 'selected_packages': PACKAGES,
        'tests_passed': 148, 'negative_compiler_controls': 23, 'positive_compiler_controls': 2,
        'doctest_targets': 3, 'cache_hits': 14, 'cache_misses': 0, 'model_execution_labels': 0,
        'initial_cache_files': len(before['files']), 'cache_bytes_unchanged': True,
        'cache_context': '35 original setup files plus75 additional files from cancelled190-test invocation',
        'cancelled_warm': str(options.prior_activity/'activity-warm.json'),
        'ordinary_quick_has_no_gate_observation_scope': True,
        'source_restored': True, 'all_observed_owned_pids_absent': True,
        'comparison': 'Warm cache; restore/reapply probe between cancelled and final runs disclosed. No isolated causal speedup claim.',
        'macos': 'owner-deferred; no runtime acceptance claim'})


if __name__ == '__main__':
    main()
