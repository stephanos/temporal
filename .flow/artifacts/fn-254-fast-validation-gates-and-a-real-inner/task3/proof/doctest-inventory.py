import json,os,pathlib,subprocess,tomllib
W=pathlib.Path(__file__).resolve().parents[5];P=pathlib.Path(__file__).resolve().parent
rows=[];commands=[]
for name,relative in [('framework','model/umpire'),('temporal','model')]:
 E=dict(os.environ)
 for k in list(E):
  if k=='BASH_ENV' or k.startswith(('CARGO_PROFILE_','RUSTFLAGS','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','CARGO_ENCODED_RUSTFLAGS')):E.pop(k)
 E.update(CARGO_TARGET_DIR='/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2543-nextest/'+name,CARGO_BUILD_JOBS='2',RUST_TEST_THREADS='2',CARGO_INCREMENTAL='0',UMPIRE_BACKENDS='require',UMPIRE_APALACHE='off')
 args=['mise','exec','--','cargo','test','--doc','--workspace','--locked','--','--ignored','--list']
 r=subprocess.run(args,cwd=W/relative,env=E,capture_output=True,text=True)
 (P/(name+'-doctest-ignored-list.out')).write_text(r.stdout);(P/(name+'-doctest-ignored-list.err')).write_text(r.stderr);assert r.returncode==0
 commands.append(dict(argv=args,cwd=str(W/relative),exit=r.returncode))
 parse=lambda s:[l.removesuffix(': test') for l in s.splitlines() if l.endswith(': test')]
 ignored=set(parse(r.stdout));all=parse((P/(name+'-doctest-list.out')).read_text());assert ignored<=set(all)
 packages=json.loads((P/(name+'-metadata.out')).read_text())['packages']
 for fqn in all:
  source=(W/relative/fqn.split(' - ')[0]).resolve()
  p=max((p for p in packages if source.is_relative_to(pathlib.Path(p['manifest_path']).parent)),key=lambda p:len(p['manifest_path']))
  targets=[t for t in p['targets'] if t['doctest'] and t['kind'][0] in ['lib','rlib','cdylib','proc-macro']];assert len(targets)==1
  rows.append(dict(workspace=name,package=p['name'],target=targets[0]['name'],target_kind='doctest',test=fqn,ignored=fqn in ignored,source=str(source.relative_to(W))))
assert len(rows)==5 and sum(r['ignored'] for r in rows)==1
(P/'doctest-identities.json').write_text(json.dumps(dict(identities=rows,commands=commands,libtest_separate=True,nextest_doctest_count=0,cargo_doc_invocations_per_successful_leaf=1,previous_inventory='task4/consolidation-identities.json doctests (4 passed, 1 ignored); every source file unchanged at this candidate'),indent=2)+'\n')
print('doctests:',len(rows),'ignored',sum(r['ignored'] for r in rows))
