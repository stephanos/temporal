"""Closed native target catalog and process-bound evidence for broad receiving stages.

Allocation targets use one exact libtest selection in a private stage working directory.
"""
from dataclasses import dataclass
from enum import Enum
from pathlib import Path
import argparse
import json
import os
import re
import stat

from driver import HERE, command, configured, guard, require, sha, write
import native_roles


class Stage(Enum):
    PROVIDER = 'cluster-provider'
    FUNCTIONAL = 'cluster-functional'
    RETAINED = 'retained-cluster'


class Role(Enum):
    HELPER = 'helper-lib'
    HOST = 'host-lib'
    HELPER_ALLOCATION = 'helper-allocation'
    HOST_ALLOCATION = 'host-allocation'
    IDENTITY = 'implementation-identity'


@dataclass(frozen=True)
class Target:
    package: str
    crate: str
    name: str
    source: str
    kind: str
    features: tuple[str, ...]
    harness: bool


TARGETS = {
    Role.HELPER: Target('gomad-wasmhost', 'wasmhost', 'gomad_wasmhost', 'src/lib.rs', 'lib', (), True),
    Role.HOST: Target('gomad', 'gomad', 'gomad', 'src/lib.rs', 'lib', ('determinism-fixture',), True),
    Role.HELPER_ALLOCATION: Target('gomad-wasmhost', 'wasmhost', 'allocation_receiving', 'tests/allocation_receiving.rs', 'test', (), True),
    Role.HOST_ALLOCATION: Target('gomad', 'gomad', 'allocation_receiving', 'tests/allocation_receiving.rs', 'test', ('determinism-fixture',), True),
    Role.IDENTITY: Target('gomad-wasmhost', 'wasmhost', 'implementation_identity', 'tests/implementation_identity.rs', 'test', (), True),
}
ALLOCATION_ROLES = (Role.HELPER_ALLOCATION, Role.HOST_ALLOCATION)
SCHEMA = 'gomad.broad-harness-build/v1'
ADMISSION = 'gomad.broad-harness-admission/v1'


def fixed():
    import identity_harness
    return identity_harness


def owned_root(config):
    return Path(config['root']) / 'broad-harness'


def stage_root(config, stage):
    require(type(stage) is Stage, 'unknown broad stage')
    path = owned_root(config) / stage.value
    require(not path.exists() or path.resolve() == path and path.is_dir(), 'foreign broad stage root')
    return path


def target(role):
    require(type(role) is Role, 'unknown broad executable role')
    return TARGETS[role]


def cargo_command(config, role):
    spec = target(role)
    _, selected = fixed().plan(config)
    argv = [selected['tools']['cargo'], 'test', '--release', '--locked', '--offline', '--no-default-features',
            '-p', spec.package]
    argv += ['--lib'] if spec.kind == 'lib' else ['--test', spec.name]
    if spec.features:
        argv += ['--features', ','.join(spec.features)]
    return argv + ['--no-run', '--message-format=json', '--manifest-path',
                   str(Path(config['repo']) / 'tools/gomad/rust/Cargo.toml')]


@dataclass(frozen=True)
class Invocation:
    name: str
    role: Role
    selector: str = ''
    ignored: bool = False
    exact: bool = False


HOST_SELECTORS = (
    ('host-retained', 'replay::cluster::retained_tests'),
    ('host-presence', 'replay::cluster::retained_presence_tests'),
    ('host-lifecycle-matrix', 'replay::cluster::retained_lifecycle_matrix_tests'),
    ('host-projection', 'replay::cluster::projection_tests'),
    ('host-scenario', 'campaign::scenario::preservation_tests'),
    ('host-fixture-cluster', 'fixture_cluster'),
    ('host-writer', 'evidence::store::publish::write_failure_tests'),
    ('host-inspection', 'replay::inspection::'),
    ('host-stress-reach', 'stress::reach_tests'),
    ('host-stress', 'stress::tests'),
)


