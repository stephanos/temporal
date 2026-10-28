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
assert len(snapshot['exception_and_selection_controls']) == 16
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
assert readiness['current_pair'] == {'temporal':'3f5f613578f2857b738a7792fad08f12b3ea0a41','umpire':'05473210b48c3db862f22051eb7154ce98258f98'}
assert readiness['receiving_state']['proof_sha256'] == sha(snapdir / 'verification.json')
assert readiness['consumer_worktree_clean_now']
assert len(readiness['tracker_statuses']['definition_vs_current_differences']) == 22
for binding in [readiness['focused_current_pair'], readiness['focused_current_pair']['workspace_observation']]:
    check_binding({'path':binding.get('receipt',binding.get('path')), 'sha256':binding['sha256']})
refs = subprocess.check_output(['git','--no-optional-locks','-C',str(TEMPORAL.parent/'umpire'),'for-each-ref','--format=%(refname)','refs/recovery/fn258/']).decode().splitlines()
assert not refs
result = {'result':'pass','historical_files_exact':len(historical['files']), 'overlay_controls':15,'snapshot_controls':16,'prepared_targets':48,'unchanged_new_custody_commit_tree_pairs':50,'real_receiving_refs':0,'R6_complete':False,'task4':'in_progress; broad proof not claimed','current_pair':readiness['current_pair'],'frozen_input_locators_and_hashes_verified':True,'old_prepared_manifest_reconstructed_exactly':True,'current_definition_status_differences':22,'current_tracker_observations':60,'dated_supersession':readiness['dated_supersession'],'script_sha256':sha(Path(__file__)), 'bindings':{n:sha(HERE/n) for n in ['disposition-overlay.json','disposition-overlay-verification.json','receiving-readiness.json','receiving-snapshots/prepared-targets.json','receiving-snapshots/verification.json','receiving-snapshots/proposed-create-only-transaction.txt']}}
(HERE/'current-refresh-verification.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
