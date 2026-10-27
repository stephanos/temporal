from pathlib import Path
import os,sys,subprocess,shutil,time,signal,json,datetime,hashlib
root=Path.cwd();out=Path(__file__).resolve().parent;target=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2546-resumed/target');tmp=Path('/tmp/umpire-fn25417-graph');tmp.mkdir(exist_ok=True)
du_races=[]
def size():
 result=subprocess.run(['du','-sk',str(target)],capture_output=True,text=True)
 if result.returncode and not (result.returncode==1 and result.stdout.strip() and all('No such file or directory' in line for line in result.stderr.splitlines())):
  raise RuntimeError('cache size measurement failed: '+result.stderr)
 if result.returncode:du_races.append(result.stderr)
 return int(result.stdout.split()[0])*1024
baseline_file=out/'compile-phase-baseline.json';baseline=json.loads(baseline_file.read_text()) if baseline_file.exists() else {'target_bytes':size(),'free_bytes':shutil.disk_usage(root).free,'created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat()};baseline_file.write_text(json.dumps(baseline,indent=2)+'\n')
mode=sys.argv[1];framework=root/'model/umpire';temporal=root/'model';cwd=framework
commands={'lib':['test','-p','xtask','--lib','--','--test-threads=1','--nocapture'],'graph':['test','-p','xtask','--test','main','gate_','--','--test-threads=1','--nocapture'],'standalone':['test','-p','xtask','--test','main','standalone::','--','--test-threads=1','--nocapture'],'temporal':['test','-p','temporal-xtask','--bin','temporal-xtask','--','--test-threads=1','--nocapture'],'clippy':['clippy','-p','xtask','--all-targets','--all-features','--','-D','warnings'],'clippy-temporal':['clippy','-p','temporal-xtask','--all-targets','--all-features','--','-D','warnings']}
if mode in ['temporal','clippy-temporal']:cwd=temporal
args=commands[mode];cmd=['mise','exec','--','cargo',args[0],'--locked','-j1',*args[1:]];env=os.environ.copy();env.pop('BASH_ENV',None)
for k in ['TRYBUILD','INSTA_UPDATE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS','UMPIRE_JVM_LEASE_FD','UMPIRE_GATE_SCOPE','UMPIRE_GATE_RECEIPTS','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_MANIFEST','UMPIRE_TOOL_ROOT']:env.pop(k,None)
env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(tmp),CARGO_BUILD_JOBS='1',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(root)+':'+str(tmp)+':'+str(target))
assert shutil.disk_usage(root).free>=8*1024**3,'ownerfreeguard8GiBfailedbeforestart';assert size()-baseline['target_bytes']<1536*1024**2,'ownerphasegrowthguardfailedbeforestart'
paths=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard'],text=True).splitlines()
source_paths=[p for p in paths if p.startswith(('model/umpire/dev/xtask/','model/dev/xtask/','model/umpire/backends/process/','model/umpire/dev/guest-driver/','model/umpire/scripts/','model/scripts/')) or p in ['model/Cargo.toml','model/Cargo.lock','model/umpire/Cargo.toml','model/umpire/Cargo.lock','mise.toml','model/umpire/mise.toml']]
source_before={p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in source_paths if Path(p).is_file()}
label=sys.argv[2] if len(sys.argv)>2 else mode;started=time.monotonic();breach=None;samples=[]
with (out/(label+'.log')).open('wb') as log:
 proc=subprocess.Popen(cmd,cwd=cwd,env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
 while proc.poll() is None:
  try:current=size()
  except Exception as error:
   breach={'measurement_error':str(error)};os.killpg(proc.pid,signal.SIGTERM)
   try:proc.wait(timeout=10)
   except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);proc.wait()
   break
  free=shutil.disk_usage(root).free;samples.append({'seconds':round(time.monotonic()-started,3),'target_growth_bytes':current-baseline['target_bytes'],'free_bytes':free})
  if free<8*1024**3 or current-baseline['target_bytes']>1536*1024**2 or time.monotonic()-started>300:
   breach=samples[-1];os.killpg(proc.pid,signal.SIGTERM)
   try:proc.wait(timeout=10)
   except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);proc.wait()
   break
  time.sleep(1)
 code=proc.wait()
source_after={p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in source_before if Path(p).is_file()}
receipt={'source_inputs_sha256':source_before,'source_unchanged':source_before==source_after,'runner_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'command':cmd,'cwd':str(cwd),'source_head':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG']},'BASH_ENV':'removed','phase_baseline':baseline,'phase_budget_bytes':1536*1024**2,'free_guard_bytes':8*1024**3,'exit_code':code,'guard_breach':breach,'du_transient_enoent_samples':du_races,'seconds':time.monotonic()-started,'samples':samples,'final_target_growth_bytes':size()-baseline['target_bytes'],'free_after_bytes':shutil.disk_usage(root).free,'log_sha256':hashlib.sha256((out/(label+'.log')).read_bytes()).hexdigest()};(out/(label+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(json.dumps({k:receipt[k] for k in ['exit_code','guard_breach','seconds','final_target_growth_bytes','free_after_bytes']}));sys.exit(code if code>=0 else 128-code)
