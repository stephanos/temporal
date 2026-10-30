from pathlib import Path
import os,subprocess,json,time,datetime,hashlib,re,shutil,signal
root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');out=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task6/resumed';scratch=root/'.flow/tmp/fn2546-resumed';candidate=scratch/'candidate';target=scratch/'target';revision='0c83ab3d1e4b8593e7415d780902789d8bad7c90'
env=os.environ.copy();env.pop('BASH_ENV',None);env.update(CARGO_TARGET_DIR=str(target),TMPDIR=str(scratch/'tmp'),CARGO_INCREMENTAL='0',CARGO_BUILD_JOBS='2',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(target))
def run(name,cmd,cwd):
 assert shutil.disk_usage(root).free>30*1024**3
 start=time.monotonic();receipt={'command':cmd,'cwd':str(cwd),'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'environment':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_INCREMENTAL','CARGO_BUILD_JOBS','RUST_TEST_THREADS','MISE_TRUSTED_CONFIG_PATHS']},'BASH_ENV':'removed'}
 with (out/(name+'.log')).open('w') as f:
  child=subprocess.Popen(cmd,cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
  while True:
   try:code=child.wait(timeout=2);break
   except subprocess.TimeoutExpired:
    if shutil.disk_usage(root).free<30*1024**3:
     os.killpg(child.pid,signal.SIGTERM);child.wait();raise RuntimeError('disk fell below30GiB; stopped admitted command')
 receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-start,log_sha256=hashlib.sha256((out/(name+'.log')).read_bytes()).hexdigest(),free_bytes=shutil.disk_usage(root).free);(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(name,code,round(receipt['elapsed_seconds'],3),flush=True);return code
assert not subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True)
assert run('joined-fetch',['git','fetch','origin','umpire'],candidate)==0
assert run('joined-checkout',['git','checkout','--detach',revision],candidate)==0
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=candidate,text=True).strip()==revision
assert not subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True)
paths=['model/umpire/dev/xtask/src/quick/mod.rs','model/umpire/dev/xtask/src/quick/selection.rs','model/umpire/dev/xtask/src/quick/tests.rs','model/umpire/backends/tlc/src/runner.rs','model/umpire/backends/tlc/src/runner/cache_tests.rs','model/dev/xtask/src/quick/mod.rs','model/umpire/Cargo.lock','model/Cargo.lock']
source_facts={p:hashlib.sha256((candidate/p).read_bytes()).hexdigest() for p in paths};assert all(source_facts[p]==hashlib.sha256((root/p).read_bytes()).hexdigest() for p in paths)
(out/'joined-source.json').write_text(json.dumps({'commit':revision,'files':source_facts,'integrated_source_equal':True,'compiler_cache':'warm reused task target','tlc_cache':'previous cache retained and new empty root starts cold'},indent=2)+'\n')
oldcache=target/'quick-cache';backup=target/'quick-cache-before-joined-0c83ab3d1e';assert not backup.exists()
if oldcache.exists():oldcache.rename(backup)
assert run('joined-metadata',['mise','exec','--','cargo','metadata','--locked','--format-version','1'],candidate/'model/umpire')==0
source=candidate/'model/umpire/dev/fixtures/fixtures/sample_kettle/model.rs';original=source.read_bytes();edit=original.replace(b'//! Model authored with the attribute surface.',b'//! Model authored with the attribute surface (joined shipping content-edit proof).',1);assert edit!=original
dirty=candidate/'.flow/artifacts/fn254-joined-dirty-evidence.json';assert not dirty.exists();dirty.write_text('{"proof":"known non-model dirty evidence must not expand selection"}\n')
expected=['umpire-apalache','umpire-fixtures','umpire-kettle-guest','umpire-reference-kettle','umpire-suite','umpire-tla']
try:
 source.write_bytes(edit)
 for mode in ['cold','warm']:
  name='joined-kettle-'+mode;code=run(name,['mise','exec','--','cargo','xtask','quick'],candidate/'model/umpire');log=(out/(name+'.log')).read_text();selection=next(json.loads(x) for x in log.splitlines() if x.startswith('{"base":'));stages=re.findall(r'quick: stage=(\w+) workspace=.* packages=([^\n]+?) elapsed=([0-9.]+)s exit=([^\n]+)',log)
  fixture=re.search(r'quick: stage=test [^\n]*packages=umpire-fixtures command=.*?(?=quick: stage=test [^\n]*packages=umpire-fixtures elapsed)',log,re.S)
  facts={'commit':revision,'exit_code':code,'selection':selection,'stages':stages,'native_interpreter':log.count('quick native/interpreter agreement '),'kettle_native_interpreter':[x for x in log.splitlines() if 'quick native/interpreter agreement fixtures::sample_kettle' in x],'cache_hits':log.count('TLC cache hit '),'cache_misses':log.count('TLC cache miss '),'executions':log.count('TLC execute '),'kettle_cache_hits':log.count('TLC cache hit Kettle'),'kettle_cache_misses':log.count('TLC cache miss Kettle'),'kettle_executions':log.count('TLC execute Kettle'),'original_sha256':hashlib.sha256(original).hexdigest(),'edited_sha256':hashlib.sha256(edit).hexdigest(),'fixture_summaries':re.findall(r'test result: [^\n]+',fixture.group()) if fixture else [],'fixture_named_tests':re.findall(r'^test ([^\n]+) \.\.\. (ok|ignored[^\n]*)$',fixture.group(),re.M) if fixture else []}
  (out/(name+'-facts.json')).write_text(json.dumps(facts,indent=2)+'\n');assert code==0
  assert [p for w in selection['workspaces'] for p in w['packages']]==expected
  assert [p for stage,p,elapsed,exit in stages if stage=='test']==expected
  assert all(exit=='0' for stage,p,elapsed,exit in stages)
  assert any('ignored known non-model input:' in reason and str(dirty) in reason for reason in selection['reasons'])
  assert facts['kettle_native_interpreter'] and facts['kettle_executions']>0
  if mode=='cold':assert facts['kettle_cache_hits']>=2 and facts['kettle_cache_misses']>=4
  else:assert facts['kettle_cache_hits']>=4 and facts['kettle_cache_misses']==2
finally:
 source.write_bytes(original);dirty.unlink()
 status=subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True)
 (out/'joined-restoration.json').write_text(json.dumps({'commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=candidate,text=True).strip(),'status':status,'source_sha256':hashlib.sha256(source.read_bytes()).hexdigest(),'source_restored':source.read_bytes()==original,'free_bytes':shutil.disk_usage(root).free},indent=2)+'\n');assert not status
