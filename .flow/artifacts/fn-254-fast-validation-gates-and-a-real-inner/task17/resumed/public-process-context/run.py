from pathlib import Path
import hashlib
import json
import os
import shutil
import signal
import subprocess
import sys
import time

out = Path(__file__).resolve().parent
root = Path(__file__).resolve().parents[6]
target = root / '.flow/tmp/fn2546-resumed/target'
tmp = Path('/tmp/fn25417-public-process-context')
tmp.mkdir(exist_ok=True)
env = os.environ.copy()
for key in list(env):
    if key == 'BASH_ENV' or key.startswith('CARGO_PROFILE_') or key in (
        'INSTA_UPDATE', 'TRYBUILD', 'UMPIRE_JVM_LEASE_FD',
        'UMPIRE_COMPLETION_FD', 'UMPIRE_CANCELLATION_FD',
    ):
        env.pop(key, None)
env.update(
    CARGO_TARGET_DIR=str(target), CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0',
    GOCACHE=str(root / '.flow/tmp/fn2545-temporal-api/go-cache'),
    GOMAXPROCS='1', GOFLAGS='-p=1', TMPDIR=str(tmp),
    UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip', UMPIRE_CACHE_DISABLE='1',
    UMPIRE_QUICK_CACHE='0', NEXTEST_PROFILE='required', NEXTEST_TEST_THREADS='4',
    MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false',
    MISE_TRUSTED_CONFIG_PATHS=os.pathsep.join(map(str, (root, target, tmp))),
)
label = sys.argv[1]
binary = target / 'xtask-bootstrap/debug/xtask'
if label == 'bootstrap':
    command = ['/bin/bash', str(root / 'model/umpire/scripts/xtask.sh')]
    env['CARGO_PROFILE_DEV_DEBUG'] = '0'
else:
    command = [str(binary), 'test-run', '--package', 'umpire-backend-process']

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def size():
    return int(subprocess.check_output(['du', '-sk', str(target)]).split()[0]) * 1024

source = {
    str(path.relative_to(root)): sha(path)
    for path in (root / 'model/umpire/backends/process').rglob('*.rs')
}
baseline = size()
started = time.monotonic()
stopped = None
with (out / (label + '.stdout.log')).open('wb') as stdout, (out / (label + '.stderr.log')).open('wb') as stderr:
    child = subprocess.Popen(command, cwd=root / 'model/umpire', env=env,
                             stdout=stdout, stderr=stderr, start_new_session=True)
    while child.poll() is None:
        if size() - baseline > 512 * 1024**2 or shutil.disk_usage(root).free < 8 * 1024**3 or time.monotonic() - started > 120:
            stopped = 'resource/time guard'
            os.killpg(child.pid, signal.SIGTERM)
            child.wait(timeout=20)
            break
        time.sleep(0.2)
    code = child.wait()
assert source == {name: sha(root / name) for name in source}, 'source changed during command'
receipt = dict(
    command=command, cwd=str(root / 'model/umpire'), exit_code=code,
    elapsed_seconds=time.monotonic() - started, stopped=stopped,
    head=subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
    source_sha256=source, native_sha256=sha(binary),
    target_growth_bytes=size() - baseline, free_bytes=shutil.disk_usage(root).free,
    environment={key: env.get(key) for key in (
        'CARGO_BUILD_JOBS', 'CARGO_INCREMENTAL', 'CARGO_PROFILE_DEV_DEBUG',
        'NEXTEST_PROFILE', 'NEXTEST_TEST_THREADS', 'UMPIRE_BACKENDS',
        'UMPIRE_APALACHE', 'UMPIRE_CACHE_DISABLE',
    )},
    log_sha256={p.name: sha(p) for p in out.glob(label + '.*.log')},
)
(out / (label + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({key: receipt[key] for key in ('exit_code', 'elapsed_seconds', 'stopped', 'target_growth_bytes')}), flush=True)
