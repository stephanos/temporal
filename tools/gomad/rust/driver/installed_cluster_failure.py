#!/usr/bin/env python3
"""Actual installed original deadlock/capacity scenarios and independent raw rereading."""
import argparse
import os
from pathlib import Path
import shutil
import sys
import time

from driver import HERE, command, read, require, sha, write
import live
import receiving
import preparation
import installed_cluster_cache

LIFECYCLE = live.load('installed-lifecycle-check')
CHECK = LIFECYCLE.check
CANONICAL = CHECK.parity.canonical
MODULE = HERE.parents[1]
FIXTURES = MODULE / 'internal/backends/wasm/command/testdata/scenario'
SCHEMA = 'gomad.installed-cluster-failure/v1'


def explore_arguments(root, case):
    require(case in ('deadlock', 'capacity'), 'unreviewed installed failure scenario')
    return ['explore', '--json', '--scenario', str(root / (case + '.json')), '--seeds=7',
            '--parallel=1', '--on-failure=all', '--keep-successes=all', '--success-limit=32',
            '--success-bytes=536870912', '--execution-timeout=3m', '--overall-timeout=4m',
            '--terminate-grace=100ms', '--artifacts', str(root / (case + '-artifacts')),
            '--working-dir', str(root / 'source'), 'go-run', '.']


def prompt_interval(row, wall):
    fields = ('start_ns', 'end_ns', 'started_monotonic_nanos', 'reaped_monotonic_nanos')
    require(set(wall) == set(fields) and all(type(wall[key]) is int for key in fields),
            'missing actual outer explore wall interval')
    require(wall['start_ns'] <= row['start_ns'] <= row['end_ns'] <= wall['end_ns']
            and wall['started_monotonic_nanos'] <= row['started_monotonic_nanos']
            <= row['reaped_monotonic_nanos'] <= wall['reaped_monotonic_nanos'],
            'outer explore interval excludes preparation or command completion')
    require(0 <= wall['reaped_monotonic_nanos'] - wall['started_monotonic_nanos'] <= 10_000_000_000,
            'failure exceeded original 10-second wall bound including preparation')


def command_identity(row, process, owner, argv, repo, status, environment, seen):
    require(row['argv'] == argv and row['cwd'] == repo and type(row['pid']) is int
            and row['pid'] > 0 and row['pid'] not in seen and row['parent_pid'] == owner['pid']
            and type(row['exit']) is int and row['exit'] == status and row['observation'] == 'completed'
            and row['clock'] == 'CLOCK_MONOTONIC' and row['environment_mode'] == 'exact'
            and row['effective_environment'] == environment, 'foreign or incomplete installed CLI command')
    require(process['stage_pid'] == owner['pid'] and process['active'] is False
            and not process.get('cleanup_signal'),
            'installed command is not completed by actual producer')
    for field in ('argv', 'pid', 'parent_pid', 'start_ns', 'exit', 'observation', 'started_monotonic_nanos', 'reaped_monotonic_nanos'):
        require(process[field] == row[field], 'installed process/command mismatch: ' + field)
    require(type(owner['pid']) is int and owner['pid'] > 0
            and all(type(row[key]) is int for key in ('start_ns', 'end_ns', 'started_monotonic_nanos', 'reaped_monotonic_nanos'))
            and owner['start_ns'] <= row['start_ns'] <= process['end_ns'] <= row['end_ns']
            and owner['started_monotonic_nanos'] <= row['started_monotonic_nanos'] < row['reaped_monotonic_nanos'],
            'installed command outside actual producer lifecycle')
    seen.add(row['pid'])


def environment(root):
    return dict(PATH='/usr/bin:/bin', HOME=str(root / 'home'), TMPDIR=str(root / 'tmp'),
                LC_ALL='C', TZ='UTC', GOENV='off', GOTOOLCHAIN='local', GOWORK='off',
                GOFLAGS='', CGO_ENABLED='0', GOMODCACHE=str(root / 'module-cache'))


def source_manifest(build, repo):
    files = {name: value for name, value in build['inputs'].items()
             if Path(name).is_relative_to(repo) and not Path(name).relative_to(repo).parts[0] == '.flow'}
    raw = ''.join(value + '  ' + name + '\n' for name, value in sorted(files.items())).encode()
    require(files and receiving.digest(raw) == build['source_sha256'],
            'main CLI original source manifest cannot be independently reconstructed')
    return raw


def inspection_totals(report, streams):
    artifact = report['artifact']
    for name in ('stdout', 'stderr'):
        total = sum(len(raw) for (_, _, stream), raw in streams.items() if stream == name)
        require(type(artifact[name]['total_bytes']) is int and artifact[name]['total_bytes'] == total,
                'top-level inspection ' + name + ' total differs from original retained cluster outputs')


