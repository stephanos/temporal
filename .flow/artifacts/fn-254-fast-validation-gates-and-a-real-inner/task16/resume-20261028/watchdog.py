import hashlib,json,os,signal,subprocess,sys,time
from pathlib import Path
GIB=1024**3
BASE=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
HERE=Path(__file__).resolve().parent
SOURCE=BASE/'.worktrees/fn25416-provider-adapt'
RETAINED=BASE/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/session-wrap-20261027'
HEAD='60f1c762a618d32478b58ae4e79c151f4355aeb3'
def sha(p): return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(p,v): Path(p).write_text(json.dumps(v,indent=2)+'\n')
def space(p):
 s=os.statvfs(p)
 return dict(path=str(p),realpath=str(Path(p).resolve()),st_dev=os.stat(p).st_dev,free=s.f_bavail*s.f_frsize)
def check_source():
 head=subprocess.check_output(['git','rev-parse','HEAD'],cwd=SOURCE,text=True).strip()
 status=subprocess.check_output(['git','status','--porcelain'],cwd=SOURCE,text=True)
 assert head==HEAD and not status,dict(head=head,status=status)
 return dict(head=head,status=status)
def check_hashes():
 manifest=json.loads((RETAINED/'preservation-manifest.json').read_text())
 results={str(RETAINED/k):sha(RETAINED/k) for k in manifest['files']}
 assert all(results[str(RETAINED/k)]==v for k,v in manifest['files'].items()),'preservation hash mismatch'
 for p,v in json.loads((RETAINED/'standalone-native-build-final.json').read_text())['binaries'].items():
  results[p]=sha(p)
  assert results[p]==v['sha256'],'native hash mismatch'
 results[str(RETAINED/'preservation-manifest.json')]=sha(RETAINED/'preservation-manifest.json')
 return results
def supervise(command,out,inspect,spaces,precheck,interval=1,popen=subprocess.Popen):
 receipt={'accepted':False,'rejected':False,'command':command,'pid':None,'terminal':False}
 child=None
 def reject(reason):
  receipt.update(rejected=True,accepted=False,rejection=reason)
  write(out/'watchdog-result.json',receipt)
 def stop(*_): raise KeyboardInterrupt
 old={s:signal.signal(s,stop) for s in (signal.SIGTERM,signal.SIGINT)}
 try:
  receipt['inputs']=precheck()
  receipt['admission_spaces']=spaces()
  assert all(v['free']>=50*GIB for v in receipt['admission_spaces']),'50GiB admission guard'
  write(out/'admission.json',receipt)
  with (out/'proof.stdout.log').open('wb') as stdout,(out/'proof.stderr.log').open('wb') as stderr,(out/'space-samples.jsonl').open('w') as samples:
   child=popen(command,stdout=stdout,stderr=stderr,env={k:v for k,v in os.environ.items() if k!='BASH_ENV'})
   receipt['pid']=child.pid
   write(out/'live-handle.json',{'pid':child.pid,'command':command,'started':time.time()})
   while child.poll() is None:
    current=spaces();samples.write(json.dumps({'at':time.time(),'spaces':current})+'\n');samples.flush()
    if any(v['free']<30*GIB for v in current):
     reject('30GiB runtime free-space guard');child.send_signal(signal.SIGTERM);break
    try: child.wait(timeout=interval)
    except subprocess.TimeoutExpired: pass
   receipt['exit_code']=child.wait()
  if not receipt['rejected']:
   receipt.update(inspect())
   receipt['accepted']=receipt['exit_code']==0 and receipt.get('validation_accepted',False)
 except BaseException as e:
  reject(type(e).__name__+': '+str(e))
 finally:
  for s in old:signal.signal(s,signal.SIG_IGN)
  if child is not None:
   if child.poll() is None:
    child.send_signal(signal.SIGTERM)
   receipt['exit_code']=child.wait()
   receipt['terminal']=True
  for s,h in old.items():signal.signal(s,h)
  write(out/'watchdog-result.json',receipt)
 return receipt

def main():
 output=HERE/'proof-output'
 command=['python3',str(RETAINED/'retry-harness/proof.py'),'--source-root',str(SOURCE),'--expected-head',HEAD,'--output',str(output),'--execute']
 def precheck():
  assert not output.exists(),'proof output already exists'
  assert json.loads((HERE/'controls.json').read_text())['success'],'wrapper controls failed'
  return dict(source=check_source(),hashes=check_hashes(),wrapper_sha256=sha(__file__),controls_sha256=sha(HERE/'test_watchdog.py'),controls=json.loads((HERE/'controls.json').read_text()),coldness='reuses existing xtask-bootstrap; no empty-target claim',growth_guard='retained harness unchanged at 30GiB',source_filesystem=space(SOURCE),actual_private_tmp_filesystem=space('/tmp'))
 def inspect():
  validation=json.loads((output/'validation.json').read_text()) if (output/'validation.json').exists() else None
  return dict(validation_accepted=bool(validation and validation.get('accepted')),validation_path=str(output/'validation.json'),source_after=check_source(),hashes_after=check_hashes(),owned_cleanup=json.loads((output/'owned-cleanup.json').read_text()) if (output/'owned-cleanup.json').exists() else None)
 result=supervise(command,HERE,inspect,lambda:[space(SOURCE),space('/tmp')],precheck)
 print(json.dumps(result,indent=2),flush=True)
 return 0 if result['accepted'] else 1
if __name__=='__main__':sys.exit(main())
