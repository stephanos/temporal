#!/usr/bin/python3
"""Deny-by-default leaf double for native admission proof; never invokes Cargo/Go/Java."""
import fcntl
import json
import os
from pathlib import Path
import subprocess
import sys
import time

root = Path(os.environ['UMPIRE_TOOL_ROOT']).resolve()
control = Path(os.environ['ADMISSION_CONTROL'])
config = json.loads((control / 'config.json').read_text())
args = sys.argv[1:]

def event(kind, **values):
    descriptor = os.open(control / 'events.jsonl', os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
    row = dict(kind=kind, monotonic_ns=time.monotonic_ns(), pid=os.getpid(), ppid=os.getppid(), root=str(root), argv=args, **values)
    os.write(descriptor, (json.dumps(row) + '\n').encode())
    os.close(descriptor)

def authority():
    assert os.environ.get('UMPIRE_JVM_LEASE_FD') == '9'
    opened = os.fstat(9)
    named = os.stat(Path('/tmp').resolve() / f'umpire-validation-{os.getuid()}' / 'jvm.lock')
    assert (opened.st_dev, opened.st_ino) == (named.st_dev, named.st_ino)
    fcntl.flock(9, fcntl.LOCK_EX | fcntl.LOCK_NB)
    for pid in config.get('must_be_gone', []):
        assert not Path('/proc', str(pid)).exists(), f'contender entered before owned PID{pid} cleanup'
    event('authority', dev=opened.st_dev, ino=opened.st_ino, environment={key: os.environ.get(key) for key in ['CARGO_TARGET_DIR', 'CARGO_HOME', 'TMPDIR', 'UMPIRE_BACKENDS', 'UMPIRE_APALACHE', 'UMPIRE_QUICK_CACHE', 'UMPIRE_CACHE_DISABLE', 'MISE_AUTO_INSTALL', 'MISE_EXEC_AUTO_INSTALL', 'NEXTEST_PROFILE', 'NEXTEST_TEST_THREADS']})

def invoke(command, environment):
    # Controlled native children retain the same descriptor; no fake scheduler is claimed.
    return subprocess.run(command, env=environment, close_fds=False, timeout=12).returncode

authority()
if args == ['tasks', 'ls', '--json']:
    print(json.dumps([dict(name='gate:tests', env=['UMPIRE_GATE_STAGE=tests', 'UMPIRE_GATE_ORDER=0', 'UMPIRE_GATE_SCOPE=framework'], depends=[])]))
    raise SystemExit(0)
if args and args[0] == 'run':
    assert args[:-1] == ['run', '--force', '--skip-tools', '--task-cache', 'off', '--jobs', '4'] and args[-1] in ['gate:all', 'gate:child']
    environment = os.environ.copy()
    environment['UMPIRE_GATE_SCOPE'] = 'framework'
    environment['UMPIRE_GATE_STAGE'] = 'tests'
    environment['UMPIRE_GATE_ORDER'] = '0'
    raise SystemExit(invoke([environment['UMPIRE_GATE_EXECUTABLE'], 'gate', '--scheduled-stage', 'tests'], environment))
assert args[:3] == ['exec', '--', 'cargo'], f'unadmitted leaf command: {args}'
cargo = args[3:]
if cargo and cargo[0] == 'metadata':
    package = 'proof-package'
    print(json.dumps(dict(packages=[dict(id=package, name='umpire-fixtures', manifest_path=str(root / 'Cargo.toml'), source=None, targets=[dict(kind=['lib'], doctest=True)])], workspace_members=[package], resolve=dict(nodes=[dict(id=package, dependencies=[])]))))
elif cargo[:2] == ['nextest', 'list']:
    print(json.dumps({'rust-suites': {'proof': {'package-name': 'umpire-fixtures', 'binary-name': 'proof', 'kind': 'test', 'status': 'listed', 'testcases': {'proof::resource_stub': {'ignored': False, 'filter-match': {'status': 'matches'}}}}}}))
elif cargo[:2] == ['nextest', 'run']:
    event('leaf-enter')
    if config.get('nested'):
        child = config['nested']
        environment = {key: os.environ[key] for key in ['PATH', 'HOME', 'LANG'] if key in os.environ}
        environment.update(child['environment'])
        environment['UMPIRE_JVM_LEASE_FD'] = '9'
        code = invoke([child['binary'], 'gate', '--standalone-child', child['manifest']], environment)
        assert code == 0, f'isolated native child refused: {code}'
        event('nested-child-return', exit_code=code)
        for invalid in child['negative_manifests']:
            result = subprocess.run([child['binary'], 'gate', '--standalone-child', invalid['path']], env=environment, close_fds=False, capture_output=True, timeout=8)
            sys.stdout.buffer.write(result.stdout)
            sys.stderr.buffer.write(result.stderr)
            assert result.returncode != 0 and invalid['expected'] in result.stderr.decode(errors='replace'), 'wrong standalone provenance refusal'
            event('nested-child-refusal', manifest=invalid['path'], exit_code=result.returncode, expected=invalid['expected'])
        environment['UMPIRE_STANDALONE_MANIFEST'] = child['manifest']
        code = invoke([child['binary'], 'gate', '--stage', 'standalone'], environment)
        assert code != 0, 'standalone recursion admitted'
        event('nested-child-recursion-refusal', exit_code=code)
    if config.get('escape'):
        # This escaped process deliberately closes FD9. Only native-supervisor custody protects cleanup.
        escaped = subprocess.Popen(['/bin/sleep', '20'], start_new_session=True, close_fds=True)
        event('escaped', escaped_pid=escaped.pid)
    if config.get('hold'):
        deadline = time.monotonic() + 15
        while not (control / 'release').exists():
            assert time.monotonic() < deadline, 'proof leaf hold deadline'
            time.sleep(0.01)
    event('leaf-return')
elif cargo and cargo[0] in ['fmt', 'clippy']:
    event('stub-compile-or-format', operation=cargo[0])
elif cargo[:2] == ['test', '--doc']:
    event('stub-doctest')
else:
    raise AssertionError(f'unadmitted Cargo command: {cargo}')
