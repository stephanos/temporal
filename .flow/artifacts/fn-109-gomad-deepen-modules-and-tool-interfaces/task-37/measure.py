import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37'
BASE = '22fe28d1d6b641e9deea5bf9b60a1da773c06381'
ADMISSION = json.loads((OUT / 'root-admission.json').read_text())
ENV = os.environ.copy()
ENV.update(PATH='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin', GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='', GOMAXPROCS='2')
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED'):
    ENV.pop(key, None)

def digest(data):
    return hashlib.sha256(data).hexdigest()

def inputs():
    paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'tests/gomadfunctional'], cwd=ROOT).decode().split('\0')
    paths += list(ADMISSION['protected_sha256']) + ADMISSION['product_paths'] + ['.flow/tmp/after-corpus-independent-owner-astra.md', str((OUT / 'root-admission.json').relative_to(ROOT))]
    return {p: digest((ROOT / p).read_bytes()) for p in sorted(set(paths)) if p and (ROOT / p).is_file()}

def tools():
    result = {}
    for name, definition in ADMISSION['tools'].items():
        result[name] = {'path': definition['path'], 'sha256': digest(Path(definition['path']).read_bytes())}
        if result[name]['sha256'] != definition['sha256']:
            raise RuntimeError('tool hash mismatch: ' + name)
    return result

def compact(source):
    return {'input_count': len(source), 'inventory_sha256': digest(json.dumps(source, sort_keys=True).encode()), 'product': {p: source[p] for p in ADMISSION['product_paths']}, 'protected': {p: source[p] for p in ADMISSION['protected_sha256']}, 'admission_sha256': source[str((OUT / 'root-admission.json').relative_to(ROOT))]}

def run(name, command, cwd):
    before = inputs()
    tool_ids = tools()
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock = time.monotonic()
    with (OUT / (name + '.log')).open('xb') as log:
        try:
            result = subprocess.run(command, cwd=cwd, env=ENV, stdout=log, stderr=subprocess.STDOUT, timeout=600, check=False)
            code = result.returncode
            status = 'terminal'
        except subprocess.TimeoutExpired:
            code = None
            status = 'inconclusive: timeout 600 seconds'
    after = inputs()
    receipt = {'name': name, 'command': command, 'cwd': str(cwd), 'base_commit': BASE, 'start': start, 'end': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'elapsed_seconds': time.monotonic() - clock, 'exit_code': code, 'status': status, 'source_before': compact(before), 'source_after': compact(after), 'source_stable': before == after, 'tools': tool_ids, 'environment': {k: ENV.get(k) for k in ('PATH', 'GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMAXPROCS', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOEXPERIMENT', 'GOCACHE', 'GOMODCACHE', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED')}, 'log': name + '.log', 'log_sha256': digest((OUT / (name + '.log')).read_bytes())}
    (OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps({k: receipt[k] for k in ('name', 'exit_code', 'status', 'elapsed_seconds', 'source_stable', 'log')}))

if sys.argv[1] == 'pin':
    current = inputs()
    for p, expected in {**ADMISSION['baseline_source_sha256'], **ADMISSION['protected_sha256']}.items():
        if current[p] != expected:
            raise RuntimeError('admission mismatch: ' + p)
    bound = {}
    for p in current:
        if p.startswith(('tools/gomad3/', 'tools/gomad3sim/', 'tools/gomad3integration/', 'tests/gomadfunctional/')) or p == '.github/.golangci.yml':
            saved = subprocess.check_output(['git', 'show', BASE + ':' + p], cwd=ROOT)
            bound[p] = {'base_sha256': digest(saved), 'working_sha256': current[p]}
            if bound[p]['base_sha256'] != current[p]:
                raise RuntimeError('BASE mismatch: ' + p)
    payload = {'base_commit': BASE, 'source': current, 'base_bindings': bound, 'tools': tools(), 'admission_sha256': digest((OUT / 'root-admission.json').read_bytes())}
    (OUT / 'pin.json').write_text(json.dumps(payload, indent=2) + '\n')
    print('Pinned ' + str(len(bound)) + ' BASE-bound inputs and all admitted tool/source hashes.')
else:
    run(sys.argv[1], sys.argv[3:], ROOT / sys.argv[2])
