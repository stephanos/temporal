from pathlib import Path
import hashlib,json,re,subprocess
P=Path(__file__).parent
load=lambda n:json.loads((P/n).read_text())
after=load('after-inventory.json');initial=load('attempt1-inventory.json');mapping=load('expected-mapping.json')
def key(c):return c['package'],c['target_kind'],c['target'],c['test']
expected={key(c):c for c in after['cases']};seen={};exeids={b['executable']:(b['package'],b['target_kind'],b['target']) for b in initial['binaries']}
logs=[('affected-tests1.log',None),('tla-repair.log',('umpire-tla','test','main')),('tlc-tests.log',('umpire-tlc',None,None)),('apalache-shared.log',('umpire-apalache','test','main')),('runtime-shared.log',('umpire-runtime','test','main')),('dsl-final.log',('umpire-dsl','test','main')),('interpreted-shared.log',('umpire-fixtures','test','fixtures'))]
for name,fixed in logs:
 current=None
 for no,line in enumerate((P/name).read_text().splitlines(),1):
  heading=re.search(r'Running (.+) \(([^)]+)\)',line)
  if heading:
   if fixed is None:current=exeids.get(heading[2])
   elif fixed[1] is None:current=(fixed[0],'lib','umpire_tlc') if 'unittests' in heading[1] else (fixed[0],'test','main')
   else:current=fixed
  case=re.match(r'test (.+) \.\.\. (ok|ignored|FAILED)',line)
  if not current or not case:continue
  test=case[1].removesuffix(' - should panic')
  if current==('umpire-dsl','test','main') and test.startswith('catalog::'):test='catalog_tests::'+test.removeprefix('catalog::')
  k=(*current,test)
  if k in expected:seen[k]={'identity':expected[k],'status':case[2],'log':name,'line':no}
missing=sorted(set(expected)-set(seen));bad=[r for r in seen.values() if r['status']!=('ignored' if r['identity']['ignored'] else 'ok')]
assert not missing and not bad,(missing,bad)
coverage={'count':len(seen),'passed':sum(v['status']=='ok' for v in seen.values()),'ignored':sum(v['status']=='ignored' for v in seen.values()),'cases':list(seen.values()),'method':'Latest per-identity observations across retained affected run, repaired TLA target, previously unexecuted TLC targets, and source-structure followups; duplicate executions are not added.'}
(P/'coverage.json').write_text(json.dumps(coverage,indent=2)+'\n')
source=[]
base=mapping['base_commit']
for line in subprocess.check_output(['git','diff','--cached','--name-status','--find-renames',base],text=True).splitlines():
 parts=line.split('\t');status=parts[0];old=parts[1];new=parts[-1]
 oldbytes=subprocess.run(['git','show',base+':'+old],capture_output=True).stdout if not status.startswith('A') else None
 newbytes=Path(new).read_bytes() if Path(new).is_file() else None
 source.append({'status':status,'before_path':old if oldbytes is not None else None,'after_path':new if newbytes is not None else None,'before_sha256':hashlib.sha256(oldbytes).hexdigest() if oldbytes is not None else None,'after_sha256':hashlib.sha256(newbytes).hexdigest() if newbytes is not None else None})
beforehash=load('before-source-hashes.json');snap={f:h for f,h in beforehash.items() if f.endswith(('.snap','.stderr')) or '/tests/ui/' in f or '/tests/admission-ui/' in f or '/tests/pass/' in f}
assert len(snap)==563
assert all(Path(f).is_file() and hashlib.sha256(Path(f).read_bytes()).hexdigest()==h for f,h in snap.items())
exceptions=[]
for m in load('author.json')['target_mappings']:
 if m.get('exception'):exceptions.append({'package':m['package'],'target':m['old_target'],'reason':m['exception']})
exceptions+=load('backends.json')['exceptions']
for m in load('catalogs.json')['targets']:
 if m.get('exception'):exceptions.append({'package':m['package'],'target':m['old_target'],'reason':m['exception']})
for pkg in load('runtime.json')['packages']:
 for m in pkg.get('exceptions',[]):exceptions.append(dict(package=pkg['package'],**m))
