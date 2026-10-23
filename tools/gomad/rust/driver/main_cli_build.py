#!/usr/bin/env python3
"""Provision and independently read the fixed prefreeze native release roles."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import time

from driver import HERE, Refusal, check_files, command, read, require, sha, write
import identity_harness as harness
import native_gnu
import native_roles

SCHEMA = 'gomad.driver-main-build/v1'
ROLES = {'release': [], 'host-fixture': ['determinism-fixture'],
         'stress': ['determinism-fixture', 'helper-stress-jitter'], 'stress-release': ['helper-stress-jitter'],
         'conformance-reference': ['conformance', 'reference-helper'], 'runtime-harness': [], 'interception-harness': [],
         'counterfactual-fixture': ['determinism-fixture', 'helper-determinism-fixture'],
         'counterfactual-harness': ['determinism-fixture', 'helper-determinism-fixture']}
EXAMPLES = {'runtime-harness': 'runtime-qualification-replay', 'interception-harness': 'interception-streaming-replay'}
COMMANDS = ('rustc-version', 'rustc-verbose', 'cargo-version', 'cargo-build')


def helper_features(role):
    require(role in ROLES, 'unknown main CLI fixed build role')
    if role == 'conformance-reference': return ['conformance', 'reference-helper']
    return [value.removeprefix('helper-') for value in ROLES[role] if value.startswith('helper-')]


def binary_name(role):
    if role == 'conformance-reference': return 'gomad-wasmhost'
    return EXAMPLES.get(role, 'gomad-fixture-libtest' if role == 'counterfactual-harness' else 'gomad')


def source_state(repo, git):
    env = dict(PATH=str(Path(git).parent), GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL='/dev/null', GIT_CONFIG_SYSTEM='/dev/null', GIT_OPTIONAL_LOCKS='0', GIT_NO_REPLACE_OBJECTS='1')
    paths = subprocess.check_output([git, '-C', str(repo), 'ls-files', '-z'], env=env).decode().split('\0')
    require(all('\n' not in name for name in paths), 'unsupported main CLI source filename')
    head = subprocess.check_output([git, '-C', str(repo), 'rev-parse', 'HEAD'], text=True, env=env).strip()
    tree = subprocess.check_output([git, '-C', str(repo), 'ls-tree', '-r', '-z', head], env=env)
    links = {}
    for raw in tree.split(b'\0'):
        if not raw:continue
        metadata, name = raw.split(b'\t', 1)
        mode, kind, object_id = metadata.split()
        name = name.decode()
        if mode == b'120000' and not name.startswith('.flow/'):
            path = repo / name
            require(kind == b'blob' and path.is_symlink(), 'main CLI tracked symlink changed type or disappeared')
            literal = os.readlink(path)
            original = subprocess.check_output([git, '-C', str(repo), 'cat-file', 'blob', object_id.decode()], env=env)
            require(hashlib.sha1(b'blob ' + str(len(original)).encode() + b'\0' + original).hexdigest() == object_id.decode(),
                    'main CLI Git symlink blob does not match its object identity')
            require(os.fsencode(literal) == original, 'main CLI tracked symlink literal differs from its actual Git HEAD blob')
            links[str(path)] = literal
    actual_links = {str(repo / name) for name in paths if name and not name.startswith('.flow/') and (repo / name).is_symlink()}
    require(actual_links == set(links), 'main CLI source has an uncommitted or missing tracked symlink')
    files = {str(repo / name): sha(repo / name) for name in sorted(paths) if name and not name.startswith('.flow/')
             and (repo / name).is_file() and not (repo / name).is_symlink()}
    require(files, 'main CLI build has no actual regular tracked source inputs')
    text = ''.join(digest + '  ' + name + '\n' for name, digest in sorted(files.items()))
    return files, hashlib.sha256(text.encode()).hexdigest(), head, links


def provision_config(tools_path, gnu_path, sent_epoch):
    tools_path, gnu_path = Path(tools_path), Path(gnu_path)
    tools = read(tools_path)
    repo = Path(tools['repo'])
    require(repo.is_absolute() and repo.resolve() == repo and tools_path.resolve() == tools_path and gnu_path.resolve() == gnu_path,
            'main CLI static tool/checkout bindings are not canonical')
    config = dict(repo=str(repo), python=tools['tools']['python'], sent_epoch=sent_epoch,
                  stages=dict(validate=dict(argv=[tools['tools']['python'], str(HERE / 'static_stages.py'), '--plan', str(tools_path)])),
                  identity_harness=dict(toolchain_plan=str(gnu_path)), _config_path=tools['driver_config'])
    _, envelope = native_gnu.inspect(config, tools_path)
    sources, source_sha256, head, symlinks = source_state(repo, envelope['git'])
    paths = harness.static_inputs(config) | set(sources)
    config.update(files={path: sha(path) for path in sorted(paths)}, source_sha256=source_sha256, head=head, symlinks=symlinks, git=envelope['git'])
    harness.preflight(config)
    return config


def build_command(config):
    _, plan = harness.plan(config)
    test = config['main_role'] == 'counterfactual-harness'
    helper = config['main_role'] == 'conformance-reference'
    target = ['--lib', '--no-run'] if test else ['--example', EXAMPLES[config['main_role']]] if config['main_role'] in EXAMPLES else ['--bin', 'gomad-wasmhost' if helper else 'gomad']
    result = [plan['tools']['cargo'], 'test' if test else 'build', '--release', '--locked', '--offline', '--no-default-features',
            '-p', 'gomad-wasmhost' if helper else 'gomad', *target,
            '--message-format=json', '--manifest-path',
            str(Path(config['repo']) / 'tools/gomad/rust/Cargo.toml')]
    values = ['conformance'] if helper else ROLES[config['main_role']]
    return result + (['--features', ','.join(values)] if values else [])


def identities(config, context):
    kinds = ('helper',) if config['main_role'] == 'conformance-reference' else ('host', 'helper')
    inventories = {kind: native_roles.source_inventory(config, kind) for kind in kinds}
    target = context['bindings']['TARGET']
    result = {kind: native_roles.implementation(files, domain, ROLES[config['main_role']] if kind == 'host' else helper_features(config['main_role']), context['compiler_utf8'], target)
              for kind, domain, (_, files) in ((kind, domain, inventories[kind]) for kind, domain in
                  (('host', b'gomad.host-implementation/v1\0'), ('helper', b'gomad.linked-implementation/v1\0')) if kind in kinds)}
    if kinds == ('helper',): return dict(host=None, **result), None, inventories
    runtime = hashlib.sha256(b'gomad-wasm-runtime-implementation/v1\0')
    for name in native_roles.RUNTIME:
        raw = inventories['host'][1]['../../internal/backends/wasm/toolchain/' + name]
        runtime.update(len(raw).to_bytes(8, 'big') + raw)
    return result, runtime.hexdigest(), inventories


def artifact(config, root, context):
    frames = [native_roles.decode(line) for line in (root / 'cargo-build.stdout').read_text().splitlines() if line]
    finished = [row for row in frames if row.get('reason') == 'build-finished']
    require(len(finished) == 1 and frames[-1] == finished[0] and finished[0]['success'] is True,
            'main CLI Cargo build has no unique successful completion')
    repo = Path(config['repo'])
    crates = {kind: repo / 'tools/gomad/rust' / name for kind, name in (('host', 'gomad'), ('helper', 'wasmhost'))}
    packages = {kind: 'path+' + path.as_uri() + ('#0.1.0' if kind == 'host' else '#' + name + '@0.1.0')
                for kind, path, name in ((kind, crates[kind], name) for kind, name in (('host', 'gomad'), ('helper', 'gomad-wasmhost')))}
    helper = config['main_role'] == 'conformance-reference'
    if helper:
        crates.pop('host'); packages.pop('host')
    selected = 'helper' if helper else 'host'
    example = config['main_role'] in EXAMPLES
    test = config['main_role'] == 'counterfactual-harness'
    name = EXAMPLES[config['main_role']] if example else 'gomad-wasmhost' if helper else 'gomad'
    rows = [row for row in frames if row.get('reason') == 'compiler-artifact' and row.get('package_id') == packages[selected]
            and row.get('target', {}).get('kind') == (['example'] if example else ['lib'] if test else ['bin'])
            and row['target'].get('name') == name and row.get('executable') is not None]
    require(len(rows) == 1, 'main CLI Cargo artifact is missing or duplicated')
    row = rows[0]
    executable = Path(row['executable']) if test else root / ('target/release/examples/' + name if example else 'target/release/' + name)
    if test:
        require(executable.parent == root / 'target/release/deps' and re.fullmatch(r'gomad-[0-9a-f]+', executable.name),
                'counterfactual harness escaped its actual Cargo test target')
    require(row['manifest_path'] == str(crates[selected] / 'Cargo.toml') and row['target']['crate_types'] == (['lib'] if test else ['bin'])
            and row['target']['src_path'] == str(crates[selected] / ('examples/' + name + '.rs' if example else 'src/lib.rs' if test else 'src/main.rs')) and row['features'] == ROLES[config['main_role']]
            and row['profile']['opt_level'] == '3' and row['profile']['debug_assertions'] is False
            and row['profile']['test'] is test and row.get('fresh') is False and row['executable'] == str(executable),
            'main CLI artifact is not its actual ordinary release executable')
    native_roles.regular(executable)
    require(executable.resolve() == executable and os.access(executable, os.X_OK), 'main CLI Cargo artifact is not its owned actual executable')
    identity, runtime, inventories = identities(config, context)
    generated = {}
    for kind, package in packages.items():
        libraries = [item for item in frames if item.get('reason') == 'compiler-artifact' and item.get('package_id') == package
                     and item.get('target', {}).get('kind') == ['lib']
                     and (not (kind == 'host' and test) or item.get('profile', {}).get('test') is True)]
        require(len(libraries) == 1 and libraries[0]['features'] == (ROLES[config['main_role']] if kind == 'host' else helper_features(config['main_role']))
                and libraries[0]['target']['src_path'] == str(crates[kind] / 'src/lib.rs'),
                'main CLI linked library feature/source closure differs')
        scripts = [item for item in frames if item.get('reason') == 'build-script-executed' and item.get('package_id') == package]
        require(len(scripts) == 1, 'main CLI build-script capture is missing or duplicated')
        script = scripts[0]
        directory = Path(script['out_dir'])
        name = 'gomad' if kind == 'host' else 'gomad-wasmhost'
        require(directory.resolve() == directory and directory.parent.parent == root / 'target/release/build'
                and re.fullmatch(name + '-[0-9a-f]+', directory.parent.name) and directory.name == 'out',
                'main CLI build-script capture escaped its owned target')
        expected = {'GOMAD_HOST_IMPLEMENTATION_SHA256': identity['host'], 'GOMAD_RUNTIME_IMPLEMENTATION_SHA256': runtime} if kind == 'host' else {
            'GOMAD_LINKED_IMPLEMENTATION_SHA256': identity['helper']}
        pairs = script['env']
        require(isinstance(pairs, list) and len(pairs) == len(expected)
                and all(isinstance(pair, list) and len(pair) == 2 and all(isinstance(value, str) for value in pair) for pair in pairs)
                and len({pair[0] for pair in pairs}) == len(pairs) and dict(pairs) == expected,
                'main CLI compiled identities differ from actual source/compiler captures')
        texts = {'measurement-source-inputs.tsv': native_roles.source_manifest(inventories[kind][1])}
        if kind == 'helper':
            texts.update({'identity-compiler.txt': context['compiler_utf8'],
                'identity-bindings.tsv': ''.join(key + '\t' + value + '\n' for key, value in sorted(native_roles.feature_bindings(helper_features(config['main_role']), context['bindings']['TARGET']).items()))})
        for name, text in texts.items():
            path = native_roles.regular(directory / name)
            require(path.read_text() == text, 'main CLI generated source/compiler inventory changed')
            generated[str(path)] = sha(path)
    return executable, identity, runtime, generated


def owner_record(root):
    owner = read(root / 'producer.json')
    require(set(owner) == {'pid', 'start_monotonic_nanos', 'end_monotonic_nanos', 'started_epoch_ns'}
            and type(owner['pid']) is int and owner['pid'] > 0
            and all(type(owner[key]) is int and owner[key] > 0 for key in ('start_monotonic_nanos', 'end_monotonic_nanos', 'started_epoch_ns'))
            and owner['start_monotonic_nanos'] < owner['end_monotonic_nanos'], 'main CLI build has no actual producer lifecycle')
    return owner


def evidence(config, root):
    owner = owner_record(root)
    require(config['sent_epoch'] == owner['started_epoch_ns'] // 1_000_000_000, 'main CLI prefreeze build epoch changed')
    context = harness.compiler_context(config, root)
    env = harness.environment(config, root, context['bindings']['TARGET'])
    harness.local_command(config, root, 'cargo-build', build_command(config), env)
    executable, identity, runtime, generated = artifact(config, root, context)
    inputs = dict(config['files'])
    inputs.update(generated)
    pids = set()
    for name in COMMANDS:
        row = read(root / (name + '.command.json'))
        process = read(root / (name + '.process.json'))
        require(type(row.get('exit')) is int and row['exit'] == 0 and type(process.get('exit')) is int and process['exit'] == 0,
                'main CLI command has no actual numeric successful exit')
        require(row['parent_pid'] == owner['pid'] and row['pid'] not in pids
                and owner['start_monotonic_nanos'] <= row['started_monotonic_nanos'] < row['reaped_monotonic_nanos'] <= owner['end_monotonic_nanos'],
                'main CLI raw command borrowed another producer or lifecycle')
        pids.add(row['pid'])
        for suffix in ('command.json', 'process.json', 'stdout', 'stderr'):
            path = root / (name + '.' + suffix)
            inputs[str(path)] = sha(path)
    binary = root / 'admitted' / binary_name(config['main_role'])
    native_roles.regular(binary)
    require(binary.resolve() == binary and binary.stat().st_nlink == 1 and stat.S_IMODE(binary.stat().st_mode) == 0o755
            and sha(binary) == sha(executable), 'main CLI published copy differs from its actual Cargo artifact')
    for path in (binary, executable, root / 'producer.json'):
        inputs[str(path)] = sha(path)
    return dict(schema=SCHEMA, profile='release', exit=0, features=ROLES[config['main_role']], helper_features=helper_features(config['main_role']), role=config['main_role'],
                binary=str(binary), binary_sha256=sha(binary), cargo_artifact=str(executable), host_sha256=identity['host'], helper_sha256=identity['helper'],
                runtime_implementation_sha256=runtime, source_sha256=config['source_sha256'], source_kind='regular-tracked-non-flow/v1',
                symlinks=config['symlinks'], head=config['head'],
                tools_plan=str(harness.plan(config)[0]), gnu_plan=config['identity_harness']['toolchain_plan'],
                root=str(root), command=build_command(config), context=context, generated=generated, inputs=inputs)


def read_build(path, frozen=None, expected_role=None):
    path = Path(path)
    receipt = read(native_roles.regular(path))
    require(isinstance(receipt, dict) and {'schema', 'exit', 'root', 'tools_plan', 'gnu_plan', 'role'} <= set(receipt)
            and receipt.get('schema') == SCHEMA and type(receipt.get('exit')) is int and receipt['exit'] == 0 and path == Path(receipt['root']) / 'main-build.json', 'foreign main CLI build receipt')
    require(receipt.get('role') in ROLES and (expected_role is None or receipt['role'] == expected_role), 'main CLI build role differs from its fixed recipe')
    root = Path(receipt['root'])
    require(root.resolve() == root and root.is_dir() and stat.S_IMODE(root.stat().st_mode) == 0o700
            and root.stat().st_uid == os.geteuid(), 'main CLI build root changed')
    owner = owner_record(root)
    config = provision_config(receipt['tools_plan'], receipt['gnu_plan'], owner['started_epoch_ns'] // 1_000_000_000)
    config['main_role'] = receipt['role']
    expected = evidence(config, root)
    require(receipt == expected, 'main CLI receipt differs from independently reopened source/compiler/Cargo/process evidence')
    check_files(receipt['inputs'])
    if frozen is not None:
        require(frozen.get('head') == receipt['head'] and frozen.get('symlinks') == receipt['symlinks'], 'main CLI Git HEAD or symlink custody differs from the frozen candidate')
        require(frozen['source_sha256'] == receipt['source_sha256'], 'main CLI build differs from the frozen source')
        if not helper_features(receipt['role']):
            require(frozen['identity']['helper'] == receipt['helper_sha256'],
                    'main CLI build differs from the frozen ordinary linked implementation')
        if receipt['role'] == 'release':
            require(frozen['binary'] == receipt['binary'] and frozen['binary_sha256'] == receipt['binary_sha256']
                    and frozen['identity']['host'] == receipt['host_sha256'], 'main CLI build differs from the frozen candidate')
        elif receipt['role'] == 'runtime-harness':
            require(frozen['identity']['host'] == receipt['host_sha256'], 'runtime harness host differs from the frozen ordinary candidate')
            require(all(frozen.get('stages', {}).get(f'runtime-{index}', {}).get('harness') == receipt['binary']
                    and frozen['stages'][f'runtime-{index}'].get('harness_build_receipt') == str(path) for index in range(32)),
                    'runtime harness is not bound to all 32 original shards')
        elif receipt['role'] == 'interception-harness':
            require(frozen['identity']['host'] == receipt['host_sha256'], 'interception harness differs from the ordinary candidate')
            require(frozen.get('interception_streaming') == dict(binary=receipt['binary'], build_receipt=str(path)),
                    'interception harness is not its separately bound role')
        else:
            selected = frozen.get('live_corpus', {}).get('roles', {}).get(receipt['role'])
            require(selected == dict(binary=receipt['binary'], build_receipt=str(path)), 'fixture main CLI is not its separately bound role')
        require(frozen['files'].get(str(path)) == sha(path)
                and all(frozen['files'].get(name) == value for name, value in receipt['inputs'].items()), 'main CLI build custody is incompletely frozen')

    return receipt


def input_paths(path):
    receipt = read_build(path)
    return {str(path)} | set(receipt['inputs'])


def build(tools_path, gnu_path, root, role='release'):
    require(role in ROLES, 'unknown main CLI fixed build role')
    started_epoch_ns = time.time_ns()
    started_monotonic_nanos = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    require(root.is_absolute() and root.resolve() == root and not root.exists(), 'main CLI build root must be fresh and canonical')
    config = provision_config(tools_path, gnu_path, started_epoch_ns // 1_000_000_000)
    config['main_role'] = role
    require(not root.is_relative_to(Path(config['repo'])), 'main CLI outputs must remain outside frozen source')
    root.mkdir(mode=0o700)
    for name in ('home', 'tmp'):(root / name).mkdir(mode=0o700)
    _, tools = harness.plan(config)
    env = harness.environment(config, root)
    for name, argv in (('rustc-version', [tools['tools']['rustc'], '--version']), ('rustc-verbose', [tools['tools']['rustc'], '-vV']),
                       ('cargo-version', [tools['tools']['cargo'], '--version'])):
        command(dict(config, environment=env), root, name, argv, timeout=60, exact_environment=env)
    context = harness.compiler_context(config, root)
    env = harness.environment(config, root, context['bindings']['TARGET'])
    command(dict(config, environment=env), root, 'cargo-build', build_command(config), timeout=3600, exact_environment=env)
    executable, _, _, _ = artifact(config, root, context)
    (root / 'admitted').mkdir(mode=0o700)
    with (root / 'admitted' / binary_name(role)).open('xb') as output:
        os.chmod(output.fileno(), 0o755)
        with executable.open('rb') as source:
            import shutil
            shutil.copyfileobj(source, output)
    check_files(config['files'])
    _, source_digest, head, symlinks = source_state(Path(config['repo']), config['git'])
    require((source_digest, head, symlinks) == (config['source_sha256'], config['head'], config['symlinks']), 'main CLI source or symlink custody changed during its build')
    write(root / 'producer.json', dict(pid=os.getpid(), start_monotonic_nanos=started_monotonic_nanos,
        end_monotonic_nanos=time.clock_gettime_ns(time.CLOCK_MONOTONIC), started_epoch_ns=started_epoch_ns))
    path = root / 'main-build.json'
    write(path, evidence(config, root))
    read_build(path)
    return path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    producer = sub.add_parser('build')
    for name in ('tools-plan', 'gnu-plan', 'root'):producer.add_argument('--' + name, type=Path, required=True)
    producer.add_argument('--role', choices=tuple(ROLES), default='release')
    reader = sub.add_parser('verify');reader.add_argument('--receipt', type=Path, required=True)
    args = parser.parse_args()
    if args.mode == 'build':print(build(args.tools_plan, args.gnu_plan, args.root, args.role))
    else:print(json.dumps(read_build(args.receipt), sort_keys=True))


if __name__ == '__main__':main()
