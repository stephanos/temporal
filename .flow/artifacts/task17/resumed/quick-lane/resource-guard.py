import os,sys,subprocess,shutil,time,signal,json,tempfile
from pathlib import Path
root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
work=root/'.worktrees/fn25417-quick-resumed'
target=root/'.flow/tmp/fn2546-selection-review/target'
evidence=work/'.flow/artifacts/task17/resumed/quick-lane'
label=sys.argv[1];args=sys.argv[2:]
size=lambda:int(subprocess.check_output(['du','-sk',str(target)]).split()[0])*1024
initial=size();private=tempfile.mkdtemp(prefix='fn25417quick-',dir='/tmp')
env=os.environ.copy();env.pop('BASH_ENV',None)
env.update(CARGO_TARGET_DIR=str(target),CARGO_BUILD_JOBS='1',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',TMPDIR=private,MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false')
started=time.monotonic();samples=[];breach=None
with open(evidence/(label+'.log'),'wb') as output:
 child=subprocess.Popen(args,cwd=work,env=env,stdout=output,stderr=subprocess.STDOUT,start_new_session=True)
 while child.poll() is None:
  free=shutil.disk_usage(work).free;current=size();samples.append({'seconds':round(time.monotonic()-started,3),'free_bytes':free,'target_bytes':current})
  if free<8*1024**3 or current-initial>768*1024**2:
   breach={'free_bytes':free,'growth_bytes':current-initial};os.killpg(child.pid,signal.SIGTERM)
   try:child.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL)
   break
  time.sleep(1)
 status=child.wait()
receipt={'command':args,'cwd':str(work),'target':str(target),'tmpdir':private,'initial_target_bytes':initial,'final_target_bytes':size(),'exit':status,'guard_breach':breach,'samples':samples,'seconds':round(time.monotonic()-started,3)}
(evidence/(label+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps({k:receipt[k] for k in ['exit','guard_breach','initial_target_bytes','final_target_bytes','seconds']}))
sys.exit(status if status>=0 else 128-status)
