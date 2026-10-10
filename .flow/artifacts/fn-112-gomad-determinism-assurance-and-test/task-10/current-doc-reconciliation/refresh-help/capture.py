import datetime
import hashlib
import json
import os
import pathlib
import re
import shlex
import subprocess
import tempfile
import time

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/task11210')
OUT = pathlib.Path(__file__).resolve().parent
PARENT = pathlib.Path('/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t')
GOROOT = pathlib.Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
MODCACHE = pathlib.Path('/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc')
GO = GOROOT / 'bin/go'
MODULE = ROOT / 'tools/gomad3'
EXPECTED_HEAD = '5da272a872195d91f21489567846824857cca4f4'
AGGREGATE = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(name, value):
    with (OUT / name).open('x') as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write('\n')


def git(*args):
    return subprocess.check_output(['/usr/bin/git', *args], cwd=ROOT).decode().strip()


def snapshot():
    paths = set()
    for directory in (MODULE, MODCACHE / 'golang.org/x/mod@v0.37.0', GOROOT / 'src', GOROOT / 'pkg', GOROOT / 'bin'):
        paths.update(path for path in directory.rglob('*') if path.is_file())
    paths.add(GOROOT / 'VERSION')
    aggregate = json.loads(AGGREGATE.read_bytes())
    paths.update(ROOT / path for path in aggregate)
    paths.update([AGGREGATE, pathlib.Path(__file__), pathlib.Path('/usr/bin/python3'), pathlib.Path('/usr/bin/git')])
    return {str(path): sha(path) for path in sorted(paths)}


def run(name, argv, expected, env):
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock = time.monotonic()
    with (OUT / (name + '.stdout')).open('xb') as stdout, (OUT / (name + '.stderr')).open('xb') as stderr:
        process = subprocess.Popen(argv, cwd=MODULE, env=env, stdout=stdout, stderr=stderr)
        timed_out = False
        try:
            code = process.wait(timeout=600)
        except subprocess.TimeoutExpired:
            timed_out = True
            process.kill()
            code = process.wait()
    record = {
        'argv': [str(arg) for arg in argv], 'cwd': str(MODULE),
        'started_utc': started, 'finished_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
        'elapsed_seconds': time.monotonic() - clock, 'exit_code': code,
        'expected_exit_code': expected, 'terminal': process.poll() is not None,
        'timed_out': timed_out, 'stdout_sha256': sha(OUT / (name + '.stdout')),
        'stderr_sha256': sha(OUT / (name + '.stderr')),
        'stdout_bytes': (OUT / (name + '.stdout')).stat().st_size,
        'stderr_bytes': (OUT / (name + '.stderr')).stat().st_size,
    }
    save(name + '.json', record)
    print(json.dumps({'command': name, 'exit_code': code, 'elapsed_seconds': record['elapsed_seconds'], 'terminal': record['terminal']}), flush=True)
    return record


if pathlib.Path.cwd().resolve() != ROOT or git('rev-parse', 'HEAD') != EXPECTED_HEAD:
    raise SystemExit('workspace or expected HEAD mismatch')
if git('status', '--porcelain', '--untracked-files=no'):
    raise SystemExit('tracked worktree changes before capture')
if any(path.name != 'capture.py' for path in OUT.iterdir()):
    raise SystemExit('refusing to overwrite an existing capture')
private = pathlib.Path(tempfile.mkdtemp(prefix='refresh-help.', dir=PARENT))
(private / 'tmp').mkdir()
(private / 'cache').mkdir()
binary = private / 'gomadtool'
env = dict(os.environ)
removed = sorted(key for key in env if key.startswith('GOMAD') or key in {
    'BASH_ENV', 'GOOS', 'GOARCH', 'GOEXPERIMENT', 'GOROOT', 'GOFLAGS', 'GODEBUG',
    'GOPATH', 'GOCACHEPROG', 'CGO_ENABLED', 'CGO_CFLAGS', 'CGO_CPPFLAGS',
    'CGO_CXXFLAGS', 'CGO_FFLAGS', 'CGO_LDFLAGS', 'CC', 'CXX', 'GOAMD64',
    'GOARM64', 'GOARM', 'GO386', 'GOTOOLDIR', 'GONOSUMDB', 'GONOPROXY', 'GOPRIVATE',
})
for key in removed:
    del env[key]
