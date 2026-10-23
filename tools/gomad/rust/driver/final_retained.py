#!/usr/bin/env python3
"""Produce and independently reread the three original retained final gates."""
import argparse
import base64
import json
import os
from pathlib import Path
import shutil
import sys
import time

from driver import HERE, Refusal, command, configured, guard, read, require, sha, write
import live
import receiving

IDS = {
    'retained': 'retained/20-fresh-observed-replays',
    'operator': 'operator/detached-installed-discover-inspect-replay',
    'inspection': 'inspection/rotated-three-node-ringpop-120s-fresh-exact-replay',
}
PROFILES = {'stock': 'gomad.wasi-stock-exploratory/v1', 'cooperative': 'gomad.wasi-cooperative/v1'}
MODULE = HERE.parents[1]
LIFECYCLE = live.load('installed-lifecycle-check')
CHECK = LIFECYCLE.check
CANONICAL = CHECK.parity.canonical


def bounded(path, limit):
    path = receiving.regular(path)
    require(path.stat().st_size <= limit, 'original retained read bound exceeded: ' + str(path))
    return path.read_bytes()


def scenario():
    value = read(MODULE / 'internal/backends/wasm/command/testdata/scenario/ringpop.json')
    for boot in value['boots']:
        require(boot['args'][4] == '90', 'original ringpop deadline changed')
        boot['args'][4] = '120'
    return value


def retained_guest(guest):
    expected = dict(Profile=PROFILES['stock'], Environment=['PWD=/workspace'], WorkingDirectory='/workspace',
        WritableDirectories=['/workspace', '/tmp'], Clock=dict(EpochNanos=946684800000000000, ReadStepNanos=1000),
        Limits=dict(OutputBytes=1 << 20, TranscriptBytes=8 << 20, Calls=10000, Descriptors=64, Files=1000,
                    FilesystemBytes=8 << 20, PendingEvents=4096))
    optional = dict(Args=None, Stdin=None, EntropyKey=[0] * 32, storage=None, network_v2=None)
    actual = {key: value for key, value in guest.items() if key != 'CapturedInputs'}
    for key, default in optional.items():
        if actual.get(key) == default:
            actual.pop(key, None)
    require(actual == expected, 'original retained guest clock/limits/environment changed')
    retained_capture(guest)


def retained_capture(guest):
    LIFECYCLE.installation_check.authenticate_captured(guest)
    captured = guest['CapturedInputs']
    descriptor = CHECK.parity.decode(base64.b64decode(captured['Descriptor'], validate=True))
    require(descriptor['mappings'] == ['/inputs'] and descriptor.get('not_exist', []) == []
            and [entry['path'] for entry in descriptor['entries']] == ['/inputs', '/inputs/input.txt'],
            'retained original captured paths changed')
    require(captured['Manifest']['Limits'] == dict(PathBytes=4096, Requests=100000, Files=10000,
            DirectoryEntries=100000, SingleFileBytes=16 << 20, TotalBytes=64 << 20),
            'retained original captured-input limits changed')
    entries = [entry for entry in descriptor['entries'] if entry['kind'] == 'file']
    require(len(entries) == 1 and base64.b64decode(captured['Payloads'][entries[0]['payload']], validate=True)
            == b'immutable captured control', 'retained original captured control changed')


def sealed(config):
    entry = config['final_retained']
    live.role(config, 'release')
    live.sealed_input(config, HERE / 'final-retained-original.json')
    for name in ('compiler_root', 'runtime_root'):
        live.sealed_input(config, entry[name])
    require(set(entry['guest_configs']) == {'retained', 'stock', 'cooperative', 'ringpop'},
            'missing original retained/operator/ringpop guest configuration')
    for path in entry['guest_configs'].values():
        live.sealed_input(config, path)
    for name, profile in PROFILES.items():
        guest = read(entry['guest_configs'][name])
        LIFECYCLE.guest_base(guest, profile, False)
        LIFECYCLE.captured_input(guest, b'immutable operator input')
    retained = read(entry['guest_configs']['retained'])
    require(retained['Profile'] == PROFILES['stock'], 'retained gate requires original stock observed profile')
    retained_guest(retained)
    require(read(entry['guest_configs']['ringpop']) == read(MODULE /
            'internal/backends/wasm/command/testdata/scenario/ringpop-config.json'),
            'original ringpop configuration changed')
    return entry


