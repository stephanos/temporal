#!/usr/bin/env python3
"""Check current refresh against immutable historical and isolated snapshot proofs."""
import hashlib
import json
from pathlib import Path
import subprocess

HERE = Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p / '.flow/specs').is_dir())
def read(p): return json.loads(p.read_bytes())
def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
def check_binding(binding):
    p = Path(binding['path'])
    if not p.is_absolute(): p = TEMPORAL / p
    assert sha(p) == binding['sha256'], str(p)

archive = HERE / 'historical-preparation-3658582'
historical = read(archive / 'preservation-manifest.json')
for row in historical['files']:
    assert sha(archive / row['file']) == row['sha256']
overlay = read(HERE / 'disposition-overlay-verification.json')
assert overlay['result'] == 'pass' and not overlay['errors']
assert overlay['overlay_sha256'] == sha(HERE / 'disposition-overlay.json')
assert overlay['script_sha256'] == sha(HERE / 'disposition-overlay.py')
for binding in overlay['input_bindings'].values(): check_binding(binding)
controls = read(HERE / 'disposition-overlay-negative-controls.json')['controls']
assert len(controls) == 15 and all(c['passed'] for c in controls)
assert overlay['original_candidate_transfers'] == overlay['r6_completed_dispositions'] == 0
snapdir = HERE / 'receiving-snapshots'
snapshot = read(snapdir / 'verification.json')
assert snapshot['result'] == 'pass' and snapshot['verified_targets'] == 48
assert snapshot['new_agent_custody_commits'] == 50
assert len(snapshot['exception_and_selection_controls']) == 24
assert all(c['passed'] for c in snapshot['exception_and_selection_controls'])
assert snapshot['prepared_manifest_sha256'] == sha(snapdir / 'prepared-targets.json')
for field, name in [('verifier_sha256','verify-snapshots.py'), ('builder_sha256','build-snapshots.py'), ('common_helper_sha256','snapshot_common.py')]:
    assert snapshot[field] == sha(snapdir / name)
manifest = read(snapdir / 'prepared-targets.json')
for binding in manifest['input_bindings'].values(): check_binding(binding)
correction = read(snapdir / 'input-provenance-correction.json')
assert correction['corrected_prepared_manifest_sha256'] == sha(snapdir / 'prepared-targets.json')
old = dict(manifest)
old['input_bindings'] = read(snapdir / 'provenance-before-correction/input-bindings.json')
assert hashlib.sha256((json.dumps(old, indent=2) + '\n').encode()).hexdigest() == correction['previous_prepared_manifest_sha256']
assert correction['all50_commit_and_tree_oids_unchanged']
assert not correction['custody_commit_messages_embed_input_schema_or_manifest_hash']
summary = read(snapdir / 'target-summary.json')
assert sha(Path(summary['bundle']['path'])) == summary['bundle']['sha256']
assert snapshot['transaction_sha256'] == sha(snapdir / 'proposed-create-only-transaction.txt')
readiness = read(HERE / 'receiving-readiness.json')
assert readiness['current_pair'] == {'temporal':'cde1ee67efd266a058193694c2ddb287598b9ea5','umpire':'05473210b48c3db862f22051eb7154ce98258f98'}
assert readiness['receiving_state']['proof_sha256'] == sha(snapdir / 'verification.json')
assert readiness['consumer_worktree_clean_now']
assert len(readiness['tracker_statuses']['definition_vs_current_differences']) == 22
observations = read(HERE / 'current-tracker-observations.json')['observations']
def unique_tasks(rows):
    requested = [row['requested_task'] for row in rows]
    assert len(requested) == len(set(requested)), 'duplicate requested task'
    assert all(row['requested_task'] == row['observation']['id'] for row in rows)
    return len(set(requested))
tracker_count = unique_tasks(observations)
assert readiness['tracker_statuses']['unique_task_count'] == tracker_count
try:
    unique_tasks(observations + [observations[0]])
except AssertionError:
    duplicate_task_refused = True
else:
    raise AssertionError('duplicate task control false pass')
current_work = read(HERE / 'receiving-ref-creation-work.json')['work']
frozen_work = read(snapdir / 'receiving-ref-creation-work.input.json')['work']
def identity_rows(rows):
    assert all(set(row) == {'candidate_id','receiver','source_original_head','source_r1_observed_head','source_recovery','provisional_path_mappings','required_before_create'} for row in rows)
    by_id = {row['candidate_id']:row for row in rows}
    assert len(rows) == len(by_id) == 48
    fields = ['candidate_id','receiver','source_original_head','source_r1_observed_head','provisional_path_mappings']
    return {key:{field:row[field] for field in fields} for key,row in by_id.items()}
