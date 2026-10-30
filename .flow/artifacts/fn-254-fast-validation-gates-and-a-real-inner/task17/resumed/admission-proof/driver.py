#!/usr/bin/python3
"""Prepare, freeze, or explicitly execute bounded Linux native admission wiring proof."""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import tomllib

HERE = Path(__file__).resolve().parent
SHA = lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest()
GUARD_ROOT = None
GUARD_OUTPUT = None

def guard():
    if GUARD_ROOT is not None:
        assert shutil.disk_usage(GUARD_ROOT).free >= 8 * 1024**3, 'owner8GiBfreeguard breached'
    if GUARD_OUTPUT is not None:
        assert sum(path.stat().st_size for path in GUARD_OUTPUT.glob('*.log')) <= 16 * 1024**2, 'owned proof log budget exceeded'


def source_files(root):
    paths = set()
    for folder in ['model/umpire/dev/xtask', 'model/dev/xtask', 'model/umpire/dev/guest-driver', 'model/umpire/backends/process']:
        paths.update(path.relative_to(root).as_posix() for path in (root / folder).rglob('*.rs'))
        paths.add(folder + '/Cargo.toml')
    paths.update(['Makefile', 'mise.toml', 'model/umpire/mise.toml', 'model/Cargo.toml', 'model/umpire/Cargo.toml', 'model/Cargo.lock', 'model/umpire/Cargo.lock', 'model/rust-toolchain.toml', 'model/umpire/rust-toolchain.toml', 'model/.config/nextest.toml', 'model/umpire/.config/nextest.toml', 'model/umpire/dev/xtask/src/gate/tasks.toml', 'model/scripts/time-gate.sh', 'model/scripts/test-time-gate.sh', 'model/scripts/test-model-validate.sh', 'model/umpire/scripts/xtask.sh', 'model/scripts/xtask.sh', 'mise.lock', 'model/umpire/mise.lock'])
    paths.update(name for name in ['model/.cargo/config.toml', 'model/umpire/.cargo/config.toml'] if (root / name).exists())
    return {path: SHA(root / path) for path in sorted(paths)}

def registry(root):
    packages = [package for name in ['model/Cargo.lock', 'model/umpire/Cargo.lock'] for package in tomllib.loads((root / name).read_text())['package']]
    return sorted({(p['name'], p['version'], p['source'], p.get('checksum', '')) for p in packages if p.get('source', '').startswith('registry+')})

def snapshot(root):
    dependencies = registry(root)
    assert len(dependencies) == 119
    files = source_files(root)
    binary_inputs = {name: value for name, value in files.items() if name.endswith(('.rs', '/Cargo.toml', '/Cargo.lock', '/rust-toolchain.toml', '/tasks.toml', '/.cargo/config.toml'))}
    task = root / '.flow/tasks/fn-254-fast-validation-gates-and-a-real-inner.17.md'
    return dict(task_spec_sha256=SHA(task), binary_source_files=binary_inputs, root=str(root), head=subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(), source_files=files, registry_tuples=[list(row) for row in dependencies], drivers={path.name: SHA(path) for path in [HERE / 'driver.py', HERE / 'stub_mise.py']})