def catalog(stage):
    require(type(stage) is Stage, 'unknown broad stage')
    rows = [Invocation('helper-lib', Role.HELPER),
            Invocation('host-inspection-scale', Role.HOST,
                       'replay::inspection::predicate_tests::native_retained_scale_admission_preserves_complete_inspection_and_owner_payloads',
                       ignored=True, exact=True)]
    rows.extend(Invocation(name, Role.HOST, selector) for name, selector in HOST_SELECTORS)
    rows.extend(Invocation(name, Role.HELPER, selector, ignored=True, exact=True) for name, selector in (
        ('helper-world-probe', 'world::owner_preservation_tests::world_owner_codec_measured_cardinality'),
        ('helper-compressed-world-probe', 'world::owner_preservation_tests::compressed_world_owner_measured_cardinality'),
        ('helper-compressed-world-large-probe', 'world::owner_preservation_tests::compressed_world_owner_measured_cardinality')))
    rows.extend((Invocation('helper-allocation', Role.HELPER_ALLOCATION, 'allocation_receiving', ignored=True, exact=True),
                 Invocation('host-allocation', Role.HOST_ALLOCATION, 'allocation_receiving', ignored=True, exact=True), Invocation('identity', Role.IDENTITY)))
    if stage in (Stage.PROVIDER, Stage.RETAINED):
        rows.append(Invocation('host-broad-guest', Role.HOST, 'replay::cluster::broad_guest_tests', True))
    return tuple(rows)


def test_argv(config, stage, name):
    rows = [row for row in catalog(stage) if row.name == name]
    require(len(rows) == 1, 'unknown broad fixed test invocation')
    row = rows[0]
    argv = [str(owned_root(config) / 'admitted' / row.role.value)]
    if target(row.role).harness:
        argv += (['--ignored'] if row.ignored and row.exact else []) + (['--exact'] if row.exact else [])
        argv += ([row.selector] if row.selector else [])
        argv += (['--ignored'] if row.ignored and not row.exact else []) + ['--nocapture', '--test-threads=1']
    return argv


def invocation_context(config, stage, row, evidence):
    require(row in catalog(stage), 'foreign broad invocation')
    destination = stage_root(config, stage)
    environment = fixed().environment(config, owned_root(config), evidence['context']['bindings']['TARGET'])
    cwd = Path(config['repo'])
    if row.role is Role.HELPER or row.name == 'host-inspection-scale':
        environment['GOMAD_INSPECTION_RAW_DIR'] = str(destination / 'helper-lib-raw')
    if row.name == 'helper-world-probe':
        environment['GOMAD_WORLD_CODEC_PROBE'] = '1'
    if row.name in ('helper-compressed-world-probe', 'helper-compressed-world-large-probe'):
        environment.update(GOMAD_COMPRESSED_WORLD_PROBE='1',
                           GOMAD_COMPRESSED_WORLD_PROBE_REQUESTS='200000' if row.name.endswith('-large-probe') else '20000')
    if row.role in ALLOCATION_ROLES:
        cwd = destination / row.name
    if row.name == 'host-broad-guest':
        import terminal_inputs
        terminal_inputs.compiler_inputs(config)
        compiler = config['final_profiles']['stock_go']
        stock_root = Path(compiler).parent.parent
        require(Path(compiler).parent.name == 'bin', 'broad guest requires its explicit stock compiler root')
        for path in fixed().tree(stock_root):
            native_roles.pinned(config, path)
        environment.update(GOMAD_STOCK_GO=compiler,
                           GOMAD_TEST_BROAD_GUEST_ROOT=str(destination / 'guest-root'))
    require(not cwd.exists() or cwd.resolve() == cwd and cwd.is_dir(), 'foreign broad test working directory')
    return dict(config, repo=str(cwd)), environment


def source_inputs(config):
    files = set(fixed().preflight(config))
    for kind in ('helper', 'host'):
        crate, sources = native_roles.source_inventory(config, kind)
        files.update(str((crate / name).resolve()) for name in sources)
    rust = Path(config['repo']) / 'tools/gomad/rust'
    for name in ('allocation_probe.rs', 'wasmhost/tests/allocation_receiving.rs',
                 'gomad/tests/allocation_receiving.rs', 'wasmhost/tests/implementation_identity.rs'):
        files.add(str(rust / name))
    for path in files:
        native_roles.pinned(config, path)
    return files


