import collections, hashlib, json, os, pathlib, re, subprocess, time
root=pathlib.Path.cwd().resolve(); proof=root/'.flow/tmp/fn2542'; original=(proof/'workspace-required-equivalence.log').read_text()
parts=re.split(r'^     Running .*?\(([^\n]+)\)\n',original,flags=re.M)
previous={}
for i in range(1,len(parts),2):
 body=parts[i+1]
 previous[parts[i]]={'complete':bool(re.search(r'^test result: ok\.',body,re.M)), 'passed':re.findall(r'^test ([\w:]+)(?: - should panic)? \.\.\. ok$',body,re.M), 'ignored':re.findall(r'^test ([^\n]+) \.\.\. ignored(?:,.*)?$',body,re.M)}
for result in previous.values():
 result['passed']=[name.removesuffix(' - should panic') for name in result['passed']]
artifacts={}
listing_env=os.environ.copy()
listing_env['LD_LIBRARY_PATH']='/home/agent/.rustup/toolchains/1.99.1-aarch64-unknown-linux-gnu/lib/rustlib/aarch64-unknown-linux-gnu/lib'
for line in (proof/'workspace-artifacts.jsonl').read_text().splitlines():
 row=json.loads(line)
 if row.get('reason')!='compiler-artifact' or not row.get('executable') or not row.get('profile',{}).get('test'):continue
 executable=row['executable']; target=row['target']; package=row['package_id'].rsplit('#',1)[-1].split('@')[0]
 listing=subprocess.run([executable,'--list'],cwd=pathlib.Path(row['manifest_path']).parent,text=True,capture_output=True,check=True,env=listing_env)
 names=re.findall(r'^(.+): test$',listing.stdout,re.M)
 kind=target['kind'][0]
 selector=['--lib'] if kind in ('lib','proc-macro') else ['--'+kind,target['name']]
 artifacts[executable]={'package':package,'target':target['name'],'kind':kind,'selector':selector,'tests':names,'sha256':hashlib.sha256(pathlib.Path(executable).read_bytes()).hexdigest(),'previous':previous.get(executable,{'complete':False,'passed':[],'ignored':[]})}
assert artifacts and all(exe in artifacts for exe in previous), 'unmatched original executable'
plan=[]
for executable,target in artifacts.items():
 prior=target['previous']; observed=set(prior['passed']+prior['ignored'])
 assert observed <= set(target['tests']), ('unexpected observed test',executable)
 if prior['complete']:continue
 remaining=[name for name in target['tests'] if name not in observed]
 base=['mise','exec','--','cargo','test','--locked','--features','umpire-tlc/strategy-equivalence','-p',target['package'],*target['selector']]
 if observed:
  for name in remaining:plan.append({'executable':executable,'expected_tests':[name],'command':base+[name,'--','--exact','--nocapture']})
 else:plan.append({'executable':executable,'expected_tests':remaining,'command':base+['--','--nocapture']})
inventory={'commit':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'feature':'umpire-tlc/strategy-equivalence','artifacts':artifacts,'remaining_plan':plan}
(proof/'workspace-resume-inventory.json').write_text(json.dumps(inventory,indent=2)+'\n')
print(json.dumps({'targets':len(artifacts),'logical_tests':sum(len(t['tests']) for t in artifacts.values()),'complete_targets_before_timeout':sum(t['previous']['complete'] for t in artifacts.values()),'remaining_commands':len(plan)}),flush=True)
results=[]
for i,item in enumerate(plan):
 log=proof/f'resume-{i:03d}.log'; status=proof/f'resume-{i:03d}.json'; started=time.monotonic()
 print(f"RUN {i:03d} {artifacts[item['executable']]['package']} {artifacts[item['executable']]['target']} tests={len(item['expected_tests'])}",flush=True)
 with log.open('w') as out: completed=subprocess.run(['timeout','2700',*item['command']],cwd=root/'model',stdout=out,stderr=subprocess.STDOUT)
 body=log.read_text(); passed=re.findall(r'^test ([\w:]+)(?: - should panic)? \.\.\. ok$',body,re.M); ignored=re.findall(r'^test ([^\n]+) \.\.\. ignored(?:,.*)?$',body,re.M)
 passed=[name.removesuffix(' - should panic') for name in passed]
 result={**item,'rc':completed.returncode,'seconds':round(time.monotonic()-started,3),'log':str(log),'passed':passed,'ignored':ignored}
 status.write_text(json.dumps(result,indent=2)+'\n');results.append(result)
 print(f"END {i:03d} rc={completed.returncode} seconds={result['seconds']}",flush=True)
 if completed.returncode!=0:break
 summaries=re.findall(r'^test result: ok\. (\d+) passed; 0 failed; (\d+) ignored;',body,re.M)
 assert summaries and sum(map(int,summaries[-1]))==len(item['expected_tests']), ('selection count mismatch',item['command'],summaries,item['expected_tests'])
 assert set(passed+ignored)<=set(item['expected_tests']), ('selection identity mismatch',item['command'],passed,ignored,item['expected_tests'])
(proof/'workspace-resume-results.json').write_text(json.dumps(results,indent=2)+'\n')
raise SystemExit(0 if len(results)==len(plan) and all(row['rc']==0 for row in results) else 1)
