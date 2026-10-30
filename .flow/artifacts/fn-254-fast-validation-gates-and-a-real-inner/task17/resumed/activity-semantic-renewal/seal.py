from pathlib import Path
import json,hashlib,re,collections,subprocess,shutil
out=Path(__file__).resolve().parent;root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');load=lambda p:json.loads(p.read_text());write=lambda name,x:(out/name).write_text(json.dumps(x,indent=2)+'\n');SHA=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
build=load(out/'successful-build-projection.json');binaries=build['binaries']
expected=['umpire-temporal','umpire-temporal-api','umpire-temporal-guest','umpire-temporal-realization-macros','umpire-temporal-tests']
stdout=(out/'activity-public-quick.stdout.log').read_text();stderr=(out/'activity-public-quick.stderr.log').read_text();selection=json.loads(next(x for x in stdout.splitlines() if x.startswith('{"base":')));assert [p for w in selection['workspaces'] for p in w['packages']]==expected and len(selection['changed'])==1
reference=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/semantic-renewal/activity-equivalence-final.json';current={tuple(r) for r in load(reference)['actual_identity_set']};assert len(current)==148 and not any(r[-1] for r in current)
names=collections.Counter(re.findall(r'^\s*test (.+) \.\.\. ok$',stderr.split('quick: stage=doctest')[0],re.M));wanted=collections.Counter(r[4] for r in current);assert not wanted-names
extra=names-wanted;assert sum(extra.values())==25 and all(p.startswith(('tests/ui/','tests/ui-pass/')) for p in extra)
assert re.search(r'Summary.*148 tests run: 148 passed.*40 skipped',stderr)
write('test-statuses.json',dict(scope='Current actual run148/all-passed summary plus direct names matched to F3.1 actual Cargo/nextest logical tuple set; no redundant rebuild/list',inventory_projection_path=str(reference.relative_to(root)),inventory_projection_sha256=SHA(reference),identities=[dict(workspace=r[0],package=r[1],kind=r[2],binary=r[3],name=r[4],ignored=False,status='passed',direct_pass_name_count=names[r[4]]) for r in sorted(current)],selected_ignored=0,filtered_out_tests=40,filtered_out_binaries=2,nested_ui_successes=dict(extra)))
write('selection.json',selection)
events=[json.loads(x) for x in (out/'owned-process-samples.jsonl').read_text().splitlines()];procs={e['process']['pid']:e['process'] for e in events if e['kind']=='new-owned-process'};samples=[e for e in events if e['kind']=='jvm-sample']
def owner(j):
 candidate=None;seen=set();p=procs.get(j['ppid'])
 while p and p['pid'] not in seen:
  seen.add(p['pid']);args=p.get('argv',[])
  if '--exact' in args and '/debug/deps/' in args[0]:candidate=dict(case=args[args.index('--exact')+1],argv0=args[0])
  parent=procs.get(p['ppid']);pa=parent.get('argv',[]) if parent else []
  if candidate and pa and Path(pa[0]).name=='cargo-nextest' and pa[1:3]==['nextest','run']:
   return dict(pid=p['pid'],start_time=p['start_time'],case=candidate['case'],argv0=candidate['argv0'],fd9=p.get('fd9'),meaning='direct child of actual nextest run controller; case supplied by same ancestry')
  p=parent
 return None
custody={tuple((p['fd9'][k] for k in ['device','inode'])) for p in procs.values() if p.get('fd9') and p.get('scope_env',{}).get('UMPIRE_JVM_LEASE_FD')=='9' and p.get('argv',[None])[0] in binaries};assert len(custody)==1
unique={};maxowners=0;overlap=[];maxrss=0;maxsum=0;maxj=0;discovery=[]
for sample in samples:
 owners={};total=0
 for j in sample['jvms']:
  key=(j['pid'],j['start_time']);unique[key]=j;assert j.get('scope_env')=={}
  if j.get('fd9'):assert (j['fd9']['device'],j['fd9']['inode']) not in custody
  o=owner(j)
  if o:owners[o['pid']]=o
  a=j['argv'];kind='version' if '-version' in a or '--version' in a else 'discovery' if '-h' in a else 'model'
  if kind!='version':assert '-Xmx512m' in a
  if kind=='discovery':discovery.append(j)
  maxrss=max(maxrss,j.get('rss_kib',0));total+=j.get('rss_kib',0)
 if len(owners)>maxowners:maxowners=len(owners);overlap=[dict(at=sample['at'],owners=list(owners.values()))]
 maxsum=max(maxsum,total);maxj=max(maxj,len(sample['jvms']))
