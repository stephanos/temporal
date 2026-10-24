#!/usr/bin/env python3
"""Actual installed typed-scenario detached replay and kill/resume families.

Runs the unchanged installed-lifecycle-check families typed-detached, kill-12
and kill-24 against the frozen release CLI in a fresh private root, then
rereads the retained raw evidence.
"""
import argparse
import os
from pathlib import Path
import shutil
import time

from driver import command, read, require, sha, write
import installed_cluster_cache
import installed_cluster_failure as typed
import live

LIFECYCLE = typed.LIFECYCLE
MODULE = typed.MODULE
SCHEMA = 'gomad.installed-lifecycle-families/v1'
# Original roots executed by each unchanged lifecycle family.
FAMILIES = {'typed-detached': 'TestInstalledOperatorTypedScenarioDetachedReplay',
            'kill-12': 'TestInstalledOperatorKillResume',
            'kill-24': 'TestInstalledOperatorTypedScenarioKillResume'}
PRIVATE = ('source', 'inputs', 'compiler', 'guest-config.json', 'cache', 'runtime')


def source_files(family):
    if family == 'kill-12':
        declaration = read(MODULE / 'testdata/parity/installed-lifecycle-corpus.json')
        return {name: raw.encode() for name, raw in declaration['kill_fixture'].items()}
    return {'go.mod': b'module gomad.wasm.typedoperator\n\ngo 1.27.1\n',
            'main.go': (typed.FIXTURES / 'main.go').read_bytes()}


def lifecycle_config(root, build):
    return dict(repo=str(MODULE.parents[1]), environment=typed.environment(root), binary=build['binary'],
                installation=str(root / 'installation.json'), source=str(root / 'source'), private_root=str(root),
                detach=[str(root / name) for name in PRIVATE])


def install_argv(root, build):
    return [build['binary'], 'install', '--output', str(root / 'installation.json'),
            '--compiler', str(root / 'compiler/bin/go'), '--cache', str(root / 'cache'),
            '--compile-cache', str(root / 'engine-cache'), '--profile', 'gomad.wasi-cooperative/v1',
            '--runtime-root', str(root / 'runtime'), '--config', str(root / 'guest-config.json'),
            '--fuel', '4000000000', '--memory-bytes', str(256 << 20)]


def admitted(root, recipe, build, family):
    """The private inputs are the original fixtures and the installation binds the frozen candidate."""
    require(family in FAMILIES, 'unreviewed installed lifecycle family')
    require({name: (root / 'source' / name).read_bytes() for name in source_files(family)} == source_files(family)
            and sorted(path.name for path in (root / 'source').iterdir()) == sorted(source_files(family)),
            'original lifecycle source fixture changed')
    require((root / 'inputs/control').read_bytes() == b'retained typed input', 'original captured source changed')
    guest = read(root / 'guest-config.json')
    require(sha(root / 'guest-config.json') == recipe['guest_config_sha256'], 'private guest configuration changed')
    LIFECYCLE.guest_base(guest, 'gomad.wasi-cooperative/v1', True)
    LIFECYCLE.captured_input(guest, b'retained typed input')
    require(live.inventory(root / 'compiler') == recipe['compiler_inventory'], 'private stock compiler changed')
    require(installed_cluster_cache.verify(root / 'module-cache', MODULE / 'go.sum') == recipe['module_cache_inventory'],
            'private genuine dependency cache changed')
    for name in ('gomad_semantics.go.txt', 'gomad_choicewire_generated.go.txt'):
        require((root / 'runtime' / name).read_bytes() == (MODULE / 'internal/model/runtimesource' / name).read_bytes(),
                'original cooperative runtime source changed')
    require(build['features'] == [] and build['helper_features'] == [] and build['role'] == 'release',
            'installed lifecycle requires the ordinary installed release CLI')
    installed = read(root / 'installation.json')
    require(installed['host_sha256'] == build['host_sha256'] and installed['helper_sha256'] == build['helper_sha256']
            and installed['runtime_implementation_sha256'] == build['runtime_implementation_sha256']
            and installed['admission']['guest_configuration'] == guest
            and installed['compiler_path'] == str(root / 'compiler/bin/go')
            and installed['runtime_root'] == str(root / 'runtime') and installed['cache_root'] == str(root / 'cache')
            and installed['fuel'] == 4000000000 and installed['memory_bytes'] == 256 << 20,
            'installation changed actual candidate/configuration/execution bounds')


def run_family(root, family, build):
    config = lifecycle_config(root, build)
    command(config, root, 'install', install_argv(root, build))
    work = root / 'work'
    work.mkdir(mode=0o700)
    run = LIFECYCLE.Run(config, work)
    LIFECYCLE.discovery(LIFECYCLE.check.parity.decode(run.command('discover', ['discover'])[0]))
    if family == 'typed-detached':
        run.detached()
    else:
        LIFECYCLE.kill_resume(run, int(family.split('-')[1]))
    return run.index


