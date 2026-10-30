from pathlib import Path
import os,subprocess,json,time,hashlib,shutil,datetime,signal,threading,re
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';candidate=r/'.flow/tmp/fn2546-resumed/candidate';target=r/'.flow/tmp/fn2546-resumed/target';gocache=r/'.flow/tmp/fn2545-temporal-api/go-cache';tmp=Path('/tmp/umpire-fn2545-temporal-api');sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();initial=json.loads((out/'temporal-api-prerequisite.json').read_text());paths=initial['current_task5_sources'];assert json.loads((out/'root-api-make-q.json').read_text())['exit_code']==0
assert all(('deleted' if not (r/p).exists() else sha(r/p))==h and ('deleted' if not (candidate/p).exists() else sha(candidate/p))==h for p,h in paths.items());assert all(sha(r/p)==sha(candidate/p)==h for group in ['declared_inputs83','generated_go_inputs40'] for p,h in initial[group].items());assert all(sha(r/p)==sha(candidate/p)==h for p,h in initial['locks'].items());assert sha(candidate/'proto/api.binpb')==initial['descriptor_reuse']['sha256']
realization=candidate/'model/temporal/realization';negative=sorted(str(p.relative_to(realization)) for p in (realization/'tests/ui').glob('registry_*.rs'));positive=sorted(str(p.relative_to(realization)) for p in (realization/'tests/ui-pass').glob('registry_*.rs'));assert len(negative)==18 and len(positive)==1
admission_files={str((realization/p).relative_to(candidate)):sha(realization/p) for p in negative+positive+[x[:-3]+'.stderr' for x in negative]};assert all(sha(r/p)==h for p,h in admission_files.items())
size=lambda p:int(subprocess.run(['du','-sb',str(p)],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,check=False).stdout.split()[0]);env=os.environ.copy();env.pop('BASH_ENV',None)
for key in ['UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_OUTPUT','UMPIRE_TIME_STEPS','TRYBUILD','INSTA_UPDATE','UMPIRE_BACKENDS','UMPIRE_APALACHE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS']:
 env.pop(key,None)
env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='1',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',GOCACHE=str(gocache),GOFLAGS='-p=1',GOMAXPROCS='1',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(target))
(out/'temporal-registry-source-before.json').write_text(json.dumps({'source_files':paths,'registry_files':admission_files,'negative_cases':negative,'positive_cases':positive,'root_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip(),'candidate_base_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=candidate,text=True).strip(),'tool_versions':{tool:subprocess.check_output(['mise','exec','--',tool,'--version'],cwd=candidate/'model',env=env,text=True).strip() for tool in ['rustc','cargo']}},indent=2)+'\n')
def baseline():return {'rust':size(target),'go':size(gocache)}
def run(name,cmd,start,budget):
 started=time.monotonic();seen={};stop=threading.Event();logpath=out/(name+'.log');receipt={'command':cmd,'cwd':str(candidate/'model'),'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','GOCACHE','GOFLAGS','GOMAXPROCS']},'BASH_ENV':'removed','snapshot_updates':'TRYBUILD/INSTA_UPDATE removed','phase_cache_baseline_bytes':start,'phase_budget_bytes':budget,'free_before_bytes':shutil.disk_usage(r).free}
 def sample(root_pid):
  while not stop.is_set():
   infos={}
   for proc in Path('/proc').iterdir():
    if not proc.name.isdigit():continue
    try:
     args=[x.decode(errors='replace') for x in (proc/'cmdline').read_bytes().split(b'\0') if x];fields=(proc/'stat').read_text().rsplit(')',1)[1].split()
     if args:infos[int(proc.name)]={'pid':int(proc.name),'ppid':int(fields[1]),'argv':args}
    except (OSError,ValueError,IndexError):pass
   for pid,info in infos.items():
    ancestor=pid;visited=set()
    while ancestor in infos and ancestor not in visited:
     if ancestor==root_pid:
      if pid not in seen:seen[pid]={**info,'first_seconds':time.monotonic()-started}
      break
     visited.add(ancestor);ancestor=infos[ancestor]['ppid']
   stop.wait(.1)
 with logpath.open('w') as log:
  child=subprocess.Popen(cmd,cwd=candidate/'model',env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True);thread=threading.Thread(target=sample,args=(child.pid,),daemon=True);thread.start();peak=0
  try:
   while True:
    try:code=child.wait(timeout=2);break
    except subprocess.TimeoutExpired:
     growth=max(0,size(target)-start['rust'])+max(0,size(gocache)-start['go'])+logpath.stat().st_size;peak=max(peak,growth);free=shutil.disk_usage(r).free
     if growth>budget or logpath.stat().st_size>128*1024**2 or free<(budget-growth)+512*1024**2:
      os.killpg(child.pid,signal.SIGTERM);child.wait();receipt.update(stopped_for_resource=True,growth_bytes=growth,free_bytes=free);raise RuntimeError(f'{name} exceededadmission growth={growth}free={free}')
  finally:
   stop.set();thread.join(timeout=10);receipt.update(elapsed_seconds=time.monotonic()-started,rust_phase_growth_bytes=size(target)-start['rust'],go_phase_growth_bytes=size(gocache)-start['go'],peak_monitored_phase_growth_bytes=peak,free_after_bytes=shutil.disk_usage(r).free,owned_processes=list(seen.values()),log_sha256=sha(logpath));(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
 receipt['exit_code']=code;(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(name,code,round(receipt['elapsed_seconds'],3),receipt['rust_phase_growth_bytes']+receipt['go_phase_growth_bytes'],flush=True);return code
macro_start=baseline();assert run('temporal-macros-lib',['mise','exec','--','cargo','build','--locked','-j1','-p','umpire-temporal-realization-macros','--lib'],macro_start,512*1024**2)==0
assert size(target)-macro_start['rust']+size(gocache)-macro_start['go']<512*1024**2
registry_start=baseline();assert run('temporal-registry-list',['mise','exec','--','cargo','test','--locked','-j1','-p','umpire-temporal','--test','registry','registry_admission_','--','--list'],registry_start,int(1.5*1024**3))==0
listing=(out/'temporal-registry-list.log').read_text();names=re.findall(r'^(registry_admission_[^\n]+): test$',listing,re.M);assert sorted(names)==['registry_admission_accepts_valid_consumers','registry_admission_uses_actual_inventory_descriptor_edges_and_rust_types']
assert run('temporal-registry-admission',['mise','exec','--','cargo','test','--locked','-j1','-p','umpire-temporal','--test','registry','registry_admission_','--','--test-threads=1','--nocapture'],registry_start,int(1.5*1024**3))==0
assert all(('deleted' if not (r/p).exists() else sha(r/p))==h and ('deleted' if not (candidate/p).exists() else sha(candidate/p))==h for p,h in paths.items());assert all(sha(r/p)==sha(candidate/p)==h for p,h in admission_files.items());assert all(sha(r/p)==sha(candidate/p)==h for p,h in initial['locks'].items())
