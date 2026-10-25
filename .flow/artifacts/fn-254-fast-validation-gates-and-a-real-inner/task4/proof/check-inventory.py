import collections,hashlib,json,pathlib,subprocess,sys,os
libdir=subprocess.check_output(["mise","exec","--","rustc","--print","target-libdir"],text=True).strip()
os.environ["LD_LIBRARY_PATH"]=libdir+os.pathsep+os.environ.get("LD_LIBRARY_PATH", "")
P=pathlib.Path(__file__).parent
meta=json.loads((P/'before-metadata.json').read_text())
pkgs={p['id']:p['name'] for p in meta['packages']}
artifacts={}
for file in sys.argv[1:]:
 for line in pathlib.Path(file).read_text().splitlines():
  try:d=json.loads(line)
  except ValueError:continue
  if d.get('reason')=='compiler-artifact' and d.get('executable') and d['profile']['test']:
   package=pkgs[d['package_id']];target=d['target']
   artifacts[package,target['kind'][0],target['name']]=d
assert artifacts,'no compiled test artifacts'
cases=[];bins=[]
for (package,kind,target),d in sorted(artifacts.items()):
 exe=d['executable']
 def listing(extra):
  run=subprocess.run([exe,'--list','--format=terse',*extra],text=True,capture_output=True,check=True)
  return {l[:-6] for l in run.stdout.splitlines() if l.endswith(': test')}
 tests=listing([]);ignored=listing(['--ignored'])
 assert ignored<=tests
 for test in sorted(tests):cases.append(dict(package=package,target_kind=kind,target=target,test=test,ignored=test in ignored))
 bins.append(dict(package=package,target_kind=kind,target=target,count=len(tests),ignored=len(ignored),executable=exe,sha256=hashlib.sha256(pathlib.Path(exe).read_bytes()).hexdigest(),features=d['features'],profile=d['profile']))
expected=json.loads((P/'expected-mapping.json').read_text())
selected={c['package'] for c in cases}
wanted=[r['after'] for r in expected['cases'] if r['after']['package'] in selected]
def key(c):return c['package'],c['target_kind'],c['target'],c['test'],c['ignored']
missing=sorted(set(map(key,wanted))-set(map(key,cases)));extra=sorted(set(map(key,cases))-set(map(key,wanted)))
result=dict(binaries=bins,cases=cases,missing=missing,extra=extra,logical_count=len(cases),ignored=sum(c['ignored'] for c in cases),packages=sorted(selected),pass_exact=not missing and not extra)
(P/os.environ.get('INVENTORY_OUTPUT','after-inventory.json')).write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({k:v for k,v in result.items() if k not in ['binaries','cases']},indent=2))
assert not missing and not extra,'inventory is not equivalent'
