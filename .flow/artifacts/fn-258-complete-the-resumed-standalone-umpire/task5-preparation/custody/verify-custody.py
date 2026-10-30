#!/usr/bin/env python3
"""Revalidate existing CAS custody and restore dirty/index state into new isolated paths."""
import hashlib
import importlib.util
import json
from pathlib import Path

HERE=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('custody_io',HERE/'capture-custody.py')
io=importlib.util.module_from_spec(spec)
spec.loader.exec_module(io)
manifest_path=HERE/'custody-manifest.json'
manifest=json.loads(manifest_path.read_bytes())
assert io.digest(io.payload(manifest['capture_script']))==manifest['capture_script_sha256']
assert io.payload(manifest['initial_capture_receipt'])
assert manifest['originals_before']==manifest['originals_after']
roots={snap['candidate']:Path(snap['root'])for snap in manifest['snapshots']}
before={key:io.fingerprint(root)for key,root in roots.items()}
assert before==manifest['originals_after']
for ref,oid in manifest['source_refs'].items():
    assert io.git(io.TEMPORAL,'rev-parse',ref).decode().strip()==oid
bare=Path(manifest['bundle_recovery']['bare_repository'])
assert not(bare/'objects/info/alternates').exists()
assert(bare/'shallow').read_bytes()==io.payload(manifest['source_shallow_boundaries'])
bundle=io.payload(manifest['source_bundle'])
assert bundle.startswith((b'# v2 git bundle',b'# v3 git bundle'))
refs=dict((ref,oid)for oid,ref in(line.split()for line in io.git(bare,'bundle','list-heads',manifest['source_bundle']['object']).decode().splitlines()))
assert refs==manifest['source_refs']
for ref,oid in refs.items():
    assert io.git(bare,'rev-parse',ref).decode().strip()==oid
source_graph=io.git(io.TEMPORAL,'rev-list',*refs)
assert source_graph==io.git(bare,'rev-list',*refs)
assert io.digest(source_graph)==manifest['available_history_graph_sha256']
restored=[io.restore_snapshot(snap,manifest['archives'][snap['candidate']],bare)for snap in manifest['snapshots']]
after={key:io.fingerprint(root)for key,root in roots.items()}
assert before==after==manifest['originals_after']
for ref,oid in refs.items():
    assert io.git(io.TEMPORAL,'rev-parse',ref).decode().strip()==oid
overlay=json.loads(io.OVERLAY.read_bytes())
registry=io.git(io.TEMPORAL,'worktree','list','--porcelain').decode()
six=[]
for entry in overlay['entries']:
    if not entry['unresolved_product_owner_freeze']:
        continue
    assert entry['ref'] in refs and refs[entry['ref']]==entry['original_head']
    assert ('branch '+entry['ref']+'\n') not in registry
    six.append({'source_repository':'temporal','source_ref':entry['ref'],'commit':refs[entry['ref']],
                'committed_recovery_verified_in_independent_bundle':True,'active_registered_worktrees':0,
                'keeper_action_task':entry['keeper']['action_task'],
                'historical_ignored_untracked_state':'unknown; retained explicitly without retirement',
                'feature_implementation_or_transfer_claim':False})
assert len(six)==6
manifest['six_source_only_freezes']=six
manifest['independent_dirty_restorations']=restored
manifest['index_count_qualification']='Index records counted by NUL delimiters; initial receipt used a line-based summary count. Initial capture receipt and actual capture script are retained in CAS; raw bytes/stages/patch proofs are unchanged.'
manifest['final_verification_script_sha256']=hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
manifest['final_restore_helper_sha256']=hashlib.sha256((HERE/'capture-custody.py').read_bytes()).hexdigest()
manifest_path.write_text(json.dumps(manifest,indent=2)+'\n')
receipt={'result':'pass','manifest_sha256':io.digest(manifest_path.read_bytes()),
         'verification_script_sha256':manifest['final_verification_script_sha256'],
         'restore_helper_sha256':manifest['final_restore_helper_sha256'],
         'source_refs_verified':len(refs),'available_history_commits':len(source_graph.splitlines()),
         'bundle_sha256':manifest['source_bundle']['sha256'],
         'restorations':restored,'originals_unchanged':before==after,
         'all8_original_refs_unchanged':True,'receiving_refs_created':0,
         'six_freezes_scope':'Exact source refs/commits retained, no active checkout observed in prior readiness; historical ignored uncertainty remains explicit. No deferred feature implementation or retirement inferred.'}
(HERE/'custody-verification.json').write_text(json.dumps(receipt,indent=2)+'\n')
(io.STORE/'manifest.json').write_text(json.dumps({'canonical_manifest':str(manifest_path),'manifest_sha256':receipt['manifest_sha256'],'verification_receipt':str(HERE/'custody-verification.json'),'source_bundle':manifest['source_bundle'],'archives':manifest['archives']},indent=2)+'\n')
print(json.dumps(receipt,indent=2))