class PrivateRetained(receiving.Retained):
    def __init__(self, config):
        super().__init__(config)
        self.private_pins = {}

    def admit_private(self, selected, before):
        for name in ('compiler_root', 'runtime_root', 'source'):
            for relative, entry in before[name].items():
                path = str(selected[name] / relative)
                require(path not in self.config['files'] and path not in self.private_pins,
                        'private original input aliases a frozen input')
                self.private_pins[path] = entry['sha256']

    def provenance(self, result):
        pinned = dict(self.config, files=dict(self.config['files'], **self.private_pins))
        return receiving.Retained.provenance(receiving.Retained(pinned), result)


def observed_command(config, root, name):
    row = read(root / (name + '.command.json'))
    process = read(root / (name + '.process.json'))
    owner = read(root / 'producer-root.json')
    require(process['stage_pid'] == owner['pid'] and row['start_ns'] >= owner['start_ns']
            and owner['start_ns'] >= config['sent_epoch'] * 1_000_000_000,
            'final command belongs to another producer/process interval')
    require(row.get('environment', {}) == config.get('environment', {})
            and row.get('temporary_paths', {}) == config.get('temporary_paths', {}),
            'final command environment changed')
    return row


def command_files(root):
    result = {}
    for work in (root, root / 'stock', root / 'cooperative'):
        if work.is_dir():
            for path in work.iterdir():
                if path.name.endswith(('.command.json', '.process.json', '.stdout', '.stderr', 'producer-root.json')):
                    receiving.regular(path)
                    result[str(path.relative_to(root))] = dict(bytes=path.stat().st_size, sha256=sha(path))
    require(result, 'missing actual final command files')
    return result


class Run:
    def __init__(self, config, root, reader=None):
        self.config, self.root, self.reader = config, root, reader
        self.index = []

    def call(self, name, arguments, status=0, installation=None, timeout=180):
        argv = [self.config['binary'], *(['--installation', str(installation)] if installation else []),
                *map(str, arguments)]
        raw = (self.reader.command(self.config, self.root, name, argv, accepted=(status,), timeout=timeout)
               if self.reader else command(self.config, self.root, name, argv, accepted=(status,), timeout=timeout))
        observed_command(self.config, self.root, name)
        self.index.append(name)
        return raw


def paths(root):
    return {name: root / name for name in ('source', 'inputs', 'compiler_root', 'runtime_root',
                                          'guest_config', 'preparation_cache', 'compile_cache')}


def source_files(kind):
    if kind == 'ringpop':
        original = MODULE / 'internal/backends/wasm/testdata/ringpop'
        return {name: (original / name).read_bytes() for name in ('go.mod', 'go.sum', 'main.go')}
    fixture = (MODULE / 'internal/backends/wasm/testdata/io_failure_module/io_failure/io_failure_test.go').read_bytes()
    if kind == 'retained':
        return {'go.mod': b'module gomad.wasm.retained.fixture\n\ngo 1.27.1\n', 'io_failure/io_failure_test.go': fixture}
    return {'go.mod': b'module gomad.wasm.operator.fixture\n\ngo 1.27.1\n', 'io_failure_test.go': fixture,
            'captured_test.go': (MODULE / 'internal/backends/wasm/command/testdata/captured_test.go').read_bytes()}


