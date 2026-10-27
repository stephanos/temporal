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
tmp = Path('/tmp/umpire-fn25417-frozen-native'); tmp.mkdir(exist_ok=True)
env = os.environ.copy(); env.pop('BASH_ENV', None)
for key in ['TRYBUILD','INSTA_UPDATE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS','UMPIRE_JVM_LEASE_FD','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_MANIFEST','UMPIRE_TOOL_ROOT','CARGO_BUILD_TARGET','CARGO_ENCODED_RUSTFLAGS','RUSTC_BOOTSTRAP','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','RUSTFLAGS']:
    env.pop(key, None)
env.update(CARGO_TARGET_DIR=str(target), TMPDIR=str(tmp), CARGO_HOME=str(Path('/home/agent/.cargo').resolve()), CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0', CARGO_PROFILE_DEV_DEBUG='0', CARGO_PROFILE_TEST_DEBUG='0', GOFLAGS='-p=1', GOMAXPROCS='1', UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip', UMPIRE_QUICK_CACHE='0', UMPIRE_CACHE_DISABLE='1', MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false', MISE_TRUSTED_CONFIG_PATHS=str(root)+':'+str(target))
size = lambda path: int(subprocess.check_output(['du','-sb',str(path)], text=True).split()[0])
initial = snapshot(); assert initial['head'].startswith('be7aeb36e2')
(out/'root-source-before.json').write_text(json.dumps(initial,indent=2)+'\n')
baseline = {'target_bytes':size(target),'free_bytes':shutil.disk_usage(root).free}
(out/'resource-baseline.json').write_text(json.dumps(baseline,indent=2)+'\n')


def run(name, command, cwd, environment=None, timeout=180):
    assert snapshot() == initial, 'immutable root input drift before command'
    assert shutil.disk_usage(root).free >= 8*1024**3
    actual_env = environment or env
    started = time.monotonic()
    receipt = {'name':name,'command':command,'cwd':str(cwd),'source_before_sha256':SHA(out/'root-source-before.json'),'source_files':initial['binary_source_files'],'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'profile':'dev,debug0,incremental0; test policy required/global4 when invoked','environment':{key:actual_env.get(key) for key in ['CARGO_TARGET_DIR','CARGO_HOME','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','GOFLAGS','GOMAXPROCS','UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS']},'BASH_ENV':'removed','phase_budget_bytes':1536*1024**2,'free_guard_bytes':8*1024**3}
    stdout, stderr = out/(name+'.stdout.log'), out/(name+'.stderr.log')
    with stdout.open('wb') as output, stderr.open('wb') as error:
        child = subprocess.Popen(command,cwd=cwd,env=actual_env,stdout=output,stderr=error,start_new_session=True)
        while True:
            try:
                code = child.wait(timeout=1); break
            except subprocess.TimeoutExpired:
                growth=max(0,size(target)-baseline['target_bytes']); free=shutil.disk_usage(root).free
                if growth>1536*1024**2 or free<8*1024**3 or time.monotonic()-started>timeout or stdout.stat().st_size+stderr.stat().st_size>32*1024**2:
                    os.killpg(child.pid,signal.SIGTERM)
                    try:child.wait(timeout=10)
                    except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL);child.wait()
                    receipt.update(stopped_for_guard=True,growth_bytes=growth,free_bytes=free)
                    (out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
                    raise RuntimeError(receipt)
    receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,target_growth_bytes=size(target)-baseline['target_bytes'],free_after_bytes=shutil.disk_usage(root).free,stdout_sha256=SHA(stdout),stderr_sha256=SHA(stderr))
    assert snapshot() == initial, 'immutable root input drift after command'
    receipt['source_unchanged_after']=True
    (out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
    print(name, code, round(receipt['elapsed_seconds'],3), receipt['target_growth_bytes'], flush=True)
    assert code == 0, name
    return receipt


for name, command in [
    ('mise-version',['mise','--version']),
    ('rustc-version',['mise','exec','--','rustc','-Vv']),
    ('cargo-version',['mise','exec','--','cargo','-Vv']),
    ('nextest-version',['mise','exec','--','cargo','nextest','--version'])]:
    run(name,command,root/'model/umpire')

run('framework-production-bootstrap',['/bin/bash',str(root/'model/umpire/scripts/xtask.sh')],root/'model/umpire')
framework = Path((out/'framework-production-bootstrap.stdout.log').read_text().strip())
assert framework.is_absolute() and framework.is_file() and framework.parent.joinpath('umpire-guest-driver').is_file()
run('temporal-native-build',['mise','exec','--','cargo','build','--locked','-j1','-p','temporal-xtask','--bin','temporal-xtask','--profile','dev'],root/'model')
binaries={str(framework):SHA(framework),str(framework.parent/'umpire-guest-driver'):SHA(framework.parent/'umpire-guest-driver'),str(target/'debug/temporal-xtask'):SHA(target/'debug/temporal-xtask')}
receipt={'exit_code':0,'command':['/bin/bash','model/umpire/scripts/xtask.sh','then','mise exec -- cargo build --locked -j1 -p temporal-xtask --bin temporal-xtask --profile dev'],'actual_command_receipts':['framework-production-bootstrap.json','temporal-native-build.json'],'source_files':initial['binary_source_files'],'broader_execution_source_files':initial['source_files'],'source_before_after_equal':True,'root_head':initial['head'],'binaries':binaries,'registry_tuples':initial['registry_tuples'],'tool_version_receipts':['mise-version.json','rustc-version.json','cargo-version.json','nextest-version.json']}
(out/'successful-build.json').write_text(json.dumps(receipt,indent=2)+'\n')
(out/'root-source-after-build.json').write_text(json.dumps(snapshot(),indent=2)+'\n')
public_env=env.copy();public_env.update(NEXTEST_PROFILE='required',NEXTEST_TEST_THREADS='4')
run('public-required-backend-process',[str(framework),'test-run','--package','umpire-backend-process'],root/'model/umpire',public_env,timeout=120)
(out/'root-source-after-public.json').write_text(json.dumps(snapshot(),indent=2)+'\n')
print('ROOT-built public backend process controls completed; prepared driver phase remains separate',flush=True)
