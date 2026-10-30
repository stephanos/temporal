from pathlib import Path
import json,hashlib,re,collections,subprocess,shutil
out=Path(__file__).resolve().parent;root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');load=lambda p:json.loads(p.read_text());write=lambda name,x:(out/name).write_text(json.dumps(x,indent=2)+'\n');SHA=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
expected=['umpire-apalache','umpire-fixtures','umpire-kettle-guest','umpire-reference-kettle','umpire-suite','umpire-tla']
stdout=(out/'kettle-public-quick.stdout.log').read_text();stderr=(out/'kettle-public-quick.stderr.log').read_text();selection=json.loads(next(x for x in stdout.splitlines() if x.startswith('{"base":')));assert [p for w in selection['workspaces'] for p in w['packages']]==expected and len(selection['changed'])==1
ns={'__name__':'proof_import','__file__':str(out.parent/'admission-proof/equivalence.py')};exec(compile((out.parent/'admission-proof/equivalence.py').read_text(),str(out.parent/'admission-proof/equivalence.py'),'exec'),ns)
current,default=ns['nextest'](load(out/'kettle-current-nextest-all.stdout.log'),'framework');assert len(current)==280 and len(default)==278
old=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/inventory-renewal/kettle-quick-nextest-all.stdout.log';prior,_=ns['nextest'](load(old),'framework');assert current==prior
names=collections.Counter(re.findall(r'^\s*test (.+) \.\.\. ok$',stderr.split('quick: stage=doctest')[0],re.M));wanted=collections.Counter(r[4] for r in default);assert not wanted-names
extra=names-wanted;assert extra==collections.Counter({'tests/ui/omitted_action.rs':1,'tests/ui/omitted_method.rs':1})
assert re.search(r'Summary.*278 tests run: 278 passed.*2 skipped',stderr)
write('test-statuses.json',dict(scope='Actual nextest inventory joined to all-passed summary and direct libtest name output; nested UI cases retained separately',identities=[dict(workspace=r[0],package=r[1],kind=r[2],binary=r[3],name=r[4],ignored=r[5],status='ignored' if r[5] else 'passed',direct_pass_name_count=names[r[4]]) for r in sorted(current)],nested_ui_successes=dict(extra)))
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
custody={tuple((p['fd9'][k] for k in ['device','inode'])) for p in procs.values() if p.get('fd9') and p.get('scope_env',{}).get('UMPIRE_JVM_LEASE_FD')=='9'};assert len(custody)==1
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
assert discovery and maxowners==2 and maxj<=4
custody={tuple((p['fd9'][k] for k in ['device','inode'])) for p in procs.values() if p.get('fd9') and p.get('scope_env',{}).get('UMPIRE_JVM_LEASE_FD')=='9'};assert len(custody)==1
observation=dict(sample_period_seconds=.2,sampled_jvm_processes=len(unique),sampled_concurrent_jvms=maxj,sampled_concurrent_framework_owners=maxowners,overlap_receipt=overlap,sampled_max_individual_jvm_rss_kib=maxrss,sampled_max_summed_jvm_rss_kib=maxsum,meaning='RSS is sampled observation, not a hard peak or total-memory limit',discovery_unique=len({(j['pid'],j['start_time']) for j in discovery}),discovery_heap_cap_observed='-Xmx512m',model_and_discovery_heap_cap_observed='-Xmx512m',java_host_lease_capability_and_scope_locators_absent_in_all_samples=True,java_fd9_reuse_note="Some JVMs reopen numeric slot9 for sockets; no sampled slot9 matches the host lease dev/inode",owned_fd9_device_inode=[list(x) for x in custody],theoretical_policy='required global4 / JVM group2 / per-check recovery2;4heap-capped checker JVMs',cleanup=load(out/'owned-cleanup.json'))
write('jvm-observations.json',observation)
before=load(out/'full-compile-inputs-before.json');after={p:SHA(root/p) for p in before};assert before==after;write('full-compile-inputs-after.json',after)
snap=load(root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed/temporal-activity-required.json')['snapshots_before'];assert len(snap)==202 and all(SHA(root/p)==h for p,h in snap.items());write('protected-snapshots.json',snap)
ns2={'__name__':'proof_import','__file__':str(out.parent/'admission-proof/driver.py')};exec(compile((out.parent/'admission-proof/driver.py').read_text(),str(out.parent/'admission-proof/driver.py'),'exec'),ns2);final=ns2['snapshot'](root);assert final==load(out/'root-source-before.json');write('root-source-final.json',final)
binaries=load(out/'successful-build.json')['binaries'];assert all(SHA(Path(p))==h for p,h in binaries.items());assert subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)==''
cachebefore=load(out/'quick-cache-before.json');cacheafter=load(out/'quick-cache-after.json');assert cachebefore==cacheafter
kettle=[x.strip() for x in stderr.splitlines() if 'TLC cache' in x and ' Kettle ' in x];assert sum('cache hit' in x for x in kettle)==4 and sum('cache miss' in x and 'configuration 2 ' in x for x in kettle)==2
negative='b84a5c5c360eab7780df01a92bd07ce91b99acf247f247963671aa3e552bcb74';assert all(negative not in p for p in cacheafter['files'])
size=lambda p:int(subprocess.check_output(['du','-sb',str(p)],text=True).split()[0]);target=root/'.flow/tmp/fn2546-resumed/target';go=root/'.flow/tmp/fn2545-temporal-api/go-cache';growth=size(target)+size(go)-load(out/'resource-baseline.json')['target_bytes'];assert growth<=3*1024**3 and shutil.disk_usage(root).free>=8*1024**3
summary=dict(root_head=final['head'],scope='F3.2 one public Kettle quick plus Linux native fixture resource/admission proof',native_resource_receipt='resources/summary.json',quick_exit=0,quick_wall_seconds=load(out/'kettle-public-quick.json')['elapsed_seconds'],stages=re.findall(r'quick: stage=(\S+).*?elapsed=([\d.]+)s exit=(\S+)',stderr),selection_packages=expected,nextest=dict(identities=280,passed=278,ignored=2,actual_current_identity_set_equals_F2=True,actual_status_receipt='test-statuses.json'),doctests=dict(packages=['umpire-apalache','umpire-reference-kettle','umpire-suite','umpire-tla'],passed=1,ignored=1,failed=0,meaning='Cargo output includes a compile-fail doctest with expected compiler diagnostic; phase exit0'),cache=dict(mode='existing retained cache; no clears, renames or cold/warm campaign',files_before=len(cachebefore['files']),files_after=len(cacheafter['files']),all_file_hashes_unchanged=True,total_hit_labels=stderr.count('TLC cache hit '),total_miss_labels=stderr.count('TLC cache miss '),total_execution_labels=stderr.count('TLC execute '),kettle_labels=kettle,kettle_execution_labels=[x.strip() for x in stderr.splitlines() if 'TLC execute Kettle ' in x],fresh_negative_configuration_2_misses=2,negative_configuration_absent_from_cache=True),native_interpreter_agreement_log_lines=stderr.count('quick native/interpreter agreement '),kettle_native_interpreter=[x.strip() for x in stderr.splitlines() if 'quick native/interpreter agreement' in x and 'sample_kettle' in x],jvm_observations=observation,source_binaries_locks_snapshots_unchanged=True,root_clean=True,registry_tuples_count=len(final['registry_tuples']),protected_snapshots=202,model_restoration=load(out/'model-restoration.json'),combined_cache_growth_bytes=growth,free_bytes=shutil.disk_usage(root).free,limits='3GiB combined phase growth /8GiB free guard; compiler1,Go1,incremental0; production quick strips CARGO_PROFILE overrides',not_run=['Activity semantic quick','Production standalone','Any new historical baseline or cold/warm campaign','Darwin portable completion proof'])
write('summary.json',summary);print(json.dumps({k:v for k,v in summary.items() if k not in ['jvm_observations','cache']},indent=2));print(json.dumps(observation,indent=2))
