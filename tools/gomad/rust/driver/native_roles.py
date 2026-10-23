"""Independent source, compiler, process and publication reader for native measurement roles."""
import hashlib
import json
from pathlib import Path
import re
import stat

from driver import Refusal, require, sha

BASE_CONFIGURATION = 'cranelift-fuel-nan-canonical-nounwind-epoch-v1'
COMMON = ('build.rs', 'Cargo.toml', '../Cargo.toml', '../Cargo.lock', '../rust-toolchain.toml')
HOST_EXTRA = ('src/measurement/testdata/sqlite_conn.go.orig', 'src/measurement/testdata/LICENSE',
              'src/measurement/testdata/guest-profile.json', '../../internal/backends/wasm/command/testdata/scenario/network.json')
CHOICE = ('choice/schema/choicewire.json', 'choice/schema/choicewire.go.tmpl', 'choice/schema/choicewire_runtime.go.tmpl',
          'choice/schema/choicewire_test.go.tmpl', 'runtimesource/gomad_semantics.go.txt', 'choice/trace.go', 'choice/tape.go')
RUNTIME = ('overlay.go', 'runtime_wasm.go.txt', 'runtime_inspect_wasm.go.txt', 'net_stream_wasm.go.txt', 'net_stream_status_generated.go.txt')
ENVIRONMENT = {'PATH', 'HOME', 'CARGO_HOME', 'RUSTUP_HOME', 'TMPDIR', 'CARGO_TARGET_DIR', 'RUSTC', 'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER'}
BUILD_FIELDS = {'schema', 'crate_root', 'executable', 'executable_sha256', 'rustc', 'cargo', 'inputs', 'command', 'environment', 'role', 'rustc_version', 'cargo_version', 'identity'}
IDENTITY_FIELDS = {'schema', 'role', 'features', 'host_implementation_sha256', 'linked_implementation_sha256', 'runtime_implementation_sha256', 'configuration', 'source_inputs', 'executable_sha256'}
ADMISSION_FIELDS = {'schema', 'build_receipt', 'build_receipt_sha256', 'cargo_artifact', 'cargo_artifact_sha256', 'executable', 'executable_sha256', 'identity'}


def regular(path):
    path = Path(path)
    info = path.lstat()
    require(path.is_absolute() and stat.S_ISREG(info.st_mode) and info.st_size <= 512<<20, 'native role requires absolute regular file: ' + str(path))
    return path


def decode(raw):
    def unique(pairs):
        value = {}
        for name, item in pairs:
            require(name not in value, 'duplicate native evidence field: ' + name)
            value[name] = item
        return value
    return json.loads(raw, object_pairs_hook=unique)


def read(path):
    path = regular(path)
    require(path.stat().st_size <= 64<<20, 'native raw JSON exceeds original byte bound')
    return decode(path.read_bytes())


def pinned(config, path):
    path = regular(path)
    require(config['files'].get(str(path.resolve())) == sha(path), 'native compiler/source input is not frozen: ' + str(path))
    return path


def retained(config, path):
    path = regular(path)
    if path.is_relative_to(Path(config['root'])):
        require(path.resolve() == path, 'native evidence has symlinked ancestor')
    else:
        pinned(config, path.resolve())
    return path


def features(role):
    require(isinstance(role, dict) and role.get('kind') in ('release', 'warm', 'stress', 'diagnostic'), 'unknown native role')
    if role['kind'] == 'diagnostic':
        require(set(role) == {'kind', 'features'}, 'unknown diagnostic role fields')
        values = sorted(role['features'])
        require(values in (['diagnostic-profile'], ['diagnostic-profile', 'guest-profile'],
                           ['diagnostic-fuel-off', 'diagnostic-profile']), 'invalid native diagnostic feature combination')
        return values
    require(set(role) == {'kind'}, 'unknown native role fields')
    return {'release': [], 'warm': ['diagnostic-profile'], 'stress': ['stress-jitter']}[role['kind']]


def role_name(role):
    values = features(role)
    return next((name for name in ('stress-jitter', 'guest-profile', 'diagnostic-fuel-off', 'diagnostic-profile') if name in values), 'ordinary')


