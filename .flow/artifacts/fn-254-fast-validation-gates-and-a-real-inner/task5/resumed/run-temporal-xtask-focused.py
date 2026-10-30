from pathlib import Path
import os,subprocess,json,hashlib,time,datetime,shutil,signal
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');o=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';target=r/'.flow/tmp/fn2546-selection-review/target';tmp=Path('/tmp/umpire-fn2545-xtask-controls');tmp.mkdir(exist_ok=True)
manifest=json.loads((o/'source-final-after.json').read_text())['source_files'];before={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in manifest};assert before==manifest
size=int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0]);env=os.environ.copy();env.pop('BASH_ENV',None);env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='2',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false')
for key in ['UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_OUTPUT','UMPIRE_TIME_STEPS','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE']:
 env.pop(key,None)
cmd=['mise','exec','--','cargo','test','--locked','-p','temporal-xtask','--bin','temporal-xtask','--','--nocapture'];started=time.monotonic();receipt={'command':cmd,'cwd':str(r/'model'),'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip(),'uncommitted_candidate':True,'source_files':before,'cache_before_bytes':size,'free_before_bytes':shutil.disk_usage(r).free,'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','RUST_TEST_THREADS']},'BASH_ENV':'removed'}
with (o/'temporal-xtask.log').open('w') as f:
 child=subprocess.Popen(cmd,cwd=r/'model',env=env,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
 while True:
  try:code=child.wait(timeout=2);break
  except subprocess.TimeoutExpired:
   delta=int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0])-size
   if delta>600*1024**2:
    os.killpg(child.pid,signal.SIGTERM);child.wait();raise RuntimeError(f'Warm target growth exceeds twice300MiBforecast: {delta}')
after={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in before};assert before==after
receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,cache_growth_bytes=int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0])-size,free_after_bytes=shutil.disk_usage(r).free,source_unchanged=True,log_sha256=hashlib.sha256((o/'temporal-xtask.log').read_bytes()).hexdigest());(o/'temporal-xtask.json').write_text(json.dumps(receipt,indent=2)+'\n');print(json.dumps({k:receipt[k] for k in ['exit_code','elapsed_seconds','cache_growth_bytes','free_after_bytes','source_unchanged']},indent=2));assert code==0
