from pathlib import Path
import os,subprocess,json,time,hashlib,shutil,signal,datetime
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';candidate=r/'.flow/tmp/fn2546-resumed/candidate';target=r/'.flow/tmp/fn2546-resumed/target';gocache=r/'.flow/tmp/fn2545-temporal-api/go-cache';env=os.environ.copy();env.pop('BASH_ENV',None)
for key in ['TRYBUILD','INSTA_UPDATE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS','UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_OUTPUT']:
 env.pop(key,None)
env.update(CARGO_TARGET_DIR=str(target),TMPDIR='/tmp/umpire-fn2545-temporal-activity',CARGO_BUILD_JOBS='1',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',GOCACHE=str(gocache),GOFLAGS='-p=1',GOMAXPROCS='1',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(target))
size=lambda p:int(subprocess.run(['du','-sb',str(p)],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,check=False).stdout.split()[0]);baseline={'rust':size(target),'go':size(gocache)};(out/'current-membership-phase-baseline.json').write_text(json.dumps(baseline,indent=2)+'\n')
def run(name,cmd,workspace):
 started=time.monotonic();receipt={'command':cmd,'cwd':str(workspace),'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'phase_baseline_bytes':baseline,'phase_budget_bytes':512*1024**2,'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','GOCACHE','GOFLAGS','GOMAXPROCS']},'BASH_ENV':'removed','execution_scope':'nextestshow-config/listonly; testbodiesnotexecuted'}
 with (out/(name+'.log')).open('w') as f:
  child=subprocess.Popen(cmd,cwd=workspace,env=env,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
  while True:
   try:code=child.wait(timeout=2);break
   except subprocess.TimeoutExpired:
    growth=max(0,size(target)-baseline['rust'])+max(0,size(gocache)-baseline['go']);free=shutil.disk_usage(r).free
    if growth>512*1024**2 or free<(512*1024**2-growth)+512*1024**2:
     os.killpg(child.pid,signal.SIGTERM);child.wait();raise RuntimeError(f'membershipresourcebudget exceeded:growth={growth},free={free}')
 receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,rust_phase_growth_bytes=size(target)-baseline['rust'],go_phase_growth_bytes=size(gocache)-baseline['go'],free_after_bytes=shutil.disk_usage(r).free,log_sha256=hashlib.sha256((out/(name+'.log')).read_bytes()).hexdigest());(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(name,code,round(receipt['elapsed_seconds'],3),receipt['rust_phase_growth_bytes']+receipt['go_phase_growth_bytes'],flush=True);return code
framework=candidate/'model/umpire';temporal=candidate/'model';scopes=[('framework-backend',['-p','umpire-tla','-p','umpire-apalache','-p','umpire-tlc','--test','main'],framework),('framework-testing',['-p','umpire-testing','--lib','-E','test(actual_tlc_fact_traces_replay_exact_vectors)'],framework),('framework-fixtures',['-p','umpire-fixtures','--test','fixtures'],framework),('temporal-fixtures',['-p','umpire-temporal-tests','--test','fixtures'],temporal)]
for name,scope,workspace in scopes:
 assert run('current-'+name+'-groups',['mise','exec','--','cargo','nextest','show-config','test-groups','--locked','--profile','required','--run-ignored','all','--groups','jvm','--no-pager',*scope],workspace)==0
 assert run('current-'+name+'-identities',['mise','exec','--','cargo','nextest','list','--locked','--profile','required','--message-format','json',*scope],workspace)==0
