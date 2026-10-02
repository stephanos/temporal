from pathlib import Path
import os,sys,subprocess,json,time,hashlib,datetime
root=Path(__file__).resolve().parents[4]
art=Path(__file__).resolve().parent
name=sys.argv[1]; argv=sys.argv[2:]; env=dict(os.environ)
for key in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED']: env.pop(key,None)
env.update(GOWORK='off',GOMAD3_STOCK_GO='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go')
env['PATH']=str(Path(env['GOMAD3_STOCK_GO']).parent)+os.pathsep+env['PATH']
if name.startswith('make-'): env['GOFLAGS']='-tags=test_dep -count=1'
if name=='make-test-host': env.pop('GOMAD3_STOCK_GO',None)
state=json.loads((art/'before-state.json').read_text())
binding={p:hashlib.sha256((root/p).read_bytes()).hexdigest() if (root/p).exists() else None for p in state['owned']}
t=time.time(); started=datetime.datetime.now(datetime.timezone.utc).isoformat()
with (art/(name+'.log')).open('wb') as log: result=subprocess.run(argv,cwd=root,env=env,stdout=log,stderr=subprocess.STDOUT)
after_binding={p:hashlib.sha256((root/p).read_bytes()).hexdigest() if (root/p).exists() else None for p in state['owned']}
metadata={'source_after':after_binding,'sources_unchanged_during_command':binding==after_binding,'argv':argv,'cwd':str(root),'environment':{k:env.get(k) for k in ['GOWORK','GOMAD3_STOCK_GO','PATH','GOFLAGS','GOROOT','GOMADSEED','GOMAD3_CHILD_SEED']},'started':started,'elapsed_seconds':time.time()-t,'exit_code':result.returncode,'sources':binding,'log_sha256':hashlib.sha256((art/(name+'.log')).read_bytes()).hexdigest()}
(art/(name+'.json')).write_text(json.dumps(metadata,indent=2)+'\n')
print(json.dumps({'name':name,'exit_code':result.returncode,'elapsed_seconds':metadata['elapsed_seconds']}))
print((art/(name+'.log')).read_text()[-12000:])
sys.exit(result.returncode)
