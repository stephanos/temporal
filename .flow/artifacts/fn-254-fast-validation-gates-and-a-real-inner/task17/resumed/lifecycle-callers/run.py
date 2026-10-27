import json,os,pathlib,shutil,signal,subprocess,sys,time
root=pathlib.Path.cwd(); cache='/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2546-resumed/target'
env=os.environ.copy();env.pop('BASH_ENV',None)
env.update(CARGO_TARGET_DIR=cache,CARGO_BUILD_JOBS='1',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',CARGO_INCREMENTAL='0',TMPDIR='/tmp/fn25417-lifecycle-callers/tmp',RUSTUP_TOOLCHAIN='1.99.1')
env['PATH']='/home/agent/.local/share/mise/installs/github-nextest-rs-nextest/0.9.149:'+env['PATH']
label=sys.argv[1];cmd=sys.argv[2:];start=time.time();guard=None
log=pathlib.Path('/tmp/fn25417-lifecycle-callers')/(label+'.log')
with log.open('wb') as output:
 p=subprocess.Popen(cmd,env=env,stdout=output,stderr=subprocess.STDOUT,start_new_session=True)
 while p.poll() is None:
  used=int(subprocess.check_output(['du','-sk',cache]).split()[0])
  free=shutil.disk_usage(root).free
  if used>6988648 or free<8*1024**3:
   guard={'used_kib':used,'free_bytes':free};os.killpg(p.pid,signal.SIGTERM);p.wait();break
  time.sleep(2)
status=p.returncode
receipt={'command':cmd,'status':status,'guard':guard,'seconds':time.time()-start,'cache_kib':int(subprocess.check_output(['du','-sk',cache]).split()[0]),'free_bytes':shutil.disk_usage(root).free}
log.with_suffix('.json').write_text(json.dumps(receipt,indent=2)+'\n');print(json.dumps(receipt));sys.exit(status if status>=0 else 128-status)
