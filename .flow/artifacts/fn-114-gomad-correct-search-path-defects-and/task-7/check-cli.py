import json,os,pathlib,subprocess,tempfile,time
artifacts=pathlib.Path(__file__).resolve().parent
root=artifacts.parents[3]
stock='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go'
env=os.environ.copy()
for name in ('GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO'):env.pop(name,None)
env['GOWORK']='off';env['GOFLAGS']='';env['GOTOOLCHAIN']='local'
receipts=[]
with tempfile.TemporaryDirectory(prefix='gomad-task7-cli-') as temp:
 temp=pathlib.Path(temp);binary=temp/'gomad'
 build=[stock,'build','-o',str(binary),'./cmd/gomad']
 start=time.monotonic()
 built=subprocess.run(build,cwd=root/'tools/gomad3',env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
 (artifacts/'cli-build.log').write_bytes(built.stdout)
 receipts.append({'name':'build','command':build,'exit_code':built.returncode,'elapsed_seconds':round(time.monotonic()-start,3)})
 if built.returncode:raise SystemExit(built.returncode)
 for name,args,attempted in [('populate',[],1),('fully-answered',[],0),('regression',['--guide-regression'],1)]:
  command=[str(binary),'explore','--json','--toolchain-root',str(root/'tools/gomad3/.toolchain'),'--guide','--corpus',str(temp/'corpus'),'--seeds','7','--parallel','1','--artifacts',str(temp/'campaigns'),'--working-dir',str(root/'tools/gomad3/internal/gomadtool/conformance/testdata'),*args,'go-run','./environment']
  start=time.monotonic();completed=subprocess.run(command,cwd=root,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
  (artifacts/('cli-'+name+'.jsonl')).write_bytes(completed.stdout)
  (artifacts/('cli-'+name+'.stderr')).write_bytes(completed.stderr)
  events=[json.loads(line) for line in completed.stdout.splitlines()]
  result=[event for event in events if event['type']=='result']
  receipt={'name':name,'command':command,'exit_code':completed.returncode,'elapsed_seconds':round(time.monotonic()-start,3),'result':result}
  receipts.append(receipt)
  (artifacts/'cli-exit-status.json').write_text(json.dumps(receipts,indent=2)+'\n')
  assert completed.returncode==0, receipt
  assert len(result)==1 and result[0].get('attempted',0)==attempted,receipt
  if name=='fully-answered':assert result[0]['guidance']=={'regression':False,'requested':1,'answered':1,'guided':0,'new_executions':0},receipt
  if name=='regression':assert result[0]['guidance']['regression'] is True and result[0]['guidance']['new_executions']==0,receipt
print(json.dumps([{'name':r['name'],'exit_code':r['exit_code'],'elapsed_seconds':r['elapsed_seconds']} for r in receipts]))
