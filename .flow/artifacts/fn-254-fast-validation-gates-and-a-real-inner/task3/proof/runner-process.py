import hashlib,json,os,pathlib,subprocess
W=pathlib.Path(__file__).resolve().parents[5];P=pathlib.Path(__file__).resolve().parent
S=W/'.flow/tmp/fn2543-runner-controls';S.mkdir(parents=True,exist_ok=True)
B=pathlib.Path('/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2543-nextest')
shim=S/'mise';shim.write_text('''#!/usr/bin/python3
import json,os,signal,sys
with open(os.environ['RECORD'],'a') as f: f.write(json.dumps({'args':sys.argv[1:],'cwd':os.getcwd(),'env':{k:os.environ.get(k) for k in ['INSTA_UPDATE','TRYBUILD','UMPIRE_BACKENDS','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE','CARGO_HOME','CARGO_TARGET_DIR']}})+'\\n')
mode=os.environ['CONTROL_MODE'];binary='nextest' in sys.argv
if mode=='signal' and binary: os.kill(os.getpid(),signal.SIGTERM)
if mode=='binary-fail' and binary: sys.exit(42)
if mode=='doc-fail' and not binary: sys.exit(37)
''');shim.chmod(0o755)
harness=S/'temporal-leaf.rs';harness.write_text(f'''#[path = "{W}/model/dev/xtask/src/test_run.rs"] mod test_run;
#[path = "{W}/model/dev/xtask/src/gate.rs"] mod gate;
fn main() {{ match test_run::run(std::path::Path::new(&std::env::args().nth(1).expect("root"))) {{ Ok(code) => std::process::exit(code), Err(e) => {{ eprintln!("{{e}}"); std::process::exit(1) }} }} }}
''')
compile=['mise','exec','--','rustc','--edition=2024','-A','dead_code',str(harness),'--crate-name','temporal_leaf','--extern','umpire_guest_driver='+str(next((B/'framework/debug/deps').glob('libumpire_guest_driver-*.rlib'))),'-L','dependency='+str(B/'framework/debug/deps'),'-o',str(S/'temporal-leaf')]
r=subprocess.run(compile,cwd=W,capture_output=True);(P/'temporal-leaf-compile.err').write_bytes(r.stderr);assert r.returncode==0
rows=[]
for route,args in [('framework-stage',[str(B/'framework/debug/xtask'),'gate','--stage','tests']),('framework-update',[str(B/'framework/debug/xtask'),'update']),('temporal-leaf',[str(S/'temporal-leaf'),str(W/'model')])]:
 for mode,expected in [('success',0),('binary-fail',42),('doc-fail',37),('signal',143),('missing',1)]:
  label=route+'-'+mode;record=S/(label+'.jsonl');record.write_text('')
  env=dict(os.environ,PATH=str(S) if mode!='missing' else str(S/'absent'),RECORD=str(record),CONTROL_MODE=mode,UMPIRE_TOOL_ROOT=str(W/'model/umpire'),UMPIRE_BACKENDS='require',CARGO_TARGET_DIR=str(B/'framework'),INSTA_UPDATE='always',TRYBUILD='overwrite');env.pop('BASH_ENV',None)
  r=subprocess.run(args,cwd=W/'model/umpire' if route.startswith('framework') else W/'model',env=env,capture_output=True)
  calls=[json.loads(x) for x in record.read_text().splitlines()];(P/(label+'.err')).write_bytes(r.stderr)
  assert r.returncode==expected,(label,r.returncode,r.stderr)
  assert len(calls)==(0 if mode=='missing' else 1 if mode in ['binary-fail','signal'] else 2)
  if calls:
   assert calls[0]['args'][:5]==['exec','--','cargo','nextest','run']
   if len(calls)==2:assert calls[1]['args']==['exec','--','cargo','test','--doc','--workspace','--locked','--quiet']
   assert all(c['env']['INSTA_UPDATE']=='always' and c['env']['TRYBUILD']=='overwrite' for c in calls)
   if route!='temporal-leaf':assert all(c['env']['UMPIRE_QUICK_CACHE']=='0' and c['env']['UMPIRE_CACHE_DISABLE']=='1' for c in calls)
  rows.append(dict(route=route,mode=mode,argv=args,exit=r.returncode,calls=calls))
(P/'runner-process.json').write_text(json.dumps({'compile':compile,'controls':rows,'passed':len(rows),'temporal_scope':'Compiled exact current Temporal leaf and gate module; entry-point registration preserved by source. Full Temporal gate deferred to task8.'},indent=2)+'\n')
print('runner process controls:',len(rows),'passed')
