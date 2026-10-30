import os,sys,subprocess,time,json,pathlib,signal,hashlib,ctypes,shutil
root=pathlib.Path(__file__).resolve().parent
proof=pathlib.Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed')
libc=ctypes.CDLL(None);libc.prctl(36,1,0,0,0)
mise=pathlib.Path('/home/agent/.local/bin/mise').resolve()
env=os.environ.copy();env.pop('BASH_ENV',None)
for k,v in {'MISE_AUTO_INSTALL':'false','MISE_EXEC_AUTO_INSTALL':'false','MISE_TRUSTED_CONFIG_PATHS':str(root),'MISE_CONFIG_DIR':str(root/'config'),'MISE_DATA_DIR':str(root/'data'),'MISE_CACHE_DIR':str(root/'cache'),'MISE_STATE_DIR':str(root/'state'),'MISE_HIDE_UPDATE_WARNING':'1','MISE_YES':'1'}.items():env[k]=v
leaf=root/'leaf.py'
leaf.write_text('''import os,sys,time,json,pathlib,subprocess
r=pathlib.Path(os.environ['PROBE_CASE_DIR']);name=sys.argv[1]
def event(kind,**extra):
 row={'task':name,'kind':kind,'monotonic':time.monotonic(),'pid':os.getpid(),**extra}
 fd=os.open(r/'events.jsonl',os.O_WRONLY|os.O_CREAT|os.O_APPEND,0o600);os.write(fd,(json.dumps(row)+'\\n').encode());os.close(fd)
event('start')
if name in ['p','q']:
 time.sleep(0.35);(r/(name+'.done')).write_text('done');event('end')
elif name=='join':
 assert (r/'p.done').exists() and (r/'q.done').exists();event('dependencies-satisfied');event('end')
elif name in ['slow','detached']:
 child=subprocess.Popen(['/bin/sleep','12'],start_new_session=name=='detached');(r/'owned-pids').write_text(str(os.getpid())+' '+str(child.pid));event('descendant',child=child.pid,detached=name=='detached');(r/'slow.active').write_text('active');time.sleep(12);event('end')
elif name=='fail':
 deadline=time.monotonic()+4
 while not (r/'slow.active').exists() and time.monotonic()<deadline:time.sleep(0.01)
 assert (r/'slow.active').exists();event('failure',exit=37);sys.exit(37)
elif name=='blocked':
 event('DEPENDENT-MUST-NOT-RUN');sys.exit(91)
''')
base='min_version = "2026.10.11"\n'
def config(directory,text):
 directory.mkdir(exist_ok=True); f=directory/'mise.toml';f.write_text(base+text);return f
success=''
for n in ['p','q']:
 success+=f'\n[tasks.{n}]\nrun = "/usr/bin/python3 {leaf} {n}"\n'
success+=f'\n[tasks.join]\ndepends = ["p", "q"]\nrun = "/usr/bin/python3 {leaf} join"\n'

def live(pid):
 try:
  state=pathlib.Path(f'/proc/{pid}/stat').read_text().split(') ')[1].split()[0]
  return state not in ['Z','X']
 except FileNotFoundError:return False

def events(d):return [json.loads(x) for x in (d/'events.jsonl').read_text().splitlines()] if (d/'events.jsonl').exists() else []
def cleanup(proc,d):
 owned=[int(x) for x in (d/'owned-pids').read_text().split()] if (d/'owned-pids').exists() else []
 observed={str(pid):live(pid) for pid in owned}
 try:os.killpg(proc.pid,signal.SIGKILL)
 except ProcessLookupError:pass
 for pid in owned:
  if live(pid):
   try:os.kill(pid,signal.SIGKILL)
   except ProcessLookupError:pass
 if proc.poll() is None:proc.wait(timeout=3)
 time.sleep(.05)
 reaped=[]
 while True:
  try:
   pid,status=os.waitpid(-1,os.WNOHANG)
   if pid==0:break
   reaped.append(pid)
  except ChildProcessError:break
 return {'owned_live_after_mise_return_or_wait':observed,'probe_cleanup_reaped':reaped,'owned_live_after_probe_cleanup':{str(pid):live(pid) for pid in owned}}

