from pathlib import Path
import os,subprocess,time,json,hashlib,datetime,re,sys
root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal'); out=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task6/resumed'; scratch=root/'.flow/tmp/fn2546-resumed'; candidate=scratch/'candidate'
env=os.environ.copy(); env.pop('BASH_ENV',None); env.update(CARGO_TARGET_DIR=str(scratch/'target'),TMPDIR=str(scratch/'tmp'),CARGO_INCREMENTAL='0',CARGO_BUILD_JOBS='2',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(candidate)+':'+str(scratch/'target'))
def run(name,cmd,cwd):
 start=time.monotonic(); receipt={'command':cmd,'cwd':str(cwd),'head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=cwd,text=True).strip(),'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'environment_overrides':{k:env[k] for k in ['CARGO_TARGET_DIR','TMPDIR','CARGO_INCREMENTAL','CARGO_BUILD_JOBS','RUST_TEST_THREADS','MISE_TRUSTED_CONFIG_PATHS']}}
 with (out/(name+'.log')).open('w') as f: p=subprocess.run(cmd,cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT)
 receipt.update(exit_code=p.returncode,elapsed_seconds=time.monotonic()-start,log_sha256=hashlib.sha256((out/(name+'.log')).read_bytes()).hexdigest());(out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n');print(name,p.returncode,round(receipt['elapsed_seconds'],3),flush=True);return p.returncode
assert run('quick-controls-current',['mise','exec','--','cargo','test','--locked','-p','xtask','--lib','quick::'],root/'model/umpire')==0
assert run('cache-controls-current',['mise','exec','--','cargo','test','--locked','-p','umpire-tlc','--lib','cache','--','--nocapture'],root/'model/umpire')==0
assert run('jar-admission-corrected',['bash','model/scripts/fetch-tlc.sh'],candidate)==0
source=candidate/'model/umpire/dev/fixtures/fixtures/sample_kettle/model.rs';original=source.read_bytes(); new=original.replace(b'//! Model authored with the attribute surface.',b'//! Model authored with the attribute surface (quick content-edit proof).',1);assert new!=original
try:
 source.write_bytes(new)
 for mode in (sys.argv[1:] or ['cold','warm']):
  code=run('kettle-'+mode,['mise','exec','--','cargo','xtask','quick'],candidate/'model/umpire')
  log=(out/('kettle-'+mode+'.log')).read_text();selection=next((json.loads(x) for x in log.splitlines() if x.startswith('{"base":')),None)
  facts={'exit_code':code,'selection':selection,'native_interpreter':log.count('quick native/interpreter agreement '),'cache_hits':log.count('TLC cache hit '),'cache_misses':log.count('TLC cache miss '),'tlc_executions':log.count('TLC execute '),'original_sha256':hashlib.sha256(original).hexdigest(),'edited_sha256':hashlib.sha256(new).hexdigest(),'stages':re.findall(r'quick: stage=(\w+) workspace=.* packages=([^\n]+?) elapsed=([0-9.]+)s exit=([^\n]+)',log)}
  (out/('kettle-'+mode+'-facts.json')).write_text(json.dumps(facts,indent=2)+'\n');assert code==0
  packages=[p for w in selection['workspaces'] for p in w['packages']];assert set(['umpire-apalache','umpire-fixtures','umpire-kettle-guest','umpire-reference-kettle','umpire-suite','umpire-tla']) <= set(packages),packages
finally:
 source.write_bytes(original)
 (out/'candidate-restored.json').write_text(json.dumps({'status':subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True),'source_sha256':hashlib.sha256(source.read_bytes()).hexdigest()},indent=2)+'\n')
