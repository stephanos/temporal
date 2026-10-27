from pathlib import Path
import os, subprocess, json, time, hashlib, shutil, signal, datetime, sys

root = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
out = Path(__file__).resolve().parent
admission = root / '.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/admission-proof'
namespace = {'__name__': 'proof_import', '__file__': str(admission / 'driver.py')}
exec(compile((admission / 'driver.py').read_text(), str(admission / 'driver.py'), 'exec'), namespace)
SHA = namespace['SHA']
snapshot = lambda: namespace['snapshot'](root)
target = root / '.flow/tmp/fn25417-standalone-renewal/target'
tmp = Path('/tmp/fn25417-standalone-tmp-dc7feca3'); tmp.mkdir(exist_ok=True)
env = os.environ.copy(); env.pop('BASH_ENV', None)
for key in ['TRYBUILD','INSTA_UPDATE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS','UMPIRE_JVM_LEASE_FD','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_MANIFEST','UMPIRE_TOOL_ROOT','CARGO_BUILD_TARGET','CARGO_ENCODED_RUSTFLAGS','RUSTC_BOOTSTRAP','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','RUSTFLAGS']:
    env.pop(key, None)
env.update(CARGO_TARGET_DIR=str(target), TMPDIR=str(tmp), CARGO_HOME=str(Path('/home/agent/.cargo').resolve()), CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0', GOFLAGS='-p=1', GOMAXPROCS='1', UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip', UMPIRE_QUICK_CACHE='0', UMPIRE_CACHE_DISABLE='1', MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false', MISE_TRUSTED_CONFIG_PATHS=str(root)+':'+str(target))
go_cache = root / '.flow/tmp/fn25417-standalone-renewal/go-cache'
env.update(GOCACHE=str(go_cache))
size = lambda path: 0 if not path.exists() else int(subprocess.run(['du','-sb',str(path)], text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0])
production = Path('/tmp/umpire-fn25417-standalone-dc7feca3')
for key in list(env):
    if key.startswith('CARGO_PROFILE_'):env.pop(key)
env.update(UMPIRE_STANDALONE_OUTPUT=str(production),UMPIRE_CACHE_DISABLE='1',UMPIRE_QUICK_CACHE='0',GOMAXPROCS='1',GOFLAGS='-p=1')
initial = snapshot(); assert initial['head'] == 'dc7feca3c497f039c3d2e6f755bddeddb59a7c1d'
(out/'root-source-before.json').write_text(json.dumps(initial,indent=2)+'\n')
baseline = json.loads((out/'resource-baseline.json').read_text()) if (out/'resource-baseline.json').exists() else {'target_bytes':size(target)+size(go_cache)+size(tmp)+size(production),'free_bytes':shutil.disk_usage(root).free}
(out/'resource-baseline.json').write_text(json.dumps(baseline,indent=2)+'\n')


def run(name, command, cwd, environment=None, timeout=180):
    assert snapshot() == initial, 'immutable root input drift before command'
    assert shutil.disk_usage(root).free >= 30*1024**3
    actual_env = environment or env
    started = time.monotonic()
    receipt = {'name':name,'command':command,'cwd':str(cwd),'source_before_sha256':SHA(out/'root-source-before.json'),'source_files':initial['binary_source_files'],'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'profile': 'native refresh dev/debug0; inventory shipping profile with no CARGO_PROFILE overrides', 'cargo_profile_overrides':{k:v for k,v in actual_env.items() if k.startswith('CARGO_PROFILE_')},'environment':{key:actual_env.get(key) for key in ['CARGO_TARGET_DIR','CARGO_HOME','TMPDIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','GOFLAGS','GOMAXPROCS','UMPIRE_BACKENDS','UMPIRE_APALACHE','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS']},'BASH_ENV':'removed','phase_budget_bytes':30*1024**3,'free_guard_bytes':30*1024**3}
    stdout, stderr = out/(name+'.stdout.log'), out/(name+'.stderr.log')
    with stdout.open('wb') as output, stderr.open('wb') as error:
        child = subprocess.Popen(command,cwd=cwd,env=actual_env,stdout=output,stderr=error,start_new_session=True)
        if name=='production-standalone':
            observer.pid=child.pid
            print('production-standalone PID',child.pid,flush=True)
            (out/'active-pid.json').write_text(json.dumps(dict(pid=child.pid,command=command,cwd=str(cwd)))+'\n')
        while True:
            try:
                code = child.wait(timeout=1); break
            except subprocess.TimeoutExpired:
                growth=max(0,size(target)+size(go_cache)+size(tmp)+size(production)-baseline['target_bytes']); free=shutil.disk_usage(root).free
                if growth>30*1024**3 or free<30*1024**3 or time.monotonic()-started>timeout or stdout.stat().st_size+stderr.stat().st_size>32*1024**2:
                    os.killpg(child.pid,signal.SIGTERM)
                    try:child.wait(timeout=10)
                    except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL);child.wait()
                    receipt.update(stopped_for_guard=True,growth_bytes=growth,free_bytes=free)
                    (out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
                    raise RuntimeError(receipt)
    receipt.update(exit_code=code,elapsed_seconds=time.monotonic()-started,target_growth_bytes=size(target)+size(go_cache)+size(tmp)+size(production)-baseline['target_bytes'],free_after_bytes=shutil.disk_usage(root).free,stdout_sha256=SHA(stdout),stderr_sha256=SHA(stderr))
    assert snapshot() == initial, 'immutable root input drift after command'
    receipt['source_unchanged_after']=True
    (out/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
    print(name, code, round(receipt['elapsed_seconds'],3), receipt['target_growth_bytes'], flush=True)
    assert code == 0, name
    return receipt




from observer import Observer
assert __import__('ctypes').CDLL(None).prctl(36,1,0,0,0)==0
observer=Observer(out)
build_env = dict(env, CARGO_PROFILE_DEV_DEBUG='0')
run('native-bootstrap', ['mise', 'exec', '--', 'cargo', 'build', '--locked', '-p', 'xtask', '--bin', 'xtask', '-p', 'umpire-guest-driver', '--bin', 'umpire-guest-driver', '--profile', 'dev', '--target-dir', str(target/'xtask-bootstrap')], root/'model/umpire', environment=build_env, timeout=900)
(out/'native-build-projection.json').write_text(json.dumps(dict(root_head=initial['head'], source_files=initial['binary_source_files'], binaries={str(p):SHA(p) for p in [target/'xtask-bootstrap/debug/xtask', target/'xtask-bootstrap/debug/umpire-guest-driver']}, reuse='none; fresh native bootstrap with dev debug=0'),indent=2)+'\n')
run('focused-quick-test', ['mise','exec','--','cargo','nextest','run','--locked','-p','xtask','--profile','required','-E','test(quick::tests::workspace_documentation_can_supply_executable_test_inputs) | test(test_run::tests::selected_packages_share_one_nextest_batch_and_only_eligible_doctests) | test(test_run::tests::inventory_admission_refuses_each_empty_package_and_exact_fixture_selection)'], root/'model/umpire', timeout=900)
run('retired-names', [str(target/'xtask-bootstrap/debug/xtask'),'gate','--stage','retired-names'], root/'model/umpire', timeout=180)
run('source-boundary', [str(target/'xtask-bootstrap/debug/xtask'),'gate','--stage','source-boundary'], root/'model/umpire', timeout=180)
try:
 run('production-standalone',[str(target/'xtask-bootstrap/debug/xtask'),'gate','--stage','standalone'],root/'model/umpire',timeout=3600)
finally:
 observer.finish()
 (out/'root-source-final.json').write_text(json.dumps(snapshot(),indent=2)+'\n')
