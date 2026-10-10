import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-23-lint-complexity-candidate')
PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23/complexity-worker'
OUTPUT = ROOT / '.flow/tmp/fn10923-complexity'
GO_ROOT = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
GO = str(GO_ROOT / 'bin/go')
TOOLS = Path('/tmp/fn109-lint-tools.ZdNe1t50')
BASE = '951c5516e9e7b3066e7e069adda9565cfd68844c'
LINT_ARGS = ['SHELL=/bin/sh', 'GOLANGCI_LINT_BASE_REV=' + BASE, 'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT=' + str(TOOLS / 'golangci-lint-v2.13.0'), 'ERRORTYPE=' + str(TOOLS / 'errortype'), 'ALL_TEST_TAGS=test_dep']

def environment():
    env = dict(os.environ)
    for key in ('BASH_ENV', 'GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'LINT_TEST_BASE_REV', 'LINT_POLICY_GOLANGCI'):
        env.pop(key, None)
    env.update(PATH=str(GO_ROOT / 'bin') + ':/usr/bin:/bin', GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOFLAGS='', CGO_ENABLED='0', TZ='UTC',
               GOCACHE='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache',
               TMPDIR='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp',
               GOTMPDIR='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp',
               GOMODCACHE='/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
               GOPROXY='file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download', GOSUMDB='off')
    return env

def digest(path):
    h = hashlib.sha256()
    with open(path, 'rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()

def sources():
    paths = subprocess.check_output(['/usr/bin/git', 'ls-files', '-z', '--cached', '--others', '--exclude-standard'], cwd=ROOT).decode().split('\0')
    result = {}
    for name in sorted(set(paths)):
        if not name or name.startswith(('.flow/', '.turbo/')):
            continue
        path = ROOT / name
        if path.is_symlink():
            result[name] = {'symlink': os.readlink(path)}
        elif path.is_file():
            result[name] = digest(path)
    for path in sorted(PACKET.glob('*.py')):
        result[str(path)] = digest(path)
    return result

def tools():
    paths = [GO_ROOT / 'bin/go', GO_ROOT / 'bin/gofmt', TOOLS / 'golangci-lint-v2.13.0', TOOLS / 'errortype', Path('/usr/bin/git'), Path('/usr/bin/make'), Path('/bin/sh'), Path('/usr/bin/python3')]
    paths.extend(sorted((GO_ROOT / 'pkg/tool/linux_arm64').iterdir()))
    return {str(path): {'realpath': str(path.resolve()), 'sha256': digest(path.resolve())} for path in paths if path.is_file()}

def settings(cwd, env):
    return json.loads(subprocess.check_output([GO, 'env', '-json'], cwd=cwd, env=env))

def normalized(value):
    result = dict(value)
    result['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)', 'go-build<VOLATILE>', result['GOGCCFLAGS'])
    return result

def process_group(pgid):
    output = subprocess.check_output(['/usr/bin/ps', '-eo', 'pid,ppid,pgid,stat,args'], text=True)
    return [line for line in output.splitlines()[1:] if len(line.split(None, 4)) >= 4 and line.split(None, 4)[2] == str(pgid)]

def write(name, value):
    (OUTPUT / name).write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')

def run(name, argv, cwd=ROOT, extra=None):
    assert ROOT.resolve() == ROOT and subprocess.check_output(['/usr/bin/git', 'rev-parse', '--show-toplevel'], cwd=ROOT, text=True).strip() == str(ROOT)
    OUTPUT.mkdir(parents=True, exist_ok=True)
    assert not (OUTPUT / (name + '.json')).exists(), 'receipts are immutable; choose a fresh name'
    env = environment()
    env.update(extra or {})
    before, tool_before, go_before = sources(), tools(), settings(cwd, env)
    selected = {key: value for key, value in env.items() if key.startswith(('GO', 'CGO', 'LINT_', 'GIT_', 'BASH_')) or key in ('PATH', 'TZ', 'TMPDIR', 'SHELL')}
    start = time.time_ns()
    binding = {'argv': argv, 'cwd': str(cwd), 'head': subprocess.check_output(['/usr/bin/git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(), 'environment': selected, 'actual_go_settings': go_before, 'sources': before, 'tools': tool_before, 'start_utc': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(start / 1e9))}
    write(name + '-binding.json', binding)
    timed_out = False
    with open(OUTPUT / (name + '.log'), 'wb') as output:
        child = subprocess.Popen(argv, cwd=cwd, env=env, stdin=subprocess.DEVNULL, stdout=output, stderr=subprocess.STDOUT, start_new_session=True)
        try:
            status = child.wait(timeout=600)
        except subprocess.TimeoutExpired:
            timed_out = True
            os.killpg(child.pid, signal.SIGTERM)
            try:
                status = child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                status = child.wait()
    end = time.time_ns()
    remaining = process_group(child.pid)
    if remaining:
        os.killpg(child.pid, signal.SIGKILL)
        for attempt in range(100):
            remaining = process_group(child.pid)
            if not remaining:
                break
            time.sleep(0.1)
    after, tool_after, go_after = sources(), tools(), settings(cwd, env)
    receipt = {'argv': argv, 'cwd': str(cwd), 'exit_code': status, 'elapsed_seconds': (end - start) / 1e9, 'start_utc': binding['start_utc'], 'end_utc': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(end / 1e9)), 'timed_out': timed_out, 'binding': name + '-binding.json', 'log': name + '.log', 'log_sha256': digest(OUTPUT / (name + '.log')), 'source_before_after_equal': before == after, 'tools_before_after_equal': tool_before == tool_after, 'settings_before_after_equal_normalized': normalized(go_before) == normalized(go_after), 'actual_go_settings_after': go_after, 'sources_after': after, 'tools_after': tool_after, 'remaining_process_group': remaining, 'all_commands_terminal': not remaining, 'limitations': 'Shared pre-existing build/module/lint caches and inherited nonselected environment are not hermetic. Only numeric go-build temporary paths in raw GOGCCFLAGS are normalized.'}
    write(name + '.json', receipt)
    print(json.dumps({key: receipt[key] for key in ('exit_code', 'elapsed_seconds', 'timed_out', 'source_before_after_equal', 'tools_before_after_equal', 'settings_before_after_equal_normalized', 'all_commands_terminal')}), flush=True)
    assert not remaining and before == after and tool_before == tool_after and normalized(go_before) == normalized(go_after), 'inconclusive binding'
    return status

if __name__ == '__main__':
    sys.exit(run(sys.argv[1], sys.argv[2:]))
