from pathlib import Path
import os,subprocess,json,time,hashlib,shutil,datetime,signal,threading,re
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';candidate=r/'.flow/tmp/fn2546-resumed/candidate';target=r/'.flow/tmp/fn2546-resumed/target';work=r/'.flow/tmp/fn2545-temporal-api';gocache=work/'go-cache';tmp=Path('/tmp/umpire-fn2545-temporal-api');work.mkdir(exist_ok=True);assert not gocache.exists();gocache.mkdir();tmp.mkdir(exist_ok=True)
files=['Makefile','go.mod','go.sum']+sorted(str(p.relative_to(r)) for p in (r/'proto/internal').rglob('*.proto'))+sorted(str(p.relative_to(r)) for p in (r/'cmd/tools/getproto').iterdir() if p.is_file());generated=sorted(str(p.relative_to(r)) for directory in ['api/testpilot','api/umpire'] for p in (r/directory).rglob('*.go'));assert len(files)==83 and len(generated)==40
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();inputs={p:sha(r/p) for p in files+generated};assert all((candidate/p).is_file() and sha(candidate/p)==h for p,h in inputs.items());manifest=json.loads((out/'parser-fixture-source-after.json').read_text());sources=manifest['source_files'];assert all(('deleted' if not (r/p).exists() else sha(r/p))==h and ('deleted' if not (candidate/p).exists() else sha(candidate/p))==h for p,h in sources.items())
locks={p:sha(r/p) for p in ['model/Cargo.lock','model/umpire/Cargo.lock']};assert locks==manifest['lock_files'] and all(sha(candidate/p)==h for p,h in locks.items());descriptor=r/'proto/api.binpb';assert descriptor.stat().st_size==241086 and sha(descriptor)=='0f8845ad30e19673abb7b5e7638cd32d4041d9ae7afe32318a673ac62eb4630d';destination=candidate/'proto/api.binpb';assert not destination.exists();destination.write_bytes(descriptor.read_bytes());assert sha(destination)==sha(descriptor)
prereqs=['go.mod','go.sum']+[p for p in files if p.startswith('proto/internal/')];latest=max((candidate/p).stat().st_mtime_ns for p in prereqs);assert destination.stat().st_mtime_ns>latest
size=lambda p:int(subprocess.run(['du','-sb',str(p)],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,check=False).stdout.split()[0]);start_target=size(target);start_go=size(gocache);env=os.environ.copy();env.pop('BASH_ENV',None)
for key in ['UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_OUTPUT','UMPIRE_TIME_STEPS','TRYBUILD','INSTA_UPDATE','UMPIRE_BACKENDS','UMPIRE_APALACHE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS']:
 env.pop(key,None)