def install(config, root, kind, reader=None):
    entry = config['final_retained']
    selected = paths(root)
    profile = PROFILES['stock' if kind == 'retained' else 'cooperative' if kind == 'ringpop' else kind]
    installation = root / 'installation.json'
    if reader is None:
        root.mkdir(mode=0o700)
        write(root / 'producer-root.json', dict(pid=os.getpid(), start_ns=time.time_ns()))
        for name in ('compiler_root', 'runtime_root'):
            shutil.copytree(entry[name], selected[name], symlinks=False)
        selected['source'].mkdir()
        selected['inputs'].mkdir()
        selected['preparation_cache'].mkdir()
        shutil.copyfile(entry['guest_configs'][kind], selected['guest_config'])
        source = selected['source']
        for name, raw in source_files(kind).items():
            (source / name).parent.mkdir(parents=True, exist_ok=True)
            (source / name).write_bytes(raw)
        selected['inputs'].joinpath('input.txt' if kind == 'retained' else 'control').write_bytes(
            b'immutable captured control' if kind == 'retained' else b'immutable operator input')
        write(root / 'before.json', {name: live.inventory(path) if path.is_dir() else {'sha256': sha(path)}
                                     for name, path in selected.items() if name not in ('compile_cache', 'preparation_cache')})
    run = Run(config, root, reader)
    argv = ['install', '--compiler', selected['compiler_root'] / 'bin/go', '--cache', selected['preparation_cache'],
            '--compile-cache', selected['compile_cache'], '--profile', profile, '--config', selected['guest_config'],
            '--memory-bytes', str(1 << 30 if kind == 'ringpop' else 256 << 20),
            '--fuel', str(1 << 50 if kind == 'ringpop' else 4000000000), '--output', installation]
    if profile == PROFILES['cooperative']:
        argv += ['--runtime-root', selected['runtime_root']]
    run.call('install', argv)
    admitted = read(installation)
    require(admitted['host_sha256'] == config['identity']['host']
            and admitted['helper_sha256'] == config['identity']['helper'] and admitted['profile'] == profile,
            'foreign installed final candidate')
    before = read(root / 'before.json')
    for name in ('compiler_root', 'runtime_root'):
        require(before[name] == live.inventory(Path(entry[name])), 'private original input copy differs from sealed template')
    require(before['source'] == {name: dict(bytes=len(raw), sha256=receiving.digest(raw))
                                 for name, raw in source_files(kind).items()}, 'original private source fixture changed')
    expected_input = b'immutable captured control' if kind == 'retained' else b'immutable operator input'
    expected_input_name = 'input.txt' if kind == 'retained' else 'control'
    require(before['guest_config'] == {'sha256': sha(entry['guest_configs'][kind])}
            and before['inputs'] == {expected_input_name: dict(bytes=len(expected_input), sha256=receiving.digest(expected_input))},
            'original private captured/config input copy changed')
    require(admitted['compiler_sha256'] == sha(Path(entry['compiler_root']) / 'bin/go'),
            'final installation compiler differs from original pin')
    require(admitted['admission']['guest_configuration'] == read(entry['guest_configs'][kind]),
            'installation did not retain original guest configuration')
    if reader is not None:
        reader.admit_private(selected, before)
    return run, selected, installation


def detach(root, selected, reader=None):
    before = read(root / 'before.json')
    require(set(before) == set(selected) - {'compile_cache', 'preparation_cache'}, 'incomplete original detach inventory')
    for name, path in selected.items():
        if name == 'compile_cache':
            continue
        moved = path.with_name(path.name + '-unavailable')
        if reader is None:
            path.rename(moved)
        require(not path.exists() and not path.is_symlink() and moved.exists() and not moved.is_symlink(),
                'original build input remains available: ' + name)
        if name != 'preparation_cache':
            require((live.inventory(moved) if moved.is_dir() else {'sha256': sha(moved)}) == before[name],
                    'detached original build input changed: ' + name)


def bounded_evidence(root):
    require(root.is_dir() and not root.is_symlink(), 'missing original retained evidence directory')
    count = 0
    for path in root.rglob('*'):
        require(not path.is_symlink(), 'symlink in original retained evidence')
        if path.is_file():
            count += 1
            limit = (128 << 20) if path.suffix == '.json' else (512 << 20)
            require(path.stat().st_size <= limit and count <= 20000, 'retained raw evidence exceeds read bound')
        else:
            require(path.is_dir(), 'nonregular retained evidence')


def candidate(config, path, record):
    require(record['identity']['implementation'] == receiving.digest(CANONICAL(config['identity']))
            and record['seed'] == 7, 'foreign final artifact candidate/seed')
    manifest = read(path / 'manifest.json')
    replay_path = 'evidence/cluster-replay.json' if 'evidence/cluster-replay.json' in manifest['files'] else 'evidence/replay.json'
    inputs = read(path / replay_path)
    require(inputs['host_sha256'] == config['identity']['host'] and inputs['helper_sha256'] == config['identity']['helper'],
            'foreign final artifact replay implementation')
    require(receiving.digest(CANONICAL(inputs['configuration'])) == record['identity']['configuration'],
            'final artifact configuration identity differs from raw replay input')


