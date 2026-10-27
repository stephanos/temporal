from pathlib import Path
import json,hashlib,subprocess,shutil
out=Path(__file__).resolve().parent
root=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
ns={'__name__':'proof_import','__file__':str(out/'equivalence.py')};exec(compile((out/'equivalence.py').read_text(),str(out/'equivalence.py'),'exec'),ns)
load=lambda name:json.loads((out/name).read_text())
write=lambda name,data:(out/name).write_text(json.dumps(data,indent=2)+'\n')
SHA=lambda path:hashlib.sha256(path.read_bytes()).hexdigest()
summary={}
for scope in ['framework','temporal']:
    rows=load(scope+'-cargo-lists.json'); inventory=load(scope+'-nextest-all.stdout.log')
    cargo=ns['cargo'](rows,scope);nxt,default=ns['nextest'](inventory,scope)
    assert cargo==nxt
    actualdefault,_=ns['nextest'](load(scope+'-nextest-default.stdout.log'),scope)
    assert actualdefault=={r for r in cargo if not r[-1]}
    cb={(r['package'],'lib' if r['kind'] in ['rlib','cdylib','dylib','staticlib'] else r['kind'],r['binary']) for r in rows}
    nb={(s['package-name'],s['kind'],s['binary-name']) for s in inventory['rust-suites'].values() if s['status']=='listed'}
    assert cb==nb
    metadata=load(scope+'-metadata.stdout.log');members=[p for p in metadata['packages'] if p['id'] in metadata['workspace_members']]
    zeros=[dict(package=p['name'],manifest=p['manifest_path'],targets=p['targets']) for p in members if not any(r[1]==p['name'] and not r[-1] for r in cargo)]
    write(scope+'-zero-harness-members.json',zeros)
    write(scope+'-identity-set.json',sorted(cargo))
    summary[scope]=dict(identities=len(cargo),ignored=sum(r[-1] for r in cargo),actual_default_subset=len(actualdefault),binary_footprint=len(cb),empty_binaries=sum(not any(r[1:4]==b for r in cargo) for b in cb),members=len(members),zero_nonignored_members=[p['package'] for p in zeros],eligible_doctest_targets=len(load(scope+'-doctest-eligibility.json')),binary_identity_default_equality=True)
for name,scope,packages,condition in [
 ('framework-inner','framework',{'umpire-checker'},lambda r:True),
 ('kettle-quick','framework',{'umpire-apalache','umpire-fixtures','umpire-kettle-guest','umpire-reference-kettle','umpire-suite','umpire-tla'},lambda r:True),
 ('temporal-boundary','temporal',{'umpire-temporal-tests'},lambda r:r[2:4]==('test','boundary'))]:
    cargo={r for r in ns['cargo'](load(scope+'-cargo-lists.json'),scope) if r[1] in packages and condition(r)}
    nxt,default=ns['nextest'](load(name+'-nextest-all.stdout.log'),scope)
    result=dict(identity_equality=cargo==nxt,identities=len(cargo),ignored=sum(r[-1] for r in cargo),missing=sorted(cargo-nxt),extra=sorted(nxt-cargo),scope='listing only; no bodies')
    write(name+'-equivalence.json',result);assert result['identity_equality'];summary[name]=result
for name,expected in [('kettle',{'umpire-apalache','umpire-fixtures','umpire-kettle-guest','umpire-reference-kettle','umpire-suite','umpire-tla'}),('activity',{'umpire-temporal','umpire-temporal-api','umpire-temporal-guest','umpire-temporal-realization-macros','umpire-temporal-tests'})]:
    result=load(name+'-public-quick-select-only.stdout.log');actual={p for w in result['workspaces'] for p in w['packages']};assert actual==expected
    assert len(result['changed'])==1
    summary[name+'-selection']=dict(packages=sorted(actual),actual_public_quick=True,nonsemantic_comment_edit=True,restored=load(name+'-edit-restoration.json')['exact_original_bytes_restored'])
historical=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed/fixture-identity-equivalence.json'
mapping=json.loads(historical.read_text());current={r[4] for r in ns['cargo'](load('framework-cargo-lists.json'),'framework') if r[1:4]==('umpire-fixtures','test','fixtures')}
expected=set(mapping['new_fixture_cases'])|set(mapping['preserved_other_identities']);assert current==expected and len(current)==84
summary['fixture_umbrella_mapping']=dict(historical_path=str(historical.relative_to(root)),sha256=SHA(historical),old_count=61,removed_umbrella='fixtures',preserved=60,added_named_cases=24,current_count=84,exact_current_names_match=True,execution=False)
before=load('full-compile-inputs-before.json');after={p:SHA(root/p) for p in before};assert before==after;write('full-compile-inputs-after.json',after)
snapfile=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed/temporal-activity-required.json';snap=json.loads(snapfile.read_text())['snapshots_before'];assert len(snap)==202 and all(SHA(root/p)==h for p,h in snap.items());write('protected-snapshots.json',snap)
ns2={'__name__':'proof_import','__file__':str(out.parent/'admission-proof/driver.py')};exec(compile((out.parent/'admission-proof/driver.py').read_text(),str(out.parent/'admission-proof/driver.py'),'exec'),ns2);after_native=ns2['snapshot'](root);assert after_native==load('root-source-before.json');write('root-source-final.json',after_native)
assert subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)==''
summary.update(root_head=after_native['head'],phase='F2 partial: inventory equivalence green; activity narrowed list guard-stopped; global quick admission gap identified',registry_tuples_count=len(after_native['registry_tuples']),compile_input_files=len(before),protected_snapshots=202,source_locks_unchanged=True,root_clean=True,no_model_or_java_test_bodies=True,doctest_scope='Cargo metadata eligible targets recorded separately; rustdoc bodies not executed',activity_narrow_list=load('activity-quick-nextest-all.json'),resource_final=dict(free_bytes=shutil.disk_usage(root).free),outstanding=['Activity narrowed nextest list incomplete at growth cap;5-package public selection proven; full235Temporal inventory complete.','Current globalquick zero-harness facade+IRderive lack explicit compilerOnly admission; caller lane assigned fix.','F3 real6+5quick semantic workload and real standalone not admitted/run.','Darwin cleanup lifecycle remains separately open.'])
write('summary.json',summary)
print(json.dumps({k:v for k,v in summary.items() if k!='activity_narrow_list'},indent=2))
