import hashlib,json,os,subprocess,sys,time
from pathlib import Path
root=Path(__file__).resolve().parents[4]
out=Path(__file__).resolve().parent
name=sys.argv[1]
config=json.loads(sys.argv[2])
env=os.environ.copy()
removed=['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO']
for key in removed: env.pop(key,None)
stock='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go'
env.update(PATH=str(Path(stock).parent)+os.pathsep+env['PATH'],GOWORK='off',GOFLAGS='-tags=test_dep',GOTOOLCHAIN='local')
env.update(config.get('env',{}))
keys=['PATH','GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO','GOWORK','GOFLAGS','GOTOOLCHAIN','GOENV','CGO_ENABLED','GOEXPERIMENT','GOCACHE','GOPATH','GOPROXY','GOSUMDB','GOMAD3_BOOTSTRAP_GO','GOLANGCI_LINT_BASE_REV','GOLANGCI_LINT_FIX']
metadata={'argv':config['argv'],'cwd':str(root/config.get('cwd','.')),'removed_environment':removed,'relevant_environment':{key:env.get(key) for key in keys},'before_source_hashes_sha256':hashlib.sha256((out/'before-source-hashes.json').read_bytes()).hexdigest()}
if (out/'frozen-source-hashes.json').is_file():metadata['frozen_source_hashes_sha256']=hashlib.sha256((out/'frozen-source-hashes.json').read_bytes()).hexdigest()
(out/(name+'.command.json')).write_text(json.dumps(metadata,indent=2)+'\n')
start=time.monotonic()
with (out/(name+'.log')).open('wb') as log:
 result=subprocess.run(config['argv'],cwd=metadata['cwd'],env=env,stdout=log,stderr=subprocess.STDOUT)
(out/(name+'.exit')).write_text(str(result.returncode)+'\n')
(out/(name+'.result.json')).write_text(json.dumps({'exit':result.returncode,'wall_seconds':time.monotonic()-start,'log_sha256':hashlib.sha256((out/(name+'.log')).read_bytes()).hexdigest()},indent=2)+'\n')
print(name,'exit',result.returncode,'elapsed',round(time.monotonic()-start,3))
print((out/(name+'.log')).read_text()[-1800:])
