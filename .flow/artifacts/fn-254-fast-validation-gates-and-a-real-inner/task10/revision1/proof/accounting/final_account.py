import collections,hashlib,json,pathlib,re,subprocess
W=pathlib.Path(__file__).resolve().parents[7];ROOT=W;P=pathlib.Path(__file__).resolve().parent;B=json.loads((P/'baseline.json').read_text());BASE=B['baseline_source_commit'];F='model/umpire/dev/xtask/src/gate.rs';T='model/dev/xtask/src/gate.rs';FW='model/umpire/dev/xtask/tests/gate_wiring.rs';TW=T
files={}
def anchor(path,needle):
 text=(W/path).read_text();files[path]=hashlib.sha256(text.encode()).hexdigest();lines=text.splitlines();matches=[n for n,line in enumerate(lines,1) if needle in line];return {'path':path,'lines':matches,'needle':needle,'sha256':files[path]}
def record(path,needle):return anchor(path,needle)
stages=[]
for row in B['stages']:
 old=row['old'];parts=[]
 if old in ['retired-names','timing-controls','protobuf-controls','wasm-guest']:
  domains={'retired-names':[('framework','framework source tree, both retired patterns'),('temporal','repository minus model/umpire, same patterns and historical exemptions')],'timing-controls':[('framework','step status/diagnostics/temporary cleanup'),('temporal','warning budgets plus combined validate and recipe wiring')],'protobuf-controls':[('framework','core,kettle,usage,pins,missing generators,protoc failure and cleanup'),('temporal','temporal schema generation/freshness/symlink and handwritten host setup')],'wasm-guest':[('framework','stock,kettle,control,caught-panic,overflow plus two native reports'),('temporal','umpire-temporal-guest')]}[old]
 else:domains=[('temporal' if row['expected_owner']=='temporal' else 'framework','complete original stage')]
 for owner,cases in domains:
  new=old.removesuffix('-framework') if owner=='framework' else old;path=F if owner=='framework' else T
  if old=='tests':path='model/dev/xtask/src/test_run.rs';needle='pub(crate) fn run'
  else:needle='"'+new+'"'
  a=record(path,needle);parts.append({'owner':owner,'new':new,'covered_cases':cases,'source':a})
 stages.append({'old':old,'partitions':parts,'overlap':False,'status':'source mapped','note':'Framework source-boundary is additional; old source-boundary remains Temporal only.' if old=='source-boundary' else ''})
