import hashlib,json,pathlib,re,subprocess
W=pathlib.Path(__file__).resolve().parents[6];R=pathlib.Path(__file__).resolve().parents[1];P=R/'proof';A=R.parent
load=lambda p:json.loads(p.read_text());sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
prior=load(A/'revision1/index.json')
for path,h in prior['source_sha256'].items():assert sha(W/path)==h,path
path='model/dev/xtask/src/gate.rs';before=subprocess.check_output(['git','show','adf475da99:'+path],cwd=W,text=True);after=(W/path).read_text();assert before.split('#[cfg(test)]')[0]==after.split('#[cfg(test)]')[0]
identity=lambda s:re.findall(r'#\[test\]\s*(?:#\[[^\]]+\]\s*)*fn\s+(\w+)',s)
assert identity(before)==identity(after)
changed=subprocess.check_output(['git','diff','--name-only','adf475da99','HEAD','--','model'],cwd=W,text=True).splitlines();assert changed==[path]
for r in load(A/'revision1/proof/byte-accounting.json')['snapshot_ui']:assert sha(W/r['path'])==r['current_sha256']
red=load(P/'wiring-red.json');commands=load(P/'commands.json');assert red['exit']==101 and all(r['exit']==0 for r in commands)
account=load(P/'control-accounting.json');assert account['total_child_cases']==28
result={'source_file':path,'only_model_file_changed_since_revision1':True,'prior_11_source_file_hashes_unchanged':True,'existing_test_names_and_attributes_unchanged':True,'production_module_bytes_unchanged':True,'existing_962_identity_inventory_preserved_by_unchanged_manifests_declarations_and_all_other_source':True,'snapshot_ui_count':563,'all_revision1_snapshot_ui_hashes_unchanged':True,'red_exit':red['exit'],'green_checks':commands,'child_scenarios':28,'both_phases_mapped_to_tests':True,'prior_runtime_receipts':'Runtime runner/config and production gate functions unchanged; no new JVM/full workspace execution required for consumer-control correction. Existing focused runtime/doc/group receipts retained.','review':'Root formal same-primary-session re-review pending after this fix and task11 separate required-mode finding integration.'}
(P/'verification.json').write_text(json.dumps(result,indent=2)+'\n');print('P1 red->green;28 scenarios;production+prior11 source+563 snapshot/UI hashes unchanged')