env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='1',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',GOCACHE=str(gocache),GOFLAGS='-p=1',GOMAXPROCS='1',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(target))
make=subprocess.run(['make','-q','proto/api.binpb'],cwd=candidate,env=env,text=True,capture_output=True);(out/'temporal-api-make-q.log').write_text(make.stdout+make.stderr);assert make.returncode==0
goenv=subprocess.run(['mise','exec','--','go','env','-json','GOCACHE','GOMODCACHE','GOVERSION'],cwd=candidate,env=env,text=True,capture_output=True);assert goenv.returncode==0;go_environment=json.loads(goenv.stdout);assert go_environment['GOCACHE']==str(gocache)
receipt={'root_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip(),'candidate_base_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=candidate,text=True).strip(),'current_task5_sources':sources,'locks':locks,'declared_inputs83':{p:inputs[p] for p in files},'generated_go_inputs40':{p:inputs[p] for p in generated},'root_candidate_input_hashes_equal':True,'descriptor_reuse':{'source':str(descriptor),'candidate':str(destination),'bytes':descriptor.stat().st_size,'sha256':sha(destination),'source_mtime_ns':descriptor.stat().st_mtime_ns,'candidate_new_mtime_ns':destination.stat().st_mtime_ns,'newest_make_prerequisite_mtime_ns':latest,'copied_without_preserving_mtime':True,'make_q_exit_code':make.returncode,'scope':'Onlydescriptoracquisitionreused; required--linked-api Go linker stillruns.'},'go_environment':go_environment,'private_go_cache_initial_bytes':start_go,'rust_target_initial_bytes':start_target,'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','GOCACHE','GOFLAGS','GOMAXPROCS']},'BASH_ENV':'removed','free_before_bytes':shutil.disk_usage(r).free,'admitted_additional_bytes':3*1024**3}
cmd=['mise','exec','--','cargo','build','--locked','-p','umpire-temporal-api','--lib','-j1'];receipt['command']=cmd;receipt['cwd']=str(candidate/'model');receipt['started_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();logpath=out/'temporal-api-build.log';started=time.monotonic();stop=threading.Event();seen={};sample_count=0

def sample(root_pid):
 global sample_count
 with (out/'temporal-api-processes.jsonl').open('w') as log:
  while not stop.is_set():
   infos={}
   for proc in Path('/proc').iterdir():
    if not proc.name.isdigit():continue
    try:
     args=[x.decode(errors='replace') for x in (proc/'cmdline').read_bytes().split(b'\0') if x];stat=(proc/'stat').read_text().rsplit(')',1)[1].split()
     if args:infos[int(proc.name)]={'pid':int(proc.name),'ppid':int(stat[1]),'argv':args}
    except (OSError,IndexError,ValueError):pass
   def owned(pid):
    visited=set()
    while pid in infos and pid not in visited:
     if pid==root_pid:return True
     visited.add(pid);pid=infos[pid]['ppid']
    return False
   owned_infos=[x for pid,x in infos.items() if owned(pid)]
   for info in owned_infos:
    if info['pid'] not in seen:seen[info['pid']]={**info,'first_seconds':time.monotonic()-started}
   if owned_infos:log.write(json.dumps({'seconds':time.monotonic()-started,'processes':owned_infos},separators=(',',':'))+'\n');sample_count+=1
   stop.wait(0.1)
with logpath.open('w') as log:
 child=subprocess.Popen(cmd,cwd=candidate/'model',env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True);thread=threading.Thread(target=sample,args=(child.pid,),daemon=True);thread.start();peak_growth=0
 try:
  while True:
   try:code=child.wait(timeout=2);break
   except subprocess.TimeoutExpired:
    growth=max(0,size(target)-start_target)+max(0,size(gocache)-start_go)+logpath.stat().st_size;peak_growth=max(peak_growth,growth);free=shutil.disk_usage(r).free
    if growth>3*1024**3 or logpath.stat().st_size>128*1024**2 or free<(3*1024**3-growth)+512*1024**2:
     os.killpg(child.pid,signal.SIGTERM);child.wait();receipt.update(stopped_for_resources=True,growth_bytes=growth,free_bytes=free);raise RuntimeError(f'APIboundedbuildstopped growth={growth}free={free}')
 finally:
  stop.set();thread.join(timeout=10);receipt.update(elapsed_seconds=time.monotonic()-started,rust_target_growth_bytes=size(target)-start_target,private_go_cache_growth_bytes=size(gocache)-start_go,peak_monitored_additional_bytes=peak_growth,log_bytes=logpath.stat().st_size,free_after_bytes=shutil.disk_usage(r).free,process_samples=sample_count,owned_processes=list(seen.values()),log_sha256=sha(logpath));(out/'temporal-api-prerequisite.json').write_text(json.dumps(receipt,indent=2)+'\n')
receipt['exit_code']=code;(out/'temporal-api-prerequisite.json').write_text(json.dumps(receipt,indent=2)+'\n');print(json.dumps({k:receipt[k] for k in ['exit_code','elapsed_seconds','rust_target_growth_bytes','private_go_cache_growth_bytes','free_after_bytes']},indent=2));assert code==0
assert all(('deleted' if not (r/p).exists() else sha(r/p))==h and ('deleted' if not (candidate/p).exists() else sha(candidate/p))==h for p,h in sources.items());assert all(sha(r/p)==sha(candidate/p)==h for p,h in inputs.items());assert all(sha(r/p)==sha(candidate/p)==h for p,h in locks.items());assert sha(destination)==sha(descriptor)