def record(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')

def events(control):
    path = control / 'events.jsonl'
    return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

def until(predicate, timeout=8):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        guard()
        result = predicate()
        if result:
            return result
        time.sleep(0.01)
    raise AssertionError('bounded admission proof condition timed out')

def wait_child(child, timeout=8):
    until(lambda: child.poll() is not None, timeout)
    return child.wait()

def processes():
    result = {}
    for path in Path('/proc').iterdir():
        if not path.name.isdigit():
            continue
        try:
            stat = (path / 'stat').read_text().rsplit(')', 1)[1].split()
            result[int(path.name)] = dict(ppid=int(stat[1]), state=stat[0], start_ticks=stat[19], argv=(path / 'cmdline').read_bytes().replace(b'\0', b' ').decode(errors='replace'))
        except (OSError, ValueError, IndexError):
            pass
    return result

def descendants(pid):
    table = processes()
    owned = {pid}
    while True:
        expanded = owned | {child for child, info in table.items() if info['ppid'] in owned}
        if expanded == owned:
            return {child: table[child] for child in owned if child in table}
        owned = expanded

def reap():
    while True:
        try:
            child, _ = os.waitpid(-1, os.WNOHANG)
            if child == 0:
                return
        except ChildProcessError:
            return

def fixture(base, label, binary, config=None, git=False):
    directory = base / label
    root = directory / 'framework'
    target, home, tmp, control, binpath = [directory / name for name in ['target', 'cargo-home', 'tmp', 'control', 'bin']]
    for path in [root, target, home, tmp, control, binpath]:
        path.mkdir(parents=True)
    files = {'Cargo.toml': '[package]\nname="umpire-fixtures"\nversion="0.0.1"\nedition="2024"\n[workspace]\nmembers=[]\n', 'src/lib.rs': 'pub const PROOF_ONLY: bool = true;\n', 'mise.toml': '# Stub-only admission fixture; not a production task graph.\n', 'rust-toolchain.toml': '[toolchain]\nchannel="1.99.1"\n', 'Cargo.lock': 'version = 4\n', 'mise.lock': '', '.cargo/config.toml': '', '.config/nextest.toml': '[profile.required]\ntest-threads=4\n', 'dev/guest-driver/Cargo.toml': '[package]\nname="umpire-guest-driver"\nversion="0.0.1"\n'}
    for name, content in files.items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
    shutil.copy2(HERE / 'stub_mise.py', binpath / 'mise')
    (binpath / 'mise').chmod(0o700)
    environment = {key: os.environ[key] for key in ['HOME', 'USER', 'LOGNAME', 'LANG', 'LC_ALL', 'TZ', 'RUSTUP_HOME', 'JAVA_HOME'] if key in os.environ}
    environment.update(PATH=str(binpath) + ':/usr/bin:/bin', UMPIRE_TOOL_ROOT=str(root), UMPIRE_FRAMEWORK_ROOT=str(root), CARGO_TARGET_DIR=str(target), CARGO_HOME=str(home), TMPDIR=str(tmp), ADMISSION_CONTROL=str(control), UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip', MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false')
    record(control / 'config.json', config or {})
    if git:
        for arguments in [['init', '-q'], ['add', '.'], ['-c', 'user.name=UmpireProof', '-c', 'user.email=proof@localhost', '-c', 'core.hooksPath=/dev/null', 'commit', '-qm', 'Admission fixture only']]:
            subprocess.run(['git', *arguments], cwd=root, env=environment, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=5)
    return dict(root=root, target=target, home=home, tmp=tmp, control=control, environment=environment, binary=binary)

def launch(item, arguments, output):
    guard()
    log = output.open('wb')
    child = subprocess.Popen([str(item['binary']), *arguments], cwd=item['root'], env=item['environment'], stdout=log, stderr=subprocess.STDOUT, start_new_session=True, close_fds=True)
    log.close()
    return child

def child_context(base, binary):
    item = fixture(base, 'copied-child', binary)
    destination = item['target'] / 'debug' / 'xtask'
    destination.parent.mkdir()
    assert binary.stat().st_size <= 64 * 1024**2, 'bounded fixture binary copy exceeded64MiB'
    shutil.copy2(binary, destination)
    guard()
    # Structural admission uses an explicitly declared stub; production standalone renewal is separate.
    (destination.parent / 'umpire-guest-driver').write_text('admission proof placeholder, never executed\n')
    item['binary'] = destination
    inputs = [dict(path_hex=path.relative_to(item['root']).as_posix().encode().hex(), sha256=SHA(path), executable=bool(path.stat().st_mode & 0o111)) for path in sorted(item['root'].rglob('*')) if path.is_file()]
    source = base / 'parent-source'
    source.mkdir()
    for path in sorted(item['root'].rglob('*')):
        if path.is_file():
            destination_input = source / path.relative_to(item['root'])
            destination_input.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(path, destination_input)
    manifest = dict(version=1, source=str(source), root=str(item['root']), target=str(item['target']), inputs=inputs, excluded=['.git', 'target', 'build', '.bin'])
    path = base / 'child-inputs.json'
    record(path, manifest)
    bad = dict(manifest, target=str(base / 'wrong-target'))
    record(base / 'child-wrong-target.json', bad)
    bad = json.loads(json.dumps(manifest))
    bad['inputs'][0]['sha256'] = '0' * 64
    record(base / 'child-wrong-input.json', bad)
    environment = dict(item['environment'], UMPIRE_STANDALONE_CHILD='1')
    return dict(binary=str(destination), manifest=str(path), environment=environment, negative_manifests=[dict(path=str(base / 'child-wrong-target.json'), expected='standalone provenance/context mismatch'), dict(path=str(base / 'child-wrong-input.json'), expected='standalone provenance input hashes changed')])

def one_case(base, output, binary, name, holder_kind, kill=False, nested=False):
    case = base / name
    case.mkdir()
    nested_context = child_context(case, binary) if nested else None
    configuration = dict(hold=True, escape=True)
    if nested:
        configuration['nested'] = nested_context
    holder = fixture(case, 'holder', binary, configuration, git=holder_kind == 'quick')
    contender_kind = 'gate' if holder_kind == 'quick' else 'quick'
    contender = fixture(case, 'contender', binary, git=contender_kind == 'quick')
    arguments = lambda item, kind: ['gate'] if kind == 'gate' else ['quick', '--workspace', str(item['root']), '--base', 'HEAD']
    children, observed = [], {}
    result = dict(holder_command=[str(binary), *arguments(holder, holder_kind)], contender_command=[str(binary), *arguments(contender, contender_kind)], fixture_paths={role: {key: str(item[key]) for key in ['root', 'target', 'home', 'tmp']} for role, item in [('holder', holder), ('contender', contender)]}, name=name, holder_kind=holder_kind, contender_kind=contender_kind, abrupt_owner_SIGKILL=kill, nested_isolated_fixture=nested)
    try:
        owner = launch(holder, arguments(holder, holder_kind), output / f'{name}-holder.log')
        children.append(owner)
        until(lambda: any(row['kind'] == 'escaped' for row in events(holder['control'])))
        if nested:
            assert any(row['kind'] == 'nested-child-return' and row['exit_code'] == 0 for row in events(holder['control']))
        tree = descendants(owner.pid)
        observed.update(tree)
        guardian = [pid for pid, row in tree.items() if row['ppid'] == owner.pid]
        assert len(guardian) == 1, tree
        expected_gone = sorted(set(tree) - {owner.pid, guardian[0]})
        record(contender['control'] / 'config.json', dict(must_be_gone=expected_gone))
        other = launch(contender, arguments(contender, contender_kind), output / f'{name}-contender.log')
        children.append(other)
        time.sleep(0.15)
        assert other.poll() is None and not events(contender['control']), 'independent invocation entered before release'
        if kill:
            owner.kill()
        else:
            (holder['control'] / 'release').touch()
        owner_code = wait_child(owner)
        assert owner_code == (-signal.SIGKILL if kill else 0), owner_code
        assert wait_child(other) == 0
        until(lambda: (reap() is None) and all(not Path('/proc', str(pid)).exists() for pid in observed))
        result.update(holder_exit=owner_code, contender_exit=other.returncode, holder_events=events(holder['control']), contender_events=events(contender['control']), owned_processes_before_release=tree, all_observed_owned_processes_absent=True, different_roots_targets_cargo_homes_tmpdirs=all(holder[key] != contender[key] for key in ['root', 'target', 'home', 'tmp']))
        if nested:
            child_events = events(Path(nested_context['environment']['ADMISSION_CONTROL']))
            child_authority = [row for row in child_events if row['kind'] == 'authority']
            holder_authority = next(row for row in events(holder['control']) if row['kind'] == 'authority')
            assert child_authority and all((row['dev'], row['ino']) == (holder_authority['dev'], holder_authority['ino']) for row in child_authority)
            result['copied_child_evidence'] = dict(events=child_events, manifest=json.loads(Path(nested_context['manifest']).read_text()), binary_sha256=SHA(Path(nested_context['binary'])), same_host_lock_identity=True, guest_driver='explicit nonexecuted fixture placeholder; production standalone renewal still required')
    finally:
        for child in children:
            if child.poll() is None:
                observed.update(descendants(child.pid))
                child.kill()
                child.wait(timeout=3)
        # Failure fallback is limited to recorded descendants of proof-owned parents.
        live = processes()
        for pid, original in observed.items():
            if pid not in live or live[pid]['start_ticks'] != original['start_ticks']:
                continue
            try:
                os.kill(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        reap()
        record(output / f'{name}.json', result)
    return result

def forge_cases(base, output, binary):
    item = fixture(base, 'forgery', binary)
    results = []
    for locator in [None, '1', '9']:
        environment = item['environment'].copy()
        if locator is not None:
            environment['UMPIRE_JVM_LEASE_FD'] = locator
        process = subprocess.run([str(binary), '__owned-command', 'gate', '--stage', 'tests'], cwd=item['root'], env=environment, capture_output=True, timeout=5, close_fds=True)
        expected = {None: 'missing owned-command descriptor', '1': 'invalid inherited JVM lease descriptor locator', '9': 'Bad file descriptor'}[locator]
        assert process.returncode != 0 and not events(item['control']) and expected in process.stderr.decode(errors='replace')
        results.append(dict(locator=locator, exit_code=process.returncode, stderr=process.stderr.decode(errors='replace')))
    victim = base / 'foreign-file'
    victim.write_text('must remain unchanged')
    environment = dict(item['environment'], UMPIRE_JVM_LEASE_FD='9')
    process = subprocess.run(['/bin/sh', '-c', 'exec 9<"$1"; exec "$2" __owned-command gate --stage tests', 'forged', str(victim), str(binary)], cwd=item['root'], env=environment, capture_output=True, timeout=5, close_fds=True)
    assert process.returncode != 0 and not events(item['control']) and victim.read_text() == 'must remain unchanged' and 'JVM lease is not a stable private owned regular file' in process.stderr.decode(errors='replace')
    results.append(dict(locator='9/foreign-regular-file', exit_code=process.returncode, stderr=process.stderr.decode(errors='replace')))
    record(output / 'forgery-cases.json', results)
    return results

def main():
    global GUARD_ROOT, GUARD_OUTPUT
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-root', required=True, type=Path)
    parser.add_argument('--prepare', action='store_true')
    parser.add_argument('--freeze', action='store_true')
    parser.add_argument('--execute', action='store_true')
    parser.add_argument('--seal', required=True, type=Path)
    parser.add_argument('--binary', type=Path)
    parser.add_argument('--build-receipt', type=Path)
    parser.add_argument('--output', type=Path)
    options = parser.parse_args()
    root = options.source_root.resolve()
    current = snapshot(root)
    assert sum([options.prepare, options.freeze, options.execute]) == 1
    if options.prepare:
        record(options.seal, dict(current, frozen=False, execution='not run; root joined source may change before freeze'))
        return
    assert options.binary and options.build_receipt
    binary = options.binary.resolve()
    build = json.loads(options.build_receipt.read_text())
    assert build['exit_code'] == 0 and build['source_files'] == current['binary_source_files']
    assert build['binaries'][str(binary)] == SHA(binary)
    if options.freeze:
        record(options.seal, dict(current, frozen=True, binary=str(binary), binary_sha256=SHA(binary), build_receipt_sha256=SHA(options.build_receipt)))
        return
    seal = json.loads(options.seal.read_text())
    assert seal['frozen'] and all(seal[key] == current[key] for key in ['source_files', 'registry_tuples', 'drivers', 'task_spec_sha256'])
    assert seal['binary_sha256'] == SHA(binary) and seal['build_receipt_sha256'] == SHA(options.build_receipt)
    assert sys.platform == 'linux', 'this protocol does not establish Darwin escaped-descendant cleanup'
    assert 'UMPIRE_JVM_LEASE_FD' not in os.environ, 'run admission proof outside an already leased gate'
    assert shutil.disk_usage(root).free >= 8 * 1024**3
    # Test harness only: reap orphaned native supervisors after deliberately killing their owner.
    assert ctypes.CDLL(None, use_errno=True).prctl(36, 1, 0, 0, 0) == 0
    assert options.output
    options.output.mkdir(parents=True, exist_ok=False)
    GUARD_ROOT, GUARD_OUTPUT = root, options.output
    with tempfile.TemporaryDirectory(prefix='umpire-fn25417-admission-', dir='/tmp') as directory:
        base = Path(directory)
        results = [one_case(base, options.output, binary, 'gate-then-quick-normal', 'gate'), one_case(base, options.output, binary, 'quick-then-gate-owner-kill', 'quick', kill=True), one_case(base, options.output, binary, 'parent-isolated-child-handoff', 'gate', nested=True)]
        forged = forge_cases(base, options.output, binary)
    assert source_files(root) == current['source_files']
    record(options.output / 'summary.json', dict(scope='Linux native public entrypoint resource/admission wiring with explicit stub leaves; NOT production-suite coverage', source_seal_sha256=SHA(options.seal), binary_sha256=SHA(binary), cases=results, forged_cases=forged, actual_Cargo_JVM_Go_executions=0, production_quick_and_standalone_semantic_renewal='separate owner admission; not executed'))

if __name__ == '__main__':
    main()
