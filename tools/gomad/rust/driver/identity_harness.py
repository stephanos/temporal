#!/usr/bin/env python3
"""Fixed build and raw reader for the native implementation_identity integration harness."""
import argparse
import json
import os
from pathlib import Path
import re
import stat
import sys

from driver import HERE, command, configured, guard, read, require, sha, write
import core_identity
import live
import native_roles
import native_gnu
from primary_stages import enclose_child

SCHEMA = 'gomad.identity-harness-build/v1'
ADMISSION = 'gomad.identity-harness-admission/v1'


def owned_root(config):
    return Path(config['root']) / 'stages/ordinary/identity-harness'


def plan(config):
    path = live.option(config['stages']['validate']['argv'], '--plan')
    require(path, 'identity harness requires the fixed static tool plan')
    value = native_roles.read(path)
    require(value['schema'] == 'gomad.driver-static-plan/v1' and value['repo'] == config['repo']
            and (not config.get('_config_path') or value['driver_config'] == config['_config_path']), 'foreign identity tool plan')
    return Path(path), value


def tree(root):
    root = Path(root)
    require(root.is_absolute() and root.resolve() == root and root.is_dir(), 'identity harness requires an actual canonical input directory')
    files = set()
    for directory, directories, names in os.walk(root):
        for name in directories + names:
            path = Path(directory) / name
            require(not path.is_symlink(), 'identity compiler/cache input has a symlink')
        for name in names:
            path = Path(directory) / name
            require(path.is_file(), 'identity compiler/cache input is not regular')
            files.add(str(path))
    return files


def static_inputs(config):
    """Expand only actual static compiler/cache inputs; no build output is known yet."""
    path, tools = plan(config)
    envelope_path, envelope = native_gnu.inspect(config, path)
    files = {str(path), str(envelope_path)} | set(envelope['files'])
    home = Path(tools['caches']['cargo_home'])
    require(home.resolve() == home and home.is_absolute() and home.is_dir(), 'identity Cargo cache is not its assigned actual directory')
    for name in ('src', 'cache', 'index'):
        files.update(tree(home / 'registry' / name))
    # Cargo configuration must not add source replacements, flags, wrappers, targets or executable hooks.
    for directory in (Path(config['repo']), *Path(config['repo']).parents, Path(config['repo']) / 'tools/gomad/rust', home):
        for name in ('config', 'config.toml'):
            candidate = directory / name if directory == home else directory / '.cargo' / name
            if candidate.exists():
                import tomllib
                native_roles.regular(candidate)
                require(not tomllib.loads(candidate.read_text()), 'identity harness refuses ambient Cargo configuration')
                files.add(str(candidate))
    return files


def preflight(config):
    """Refuse incomplete frozen inputs before starting the fixed Cargo build."""
    static = static_inputs(config)
    _, envelope = native_gnu.inspect(config, plan(config)[0])
    require(all(config['files'].get(path) == digest for path, digest in envelope['files'].items()),
            'identity GNU envelope differs from the frozen input table')
    sources = core_identity.source_inputs(config)
    crate = Path(config['repo']) / 'tools/gomad/rust/wasmhost'
    static.update(str((crate / name).resolve()) for name in sources)
    static.add(str(crate / 'tests/implementation_identity.rs'))
    require(all(path in config['files'] for path in static), 'identity harness omits a static frozen compiler/source/cache input')
    for path in static:
        native_roles.pinned(config, path)
    return static


def environment(config, root, target=None):
    _, selected = plan(config)
    tools = selected['tools']
    _, envelope = native_gnu.inspect(config, plan(config)[0])
    result = dict(PATH=str(Path(tools['rustc']).parent) + ':' + str(Path(envelope['root']) / 'bin'), HOME=str(root / 'home'), TMPDIR=str(root / 'tmp'),
                  CARGO_HOME=selected['caches']['cargo_home'], CARGO_TARGET_DIR=str(root / 'target'),
                  CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0', RUSTC=tools['rustc'], RUSTDOC=tools['rustdoc'],
                  RUSTC_WRAPPER='', RUSTC_WORKSPACE_WRAPPER='', RUSTFLAGS='', CARGO_ENCODED_RUSTFLAGS='',
                  RUSTDOCFLAGS='', RUST_TEST_THREADS='1', LC_ALL='C', TZ='UTC', CC=envelope['linker'], CXX=envelope['linker'], AR=envelope['programs']['ar'],
                  RANLIB=envelope['programs']['ranlib'], PYTHONNOUSERSITE='1', PYTHONDONTWRITEBYTECODE='1')
    if target:
        require(re.fullmatch('[a-zA-Z0-9_-]+', target), 'identity compiler host target is malformed')
        require(target == {'aarch64-linux-gnu': 'aarch64-unknown-linux-gnu', 'x86_64-linux-gnu': 'x86_64-unknown-linux-gnu'}[envelope['machine']],
                'identity Rust compiler and sealed GNU linker target differ')
        result['CARGO_TARGET_' + target.replace('-', '_').upper() + '_LINKER'] = envelope['linker']
    return result