def artifact(run, destination, expected):
    campaigns = list((destination / 'campaigns').glob('campaign-*'))
    require(len(campaigns) == 1, 'original final gate requires exactly one campaign')
    bounded_evidence(campaigns[0])
    validated = CHECK.authenticate_campaign(campaigns[0])
    executions = validated['executions']
    require(len(executions) == 1 and executions[0]['seed'] == 7 and executions[0]['ordinal'] == 0,
            'original seed7 single execution changed')
    path = campaigns[0] / executions[0]['artifact']
    record = CHECK.authenticate_artifact(path)
    candidate(run.config, path, record)
    require(record['outcome']['class'] == expected, 'original final application outcome changed')
    run.call('inspect', ['inspect', '--json', path])
    return path, record


def backend_metrics(path):
    bounded(path, 8192)
    return metric_values(read(path))


def metric_values(value):
    names = {'prepare_nanos', 'compile_nanos', 'cache_load_nanos', 'cache_write_nanos', 'cache_hit',
             'instantiate_nanos', 'execute_nanos', 'process_user_nanos', 'process_system_nanos',
             'peak_rss_bytes', 'resource_usage_available'}
    require(isinstance(value, dict) and set(value) == names, 'missing original raw backend metric fields')
    require(all(type(item) is bool if key in ('cache_hit', 'resource_usage_available') else type(item) is int and item >= 0
                for key, item in value.items()), 'invalid original raw backend metric value')
    return value


def original_backend_metrics(artifact, root):
    paths = list((artifact.parents[1] / 'execution-observations').glob('*.json'))
    require(len(paths) == 1, 'missing/duplicate original backend metrics observation')
    path = paths[0]
    raw = bounded(path, 8192)
    value = read(path)
    require(path.name == '0-' + receiving.digest(raw) + '.json'
            and value['schema'] == 'gomad.backend-metrics-observation/v1' and value['ordinal'] == 0
            and value['record_sha256'] == read(artifact / 'manifest.json')['record_sha256']
            and value['operator_process_id'] == read(root / 'explore.command.json')['pid'],
            'original backend metrics borrowed another artifact/operator')
    metric_values(value['metrics'])
    return dict(path=str(path), sha256=sha(path))


def retained(config, root, reader=None):
    run, selected, installation = install(config, root, 'retained', reader)
    arguments = ['explore', '--json', '--seeds=7', '--parallel=1', '--on-failure=first', '--failure-budget=1',
                 '--execution-timeout=1m', '--overall-timeout=3m', '--terminate-grace=100ms',
                 '--output-limit=1048576', '--world-transition-limit=1048576', '--artifacts', root / 'artifacts',
                 '--working-dir', selected['source'], '--build-tag=gomad_fixture', '--build-tag=test_dep',
                 'go-test', './io_failure', '--', '-test.run=^TestDeterministicIOFailure$', '-test.v']
    run.call('explore', arguments, 1, installation)
    path, record = artifact(run, root / 'artifacts', 'target_failure')
    require(record['profile'] == PROFILES['stock'] and not any('choice' in key for key in record['streams']),
            'original observed gate mislabeled exact choice replay')
    require(b'deterministic failure after reading "pending"' in bounded(path / 'output/stdout', 1 << 20),
            'original retained failure fixture did not execute')
    before = live.inventory(path)
    detach(root, selected, reader)
    original_metrics = original_backend_metrics(path, root)
    stable = read(path / 'evidence/backend-outcome.json')
    require(stable['schema'] == 'gomad.backend-outcome/v1' and stable['type'] == 'result' and 'metrics' not in stable,
            'retained original backend evidence schema/metrics boundary changed')
    expected = {'stdout': bounded(path / 'output/stdout', 1 << 20),
                'stderr': bounded(path / 'output/stderr', 1 << 20),
                'backend-evidence.bin': bounded(path / 'evidence/backend-outcome.json', 1 << 20)}
    pids, observations = set(), []
    for index in range(20):
        name = f'replay-{index + 1:02d}'
        observed = root / 'observed' / f'{index + 1:03d}'
        raw = run.call(name, ['replay', '--observed', observed, path], 1, installation)
        require(b'reproduced=true diagnostic=false result=target_failure choice-replay=none' in raw,
                'fresh observed replay did not reproduce')
        for filename, value in expected.items():
            require(bounded(observed / filename, 1 << 20) == value, 'fresh observed raw evidence differs')
        backend_metrics(observed / 'backend-metrics.json')
        observations.append(dict(path=str(observed / 'backend-metrics.json'), sha256=sha(observed / 'backend-metrics.json')))
        pid = read(root / (name + '.command.json'))['pid']
        require(pid not in pids, 'reused observed replay process')
        pids.add(pid)
    require(live.inventory(path) == before, '20 fresh replays changed immutable artifact')
    return dict(artifact=str(path), artifact_files=before, replay_processes=sorted(pids),
                original_metrics=original_metrics, replay_metrics=observations)