for target,reason in load('dev.json')['exceptions'].items():exceptions.append({'target':target,'reason':reason})
assert len({tuple(sorted(r['after'].items())) for r in mapping['cases']})==925
strategy=load('strategy-inventory.json');assert strategy['pass_exact'] and strategy['logical_count']==23
allsource=subprocess.check_output(['git','diff',base,'--','model/umpire'],text=True)
# Runtime/library source remains byte-identical; only tests, fixture harnesses, manifests and command mirrors change.
production=[f for f,h in beforehash.items() if '/src/' in f and '/tests/' not in f and Path(f).is_file() and hashlib.sha256(Path(f).read_bytes()).hexdigest()!=h]
assert not production,production
artifact={'task':'fn-254-fast-validation-gates-and-a-real-inner.4','base_commit':base,'inventory_reference':'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/planning/task13-pre-move-20261025/pre-move-logical-inventory.json','inventory_reference_source':mapping['inventory_source'],'inventory_source_equivalence':'git diff e160d7213fc02c6425cbcd840e50e205abbd7006..76408c558588eec6e226335c54f42f8bf2e5f496 -- model/umpire model/Cargo.toml was empty before edits','libtest':{'before':925,'after':925,'ignored_before':sum(r['before']['ignored'] for r in mapping['cases']),'ignored_after':sum(r['after']['ignored'] for r in mapping['cases']),'affected_observed':605,'affected_passed':coverage['passed'],'affected_ignored':coverage['ignored'],'unrelated_deferred':320,'cases':[dict(r,observed_after=key(r['after']) in expected) for r in mapping['cases']]},'modes':{'old_inventory':'umpire-tlc/strategy-equivalence; original per-binary features retained in task2/proof/workspace-frozen-inventory.json','after_affected':'default features, UMPIRE_BACKENDS=require, UMPIRE_APALACHE unset, INSTA_UPDATE=no','strategy_equivalence':{'listed':23,'ignored':0,'same_logical_identities':True,'executed':'15 unit tests plus controls::real_tlc_failed_batches_preserve_independent_traces','full_corpus':'not repeated; prior task2 proof retained; final combined validation remains .8'},'concurrency':'Initial affected run had RUST_TEST_THREADS unset; total peak unknown. Subsequent tests RUST_TEST_THREADS=2; Cargo jobs=2, incremental=0.'},'binary_counts':load('binary-counts.json'),'profile':{'test_debug':'line-tables-only','test_overflow_checks':True,'observed_artifact_profiles':[b['profile'] for b in after['binaries']],'masking_debug_env':'CARGO_PROFILE_TEST_DEBUG and CARGO_PROFILE_DEV_DEBUG explicitly unset; RUSTFLAGS and CARGO_ENCODED_RUSTFLAGS unset'},'compile_observations_seconds':{'failed_initial_dev':21.760,'failed_initial_affected':41.958,'successful_continuation':44.932,'final_warm_continuation':23.854,'before_compile_link':'unknown; fresh timing baseline waived','link_only_before_after':'unknown','speedup':'not established; repeated attempts share the same task-owned cache'},'linker':'No change: lld/mold absent from PATH, no alternative linker admitted through mise or measured. No new dependency/pin.','exceptions':exceptions,'snapshot_ui_sha256':snap,'doctests':{'separate_from_libtest':True,'affected_passed':4,'affected_ignored':1,'source_identity':'All framework library/proc-macro source bytes unchanged and doctest-capable target metadata identical (20 framework targets); no source doc includes README. Current affected exact doctest names retained below.','cases':[line.removeprefix('test ') for line in (P/'affected-doctests.log').read_text().splitlines() if line.startswith('test ') and not line.startswith('test result:')]},'source_delta':source,'post_run_attribution':{'production_source_changed':production,'reexecuted_structural_targets':['umpire-tla/main (96 tests)','umpire-apalache/main','umpire-runtime/main','umpire-dsl/main','umpire-fixtures/fixtures interpreted::'], 'other_completed_groups':'Only crate documentation was added to remaining new wrappers after their successful observations. Original fixture/Model bodies, test declarations and assertion strength are preserved. Runtime/public arithmetic fixture visibility was restored exactly; visibility is preserved at enclosing wrappers.','shared_fixture_imports':load('shared-fixture-imports.json')},'failures_retained':['dev-compile.log / dev-compile.jsonl (initial alias error; first /usr/bin/time launch127 separately disclosed)','affected-compile1.log / affected-compile1.jsonl (moved include mirror)','affected-tests1.log (578 passed,2 failed,2 ignored; dynamic self-source paths fixed and 96-test TLA target rerun green)','affected-clippy.log and affected-clippy2.log through affected-clippy4.log (new nesting fallout; final affected-clippy5.log rc0)'],'deferred':['unrelated 320 libtest identities and unrelated doctests','unfiltered combined model-check/model-validate until task .8','required Apalache real backend mode and final timing/link attribution; no broad-gate green claimed']}
output=Path('.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task4/consolidation-identities.json');output.parent.mkdir(parents=True,exist_ok=True);output.write_text(json.dumps(artifact,indent=2)+'\n')
print(json.dumps({'coverage':{k:v for k,v in coverage.items() if k!='cases'},'durable_proof':str(output),'source_delta_entries':len(source),'snapshot_ui_hashes':len(snap),'exceptions':len(exceptions)},indent=2))