def cargo_command(config):
    _, selected = plan(config)
    return [selected['tools']['cargo'], 'test', '--release', '--locked', '--offline', '--no-default-features',
            '-p', 'gomad-wasmhost', '--test', 'implementation_identity', '--no-run', '--message-format=json',
            '--manifest-path', str(Path(config['repo']) / 'tools/gomad/rust/Cargo.toml')]


def local_command(config, root, name, argv, expected_environment):
    receipt = native_roles.read(root / (name + '.command.json'))
    process = native_roles.read(root / (name + '.process.json'))
    live.completed_command(receipt, process, argv, config['repo'])
    require(receipt['environment'] == expected_environment and receipt['effective_environment'] == expected_environment
            and receipt['environment_mode'] == 'exact' and receipt['temporary_paths'] == config.get('temporary_paths', {}),
            'identity command substituted its actual build environment')
    for field in ('clock', 'parent_pid', 'started_monotonic_nanos', 'reaped_monotonic_nanos'):
        require(receipt[field] == process[field], 'identity command/process lifecycle differs')
    require(receipt['clock'] == 'CLOCK_MONOTONIC' and receipt['parent_pid'] == process['stage_pid']
            and type(receipt['parent_pid']) is int and receipt['parent_pid'] > 0
            and type(receipt['started_monotonic_nanos']) is int and type(receipt['reaped_monotonic_nanos']) is int
            and 0 < receipt['started_monotonic_nanos'] < receipt['reaped_monotonic_nanos']
            and receipt['start_ns'] >= config['sent_epoch'] * 1_000_000_000
            and receipt['end_ns'] - receipt['start_ns'] <= 3600_000_000_000, 'identity command has no actual current process boundary')
    for stream in ('stdout', 'stderr'):
        native_roles.regular(root / (name + '.' + stream))
    return receipt


def compiler_context(config, root):
    _, selected = plan(config)
    base = environment(config, root)
    for name, argv in (('rustc-version', [selected['tools']['rustc'], '--version']),
                       ('rustc-verbose', [selected['tools']['rustc'], '-vV']),
                       ('cargo-version', [selected['tools']['cargo'], '--version'])):
        local_command(config, root, name, argv, base)
    version = (root / 'rustc-version.stdout').read_text()
    verbose = (root / 'rustc-verbose.stdout').read_text()
    cargo = (root / 'cargo-version.stdout').read_text()
    targets = re.findall(r'^host: ([a-zA-Z0-9_-]+)$', verbose, re.MULTILINE)
    require(version.startswith('rustc 1.99.1 ') and version.endswith('\n') and len(version.splitlines()) == 1
            and verbose.splitlines()[0] + '\n' == version and re.findall(r'^release: (.*)$', verbose, re.MULTILINE) == ['1.99.1']
            and len(targets) == 1 and cargo.startswith('cargo 1.99.1 ') and len(cargo.splitlines()) == 1,
            'identity harness compiler version/target differs from actual pinned Rust 1.99.1')
    return dict(compiler_utf8=version, bindings=dict(PROFILE='release', TARGET=targets[0]))


