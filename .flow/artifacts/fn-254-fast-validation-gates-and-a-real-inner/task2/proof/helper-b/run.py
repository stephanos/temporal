import hashlib,json,os,pathlib,re,subprocess,time
root=pathlib.Path.cwd().resolve();proof=root/'.flow/tmp/fn2542';out=proof/'helper-b'
allocation=json.loads((proof/'parallel-proof-allocation.json').read_text());indices=allocation['helper_b'];plan=json.loads((proof/'frozen-execution-plan.json').read_text());inventory=json.loads((proof/'workspace-frozen-inventory.json').read_text());metadata=json.loads((proof/'workspace-metadata.json').read_text());packages={p['id']:p for p in metadata['packages']}
assert subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()==allocation['source_commit']
builds={row['package_id']:row for row in map(json.loads,(proof/'workspace-artifacts.jsonl').read_text().splitlines()) if row.get('reason')=='build-script-executed'}
for index in indices:
 item=plan[index];artifact=inventory['artifacts'][item['executable']];binary=pathlib.Path(item['command'][0]);assert str(binary)==artifact['frozen_copy'];assert hashlib.sha256(binary.read_bytes()).hexdigest()==artifact['sha256'];assert artifact['package_id'] in packages
print(json.dumps({'cohort':'helper_b','targets':len(indices),'logical_cases':sum(len(plan[i]['expected_tests']) for i in indices),'source_commit':allocation['source_commit'],'threads':1}),flush=True)
results=[]
for ordinal,index in enumerate(indices):
 item=plan[index];artifact=inventory['artifacts'][item['executable']];env=os.environ.copy()
 for key in list(env):
  if key.startswith('CARGO_PKG_') or key in ('CARGO_MANIFEST_DIR','CARGO_MANIFEST_PATH','OUT_DIR'):env.pop(key,None)
 env.update(item['runtime_bindings']);env.update(dict(builds.get(artifact['package_id'],{}).get('env',[])))
 env.update(CARGO_TARGET_DIR='/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2542-tlc',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',CARGO_INCREMENTAL='0',CARGO_BUILD_JOBS='2',UMPIRE_BACKENDS='require',RUST_TEST_THREADS='1');env.pop('BASH_ENV',None)
 binary=pathlib.Path(item['command'][0]);binary_sha=hashlib.sha256(binary.read_bytes()).hexdigest();assert binary_sha==artifact['sha256']
 log=out/f'{index:03d}.log';started=time.monotonic();print(f'RUN {index:03d} {artifact["package"]} {artifact["target"]} cases={len(item["expected_tests"])}',flush=True)
 with log.open('w') as captured:completed=subprocess.run(['timeout','2700',*item['command']],cwd=item['cwd'],env=env,stdout=captured,stderr=subprocess.STDOUT)
 body=log.read_text();summaries=re.findall(r'^test result: (ok|FAILED)\. (\d+) passed; (\d+) failed; (\d+) ignored;',body,re.M)
 selection_ok=bool(summaries and summaries[-1][0]=='ok' and int(summaries[-1][2])==0 and int(summaries[-1][1])+int(summaries[-1][3])==len(item['expected_tests']))
 status='passed' if completed.returncode==0 and selection_ok else 'failed' if completed.returncode!=0 else 'inconclusive_selection'
 row={'index':index,'package':artifact['package'],'target':artifact['target'],'kind':artifact['kind'],'source_commit':allocation['source_commit'],'binary_sha256':binary_sha,'command':['timeout','2700',*item['command']],'cwd':item['cwd'],'runtime_bindings':{**item['runtime_bindings'],**dict(builds.get(artifact['package_id'],{}).get('env',[])),'RUST_TEST_THREADS':'1','UMPIRE_BACKENDS':'require','CARGO_TARGET_DIR':env['CARGO_TARGET_DIR']},'expected_tests':item['expected_tests'],'rc':completed.returncode,'status':status,'summaries':summaries,'seconds':round(time.monotonic()-started,3),'log':str(log),'log_sha256':hashlib.sha256(log.read_bytes()).hexdigest()}
 (out/f'{index:03d}.json').write_text(json.dumps(row,indent=2)+'\n');results.append(row);(out/'results.json').write_text(json.dumps(results,indent=2)+'\n')
 (out/'progress.json').write_text(json.dumps({'completed':len(results),'total':len(indices),'last_index':index,'last_status':status,'remaining':indices[ordinal+1:]},indent=2)+'\n')
 print(f'END {index:03d} rc={completed.returncode} status={status} seconds={row["seconds"]}',flush=True)
 if status!='passed':break
success=len(results)==len(indices) and all(r['status']=='passed' for r in results)
summary={'cohort':'helper_b','source_commit':allocation['source_commit'],'status':'passed' if success else 'failed_or_inconclusive','completed_targets':len(results),'allocated_targets':len(indices),'selected_cases':sum(len(row['expected_tests']) for row in results),'results':str(out/'results.json'),'remaining_indices':indices[len(results):]}
(out/'summary.json').write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps(summary),flush=True)
raise SystemExit(0 if success else 1)