assert identity_rows(current_work) == identity_rows(frozen_work)
changed_plan = json.loads(json.dumps(current_work))
changed_plan[0]['provisional_path_mappings'][0]['destination_path'] = 'invented-destination'
assert identity_rows(changed_plan) != identity_rows(frozen_work)
prerequisite_changes = [{'candidate_id':a['candidate_id'],'frozen':a['required_before_create'],'current':b['required_before_create']} for a,b in zip(frozen_work,current_work) if a['required_before_create'] != b['required_before_create']]
assert all(a['source_recovery'] == b['source_recovery'] for a,b in zip(frozen_work,current_work))
assert len(prerequisite_changes) == 48
committed_bundle = snapdir / 'prepared-snapshots.bundle'
def check_committed_bundle(data):
    assert len(data) == summary['bundle']['size']
    assert hashlib.sha256(data).hexdigest() == summary['bundle']['sha256']
committed_bytes = committed_bundle.read_bytes()
check_committed_bundle(committed_bytes)
try:
    check_committed_bundle(committed_bytes[:-1] + bytes([committed_bytes[-1] ^ 1]))
except AssertionError:
    same_size_bundle_corruption_refused = True
else:
    raise AssertionError('same-size bundle corruption false pass')
assert snapshot['computed_recovery_coverage_sha256'] == sha(snapdir / 'computed-recovery-coverage.json')
for binding in [readiness['historical_focused_pair'], readiness['historical_focused_pair']['workspace_observation']]:
    check_binding({'path':binding.get('receipt',binding.get('path')), 'sha256':binding['sha256']})
def source_git(repo, *args):
    return subprocess.check_output(['git','--no-optional-locks','--no-replace-objects','-C',str(repo),*args])
consumer_worktree = Path(readiness['consumer_worktree'])
assert source_git(consumer_worktree,'rev-parse','HEAD').decode().strip() == readiness['current_pair']['temporal']
assert source_git(TEMPORAL,'rev-parse','refs/heads/work/fn2583-consumers').decode().strip() == readiness['current_pair']['temporal']
assert not source_git(consumer_worktree,'status','--porcelain=v1','--untracked-files=all')
prior_consumer = '3f5f613578f2857b738a7792fad08f12b3ea0a41'
assert source_git(TEMPORAL,'diff','--name-only',prior_consumer,readiness['current_pair']['temporal']).decode().splitlines() == ['model/scripts/test-model-wiring.sh']
source_delta = source_git(TEMPORAL,'diff','--binary',prior_consumer,readiness['current_pair']['temporal'])
failed_receipt = HERE.parent / 'validation/final-pair-3f5-054/failed-default-3f5/execution/00-paired-default-check.json'
assert read(failed_receipt)['exit'] == 2
refs = subprocess.check_output(['git','--no-optional-locks','-C',str(TEMPORAL.parent/'umpire'),'for-each-ref','--format=%(refname)','refs/recovery/fn258/']).decode().splitlines()
assert not refs
result = {'result':'pass','historical_files_exact':len(historical['files']), 'overlay_controls':15,'snapshot_controls':len(snapshot['exception_and_selection_controls']),'prepared_targets':48,'unchanged_new_custody_commit_tree_pairs':50,'real_receiving_refs':0,'R6_complete':False,'task4':'in_progress; broad proof not claimed','current_pair':readiness['current_pair'],'frozen_input_locators_and_hashes_verified':True,'old_prepared_manifest_reconstructed_exactly':True,'current_definition_status_differences':22,'current_tracker_observations':tracker_count,'duplicate_tracker_task_control_refused':duplicate_task_refused,'current_frozen_identity_and_mappings_exact':True,'changed_mapping_control_detected':True,'observational_source_recovery_equal':True,'administrative_prerequisite_supersession':prerequisite_changes,'committed_bundle_sha256':sha(committed_bundle),'committed_bundle_size':committed_bundle.stat().st_size,'same_size_bundle_corruption_control_refused':same_size_bundle_corruption_refused,'dated_supersession':readiness['dated_supersession'],'current_source_ownHEAD_branch_and_clean_verified':True,'historical_focused_pair':readiness['historical_focused_pair']['pair'],'current_source_owner':'fn-258-complete-the-resumed-standalone-umpire.4','current_fresh_pair_empty_target_proof':'PENDING','only_consumer_changed_path':'model/scripts/test-model-wiring.sh','consumer_delta_sha256':hashlib.sha256(source_delta).hexdigest(),'prior_failed_fullgate_receipt_sha256':sha(failed_receipt),'prior_fullgate_exit':2,'script_sha256':sha(Path(__file__)), 'bindings':{n:sha(HERE/n) for n in ['disposition-overlay.json','disposition-overlay-verification.json','receiving-readiness.json','receiving-snapshots/prepared-targets.json','receiving-snapshots/verification.json','receiving-snapshots/proposed-create-only-transaction.txt']}}
(HERE/'current-refresh-verification.json').write_text(json.dumps(result,indent=2)+'\n')
print(f'PASS: currentcde ownHEAD/ref/clean;48 exact identity/mapping rows;24 snapshot+15 overlay controls;{tracker_count} unique observed tasks; committed bundle hash/size; historical3f5 focusedPASS and fullgateFAILED2 retained; fresh wholepair PENDING')