def artifact(config, root):
    path = root / 'cargo-build.stdout'
    frames = [native_roles.decode(line) for line in path.read_text().splitlines() if line]
    artifacts = [frame for frame in frames if frame.get('reason') == 'compiler-artifact'
                 and frame.get('target', {}).get('name') == 'implementation_identity' and frame.get('executable') is not None]
    finished = [frame for frame in frames if frame.get('reason') == 'build-finished']
    scripts = [frame for frame in frames if frame.get('reason') == 'build-script-executed'
               and frame.get('package_id', '').endswith('#gomad-wasmhost@0.1.0')]
    require(len(artifacts) == len(finished) == len(scripts) == 1 and frames[-1] == finished[0] and finished[0]['success'] is True,
            'identity Cargo build did not produce one completed test artifact and helper build script')
    frame = artifacts[0]
    crate = Path(config['repo']) / 'tools/gomad/rust/wasmhost'
    executable = Path(frame['executable'])
    require(frame['target']['kind'] == ['test'] and frame['target']['crate_types'] == ['bin']
            and frame['target']['src_path'] == str(crate / 'tests/implementation_identity.rs')
            and frame['manifest_path'] == str(crate / 'Cargo.toml') and frame['features'] == []
            and frame['profile']['opt_level'] == '3' and frame['profile']['debug_assertions'] is False and frame['profile']['test'] is True and frame.get('fresh') is False
            and executable.parent == root / 'target/release/deps' and re.fullmatch(r'implementation_identity-[0-9a-f]+', executable.name),
            'identity Cargo artifact has foreign source, feature, profile or output root')
    require(frame['package_id'] == scripts[0]['package_id'] == 'path+' + crate.as_uri() + '#gomad-wasmhost@0.1.0',
            'identity Cargo artifact borrowed another package/build script')
    native_roles.regular(executable)
    require(executable.resolve() == executable and executable.stat().st_mode & 0o111, 'identity Cargo executable is not its actual owned file')
    output = Path(scripts[0]['out_dir'])
    require(output.parent.parent == root / 'target/release/build' and output.name == 'out'
            and re.fullmatch('gomad-wasmhost-[0-9a-f]+', output.parent.name) and output.resolve() == output,
            'identity build-script captures borrowed another target root')
    return executable, output


def build_evidence(config, root):
    context = compiler_context(config, root)
    local_command(config, root, 'cargo-build', cargo_command(config), environment(config, root, context['bindings']['TARGET']))
    executable, generated = artifact(config, root)
    sources = core_identity.source_inputs(config)
    manifest = native_roles.source_manifest(sources)
    bindings = ''.join(name + '\t' + value + '\n' for name, value in sorted(context['bindings'].items()))
    expected = {'measurement-source-inputs.tsv': manifest, 'identity-bindings.tsv': bindings, 'identity-compiler.txt': context['compiler_utf8']}
    for name, value in expected.items():
        path = native_roles.regular(generated / name)
        require(path.resolve() == path and path.read_text() == value, 'identity build-script source/compiler/context capture differs from actual frozen inputs')
    static = static_inputs(config)
    static.update(str((Path(config['repo']) / 'tools/gomad/rust/wasmhost' / name).resolve()) for name in sources)
    static.add(str(Path(config['repo']) / 'tools/gomad/rust/wasmhost/tests/implementation_identity.rs'))
    require(all(path in config['files'] for path in static), 'identity harness omits a static frozen compiler/source/cache input')
    commands = {name: {suffix: sha(root / (name + '.' + suffix)) for suffix in ('command.json', 'process.json', 'stdout', 'stderr')}
                for name in ('rustc-version', 'rustc-verbose', 'cargo-version', 'cargo-build')}
    return dict(schema=SCHEMA, source_sha256=config['source_sha256'], config_sha256=config['_config_sha256'],
                context=context, source_inputs={name: sha(Path(config['repo']) / 'tools/gomad/rust/wasmhost' / name) for name in sources},
                inputs={path: config['files'][path] for path in sorted(static)}, command=cargo_command(config),
                executable=str(executable), executable_sha256=sha(executable), generated={name: sha(generated / name) for name in expected}, commands=commands)


def admission(config, root, build):
    path = root / 'admitted/implementation_identity'
    native_roles.regular(path)
    require(path.resolve() == path and path.stat().st_nlink == 1 and stat.S_IMODE(path.stat().st_mode) == 0o755
            and sha(path) == build['executable_sha256'] and path != Path(build['executable']), 'identity derived executable lost its exact Cargo artifact custody')
    expected = dict(schema=ADMISSION, build_receipt=str(root / 'build-receipt.json'), build_receipt_sha256=sha(root / 'build-receipt.json'),
                    cargo_artifact=build['executable'], cargo_artifact_sha256=build['executable_sha256'], executable=str(path), executable_sha256=sha(path))
    require(native_roles.read(root / 'admission.json') == expected, 'identity derived-exec admission changed')
    return path


def producer_argv(config, root, mode):
    return [config['python'], str(HERE / 'identity_harness.py'), mode, '--config', config['_config_path'], '--root', str(root)]


