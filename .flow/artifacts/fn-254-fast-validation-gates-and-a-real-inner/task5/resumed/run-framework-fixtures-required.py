from pathlib import Path
import os,subprocess,json,time,hashlib,shutil,datetime,signal,re,threading
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';candidate=r/'.flow/tmp/fn2546-resumed/candidate';target=r/'.flow/tmp/fn2546-resumed/target';workspace=candidate/'model/umpire';tlc_scratch=workspace/'build/tlc';tmp=Path('/tmp/umpire-fn2545-required-fixtures');tmp.mkdir(exist_ok=True)
source=json.loads((out/'parser-fixture-source-after.json').read_text());paths=source['source_files'];rootfiles={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in paths};copy={p:('deleted' if not (candidate/p).exists() else hashlib.sha256((candidate/p).read_bytes()).hexdigest()) for p in paths};assert rootfiles==copy==paths
snapshot_paths=lambda base:{str(p.relative_to(base)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted((base/'model/umpire/dev/fixtures/fixtures').glob('*/expected/**/*')) if p.is_file()}
snapshots=snapshot_paths(candidate);root_snapshots=snapshot_paths(r);assert snapshots==root_snapshots
expected=set(re.findall(r'^(.+): test$',(out/'fixture-identities.log').read_text(),re.M));assert len(expected)==84
size=lambda p:int(subprocess.run(['du','-sb',str(p)],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,check=False).stdout.split()[0]);start_target=size(target);start_scratch=size(tlc_scratch);env=os.environ.copy();env.pop('BASH_ENV',None)
for key in ['UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_OUTPUT','UMPIRE_TIME_STEPS','TRYBUILD','INSTA_UPDATE']:
 env.pop(key,None)
env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='2',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(target),UMPIRE_BACKENDS='require',UMPIRE_APALACHE='skip',UMPIRE_QUICK_CACHE='0',UMPIRE_CACHE_DISABLE='1',NEXTEST_PROFILE='required',NEXTEST_TEST_THREADS='4')
cmd=['mise','exec','--','cargo','nextest','run','--locked','--profile','required','-p','umpire-fixtures','--test','fixtures','--status-level','all','--final-status-level','all','--success-output','immediate','--failure-output','immediate-final'];logpath=out/'framework-fixtures-required.log';started=time.monotonic();receipt={'command':cmd,'cwd':str(workspace),'root_head_before':subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip(),'candidate_base_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=candidate,text=True).strip(),'source_files':paths,'snapshots_before':snapshots,'source_root_candidate_equal':True,'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','RUST_TEST_THREADS','UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS']},'BASH_ENV':'removed','snapshot_update':'TRYBUILD/INSTA_UPDATE removed','started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'target_before_bytes':start_target,'scratch_before_bytes':start_scratch,'free_before_bytes':shutil.disk_usage(r).free,'admitted_growth_bytes':512*1024**2,'log_limit_bytes':128*1024**2,'sampler_interval_seconds':0.05,'rss_is_observed_not_a_hard_cap':True}
stop=threading.Event();seen={};maxima={'test_owners':0,'jvm_owners':0,'checker_jvms':0,'all_jvms':0,'aggregate_jvm_rss_bytes':0};samples=0;peaks=[]
def sampler(root_pid):
 global samples
 with (out/'framework-fixtures-process-samples.jsonl').open('w') as output:
  while not stop.is_set():
   infos={}
   for proc in Path('/proc').iterdir():
    if not proc.name.isdigit():continue
    try:
     args=[x.decode(errors='replace') for x in (proc/'cmdline').read_bytes().split(b'\0') if x]
     if not args:continue
     fields=(proc/'stat').read_text().rsplit(')',1)[1].split();infos[int(proc.name)]={'pid':int(proc.name),'ppid':int(fields[1]),'argv':args}
    except (OSError,ValueError,IndexError):continue
   def owner(pid):
    visited=set()
    while pid in infos and pid not in visited:
     visited.add(pid);info=infos[pid]
     if Path(info['argv'][0]).name.startswith('fixtures-'):
      name=next((arg for arg in info['argv'][1:] if arg in expected),None)
      if name:return name
     pid=info['ppid']
    return None
   owners=[];jvms=[]
   for pid,info in infos.items():
    exe=Path(info['argv'][0]).name
    if exe.startswith('fixtures-'):
     case=owner(pid)
     if case and next((arg for arg in info['argv'][1:] if arg in expected),None):owners.append({'pid':pid,'case':case})
    if exe!='java':continue
    case=owner(pid)
    if not case:
     try:
      if not Path(os.readlink(f'/proc/{pid}/cwd')).is_relative_to(candidate):continue
     except OSError:continue
    try:
     status=Path(f'/proc/{pid}/status').read_text();match=re.search(r'^VmRSS:\s+(\d+) kB',status,re.M);rss=int(match[1])*1024 if match else 0
    except OSError:rss=0
    args=info['argv'];kind='tlc-discovery' if 'tlc2.TLC' in args and '-h' in args else 'tlc-checker' if 'tlc2.TLC' in args else 'java-other';entry={**info,'owner_case':case,'kind':kind,'heap_flags':[arg for arg in args if arg.startswith('-Xmx')],'rss_bytes':rss};jvms.append(entry)
    previous=seen.get(pid)
    if previous is None:seen[pid]={**entry,'first_seconds':time.monotonic()-started,'peak_rss_bytes':rss}
    else:previous.update(peak_rss_bytes=max(previous['peak_rss_bytes'],rss))
   checker=sum(x['kind']=='tlc-checker' for x in jvms);jvmowners={x['owner_case'] for x in jvms if x['owner_case']};aggregate=sum(x['rss_bytes'] for x in jvms);counts={'test_owners':len(owners),'jvm_owners':len(jvmowners),'checker_jvms':checker,'all_jvms':len(jvms),'aggregate_jvm_rss_bytes':aggregate};newpeak=any(counts[key]>maxima[key] for key in counts)
   for key in counts:maxima[key]=max(maxima[key],counts[key])
   if owners or jvms:
    record={'seconds':time.monotonic()-started,'test_owners':owners,'jvms':jvms,'counts':counts};output.write(json.dumps(record,separators=(',',':'))+'\n');samples+=1
    if newpeak:peaks.append(record)
   stop.wait(0.05)
