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
ARCH = ROOT / 'tools/gomad3/internal/gomadtool/architecture'
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERROR = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
CONFIG = ROOT / '.github/.golangci.yml'

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def snapshot():
    return {str(p.relative_to(ROOT)): digest(p) for p in sorted(ARCH.glob('*.go'))}

def protected():
    admission = json.loads((OUT / 'source-admission.json').read_text())
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {p: digest(ROOT / p) for p in paths if p not in admission['protected_exclusions']}
    result = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest())
    assert result['files'] == admission['protected_files']
    assert result['aggregate_sha256'] == admission['protected_aggregate_sha256']
    return result

name, kind = sys.argv[1:3]
assert name.startswith('review-')
assert not (OUT / (name + '.json')).exists()
environment = dict(os.environ)
environment.update(GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='')
environment['PATH'] = str(Path(GO).parent) + ':' + environment['PATH']
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    environment.pop(key, None)
cwd = ROOT / 'tools/gomad3'
if kind == 'test':
    command = [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', *sys.argv[3:]]
elif kind == 'lint':
    command = [LINT, 'run', '--config', str(CONFIG), '--build-tags', 'test_dep', '--fix=false', './internal/gomadtool/architecture']
elif kind == 'errortype':
    command = [ERROR, '-tags', 'test_dep', './internal/gomadtool/architecture']
elif kind == 'validate':
    command = ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1']
elif kind == 'static':
    files = ['internal/gomadtool/architecture/' + item for item in ('effects.go', 'standard.go', 'error_provenance_test.go')]
    command = ['bash', '-c', 'git diff --check -- ' + ' '.join(files) + ' && ' + str(Path(GO).parent / 'gofmt') + ' -l ' + ' '.join(files)]
elif kind == 'audit':
    cwd = ROOT
    command = ['python3', str(OUT / 'review-allocation-worker-audit.py')]
elif kind == 'alias-probe':
    command = ['python3', str(OUT / 'review-allocation-probe.py'), sys.argv[3]]
else:
    raise ValueError(kind)
before, protection_before = snapshot(), protected()
start = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
with (OUT / (name + '.log')).open('x') as log:
    child = subprocess.run(command, cwd=cwd, env=environment, stdout=log, stderr=subprocess.STDOUT)
receipt = dict(command=command, cwd=str(cwd), environment={k: environment.get(k) for k in ('PATH', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')}, started=start, ended=datetime.datetime.now(datetime.timezone.utc).isoformat(), elapsed_seconds=time.monotonic()-clock, exit=child.returncode, timed_out=False, timeout_seconds=None, source_before=before, source_after=snapshot(), protected_before=protection_before, protected_after=protected(), tools={p: digest(p) for p in (GO, LINT, ERROR, str(Path(GO).parent / 'gofmt'))}, config_sha256=digest(CONFIG), log=name+'.log', log_sha256=digest(OUT / (name+'.log')))
receipt['stable'] = receipt['source_before'] == receipt['source_after'] and receipt['protected_before'] == receipt['protected_after']
receipt['runner_sha256'] = digest(__file__)
if kind == 'alias-probe':
    receipt['probe_sha256'] = digest(OUT / 'review-allocation-probe.py')
    receipt['baseline_production_sha256'] = {name: digest(OUT / 'sources/baseline' / name) for name in ('effects.go', 'standard.go')}
with (OUT / (name + '.json')).open('x') as stream:
    stream.write(json.dumps(receipt, indent=2)+'\n')
print(json.dumps({k: receipt[k] for k in ('command', 'exit', 'elapsed_seconds', 'stable', 'log')}))
sys.exit(child.returncode)