def artifact(config, root, role, frames):
    spec = target(role)
    crate = Path(config['repo']) / 'tools/gomad/rust' / spec.crate
    package_id = 'path+' + crate.as_uri() + '#' + spec.package + '@0.1.0'
    artifacts = [frame for frame in frames if frame.get('reason') == 'compiler-artifact'
                 and frame.get('package_id') == package_id and frame.get('target', {}).get('name') == spec.name
                 and frame.get('executable') is not None]
    finished = [frame for frame in frames if frame.get('reason') == 'build-finished']
    scripts = [frame for frame in frames if frame.get('reason') == 'build-script-executed' and frame.get('package_id') == package_id]
    require(len(artifacts) == len(finished) == len(scripts) == 1 and frames[-1] == finished[0]
            and finished[0]['success'] is True, 'broad Cargo build lacks one exact completed artifact and source capture')
    frame = artifacts[0]
    metadata = frame['target']
    executable = Path(frame['executable'])
    require(metadata['kind'] == [spec.kind] and metadata['crate_types'] == (['lib'] if spec.kind == 'lib' else ['bin'])
            and metadata['src_path'] == str(crate / spec.source) and metadata['test'] is True
            and frame['manifest_path'] == str(crate / 'Cargo.toml') and frame['features'] == list(spec.features)
            and frame['profile']['opt_level'] == '3' and frame['profile']['debug_assertions'] is False
            and frame['profile']['test'] is True and frame.get('fresh') is False
            and executable.parent == root / 'target/release/deps'
            and re.fullmatch(re.escape(spec.name) + r'-[0-9a-f]+', executable.name),
            'broad Cargo artifact has foreign source, role, features, profile or output root')
    native_roles.regular(executable)
    require(executable.resolve() == executable and executable.stat().st_mode & 0o111,
            'broad Cargo artifact is not an actual owned executable')
    generated = Path(scripts[0]['out_dir'])
    require(generated.parent.parent == root / 'target/release/build' and generated.name == 'out'
            and re.fullmatch(re.escape(spec.package) + '-[0-9a-f]+', generated.parent.name)
            and generated.resolve() == generated, 'broad build script borrowed another output root')
    return executable, generated


def captures(config, generated, role, context):
    spec = target(role)
    _, sources = native_roles.source_inventory(config, 'helper' if spec.crate == 'wasmhost' else 'host')
    expected = {'measurement-source-inputs.tsv': native_roles.source_manifest(sources)}
    if spec.crate == 'wasmhost':
        bindings = dict(context['bindings'])
        bindings.update({'CARGO_FEATURE_' + name.replace('-', '_').upper(): '1' for name in spec.features})
        expected.update({'identity-bindings.tsv': ''.join(name + '\t' + value + '\n' for name, value in sorted(bindings.items())),
                         'identity-compiler.txt': context['compiler_utf8']})
    for name, value in expected.items():
        path = native_roles.regular(generated / name)
        require(path.resolve() == path and path.read_text() == value, 'broad generated source/compiler capture differs from frozen inputs')
    return {name: sha(generated / name) for name in expected}


def verify_command(config, root, name, argv, environment, timeout=3600):
    import receiving
    receiving.Retained(config).command(config, root, name, argv, timeout=timeout)
    return fixed().local_command(config, root, name, argv, environment)


def command_hashes(root, name):
    return {suffix: sha(root / (name + '.' + suffix)) for suffix in ('command.json', 'process.json', 'stdout', 'stderr')}


def build_evidence(config):
    root = owned_root(config)
    inputs = source_inputs(config)
    context = fixed().compiler_context(config, root)
    environment = fixed().environment(config, root, context['bindings']['TARGET'])
    roles = {}
    pids = set()
    _, selected = fixed().plan(config)
    for name, argv in (('rustc-version', [selected['tools']['rustc'], '--version']),
                       ('rustc-verbose', [selected['tools']['rustc'], '-vV']),
                       ('cargo-version', [selected['tools']['cargo'], '--version'])):
        receipt = verify_command(config, root, name, argv, fixed().environment(config, root), timeout=60)
        pid = receipt['pid']
        require(pid not in pids, 'broad build reused a compiler subprocess PID')
        pids.add(pid)
    for role in Role:
        name = 'cargo-' + role.value
        receipt = verify_command(config, root, name, cargo_command(config, role), environment)
        require(receipt['pid'] not in pids, 'broad build reused a subprocess PID')
        pids.add(receipt['pid'])
        frames = [native_roles.decode(line) for line in (root / (name + '.stdout')).read_text().splitlines() if line]
        executable, generated = artifact(config, root, role, frames)
        roles[role.value] = dict(executable=str(executable), executable_sha256=sha(executable),
                                 generated=captures(config, generated, role, context), command=cargo_command(config, role),
                                 captures=command_hashes(root, name))
    return dict(schema=SCHEMA, source_sha256=config['source_sha256'], config_sha256=config['_config_sha256'],
                inputs={path: config['files'][path] for path in sorted(inputs)}, context=context, roles=roles,
                compiler={name: command_hashes(root, name) for name in ('rustc-version', 'rustc-verbose', 'cargo-version')})


