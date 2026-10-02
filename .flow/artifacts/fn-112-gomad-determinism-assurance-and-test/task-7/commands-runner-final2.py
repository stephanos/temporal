from pathlib import Path
import json,os,subprocess,sys,time
root=Path('/Users/stephan/Workspace/temporal/gomad')
art=root/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-7'
label,cwd=sys.argv[1:3];command=sys.argv[3:]
env=os.environ.copy()
for key in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED']:env.pop(key,None)
stock='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin'
env.update({'PATH':stock+':'+env['PATH'],'GOMAD3_STOCK_GO':stock+'/go','GOFLAGS':'-tags=test_dep','GOWORK':'off','GOTOOLCHAIN':'local'})
build_key_before=(root/"tools/gomad3/.toolchain/build-key").read_text().strip()
started=time.time()
with (art/(label+'.log')).open('wb') as log:
 code=subprocess.call(command,cwd=root/cwd,env=env,stdout=log,stderr=subprocess.STDOUT)
record={'label':label,'command':command,'cwd':str(root/cwd),'environment':{k:env[k] for k in ['PATH','GOMAD3_STOCK_GO','GOFLAGS','GOWORK','GOTOOLCHAIN']},'removed_environment':['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED'],'exit_code':code,'elapsed_seconds':time.time()-started,'output':label+'.log','build_key_before':build_key_before,'build_key_after':(root/'tools/gomad3/.toolchain/build-key').read_text().strip()}
with (art/'commands.jsonl').open('a') as stream:stream.write(json.dumps(record)+'\n')
print(json.dumps({'label':label,'exit_code':code,'elapsed_seconds':record['elapsed_seconds'],'output':str(art/(label+'.log'))}))
sys.exit(code)