def operator_capability(path, kind):
    manifest = read(path / 'manifest.json')
    details = read(path / 'evidence/inspection.json') if 'evidence/inspection.json' in manifest['files'] else {}
    choices = details.get('choices')
    require(choices is not None and choices['profile'] == 'gomad-choice-trace/v4' if kind == 'cooperative'
            else choices is None, 'original operator exact choice capability changed')


def operator(config, root, reader=None):
    result = {}
    if reader is None:
        root.mkdir(mode=0o700)
    for kind in PROFILES:
        work = root / kind
        run, selected, installation = install(config, work, kind, reader)
        lifecycle_config = dict(config, **{name: str(path) for name, path in selected.items()},
            installation=str(installation), profile=PROFILES[kind], private_root=str(work),
            qualification_config=config['_config_path'], detach=[str(path) for name, path in selected.items()
                                                               if name != 'compile_cache'])
        if reader is None:
            native = LIFECYCLE.Run(lifecycle_config, work)
            LIFECYCLE.operator(native)
        else:
            journals = sorted(work.glob('[0-9][0-9][0-9][0-9]-*.command.json'))
            require(journals, 'missing installed operator commands')
            commands = []
            for journal in journals:
                row = observed_command(config, work, journal.name.removesuffix('.command.json'))
                name = journal.name.removesuffix('.command.json')
                output = reader.command(config, work, name, row['argv'], accepted=(row['exit'],), timeout=180)
                require(row.get('environment', {}) == config.get('environment', {})
                        and row.get('temporary_paths', {}) == config.get('temporary_paths', {}),
                        'operator command environment changed')
                commands.append((name.split('-', 1)[1], row, output))
            names = [name for name, _, _ in commands]
            require(names == ['discover-profile', 'operator-explore', 'inspect', 'operator-qualify',
                             'detached-qualification-report', 'discover-source-less', 'verify-only', 'replay',
                             'advertised-replay', 'cache-poison', 'cache-symlink', 'replay'],
                    'missing/extra original operator command')
            require([row['exit'] for _, row, _ in commands] == [0, 1, 0, 1, 0, 0, 0, 1, 1, 3, 3, 1],
                    'original operator refusal or application exit changed')
            for name, row, output in commands:
                require(row['argv'][0] == config['binary'], 'foreign operator executable')
                if name in ('replay', 'advertised-replay'):
                    require(b'reproduced=true' in output, 'detached operator replay failed')
            explored = CHECK.parity.decode(commands[1][2])
            advertised_rows = explored['artifacts']
            require(len(advertised_rows) == 1, 'operator advertised artifact count')
            retained_path = Path(advertised_rows[0]['path'])
            require(retained_path.is_relative_to(work / 'artifacts') and '..' not in retained_path.parts,
                    'operator borrowed external artifact')
            replay_argv = [config['binary'], '--installation', str(installation), 'replay', str(retained_path)]
            explore_argv = [config['binary'], '--installation', str(installation), 'explore', '--json',
                '--seeds=7', '--parallel=1', '--artifacts', str(work / 'artifacts'), '--working-dir', str(selected['source']),
                '--build-tag=gomad_fixture', '--build-tag=test_dep']
            if kind == 'cooperative':
                explore_argv += ['--choices', '--diagnostics']
            explore_argv += ['go-test', '.', '--', '-test.run=^(TestDeterministicIOFailure|TestCapturedInputFailure)$', '-test.v']
            spec = work / 'qualification-spec.json'
            frozen = configured(Path(config['_config_path']))
            require(read(spec) == LIFECYCLE.operator_qualification_spec(str(selected['source']), work / 'qualification', frozen),
                    'operator original qualification spec changed')
            qualified = CHECK.parity.decode(commands[3][2])
            result_path, measured = LIFECYCLE.qualification_check.authenticate(qualified, work / 'qualification')
            LIFECYCLE.operator_qualification_measurements(measured)
            require(measured['observations']['operator_process_id'] == commands[3][1]['pid'],
                    'operator qualification belongs to another process')
            prefix = [config['binary'], '--installation', str(installation)]
            expected_argv = [prefix + ['discover'], explore_argv,
                [config['binary'], 'inspect', '--json', str(retained_path)],
                prefix + ['qualify', '--spec', str(spec)],
                [config['binary'], '--installation', '/missing/source-less-report-manifest', 'report', '--result', str(result_path)],
                prefix + ['discover'], prefix + ['replay', '--verify-only', str(retained_path)], replay_argv,
                [config['binary'], 'replay', str(retained_path)], replay_argv, replay_argv, replay_argv]
            require([row['argv'] for _, row, _ in commands] == expected_argv, 'original operator command argv changed')
            report = CHECK.parity.decode(commands[4][2])
            require(report['result'] == measured and report['publication']['status'] == 'within'
                    and report['receipt']['operator_peak_rss_bytes'] is not None, 'detached operator report changed')
            require(len({row['pid'] for _, row, _ in commands}) == len(commands), 'operator reused a CLI process')
            require(all(left[1]['end_ns'] <= right[1]['start_ns'] for left, right in zip(commands, commands[1:])),
                    'operator original command chronology changed')
            detached_at = commands[4][1]['start_ns']
            require(all(row['start_ns'] >= detached_at for _, row, _ in commands[4:]), 'operator detach command order')
        detach(work, selected, reader or True)
        campaigns = list((work / 'artifacts/campaigns').glob('campaign-*'))
        require(len(campaigns) == 1, 'operator requires one retained campaign')
        bounded_evidence(campaigns[0])
        executions = CHECK.authenticate_campaign(campaigns[0])['executions']
        require(len(executions) == 1, 'operator requires one original failure')
        path = campaigns[0] / executions[0]['artifact']
        record = CHECK.authenticate_artifact(path)
        candidate(config, path, record)
        operator_capability(path, kind)
        require(record['profile'] == PROFILES[kind] and record['outcome']['class'] == 'target_failure',
                'operator original profile/outcome changed')
        stdout = bounded(path / 'output/stdout', 8 << 20)
        for marker in (b'intentional failure after captured input "immutable operator input"',
                       b'deterministic failure after reading "pending"'):
            require(marker in stdout, 'operator omitted original source/captured fixture')
        result[kind] = dict(artifact=str(path), files=live.inventory(path))
    return result


