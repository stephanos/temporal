import datetime
import hashlib
import json
import os
import pathlib
import shlex
import subprocess
import sys
import time

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
TOOLS = [GO, str(pathlib.Path(GO).with_name('gofmt')),
         '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',
         '/tmp/fn109-lint-tools.ZdNe1t50/errortype', str(pathlib.Path(__file__).resolve())]
ENV = {
    'GOCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',
    'TMPDIR': '/tmp/fn11210-diagnostic.dzr2T3sO',
    'GOTMPDIR': '/tmp/fn11210-diagnostic.dzr2T3sO',
    'GOMODCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
    'GOPROXY': 'file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download',
    'GOSUMDB': 'off', 'GOENV': 'off', 'GOWORK': 'off', 'GOTOOLCHAIN': 'local', 'GOFLAGS': '',
    'PATH': str(pathlib.Path(GO).parent) + ':' + os.environ['PATH'],
}

def digest(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()

def bindings():
    paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
                                     '.github/workflows/gomad3.yml', '.github/.golangci.yml',
                                     'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode'], cwd=ROOT).split(b'\0')
    result = {os.fsdecode(raw): digest(ROOT / os.fsdecode(raw)) for raw in paths if raw}
    return hashlib.sha256(json.dumps(result, sort_keys=True).encode()).hexdigest()

name, directory, command = sys.argv[1:4]
receipt = OUT / (name + '.json')
stdout = OUT / (name + '.stdout')
stderr = OUT / (name + '.stderr')
if any(path.exists() for path in [receipt, stdout, stderr]):
    raise SystemExit('Refusing to overwrite retained gate evidence')
before = bindings()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
environment = {**os.environ, **ENV}
for key in ['GOMADSEED', 'GOMAD3_CHILD_SEED']:
    environment.pop(key, None)
asserted = 'cd -- ' + shlex.quote(str(ROOT / directory)) + ' && test "$(pwd -P)" = ' + shlex.quote(str(ROOT / directory)) + ' && ' + command
with stdout.open('xb') as output, stderr.open('xb') as errors:
    child = subprocess.Popen(['bash', '-c', asserted], cwd=ROOT / directory, env=environment,
                             stdout=output, stderr=errors, start_new_session=True)
    try:
        status = child.wait(timeout=600)
    except subprocess.TimeoutExpired:
        import signal
        os.killpg(child.pid, signal.SIGKILL)
        child.wait()
        status = 'inconclusive: timeout 600s; process group terminated and reaped'
after = bindings()
events = []
for line in stdout.read_text(errors='replace').splitlines():
    try:
        event = json.loads(line)
    except json.JSONDecodeError:
        continue
    if isinstance(event, dict) and event.get('Action') in ['pass', 'fail', 'skip']:
        events.append({key: event[key] for key in ['Action', 'Package', 'Test'] if key in event})
record = {'command': command, 'shell_command': asserted, 'cwd': str(ROOT / directory), 'started_at': started,
          'ended_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'elapsed_seconds': round(time.monotonic() - clock, 3), 'exit_code': status,
          'source_before_sha256': before, 'source_after_sha256': after, 'source_unchanged': before == after,
          'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
          'tools': {path: digest(path) for path in TOOLS}, 'environment': ENV,
          'filesystem': subprocess.check_output(['stat', '-f', '-c', '%T %i', ENV['TMPDIR']], text=True).strip(),
          'stdout_sha256': digest(stdout), 'stderr_sha256': digest(stderr), 'test_events': events,
          'terminal': child.poll() is not None, 'pid': child.pid}
receipt.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({key: record[key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_unchanged', 'terminal']}))
sys.exit(status if isinstance(status, int) else 3)