def cargo_command(cargo, crate_root, role):
    command = [cargo, 'build', '--manifest-path', str(Path(crate_root) / 'Cargo.toml'), '--release', '--locked',
               '--jobs=1', '--no-default-features', '--message-format=json', '--bin', 'gomad-measure']
    values = ['helper-' + value for value in features(role)]
    return command + (['--features', ','.join(values)] if values else [])


def source_inventory(config, kind):
    root = Path(config['repo']) / 'tools/gomad/rust' / ('gomad' if kind == 'host' else 'wasmhost')
    names = {str(path.relative_to(root)) for path in (root / 'src').rglob('*.rs')}
    require(names and all(not path.is_symlink() for path in (root / 'src').rglob('*')), 'native source closure is empty or contains symlinks')
    names.update(COMMON)
    if kind == 'helper':
        import core_identity
        return root, core_identity.source_inputs(config)
    if kind == 'host':
        names.update(HOST_EXTRA)
        names.update('../../internal/model/' + name for name in CHOICE)
        names.update('../../internal/backends/wasm/toolchain/' + name for name in RUNTIME)
    files = {}
    for name in sorted(names):
        path = (root / name).resolve()
        pinned(config, path)
        files[name] = path.read_bytes()
    return root, files


def source_manifest(files):
    return ''.join(hashlib.sha256(raw).hexdigest() + '\t' + name + '\n' for name, raw in sorted(files.items()))


def feature_bindings(role_features, target):
    environment = dict(PROFILE='release', TARGET=target)
    environment.update({'CARGO_FEATURE_' + feature.replace('-', '_').upper(): '1' for feature in role_features})
    return environment


def implementation(files, domain, role_features, compiler_version, target):
    digest = hashlib.sha256(domain)
    environment = feature_bindings(role_features, target)
    for name, value in sorted(environment.items()):
        digest.update(name.encode() + b'\0' + value.encode() + b'\0')
    digest.update(compiler_version.encode())
    for name, raw in sorted(files.items()):
        digest.update(len(name.encode()).to_bytes(8, 'big') + name.encode() + len(raw).to_bytes(8, 'big') + raw)
    return digest.hexdigest()


def validate_identity(config, identity, role, executable_sha256, rustc_version):
    require(set(identity) == IDENTITY_FIELDS, 'unknown/missing native identity fields')
    inventories = {kind: source_inventory(config, kind) for kind in ('host', 'helper')}
    target = re.findall(r'^host: (\S+)$', rustc_version, re.MULTILINE)
    require(len(target) == 1 and rustc_version.startswith('rustc 1.99.1 '), 'native compiler target/version missing')
    compiler = rustc_version.splitlines()[0] + '\n'
    values = features(role)
    runtime = hashlib.sha256(b'gomad-wasm-runtime-implementation/v1\0')
    for name in RUNTIME:
        raw = inventories['host'][1]['../../internal/backends/wasm/toolchain/' + name]
        runtime.update(len(raw).to_bytes(8, 'big') + raw)
    name = role_name(role)
    expected = dict(schema='gomad.native-measurement-identity/v1', role=name, features=values,
        host_implementation_sha256=implementation(inventories['host'][1], b'gomad.host-implementation/v1\0',
            ['helper-' + value for value in values], compiler, target[0]),
        linked_implementation_sha256=implementation(inventories['helper'][1], b'gomad.linked-implementation/v1\0', values, compiler, target[0]),
        runtime_implementation_sha256=runtime.hexdigest(), configuration=BASE_CONFIGURATION + ('' if name == 'ordinary' else '-' + name),
        source_inputs={kind: source_manifest(files) for kind, (_, files) in inventories.items()}, executable_sha256=executable_sha256)
    require(identity == expected, 'native compiled identity/role/source closure differs from frozen recipe')
    return {str(root / name): hashlib.sha256(raw).hexdigest()
            for root, files in inventories.values() for name, raw in files.items()}


