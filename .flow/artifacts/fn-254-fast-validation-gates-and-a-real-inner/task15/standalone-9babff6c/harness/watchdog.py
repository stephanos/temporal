import argparse,hashlib,json,os,re,signal,subprocess,sys,time
from pathlib import Path
GIB=1024**3
BASE=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
HERE=Path(__file__).resolve().parent
SOURCE=BASE
def sha(p): return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(p,v): Path(p).write_text(json.dumps(v,indent=2)+'\n')
def space(p):
 s=os.statvfs(p)
 return dict(path=str(p),realpath=str(Path(p).resolve()),st_dev=os.stat(p).st_dev,free=s.f_bavail*s.f_frsize)
def check_source(expected_head,source=SOURCE):
 head=subprocess.check_output(['git','rev-parse','HEAD'],cwd=source,text=True).strip()
 status=subprocess.check_output(['git','status','--porcelain'],cwd=source,text=True)
 assert head==expected_head and not status,dict(head=head,status=status)
 return dict(head=head,status=status)
def check_hashes(directory=HERE):
 manifest_path=directory/'input-manifest.json'
 manifest=json.loads(manifest_path.read_text())
 results={name:sha(directory/name) for name in manifest['files']}
 assert results==manifest['files'],'prepared harness hash mismatch'
 return dict(files=results,manifest_sha256=sha(manifest_path))
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
 parser=argparse.ArgumentParser(description='Prepared task15 standalone proof watchdog; execution requires a clean admitted HEAD.')
 parser.add_argument('--expected-head',required=True)
 parser.add_argument('--execute',action='store_true')
 options=parser.parse_args()
 assert re.fullmatch('[0-9a-f]{40}',options.expected_head),'expected HEAD must be a full Git SHA'
 if not options.execute:
  print('Prepared only; no source inspection, process, build or proof execution requested.')
  return 0
 output=HERE/'fn25415-proof-output'
 command=[sys.executable,str(HERE/'proof.py'),'--source-root',str(SOURCE),'--expected-head',options.expected_head,'--output',str(output),'--execute']
 def precheck():
  assert not output.exists(),'proof output already exists'
  hashes=check_hashes()
  controls=json.loads((HERE/'controls.json').read_text())
  assert controls['success'] and controls['tests_run']==4 and controls['failures']==controls['errors']==0,'wrapper controls failed'
  assert controls['input_manifest_sha256']==hashes['manifest_sha256'],'controls are stale for prepared inputs'
  return dict(source=check_source(options.expected_head),hashes=hashes,controls=controls,
   coldness='task15 owns reuse of existing task16-named target; fresh source native bootstrap; no empty-target claim',
   target=str(BASE/'.flow/tmp/fn25416-provider-validation/target'),
   growth_guard='30GiB retained in proof harness',source_filesystem=space(SOURCE),actual_private_tmp_filesystem=space('/tmp'))
 def inspect():
  validation=json.loads((output/'validation.json').read_text()) if (output/'validation.json').exists() else None
  return dict(validation_accepted=bool(validation and validation.get('accepted')),validation_path=str(output/'validation.json'),
   source_after=check_source(options.expected_head),hashes_after=check_hashes(),
   owned_cleanup=json.loads((output/'owned-cleanup.json').read_text()) if (output/'owned-cleanup.json').exists() else None)
 result=supervise(command,HERE,inspect,lambda:[space(SOURCE),space('/tmp')],precheck)
 print(json.dumps(result,indent=2),flush=True)
 return 0 if result['accepted'] else 1
if __name__=='__main__':sys.exit(main())