with logpath.open('w') as log:
 child=subprocess.Popen(cmd,cwd=workspace,env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True);thread=threading.Thread(target=sampler,args=(child.pid,),daemon=True);thread.start();peak_growth=0
 try:
  while True:
   try:code=child.wait(timeout=2);break
   except subprocess.TimeoutExpired:
    growth=max(0,size(target)-start_target)+max(0,size(tlc_scratch)-start_scratch)+logpath.stat().st_size;peak_growth=max(peak_growth,growth);free=shutil.disk_usage(r).free
    if growth>512*1024**2 or logpath.stat().st_size>128*1024**2 or free<(512*1024**2-growth)+512*1024**2:
     os.killpg(child.pid,signal.SIGTERM);child.wait();receipt.update(stopped_for_resource=True,growth_bytes=growth,free_bytes=free);raise RuntimeError(f'Bounded requiredfixture resource admission stopped: growth={growth}free={free}')
 finally:
  stop.set();thread.join(timeout=10)
  receipt.update(elapsed_seconds=time.monotonic()-started,target_growth_bytes=size(target)-start_target,scratch_growth_bytes=size(tlc_scratch)-start_scratch,log_bytes=logpath.stat().st_size,peak_monitored_growth_bytes=peak_growth,free_after_bytes=shutil.disk_usage(r).free,log_sha256=hashlib.sha256(logpath.read_bytes()).hexdigest());(out/'framework-fixtures-required.json').write_text(json.dumps(receipt,indent=2)+'\n');(out/'framework-fixtures-process-summary.json').write_text(json.dumps({'samples':samples,'maxima':maxima,'java_processes':list(seen.values()),'peak_samples':peaks,'rss_is_observed_not_a_hard_cap':True},indent=2)+'\n')
receipt['exit_code']=code;(out/'framework-fixtures-required.json').write_text(json.dumps(receipt,indent=2)+'\n');print(json.dumps({'exit_code':code,'elapsed_seconds':receipt['elapsed_seconds'],'maxima':maxima,'java_process_count':len(seen),'free_after_bytes':receipt['free_after_bytes']},indent=2));assert code==0
source_after={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in paths};candidate_after={p:('deleted' if not (candidate/p).exists() else hashlib.sha256((candidate/p).read_bytes()).hexdigest()) for p in paths};assert candidate_after==paths;assert source_after==paths;assert snapshot_paths(candidate)==snapshots and snapshot_paths(r)==root_snapshots
