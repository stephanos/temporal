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
ARCH = ROOT / 'tools/gomad3/internal/gomadtool/architecture'
ENV = dict(os.environ)
ENV.update(GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='')
ENV['PATH'] = str(Path(GO).parent) + ':' + ENV['PATH']
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    ENV.pop(key, None)

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def snapshot():
    return {str(path.relative_to(ROOT)): digest(path) for path in sorted(ARCH.glob('*.go'))}

def protect():
    admission = json.loads((OUT / 'source-admission.json').read_text())
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim',
        'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {path: digest(ROOT / path) for path in paths if path not in admission['protected_exclusions']}
    result = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest())
    assert result['files'] == admission['protected_files']
    assert result['aggregate_sha256'] == admission['protected_aggregate_sha256']
    return result

name, kind = sys.argv[1:3]
assert name.startswith('allocation-repair-')
assert not (OUT / (name + '.json')).exists(), 'immutable receipt already exists'
cwd = ROOT / 'tools/gomad3'
if kind == 'snapshot':
    stage = OUT / 'sources' / name
    stage.mkdir(parents=True, exist_ok=False)
    for item in ('effects.go', 'standard.go', 'error_provenance_test.go'):
        if (ARCH / item).exists():
            (stage / item).write_bytes((ARCH / item).read_bytes())
    admission = json.loads((OUT / 'source-admission.json').read_text())
    if name == 'baseline':
        for item, expected in admission['source_plan_bounds'].items():
            assert digest(ROOT / item) == expected, item
        assert not (ARCH / 'error_provenance_test.go').exists()
    result = dict(source=snapshot(), protected=protect(), base_commit=admission['base_commit'])
    (OUT / (name + '.json')).write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result))
    sys.exit(0)
if kind in ('test', 'baseline-test'):
    command = [GO, 'test', '-count=1', '-tags', 'test_dep', *sys.argv[3:]]
elif kind == 'lint':
    command = [LINT, 'run', '--config', str(CONFIG), '--build-tags', 'test_dep', '--fix=false', './internal/gomadtool/architecture']
elif kind == 'errortype':
    command = [ERROR, '-tags', 'test_dep', './internal/gomadtool/architecture']
elif kind == 'validate':
    command = ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1']
elif kind == 'environment':
    command = [GO, 'env', '-json', 'GOOS', 'GOARCH', 'GOROOT', 'GOVERSION', 'GOEXPERIMENT', 'GOCACHE', 'GOMODCACHE', 'CGO_ENABLED']
elif kind == 'static':
    files = ['internal/gomadtool/architecture/' + item for item in ('effects.go', 'standard.go', 'error_provenance_test.go')]
    command = ['bash', '-c', 'git diff --check -- ' + ' '.join(files) + ' && ' + str(Path(GO).parent / 'gofmt') + ' -l ' + ' '.join(files)]
else:
    raise ValueError(kind)
overlay = OUT / 'allocation-repair-baseline-overlay.json'
if kind == 'baseline-test':
    command[1:1] = []
    command[6:6] = ['-overlay', str(overlay)]
before = snapshot()
protected_before = protect()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
with (OUT / (name + '.log')).open('x') as log:
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
    protected_before=protected_before, protected_after=protect(),
    tools={path: digest(path) for path in (GO, LINT, ERROR, str(Path(GO).parent / 'gofmt'))}, config_sha256=digest(CONFIG),
    log=name + '.log', log_sha256=digest(OUT / (name + '.log')))
if kind == 'baseline-test':
    receipt['overlay'] = json.loads(overlay.read_text())
    receipt['overlay_sha256'] = digest(overlay)
    receipt['effective_source'] = dict(before)
    for source, bound in receipt['overlay']['Replace'].items():
        receipt['effective_source'][str(Path(source).relative_to(ROOT))] = digest(bound)
(OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({key: receipt[key] for key in ('command', 'exit', 'timed_out', 'elapsed_seconds', 'stable', 'log')}))
sys.exit(124 if timed_out else child_exit)
