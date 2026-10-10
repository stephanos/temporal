import datetime
import hashlib
import json
import os
import pathlib
import shlex
import signal
import subprocess
import sys
import time

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-62').resolve()
OUT = pathlib.Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
PRIVATE = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-62-private')
ENV = {
    'GOCACHE': str(PRIVATE / 'cache'),
    'GOMODCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
    'GOPROXY': 'off', 'GOSUMDB': 'off', 'GOENV': 'off', 'GOWORK': 'off',
    'GOTOOLCHAIN': 'local', 'GOFLAGS': '',
    'TMPDIR': str(PRIVATE / 'tmp'), 'GOTMPDIR': str(PRIVATE / 'tmp'),
    'PATH': str(pathlib.Path(GO).parent) + ':' + os.environ['PATH'],
}
TOOLS = [GO, str(pathlib.Path(GO).with_name('gofmt')),
         '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',
         '/tmp/fn109-lint-tools.ZdNe1t50/errortype']


def digest(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def bindings(module):
    paths = subprocess.check_output(
        ['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
         '.github/.golangci.yml', 'Makefile', 'AGENTS.md', 'MILESTONES.md',
         'cmd/tools/lintcode'], cwd=ROOT).split(b'\0')
    paths = {os.fsdecode(path) for path in paths if path}
    if (module / 'runner/progress_start_test.go').exists():
        paths.add('tools/gomad3/runner/progress_start_test.go')
    files = {}
    for path in sorted(paths):
        actual = module / path.removeprefix('tools/gomad3/') if path.startswith('tools/gomad3/') else ROOT / path
        files[path] = digest(actual)
    return files


if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
name, directory, command = sys.argv[1:4]
module = pathlib.Path(sys.argv[4]).resolve() if len(sys.argv) > 4 else ROOT / 'tools/gomad3'
if module != ROOT / 'tools/gomad3' and PRIVATE not in module.parents:
    raise SystemExit('Private source must belong to task 62')
receipt, stdout, stderr = [OUT / (name + suffix) for suffix in ['.json', '.stdout', '.stderr']]
if any(path.exists() for path in [receipt, stdout, stderr]):
    raise SystemExit('Refusing to overwrite retained evidence')
for path in [PRIVATE / 'cache', PRIVATE / 'tmp']:
    path.mkdir(parents=True, exist_ok=True)
environment = {**os.environ, **ENV}
for key in ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'BASH_ENV']:
    environment.pop(key, None)
environment['SANDBOX_START_DIR'] = str(ROOT)
cwd = pathlib.Path(directory).resolve() if pathlib.Path(directory).is_absolute() else ROOT / directory
shell_command = 'cd -- ' + shlex.quote(str(cwd)) + ' && test "$(pwd -P)" = ' + shlex.quote(str(cwd)) + ' && ' + command
before = bindings(module)
started, clock = datetime.datetime.now(datetime.timezone.utc).isoformat(), time.monotonic()
with stdout.open('xb') as output, stderr.open('xb') as errors:
    child = subprocess.Popen(['bash', '-c', shell_command], cwd=cwd, env=environment,
                             stdout=output, stderr=errors, start_new_session=True)
    timed_out = False
    try:
        status = child.wait(timeout=600)
    except subprocess.TimeoutExpired:
        timed_out = True
        os.killpg(child.pid, signal.SIGKILL)
        status = child.wait()
after = bindings(module)
observation = 'process-timeout' if timed_out else 'ordinary-termination'
if b'panic: test timed out after' in stdout.read_bytes() + stderr.read_bytes():
    observation = 'diagnostic-test-watchdog'
record = {
    'command': command, 'shell_command': shell_command, 'cwd': str(cwd),
    'source_module': str(module), 'started_at': started,
    'elapsed_seconds': round(time.monotonic() - clock, 3),
    'exit_code': status, 'timed_out': timed_out, 'observation': observation,
    'source_files': before,
    'source_before_sha256': hashlib.sha256(json.dumps(before, sort_keys=True).encode()).hexdigest(),
    'source_after_sha256': hashlib.sha256(json.dumps(after, sort_keys=True).encode()).hexdigest(),
    'source_unchanged': before == after,
    'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
    'tools': {path: digest(path) for path in TOOLS}, 'gate_runner_sha256': digest(__file__),
    'environment': ENV, 'removed_environment': ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'BASH_ENV'],
    'sandbox_start_dir': str(ROOT), 'terminal': child.poll() is not None,
    'stdout_sha256': digest(stdout), 'stderr_sha256': digest(stderr),
}
receipt.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({key: record[key] for key in ['command', 'exit_code', 'elapsed_seconds',
                                            'source_unchanged', 'terminal', 'observation']}))
sys.exit(124 if timed_out else status)
