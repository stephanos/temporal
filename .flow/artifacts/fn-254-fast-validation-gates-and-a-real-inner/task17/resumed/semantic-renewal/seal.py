from pathlib import Path
import json,hashlib,subprocess,shutil
out=Path(__file__).resolve().parent;root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');old=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/inventory-renewal'
load=lambda p:json.loads(p.read_text());write=lambda name,x:(out/name).write_text(json.dumps(x,indent=2)+'\n');SHA=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
ns={'__name__':'proof_import','__file__':str(out.parent/'admission-proof/equivalence.py')};exec(compile((out.parent/'admission-proof/equivalence.py').read_text(),str(out.parent/'admission-proof/equivalence.py'),'exec'),ns)
canonical=lambda k:'lib' if k in ['rlib','cdylib','dylib','staticlib'] else k
result={}
for scope,workspace,packages in [('activity','temporal',{'umpire-temporal','umpire-temporal-api','umpire-temporal-realization-macros','umpire-temporal-guest','umpire-temporal-tests'}),('framework-affected','framework',{'umpire-backend-process','xtask'}),('temporal-xtask','temporal',{'temporal-xtask'})]:
 rows=load(out/(scope+'-cargo-lists.json'));inventory=load(out/(scope+'-nextest-all.stdout.log'));cargo=ns['cargo'](rows,workspace)
 keep=lambda r:scope!='activity' or r[1]!='umpire-temporal-tests' or r[2:5]==('test','fixtures','fixtures')
 cargo={r for r in cargo if keep(r)};nxt,default=ns['nextest'](inventory,workspace);assert cargo==nxt
 cb={(r['package'],canonical(r['kind']),r['binary']) for r in rows if scope!='activity' or r['package']!='umpire-temporal-tests' or (canonical(r['kind']),r['binary'])==('test','fixtures')}
 nb={(s['package-name'],s['kind'],s['binary-name']) for s in inventory['rust-suites'].values() if s['status']=='listed'}
 assert cb==nb,(scope,cb-nb,nb-cb)
 prior={tuple(r) for r in load(old/(workspace+'-identity-set.json')) if r[1] in packages and keep(tuple(r))}
 added=sorted(cargo-prior);removed=sorted(prior-cargo)
 r=dict(equal=True,scope='Cargo/nextest shipping-profile listings; no test bodies',identities=len(cargo),nonignored=len(default),ignored=sum(r[-1] for r in cargo),binary_footprint=len(cb),prior_F2_identities=len(prior),added=added,removed=removed,per_package={p:sum(r[1]==p for r in cargo) for p in sorted(packages)},eligible_doctest_targets=load(out/(scope+'-doctest-eligibility.json')),actual_identity_set=sorted(cargo))
 write(scope+'-equivalence-final.json',r);result[scope]={k:v for k,v in r.items() if k!='actual_identity_set'}
 if scope!='temporal-xtask':assert not removed
 else:assert [r[4] for r in removed]==['gate::tests::spawn_failure_names_the_stage']
 if scope=='activity':assert len(cargo)==148 and not added and not removed and not any(r[-1] for r in cargo)
oldframework=load(old/'framework-identity-set.json');oldtemporal=load(old/'temporal-identity-set.json')
result['historical_full_workspace_projection']={workspace:dict(F2_count=len(prior),projected_current_count=len(prior)+len(result[scope]['added'])-len(result[scope]['removed']),meaning='Historical F2 identity set plus freshly listed affected-package delta; not a new full-workspace inventory or execution') for workspace,prior,scope in [('framework',oldframework,'framework-affected'),('temporal',oldtemporal,'temporal-xtask')]}
before=load(out/'full-compile-inputs-before.json');after={p:SHA(root/p) for p in before};assert before==after;write('full-compile-inputs-after.json',after)
snap=load(root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed/temporal-activity-required.json')['snapshots_before'];assert len(snap)==202 and all(SHA(root/p)==h for p,h in snap.items());write('protected-snapshots.json',snap)
ns2={'__name__':'proof_import','__file__':str(out.parent/'admission-proof/driver.py')};exec(compile((out.parent/'admission-proof/driver.py').read_text(),str(out.parent/'admission-proof/driver.py'),'exec'),ns2);final=ns2['snapshot'](root);assert final==load(out/'root-source-before.json');assert final['registry_tuples']==load(old/'root-source-final.json')['registry_tuples'];write('root-source-final.json',final)
binaries=load(out/'successful-build-final.json')['binaries'];assert all(SHA(Path(p))==h for p,h in binaries.items())
assert subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)==''
target=root/'.flow/tmp/fn2546-resumed/target';go=root/'.flow/tmp/fn2545-temporal-api/go-cache';size=lambda p:int(subprocess.check_output(['du','-sb',str(p)],text=True).split()[0]);base=load(out/'resource-baseline.json');growth=size(target)+size(go)-base['target_bytes'];assert growth<=3*1024**3 and shutil.disk_usage(root).free>=8*1024**3
result.update(root_head=final['head'],source_binaries_locks_snapshots_unchanged=True,registry_tuples=119,protected_snapshots=202,compile_input_files=len(before),binary_input_files=len(final['binary_source_files']),broader_execution_input_files=len(final['source_files']),combined_growth_bytes=growth,free_bytes=shutil.disk_usage(root).free,phase_complete='F3.1 inventories and native refresh only',test_bodies_executed=False,java_invoked=False,shipping_profile_overrides='none: all CARGO_PROFILE_* removed for inventories',native_refresh_profile='dev; CARGO_PROFILE_DEV_DEBUG=0; incremental0',pending=['Root identified locator-only lifecycle control needs closing inherited FD10/11 before body execution; listing is unaffected.','No actual six/five quick workloads or production standalone executed.','No new global-full-workspace execution or baseline timing campaign.'])
write('summary.json',result)
print(json.dumps({k:({x:y for x,y in v.items() if x!='eligible_doctest_targets'} if isinstance(v,dict) else v) for k,v in result.items()},indent=2))
