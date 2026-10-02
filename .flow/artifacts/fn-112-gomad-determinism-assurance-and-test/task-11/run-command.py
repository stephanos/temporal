import sys, os, json, subprocess, time, hashlib
from pathlib import Path
art=Path(__file__).resolve().parent
root=art.parents[3]
label, cwd, *argv=sys.argv[1:]
env=dict(os.environ)
for name in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO']:
 env.pop(name,None)
env.update(GOWORK='off',GOTOOLCHAIN='local',GOFLAGS='-tags=test_dep')
if label.startswith(('focused','vet')):
 env['GOMAD3_STOCK_GO']='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go'
if label.startswith('lint'):
 env['PATH']='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:'+env['PATH']
if label.startswith('host'):
 env['GOFLAGS']='-tags=test_dep -count=1'
 env['GOTOOLCHAIN']='local'
 env['PATH']='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:'+env['PATH']
sources={str(p.relative_to(root)):hashlib.sha256(p.read_bytes()).hexdigest() for p in (root/'tools/gomad3').rglob('*') if p.is_file() and p.suffix in ['.go','.mod','.sum'] and '.toolchain' not in str(p) and '.bin' not in str(p)}
for extra in ['Makefile','tools/gomad3/Makefile','tools/gomad3/version_generated.mk','tools/gomad3/.toolchain/build-key']:
 sources[extra]=hashlib.sha256((root/extra).read_bytes()).hexdigest()
start=time.time()
with (art/(label+'.log')).open('wb') as log:
 result=subprocess.run(argv,cwd=cwd,env=env,stdout=log,stderr=subprocess.STDOUT)
data=dict(label=label,argv=argv,cwd=cwd,environment={n:env.get(n) for n in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO','GOFLAGS','GOWORK','GOTOOLCHAIN','PATH']},exit=result.returncode,started_unix=start,elapsed_seconds=time.time()-start,toolchain_build_key=(root/'tools/gomad3/.toolchain/build-key').read_text().strip(),toolchain_go_sha256=hashlib.sha256((root/'tools/gomad3/.toolchain/bin/go').read_bytes()).hexdigest(),sources=sources,log_sha256=hashlib.sha256((art/(label+'.log')).read_bytes()).hexdigest())
(art/(label+'.json')).write_text(json.dumps(data,indent=2)+'\n')
print(json.dumps({k:v for k,v in data.items() if k!='sources'}))
print((art/(label+'.log')).read_text()[-15000:])