def admission(config, build):
    root = owned_root(config)
    roles = {}
    for role in Role:
        source = build['roles'][role.value]
        path = native_roles.regular(root / 'admitted' / role.value)
        require(path.resolve() == path and path.stat().st_nlink == 1 and stat.S_IMODE(path.stat().st_mode) == 0o755
                and sha(path) == source['executable_sha256'] and str(path) != source['executable'],
                'broad derived executable lost its Cargo artifact custody')
        roles[role.value] = dict(cargo_artifact=source['executable'], cargo_artifact_sha256=source['executable_sha256'],
                                 executable=str(path), executable_sha256=sha(path))
    expected = dict(schema=ADMISSION, build_receipt=str(root / 'build-receipt.json'),
                    build_receipt_sha256=sha(root / 'build-receipt.json'), roles=roles)
    require(native_roles.read(root / 'admission.json') == expected, 'broad executable admission changed')
    return roles


def producer_argv(config, mode, stage=None):
    require(mode in ('build', 'run'), 'unknown broad producer mode')
    require((mode == 'run') == (type(stage) is Stage), 'broad producer requires its fixed stage')
    argv = [config['python'], str(HERE / 'broad_harness.py'), mode, '--config', config['_config_path']]
    return argv + (['--stage', stage.value] if stage else [])


def producer(config, root, name, argv, environment):
    receipt = verify_command(config, root, name, argv, environment)
    return receipt, native_roles.read(root / (name + '.process.json'))


def read_build(config):
    guard(config)
    root = owned_root(config)
    require(root.resolve() == root and root.is_dir(), 'foreign broad harness root')
    evidence = build_evidence(config)
    require(native_roles.read(root / 'build-receipt.json') == evidence, 'broad build receipt differs from reopened raw build')
    admission(config, evidence)
    outer, process = producer(config, root, 'build-producer', producer_argv(config, 'build'), fixed().environment(config, root))
    from primary_stages import enclose_child
    for name in ('rustc-version', 'rustc-verbose', 'cargo-version', *('cargo-' + role.value for role in Role)):
        enclose_child(outer, process, native_roles.read(root / (name + '.command.json')))
    guard(config)
    return evidence


def build(config):
    guard(config)
    source_inputs(config)
    root = owned_root(config)
    require(root.is_dir() and root.resolve() == root, 'foreign broad build root')
    for name in ('home', 'tmp'):
        (root / name).mkdir(mode=0o700)
    require(not (root / 'target').exists(), 'broad target must be fresh and isolated')
    _, selected = fixed().plan(config)
    environment = fixed().environment(config, root)
    for name, argv in (('rustc-version', [selected['tools']['rustc'], '--version']),
                       ('rustc-verbose', [selected['tools']['rustc'], '-vV']),
                       ('cargo-version', [selected['tools']['cargo'], '--version'])):
        command(dict(config, environment=environment), root, name, argv, timeout=60, exact_environment=environment)
    context = fixed().compiler_context(config, root)
    environment = fixed().environment(config, root, context['bindings']['TARGET'])
    for role in Role:
        command(dict(config, environment=environment), root, 'cargo-' + role.value, cargo_command(config, role),
                timeout=3600, exact_environment=environment)
    evidence = build_evidence(config)
    write(root / 'build-receipt.json', evidence)
    (root / 'admitted').mkdir(mode=0o700)
    roles = {}
    for role in Role:
        source = evidence['roles'][role.value]
        path = root / 'admitted' / role.value
        with path.open('xb') as output:
            os.chmod(path, 0o755)
            output.write(Path(source['executable']).read_bytes())
        roles[role.value] = dict(cargo_artifact=source['executable'], cargo_artifact_sha256=source['executable_sha256'],
                                 executable=str(path), executable_sha256=sha(path))
    write(root / 'admission.json', dict(schema=ADMISSION, build_receipt=str(root / 'build-receipt.json'),
          build_receipt_sha256=sha(root / 'build-receipt.json'), roles=roles))
    admission(config, evidence)
    guard(config)
    return evidence