def original_inputs(root, recipe, build):
    require((root / 'source-inputs.tsv').read_bytes() == source_manifest(build, Path(recipe['repo'])),
            'actual main CLI source manifest changed')
    declaration = read(MODULE / 'testdata/parity/installed-lifecycle-corpus.json')
    for name in ('main.go', 'config.json', 'deadlock.json', 'capacity.json'):
        path = FIXTURES / name
        original = 'tools/gomad/' + str(path.relative_to(MODULE))
        require(dict(bytes=path.stat().st_size, sha256=sha(path)) == declaration['fixtures'][original],
                'original operator fixture changed: ' + name)
    LIFECYCLE.source_inputs(dict(source=str(root / 'source')), 'prompt')
    for case in ('deadlock', 'capacity'):
        require(read(root / (case + '.json')) == read(FIXTURES / (case + '.json')),
                'original scenario actions/topology/limits changed')
    guest = read(root / 'guest-config.json')
    LIFECYCLE.guest_base(guest, 'gomad.wasi-cooperative/v1', True)
    LIFECYCLE.captured_input(guest, b'retained typed input')
    require((root / 'inputs/control').read_bytes() == b'retained typed input', 'original captured source changed')
    require(live.inventory(root / 'compiler') == recipe['compiler_inventory'], 'private stock compiler changed')
    require(recipe['compiler_inventory'] == live.inventory(Path(recipe['compiler_root'])),
            'private stock compiler differs from admitted original compiler')
    require(installed_cluster_cache.verify(root / 'module-cache', MODULE / 'go.sum') == recipe['module_cache_inventory']
            and live.inventory(root / 'module-cache') == recipe['module_cache_inventory'],
            'private genuine dependency cache changed')
    require(sha(root / 'guest-config.json') == sha(recipe['guest_config']), 'private guest configuration changed')
    for name in ('gomad_semantics.go.txt', 'gomad_choicewire_generated.go.txt'):
        require((root / 'runtime' / name).read_bytes() == (MODULE / 'internal/model/runtimesource' / name).read_bytes(),
                'original cooperative runtime source changed')
    require(build['features'] == [] and build['helper_features'] == [] and build['role'] == 'release',
            'operator acceptance requires ordinary installed release CLI')


class Run:
    def __init__(self, root, recipe, build, reread):
        self.root, self.recipe, self.build, self.reread = root, recipe, build, reread
        self.names, self.seen = [], set()
        files = dict(build['inputs'])
        for name in ('compiler', 'source', 'runtime', 'module-cache'):
            for relative, entry in live.inventory(root / name).items():
                files[str(root / name / relative)] = entry['sha256']
        files[str(root / 'source-inputs.tsv')] = build['source_sha256']
        self.config = dict(source_manifest=str(root / 'source-inputs.tsv'), repo=recipe['repo'], environment={}, root=str(root), files=files,
            binary=build['binary'], binary_sha256=build['binary_sha256'], source_sha256=build['source_sha256'],
            identity=dict(host=build['host_sha256'], helper=build['helper_sha256']))
        self.receiver = receiving.Retained(self.config)
        self.owner = read(root / 'producer.json')

    def call(self, name, arguments, status=0, installed=False):
        argv = [self.build['binary'], *(['--installation', str(self.root / 'installation.json')] if installed else []),
                *map(str, arguments)]
        if not self.reread:
            before = dict(start_ns=time.time_ns(), started_monotonic_nanos=time.monotonic_ns())
            command(self.config, self.root, name, argv, accepted=(status,), timeout=180,
                    exact_environment=environment(self.root),
                    private_observations=self.config if name in ('deadlock', 'capacity') else None)
            if name in ('deadlock', 'capacity'):
                write(self.root / (name + '.wall.json'), dict(before, end_ns=time.time_ns(),
                      reaped_monotonic_nanos=time.monotonic_ns()))
        row = read(receiving.regular(self.root / (name + '.command.json')))
        process = read(receiving.regular(self.root / (name + '.process.json')))
        command_identity(row, process, self.owner, argv, self.recipe['repo'], status, environment(self.root), self.seen)
        require(row['end_ns'] - row['start_ns'] <= 180_000_000_000, 'original operator command watchdog exceeded')
        if name in ('deadlock', 'capacity'):
            wall = read(self.root / (name + '.wall.json'))
            prompt_interval(row, wall)
            require('private_observations' in row, 'actual original preparation observation is missing')
            self.receiver.capture_preparations(row)
        self.names.append(name)
        return receiving.regular(self.root / (name + '.stdout')).read_bytes()