def observation(path, argv, directory, environment):
    value = read(regular(path))
    require(isinstance(value, dict) and {'schema', 'command', 'directory', 'environment', 'pid', 'exit_code', 'signal',
            'termination', 'error', 'cleanup_errors', 'duration_nanos', 'clock', 'parent_pid', 'started_monotonic_nanos',
            'reaped_monotonic_nanos', 'stdout', 'stderr', 'stdout_sha256', 'stderr_sha256'} <= set(value),
            'native command observation is missing required process or lifecycle fields')
    require(value['schema'] == 'gomad.native-process-observation/v1' and value['command'] == argv
            and value['directory'] == str(directory) and value['environment'] == environment
            and type(value['pid']) is int and value['pid'] > 0 and value['exit_code'] == 0 and value['signal'] is None
            and value['termination'] == 'exit' and value['error'] is None and not value['cleanup_errors']
            and type(value['duration_nanos']) is int and value['duration_nanos'] > 0
            and value['clock'] == 'CLOCK_MONOTONIC' and type(value['parent_pid']) is int and value['parent_pid'] > 0
            and type(value['started_monotonic_nanos']) is int and type(value['reaped_monotonic_nanos']) is int
            and 0 < value['started_monotonic_nanos'] < value['reaped_monotonic_nanos'],
            'native command observation differs from actual fixed invocation')
    for stream in ('stdout', 'stderr'):
        stream_path = Path(path).with_suffix('.' + stream)
        capture = value[stream]
        size = regular(stream_path).stat().st_size
        require(size <= 64<<20 and capture == dict(path=str(stream_path), bytes=size, captured_bytes=size,
                sha256=sha(stream_path), truncated=False, error=None) and value[stream + '_sha256'] == sha(stream_path),
                'native process raw capture/hash differs')
    return value


def cargo_artifact(path, role, executable, crate_root):
    frames = [decode(line) for line in regular(path).read_text().splitlines() if line]
    artifacts = [frame for frame in frames if frame.get('reason') == 'compiler-artifact'
                 and frame.get('target', {}).get('name') == 'gomad-measure' and frame.get('executable') is not None]
    finished = [frame for frame in frames if frame.get('reason') == 'build-finished']
    require(len(artifacts) == 1 and len(finished) == 1 and frames[-1] == finished[0] and finished[0].get('success') is True,
            'missing/duplicate native Cargo executable or unsuccessful build')
    artifact = artifacts[0]
    require(artifact['executable'] == executable and artifact['target']['kind'] == ['bin']
            and artifact['target']['src_path'] == str(Path(crate_root) / 'src/bin/measure.rs')
            and sorted(artifact['features']) == ['helper-' + value for value in features(role)]
            and artifact['profile']['opt_level'] == '3' and artifact['profile']['debug_assertions'] is False
            and artifact['profile']['test'] is False, 'wrong native Cargo binary/features/release profile')


