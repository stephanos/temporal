#!/usr/bin/env python3
"""Independently verify retained source path sets, bound children and outputs."""
import hashlib
import json
from pathlib import Path
import stat

HERE = Path(__file__).resolve().parent
RUNS = HERE/'runs'
BINDINGS = dict(GOENV='off', GOFLAGS='', GOEXPERIMENT='', GODEBUG='', GOGC='100', GOMEMLIMIT='off', GOMAXPROCS='2', CGO_ENABLED='1', GOOS='linux', GOARCH='arm64', GOARM64='v8.0', GOWORK='off', GOTOOLCHAIN='local', TZ='UTC')

def read(name):
    return json.loads((RUNS/name).read_text())

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def inventory(root):
    result = []
    for path in sorted(root.rglob('*')):
        assert not path.is_symlink(), path
        if path.is_file():
            result.append(dict(path=str(path.relative_to(root)), sha256=digest(path), mode=stat.S_IMODE(path.stat().st_mode)))
    return result

evidence = read('execution-evidence.json')
initial = read('initial-source-inventory.json')
assert len(initial) == 675
assert [row['path'] for row in initial] == sorted(row['path'] for row in initial)
assert len({row['path'] for row in initial}) == len(initial)
initial_by_path = {row['path']:row for row in initial}
assert read('bound-environment-before-launch.json')['bindings'] == BINDINGS
assert evidence['bindings'] == BINDINGS
assert all(command['exit_code'] == 0 for command in evidence['commands'])
for command in evidence['commands']:
    subset = command['child_environment_subset']
    assert all(subset[name] == value for name, value in BINDINGS.items()), command
    assert command['elapsed_seconds'] >= 0
    assert (RUNS/command['output']).is_file()
effective = read('effective-go-env-before-build.json')
for name in ['GOFLAGS', 'GOEXPERIMENT', 'CGO_ENABLED', 'GOOS', 'GOARCH', 'GOARM64', 'GOWORK', 'GOTOOLCHAIN']:
    assert effective[name] == BINDINGS[name]
assert effective['GOENV'] == '' and effective['GOVERSION'] == 'go1.27.1'
assert all(not row['added'] and not row['removed'] and not row['changed'] for row in evidence['inventory_checks'])
campaigns = [command for command in evidence['commands'] if command['output'] in ['discard-10.log', 'discard-100.log', 'novel-10.log', 'novel-100.log']]
assert [command['output'] for command in campaigns] == ['discard-10.log', 'discard-100.log', 'novel-10.log', 'novel-100.log']
cases = []
for command in campaigns:
    case = command['output'][:-4]
    assert read(case+'-source-before.json') == initial
    assert read(case+'-source-after.json') == initial
    measured = read(case+'/measurement.json')
    count = int(case.split('-')[1])
    retained = int(case.startswith('novel'))
    assert measured['attempted'] == measured['succeeded'] == count
    assert measured['retained_success_count'] == measured['artifact_count'] == retained
    assert measured['maximum_active'] == 2 and measured['excluded_transport_calls'] == 0
    assert measured['actual_per_execution_transcript_bytes'] == 1048576
    assert measured['observed_runtime_settings'] == dict(gogc_percent=100, gomaxprocs_threads=2, gomemlimit_bytes=9223372036854775807)
    assert [(row['Returned'], row['Committed'], row['Active']) for row in measured['snapshots']] == [(0,0,2), (count-2,count-2,2), (count,count,0)]
    metadata = (RUNS/(case+'-binary-build-metadata.log')).read_text()
    for setting in ['go1.27.1', 'CGO_ENABLED=1', 'GOOS=linux', 'GOARCH=arm64', 'GOARM64=v8.0']:
        assert setting in metadata, setting
    cases.append(dict(case=case, exit_code=command['exit_code'], elapsed_seconds=command['elapsed_seconds'], binary_sha256=digest(RUNS/case/'runner.test')))
assert read('final-source-inventory.json') == initial
assert inventory(Path(evidence['scratch'])) == initial
for retained, destination in evidence['predeclared_supplementary_paths'].items():
    assert digest(HERE/retained) == initial_by_path[destination]['sha256']
protected = {'original-baseline':Path('/tmp/fn109-baseline-reconstruction.lDSSw8Gx/tools/gomad3'), 'historical-artifacts':HERE.parent/'baseline-measurement', 'historical-scratch':Path('/tmp/fn109-r19-baseline-measurement.Bq9eSbCF/gomad3')}
protected_counts = {}
for name, path in protected.items():
    before = read(name+'-before.json')
    assert before == read(name+'-after.json') == inventory(path)
    protected_counts[name] = len(before)
assert protected_counts['original-baseline'] == 670
original_by_path = {row['path']:row for row in read('original-baseline-before.json')}
assert sorted(initial_by_path.keys()-original_by_path.keys()) == sorted(evidence['predeclared_supplementary_paths'].values())
changed_original = sorted(path for path in original_by_path if original_by_path[path] != initial_by_path[path])
assert changed_original == ['deterministicio/profile.go', 'runner/internal/execution/bootstrap_unix.go', 'runner/internal/execution/launch_plan_unix.go']
checked = 0
for line in (RUNS/'retained-output.sha256').read_text().splitlines():
    expected, relative = line.split('  ', 1)
    assert digest(RUNS/relative) == expected, relative
    checked += 1
for name, expected in evidence['hashes'].items():
    path = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go') if name == 'pinned-go-binary' else HERE/name.removeprefix('retained-input/') if name.startswith('retained-input/') else RUNS/name
    assert digest(path) == expected, name
driver = RUNS/'executed-driver-post-run-snapshot.py'
assert digest(driver) == evidence['hashes']['retained-input/run_bound.py'] == digest(HERE/'run_bound.py')
profiles = read('profile-attribution.json')['cases']
for case in profiles.values():
    for stage_name, stage in case['profiles'].items():
        for metric, total in stage['totals'].items():
            for rows in ['all_allocation_leaf_sites', 'allocation_size_bucket_sites']:
                assert sum(row[metric] for row in stage[rows]) == total
            assert sum(row[metric] for row in stage['disjoint_stack_origin_categories'].values()) == total
        publication = stage['disjoint_stack_origin_categories'].get('artifact_publication')
        if publication and publication['publication_denominator']:
            assert publication['publication_denominator'] == 1
            assert publication['alloc_bytes_per_publication'] == publication['alloc_bytes']
        stream = sum(row['inuse_objects'] for row in stage['allocation_size_bucket_sites'] if row['category'] == 'fixture_stream_producer' and row['allocation_size_bucket_bytes'] == 1048576)
        transcript = sum(row['inuse_objects'] for row in stage['allocation_size_bucket_sites'] if row['category'] == 'fixture_transcript_encoder' and row['allocation_size_bucket_bytes'] == 1048576)
        assert (stream, transcript) == ((4,2) if stage_name in ['early-pair', 'late-pair'] else (0,0))
result = dict(passed=True, source_files=675, complete_source_path_hash_mode_checks=10, protected_file_counts=protected_counts, successful_commands=len(evidence['commands']), retained_output_hashes_checked=checked, cases=cases, driver_snapshot_capture='post-run; matches executing driver completion hash, not a pre-launch claim', current_comparison_run=False, qualification_admitted=False)
(HERE/'verification-result.json').write_text(json.dumps(result, indent=2)+'\n')
print(json.dumps(result, indent=2))