shell=[]
for x in B['shell']:
 if x['granularity']=='aggregate':continue
 old=x['old'];path=x['source_path'];name=old.split('::',1)[1];new=x['proposed_new'];status='source mapped';notes=[]
 if path=='model/scripts/test-model-wiring.sh':
  c=x['source_case'];stage=c['stage'];owner='temporal' if stage in ['fmt','clippy','docs','tests'] else 'framework';path=TW if owner=='temporal' else FW
  needle='let expected = [' if owner=='temporal' else 'fn full_dispatch_runs_each_leaf_once_in_order_and_bypasses_cache_from_first_command'
  if 'target' in c:
   if c['target']!='relative':status='missing exact target-mode assertion';notes.append('Current framework Fixture::command and shell build controls hardcode relative target. Need unset/absolute modes.')
   if stage=='builder-target':path='model/umpire/scripts/tests/build-guests-controls.sh';needle='cmp "$scratch/expected" "$scratch/calls"'
  if stage in ['fmt-framework','clippy-framework','docs-framework']:status='missing exact argv assertion';notes.append('Production command preserved, but fake mise labels first command tokens and no full exact argv assertion found.')
  if 'target' in c and 'fn guest_and_go_stage_target_matrix_preserves_unset_relative_and_absolute_paths' in (W/FW).read_text():
   path=FW;needle='fn guest_and_go_stage_target_matrix_preserves_unset_relative_and_absolute_paths';status='source mapped';notes=['Case selector: '+str(c)]
  if stage in ['fmt-framework','clippy-framework','docs-framework'] and 'CONTROL_EXPECT_FMT:-cargo fmt --all --check' in (W/FW).read_text():
   path=FW;needle={'fmt-framework':"'cargo fmt')",'clippy-framework':"'cargo clippy')",'docs-framework':"'cargo doc')"}[stage];status='source mapped';notes=['Whole argv enforced in fake mise; both timed modes in full dispatcher matrix.']
  new=f'{path}::{name}'
 elif path=='model/scripts/test-model-validate.sh':
  owner='temporal';needle='run_control' if name.startswith(('validate-gate','validate-benchmark','timer-benchmark')) else 'for mode in' if name.startswith('time-backends') else 'for conflict in';new=old
 elif path=='model/umpire/scripts/test-capabilities.sh':owner='framework';needle='for root in' if name.startswith('unsafe-override') else 'deny(unsafe_code)' if name.startswith('deny-') else 'for pair in';new=old
 elif path=='model/umpire/scripts/tests/proto-tools.sh':owner='framework';needle='bash "$script"';new=old
 elif path=='model/umpire/scripts/tests/bootstrap.sh':owner='framework';needle='guest-freshness.sh' if name.startswith('freshness') else 'guest.sh" example';new=old
 else:
  path=new.split('::',1)[0];owner='framework' if path.startswith('model/umpire/') else 'temporal';needle=''
  if 'test-time-gate' in path:
   needle='for control in' if name.startswith('step-') else 'for expected in' if name.startswith(('diagnostics-','temporary-')) else 'expect_warnings'
  elif 'proto-go' in path:
   needle={'generate-selected-input':'grep -Fx','grpc-selection':'--go-grpc_out=','fresh-check':'--check','stale-refusal':"refuse 'stale.go'",'symlink-refusal':"refuse 'not generated:'",'clean-after-removal':'rm "$destination','usage-':'for args in','invalid-selectors-before-protoc':'[ ! -e "$PROTO_LOG" ]','wrong-go-pin':'GO_VERSION=wrong','wrong-grpc-pin':'GRPC_VERSION=wrong','wrong-protoc-pin':'PROTO_VERSION=wrong','core-without-grpc':'mv "$scratch/','missing-grpc':'scripts/proto-tools.sh','missing-go':'scripts/proto-tools.sh','protoc-failure-status-42':'if PROTO_FAIL=1','protoc-failure-temporary-cleanup':'find "$scratch/tmp"','handwritten-host-outside-generated-tree':"printf 'handwritten"}.get(name,'')
   if not needle:
    for suffix,token in [('generate-selected-input','grep -Fx'),('grpc-selection','--go-grpc_out='),('fresh-check','--check'),('stale-refusal',"refuse 'stale.go'"),('symlink-refusal',"refuse 'not generated:'"),('clean-after-removal','rm "$destination'),('usage-','for args in')]:
     if suffix in name:needle=token
  elif path.endswith('tests/build-guests.sh'):
   if owner=='framework':needle='for pair in'
   else:needle='umpire-temporal-guest'
  elif path.endswith('framework-boundary.txt'):needle='retired_pattern'
  elif path.endswith('retired-names.sh'):needle='historical=(' if 'exclusions' in name else 'statuses=(' if 'raw-git' in name else "scan '"
  if not needle:needle='#!/usr/bin/env bash'
 a=record(path,needle)
 if not a['lines']:status='missing source anchor';notes.append('Need a precise selector for source accounting.')
 shell.append({'old':old,'new':new,'owner':owner,'source':a,'status':status,'notes':notes})
