import datetime
import hashlib
import json
import os
import pathlib
import shlex
import subprocess
import sys
import time

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-60').resolve()
OUT = pathlib.Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
ENV = {
    'GOCACHE': '/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task60-gocache',
    'GOMODCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
    'GOPROXY': 'off', 'GOSUMDB': 'off', 'GOENV': 'off', 'GOWORK': 'off',
    'GOTOOLCHAIN': 'local', 'GOFLAGS': '',
    'TMPDIR': '/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task60-tmp',
    'GOTMPDIR': '/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task60-tmp',
    'PATH': str(pathlib.Path(GO).parent) + ':' + os.environ['PATH'],
}

def digest(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()

def bindings():
    paths = subprocess.check_output([
        'git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z',
        'tools/gomad3', 'tools/gomad3integration', '.github/workflows/gomad3.yml',
        '.github/.golangci.yml', 'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode',
    ], cwd=ROOT).split(b'\0')
    files = {os.fsdecode(path): digest(ROOT / os.fsdecode(path)) for path in paths if path}
    return files, hashlib.sha256(json.dumps(files, sort_keys=True).encode()).hexdigest()

if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
name, directory, command = sys.argv[1:4]
receipt, stdout, stderr, manifest = [OUT / (name + suffix) for suffix in ['.json', '.stdout', '.stderr', '-sources.json']]
if any(path.exists() for path in [receipt, stdout, stderr, manifest]):
    raise SystemExit('Refusing to overwrite retained evidence')
environment = {**os.environ, **ENV}
for key in ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'BASH_ENV']:
    environment.pop(key, None)
environment['SANDBOX_START_DIR'] = str(ROOT)
files, before = bindings()
manifest.write_text(json.dumps(files, indent=2, sort_keys=True) + '\n')
started, clock = datetime.datetime.now(datetime.timezone.utc).isoformat(), time.monotonic()
shell_command = 'cd -- ' + shlex.quote(str(ROOT / directory)) + ' && test "$(pwd -P)" = ' + shlex.quote(str(ROOT / directory)) + ' && ' + command
with stdout.open('xb') as output, stderr.open('xb') as errors:
    child = subprocess.Popen(['env', '-u', 'BASH_ENV', 'bash', '-c', shell_command], cwd=ROOT / directory, env=environment, stdout=output, stderr=errors, start_new_session=True)
    timed_out = False
    try:
        status = child.wait(timeout=600)
    except subprocess.TimeoutExpired:
        timed_out = True
        os.killpg(child.pid, 9)
        status = child.wait()
record = {
    'command': command, 'shell_command': shell_command, 'cwd': str(ROOT / directory),
    'started_at': started, 'elapsed_seconds': round(time.monotonic() - clock, 3),
    'exit_code': status, 'timed_out': timed_out, 'terminal': child.poll() is not None,
    'source_before_sha256': before, 'source_after_sha256': bindings()[1],
    'source_manifest_sha256': digest(manifest), 'source_file_count': len(files),
    'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
    'tools': {path: digest(path) for path in [GO, str(pathlib.Path(GO).with_name('gofmt')), '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype']},
    'gate_runner_sha256': digest(__file__), 'environment': ENV,
    'removed_environment': ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'BASH_ENV'],
    'sandbox_start_dir': str(ROOT), 'stdout_sha256': digest(stdout), 'stderr_sha256': digest(stderr),
}
record['source_unchanged'] = record['source_before_sha256'] == record['source_after_sha256']
receipt.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({key: record[key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_unchanged', 'terminal', 'timed_out']}))
sys.exit(124 if timed_out else status)
