import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERROR = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
CONFIG = ROOT / '.github/.golangci.yml'
ENV = dict(os.environ)
ENV.update(GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='')
ENV['PATH'] = str(Path(GO).parent) + ':' + ENV['PATH']
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    ENV.pop(key, None)

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def snapshot():
    return {str(path.relative_to(ROOT)): digest(path)
            for path in sorted((ROOT / 'tools/gomad3/artifact').glob('*.go'))}

name, kind = sys.argv[1:3]
cwd = ROOT / 'tools/gomad3'
if kind == 'protect':
    admission = json.loads((OUT / 'source-admission.json').read_text())
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim',
        'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {path: digest(ROOT / path) for path in paths if path not in admission['protected_exclusions']}
    receipt = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest(),
        captured=datetime.datetime.now(datetime.timezone.utc).isoformat(), selection=admission['selection'],
        pinned_close_join_sources={str(Path(GO).parents[1] / 'src' / path): digest(Path(GO).parents[1] / 'src' / path)
            for path in ('os/file_posix.go', 'os/file_unix.go', 'os/root_openat.go', 'errors/join.go')},
        platform=subprocess.check_output([GO, 'env', 'GOOS', 'GOARCH'], env=ENV, text=True).splitlines(),
        patched_go_exists=(cwd / '.toolchain/bin/go').exists())
    assert receipt['files'] == admission['protected_files']
    assert receipt['aggregate_sha256'] == admission['protected_aggregate_sha256']
    (OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps(receipt))
    sys.exit(0)
if kind == 'test':
    command = [GO, 'test', '-count=1', '-tags', 'test_dep', *sys.argv[3:]]
elif kind == 'lint':
    command = [LINT, 'run', '--config', str(CONFIG), '--build-tags', 'test_dep', '--fix=false', './artifact']
elif kind == 'errortype':
    command = [ERROR, '-tags', 'test_dep', './artifact']
elif kind == 'validate':
    command = ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1']
elif kind == 'static':
    paths = ['artifact/' + item for item in ('store.go', 'target_pool.go', 'store_test.go', 'target_pool_test.go', 'publication_test.go')]
    command = ['bash', '-c', 'git diff --check -- ' + ' '.join(paths) + ' && ' + str(Path(GO).parent / 'gofmt') + ' -l ' + ' '.join(paths)]
else:
    raise ValueError(kind)
before = snapshot()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
with (OUT / (name + '.log')).open('w') as log:
    try:
        result = subprocess.run(command, cwd=cwd, env=ENV, stdout=log, stderr=subprocess.STDOUT, timeout=600)
        child_exit, timed_out = result.returncode, False
    except subprocess.TimeoutExpired:
        child_exit, timed_out = None, True
after = snapshot()
receipt = dict(command=command, cwd=str(cwd), environment={key: ENV.get(key) for key in
    ('PATH', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')},
    started=started, ended=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    elapsed_seconds=time.monotonic() - clock, exit=child_exit, timed_out=timed_out, timeout_seconds=600,
    source_before=before, source_after=after, stable=before == after,
    tools={path: digest(path) for path in (GO, LINT, ERROR)}, config_sha256=digest(CONFIG),
    log=name + '.log', log_sha256=digest(OUT / (name + '.log')))
(OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({key: receipt[key] for key in ('command', 'exit', 'timed_out', 'elapsed_seconds', 'stable', 'log')}))
