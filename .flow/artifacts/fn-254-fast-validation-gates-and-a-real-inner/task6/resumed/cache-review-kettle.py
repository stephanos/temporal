from pathlib import Path
import os,subprocess,re,json,time,hashlib
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');o=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task6/resumed';s=r/'.flow/tmp/fn2546-resumed';cache=s/'cache-review-kettle';assert not cache.exists()
log=(o/'cache-review-green.log').read_text();binary=r/re.search(r'Running unittests src/lib.rs \(([^)]+)\)',log)[1]
env=os.environ.copy();env.pop('BASH_ENV',None);env.update(TMPDIR=str(s/'tmp'),UMPIRE_CACHE_TEST_ROOT=str(r/'model/umpire'),UMPIRE_CACHE_TEST_REAL_KETTLE='1',UMPIRE_QUICK_CACHE='1',UMPIRE_CACHE_DISABLE='0',UMPIRE_QUICK_CACHE_ROOT=str(cache),MISE_TRUSTED_CONFIG_PATHS=str(r))
proof={}
for mode in ['cold','warm']:
 cmd=[str(binary),'--exact','runner::cache_tests::cache_child','--nocapture'];start=time.monotonic()
 with (o/f'cache-review-kettle-{mode}.log').open('w') as f:result=subprocess.run(cmd,cwd=r,env=env,stdout=f,stderr=subprocess.STDOUT)
 text=(o/f'cache-review-kettle-{mode}.log').read_text();facts={'command':cmd,'cwd':str(r),'exit_code':result.returncode,'elapsed_seconds':time.monotonic()-start,'cache_hits':text.count('TLC cache hit Kettle'),'cache_misses':text.count('TLC cache miss Kettle'),'executions':text.count('TLC execute Kettle'),'log_sha256':hashlib.sha256(text.encode()).hexdigest(),'cache_entries':len(list((cache/'quick-tlc-v1').iterdir())) if cache.exists() else 0}
 proof[mode]=facts;(o/'cache-review-kettle.json').write_text(json.dumps(proof,indent=2)+'\n');print(mode,json.dumps(facts),flush=True);assert result.returncode==0
 if mode=='cold':assert facts['cache_hits']==0 and facts['cache_misses']==3
 else:assert facts['cache_hits']==2 and facts['cache_misses']==1 and facts['executions']>0
 assert facts['cache_entries']==2
