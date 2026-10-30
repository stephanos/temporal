import json, subprocess, time, pathlib, os, statistics, hashlib, platform
root=pathlib.Path(__file__).resolve().parent
args=json.loads((root/'link-args.json').read_text())
args[args.index('-o')+1]=str(root/'measured-output')
config=pathlib.Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/model/umpire')
env=dict(os.environ,MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',BASH_ENV='/dev/null')
env.pop('RUSTC_WRAPPER',None);env.pop('RUSTC_WORKSPACE_WRAPPER',None)
commands={'baseline':['/usr/bin/cc',*args], 'mold':[str(config/'scripts/mold-linker.sh'),*args]}
rows=[]
for index,arm in enumerate(['baseline','mold']+['baseline','mold','mold','baseline']*3):
 start=time.monotonic();r=subprocess.run(commands[arm],cwd=config,env=env,capture_output=True);wall=time.monotonic()-start
 (root/f'relink-{index}-{arm}.stderr').write_bytes(r.stderr)
 row={'index':index,'arm':arm,'warmup':index<2,'seconds':wall,'exit':r.returncode}
 if r.returncode: raise RuntimeError(row,r.stderr)
 row['bytes']=(root/'measured-output').stat().st_size
 row['sha256']=hashlib.sha256((root/'measured-output').read_bytes()).hexdigest()
 comment=subprocess.check_output(['readelf','-p','.comment',str(root/'measured-output')],text=True)
 (root/f'relink-{index}-{arm}.comment').write_text(comment)
 row['mold_comment']='mold 3.0.0' in comment
 assert row['mold_comment']==(arm=='mold')
 rows.append(row)
result={'host':platform.uname()._asdict(),'cpus':os.cpu_count(),'load':os.getloadavg(),'commands':commands,'rows':rows,'medians':{a:statistics.median(r['seconds'] for r in rows if not r['warmup'] and r['arm']==a) for a in commands}}
(root/'measurement.json').write_text(json.dumps(result,indent=2));print(json.dumps(result['medians']))