overrides = {
    'GOMODCACHE': str(MODCACHE), 'GOCACHE': str(private / 'cache'),
    'GOSUMDB': 'off', 'GOWORK': 'off', 'GOENV': 'off', 'GOTOOLCHAIN': 'local',
    'GOPROXY': 'file://' + str(MODCACHE / 'cache/download'),
    'GOFLAGS': '-x -work', 'TMPDIR': str(private / 'tmp'),
}
env.update(overrides)
metadata = {
    'head': EXPECTED_HEAD, 'private_directory': str(private), 'binary': str(binary),
    'environment_overrides': overrides, 'cleared_present_variables': removed,
    'python_executable': '/usr/bin/python3', 'go_executable': str(GO),
    'host_uname': list(os.uname()),
    'stock_toolchain_version_file': (GOROOT / 'VERSION').read_text(),
    'scope': 'one stock-Go build and one refresh help observation; no native qualification',
    'build_instrumentation': 'Fresh private GOCACHE and GOFLAGS=-x -work expose compiler commands without an additional Go invocation.',
}
save('admission.json', metadata)
before = snapshot()
save('inputs-before.json', before)
build = run('build', [str(GO), 'build', '-trimpath', '-o', str(binary), './cmd/gomadtool'], 0, env)
help_record = None
if build['exit_code'] == 0 and not build['timed_out']:
    save('binary-before.json', {'path': str(binary), 'sha256': sha(binary), 'bytes': binary.stat().st_size})
    help_record = run('refresh-help', [str(binary), 'compatibility-pack', 'refresh', '-h'], 2, env)
    save('binary-after.json', {'path': str(binary), 'sha256': sha(binary), 'bytes': binary.stat().st_size})
after = snapshot()
save('inputs-after.json', after)
consumed = {}
packages = []
unbound = []
for line in (OUT / 'build.stderr').read_text().splitlines():
    if '/compile ' not in line or not line.startswith(str(GOROOT / 'pkg/tool')):
        continue
    args = shlex.split(line)
    packages.append(args[args.index('-p') + 1])
    for arg in args:
        if not arg.endswith('.go'):
            continue
        path = pathlib.Path(arg)
        if not path.is_absolute():
            path = MODULE / path
        key = str(path)
        if key in before:
            consumed[key] = {'before_sha256': before[key], 'after_sha256': after.get(key)}
        elif path.is_relative_to(private):
            consumed[key] = {'generated_private_build_input_sha256': sha(path)}
        else:
            unbound.append(key)
aggregate = json.loads(AGGREGATE.read_bytes())
aggregate_mismatches = [path for path, expected in aggregate.items() if before.get(str(ROOT / path)) != expected or after.get(str(ROOT / path)) != expected]
stderr = (OUT / 'refresh-help.stderr').read_text() if help_record else ''
flags = sorted(re.findall(r'^  -([a-z-]+)(?:\s|$)', stderr, re.M))
expected_flags = ['baseline-ref', 'compatibility-root', 'go', 'impact-report', 'root']
sealed = OUT.parent
sealed_changes = [path for path in git('diff', '--name-only', EXPECTED_HEAD, '--', str(sealed)).splitlines() if '/refresh-help/' not in path]
result = {
    'head_before': EXPECTED_HEAD, 'head_after': git('rev-parse', 'HEAD'),
    'input_count': len(before), 'inputs_unchanged': before == after,
    'input_deltas': sorted(path for path in before.keys() | after.keys() if before.get(path) != after.get(path)),
    'aggregate_manifest': str(AGGREGATE.relative_to(ROOT)), 'aggregate_input_count': len(aggregate),
    'aggregate_manifest_sha256': sha(AGGREGATE), 'aggregate_mismatches': aggregate_mismatches,
    'compiled_package_count': len(packages), 'compiled_packages': sorted(packages),
    'consumed_go_input_count': len(consumed), 'consumed_go_inputs': consumed, 'unbound_consumed_go_inputs': unbound,
    'build': build, 'refresh_help': help_record, 'refresh_flags': flags,
    'expected_refresh_flags': expected_flags, 'sealed_packet_changes': sealed_changes,
    'historical_refresh_provenance_gaps': 'Both prior hash-only observations remain unchanged; this is one new current-source execution.',
}
result['checks_pass'] = bool(
    build['exit_code'] == 0 and help_record and help_record['exit_code'] == 2
    and not build['timed_out'] and not help_record['timed_out']
    and help_record['stdout_bytes'] == 0 and flags == expected_flags
    and 'Usage of gomadtool compatibility-pack refresh:' in stderr
    and 'usage: gomadtool compatibility-pack refresh --root=DIR' in stderr
    and before == after and not aggregate_mismatches and not unbound and packages
    and not sealed_changes and result['head_after'] == EXPECTED_HEAD
    and sha(binary) == json.loads((OUT / 'binary-before.json').read_text())['sha256'])
save('evidence.json', result)
print(json.dumps({'checks_pass': result['checks_pass'], 'input_count': len(before), 'compiled_packages': len(packages), 'consumed_go_inputs': len(consumed), 'unbound': unbound, 'aggregate_mismatches': aggregate_mismatches}), flush=True)
raise SystemExit(0 if result['checks_pass'] else 1)