def run_case(name,text,task,jobs,cancel=False):
 d=root/name;f=config(d,text);local=env|{'MISE_CONFIG_FILE':str(f),'PROBE_CASE_DIR':str(d)}
 args=[str(mise),'run','--force','--skip-tools','--task-cache','off','--jobs',str(jobs),task]
 start=time.monotonic();log=open(d/'output.log','wb');proc=subprocess.Popen(args,cwd=d,env=local,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
 cancelled=False
 if cancel:
  deadline=time.monotonic()+5
  while not (d/'slow.active').exists() and proc.poll() is None and time.monotonic()<deadline:time.sleep(.02)
  if (d/'slow.active').exists():os.kill(proc.pid,signal.SIGTERM);cancelled=True
 timedout=False
 try:code=proc.wait(timeout=5)
 except subprocess.TimeoutExpired:code=None;timedout=True
 time.sleep(.15);result={'command':args,'cwd':str(d),'config':f.read_text(),'exit':code,'elapsed_seconds':time.monotonic()-start,'timed_out':timedout,'sent_sigterm_to_mise':cancelled,'events':events(d)}
 result.update(cleanup(proc,d));log.close();result['output']=(d/'output.log').read_text();result['output_sha256']=hashlib.sha256((d/'output.log').read_bytes()).hexdigest();return result
results={}
results['jobs2']=run_case('jobs2',success,'join',2)
results['jobs1']=run_case('jobs1',success,'join',1)
for n in ['slow','detached']:
 text=f'\n[tasks.slow]\nrun = "/usr/bin/python3 {leaf} {n}"\n[tasks.fail]\nrun = "/usr/bin/python3 {leaf} fail"\n[tasks.blocked]\ndepends=["fail"]\nrun = "/usr/bin/python3 {leaf} blocked"\n[tasks.all]\ndepends=["slow","blocked"]\n'
 results['failure_'+n]=run_case('failure_'+n,text,'all',2)
text=f'\n[tasks.slow]\nrun = "/usr/bin/python3 {leaf} slow"\n'
results['cancel']=run_case('cancel',text,'slow',1,True)
d=root/'missing-tool';f=config(d,'\n[tools]\npython="0.0.0"\n');args=[str(mise),'exec','--','python','--version'];t=time.monotonic();proc=subprocess.run(args,cwd=d,env=env|{'MISE_CONFIG_FILE':str(f)},capture_output=True,timeout=8);results['no_auto_install']={'command':args,'config':f.read_text(),'exit':proc.returncode,'elapsed_seconds':time.monotonic()-t,'stdout':proc.stdout.decode(),'stderr':proc.stderr.decode(),'install_tree_exists':(root/'data/installs/python/0.0.0').exists()}
version=subprocess.run([str(mise),'--version'],env=env,capture_output=True,text=True)
report={'scope':'Tool-only isolated mise probe; no Cargo/Go/JVM/protoc/source/Git/Flow operations. Probe-owned descendants explicitly inspected and cleaned after observations.','binary':str(mise),'binary_sha256':hashlib.sha256(mise.read_bytes()).hexdigest(),'version_stdout':version.stdout,'version_stderr':version.stderr,'verified_version':'2026.10.11','minimum_admission':'Only this exact2026.10.11 binary was executed; no older release verified.','environment_overrides':{k:v for k,v in env.items() if k.startswith('MISE_')},'probe_script_sha256':hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),'leaf_script':leaf.read_text(),'results':results}
for name in ['jobs1','jobs2']:
 e=results[name]['events'];times={n:{x['kind']:x['monotonic'] for x in e if x['task']==n} for n in ['p','q','join']};results[name]['producers_overlap']=max(times['p']['start'],times['q']['start'])<min(times['p']['end'],times['q']['end']);results[name]['join_after_both_producers']=times['join']['start']>=max(times['p']['end'],times['q']['end'])
report['findings']={'jobs2_real_overlap':results['jobs2']['producers_overlap'],'jobs1_serial':not results['jobs1']['producers_overlap'],'dependency_edges_hold':all(results[x]['join_after_both_producers'] for x in ['jobs1','jobs2']),'failure_blocks_dependents':all(not any(e['kind']=='DEPENDENT-MUST-NOT-RUN' for e in results[x]['events']) for x in ['failure_slow','failure_detached']),'surviving_descendant_cases':[n for n,x in results.items() if any(x.get('owned_live_after_mise_return_or_wait',{}).values())],'no_auto_install_missing_tool_fails':results['no_auto_install']['exit']!=0 and not results['no_auto_install']['install_tree_exists']}
f=proof/'mise-capability-probe.json';f.write_text(json.dumps(report,indent=2)+'\n');shutil.copyfile(__file__,proof/'mise-capability-probe.py');print(json.dumps({'report':str(f),'sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'findings':report['findings'],'exits':{n:x['exit'] for n,x in results.items()}},indent=2))
