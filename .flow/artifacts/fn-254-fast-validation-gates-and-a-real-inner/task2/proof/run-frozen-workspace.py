import hashlib,json,os,pathlib,re,subprocess,time
root=pathlib.Path.cwd().resolve();proof=root/'.flow/tmp/fn2542';inventory=json.loads((proof/'workspace-frozen-inventory.json').read_text());metadata=json.loads((proof/'workspace-metadata.json').read_text());packages={p['name']:p for p in metadata['packages']}; package_names={p['id']:p['name'] for p in metadata['packages']}
reference=json.loads((proof/'cargo-test-runtime-environment.json').read_text())[0]['environment'];builds={}
for line in (proof/'workspace-artifacts.jsonl').read_text().splitlines():
 row=json.loads(line)
 if row.get('reason')=='build-script-executed':builds[package_names.get(row['package_id'],row['package_id'])]=row
plan=[]
for item in inventory['remaining_plan']:
 target=inventory['artifacts'][item['executable']];package=packages[target['package']];env=os.environ.copy()
 for key in list(env):
  if key.startswith('CARGO_PKG_') or key in ('CARGO_MANIFEST_DIR','CARGO_MANIFEST_PATH','OUT_DIR'):env.pop(key,None)
 for key in ('CARGO','LD_LIBRARY_PATH','PATH'):env[key]=reference[key]
 env['CARGO_MANIFEST_PATH']=package['manifest_path'];env['CARGO_MANIFEST_DIR']=str(pathlib.Path(package['manifest_path']).parent)
 env['CARGO_PKG_NAME']=package['name'];env['CARGO_PKG_VERSION']=package['version'];version,_,pre=package['version'].partition('-')
 for key,value in zip(('MAJOR','MINOR','PATCH'),version.split('.')):env['CARGO_PKG_VERSION_'+key]=value
 env['CARGO_PKG_VERSION_PRE']=pre
 for key,field in [('DESCRIPTION','description'),('HOMEPAGE','homepage'),('REPOSITORY','repository'),('LICENSE','license'),('LICENSE_FILE','license_file'),('README','readme'),('RUST_VERSION','rust_version')]:env['CARGO_PKG_'+key]=package.get(field) or ''
 env['CARGO_PKG_AUTHORS']=':'.join(package.get('authors',[]))
 if target['package'] in builds:
  build=builds[target['package']];env['OUT_DIR']=build['out_dir'];env.update(dict(build.get('env',[])))
 binary=pathlib.Path(target['frozen_copy']);assert hashlib.sha256(binary.read_bytes()).hexdigest()==target['sha256']
 observed=target['previous']['passed']+target['previous']['ignored'];args=[]
 if observed:args=[item['expected_tests'][0],'--exact']
 command=[str(binary),*args,'--nocapture'];plan.append({**item,'command':command,'cwd':env['CARGO_MANIFEST_DIR'],'runtime_bindings':{key:value for key,value in env.items() if key.startswith('CARGO_PKG_') or key in ('CARGO','CARGO_MANIFEST_DIR','CARGO_MANIFEST_PATH','OUT_DIR','LD_LIBRARY_PATH','PATH','RUST_TEST_THREADS','UMPIRE_BACKENDS')},'_env':env})
(proof/'frozen-execution-plan.json').write_text(json.dumps([{k:v for k,v in item.items() if k!='_env'} for item in plan],indent=2)+'\n')
print(f'FROZEN_PLAN commands={len(plan)} source={inventory["commit"]}',flush=True)
results=[]
for i,item in enumerate(plan):
 target=inventory['artifacts'][item['executable']];log=proof/f'frozen-{i:03d}.log';started=time.monotonic();print(f'RUN {i:03d} {target["package"]} {target["target"]} tests={len(item["expected_tests"])}',flush=True)
 with log.open('w') as out:completed=subprocess.run(['timeout','2700',*item['command']],cwd=item['cwd'],env=item['_env'],stdout=out,stderr=subprocess.STDOUT)
 body=log.read_text();summaries=re.findall(r'^test result: (ok|FAILED)\. (\d+) passed; (\d+) failed; (\d+) ignored;',body,re.M)
 result={k:v for k,v in item.items() if k!='_env'};result.update(rc=completed.returncode,seconds=round(time.monotonic()-started,3),log=str(log),summaries=summaries)
 (proof/f'frozen-{i:03d}.json').write_text(json.dumps(result,indent=2)+'\n');results.append(result);print(f'END {i:03d} rc={completed.returncode} seconds={result["seconds"]}',flush=True)
 if completed.returncode!=0:break
 assert summaries and summaries[-1][0]=='ok' and int(summaries[-1][2])==0 and int(summaries[-1][1])+int(summaries[-1][3])==len(item['expected_tests']),('selection mismatch',i,summaries,item['expected_tests'])
(proof/'frozen-execution-results.json').write_text(json.dumps(results,indent=2)+'\n')
raise SystemExit(0 if len(results)==len(plan) and all(row['rc']==0 for row in results) else 1)
