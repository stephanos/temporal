#!/usr/bin/env python3
"""Verify the matched fixture evidence and produce a bounded site comparison."""
import hashlib
import json
from pathlib import Path
import stat

HERE = Path(__file__).resolve().parent
RUNS = HERE / 'runs'
BASE = HERE.parent / 'bound-baseline-measurement'

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def read(path):
    return json.loads(path.read_text())

def inventory(root):
    rows = []
    for path in sorted(root.rglob('*')):
        assert not path.is_symlink(), path
        if path.is_file():
            rows.append(dict(path=str(path.relative_to(root)), sha256=digest(path), mode=stat.S_IMODE(path.stat().st_mode)))
    return rows

current = read(RUNS / 'execution-evidence.json')
baseline = read(BASE / 'runs/execution-evidence.json')
assert current['bindings'] == baseline['bindings'] and len(current['bindings']) == 14
assert digest(HERE / 'run_current.py') == current['driver_prelaunch_sha256']
assert digest(Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')) == baseline['hashes']['pinned-go-binary'] == current['hashes']['pinned-go-binary']
assert digest(BASE / 'handoff-output.sha256') == 'd1381842f5eb1b8a1a4b1b5b24d0d54544f61ab30cfea311647e65225abc1133'
baseline_outputs = 0
for line in (BASE / 'handoff-output.sha256').read_text().splitlines():
    expected, relative = line.split('  ', 1)
    assert digest(BASE / relative) == expected, relative
    baseline_outputs += 1
initial = read(RUNS / 'initial-source-inventory.json')
assert len(initial) == current['source_file_count'] == 982
assert len({row['path'] for row in initial}) == 982
assert inventory(Path(current['scratch'])) == initial
assert read(RUNS / 'final-source-inventory.json') == initial
shipped = read(RUNS / 'shipped-source-before-overlay.json')
assert len(shipped) == 978
source_by_path = {row['path']: row for row in shipped}
frozen_by_path = {row['path']: row for row in initial}
assert sorted(frozen_by_path.keys() - source_by_path.keys()) == sorted(value for key,value in current['predeclared_supplementary_paths'].items() if key != 'descriptor_dup_linux.go')
assert [path for path in source_by_path if source_by_path[path] != frozen_by_path[path]] == ['deterministicio/profile.go']
repo = HERE.parents[4]
for row in shipped:
    path = repo / 'tools/gomad3' / row['path']
    assert digest(path) == row['sha256'] and stat.S_IMODE(path.stat().st_mode) == row['mode'], path
for name,destination in current['predeclared_supplementary_paths'].items():
    assert digest(HERE / name) == frozen_by_path[destination]['sha256']
assert read(RUNS / 'bound-environment-before-launch.json')['bindings'] == baseline['bindings']
assert all(command['exit_code'] == 0 for command in current['commands'])
for command in current['commands']:
    assert all(command['child_environment_subset'][name] == value for name,value in baseline['bindings'].items())
    assert command['elapsed_seconds'] >= 0 and (RUNS / command['output']).is_file()
for check in current['inventory_checks']:
    assert not check['added'] and not check['removed'] and not check['changed']
for name,path in [('original-baseline',Path('/tmp/fn109-baseline-reconstruction.lDSSw8Gx/tools/gomad3')), ('historical-artifacts',BASE), ('historical-scratch',Path(baseline['scratch']))]:
    assert read(RUNS / (name+'-before.json')) == read(RUNS / (name+'-after.json')) == inventory(path)
checked_outputs = 0
for line in (RUNS / 'retained-output.sha256').read_text().splitlines():
    expected, relative = line.split('  ', 1)
    assert digest(RUNS / relative) == expected, relative
    checked_outputs += 1
campaigns = [command for command in current['commands'] if command['output'] in ['discard-10.log','discard-100.log','novel-10.log','novel-100.log']]
assert [command['output'] for command in campaigns] == ['discard-10.log','discard-100.log','novel-10.log','novel-100.log']
current_profiles = read(RUNS / 'profile-attribution.json')['cases']
baseline_profiles = read(BASE / 'runs/profile-attribution.json')['cases']
comparison = {'scope':'Same paired injected seed fixture on stock linux/arm64. Site attribution is not native qualification, universal copy absence, or private-map cardinality.', 'cases':{}}
for command in campaigns:
    case = command['output'][:-4]
    count = int(case.split('-')[1])
    now = current_profiles[case]
    old = baseline_profiles[case]
    measured = now['measurement']
    assert read(RUNS / (case+'-source-before.json')) == read(RUNS / (case+'-source-after.json')) == initial
    for key in ['attempted','succeeded','maximum_active','parallel','per_execution_stream_bytes','actual_per_execution_transcript_bytes','actual_per_execution_transcript_records','transcript_vocabulary','retained_success_count','artifact_count','limits','observed_runtime_settings']:
        assert measured[key] == old['measurement'][key], (case,key)
    assert measured['excluded_transport_calls'] == 0
    assert [(row['Returned'],row['Committed'],row['Active']) for row in measured['snapshots']] == [(0,0,2),(count-2,count-2,2),(count,count,0)]
    current_metadata = (RUNS / (case+'-binary-build-metadata.log')).read_text().splitlines()[1:]
    baseline_metadata = (BASE / 'runs' / (case+'-binary-build-metadata.log')).read_text().splitlines()[1:]
    assert current_metadata == baseline_metadata, case
    for stage_name,stage in now['profiles'].items():
        for metric,total in stage['totals'].items():
            for rows in ['all_allocation_leaf_sites','allocation_size_bucket_sites']:
                assert sum(row[metric] for row in stage[rows]) == total
            assert sum(row[metric] for row in stage['disjoint_stack_origin_categories'].values()) == total
        stream = sum(row['inuse_objects'] for row in stage['allocation_size_bucket_sites'] if row['category']=='fixture_stream_producer' and row['allocation_size_bucket_bytes']==1048576)
        transcript = sum(row['inuse_objects'] for row in stage['allocation_size_bucket_sites'] if row['category']=='fixture_transcript_encoder' and row['allocation_size_bucket_bytes']==1048576)
        assert (stream,transcript) == ((4,2) if stage_name in ['early-pair','late-pair'] else (0,0)), (case,stage_name)
    categories = {}
    for category in ['fixture_stream_producer','fixture_transcript_encoder','coverage_outcome_assessment','world_assessment','artifact_publication','named_campaign_policy_sites','direct_runLocal_allocation_sites']:
        categories[category] = {label:data['profiles']['completed']['disjoint_stack_origin_categories'].get(category,{}) for label,data in [('baseline',old),('current',now)]}
    comparison['cases'][case] = dict(command_seconds=command['elapsed_seconds'], exit_code=command['exit_code'], baseline_binary_sha256=baseline['hashes'][case+'/runner.test'], current_binary_sha256=current['hashes'][case+'/runner.test'], completed_categories=categories, baseline_snapshots=old['measurement']['snapshots'], current_snapshots=measured['snapshots'])
policy_old = read(BASE / 'runs/logical-policy-both-roles.json')
policy_now = read(RUNS / 'logical-policy-both-roles.json')
assert policy_old == policy_now
comparison['logical_policy_both_roles'] = policy_now
comparison['verification'] = dict(passed=True,source_files=982, shipped_files=978, complete_source_inventories=10, successful_commands=len(current['commands']), current_output_hashes=checked_outputs, baseline_handoff_hashes=baseline_outputs, driver_prelaunch_sha256=current['driver_prelaunch_sha256'], qualification_complete=False)
(HERE / 'comparison.json').write_text(json.dumps(comparison,indent=2)+'\n')
print(json.dumps(comparison['verification'],indent=2))
