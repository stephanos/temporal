from pathlib import Path
import subprocess,os,json,hashlib,time
r=Path.cwd();scratch=r/'.flow/tmp/fn25417-graph-tool';scratch.mkdir(parents=True,exist_ok=True);out=Path(__file__).resolve().parent
leaf=scratch/'leaf.py'
leaf.write_text("""#!/usr/bin/python3
import os,json,pathlib,time
row={key:os.environ.get(key) for key in ['UMPIRE_GATE_STAGE','UMPIRE_GATE_SCOPE','UMPIRE_TOOL_ROOT']};row['cwd']=os.getcwd();row['start']=time.monotonic();time.sleep(.04);row['end']=time.monotonic()
fd=os.open(os.environ['GRAPH_TOOL_LOG'],os.O_WRONLY|os.O_CREAT|os.O_APPEND,0o600);os.write(fd,(json.dumps(row)+'\\n').encode());os.close(fd)
""")
leaf.chmod(0o755)
env=os.environ.copy();env.pop('BASH_ENV',None);env.pop('MISE_CONFIG_FILE',None);env.update(MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_TRUSTED_CONFIG_PATHS=str(r),TMPDIR=str(scratch),UMPIRE_FRAMEWORK_ROOT=str(r/'model/umpire'),UMPIRE_TEMPORAL_ROOT=str(r/'model'),UMPIRE_GATE_EXECUTABLE=str(leaf),UMPIRE_TEMPORAL_GATE_EXECUTABLE=str(leaf))
results={}
for name,cwd,task in [('combined',r,'model:all'),('framework',r/'model/umpire','gate:child')]:
 x=subprocess.run(['/home/agent/.local/bin/mise','tasks','ls','--json'],cwd=cwd,env=env,capture_output=True,timeout=8);assert x.returncode==0,x.stderr.decode();tasks=json.loads(x.stdout);by={t['name']:t for t in tasks};selected={}
 for t in tasks:
  directives=dict(e.split('=',1) for e in t['env'] if '=' in e)
  if 'UMPIRE_GATE_STAGE' in directives and (name=='combined' or t['name'].startswith('gate:')):selected[directives.get('UMPIRE_GATE_SCOPE','')+'/'+directives['UMPIRE_GATE_STAGE']]=t['name']
 def closure(task,active=()):
  assert task not in active,('cycle',task,active)
  values={task}
  for dep in by[task]['depends']:values|=closure(dep,active+(task,))
  return values
 reached=closure(task);assert all(t in reached for identity,t in selected.items() if name=='combined' or not identity.endswith('/standalone'))
 if name=='combined':
  assert len(selected)==31,len(selected)
  assert 'model:framework' not in by
  assert 'model:wasm-guest' in by['gate:go-host']['wait_for']
  assert 'model:wasm-guest' in reached
  assert 'gate:wasm-guest' in closure('model:temporal-prerequisites')
  assert 'model:temporal-prerequisites' in closure('model:temporal-guest-freshness')
  assert 'gate:standalone' in closure('model:temporal-go-host')
  assert closure('model:fmt')=={'model:fmt'}
 else:
  assert len(selected)==18,len(selected)
  assert not any(n.startswith('model:') for n in reached),reached
 log=scratch/(name+'.jsonl');log.write_text('');env['GRAPH_TOOL_LOG']=str(log)
 args=['/home/agent/.local/bin/mise','run','--force','--skip-tools','--task-cache','off','--jobs','4',task];start=time.monotonic();y=subprocess.run(args,cwd=cwd,env=env,capture_output=True,timeout=8);assert y.returncode==0,y.stderr.decode()
 events=[json.loads(line) for line in log.read_text().splitlines()]
 for event in events:
  expected_root=r/'model/umpire' if event['UMPIRE_GATE_SCOPE']=='framework' else r/'model'
  assert event['cwd']==str(expected_root),event
  if event['UMPIRE_GATE_SCOPE']=='framework':assert event['UMPIRE_TOOL_ROOT']==str(expected_root),event
 assert len(events)==(31 if name=='combined' else 17),len(events)
 overlaps=[(a['UMPIRE_GATE_SCOPE']+'/'+a['UMPIRE_GATE_STAGE'],b['UMPIRE_GATE_SCOPE']+'/'+b['UMPIRE_GATE_STAGE']) for i,a in enumerate(events) for b in events[i+1:] if max(a['start'],b['start'])<min(a['end'],b['end'])]
 assert overlaps
 results[name]={'leaf_events':events,'overlapping_stand_in_pairs':overlaps,'cwd':str(cwd),'inventory':tasks,'selected_stage_ids':selected,'reachable_tasks':sorted(reached),'run':args,'exit':y.returncode,'seconds':time.monotonic()-start,'stdout':y.stdout.decode(),'stderr':y.stderr.decode()}
report={'scope':'Actual mise include/metadata-override/dependency execution with tool-only Python leaf stand-ins; NOT native gate, model, standalone, build or suite evidence. No Cargo/Go/JVM launched.','version':subprocess.check_output(['/home/agent/.local/bin/mise','--version'],env=env,text=True).strip(),'source_sha256':{str(p.relative_to(r)):hashlib.sha256(p.read_bytes()).hexdigest() for p in [r/'mise.toml',r/'model/umpire/mise.toml',r/'model/umpire/dev/xtask/src/gate/tasks.toml']},'results':results};(out/'mise-shared-graph-tool-31.json').write_text(json.dumps(report,indent=2)+'\n');print('Actual shared graph tool controls passed:',[(n,len(v['selected_stage_ids']),v['exit']) for n,v in results.items()])
