from pathlib import Path
import os,sys,json,subprocess,time,signal,hashlib,ctypes
r=Path(__file__).resolve().parent;p=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed');f=p/'mise-capability-probe.json';d=json.loads(f.read_text());env=os.environ.copy();env.pop('BASH_ENV',None);env.update(d['environment_overrides']);ctypes.CDLL(None).prctl(36,1,0,0,0)
help=subprocess.run([d['binary'],'run','--help'],env=env,capture_output=True,text=True);(p/'mise-run-help.txt').write_text(help.stdout)
leaf=r/'supplement-leaf.py';leaf.write_text('''import os,sys,time,pathlib,subprocess,json
r=pathlib.Path(os.environ['PROBE_CASE_DIR']);n=sys.argv[1]
def event(kind,**kw):
 fd=os.open(r/'events.jsonl',os.O_WRONLY|os.O_CREAT|os.O_APPEND,0o600);os.write(fd,(json.dumps(dict(task=n,kind=kind,time=time.monotonic(),pid=os.getpid(),**kw))+'\\n').encode());os.close(fd)
event('start')
if n=='fail':
 if os.environ.get('WAIT_ACTIVE')=='1':
  until=time.monotonic()+3
  while not (r/'active').exists() and time.monotonic()<until:time.sleep(.01)
 event('exit',code=37);sys.exit(37)
if n=='slow':
 (r/'active').write_text('active');time.sleep(2);event('end')
if n=='descendant':
 c=subprocess.Popen(['/bin/sleep','12']);(r/'pids').write_text(str(os.getpid())+' '+str(c.pid));(r/'active').write_text('active');event('child',child=c.pid);time.sleep(12)
if n=='post':event('post')
''')
def live(pid):
 try:return Path(f'/proc/{pid}/stat').read_text().split(') ')[1][0] not in ['Z','X']
 except FileNotFoundError:return False
results={}
for name,slow,extra in [('single_failure',None,[]),('failure_plain2sec','slow',[]),('failure_exec_descendant','descendant',[]),('failure_timeout_descendant','descendant',['--timeout','2s'])]:
 c=r/name;c.mkdir(exist_ok=True);config='min_version="2026.10.11"\n'
 config+=f'[tasks.fail]\nrun="exec /usr/bin/python3 {leaf} fail"\ndepends_post=["post"]\n[tasks.post]\nrun="exec /usr/bin/python3 {leaf} post"\n'
 if slow:config+=f'[tasks.slow]\nrun="exec /usr/bin/python3 {leaf} {slow}"\n[tasks.all]\ndepends=["fail","slow"]\n'
 cf=c/'mise.toml';cf.write_text(config);args=[d['binary'],'run','--force','--skip-tools','--task-cache','off','--jobs','2',*extra,'all' if slow else 'fail'];log=open(c/'output.log','wb');start=time.monotonic();proc=subprocess.Popen(args,cwd=c,env=env|{'MISE_CONFIG_FILE':str(cf),'PROBE_CASE_DIR':str(c),'WAIT_ACTIVE':'1' if slow else '0'},stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
 timeout=False
 try:code=proc.wait(timeout=8)
 except subprocess.TimeoutExpired:code=None;timeout=True
 elapsed=time.monotonic()-start;time.sleep(.1);pids=[int(x) for x in (c/'pids').read_text().split()] if (c/'pids').exists() else [];states={str(pid):live(pid) for pid in pids};snapshot={}
 for pid in pids:
  try:snapshot[str(pid)]={'stat':Path(f'/proc/{pid}/stat').read_text(),'cmdline':Path(f'/proc/{pid}/cmdline').read_bytes().replace(b'\0',b' ').decode()}
  except FileNotFoundError:pass
 try:os.killpg(proc.pid,signal.SIGKILL)
 except ProcessLookupError:pass
 for pid in pids:
  if live(pid):
   try:os.kill(pid,signal.SIGKILL)
   except ProcessLookupError:pass
 if proc.poll() is None:proc.wait(timeout=2)
 time.sleep(.05)
 while True:
  try:
   z,_=os.waitpid(-1,os.WNOHANG)
   if not z:break
  except ChildProcessError:break
 log.close();results[name]={'command':args,'config':config,'exit':code,'elapsed_seconds':elapsed,'watchdog_seconds':8,'timed_out':timeout,'owned_live_before_cleanup':states,'owned_process_snapshots':snapshot,'owned_live_after_cleanup':{str(pid):live(pid) for pid in pids},'events':[json.loads(x) for x in (c/'events.jsonl').read_text().splitlines()] if (c/'events.jsonl').exists() else [],'output':(c/'output.log').read_text()}
d['supplemental_results']=results;d['supported_option_audit']={'run_help':str(p/'mise-run-help.txt'),'run_help_sha256':hashlib.sha256(help.stdout.encode()).hexdigest(),'failure_option':'--continue-on-error is the only failure-mode option; omitted in every production-flags probe. No --fail-fast or --kill-others option appears in complete actual binary help.','other_options':'--timeout bounds entire run; explicitly probed. --raw serializes tasks and cannot provide required overlap.','task_dependencies':'v2026.10.11 official reference says wait_for only waits already scheduled tasks; depends_post runs cleanup after started parent even on failure. Supplemental probes include depends_post to inspect actual failure behavior. Neither declarative edge is a generic descendant reaper.','official_sources':[{'url':'https://raw.githubusercontent.com/jdx/mise/v2026.10.11/'+('src/'+x if x.endswith('.rs') else 'docs/tasks/'+x),'scratch':str(r/'upstream'/x.replace('/','-')),'sha256':hashlib.sha256((r/'upstream'/x.replace('/','-')).read_bytes()).hexdigest()} for x in ['cli/run.rs']] if False else []}
d['attempt1_harness_correction']={'issue':'With custom MISE_CONFIG_FILE and isolated config directory, mise task working directory resolved /home/agent instead of requested case cwd. Initial instrumentation wrote five solely owned temporary files there.','correction':'Retained those exact files under scratch/attempt1; later leaves anchor every write to explicit absolute PROBE_CASE_DIR. No repository source changed. First attempt produced no claims and was rerun.','preserved':'scratch/attempt1'}
f.write_text(json.dumps(d,indent=2)+'\n');(p/'mise-capability-supplement.py').write_bytes(Path(__file__).read_bytes());print(json.dumps({'sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'supplemental':{n:{k:x[k] for k in ['exit','elapsed_seconds','timed_out','owned_live_before_cleanup']} for n,x in results.items()}},indent=2))
