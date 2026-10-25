#!/usr/bin/env python3
"""Read frozen Git/proof inputs; compare owner-produced post-freeze inventories, never run tests."""
import argparse,collections,hashlib,json,pathlib,re,subprocess,tomllib
ROOT=pathlib.Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
OUT=ROOT/'.flow/tmp/fn254-task10-ledger'
BASE='208610049cb93d2fb6cc107898497461f5ab4a6e'
ART='.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner'
P13=ART+'/task13'
def sha(b):return hashlib.sha256(b).hexdigest()
def git(path,commit=BASE):return subprocess.check_output(['git','show',commit+':'+path],cwd=ROOT)
def write(name,data): (OUT/name).write_text(json.dumps(data,indent=2,sort_keys=True)+'\n')
def key(x):return tuple(x[k] for k in ('package','target_kind','target','test','ignored'))
def build():
 inputs={}
 def proof(path):
  b=git(path);inputs[path]=sha(b);return json.loads(b)
 pointer=proof(P13+'/current-index.json');idx=proof(P13+'/'+pointer['index'])
 assert inputs[P13+'/'+pointer['index']]==pointer['index_sha256']
 prior=proof(P13+'/index.json');mapping=proof(P13+'/proof/identity-accounting.json')
 delta=proof(P13+'/revision1/proof/logical-identity-delta.json')
 assert inputs[P13+'/proof/identity-accounting.json']==prior['files']['proof/identity-accounting.json']
 assert inputs[P13+'/revision1/proof/logical-identity-delta.json']==idx['files']['proof/logical-identity-delta.json']
 rust=[x['after'] for x in mapping['preserved']]+mapping['added']
 oldframework=proof(P13+'/proof/framework-identities.json')['identities'];oldouter=proof(P13+'/proof/temporal-identities.json')['identities']
 assert collections.Counter(map(key,rust))==collections.Counter(map(key,oldframework+oldouter))
 for pkg,v in delta['current_driver_cli_main_binaries'].items():
  subset=[x for x in rust if x['package']==pkg and x['target_kind']=='test' and x['target']=='main']
  assert sorted(x['test'] for x in subset)==sorted(set(v['current'])-set(v['added']))
  rust += [dict(package=pkg,target_kind='test',target='main',test=name,ignored=False) for name in v['added']]
 assert len(rust)==930 and len(set(map(key,rust)))==930 and sum(x['ignored'] for x in rust)==4
 frameworkpkgs={x['package'] for x in oldframework}
 rows=[dict(old=x,new=None,expected_owner='framework' if x['package'] in frameworkpkgs else 'temporal') for x in sorted(rust,key=key)]
 make=git('Makefile').decode();stage_rows=[]
 temporal={'temporal-guest-freshness','source-boundary','temporal-protobuf-freshness','fmt','clippy','docs','tests','temporal-go-host'}
 mixed={'retired-names','timing-controls','protobuf-controls','wasm-guest'}
 for n,line in enumerate(make.splitlines(),1):
  m=re.search(r'call RUST_MODEL_STEP,([^)]*)',line)
  if m:stage_rows.append(dict(old=m[1],old_source={'path':'Makefile','line':n,'text':line},new=[],expected_owner='split' if m[1] in mixed else 'temporal' if m[1] in temporal else 'framework'))
 assert len(stage_rows)==23
 paths=['Makefile','model/scripts/test-time-gate.sh','model/scripts/test-model-validate.sh','model/scripts/test-model-wiring.sh','model/scripts/tests/proto-go.sh','model/scripts/tests/build-guests.sh','model/scripts/retired-names.sh','model/umpire/scripts/tests/build-guests.sh','model/umpire/scripts/test-capabilities.sh','model/umpire/scripts/tests/proto-tools.sh','model/umpire/scripts/tests/bootstrap.sh']
 source={}
 for path in paths:
  b=git(path);dest=OUT/'frozen-source'/path;dest.parent.mkdir(parents=True,exist_ok=True);dest.write_bytes(b);source[path]={'sha256':sha(b),'git_blob':subprocess.check_output(['git','rev-parse',BASE+':'+path],cwd=ROOT,text=True).strip()}
 ledgerpath=ROOT/'.worktrees/fn25410-gate'/ART/'task10/proof/shell-control-identity-ledger.json'
 ledgerbytes=ledgerpath.read_bytes();ledger=json.loads(ledgerbytes)
 shell=[]
 for row in ledger['preserved']:
  path,name=row['old'].split('::',1)
  shell.append(dict(old=row['old'],proposed_new=row['new'],source=source[path],source_path=path,granularity='aggregate' if name.startswith('combined-') else 'logical control'))
 def add(path,name,detail):shell.append(dict(old=path+'::'+name,proposed_new=None,source=source[path],source_path=path,source_case=detail,granularity='expanded case'))
 for conflict in ['absent','environment','command-line','makeflags','command-line-makeflags']:
  for mode in ['skip','require']:add('model/scripts/test-model-validate.sh','validate-'+conflict+'-apalache-'+mode,{'conflict':conflict,'apalache':mode,'status':0})
 for mode in ['unset','skip','require']:add('model/scripts/test-model-validate.sh','time-backends-'+mode,{'mode':mode,'status':0})
 for name,status in [('validate-gate-failure',2),('validate-benchmark-failure',2),('timer-benchmark-failure',37)]:add('model/scripts/test-model-validate.sh',name,{'status':status})
 for timed in [0,1]:
  for stage in ['fmt','clippy','docs','tests','fmt-framework','clippy-framework','docs-framework','tests-framework','wasm32-wasip1','core-wasm-clippy']:add('model/scripts/test-model-wiring.sh',f'{stage}-timed-{timed}',{'stage':stage,'timed':timed})
  for target in ['unset','relative','absolute']:
   for stage in ['builder-target','go-host','kettle-go-host']:add('model/scripts/test-model-wiring.sh',f'{stage}-target-{target}-timed-{timed}',{'stage':stage,'target':target,'timed':timed})
 symbols=['std::fs::read','std::time::Instant::now','std::thread::spawn','std::fs::File','std::time::SystemTime','std::thread::JoinHandle','std::path::Path::exists','std::thread::yield_now','std::fs::DirBuilder::new']
 for core in ['dsl','checker']:
  add('model/umpire/scripts/test-capabilities.sh',core+'-negative-compile',{'core':core,'compile':'reject'})
  for symbol in symbols:add('model/umpire/scripts/test-capabilities.sh',core+'-'+symbol,{'core':core,'required_diagnostic':symbol})
 for root in ['author/dsl/src/lib.rs','dev/xtask/src/main.rs','dev/guest-driver/src/main.rs','backends/native/build.rs','author/dsl/tests/main.rs','author/dsl/tests/ui.rs']:add('model/umpire/scripts/test-capabilities.sh','unsafe-override-'+root,{'root':root,'diagnostic':'E0453'})
 for name in ['deny-local-allow-accepted','deny-unaudited-rejected']:add('model/umpire/scripts/test-capabilities.sh',name,{'unsafe':name})
 for name in ['provision-pins','repeat-provision','successful-lock-cleanup','lock-refusal','generator-symlink-refusal','install-failure-status-42','failed-lock-cleanup','recovery-provision','bin-directory-symlink-refusal']:add('model/umpire/scripts/tests/proto-tools.sh',name,{'control':name})
 for name in ['guest-caller-paths-environment-driver','freshness-caller-paths-driver']:add('model/umpire/scripts/tests/bootstrap.sh',name,{'control':name})
 assert len(set(x['old'] for x in shell))==len(shell)
 for row in shell:
  path=row['source_path'];text=git(path).decode();row['frozen_source_anchor']={'path':'frozen-source/'+path,'first_line':1,'last_line':len(text.splitlines()),'case_selector':row.get('source_case',row['old'].split('::',1)[1])}

 task4=proof(ART+'/task4/consolidation-identities.json');snap=task4['snapshot_ui_sha256'];snapchecks=[]
 for path,digest in snap.items():snapchecks.append({'path':path,'expected':digest,'actual':sha(git(path))})
 assert all(x['expected']==x['actual'] for x in snapchecks)
 registry=[]
 for path in ['model/Cargo.lock','model/umpire/Cargo.lock']:
  b=git(path);inputs[path]=sha(b)
  registry += [(x['name'],x['version'],x['source'],x.get('checksum')) for x in tomllib.loads(b.decode())['package'] if x.get('source','').startswith('registry+')]
 registry=sorted(set(registry));assert len(registry)==119
 write('baseline.json',dict(schema_version=1,baseline_source_commit=BASE,proof_source_commit=idx['source_commit'],proof_pointer=pointer,input_sha256=inputs,frozen_source=source,rust=rows,ignored=[x for x in rust if x['ignored']],doctests=mapping['doctests_preserved_reference'],stages=stage_rows,shell=shell,shell_ledger_input={'path':str(ledgerpath.relative_to(ROOT)),'sha256':sha(ledgerbytes),'preserved':len(ledger['preserved']),'new_controls':ledger['new_controls']},snapshot_ui=snapchecks,registry=[dict(zip(['name','version','source','checksum'],x)) for x in registry],limitations=['Baseline source/proof accounting only; no execution or final ownership equivalence claimed.','Shell aggregate IDs combined-validate-controls and combined-wiring-controls remain parent groups; their expanded cases are separately inventoried and must not count twice.','Final owner must supply exact Rust and stage/shell migration mappings after source freeze; newly added controls cannot substitute missing baseline identities.']))
 write('final-inventory-template.json',dict(source_commit=None,rust=[],rust_mappings=[],new_rust=[],stages=[],shell=[],new_controls=[],notes=['rust: final exact {package,target_kind,target,test,ignored} rows','rust_mappings: {old:<baseline identity>,new:<final identity>} only for changed logical coordinates','stages: {old:<baseline stage>,new:<final stage>,owner:<framework|temporal>,source:<path:line>} one row per preserved stage; mixed splits need covered_cases and explicit no-overlap proof','shell: {old:<baseline shell ID>,new:<final shell ID>,owner:<framework|temporal>,source:<path:line>} exclude aggregate parent rows; new controls separate']))
 print(json.dumps({'rust':len(rust),'ignored':sum(x['ignored'] for x in rust),'stages':len(stage_rows),'shell_rows':len(shell),'shell_parent_groups':2,'snapshots':len(snapchecks),'registry':len(registry)}))