def run(config, stage):
    destination = stage_root(config, stage)
    evidence = read_build(config)
    require(destination.is_dir(), 'broad stage root must be owned by its producer boundary')
    files = dict(config['files'])
    files.update({str(owned_root(config) / 'admitted' / role.value): evidence['roles'][role.value]['executable_sha256'] for role in Role})
    for row in catalog(stage):
        observed, environment = invocation_context(config, stage, row, evidence)
        if row.name == 'helper-lib':
            Path(environment['GOMAD_INSPECTION_RAW_DIR']).mkdir(mode=0o700)
        if row.role in ALLOCATION_ROLES:
            Path(observed['repo']).mkdir(mode=0o700)
        if row.name == 'host-broad-guest':
            require(not (destination / 'guest-root').exists(), 'broad guest root must be fresh')
            (destination / 'guest-root').mkdir(mode=0o700)
        command(dict(observed, files=files, environment=environment), destination, row.name,
                test_argv(config, stage, row.name), timeout=3600, exact_environment=environment)
    guard(config)


def read(config, stage):
    destination = stage_root(config, stage)
    evidence = read_build(config)
    rows = {}
    pids = set()
    outer, process = producer(config, destination, 'run-producer', producer_argv(config, 'run', stage),
                               fixed().environment(config, owned_root(config)))
    from primary_stages import enclose_child
    for candidate in Stage:
        for row in catalog(candidate):
            path = stage_root(config, candidate) / (row.name + '.command.json')
            if not path.exists():
                continue
            observed, environment = invocation_context(config, candidate, row, evidence)
            files = dict(config['files'])
            files.update({str(owned_root(config) / 'admitted' / role.value): evidence['roles'][role.value]['executable_sha256'] for role in Role})
            receipt = verify_command(dict(observed, files=files, environment=environment), path.parent, row.name,
                                     test_argv(config, candidate, row.name), environment)
            require(receipt['pid'] not in pids, 'broad stages reused a subprocess PID')
            pids.add(receipt['pid'])
            if candidate is stage:
                enclose_child(outer, process, receipt)
                rows[row.name] = dict(role=row.role.value, cwd=observed['repo'], stdout=str(destination / (row.name + '.stdout')),
                                      stderr=str(destination / (row.name + '.stderr')), pid=receipt['pid'],
                                      argv=test_argv(config, candidate, row.name), source=target(row.role).source,
                                      features=list(target(row.role).features), ignored=row.ignored, exact=row.exact, harness=target(row.role).harness, command=command_hashes(destination, row.name),
                                      build=evidence['roles'][row.role.value])
                if row.role is Role.HELPER or row.name == 'host-inspection-scale':
                    artifact_root = Path(environment['GOMAD_INSPECTION_RAW_DIR'])
                    require(artifact_root.resolve() == artifact_root and artifact_root.is_dir(),
                            'broad helper inspection artifact root is foreign')
                    rows[row.name]['raw_artifact_root'] = str(artifact_root)
    require(set(rows) == {row.name for row in catalog(stage)}, 'broad stage lacks a fixed native process')
    guard(config)
    return dict(schema='gomad.broad-harness-raw/v1', stage=stage.value, source_sha256=config['source_sha256'],
                config_sha256=config['_config_sha256'], build_receipt_sha256=sha(owned_root(config) / 'build-receipt.json'),
                admission_sha256=sha(owned_root(config) / 'admission.json'), roles=rows)


def execute_build(config):
    guard(config)
    source_inputs(config)
    root = owned_root(config)
    root.mkdir(mode=0o700)
    environment = fixed().environment(config, root)
    command(dict(config, environment=environment), root, 'build-producer', producer_argv(config, 'build'),
            timeout=3600, exact_environment=environment)
    return read_build(config)


def execute(config, stage):
    read_build(config)
    destination = stage_root(config, stage)
    destination.mkdir(mode=0o700)
    environment = fixed().environment(config, owned_root(config))
    command(dict(config, environment=environment), destination, 'run-producer', producer_argv(config, 'run', stage),
            timeout=3600, exact_environment=environment)
    return read(config, stage)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('build', 'run', 'read'))
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--stage', choices=tuple(stage.value for stage in Stage))
    args = parser.parse_args()
    require((args.mode != 'build') == (args.stage is not None), 'broad stage is required only for run/read')
    config = configured(args.config)
    config.update(_config_path=str(args.config.resolve()), _config_sha256=sha(args.config))
    if args.mode == 'build':
        build(config)
    elif args.mode == 'run':
        run(config, Stage(args.stage))
    else:
        print(json.dumps(read(config, Stage(args.stage)), sort_keys=True))


if __name__ == '__main__':
    main()
