import hashlib,json,os,pathlib,re,subprocess,time
root=pathlib.Path.cwd().resolve();proof=root/'.flow/tmp/fn2542';own=proof/'helper-a'
allocation=json.loads((proof/'parallel-proof-allocation.json').read_text());indices=allocation['helper_a'];plan=json.loads((proof/'frozen-execution-plan.json').read_text());inventory=json.loads((proof/'workspace-frozen-inventory.json').read_text())
metadata=json.loads((proof/'workspace-metadata.json').read_text());packages={p['name']:p for p in metadata['packages']};builds={}
for line in (proof/'workspace-artifacts.jsonl').read_text().splitlines():
 row=json.loads(line)
 if row.get('reason')=='build-script-executed':builds[row['package_id']]=row
head=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip();assert head==allocation['source_commit']==inventory['commit']
source=json.loads((proof/'frozen-source-hashes.json').read_text());changed=[p for p,sha in source['tracked_active_model_files'].items() if hashlib.sha256((root/p).read_bytes()).hexdigest()!=sha];assert not changed,changed
checked={}
for index in indices:
 item=plan[index];target=inventory['artifacts'][item['executable']];binary=pathlib.Path(item['command'][0]);sha=hashlib.sha256(binary.read_bytes()).hexdigest();assert sha==target['sha256'],str(binary);checked[str(index)]={'binary':str(binary),'sha256':sha,'package':target['package'],'target':target['target'],'expected_tests':item['expected_tests']}
(own/'preflight.json').write_text(json.dumps({'source_commit':head,'verified_active_source_files':len(source['tracked_active_model_files']),'indices':indices,'binaries':checked,'thread_amendment':'owner authorized RUST_TEST_THREADS=1 per proof helper, max two simultaneous helpers / four model JVMs'},indent=2)+'\n')
results=[];cohort_started=time.monotonic()
print(f'BEGIN helper_a targets={len(indices)} source={head} test_threads=1',flush=True)
for index in indices:
 item=plan[index];target=inventory['artifacts'][item['executable']];package=packages[target['package']]
 env=os.environ.copy()
 for key in list(env):
  if key.startswith('CARGO_PKG_') or key in ('CARGO_MANIFEST_DIR','CARGO_MANIFEST_PATH','OUT_DIR'):env.pop(key,None)
 env.update(item['runtime_bindings'])
 build=builds.get(package['id'])
 if build:env.update(dict(build.get('env',[])))
 env.update(CARGO_TARGET_DIR='/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2542-tlc',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',CARGO_INCREMENTAL='0',CARGO_BUILD_JOBS='2',UMPIRE_BACKENDS='require',RUST_TEST_THREADS='1')
 env.pop('BASH_ENV',None)
 assert hashlib.sha256(pathlib.Path(item['command'][0]).read_bytes()).hexdigest()==target['sha256']
 command=['timeout','2700',*item['command']];log=own/f'{index:03d}.log';started=time.monotonic()
 print(f'RUN {index:03d} {target["package"]} {target["target"]} tests={len(item["expected_tests"])}',flush=True)
 with log.open('w') as out:result=subprocess.run(command,cwd=item['cwd'],env=env,stdout=out,stderr=subprocess.STDOUT)
 body=log.read_text();summaries=re.findall(r'^test result: (ok|FAILED)\. (\d+) passed; (\d+) failed; (\d+) ignored;',body,re.M)
 selection_ok=bool(summaries and summaries[-1][0]=='ok' and int(summaries[-1][2])==0 and int(summaries[-1][1])+int(summaries[-1][3])==len(item['expected_tests']))
 row={'index':index,'package':target['package'],'target':target['target'],'command':command,'cwd':item['cwd'],'runtime_bindings':item['runtime_bindings'],'overrides':{'RUST_TEST_THREADS':'1','UMPIRE_BACKENDS':'require','CARGO_TARGET_DIR':env['CARGO_TARGET_DIR']},'original_build_script_env':build.get('env',[]) if build else [],'sha256':target['sha256'],'expected_tests':item['expected_tests'],'rc':result.returncode,'seconds':round(time.monotonic()-started,3),'summaries':summaries,'selection_ok':selection_ok,'log':str(log)}
 (own/f'{index:03d}.json').write_text(json.dumps(row,indent=2)+'\n');results.append(row)
 (own/'results.json').write_text(json.dumps(results,indent=2)+'\n')
 print(f'END {index:03d} rc={result.returncode} selected_count_ok={selection_ok} seconds={row["seconds"]}',flush=True)
 if result.returncode!=0 or not selection_ok:break
passed=len(results)==len(indices) and all(r['rc']==0 and r['selection_ok'] for r in results)
summary={'cohort':'helper_a','source_commit':head,'assigned_indices':indices,'completed_indices':[r['index'] for r in results],'all_passed':passed,'seconds':round(time.monotonic()-cohort_started,3),'case_count':sum(len(r['expected_tests']) for r in results),'result_file':str(own/'results.json'),'running_commands':[]}
(own/'summary.json').write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps(summary),flush=True)
raise SystemExit(0 if passed else 1)