def ringpop_reference(config):
    reference = config['final_retained'].get('ringpop_preservation')
    require(isinstance(reference, dict), 'UNEXECUTED: missing original ringpop World/network/choice reference')
    require(set(reference) == {'receipt', 'world', 'network', 'choices'}
            and set(reference['choices']) == {'a', 'b', 'c'}, 'incomplete original ringpop reference')
    for path in [reference['receipt'], reference['world'], reference['network'], *reference['choices'].values()]:
        live.sealed_input(config, path)
    receipt = read(reference['receipt'])
    require(receipt['schema'] == 'gomad.ringpop-preserved-records/v1' and not receipt.get('synthetic', False)
            and receipt['seed'] == 7 and receipt['virtual_elapsed_nanos'] == 120000000001
            and receipt['scenario'] == scenario(), 'foreign/synthetic original ringpop reference')
    require(receipt['source'] == 'fn235/cohort' and receipt['helper_sha256'] != config['identity']['helper'],
            'ringpop reference must preserve the actual earlier helper lineage')
    expected_files = {'world': reference['world'], 'network': reference['network'],
                      **{'choice/' + node: path for node, path in reference['choices'].items()}}
    require(receipt['files'] == {name: dict(bytes=Path(path).stat().st_size, sha256=sha(path))
                                for name, path in expected_files.items()}, 'ringpop reference raw receipt differs')
    return reference


def ringpop_preservation(config, artifact, inputs):
    reference = ringpop_reference(config)
    for name in ('world', 'network'):
        require(bounded(artifact / ('evidence/' + name), 512 << 20) == bounded(reference[name], 512 << 20),
                'ringpop changed original canonical ' + name + ' records')
    for row in inputs['retained']:
        descriptor = row['evidence']['choice']
        previous = bounded(reference['choices'][row['handle']['node']], 64 << 20)
        require(previous and len(previous) % 96 == 0 and not previous.startswith(b'GOMADTP'),
                'original ringpop choice reference must be raw trace records, not a rotated tape header')
        require(bounded(CHECK.parity.contained(artifact, descriptor['file']), 64 << 20)
                == previous,
                'ringpop changed original per-node choice records')