def produce(config, root):
    require(root == owned_root(config) and root.resolve() == root and root.is_dir(), 'foreign identity harness root')
    guard(config)
    preflight(config)
    for name in ('home', 'tmp'):
        (root / name).mkdir(mode=0o700)
    require(not (root / 'target').exists(), 'identity target must be fresh and isolated')
    _, selected = plan(config)
    base = environment(config, root)
    for name, argv in (('rustc-version', [selected['tools']['rustc'], '--version']), ('rustc-verbose', [selected['tools']['rustc'], '-vV']),
                       ('cargo-version', [selected['tools']['cargo'], '--version'])):
        command(dict(config, environment=base), root, name, argv, timeout=60, exact_environment=base)
    context = compiler_context(config, root)
    build_env = environment(config, root, context['bindings']['TARGET'])
    command(dict(config, environment=build_env), root, 'cargo-build', cargo_command(config), timeout=3600, exact_environment=build_env)
    build = build_evidence(config, root)
    write(root / 'build-receipt.json', build)
    (root / 'admitted').mkdir(mode=0o700)
    executable = root / 'admitted/implementation_identity'
    with executable.open('xb') as output:
        os.chmod(executable, 0o755)
        output.write(Path(build['executable']).read_bytes())
    write(root / 'admission.json', dict(schema=ADMISSION, build_receipt=str(root / 'build-receipt.json'), build_receipt_sha256=sha(root / 'build-receipt.json'),
        cargo_artifact=build['executable'], cargo_artifact_sha256=build['executable_sha256'], executable=str(executable), executable_sha256=sha(executable)))
    derived = dict(config, files=dict(config['files'], **{str(executable): sha(admission(config, root, build))}), environment=build_env)
    command(derived, root, 'identity-test', [str(executable), '--nocapture'], timeout=3600, exact_environment=build_env)
    core_identity.observations(config, (root / 'identity-test.stdout').read_text(), context)
    guard(config)
    preflight(config)


def collect(config, root):
    require(root == owned_root(config) and root.resolve() == root, 'foreign identity harness reader root')
    guard(config)
    preflight(config)
    import receiving
    verifier = receiving.Retained(config)
    verifier.command(config, root, 'producer', producer_argv(config, root, 'produce'), timeout=3600)
    outer, process = native_roles.read(root / 'producer.command.json'), native_roles.read(root / 'producer.process.json')
    base = environment(config, root)
    require(outer['environment_mode'] == 'exact' and outer['effective_environment'] == base and outer['environment'] == base,
            'identity producer inherited an ambient execution environment')
    build = build_evidence(config, root)
    require(native_roles.read(root / 'build-receipt.json') == build, 'identity build receipt differs from independently reopened raw build')
    executable = admission(config, root, build)
    build_env = environment(config, root, build['context']['bindings']['TARGET'])
    derived = dict(config, files=dict(config['files'], **{str(executable): sha(executable)}), environment=build_env)
    receiving.Retained(derived).command(derived, root, 'identity-test', [str(executable), '--nocapture'], timeout=3600)
    local_command(config, root, 'identity-test', [str(executable), '--nocapture'], build_env)
    pids = set()
    for name in ('rustc-version', 'rustc-verbose', 'cargo-version', 'cargo-build', 'identity-test'):
        row = native_roles.read(root / (name + '.command.json'))
        require(row['pid'] not in pids, 'identity harness reused a subprocess PID')
        pids.add(row['pid'])
        enclose_child(outer, process, row)
    text = (root / 'identity-test.stdout').read_text()
    require(all('test ' + name in text for name in core_identity.TESTS)
            and re.search(r'test result: ok\. 5 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out;', text),
            'identity harness did not execute its exact five original native tests')
    facts = core_identity.observations(config, text, build['context'])
    guard(config)
    return dict(schema='gomad.identity-harness-facts/v1', source_sha256=config['source_sha256'], identity=config['identity'],
                build_receipt_sha256=sha(root / 'build-receipt.json'), admission_sha256=sha(root / 'admission.json'),
                command_sha256=sha(root / 'identity-test.command.json'), facts=facts)


def execute(config):
    preflight(config)
    root = owned_root(config)
    root.mkdir(mode=0o700)
    base = environment(config, root)
    observed = dict(config, environment=base)
    command(observed, root, 'producer', producer_argv(config, root, 'produce'), timeout=3600, exact_environment=base)
    return json.loads(command(observed, root, 'reader', producer_argv(config, root, 'read'), timeout=3600, exact_environment=base))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('produce', 'read'))
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    config = configured(args.config)
    config.update(_config_path=str(args.config.resolve()), _config_sha256=sha(args.config))
    if args.mode == 'produce':
        produce(config, args.root)
    else:
        print(json.dumps(collect(config, args.root), sort_keys=True))


if __name__ == '__main__':
    main()
