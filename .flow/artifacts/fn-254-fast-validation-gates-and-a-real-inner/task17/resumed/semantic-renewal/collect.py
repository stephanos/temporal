from pathlib import Path
import os, subprocess, json, time, hashlib, shutil, signal, datetime, sys

own = Path(__file__).resolve().parents[6]
root = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
out = Path(__file__).resolve().parent
admission = out.parent / 'admission-proof'
namespace = {'__name__': 'proof_import', '__file__': str(admission / 'driver.py')}
exec(compile((admission / 'driver.py').read_text(), str(admission / 'driver.py'), 'exec'), namespace)
SHA = namespace['SHA']
snapshot = lambda: namespace['snapshot'](root)
target = root / '.flow/tmp/fn2546-resumed/target'
tmp = Path('/tmp/umpire-fn25417-semantic-renewal'); tmp.mkdir(exist_ok=True)
env = os.environ.copy(); env.pop('BASH_ENV', None)
for key in ['TRYBUILD','INSTA_UPDATE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS','UMPIRE_JVM_LEASE_FD','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_MANIFEST','UMPIRE_TOOL_ROOT','CARGO_BUILD_TARGET','CARGO_ENCODED_RUSTFLAGS','RUSTC_BOOTSTRAP','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','RUSTFLAGS']:
    env.pop(key, None)
