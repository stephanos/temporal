import pathlib,sys,json,subprocess,hashlib
from importlib.machinery import SourceFileLoader
m=SourceFileLoader('proof','/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-task6/run-proof.py').load_module()
source=m.candidate/'model/temporal/feature/activity/model.rs'
original=source.read_bytes()
mode=sys.argv[1]
extra={}
command=['make','model-quick']
if mode=='baseline': pass
elif mode=='preserving':
    source.write_bytes(original.replace(b'//! Bounded activity retries, optional deadlines and cooperative cancellation.',b'//! Bounded activity retries, optional deadlines and cooperative cancellation (quick proof).',1))
elif mode=='emission':
    before=b'Client::Cancel if world.asynchronous_cancel && self.phase == Phase::Started =>'
    after=b'Client::Cancel if self.phase == Phase::Started && world.asynchronous_cancel =>'
    assert original.count(before)==1
    source.write_bytes(original.replace(before,after,1))
    extra['INSTA_UPDATE']='always'
elif mode=='bypass':
    command=['mise','exec','--','cargo','test','--locked','--manifest-path','model/Cargo.toml','-p','umpire-temporal-tests','--test','fixtures','fixtures','--','--exact','--nocapture','--test-threads=2']
    extra.update(UMPIRE_QUICK_CACHE='0',UMPIRE_CACHE_DISABLE='1',UMPIRE_QUICK_CACHE_ROOT=str(m.cache/'candidate/quick-cache'),UMPIRE_BACKENDS='require')
else: raise SystemExit('unknown mode')
try:
    code=m.run('activity-'+mode,command,extra=extra)
    log=(m.proof/('activity-'+mode+'.log')).read_text()
    facts={'mode':mode,'status':code,'cache_hits':log.count('TLC cache hit '),'cache_misses':log.count('TLC cache miss '),'executions':log.count('TLC execute '),'native_interpreter':log.count('quick native/interpreter agreement '),'original_sha256':hashlib.sha256(original).hexdigest(),'edited_sha256':hashlib.sha256(source.read_bytes()).hexdigest(),'tla_snapshot_sha256':hashlib.sha256((source.parent/'expected/Activity.tla.snap').read_bytes()).hexdigest()}
    (m.proof/('activity-'+mode+'-facts.json')).write_text(json.dumps(facts,indent=2)+'\n')
    print(json.dumps(facts),flush=True)
    if code: raise SystemExit(code)
    if mode in ['baseline','preserving','emission']: assert facts['native_interpreter']>0
    if mode=='preserving': assert facts['cache_hits']>0 and facts['cache_misses']==0 and facts['executions']==0
    if mode in ['baseline','emission']: assert facts['cache_misses']>0 and facts['executions']>0
    if mode=='bypass': assert facts['cache_hits']==0 and facts['cache_misses']==0 and facts['executions']>0
finally:
    source.write_bytes(original)
    if mode=='emission':
        subprocess.run(['git','restore','--','model/temporal/feature'],cwd=m.candidate,check=True)
