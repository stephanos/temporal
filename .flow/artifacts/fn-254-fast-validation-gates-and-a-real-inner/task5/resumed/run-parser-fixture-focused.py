from pathlib import Path
import os,subprocess,time,json,hashlib,shutil,datetime,signal,re
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';scratch=r/'.flow/tmp/fn2546-resumed';candidate=scratch/'candidate';target=scratch/'target';tmp=Path('/tmp/umpire-fn2545-parser-fixture-controls');tmp.mkdir(exist_ok=True);revision='41daa86dcae4c047bd9ef741d135364deedb2045'
manifest=json.loads((out/'source-final-after.json').read_text())['source_files'];current={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in manifest};assert current==manifest
start_size=int(subprocess.run(['du','-sb',str(target)],text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0]);env=os.environ.copy();env.pop('BASH_ENV',None);env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='2',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(target),UMPIRE_BACKENDS='skip',UMPIRE_APALACHE='skip',UMPIRE_QUICK_CACHE='0',UMPIRE_CACHE_DISABLE='1')
def run(name,cmd,cwd):
 started=time.monotonic();receipt={'command':cmd,'cwd':str(cwd),'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','RUST_TEST_THREADS','UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE']},'BASH_ENV':'removed','cache_initial_bytes':start_size}
 with (out/(name+'.log')).open('w') as f:
  child=subprocess.Popen(cmd,cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
  while True:
   try:code=child.wait(timeout=2);break
   except subprocess.TimeoutExpired:
    delta=int(subprocess.run(['du','-sb',str(target)],text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0])-start_size
    if delta>512*1024**2:
     os.killpg(child.pid,signal.SIGTERM);child.wait();raise RuntimeError(f'Additional cache growth exceeded512MiB admission: {delta}')
 receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,cache_growth_bytes=int(subprocess.run(['du','-sb',str(target)],text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0])-start_size,free_bytes=shutil.disk_usage(r).free,log_sha256=hashlib.sha256((out/(name+'.log')).read_bytes()).hexdigest());(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(name,code,round(receipt['elapsed_seconds'],3),receipt['cache_growth_bytes'],flush=True);return code
assert not subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True)
assert run('parser-candidate-fetch',['git','fetch','origin','umpire'],candidate)==0
assert run('parser-candidate-checkout',['git','checkout','--detach',revision],candidate)==0
for path,sha in manifest.items():
 destination=candidate/path
 if sha=='deleted':destination.unlink(missing_ok=True)
 else:destination.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(r/path,destination)
snapshot={p:('deleted' if not (candidate/p).exists() else hashlib.sha256((candidate/p).read_bytes()).hexdigest()) for p in manifest};assert snapshot==manifest
locks={p:hashlib.sha256((r/p).read_bytes()).hexdigest() for p in ['model/Cargo.lock','model/umpire/Cargo.lock']};assert all(hashlib.sha256((candidate/p).read_bytes()).hexdigest()==sha for p,sha in locks.items());(out/'parser-candidate-source.json').write_text(json.dumps({'base_commit':revision,'source_files':snapshot,'lock_files':locks,'root_unchanged':True,'cache_initial_bytes':start_size,'free_before_bytes':shutil.disk_usage(r).free},indent=2)+'\n')
commands=[('parser-table',['mise','exec','--','cargo','test','--locked','-p','umpire-dsl-macros','--test','fixtures','parser_refusals::','--','--nocapture']),('fixture-identities',['mise','exec','--','cargo','test','--locked','-p','umpire-fixtures','--test','fixtures','--','--list']),('fixture-nextest-identities',['mise','exec','--','cargo','nextest','list','--locked','-p','umpire-fixtures','--test','fixtures','--message-format','json']),('fixture-consumers',['mise','exec','--','cargo','test','--locked','-p','umpire-testing','--test','consumers','--','--nocapture'])]
for name,command in commands:assert run(name,command,candidate/'model/umpire')==0
root_after={p:('deleted' if not (r/p).exists() else hashlib.sha256((r/p).read_bytes()).hexdigest()) for p in manifest};candidate_after={p:('deleted' if not (candidate/p).exists() else hashlib.sha256((candidate/p).read_bytes()).hexdigest()) for p in manifest};assert root_after==candidate_after==manifest
(out/'parser-fixture-source-after.json').write_text(json.dumps({'base_commit':revision,'source_files':candidate_after,'root_unchanged':True,'candidate_source_unchanged':True,'cache_growth_bytes':int(subprocess.run(['du','-sb',str(target)],text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0])-start_size,'free_bytes':shutil.disk_usage(r).free},indent=2)+'\n')