def inspection(config, root, reader=None):
    run, selected, installation = install(config, root, 'ringpop', reader)
    authored = root / 'ringpop-120s.json'
    if reader is None:
        write(authored, scenario())
    require(read(authored) == scenario(), 'original ringpop scenario changed beyond declared deadline')
    argv = ['explore', '--json', '--seeds=7', '--parallel=1', '--execution-timeout=40m', '--overall-timeout=45m',
            '--choices', '--choice-bytes=67108864', '--output-limit=8388608', '--keep-successes=all',
            '--success-limit=1', '--success-bytes=4294967296', '--artifacts', root / 'artifacts',
            '--working-dir', selected['source'], '--scenario', authored, 'go-run', '.']
    run.call('explore', argv, installation=installation, timeout=2700)
    path, record = artifact(run, root / 'artifacts', 'success')
    LIFECYCLE.scenario_check.authenticate_scenario(path.parents[1], scenario())
    inputs, summary, streams = LIFECYCLE.cluster(path, scenario())
    inspection_bytes = bounded(path / 'evidence/inspection', 64 << 20)
    require(json.loads(inspection_bytes)['state_encoding'] == 'gomad.inspection-state-sha256/v2',
            'ringpop inspection state encoding did not rotate')
    for node, address in (('a', '10.0.0.1:7946'), ('b', '10.0.0.2:7946'), ('c', '10.0.0.3:7946')):
        output = streams[node, 1, 'stdout']
        for marker in (b'bootstrapped ', b'partitioned view ', b'converged '):
            require(marker + address.encode() in output, 'ringpop omitted actual membership phase witness')
    require(summary['virtual_elapsed_nanos'] == 120000000001 and len(summary['incarnations']) == 3,
            'ringpop original virtual duration/topology changed')
    require({(row['handle']['node'], row['handle']['incarnation']) for row in inputs['retained']}
            == {('a', 1), ('b', 1), ('c', 1)}, 'ringpop incarnation coverage changed')
    for row in inputs['retained']:
        require(row['evidence']['choice']['bytes'] > 0 and not row['interrupted'], 'ringpop lost complete exact guest tape')
    ringpop_preservation(config, path, inputs)
    before = live.inventory(path)
    raw = run.call('replay', ['replay', path], installation=installation, timeout=2700)
    require(b'reproduced=true diagnostic=false result=success choice-replay=none' in raw,
            'fresh ringpop exact replay failed')
    require(live.inventory(path) == before, 'ringpop replay changed canonical World/network/choices')
    return dict(artifact=str(path), artifact_files=before, virtual_elapsed_nanos=summary['virtual_elapsed_nanos'])


READERS = {'retained': retained, 'operator': operator, 'inspection': inspection}


def collect(config, root, section, read_only=False):
    guard(config)
    require(root == Path(config['root']), 'wrong retained final attempt root')
    sealed(config)
    if section == 'inspection':
        ringpop_reference(config)
    work = root / ('final-' + section)
    if not work.exists():
        require(not read_only, 'UNEXECUTED: missing original final producer')
        produced = READERS[section](config, work)
        produced['command_files'] = command_files(work)
        write(work / 'producer-proof.json', produced)
    else:
        require((work / 'producer-proof.json').is_file(), 'incomplete original final producer')
    proof = READERS[section](config, work, PrivateRetained(config))
    proof['command_files'] = command_files(work)
    require(proof == read(work / 'producer-proof.json'), 'independent final reader differs from actual producer')
    if (work / 'final-proof.json').exists():
        require(read(work / 'final-proof.json') == proof, 'final raw evidence changed since earlier independent read')
    else:
        require(not read_only, 'UNEXECUTED: missing independent final proof')
        write(work / 'final-proof.json', proof)
    guard(config)
    return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                identity=config['identity'], subgates=[dict(id=IDS[section], status='PASS', executed=True, exit=0)])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--section', choices=IDS, required=True)
    parser.add_argument('--read-only', action='store_true')
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config['_config_path'], config['_config_sha256'] = str(args.config.resolve()), sha(args.config)
        print(json.dumps(collect(config, args.root.resolve(), args.section, args.read_only), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, AssertionError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
