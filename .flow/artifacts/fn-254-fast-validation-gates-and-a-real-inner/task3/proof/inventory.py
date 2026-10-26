import collections, hashlib, json, os, pathlib, subprocess, time
W=pathlib.Path(__file__).resolve().parents[5]
P=pathlib.Path(__file__).resolve().parent
ENV=dict(os.environ)
for k in list(ENV):
    if k.startswith(('CARGO_PROFILE_', 'RUSTFLAGS', 'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER', 'CARGO_ENCODED_RUSTFLAGS')) or k=='BASH_ENV': ENV.pop(k)
ENV.update(CARGO_BUILD_JOBS='2', RUST_TEST_THREADS='2', CARGO_INCREMENTAL='0', UMPIRE_BACKENDS='require', UMPIRE_APALACHE='off')
records=[]
def run(label, cwd, args, env=None):
    start=time.monotonic()
    with (P/(label+'.out')).open('wb') as out, (P/(label+'.err')).open('wb') as err:
        r=subprocess.run(args,cwd=cwd,env=env or ENV,stdout=out,stderr=err)
    records.append(dict(label=label,cwd=str(cwd),argv=args,exit=r.returncode,seconds=time.monotonic()-start))
    (P/'commands.json').write_text(json.dumps(records,indent=2)+'\n')
    if "libtest" not in label: print(label,r.returncode,round(records[-1]['seconds'],2),flush=True)
    if r.returncode: raise SystemExit(r.returncode)
    return (P/(label+'.out')).read_text()
all_cargo=[];all_nextest=[];docs={}
for name, relative in [('framework','model/umpire'),('temporal','model')]:
    cwd=W/relative
    env=ENV|{'CARGO_TARGET_DIR':'/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2543-nextest/'+name}
    sysroot=run(name+'-sysroot',cwd,['mise','exec','--','rustc','--print','sysroot'],env).strip()
    env['LD_LIBRARY_PATH']=env['CARGO_TARGET_DIR']+'/debug/deps:'+sysroot+'/lib/rustlib/aarch64-unknown-linux-gnu/lib:'+sysroot+'/lib'
    run(name+'-version',cwd,['mise','exec','--','cargo','nextest','--version'],env)
    run(name+'-fmt',cwd,['mise','exec','--','cargo','fmt','--all','--check'],env)
    selection=['-p','xtask','--lib'] if name=='framework' else ['-p','temporal-xtask','--bin','temporal-xtask']
    run(name+'-runner-controls',cwd,['mise','exec','--','cargo','test',*selection,'--locked','test_run::'],env)
    output=run(name+'-cargo-build-list',cwd,['mise','exec','--','cargo','test','--workspace','--locked','--no-run','--message-format=json'],env)
    artifacts=[json.loads(x) for x in output.splitlines() if x.startswith('{')]
    cargo=[]
    packages=json.loads(run(name+'-metadata',cwd,['mise','exec','--','cargo','metadata','--locked','--format-version=1','--no-deps'],env))['packages']
    names={p['id']:p['name'] for p in packages}
    for a in artifacts:
        if a.get('reason')!='compiler-artifact' or not a.get('profile',{}).get('test') or not a.get('executable'): continue
        binary=a['executable'];target=a['target'];label=name+'-libtest-'+hashlib.sha256(binary.encode()).hexdigest()[:12]
        outputs=[run(label+suffix,cwd,[binary,*args],env) for suffix,args in [('', ['--list','--format','terse']),('-ignored',['--ignored','--list','--format','terse'])]]
        parse=lambda s:[l.removesuffix(': test') for l in s.splitlines() if l.endswith(': test')]
        tests=parse(outputs[0]);ignored=set(parse(outputs[1]))
        assert len(tests)==len(set(tests));assert ignored<=set(tests)
        for test in tests: cargo.append(dict(package=names[a['package_id']],target_kind=target['kind'][0],target=target['name'],test=test,ignored=test in ignored))
    nxt=json.loads(run(name+'-nextest-list',cwd,['mise','exec','--','cargo','nextest','list','--workspace','--locked','--message-format','json','--run-ignored','all','--ignore-default-filter'],env))
    run(name+'-groups',cwd,['mise','exec','--','cargo','nextest','show-config','test-groups','--workspace','--locked','--show-default','--no-pager'],env)
    nextest=[]
    for suite in nxt['rust-suites'].values():
        binary=suite['binary-id'];package=suite['package-name'];kind=suite['kind'];target=suite['binary-name']
        for test, detail in suite.get('testcases',{}).items():nextest.append(dict(package=package,target_kind=kind,target=target,test=test,ignored=detail['ignored']))
    key=lambda row:tuple(row[k] for k in ['package','target_kind','target','test','ignored'])
    assert collections.Counter(map(key,cargo))==collections.Counter(map(key,nextest)), name+' inventory mismatch'
    assert all(v==1 for v in collections.Counter(map(key,cargo)).values())
    docs[name]=run(name+'-doctest-list',cwd,['mise','exec','--','cargo','test','--doc','--workspace','--locked','--','--list'],env)
    run(name+'-doctests',cwd,['mise','exec','--','cargo','test','--doc','--workspace','--locked','--quiet'],env)
    run(name+'-runner-clippy',cwd,['mise','exec','--','cargo','clippy',*selection[:2],'--all-targets','--locked','--','-D','warnings'],env)
    (P/(name+'-cargo-identities.json')).write_text(json.dumps(sorted(cargo,key=key),indent=2)+'\n')
    (P/(name+'-nextest-identities.json')).write_text(json.dumps(sorted(nextest,key=key),indent=2)+'\n')
    all_cargo+=cargo;all_nextest+=nextest
(P/'inventory-result.json').write_text(json.dumps({'cargo':all_cargo,'nextest':all_nextest,'count':len(all_cargo),'ignored':sum(x['ignored'] for x in all_cargo),'parity':True,'env':{k:ENV[k] for k in ['CARGO_BUILD_JOBS','RUST_TEST_THREADS','CARGO_INCREMENTAL','UMPIRE_BACKENDS','UMPIRE_APALACHE']},'docs':docs},indent=2)+'\n')
