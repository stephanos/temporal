from pathlib import Path
import os,subprocess,time,json,hashlib,shutil,datetime,signal
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';out.mkdir(parents=True,exist_ok=True);target=r/'.flow/tmp/fn2546-selection-review/target';tmp=Path('/tmp/umpire-fn2545-xtask-controls');tmp.mkdir(exist_ok=True)
env=os.environ.copy();env.pop('BASH_ENV',None);env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='2',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false')
for key in ['UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_OUTPUT','UMPIRE_CACHE_TEST_ROOT','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE']:
 env.pop(key,None)
manifest=json.loads((r/'.flow/tmp/fn2545-source-integration/source-manifest.json').read_text());files={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in manifest['source_files']};assert all(files[p]==h for p,h in manifest['source_files'].items() if p!='model/umpire/dev/xtask/tests/gate_wiring.rs');start_size=int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0]);initial_free=shutil.disk_usage(r).free
(out/'source-final-before.json').write_text(json.dumps({'head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip(),'source_files':files,'cache_before_bytes':start_size,'free_before_bytes':initial_free,'forecast_growth_bytes':300*1024**2,'scope':'Uncommitted task5 integrated source; framework xtask only, synthetic gate leaves, refusal-only standalone. No real gates, Go, JVM, backend or Temporal builds.'},indent=2)+'\n')
def run(name,args):
 cmd=['mise','exec','--','cargo',*args];started=time.monotonic();receipt={'command':cmd,'cwd':str(r/'model/umpire'),'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','RUST_TEST_THREADS']},'BASH_ENV':'removed','started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat()}
 with (out/(name+'.log')).open('w') as f:
  child=subprocess.Popen(cmd,cwd=r/'model/umpire',env=env,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
  while True:
   try:code=child.wait(timeout=2);break
   except subprocess.TimeoutExpired:
    size=int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0]);growth=size-start_size
    if growth>600*1024**2:
     os.killpg(child.pid,signal.SIGTERM);child.wait();raise RuntimeError(f'Warm target growth exceeded twice300MiB forecast: {growth}')
 receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,cache_bytes=int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0]),free_bytes=shutil.disk_usage(r).free,log_sha256=hashlib.sha256((out/(name+'.log')).read_bytes()).hexdigest());(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(name,code,round(receipt['elapsed_seconds'],3),receipt['cache_bytes']-start_size,flush=True);return code
assert run('final-xtask-lib',['test','--locked','-p','xtask','--lib','--','--nocapture'])==0
assert run('final-xtask-main-list',['test','--locked','-p','xtask','--test','main','--','--list'])==0
assert run('final-xtask-gate-wiring',['test','--locked','-p','xtask','--test','main','gate_wiring::','--','--nocapture'])==0
assert run('final-xtask-standalone',['test','--locked','-p','xtask','--test','main','standalone::','--','--nocapture'])==0
final={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in files};assert final==files
(out/'source-final-after.json').write_text(json.dumps({'unchanged':True,'source_files':final,'cache_growth_bytes':int(subprocess.check_output(['du','-sb',str(target)],text=True).split()[0])-start_size,'free_bytes':shutil.disk_usage(r).free},indent=2)+'\n')
