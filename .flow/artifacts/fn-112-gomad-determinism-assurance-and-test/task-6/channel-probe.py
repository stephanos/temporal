from pathlib import Path
import subprocess,os,json,hashlib,datetime
root=Path(__file__).resolve().parent; repo=Path.cwd(); fixtures=repo/'tools/gomad3/internal/gomadtool/conformance/testdata'
probe=root/'positive-controls';probe.mkdir(exist_ok=True)
env=dict(os.environ)
for key in ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED']:env.pop(key,None)
env.update(CGO_ENABLED='0',TZ='UTC',GOFLAGS='-tags=test_dep',GOEXPERIMENT='nogreenteagc')
report={'started':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source_manifest_sha256':hashlib.sha256((root/'source-bindings.json').read_bytes()).hexdigest(),'toolchain_key':(repo/'tools/gomad3/.toolchain/build-key').read_text().strip(),'platform':'darwin/arm64','commands':[],'fixtures':{}}
for fixture,count in [('timer_ties',24),('runq_shuffle',1024)]:
 binary=probe/fixture;command=[str(repo/'tools/gomad3/.toolchain/bin/go'),'build','-tags=test_dep','-o',str(binary),'./'+fixture]
 result=subprocess.run(command,cwd=fixtures,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
 (probe/(fixture+'-build.stdout')).write_bytes(result.stdout);(probe/(fixture+'-build.stderr')).write_bytes(result.stderr)
 report['commands'].append({'argv':command,'cwd':str(fixtures),'exit':result.returncode})
 if result.returncode:raise RuntimeError(result.stderr.decode())
 observations=[];hashes={}
 for seed,iteration in [(seed,0) for seed in range(32)]+[(seed,iteration) for seed in [0,1,18446744073709551615] for iteration in range(1,11)]:
  child=dict(env,GOMADSEED=str(seed),GOMAXPROCS='1',GODEBUG='asyncpreemptoff=1')
  result=subprocess.run([str(binary)],cwd=fixtures,env=child,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=10)
  label=f'{fixture}-seed-{seed}-{iteration}';(probe/(label+'.stdout')).write_bytes(result.stdout);(probe/(label+'.stderr')).write_bytes(result.stderr)
  values=[int(value) for value in result.stdout.decode().strip().strip('[]').split()]
  valid=sorted(values)==list(range(count))
  digest=hashlib.sha256(result.stdout).hexdigest();hashes.setdefault(seed,set()).add(digest)
  observations.append({'seed':str(seed),'iteration':iteration,'exit':result.returncode,'stdout_sha256':digest,'stderr_sha256':hashlib.sha256(result.stderr).hexdigest(),'permutation_valid':valid})
  if result.returncode or not valid or result.stderr:raise RuntimeError(f'{label} failed')
 report['fixtures'][fixture]={'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'diversity_seeds':32,'distinct_diversity_orders':len({item['stdout_sha256'] for item in observations[:32]}),'observations':observations,'same_seed_hash_counts':{str(seed):len(hashes[seed]) for seed in [0,1,18446744073709551615]}}
 if report['fixtures'][fixture]['distinct_diversity_orders']<2 or any(len(values)!=1 for values in hashes.values()):raise RuntimeError(f'{fixture} did not meet repeatability/diversity')
report['finished']=datetime.datetime.now(datetime.timezone.utc).isoformat()
(root/'positive-controls.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({name:{key:value for key,value in info.items() if key!='observations'} for name,info in report['fixtures'].items()}))