def produce(main_build, compiler_root, guest_config, module_cache, root, family):
    import main_cli_build
    build = main_cli_build.read_build(main_build, expected_role='release')
    require(family in FAMILIES, 'unreviewed installed lifecycle family')
    require(root.is_absolute() and root.resolve() == root and not root.exists(), 'lifecycle output root must be fresh/canonical')
    require(compiler_root.is_absolute() and compiler_root.is_dir() and not compiler_root.is_symlink(), 'stock compiler root required')
    require(not root.is_relative_to(MODULE.parents[1]), 'lifecycle evidence must remain outside frozen checkout')
    root.mkdir(mode=0o700)
    write(root / 'producer.json', dict(pid=os.getpid(), start_ns=time.time_ns(), started_monotonic_nanos=time.monotonic_ns()))
    for name in ('home', 'tmp', 'source', 'inputs', 'runtime'):
        (root / name).mkdir(mode=0o700)
    shutil.copytree(compiler_root, root / 'compiler', symlinks=False)
    for name, raw in source_files(family).items():
        (root / 'source' / name).write_bytes(raw)
    (root / 'inputs/control').write_bytes(b'retained typed input')
    for name in ('gomad_semantics.go.txt', 'gomad_choicewire_generated.go.txt'):
        shutil.copyfile(MODULE / 'internal/model/runtimesource' / name, root / 'runtime' / name)
    shutil.copyfile(guest_config, root / 'guest-config.json')
    recipe = dict(family=family, main_build=str(main_build), guest_config=str(guest_config),
                  guest_config_sha256=sha(guest_config), compiler_inventory=live.inventory(compiler_root),
                  module_cache_inventory=installed_cluster_cache.snapshot(module_cache, root / 'module-cache', MODULE / 'go.sum'),
                  repo=str(MODULE.parents[1]))
    write(root / 'recipe.json', recipe)
    commands = run_family(root, family, build)
    admitted(root, recipe, build, family)
    write(root / 'result.json', dict(schema=SCHEMA, family=family, root=FAMILIES[family], commands=commands,
                                     files=tree_without_result(root)))
    read_result(root / 'result.json', main_build, family)
    return root / 'result.json'


def read_result(path, expected_build=None, expected_family=None):
    import main_cli_build
    result = read(path)
    root = path.parent
    recipe = read(root / 'recipe.json')
    require(result['schema'] == SCHEMA and result['family'] == recipe['family']
            and (expected_family is None or result['family'] == expected_family)
            and result['root'] == FAMILIES[result['family']], 'foreign installed lifecycle result')
    require(expected_build is None or recipe['main_build'] == str(expected_build), 'foreign lifecycle main CLI build')
    require(recipe['repo'] == str(MODULE.parents[1]), 'lifecycle checkout differs from current source')
    build = main_cli_build.read_build(Path(recipe['main_build']), expected_role='release')
    require(result['files'] == tree_without_result(root), 'lifecycle raw evidence changed after its producer')
    install = read(root / 'install.command.json')
    require(install['argv'] == install_argv(root, build) and install['exit'] == 0, 'foreign lifecycle installation command')
    receipts = sorted((root / 'work').glob('[0-9][0-9][0-9][0-9]-*.command.json'))
    require(len(receipts) == result['commands'] and receipts and receipts[0].name == '0001-discover.command.json',
            'missing/extra installed lifecycle commands')
    pids = {install['pid']}
    for receipt in receipts:
        row = read(receipt)
        require(row['argv'][0] == build['binary'] and row['pid'] not in pids, 'foreign or reused lifecycle CLI process')
        pids.add(row['pid'])
    if result['family'] == 'typed-detached':
        require(read(root / 'work/detached.json')[:len(PRIVATE)] == [str(root / name) for name in PRIVATE],
                'typed detached replay ran with build inputs available')
    return result


def tree_without_result(root):
    return {name: entry for name, entry in LIFECYCLE.tree(root).items() if name != 'result.json'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    producer = sub.add_parser('produce')
    for name in ('main-build', 'compiler-root', 'guest-config', 'module-cache', 'root'):
        producer.add_argument('--' + name, type=Path, required=True)
    producer.add_argument('--family', choices=FAMILIES, required=True)
    reader = sub.add_parser('verify')
    reader.add_argument('--receipt', type=Path, required=True)
    for entry in (producer, reader):
        entry.add_argument('--config', type=Path)
    args = parser.parse_args()
    if args.config is not None:
        from driver import configured
        configured(args.config)
    if args.mode == 'produce':
        print(produce(args.main_build, args.compiler_root, args.guest_config, args.module_cache, args.root, args.family))
    else:
        read_result(args.receipt)
        print('installed lifecycle family raw evidence verified')


if __name__ == '__main__':
    main()