# Existing Rust source tree bytes, covering all prior declared targets without compiling them.
paths=subprocess.check_output(['git','ls-tree','-r','--name-only',BASE,'--','model'],cwd=ROOT,text=True).splitlines();unchanged=[];changed=[]
for path in paths:
 if not path.endswith(('.rs','Cargo.toml')):continue
 before=subprocess.check_output(['git','show',BASE+':'+path],cwd=ROOT);after=(W/path).read_bytes() if (W/path).exists() else None
 before_functions=collections.Counter(re.findall(rb'\bfn\s+(\w+)\s*(?:<[^>]*>)?\s*\(',before))
 after_functions=collections.Counter(re.findall(rb'\bfn\s+(\w+)\s*(?:<[^>]*>)?\s*\(',after or b''))
 removed=[k.decode() for k in (before_functions-after_functions)]
 row={'removed_function_names':removed,'path':path,'before_sha256':hashlib.sha256(before).hexdigest(),'after_sha256':hashlib.sha256(after).hexdigest() if after is not None else None}
 (unchanged if before==after else changed).append(row)
new=[];proofs={};baseline_rust={tuple(x['old'][k] for k in ['package','target_kind','target','test','ignored']) for x in B['rust']}
for filename,pkg in [('framework-native-tests.log','xtask'),('outer-native-tests.log','temporal-xtask')]:
 path='.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task10/revision1/proof/'+filename;text=(W/path).read_text();proofs[path]=hashlib.sha256(text.encode()).hexdigest();kind=None;target=None
 for line in text.splitlines():
  if 'Running unittests src/lib.rs' in line:kind='lib';target='xtask'
  elif 'Running unittests src/main.rs' in line:kind='bin';target='temporal-xtask'
  elif 'Running tests/main.rs' in line:kind='test';target='main'
  m=re.match(r'test (\S+) \.\.\. (ok|ignored)$',line)
  if m and kind:new.append({'package':pkg,'target_kind':kind,'target':target,'test':m[1],'ignored':m[2]=='ignored'})
new=[x for x in new if tuple(x[k] for k in ['package','target_kind','target','test','ignored']) not in baseline_rust]
# New declaration not in prior log is intentionally separate until conductor reruns it.
source_names=re.findall(r'#\[test\]\s*fn (\w+)\(', (W/FW).read_text())
seen={x['test'] for x in new}
unexecuted=[n for n in source_names if 'gate_wiring::'+n not in seen]
result={'schema_version':2,'source_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=W,text=True).strip(),'working_tree_input_sha256':files,'baseline_source_commit':BASE,'rust':[x['old'] for x in B['rust']]+new,'rust_mappings':[],'new_rust':new,'rust_source_proof':{'unchanged_files':unchanged,'changed_existing_files':changed,'prior930_identical_coordinates':True,'changed_test_identity':'temporal-xtask/test/main/actual_make_freshness_recipe_propagates_stale_failure','changed_test_identity_evidence':record('model/dev/xtask/tests/main.rs','fn actual_make_freshness_recipe_propagates_stale_failure'),'limits':'Source preservation plus affected package proof logs; not full930 execution.'},'stages':stages,'shell':shell,'new_controls':B['shell_ledger_input']['new_controls'],'affected_native_test_proof_sha256':proofs,'new_declared_framework_integration_cases_not_in_current_log':unexecuted,'gaps':[x for x in shell if x['status']!='source mapped'],'execution_claim':False}
(P/'final-inventory.json').write_text(json.dumps(result,indent=2,sort_keys=True)+'\n')
summary={'stage_old_identities':len(stages),'stage_partitions':sum(len(x['partitions']) for x in stages),'shell_old_leaf_identities':len(shell),'shell_gaps':len(result['gaps']),'new_rust_in_proof_logs':len(new),'unlogged_declared_framework_integration_cases':unexecuted,'changed_existing_rust_and_manifests':changed,'stage_partition_uniqueness':len({(p['owner'],p['new']) for x in stages for p in x['partitions']})==sum(len(x['partitions']) for x in stages),'final_source_mapping_complete':not result['gaps'],'final_coverage_equivalence':False,'reason':'Source mapped; conductor owns final frozen source/evidence hash renewal and execution evidence.'}
(P/'final-comparison.json').write_text(json.dumps(summary,indent=2,sort_keys=True)+'\n');print(json.dumps({k:v for k,v in summary.items() if k!='changed_existing_rust_and_manifests'}))