def evaluate(root, recipe, build, reread):
    original_inputs(root, recipe, build)
    run = Run(root, recipe, build, reread)
    run.call('install', ['install', '--output', root / 'installation.json', '--compiler', root / 'compiler/bin/go',
        '--cache', root / 'cache', '--compile-cache', root / 'engine-cache', '--profile', 'gomad.wasi-cooperative/v1',
        '--runtime-root', root / 'runtime', '--config', root / 'guest-config.json', '--fuel', '4000000000',
        '--memory-bytes', str(256 << 20)])
    installed = read(root / 'installation.json')
    require(installed['host_sha256'] == build['host_sha256'] and installed['helper_sha256'] == build['helper_sha256']
            and installed['runtime_implementation_sha256'] == build['runtime_implementation_sha256']
            and installed['admission']['guest_configuration'] == read(root / 'guest-config.json')
            and installed['compiler_path'] == str(root / 'compiler/bin/go')
            and installed['compiler_sha256'] == sha(root / 'compiler/bin/go')
            and installed['runtime_root'] == str(root / 'runtime') and installed['cache_root'] == str(root / 'cache')
            and installed['fuel'] == 4000000000 and installed['memory_bytes'] == 256 << 20,
            'installation changed actual candidate/configuration/execution bounds')
    for name, expected in installed['admission']['compiler_inputs'].items():
        path = Path(name)
        if name not in run.config['files']:
            require(path.is_relative_to(root / 'cache/runtime-overlay') and sha(receiving.regular(path)) == expected,
                    'installation admitted an unowned generated compiler input')
            run.config['files'][name] = expected
        require(run.config['files'][name] == expected, 'installation compiler inventory differs from original input')
    LIFECYCLE.discovery(CHECK.parity.decode(run.call('discover', ['discover'], installed=True)))
    artifacts = {}
    for case in ('deadlock', 'capacity'):
        output = run.call(case, explore_arguments(root, case), 3, installed=True)
        wall = read(root / (case + '.wall.json'))
        LIFECYCLE.prompt(output, wall['reaped_monotonic_nanos'] - wall['started_monotonic_nanos'], case)
        campaigns_observed = [value for key, value in run.receiver.preparations.items()
                              if Path(key).is_relative_to(root / (case + '-artifacts'))]
        require(len(campaigns_observed) == 1, 'original explore lacks unique actual preparation')
        observed = campaigns_observed[0]
        provenance = observed['provenance']
        require(provenance['inputs']['kind'] == 'go-run'
                and provenance['inputs']['directory'] == str(root / 'source'),
                'original Go guest was not prepared from private original source')
        preparation.Inventory(run.config).project(observed['sidecar'], installed, provenance['inputs'],
                                                   owner_pid=observed['receipt']['owner_pid'])
        if case == 'deadlock':
            continue
        campaigns = list((root / 'capacity-artifacts/campaigns').glob('campaign-*'))
        require(len(campaigns) == 1, 'capacity requires exactly one retained campaign')
        validated = CHECK.authenticate_campaign(campaigns[0])
        require(len(validated['executions']) == 1, 'capacity requires exactly one retained artifact')
        execution = validated['executions'][0]
        require(execution['seed'] == 7 and execution['ordinal'] == 0, 'original seed/ordinal changed')
        artifact = CHECK.parity.contained(campaigns[0], execution['artifact'])
        before = LIFECYCLE.tree(artifact)
        inputs, summary, streams = LIFECYCLE.cluster(artifact, read(root / 'capacity.json'))
        require(CHECK.authenticate_artifact(artifact)['identity']['target'] == provenance['module_sha256'],
                'retained abort module differs from actual original guest compilation')
        require(inputs['host_sha256'] == build['host_sha256'] and inputs['helper_sha256'] == build['helper_sha256'],
                'retained cluster belongs to another candidate')
        require(inputs['configuration']['original_limits']['observation_bytes'] == 4096
                and summary['virtual_elapsed_nanos'] > 0 and len(inputs['retained']) == 2,
                'original capacity bound/elapsed/incarnation count changed')
        require({(row['handle']['node'], row['handle']['incarnation']) for row in inputs['retained']}
                == {('a', 1), ('b', 1)}, 'capacity retained owners changed')
        for row in inputs['retained']:
            require(row['interrupted'] is True and row['evidence']['choice']['bytes'] > 0,
                    'capacity lost interrupted nonempty replayable choice prefix')
        for key, marker in {('a', 1, 'stderr'): b'ping client dialing', ('a', 1, 'stdout'): b'ping client connected',
                            ('b', 1, 'stderr'): b'echo server listening', ('b', 1, 'stdout'): b'echo server accepted'}.items():
            require(marker in streams[key], 'capacity dropped original partial stream ' + str(key))
        report = CHECK.parity.decode(run.call('inspect-json', ['inspect', '--json', artifact]))
        LIFECYCLE.capacity_inspection(report, summary, streams)
        inspection_totals(report, streams)
        text = run.call('inspect-text', ['inspect', artifact])
        for marker in (b'ping round 2', b'exhausted=payload-bytes limit=4096 used=4096 delta=64',
                       b'cluster-instance: node=a incarnation=1'):
            require(marker in text, 'capacity text inspection omitted ' + marker.decode())
        replay = run.call('replay', ['replay', artifact], 1, installed=True)
        require(b'reproduced=true' in replay, 'capacity retained prefix failed fresh replay')
        require(before == LIFECYCLE.tree(artifact), 'inspection/replay mutated retained evidence')
        artifacts[str(artifact)] = before
    return dict(commands=run.names, artifacts=artifacts)


