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
ENV = {
    'GOCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',
    'TMPDIR': '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',
    'GOTMPDIR': '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',
    'GOMODCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
    'GOPROXY': 'file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download',
    'GOSUMDB': 'off', 'GOENV': 'off', 'GOWORK': 'off', 'GOTOOLCHAIN': 'local', 'GOFLAGS': '',
    'PATH': str(pathlib.Path(GO).parent) + ':' + os.environ['PATH'],
}

def bindings():
    paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
                                     '.github/workflows/gomad3.yml', '.github/.golangci.yml',
                                     'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode'], cwd=ROOT).split(b'\0')
    result = {}
    for raw in paths:
        if raw:
            path = os.fsdecode(raw)
            result[path] = hashlib.sha256((ROOT / path).read_bytes()).hexdigest()
    return hashlib.sha256(json.dumps(result, sort_keys=True).encode()).hexdigest()

name, directory, command = sys.argv[1:4]
if len(sys.argv) > 4:
    ENV['TMPDIR'] = ENV['GOTMPDIR'] = sys.argv[4]
receipt = OUT / (name + '.json')
log = OUT / (name + '.log')
if receipt.exists() or log.exists():
    raise SystemExit('Refusing to overwrite retained gate evidence')
before = bindings()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
with log.open('xb') as output:
    try:
        asserted = 'cd -- ' + shlex.quote(str(ROOT / directory)) + ' && test "$(pwd -P)" = ' + shlex.quote(str(ROOT / directory)) + ' && ' + command
        child = subprocess.run(['bash', '-c', asserted], cwd=ROOT / directory,
                               env={**os.environ, **ENV}, stdout=output, stderr=subprocess.STDOUT,
                               timeout=600)
        status = child.returncode
    except subprocess.TimeoutExpired:
        status = 'inconclusive: timeout 600s'
after = bindings()
record = {'command': command, 'cwd': str(ROOT / directory), 'started_at': started,
          'ended_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'elapsed_seconds': round(time.monotonic() - clock, 3), 'exit_code': status,
          'source_before_sha256': before, 'source_after_sha256': after, 'source_unchanged': before == after,
          'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
          'go': GO, 'go_sha256': hashlib.sha256(pathlib.Path(GO).read_bytes()).hexdigest(),
          'environment': ENV, 'log_sha256': hashlib.sha256(log.read_bytes()).hexdigest()}
receipt.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({key: record[key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_unchanged']}))
sys.exit(status if isinstance(status, int) else 3)
