import hashlib,json,pathlib,subprocess
W=pathlib.Path(__file__).resolve().parents[6];A=pathlib.Path(__file__).resolve().parents[1];P=A/'proof';OLD=A.parent/'proof'
load=lambda p:json.loads(p.read_text());sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
correction=load(P/'prefix-correction.json');changed={r['path']:r for r in correction['files']};original=load(OLD/'snapshot-registry-renewal.json');rows=[]
for r in original['snapshot_ui']:
 path=r['path'];actual=sha(W/path)
 if path in changed:
  c=changed[path];assert r['sha256']==c['old_sha256'] and actual==c['new_sha256'];after=(W/path).read_bytes();assert after==(OLD/('generated-'+pathlib.Path(path).name)).read_bytes();before=after.replace(correction['new_prefix'].encode(),correction['old_prefix'].encode());assert hashlib.sha256(before).hexdigest()==r['sha256'];rows.append(dict(path=path,previous_sha256=r['sha256'],current_sha256=actual,kind='admitted prefix-only correction'))
 else:assert actual==r['sha256'];rows.append(dict(path=path,previous_sha256=r['sha256'],current_sha256=actual,kind='unchanged'))
assert len(rows)==563 and len(changed)==3
for f,h in original['cargo_lock_sha256'].items():assert sha(W/f)==h
rust_changes=subprocess.check_output(['git','diff','a21f91b116','--','model/**/*.rs','model/**/Cargo.toml','model/**/Cargo.lock'],cwd=W,text=True);assert not rust_changes
prior=load(A.parent/'index.json')
for path,h in prior['source_sha256'].items():assert sha(W/path)==h
result={'scope_admission':'8cf15a04e6','snapshot_ui':rows,'snapshot_ui_count':563,'unchanged_snapshot_ui_count':560,'admitted_prefix_only_corrections':3,'all_other_snapshot_bytes_unchanged':True,'all_diagnostic_bytes_preserved_modulo_exact_workspace_prefix':True,'cargo_lock_sha256':original['cargo_lock_sha256'],'registry_tuples':original['registry_tuples'],'registry_count':119,'rust_and_manifest_changes_since_runner_candidate':False,'prior_runner_source_hashes_still_exact':True,'inventory_receipt_retained':'../proof/producer-comparison.json (962 Cargo==nextest identities;4ignored)','doctest_receipt_retained':'../proof/doctest-identities.json (5 names/1ignored)','representative_receipts_retained':['../proof/representative.json','../proof/representative-temporal.json'],'full_temporal_run':'not launched; root cadence clarification retains representative proof and defers unrelated broad execution','framework_full_run':'Task11 owns isolated-copy full framework gate on joint candidate; parent integrates its exact receipt.'}
(P/'byte-accounting.json').write_text(json.dumps(result,indent=2)+'\n');print('560 unchanged snapshot/UI files +3 exact prefix corrections;119 registry tuples;all runner/Rust source unchanged')
