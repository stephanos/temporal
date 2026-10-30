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
tmp = Path('/tmp/umpire-fn25417-kettle-semantic-renewal'); tmp.mkdir(exist_ok=True)
env = os.environ.copy(); env.pop('BASH_ENV', None)
for key in ['TRYBUILD','INSTA_UPDATE','NEXTEST_PROFILE','NEXTEST_TEST_THREADS','UMPIRE_JVM_LEASE_FD','UMPIRE_STANDALONE_CHILD','UMPIRE_STANDALONE_MANIFEST','UMPIRE_TOOL_ROOT','CARGO_BUILD_TARGET','CARGO_ENCODED_RUSTFLAGS','RUSTC_BOOTSTRAP','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','RUSTFLAGS']:
    env.pop(key, None)
env.update(CARGO_TARGET_DIR=str(target), TMPDIR=str(tmp), CARGO_HOME=str(Path('/home/agent/.cargo').resolve()), CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0', CARGO_PROFILE_DEV_DEBUG='0', CARGO_PROFILE_TEST_DEBUG='0', GOFLAGS='-p=1', GOMAXPROCS='1', UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip', UMPIRE_QUICK_CACHE='0', UMPIRE_CACHE_DISABLE='1', MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false', MISE_TRUSTED_CONFIG_PATHS=str(root)+':'+str(target))
go_cache = root / '.flow/tmp/fn2545-temporal-api/go-cache'
env.update(GOCACHE=str(go_cache))
size = lambda path: int(subprocess.run(['du','-sb',str(path)], text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,check=False).stdout.split()[0])
initial = snapshot(); assert initial['head'].startswith('9a4b34f0d4')
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



run('native-resource-admission',['python3',str(admission/'driver.py'),'--source-root',str(root),'--execute','--seal',str(out/'resource-seal.json'),'--binary',str(target/'xtask-bootstrap/debug/xtask'),'--build-receipt',str(out/'successful-build.json'),'--output',str(out/'resources')],root,timeout=120)
