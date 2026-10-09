import datetime
import hashlib
import json
import os
import pathlib
import shlex
import subprocess
import sys
import time

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal').resolve()
OUT = pathlib.Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERRORTYPE = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
ENV = {
    'GOCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',
    'GOMODCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
    'GOPROXY': 'file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download',
    'TMPDIR': '/Users/stephan/Workspace/skunkworks/.gomad-fn109-combined-gates.hSxRA0go',
    'GOTMPDIR': '/Users/stephan/Workspace/skunkworks/.gomad-fn109-combined-gates.hSxRA0go',
    'GOSUMDB': 'off', 'GOENV': 'off', 'GOWORK': 'off', 'GOTOOLCHAIN': 'local', 'GOFLAGS': '',
    'PATH': str(pathlib.Path(GO).parent) + ':' + os.environ['PATH'],
}
REMOVED = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'BASH_ENV', 'GOOS', 'GOARCH', 'GOEXPERIMENT']


def digest(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def bindings():
    paths = subprocess.check_output([
        'git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
        '.github/workflows/gomad3.yml', '.github/.golangci.yml',
        'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode',
    ], cwd=ROOT).split(b'\0')
    files = {os.fsdecode(path): digest(ROOT / os.fsdecode(path)) for path in paths if path}
    fingerprint = hashlib.sha256(json.dumps(files, sort_keys=True).encode()).hexdigest()
    manifest = OUT / ('sources-' + fingerprint + '.json')
    encoded = json.dumps(files, indent=2, sort_keys=True) + '\n'
    if manifest.exists():
        if manifest.read_text() != encoded:
            raise SystemExit('Source manifest collision')
    else:
        manifest.write_text(encoded)
    return fingerprint, manifest


def orchestration():
    return {path.name: digest(path) for path in [pathlib.Path(__file__), OUT / 'run_gates.py']}


if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
name, directory, command = sys.argv[1:4]
receipt, stdout, stderr = [OUT / (name + suffix) for suffix in ['.json', '.stdout', '.stderr']]
if any(path.exists() for path in [receipt, stdout, stderr]):
    raise SystemExit('Refusing to overwrite retained evidence')
environment = {**os.environ, **ENV}
for key in REMOVED:
    environment.pop(key, None)
environment['SANDBOX_START_DIR'] = str(ROOT)
before, manifest = bindings()
scripts_before = orchestration()
started, clock = datetime.datetime.now(datetime.timezone.utc).isoformat(), time.monotonic()
shell_command = 'cd -- ' + shlex.quote(str(ROOT / directory)) + ' && test "$(pwd -P)" = ' + shlex.quote(str(ROOT / directory)) + ' && ' + command
with stdout.open('xb') as output, stderr.open('xb') as errors:
    child = subprocess.Popen(['bash', '-c', shell_command], cwd=ROOT / directory, env=environment, stdout=output, stderr=errors, start_new_session=True)
    timed_out = False
    try:
        status = child.wait(timeout=600)
    except subprocess.TimeoutExpired:
        timed_out = True
        os.killpg(child.pid, 9)
        status = child.wait()
after, _ = bindings()
scripts_after = orchestration()
record = {
    'command': command, 'shell_command': shell_command, 'cwd': str(ROOT / directory),
    'started_at': started, 'elapsed_seconds': round(time.monotonic() - clock, 3),
    'exit_code': status, 'timed_out': timed_out, 'child_pid': child.pid,
    'source_before_sha256': before, 'source_after_sha256': after,
    'source_manifest': manifest.name, 'source_manifest_sha256': digest(manifest),
    'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
    'tools': {path: digest(path) for path in [GO, str(pathlib.Path(GO).with_name('gofmt')), LINT, ERRORTYPE]},
    'gate_runner_sha256': digest(__file__),
    'orchestration_before': scripts_before, 'orchestration_after': scripts_after,
    'environment': ENV, 'removed_environment': REMOVED, 'sandbox_start_dir': str(ROOT),
    'stdout_sha256': digest(stdout), 'stderr_sha256': digest(stderr),
    'terminal': child.poll() is not None,
}
record['source_unchanged'] = before == after
record['orchestration_unchanged'] = scripts_before == scripts_after
receipt.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({key: record[key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_unchanged', 'orchestration_unchanged', 'terminal', 'timed_out']}))
if not record['source_unchanged'] or not record['orchestration_unchanged']:
    raise SystemExit(125)
sys.exit(124 if timed_out else status)
