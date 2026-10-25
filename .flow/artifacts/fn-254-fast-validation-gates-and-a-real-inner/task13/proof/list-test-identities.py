import json,subprocess,sys,pathlib,os
prefix=sys.argv[1]
rows=[json.loads(x) for x in open(f'.flow/tmp/fn25413/{prefix}-test-compile.jsonl')]
metadata=json.load(open(f'.flow/tmp/fn25413/{prefix}-metadata.json'))
packages={p['id']:p['name'] for p in metadata['packages']}
artifacts={r['executable']:r for r in rows if r.get('reason')=='compiler-artifact' and r['profile']['test'] and r.get('executable')}
identities=[]
stdlib=subprocess.check_output(["mise","exec","--","rustc","--print","target-libdir"],text=True).strip()
for binary,r in artifacts.items():
 def listing(extra):
  result=subprocess.run([binary,'--list','--format','terse',*extra],capture_output=True,text=True,env={**os.environ,"LD_LIBRARY_PATH":str(pathlib.Path(binary).parent)+":"+stdlib,"DYLD_FALLBACK_LIBRARY_PATH":str(pathlib.Path(binary).parent)+":"+stdlib})
  assert result.returncode==0,(binary,result.stderr)
  return [line.removesuffix(': test') for line in result.stdout.splitlines() if line.endswith(': test')]
 names=listing([]); ignored=set(listing(['--ignored']))
 for name in names:
  identities.append({'package':packages[r['package_id']],'target_kind':r['target']['kind'][0],'target':r['target']['name'],'test':name,'ignored':name in ignored})
identities.sort(key=lambda r:(r['package'],r['target_kind'],r['target'],r['test']))
pathlib.Path(f'.flow/tmp/fn25413/{prefix}-identities.json').write_text(json.dumps({'identities':identities,'binary_count':len(artifacts),'count':len(identities),'ignored':sum(x['ignored'] for x in identities)},indent=2)+'\n')
print(prefix,'binaries',len(artifacts),'identities',len(identities),'ignored',sum(x['ignored'] for x in identities))