assert maxowners<=2 and maxj<=4
custody={tuple((p['fd9'][k] for k in ['device','inode'])) for p in procs.values() if p.get('fd9') and p.get('scope_env',{}).get('UMPIRE_JVM_LEASE_FD')=='9' and p.get('argv',[None])[0] in binaries};assert len(custody)==1
observation=dict(sample_period_seconds=.2,sampled_jvm_processes=len(unique),sampled_concurrent_jvms=maxj,sampled_concurrent_framework_owners=maxowners,overlap_receipt=overlap,sampled_max_individual_jvm_rss_kib=maxrss,sampled_max_summed_jvm_rss_kib=maxsum,meaning='RSS is sampled observation, not a hard peak or total-memory limit',discovery_unique=len({(j['pid'],j['start_time']) for j in discovery}),discovery_heap_cap_observed='-Xmx512m' if discovery else None,model_and_discovery_heap_cap_observed='-Xmx512m' if discovery else None,java_host_lease_capability_and_scope_locators_absent_in_all_samples=True,java_fd9_reuse_note="Some JVMs reopen numeric slot9 for sockets; no sampled slot9 matches the host lease dev/inode",owned_fd9_device_inode=[list(x) for x in custody],theoretical_policy='required global4 / JVM group2 / per-check recovery2;4heap-capped checker JVMs',cleanup=load(out/'owned-cleanup.json'))
observation.update(model_heap_cap_observed='-Xmx512m' if any('-h' not in j['argv'] and '-version' not in j['argv'] and '--version' not in j['argv'] for j in unique.values()) else None,discovery_observation_note='No discovery JVM caught by200ms sampler in this run; unchanged process/tool input hashes permit projection of prior Kettle24discovery cap observations. No additional Java probe ran.' if not discovery else 'Discovery argv samples retained')
write('jvm-observations.json',observation)

before=load(out/'full-compile-inputs-before.json');after={p:SHA(root/p) for p in before};assert before==after;write('full-compile-inputs-after.json',after)
snap=load(root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed/temporal-activity-required.json')['snapshots_before'];assert len(snap)==202 and all(SHA(root/p)==h for p,h in snap.items());write('protected-snapshots.json',snap)
all_snap={p:h for p,h in before.items() if p.endswith('.snap')};assert all(SHA(root/p)==h for p,h in all_snap.items());write('all-current-model-snapshots.json',all_snap)
ns={'__name__':'proof_import','__file__':str(out.parent/'admission-proof/driver.py')};exec(compile((out.parent/'admission-proof/driver.py').read_text(),str(out.parent/'admission-proof/driver.py'),'exec'),ns);final=ns['snapshot'](root);assert final==load(out/'root-source-before.json');assert final['binary_source_files']==build['source_files'];write('root-source-final.json',final)
assert all(SHA(Path(p))==h for p,h in binaries.items());assert subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)==''
cachebefore=load(out/'quick-cache-before.json');cacheafter=load(out/'quick-cache-after.json');assert all(cacheafter['files'].get(p)==h for p,h in cachebefore['files'].items())
labels=[x.strip() for x in stderr.splitlines() if 'TLC cache' in x];misses=[x for x in labels if 'cache miss Activity ' in x];hits=[x for x in labels if 'cache hit Activity ' in x];executions=[x.strip() for x in stderr.splitlines() if 'TLC execute Activity ' in x];assert len(misses)==7 and len(hits)==7 and len(executions)==14
added={p:h for p,h in cacheafter['files'].items() if p not in cachebefore['files']};write('cache-added-files.json',added)
size=lambda p:int(subprocess.check_output(['du','-sb',str(p)],text=True).split()[0]);target=root/'.flow/tmp/fn2546-resumed/target';go=root/'.flow/tmp/fn2545-temporal-api/go-cache';tmp=Path('/tmp/umpire-fn25417-activity-semantic-renewal');growth=size(target)+size(go)+size(tmp)-load(out/'resource-baseline.json')['target_bytes'];assert growth<=3*1024**3 and shutil.disk_usage(root).free>=8*1024**3
summary=dict(root_head=final['head'],scope='One actual public five-consumer Activity quick; no standalone',quick_exit=0,quick_wall_seconds=load(out/'activity-public-quick.json')['elapsed_seconds'],stages=re.findall(r'quick: stage=(\S+).*?elapsed=([\d.]+)s exit=(\S+)',stderr),selection_packages=expected,nextest=dict(passed=148,selected_ignored=0,filtered_out_tests=40,filtered_out_binaries=2,actual_nextest_test_seconds=201.389,identity_receipt='test-statuses.json',nested_ui_negative_cases=sum(p.startswith('tests/ui/') for p in extra),nested_ui_positive_cases=sum(p.startswith('tests/ui-pass/') for p in extra)),doctests=dict(packages=['umpire-temporal','umpire-temporal-api','umpire-temporal-realization-macros'],targets=3,passed=0,ignored=0,failed=0,meaning='All three Cargo doc targets executed successfully with zero doctest bodies'),cache=dict(mode='Existing retained cache; no clears, renames or repeated baseline campaign',files_before=len(cachebefore['files']),files_after=len(cacheafter['files']),prior_files_unchanged=True,new_files=len(added),miss_labels=misses,hit_labels=hits,execution_labels=executions,meaning='Seven new Activity configurations execute14records then hit within this single workload; no separate warm run'),native_interpreter_activity=[x.strip() for x in stderr.splitlines() if 'quick native/interpreter agreement' in x],jvm_observations=observation,model_restoration=load(out/'model-restoration.json'),source_binaries_locks_snapshots_unchanged=True,root_clean=True,registry_tuples_count=len(final['registry_tuples']),historical_protected_snapshots=202,all_current_tracked_model_snapshots=len(all_snap),combined_target_go_tmp_growth_bytes=growth,private_tmp_final_bytes=size(tmp),free_bytes=shutil.disk_usage(root).free,not_run=['Production standalone','Other full model gates','A new old baseline or repeat cold/warm campaign','Darwin portable cleanup completion'])
write('summary.json',summary);print(json.dumps({k:v for k,v in summary.items() if k not in ['cache','jvm_observations']},indent=2));print(json.dumps(observation,indent=2))
