import hashlib,json,os,pathlib,subprocess,sys,time
root=pathlib.Path('/Users/stephan/Workspace/temporal/gomad')
out=root/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-13'
def digest(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def sources():
 result={}
 for directory,dirs,files in os.walk(root/'tools/gomad3'):
  dirs[:]=[d for d in dirs if not d.startswith('.')]
  for name in files:
   p=pathlib.Path(directory)/name
   if p.suffix in ('.go','.mod','.sum','.md'):result[str(p.relative_to(root))]=digest(p)
 return result
if sys.argv[1]=='before':
 owned=['tools/gomad3/runner/replay_operation.go','tools/gomad3/runner/watchdog_replay_test.go','tools/gomad3/cmd/gomad/watchdog_replay_e2e_test.go']
 for name in owned:
  p=root/name;q=out/'before'/name;q.parent.mkdir(parents=True,exist_ok=True)
  if p.exists():q.write_bytes(p.read_bytes())
 (out/'before-sources.json').write_text(json.dumps(sources(),indent=2)+'\n')
 (out/'owned-sources.json').write_text(json.dumps(owned,indent=2)+'\n')
else:
 label=sys.argv[1];argv=sys.argv[2:];env=os.environ.copy()
 for k in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO']:env.pop(k,None)
 env.update(GOWORK='off',GOFLAGS='-tags=test_dep -count=1',GOTOOLCHAIN='local')
 env['PATH']='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:'+env['PATH']
 before=sources();start=time.time()
 with (out/(label+'.log')).open('wb') as log:result=subprocess.run(argv,cwd=root/'tools/gomad3',env=env,stdout=log,stderr=subprocess.STDOUT)
 info=dict(label=label,argv=argv,cwd=str(root/'tools/gomad3'),environment={k:env.get(k) for k in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_STOCK_GO','GOWORK','GOFLAGS','GOTOOLCHAIN','PATH']},exit=result.returncode,started_unix=start,elapsed_seconds=time.time()-start,sources=before,log_sha256=digest(out/(label+'.log')))
 (out/(label+'.json')).write_text(json.dumps(info,indent=2)+'\n');print(label,result.returncode,round(info['elapsed_seconds'],2));print((out/(label+'.log')).read_text()[-3500:])