def compare(path):
 b=json.loads((OUT/'baseline.json').read_text());f=json.loads(pathlib.Path(path).read_text());issues=[]
 actual=collections.Counter(map(key,f['rust']));
 if not f.get('source_commit'):issues.append({'missing_final_source_commit':True})
 expected=collections.Counter(key(x['old']) for x in b['rust']);maps={key(x['old']):key(x['new']) for x in f.get('rust_mappings',[])}
 mapped=collections.Counter(maps.get(old,old) for old in expected)
 for target,n in mapped.items():
  if n!=1:issues.append({'many_old_rust_to_one_new':target,'occurrences':n})
 admitted=mapped+collections.Counter(map(key,f.get('new_rust',[])))
 for target,n in (actual-admitted).items():issues.append({'unclassified_new_rust':target,'occurrences':n})
 for target,n in (admitted-actual).items():issues.append({'missing_admitted_rust':target,'occurrences':n})
 for old,count in expected.items():
  new=maps.get(old,old)
  if actual[new]!=1:issues.append({'rust_old':old,'mapped_new':new,'occurrences':actual[new]})
 for new,n in actual.items():
  if n!=1:issues.append({'duplicate_final_rust':new,'occurrences':n})
 def check(kind,olds):
  for row in f[kind]:
   if not row.get('source') or row.get('owner') not in ['framework','temporal']:issues.append({kind+'_missing_ownership_or_source':row})
  c=collections.Counter(x['old'] for x in f[kind]);targets=collections.Counter((x.get('owner'),x['new']) for x in f[kind])
  for old in olds:
   if c[old]!=1:issues.append({kind+'_old':old,'occurrences':c[old]})
  for old in c.keys()-set(olds):issues.append({kind+'_unmatched_old':old})
  for target,n in targets.items():
   if n!=1:issues.append({kind+'_duplicate_new':target,'occurrences':n})
 check('stages',[x['old'] for x in b['stages']]);check('shell',[x['old'] for x in b['shell'] if x['granularity']!='aggregate'])
 write('comparison.json',{'source_commit':f.get('source_commit'),'issues':issues,'pass_exact_identity_accounting':not issues,'execution_claim':False,'note':'Mixed split stages need case decomposition before use; source anchors and semantic equivalence require owner verification.'});print(json.dumps({'issues':len(issues)}))
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--compare');a=p.parse_args();compare(a.compare) if a.compare else build()