def read_result(path, expected_build=None):
    import main_cli_build
    path = receiving.regular(path)
    result = read(path)
    root = path.parent
    recipe = read(receiving.regular(root / 'recipe.json'))
    require(path == root / 'result.json' and result['schema'] == SCHEMA, 'foreign operator receiving result')
    require(expected_build is None or recipe['main_build'] == str(expected_build), 'foreign operator main CLI build')
    build = main_cli_build.read_build(Path(recipe['main_build']), expected_role='release')
    require(recipe['repo'] == str(MODULE.parents[1]) and Path(recipe['repo']).resolve() == Path(recipe['repo']),
            'operator receiving checkout differs from current source')
    expected = evaluate(root, recipe, build, True)
    require(result['observed'] == expected and result['files'] == tree_without_result(root),
            'operator result differs from independently reopened raw commands/artifacts')
    return result


def produce(main_build, compiler_root, guest_config, module_cache, root):
    import main_cli_build
    build = main_cli_build.read_build(main_build, expected_role='release')
    require(root.is_absolute() and root.resolve() == root and not root.exists(), 'operator output root must be fresh/canonical')
    require(compiler_root.is_absolute() and compiler_root.is_dir() and not compiler_root.is_symlink(), 'stock compiler root required')
    require(not root.is_relative_to(MODULE.parents[1]), 'operator evidence must remain outside frozen checkout')
    root.mkdir(mode=0o700)
    (root / 'source-inputs.tsv').write_bytes(source_manifest(build, MODULE.parents[1]))
    write(root / 'producer.json', dict(pid=os.getpid(), start_ns=time.time_ns(), started_monotonic_nanos=time.monotonic_ns()))
    for name in ('home', 'tmp', 'source', 'inputs', 'runtime'):
        (root / name).mkdir(mode=0o700)
    shutil.copytree(compiler_root, root / 'compiler', symlinks=False)
    (root / 'source/go.mod').write_bytes(b'module gomad.wasm.typedoperator\n\ngo 1.27.1\n')
    shutil.copyfile(FIXTURES / 'main.go', root / 'source/main.go')
    (root / 'inputs/control').write_bytes(b'retained typed input')
    for name in ('gomad_semantics.go.txt', 'gomad_choicewire_generated.go.txt'):
        shutil.copyfile(MODULE / 'internal/model/runtimesource' / name, root / 'runtime' / name)
    for case in ('deadlock', 'capacity'):
        (root / (case + '.json')).write_bytes(CANONICAL(read(FIXTURES / (case + '.json'))))
    shutil.copyfile(guest_config, root / 'guest-config.json')
    dependency_inventory = installed_cluster_cache.snapshot(module_cache, root / 'module-cache', MODULE / 'go.sum')
    recipe = dict(main_build=str(main_build), compiler_root=str(compiler_root), guest_config=str(guest_config),
                  module_cache_source=str(module_cache), module_cache_inventory=dependency_inventory,
                  compiler_inventory=live.inventory(compiler_root), repo=str(MODULE.parents[1]))
    write(root / 'recipe.json', recipe)
    observed = evaluate(root, recipe, build, False)
    write(root / 'result.json', dict(schema=SCHEMA, observed=observed, files=tree_without_result(root)))
    read_result(root / 'result.json', main_build)
    return root / 'result.json'


def tree_without_result(root):
    return {name: entry for name, entry in LIFECYCLE.tree(root).items() if name != 'result.json'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    producer = sub.add_parser('produce')
    for name in ('main-build', 'compiler-root', 'guest-config', 'module-cache', 'root'):
        producer.add_argument('--' + name, type=Path, required=True)
    reader = sub.add_parser('verify')
    reader.add_argument('--receipt', type=Path, required=True)
    args = parser.parse_args()
    if args.mode == 'produce':
        print(produce(args.main_build, args.compiler_root, args.guest_config, args.module_cache, args.root))
    else:
        read_result(args.receipt)
        print('installed cluster failure raw evidence verified')


if __name__ == '__main__':
    main()