def validate_build(config, path, role):
    """Read only; return the native receipt after reopening every source and raw command."""
    receipt = read(retained(config, path))
    require(set(receipt) == BUILD_FIELDS and receipt['schema'] == 'gomad.native-measurement-build/v1'
            and receipt['role'] == role and receipt['crate_root'] == str(Path(config['repo']) / 'tools/gomad/rust/gomad'),
            'foreign native build receipt/schema/role')
    environment = receipt['environment']
    require(set(environment) <= ENVIRONMENT and environment.get('RUSTC') == receipt['rustc']
            and environment.get('RUSTC_WRAPPER') == environment.get('RUSTC_WORKSPACE_WRAPPER') == '', 'native compiler environment changed')
    for tool in ('cargo', 'rustc'):
        pinned(config, receipt[tool])
    require(receipt['command'] == cargo_command(receipt['cargo'], receipt['crate_root'], role)
            and receipt['rustc_version'].startswith('rustc 1.99.1 ') and receipt['cargo_version'].startswith('cargo 1.99.1 '),
            'native pinned compiler/build command changed')
    target = Path(environment['CARGO_TARGET_DIR'])
    logs = target.parent
    require(target.is_absolute() and receipt['executable'] == str(target / 'release/gomad-measure')
            and retained(config, receipt['executable']).stat().st_mode & 0o111 and sha(receipt['executable']) == receipt['executable_sha256'],
            'native isolated Cargo executable changed')
    semantic = validate_identity(config, receipt['identity'], role, receipt['executable_sha256'], receipt['rustc_version'])
    for tool in ('cargo', 'rustc'):
        semantic[receipt[tool]] = sha(receipt[tool])
    ancestors = list(Path(receipt['crate_root']).parents)
    ancestors.insert(0, Path(receipt['crate_root']))
    candidates = [ancestor / suffix for ancestor in ancestors for suffix in ('.cargo/config', '.cargo/config.toml')]
    home = environment.get('CARGO_HOME') or (str(Path(environment['HOME']) / '.cargo') if 'HOME' in environment else None)
    if home:
        candidates += [Path(home) / suffix for suffix in ('config', 'config.toml')]
    for candidate in candidates:
        if candidate.exists():
            semantic[str(pinned(config, candidate))] = sha(candidate)
    require(all(receipt['inputs'].get(name) == digest for name, digest in semantic.items()), 'native build omits frozen semantic input')
    for name, digest in receipt['inputs'].items():
        require(sha(retained(config, name)) == digest, 'native build input changed: ' + name)
    commands = {
        'rustc-path': ['rustup', 'which', '--toolchain', '1.99.1', 'rustc'],
        'cargo-path': ['rustup', 'which', '--toolchain', '1.99.1', 'cargo'],
        'rustc-version': [receipt['rustc'], '-vV'], 'cargo-version': [receipt['cargo'], '--version'],
        'cargo-build': receipt['command'], 'artifact-identity': [receipt['executable'], '--identity']}
    parents = set()
    for name, argv in commands.items():
        for suffix in ('stdout', 'stderr', 'json'):
            command_path = logs / (name + '.' + suffix)
            require(receipt['inputs'].get(str(command_path)) == sha(regular(command_path)), 'native receipt omits actual command evidence')
        # Compiler discovery precedes native insertion of the exact RUSTC controls.
        actual_env = environment if name in ('cargo-build', 'artifact-identity') else {key:value for key,value in environment.items()
            if key not in ('RUSTC', 'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER')}
        observed = observation(logs / (name + '.json'), argv, logs if name == 'artifact-identity' else receipt['crate_root'], actual_env)
        parents.add(observed['parent_pid'])
    require(len(parents) == 1, 'native build commands borrowed another actual producer')
    require((logs / 'rustc-path.stdout').read_text().strip() == receipt['rustc']
            and (logs / 'cargo-path.stdout').read_text().strip() == receipt['cargo']
            and (logs / 'rustc-version.stdout').read_text() == receipt['rustc_version']
            and (logs / 'cargo-version.stdout').read_text() == receipt['cargo_version']
            and read(logs / 'artifact-identity.stdout') == receipt['identity'], 'native raw compiler/identity outputs differ')
    cargo_artifact(logs / 'cargo-build.stdout', role, receipt['executable'], receipt['crate_root'])
    return receipt


def validate_admission(config, path, role):
    admission = read(retained(config, path))
    require(set(admission) == ADMISSION_FIELDS and admission['schema'] == 'gomad.native-measurement-admission/v1', 'foreign native admission schema')
    receipt = validate_build(config, admission['build_receipt'], role)
    executable = retained(config, admission['executable'])
    require(admission['build_receipt_sha256'] == sha(admission['build_receipt'])
            and admission['cargo_artifact'] == receipt['executable'] and admission['identity'] == receipt['identity']
            and admission['cargo_artifact_sha256'] == admission['executable_sha256'] == receipt['executable_sha256'] == sha(executable)
            and executable != Path(receipt['executable']) and executable.stat().st_nlink == 1
            and stat.S_IMODE(executable.stat().st_mode) == 0o755, 'native admission publication/identity differs')
    for name, binary in (('artifact-identity', receipt['executable']), ('staged-identity', str(executable))):
        observation(executable.parent / (name + '.json'), [binary, '--identity'], executable.parent, receipt['environment'])
        require(read(executable.parent / (name + '.stdout')) == receipt['identity'], 'native admission role changed during publication')
    return admission