env.update(CARGO_TARGET_DIR=str(target), TMPDIR=str(tmp), CARGO_HOME=str(Path('/home/agent/.cargo').resolve()), CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0', CARGO_PROFILE_DEV_DEBUG='0', CARGO_PROFILE_TEST_DEBUG='0', GOFLAGS='-p=1', GOMAXPROCS='1', UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip', UMPIRE_QUICK_CACHE='0', UMPIRE_CACHE_DISABLE='1', MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false', MISE_TRUSTED_CONFIG_PATHS=str(root)+':'+str(target))
go_cache = root / '.flow/tmp/fn2545-temporal-api/go-cache'
env.update(GOCACHE=str(go_cache))
size = lambda path: int(subprocess.run(['du','-sb',str(path)], text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0])
initial = snapshot(); assert initial['head'].startswith('ede5fadc61')
(out/'root-source-before.json').write_text(json.dumps(initial,indent=2)+'\n')
baseline = json.loads((out/'resource-baseline.json').read_text()) if (out/'resource-baseline.json').exists() else {'target_bytes':size(target)+size(go_cache),'free_bytes':shutil.disk_usage(root).free}
(out/'resource-baseline.json').write_text(json.dumps(baseline,indent=2)+'\n')


def run(name, command, cwd, environment=None, timeout=180):
    assert snapshot() == initial, 'immutable root input drift before command'
    assert shutil.disk_usage(root).free >= 8*1024**3
    actual_env = environment or env
    started = time.monotonic()
    receipt = {'name':name,'command':command,'cwd':str(cwd),'source_before_sha256':SHA(out/'root-source-before.json'),'source_files':initial['binary_source_files'],'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'profile': 'native refresh dev/debug0; inventory shipping profile with no CARGO_PROFILE overrides', 'cargo_profile_overrides':{k:v for k,v in actual_env.items() if k.startswith('CARGO_PROFILE_')},'environment':{key:actual_env.get(key) for key in ['CARGO_TARGET_DIR','CARGO_HOME','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','GOFLAGS','GOMAXPROCS','UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS']},'BASH_ENV':'removed','phase_budget_bytes':3*1024**3,'free_guard_bytes':8*1024**3}
    stdout, stderr = out/(name+'.stdout.log'), out/(name+'.stderr.log')
    with stdout.open('wb') as output, stderr.open('wb') as error:
        child = subprocess.Popen(command,cwd=cwd,env=actual_env,stdout=output,stderr=error,start_new_session=True)
        while True:
            try:
                code = child.wait(timeout=1); break
            except subprocess.TimeoutExpired:
                growth=max(0,size(target)+size(go_cache)-baseline['target_bytes']); free=shutil.disk_usage(root).free
                if growth>3*1024**3 or free<8*1024**3 or time.monotonic()-started>timeout or stdout.stat().st_size+stderr.stat().st_size>32*1024**2:
                    os.killpg(child.pid,signal.SIGTERM)
                    try:child.wait(timeout=10)
                    except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL);child.wait()
                    receipt.update(stopped_for_guard=True,growth_bytes=growth,free_bytes=free)
                    (out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
                    raise RuntimeError(receipt)
    receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,target_growth_bytes=size(target)+size(go_cache)-baseline['target_bytes'],free_after_bytes=shutil.disk_usage(root).free,stdout_sha256=SHA(stdout),stderr_sha256=SHA(stderr))
    assert snapshot() == initial, 'immutable root input drift after command'
    receipt['source_unchanged_after']=True
    (out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
    print(name, code, round(receipt['elapsed_seconds'],3), receipt['target_growth_bytes'], flush=True)
    assert code == 0, name
    return receipt



for key in list(env):
    if key.startswith('CARGO_PROFILE_'):env.pop(key)

def metadata(scope, cwd):
    run(scope+'-metadata',['mise','exec','--','cargo','metadata','--locked','--format-version','1'],cwd)
    data=json.loads((out/(scope+'-metadata.stdout.log')).read_text())
    return {p['id']:p for p in data['packages'] if p['id'] in data['workspace_members']}

def collect(scope,cwd,names):
    packages={k:p for k,p in metadata(scope,cwd).items() if p['name'] in names}
    assert len(packages)==len(names)
    package_args=[arg for name in names for arg in ['-p',name]]
    run(scope+'-cargo-no-run',['mise','exec','--','cargo','test','--locked','--no-run','--message-format=json']+package_args,cwd,timeout=1200)
    harnesses={}
    for line in (out/(scope+'-cargo-no-run.stdout.log')).read_text().splitlines():
        row=json.loads(line)
        if row.get('reason')!='compiler-artifact' or not row.get('executable') or not row['profile']['test'] or row['package_id'] not in packages:continue
        test_target=row['target']; kind=test_target['kind'][0]
        if not test_target['test']:continue
        key=(packages[row['package_id']]['name'],kind,test_target['name'])
        assert key not in harnesses or harnesses[key]==row['executable']
        harnesses[key]=row['executable']
    run(scope+'-rustc-target-libdir',['mise','exec','--','rustc','--print','target-libdir'],cwd)
    harness_env=env.copy();harness_env['LD_LIBRARY_PATH']=str(target/'debug/deps')+':'+(out/(scope+'-rustc-target-libdir.stdout.log')).read_text().strip()
    rows=[]
    for index,((package,kind,binary),exe) in enumerate(sorted(harnesses.items())):
        stem=f'{scope}-harness-{index:03}'
        run(stem+'-all',[exe,'--list','--format','terse'],cwd,harness_env)
        run(stem+'-ignored',[exe,'--ignored','--list','--format','terse'],cwd,harness_env)
        rows.append(dict(package=package,kind=kind,binary=binary,executable=exe,sha256=SHA(Path(exe)),all_list=str(out/(stem+'-all.stdout.log')),ignored_list=str(out/(stem+'-ignored.stdout.log'))))
    (out/(scope+'-cargo-lists.json')).write_text(json.dumps(rows,indent=2)+'\n')
    (out/(scope+'-doctest-eligibility.json')).write_text(json.dumps([dict(package=p['name'],name=t['name'],kind=t['kind'],doctest=t['doctest']) for p in packages.values() for t in p['targets'] if t['doctest'] and any(k in ['lib','proc-macro'] for k in t['kind'])],indent=2)+'\n')


if __name__=='__main__':
    scope=sys.argv[1]
    names={'activity':['umpire-temporal','umpire-temporal-api','umpire-temporal-guest','umpire-temporal-realization-macros','umpire-temporal-tests'],'framework-affected':['umpire-backend-process','xtask'],'temporal-xtask':['temporal-xtask']}[scope]
    cwd=root/('model/umpire' if scope=='framework-affected' else 'model')
    collect(scope,cwd,names)
    args=['mise','exec','--','cargo','nextest','list','--locked','--profile','required','--message-format','json','--run-ignored','all']+[arg for name in names for arg in ['-p',name]]
    if scope=='activity':args+=['-E','package(=umpire-temporal) | package(=umpire-temporal-api) | package(=umpire-temporal-guest) | package(=umpire-temporal-realization-macros) | (package(=umpire-temporal-tests) & kind(test) & binary(=fixtures) & test(=fixtures))']
    run(scope+'-nextest-all',args,cwd,timeout=1200)
