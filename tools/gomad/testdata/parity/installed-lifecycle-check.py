#!/usr/bin/env python3
"""Installed public CLI lifecycle predicates; actual execution requires sealed private inputs."""
import argparse
import base64
import copy
import importlib.util
import os
from pathlib import Path
import shutil
import shlex
import signal
import stat
import subprocess
import sys
import threading
import time

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
DRIVER = HERE.parents[1] / 'rust/driver'
sys.path.insert(0, str(DRIVER))
import driver
import live


def load(name):
    spec = importlib.util.spec_from_file_location('installed_' + name.replace('-', '_'), HERE / (name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


check = load('campaign-check')
scenario_check = load('campaign-scenario-check')
installation_check = load('installation-lane')
qualification_check = load('qualify-lane')
require = driver.require


def discovery(report):
    capabilities = report['capabilities']
    require(capabilities['TypedScenarios'] is True and capabilities['Simulation'] is False,
            'incorrect typed scenario discovery')
    require(report['manifest']['profile'] == 'gomad.wasi-cooperative/v1', 'incorrect discovery profile')
    for command in ('explore', 'inspect', 'replay', 'recover', 'resume'):
        require(command in capabilities['Commands'], 'discovery omitted ' + command)


def lifecycle(report, validated, total, mixed=False):
    public = report.get('campaign')
    require(isinstance(public, dict) and {'executions', 'stop_reason', 'events'} <= public.keys(),
            'missing native campaign inspection')
    # Native inspection projects authenticated outcomes; the journal supplies committed admissions and receipts.
    require(public['executions'] == validated['executions'] and public['events'] == len(validated['journal']),
            'campaign inspection differs from authenticated committed prefix')
    count = len(validated['executions'])
    require(0 < count < total and public['stop_reason'] is None,
            'not an incomplete running committed prefix')
    pending = {}
    complete = False
    for row in validated['journal']:
        event = row['event']
        if event['kind'] == 'admit':
            pending.update((job['ordinal'], False) for job in event['jobs'])
        elif event['kind'] == 'receipt':
            pending[event['execution']['ordinal']] = True
        elif event['kind'] == 'outcome':
            for item in event['executions']:
                pending.pop(item['ordinal'], None)
            pending.update((job['ordinal'], False) for job in event['admitted'])
        elif event['kind'] == 'complete':
            complete = True
    require(not complete, 'published campaign cannot establish stopped running prefix')
    if mixed:
        require(any(not received for received in pending.values()), 'mixed prefix has no unreceipted admission')
    return count


def semantics(validated):
    return [dict(ordinal=row['ordinal'], seed=row['seed'], record=row['record'],
                 **{key: row[key] for key in ('divergence', 'soft_assertions', 'scenario_steps', 'semantic_probes', 'choice_features') if key in row})
            for row in validated['executions']]


def asymmetric(original, third=False):
    value = copy.deepcopy(original)
    value['boots'][0]['args'] = ['fail-peer']
    value['boots'][1]['args'] = ['wait-peer']
    value['actions'][2]['handle']['node'] = 'b'
    value['actions'][3]['handle']['node'] = 'a'
    if third:
        value['spec']['limits']['nodes'] = 3
        peer = dict(value['spec']['nodes'][1], id='c', address='10.0.0.3')
        value['spec']['nodes'].append(peer)
        handle = dict(node='c', incarnation=1)
        value['actions'].insert(2, dict(action='start', handle=handle))
        value['actions'].append(dict(action='wait', handle=handle))
    return value


def prompt(output, elapsed, case):
    require(0 <= elapsed <= 10_000_000_000, 'failure exceeded original 10-second wall bound')
    markers = {
        'deadlock': [b'"classification":"runner_failure"', b'backend infrastructure: ', b'WASM broker global deadlock'],
        'capacity': [b'"classification":"runner_failure"',
                     b'backend capacity: WASI capacity: payload-bytes limit=4096 used=4096 delta=64'],
        'two': [b'"classification":"target_failure"'],
        'three': [b'"classification":"target_failure"'],
    }[case]
    for marker in markers:
        require(marker in output, 'failure output omitted ' + marker.decode())


def advertised(result, artifact, record_hash):
    rows = result.get('artifacts')
    require(isinstance(rows, list) and len(rows) == 1, 'missing advertised retained replay invocation')
    row = rows[0]
    expected = ['gomad', 'replay', str(artifact)]
    require(row['path'] == str(artifact) and row['record_hash'] == record_hash and row['replay_argv'] == expected
            and shlex.split(row['replay_command']) == expected, 'advertised replay differs from authenticated retained artifact')
    return expected[1:]


def durable_volumes(inspected):
    storage = inspected['artifact'].get('storage')
    require(isinstance(storage, dict) and storage['volume_count'] == 1 and storage['instances'] == [dict(
        node='a', incarnation=2, volume_ids=['a-store'])],
        'actual decoded terminal volume snapshot count/ownership changed')


def capacity_inspection(report, summary, streams):
    artifact = report.get('artifact', {})
    require(isinstance(artifact.get('files'), dict) and 'evidence/cluster_summary' in artifact['files'],
            'missing native cluster inspection evidence')
    require(summary['virtual_elapsed_nanos'] > 0 and len(summary['incarnations']) == 2,
            'capacity inspection lost elapsed time or incarnations')
    for name in ('stdout', 'stderr'):
        expected = sum(len(raw) for (node, number, stream), raw in streams.items() if stream == name)
        projected = sum(entry['bytes'] for file, entry in artifact['files'].items()
                        if file.startswith('evidence/cluster/') and file.endswith('/' + name))
        require(projected == expected, 'inspection output accounting mismatch')
        require(artifact['cluster'][name + '_bytes'] == expected,
                'combined inspection output accounting mismatch')
    projected = artifact['cluster']
    require(projected['virtual_elapsed_nanos'] == summary['virtual_elapsed_nanos']
            and projected['exhausted'] == 'payload-bytes limit=4096 used=4096 delta=64',
            'cluster inspection elapsed time/exhausted bound mismatch')
    instances = {(row['node'], row['incarnation']): row for row in projected['instances']}
    owners = {(row['handle']['node'], row['handle']['incarnation']): row for row in summary['incarnations']}
    require(len(instances) == len(projected['instances']) and instances.keys() == owners.keys(),
            'cluster inspection instance ownership mismatch')
    for handle, row in owners.items():
        instance = instances[handle]
        require(row['interrupted'] is True and instance['interrupted'] is True
                and instance['offered_output_bytes'] == row['offered_output_bytes'],
                'cluster inspection interruption/offered output mismatch')
        for name in ('stdout', 'stderr'):
            raw = streams.get((*handle, name), b'')
            require(instance[name + '_bytes'] == len(raw), 'cluster inspection instance output mismatch')
        require(instance['stdout_tail'] == streams.get((*handle, 'stdout'), b'')[-4096:].decode(errors='replace'),
                'cluster inspection authenticated stdout tail mismatch')
        retained = instance['stdout_bytes'] + instance['stderr_bytes']
        require(instance['offered_output_bytes'] >= retained
                and instance['discarded_output_bytes'] == instance['offered_output_bytes'] - retained
                and instance['truncated'] is (instance['discarded_output_bytes'] != 0),
                'cluster inspection instance truncation mismatch')
    retained = projected['stdout_bytes'] + projected['stderr_bytes']
    require(projected['offered_output_bytes'] == sum(row['offered_output_bytes'] for row in owners.values())
            and projected['offered_output_bytes'] >= retained
            and projected['discarded_output_bytes'] == projected['offered_output_bytes'] - retained
            and projected['truncated'] is (projected['discarded_output_bytes'] != 0),
            'cluster inspection retained/offered output accounting mismatch')
    message = artifact['record']['outcome']['reason']
    require('payload-bytes limit=4096 used=4096 delta=64' in message, 'inspection omitted exact exhausted bound')


def tree(root):
    require(root.is_dir() and not root.is_symlink(), 'missing/nonregular evidence tree')
    result = {}
    for path in sorted(root.rglob('*')):
        require(not path.is_symlink(), 'symlink evidence payload')
        if path.is_file():
            result[str(path.relative_to(root))] = dict(bytes=path.stat().st_size, sha256=driver.sha(path))
        else:
            require(path.is_dir(), 'nonregular evidence payload')
    require(result, 'empty evidence tree')
    return result



def instance_stream(artifact, row, name):
    descriptors = row['streams']
    require(isinstance(descriptors, dict), 'missing native instance streams')
    handle = row['handle']
    file = artifact / f"evidence/cluster/{handle['node']}/{handle['incarnation']}/{name}"
    descriptor = descriptors.get(name)
    require(not file.is_symlink() and (name in descriptors) == file.exists(),
            'native instance stream descriptor/payload mismatch')
    if name not in descriptors:
        return b''
    require(isinstance(descriptor, dict), 'missing native instance stream descriptor')
    raw = file.read_bytes()
    require((len(raw), check.parity.digest(raw)) == (descriptor['bytes'], descriptor['sha256']),
            'per-instance stream identity changed')
    return raw



def instance_catalogue(artifact, row, limits):
    descriptors = row['streams']
    require(isinstance(descriptors, dict), 'missing native instance streams')
    handle = row['handle']
    prefix = f"evidence/cluster/{handle['node']}/{handle['incarnation']}/"
    directory = artifact / prefix
    require(directory.resolve().is_relative_to(artifact.resolve()), 'native catalogue escapes artifact')
    evidence_bytes, output_bytes = 0, 0
    for name, descriptor in descriptors.items():
        require(isinstance(name, str) and name and '\\' not in name and '\0' not in name
                and all(part not in ('', '.', '..') for part in name.split('/')),
                'unsafe native evidence path')
        require(isinstance(descriptor, dict) and type(descriptor.get('bytes')) is int
                and descriptor['bytes'] >= 0, 'missing native stream byte count')
        evidence_bytes += descriptor['bytes']
        require(evidence_bytes <= limits['evidence_bytes'], 'native evidence exceeds retained evidence budget')
        if name in ('stdout', 'stderr'):
            output_bytes += descriptor['bytes']
            require(output_bytes <= limits['output_bytes'], 'native output exceeds retained output budget')
        file = check.parity.contained(artifact, prefix + name)
        require((file.stat().st_size, driver.sha(file)) == (descriptor['bytes'], descriptor['sha256']),
                'native stream descriptor/payload identity mismatch')
    present = set()
    if directory.exists():
        for file in directory.rglob('*'):
            require(not file.is_symlink(), 'native catalogue contains symlink')
            if file.is_file():
                present.add(str(file.relative_to(directory)))
    require(present == set(descriptors), 'native stream descriptor/payload catalogue mismatch')


def cluster(artifact, declared=None):
    record = check.authenticate_artifact(artifact)
    inputs = check.canonical_file(artifact / 'evidence/cluster-replay.json')
    require(inputs['schema'] == 'gomad.cluster-replay-inputs/v1', 'not typed retained cluster evidence')
    require((artifact / 'evidence/cluster-replay.json').stat().st_size <= 128 << 20, 'retained payload exceeds original cap')
    summary = check.canonical_file(artifact / 'evidence/cluster_summary')
    require(summary['schema'] == 'gomad.cluster-observation/v1', 'missing native cluster observation')
    retained = {(row['handle']['node'], row['handle']['incarnation']): row for row in inputs['retained']}
    require(len(retained) == len(inputs['retained']) == len(summary['incarnations']), 'duplicate/missing incarnations')
    streams = {}
    for row in summary['incarnations']:
        key = row['handle']['node'], row['handle']['incarnation']
        require(key in retained, 'summary not bound to retained incarnation')
        instance_catalogue(artifact, row, record['limits'])
        for name in ('stdout', 'stderr'):
            streams[*key, name] = instance_stream(artifact, row, name)
        for descriptor in retained[key]['evidence'].values():
            require(0 <= descriptor['bytes'] <= descriptor['limit'], 'unbounded incarnation payload')
            file = check.parity.contained(artifact, descriptor['file'])
            require((file.stat().st_size, driver.sha(file)) == (descriptor['bytes'], descriptor['sha256']),
                    'retained incarnation payload changed')
    require(record['profile'] != 'gomad-choice-trace/v4', 'cluster mislabeled as global exact choice tape')
    if declared is not None:
        require(inputs['configuration']['steps'] == declared['actions'], 'retained authored lifecycle changed')
    return inputs, summary, streams


class Run:
    def __init__(self, config, root):
        self.config, self.root, self.index = config, root, 0
        self.binary = str(Path(config['binary']))
        self.installation = str(Path(config['installation']))

    def command(self, name, arguments, status=0, installation=True, timeout=180):
        self.index += 1
        label = f'{self.index:04d}-{name}'
        argv = [self.binary, *(['--installation', self.installation] if installation else []), *map(str, arguments)]
        output = driver.command(self.config, self.root, label, argv, accepted=(status,), timeout=timeout)
        receipt = driver.read(self.root / (label + '.command.json'))
        require(receipt['argv'] == argv and receipt['exit'] == status, 'wrong installed CLI command capture')
        return output, receipt

    def inspect(self, artifact):
        output, _ = self.command('inspect', ['inspect', '--json', artifact], installation=False, timeout=30)
        return check.parity.decode(output)

    def replay(self, artifact, status, typed=True):
        before = tree(artifact)
        output, _ = self.command('replay', ['replay', artifact], status)
        require(b'reproduced=true' in output and (not typed or b'choice-replay=none' in output),
                'fresh replay did not reproduce sealed lifecycle')
        require(tree(artifact) == before, 'replay modified retained evidence')

    def explore(self, destination, scenario=None, seeds='7', prompt_case=False):
        arguments = ['explore', '--json', '--seeds=' + seeds, '--parallel=1', '--on-failure=all',
                     '--execution-timeout=' + ('3m' if prompt_case else '30s'),
                     '--overall-timeout=' + ('4m' if prompt_case and scenario.name.startswith(('deadlock', 'capacity'))
                                             else '3m' if scenario else '2m'), '--terminate-grace=100ms',
                     '--artifacts', destination, '--working-dir', self.config['source']]
        if scenario:
            arguments += ['--scenario', scenario, '--keep-successes=all', '--success-limit=32',
                          '--success-bytes=536870912', 'go-run', '.']
        else:
            arguments += ['--choices', '--diagnostics', 'go-test', '.', '--', '-test.run=^TestWASMKillResumeWork$']
        return arguments

    def campaign(self, destination, complete=True):
        paths = list((destination / 'campaigns').glob('campaign-*'))
        require(len(paths) == 1, 'expected exactly one native campaign')
        return paths[0], check.authenticate_campaign(paths[0], complete=complete)

    def scenario(self, fixture, mode=''):
        value = driver.read(HERE.parents[1] / 'internal/backends/wasm/command/testdata/scenario' / (fixture + '.json'))
        if mode:
            for boot in value['boots']:
                if boot['id'] == 'recover':
                    boot['args'] = [mode]
        path = self.root / (fixture + '-' + (mode or 'default') + '.json')
        path.write_bytes(check.parity.canonical(value))
        return path, value

    def detach(self, extra=()):
        paths = [Path(path) for path in self.config['detach']] + list(extra)
        require(len(paths) == len(set(paths)), 'duplicate detach paths')
        for path in paths:
            require(path.is_relative_to(Path(self.config['private_root'])) and path.exists()
                    and not path.is_symlink(), 'detach requires existing private owned path')
            require(not path.with_name(path.name + '-unavailable').exists(), 'detach destination exists')
        for path in paths:
            path.rename(path.with_name(path.name + '-unavailable'))
        driver.write(self.root / 'detached.json', [str(path) for path in paths])

    def prompt_cases(self):
        for name in ('deadlock', 'capacity', 'two', 'three'):
            if name in ('two', 'three'):
                _, original = self.scenario('deadlock', name)
                declared = asymmetric(original, name == 'three')
                path = self.root / (name + '-asymmetric.json')
                path.write_bytes(check.parity.canonical(declared))
            else:
                path, declared = self.scenario(name)
            destination = self.root / (name + '-artifacts')
            output, receipt = self.command(name, self.explore(destination, path, prompt_case=True),
                                           1 if name in ('two', 'three') else 3)
            prompt(output, receipt['end_ns'] - receipt['start_ns'], name)
            if name == 'deadlock':
                continue
            root, validated = self.campaign(destination)
            require(len(validated['executions']) == 1, 'failure retained outcome count')
            artifact = root / validated['executions'][0]['artifact']
            inputs, summary, streams = cluster(artifact, declared)
            inspected = self.inspect(artifact)
            if name == 'capacity':
                require(inputs['configuration']['original_limits']['observation_bytes'] == 4096,
                        'capacity retained original bound changed')
                for row in inputs['retained']:
                    require(row['interrupted'], 'capacity lost interrupted replay prefix')
                    require(row['evidence']['choice']['bytes'] > 0, 'capacity lost choice tape')
                for key, marker in {('a', 1, 'stderr'): b'ping client dialing',
                                    ('a', 1, 'stdout'): b'ping client connected',
                                    ('b', 1, 'stderr'): b'echo server listening',
                                    ('b', 1, 'stdout'): b'echo server accepted'}.items():
                    require(marker in streams[key], 'capacity dropped partial per-instance output')
                capacity_inspection(inspected, summary, streams)
                text, _ = self.command('inspect-text', ['inspect', artifact], installation=False)
                for marker in (b'ping round 2', b'exhausted=payload-bytes limit=4096 used=4096 delta=64',
                               b'cluster-instance: node=a incarnation=1'):
                    require(marker in text, 'public text inspection omitted ' + marker.decode())
            else:
                require(len(inputs['retained']) == len(declared['spec']['nodes']), 'asymmetric failure lost peer')
                for node in declared['spec']['nodes']:
                    marker = b'failing peer started' if node['id'] == 'a' else b'waiting peer started'
                    require(any(marker in raw for key, raw in streams.items() if key[0] == node['id']),
                            'asymmetric failure lost node stream')
                stderr = b''.join(raw for key, raw in streams.items() if key[2] == 'stderr')
                require(len(stderr) <= 64 << 10, 'asymmetric stderr exceeds original read bound')
                for marker in (b'failing peer started', b'waiting peer started', b'intentional asymmetric peer failure'):
                    require(marker in stderr, 'asymmetric combined stderr omitted marker')
            self.replay(artifact, 1)

    def damage(self, artifact, files, status):
        for file in files:
            for damage in ('missing', 'tampered'):
                target = self.root / ('damage-' + str(self.index) + '-' + damage)
                shutil.copytree(artifact, target)
                check.authenticate_artifact(target)
                path = check.parity.contained(target, file)
                if damage == 'missing':
                    path.unlink()
                else:
                    raw = bytearray(path.read_bytes())
                    if raw:
                        raw[-1] ^= 1
                    else:
                        raw.append(1)
                    path.write_bytes(raw)
                self.command('refuse-' + damage, ['replay', target], status)

    def detached(self):
        retained, scenarios = [], []
        for mode in ('recover', 'canary', 'network'):
            scenario, declared = self.scenario('network' if mode == 'network' else 'durable',
                                               '' if mode == 'network' else mode)
            scenarios.append(scenario)
            destination = self.root / (mode + '-artifacts')
            self.command(mode, self.explore(destination, scenario), 1 if mode == 'canary' else 0)
            root, validated = self.campaign(destination)
            require(len(validated['executions']) == 1, 'detached scenario artifact count')
            retained.append((mode, root / validated['executions'][0]['artifact'], declared))
        self.detach(scenarios)
        discovery(check.parity.decode(self.command('discover-detached', ['discover'])[0]))
        for mode, artifact, declared in retained:
            inspected = self.inspect(artifact)
            require(inspected['artifact']['choices'] is None, 'cluster mislabeled as global exact choice tape')
            executed = dict(declared, actions=declared['actions'][:5] if mode == 'canary' else declared['actions'])
            scenario_check.authenticate_scenario(artifact.parents[1], executed)
            inputs, summary, streams = cluster(artifact, executed)
            expected_count = {'recover': 6, 'canary': 5, 'network': 8}[mode]
            require(len(inputs['configuration']['steps']) == expected_count, 'retained lifecycle transition count')
            if mode != 'network':
                require(len(inputs['retained']) == 3, 'missing durable incarnations')
                declared_volumes = declared['spec']['volumes']
                require(len(declared_volumes) == 1, 'durable fixture volume topology changed')
                durable_volumes(inspected)
                for row in inputs['retained']:
                    handle = row['handle']
                    expected_interrupted = (handle == dict(node='a', incarnation=1)
                                            or mode == 'canary' and handle['node'] == 'b')
                    require(row['interrupted'] == expected_interrupted, 'durable interruption ownership changed')
                    require(row['evidence']['choice']['bytes'] > 0, 'durable incarnation lost exact choice tape')
                    storage = row['evidence'].get('storage')
                    require(storage is not None and storage['sha256'] if handle['node'] == 'a' else True,
                            'durable incarnation lost storage identity')
            output = b''.join(streams.values())
            markers = {'recover': [b'committed durable-a', b'recovered durable-a', b'independent peer'],
                       'canary': [b'committed durable-a', b'recovered durable-a', b'intentional replayable scenario failure'],
                       'network': [b'partition blocked dial', b'partition blocked accept', b'healed server ping',
                                   b'healed client ping']}[mode]
            for marker in markers:
                require(marker in output, 'detached lifecycle omitted marker')
            if mode == 'canary':
                require(b'independent peer' not in output, 'failed canary let pending peer continue')
            # The authenticated campaign outcome and replay configuration retain the executed lifecycle.
            require(len(inputs['configuration']['steps']) == expected_count, 'missing retained lifecycle transitions')
            self.replay(artifact, 1 if mode == 'canary' else 0)
            manifest = check.canonical_file(artifact / 'manifest.json')
            files = [name for name in manifest['files'] if name == 'target/module.wasm'
                     or name.startswith('evidence/')]
            require('target/module.wasm' in files and 'evidence/cluster-replay.json' in files, 'missing retained references')
            self.damage(artifact, files, 3)
            self.command('missing-installation', ['--installation', '/missing/incompatible-installation', 'replay', artifact],
                         3, installation=False)


def processes():
    output = subprocess.run(['ps', '-axo', 'pid=,ppid=,pgid=,stat=,args='], check=True,
                            capture_output=True, timeout=5).stdout
    rows = []
    for line in output.decode().splitlines():
        fields = line.split(None, 4)
        if len(fields) == 5:
            rows.append(dict(pid=int(fields[0]), parent=int(fields[1]), group=int(fields[2]),
                             state=fields[3], argv=fields[4]))
    return rows


def owned_processes(owned, rows):
    while True:
        added = {row['pid'] for row in rows if row['parent'] in owned} - owned
        if not added:
            return
        owned.update(added)


def wait_for(predicate, seconds=120):
    deadline = time.monotonic() + seconds
    while True:
        value = predicate()
        if value:
            return value
        require(time.monotonic() < deadline, 'original lifecycle polling bound exceeded')
        threading.Event().wait(0.01)


def completed_prefix(root, validated, count):
    result, classes = {}, set()
    for execution in validated['executions']:
        if execution['ordinal'] < count and execution['artifact'] is not None:
            artifact = root / execution['artifact']
            check.authenticate_artifact(artifact)
            result[execution['artifact']] = tree(artifact)
            classes.add(execution['record']['outcome']['class'])
    require(result and {'success', 'target_failure'} <= classes,
            'committed mixed prefix lacks retained success/failure evidence')
    return result


def cluster_identities(root, validated):
    result = {}
    for execution in validated['executions']:
        artifact = root / execution['artifact']
        cluster(artifact)
        record = check.authenticate_artifact(artifact)
        # Native record binds lifecycle, World, network, storage and per-instance tapes independently.
        result[execution['seed']] = check.parity.digest(check.parity.canonical(record))
    return result


def kill_resume(run, total):
    require(sys.platform == 'linux', 'actual SIGSTOP/SIGKILL lane requires Linux')
    mixed = total == 24
    scenario = run.scenario('durable', 'mixed')[0] if mixed else None
    want = 1 if mixed else 0
    destination = run.root / 'baseline'
    run.command('baseline', run.explore(destination, scenario, '1-' + str(total)), want)
    baseline_root, baseline = run.campaign(destination)
    classes = [row['record']['outcome']['class'] for row in baseline['executions']]
    require(len(classes) == total and all(value in ('success', 'target_failure') for value in classes),
            'baseline attempted/succeeded/failure count changed')
    require({'success', 'target_failure'} <= set(classes) if mixed else set(classes) == {'success'},
            'baseline outcome domains changed')
    minimum = next(index + 1 for index in range(total) if {'success', 'target_failure'} <= set(classes[:index + 1])) if mixed else 1
    require(minimum < total, 'mixed baseline has no interruptible completed prefix')
    destination = run.root / 'interrupted'
    run.index += 1
    label = f'{run.index:04d}-actual-kill'
    argv = [run.binary, '--installation', run.installation, *map(str, run.explore(destination, scenario, '1-' + str(total))) ]
    results, errors, owned, observations = [], [], set(), []

    def launch():
        try:
            results.append(driver.command(run.config, run.root, label, argv, accepted=(-int(signal.SIGKILL),), timeout=180))
        except BaseException as error:
            errors.append(error)

    worker = threading.Thread(target=launch)
    worker.start()
    process_path = run.root / (label + '.process.json')
    pid = None
    try:
        wait_for(lambda: process_path.exists(), 30)
        pid = driver.read(process_path)['pid']
        owned.add(pid)

        def running():
            require(worker.is_alive(), 'explore exited before actual signal freeze: ' + str(errors))
            paths = list((destination / 'campaigns').glob('campaign-*'))
            if len(paths) != 1 or not (paths[0] / 'journal.jsonl').exists():
                return False
            # A reader racing an append may see a partial final line; no partial evidence earns credit.
            try:
                validated = check.authenticate_campaign(paths[0], complete=False)
            except (ValueError, FileNotFoundError):
                return False
            if not minimum <= len(validated['executions']) < total:
                return False
            report = run.inspect(paths[0])
            try:
                lifecycle(report, validated, total, mixed)
            except ValueError:
                return False
            return paths[0], validated

        root, initial = wait_for(running)
        os.killpg(pid, signal.SIGSTOP)
        rows = processes()
        owned_processes(owned, rows)
        for member in sorted(owned):
            try:
                os.kill(member, signal.SIGSTOP)
            except ProcessLookupError:
                pass
        observations.append(dict(signal=int(signal.SIGSTOP), pid=pid, owned=sorted(owned), processes=rows,
                                 time_ns=time.time_ns()))

        def stopped():
            rows = processes()
            members = [row for row in rows if row['pid'] in owned]
            return rows if any(row['pid'] == pid for row in members) and all(
                any(state in row['state'] for state in 'TZ') for row in members) else False

        frozen_processes = wait_for(stopped)
        validated = check.authenticate_campaign(root, complete=False)
        committed = lifecycle(run.inspect(root), validated, total, mixed)
        require(committed >= len(initial['executions']), 'stopped committed prefix shrank')
        frozen_log = (root / 'journal.jsonl').read_bytes()
        frozen = completed_prefix(root, validated, committed) if mixed else None
        owned_processes(owned, processes())
        os.killpg(pid, signal.SIGKILL)
        for member in sorted(owned - {pid}):
            try:
                os.kill(member, signal.SIGKILL)
            except ProcessLookupError:
                pass
        observations.append(dict(signal=int(signal.SIGKILL), pid=pid, owned=sorted(owned),
                                 stopped_processes=frozen_processes, time_ns=time.time_ns()))
        worker.join(timeout=30)
        require(not worker.is_alive(), 'SIGKILLed CLI failed to exit')
        require(not errors and len(results) == 1, 'actual SIGKILL capture failed: ' + str(errors))
        wait_for(lambda: all(row['pid'] not in owned - {pid} for row in processes()))
        process = driver.read(process_path)
        require(process['exit'] == -int(signal.SIGKILL) and process['active'] is False,
                'actual CLI did not die from SIGKILL')
        after = check.authenticate_campaign(root, complete=False)
        require(lifecycle(run.inspect(root), after, total, mixed) == committed,
                'post-kill committed prefix changed')
        require((root / 'journal.jsonl').read_bytes() == frozen_log, 'frozen durable journal changed after kill')
        if mixed:
            run.detach([scenario])
            incompatible = driver.read(run.installation)
            incompatible['model']['identity'] = 'sha256:' + '0' * 64
            path = run.root / 'incompatible-installation.json'
            path.write_bytes(check.parity.canonical(incompatible))
            run.command('refuse-incompatible-resume', ['--installation', path, 'resume', '--json', root],
                        3, installation=False)
            require(completed_prefix(root, after, committed) == frozen, 'refused resume overwrote frozen artifacts')
            for damage in ('missing-module', 'corrupt-log'):
                target = run.root / damage
                shutil.copytree(root, target)
                before = completed_prefix(target, after, committed)
                if damage == 'missing-module':
                    prepared = target / after['plan']['prepared_artifact']
                    modules = list(prepared.rglob('module.wasm'))
                    require(modules, 'no retained prepared module to remove')
                    for module in modules:
                        module.unlink()
                    run.command(damage, ['resume', '--json', target], 2)
                else:
                    path = target / 'journal.jsonl'
                    raw = bytearray(path.read_bytes())
                    require(raw, 'empty frozen journal')
                    raw[0] ^= 1
                    path.write_bytes(raw)
                    run.command(damage, ['recover', '--json', target], 2, installation=False)
                require(completed_prefix(target, after, committed) == before,
                        'damage refusal overwrote completed artifacts')
            run.command('recover', ['recover', '--json', root], installation=False)
            recovered = check.authenticate_campaign(root, complete=False)
            require(lifecycle(run.inspect(root), recovered, total, mixed) == committed,
                    'recover published or changed incomplete committed prefix')
        run.command('resume', ['resume', '--json', root], want)
        resumed = check.authenticate_campaign(root)
        require(semantics(baseline) == semantics(resumed), 'resume campaign semantics changed')
        public = run.inspect(root)['campaign']
        require(public['executions'] == resumed['executions'] and public['stop_reason'] is not None,
                'resumed campaign was not publicly completed')
        if mixed:
            require(completed_prefix(root, resumed, committed) == frozen, 'resume changed frozen completed evidence')
            require(cluster_identities(baseline_root, baseline) == cluster_identities(root, resumed),
                    'resume changed retained lifecycle/network/storage logical records')
    finally:
        for member in sorted(owned):
            try:
                os.kill(member, signal.SIGKILL)
            except ProcessLookupError:
                pass
        worker.join(timeout=30)
        driver.write(run.root / 'actual-signals.json', observations)



def operator_qualification_spec(source, root, frozen):
    # Native publication layout is declared by the existing qualification receiver.
    return dict(Manifest=dict(Cell='canary', Provisional=False,
        IntegratedSourceManifestPath=frozen['source_manifest'], ID='installed-operator-canary',
        Workload='fixture', IntegratedSourceSHA256=frozen['source_sha256'], Seed=7, Repeat=2,
        Unsupported=['production Temporal support', 'cross-host exact portability', 'weak-memory coverage']),
        Campaign=dict(Target=dict(Backend='wasm', Kind='go-test', Source='.', WorkingDir=source,
            PreparationRoot=str(root / 'preparation'), BuildTags=['gomad_fixture', 'test_dep'],
            Args=['-test.run=^(TestDeterministicIOFailure|TestCapturedInputFailure)$', '-test.v']),
            ExecutionTimeout=60_000_000_000, OverallTimeout=180_000_000_000, TerminateGrace=100_000_000,
            OnFailure='all', FailureBudget=1, OutputLimit=8 << 20, WorldTransitionLimit=1 << 20,
            Artifacts=str(root / 'artifacts'), RunnerBuild='installed-operator-qualification'),
        ArtifactRoot=str(root))


def operator_qualification_measurements(result):
    require(result['manifest']['cell'] == 'canary' and result['report']['deterministic'] is True
            and result['report']['target_success'] is False and result['cost'] is not None
            and result['cost_failure'] is None and result['cost']['executions'] == 2
            and result['cost']['replays'] == 2, 'operator qualification missing measured failing-target replay')


def operator_qualification(run):
    frozen = driver.configured(Path(run.config['qualification_config']))
    root = run.root / 'qualification'
    spec = run.root / 'qualification-spec.json'
    driver.write(spec, operator_qualification_spec(run.config['source'], root, frozen))
    output, receipt = run.command('operator-qualify', ['qualify', '--spec', spec], 1)
    require(len(output) <= 8 << 10, 'qualification response exceeds original 8 KiB bound')
    response = check.parity.decode(output)
    path, result = qualification_check.authenticate(response, root)
    operator_qualification_measurements(result)
    require(result['observations']['operator_process_id'] == receipt['pid'],
            'qualification publication is not the captured operator process')
    return path, result, tree(root)


def detached_qualification_report(run, qualified):
    path, expected, inventory = qualified
    output, _ = run.command('detached-qualification-report',
        ['--installation', '/missing/source-less-report-manifest', 'report', '--result', path],
        installation=False)
    report = check.parity.decode(output)
    require(report['result'] == expected and report['publication']['status'] == 'within'
            and report['receipt']['operator_peak_rss_bytes'] is not None,
            'separate detached reader rejected measured publication receipt')
    require(tree(path.parents[2]) == inventory, 'detached report modified qualification evidence')


def operator(run):
    profile = run.config['profile']
    require(profile in ('gomad.wasi-stock-exploratory/v1', 'gomad.wasi-cooperative/v1'), 'unsupported operator profile')
    discovered = check.parity.decode(run.command('discover-profile', ['discover'])[0])
    require(discovered['manifest']['profile'] == profile, 'discovery omitted selected profile')
    arguments = ['explore', '--json', '--seeds=7', '--parallel=1', '--artifacts', run.root / 'artifacts',
                 '--working-dir', run.config['source'], '--build-tag=gomad_fixture', '--build-tag=test_dep']
    if profile == 'gomad.wasi-cooperative/v1':
        arguments += ['--choices', '--diagnostics']
    arguments += ['go-test', '.', '--', '-test.run=^(TestDeterministicIOFailure|TestCapturedInputFailure)$', '-test.v']
    output, _ = run.command('operator-explore', arguments, 1)
    result = check.parity.decode(output)
    root, validated = run.campaign(run.root / 'artifacts')
    require(len(validated['executions']) == 1 and validated['executions'][0]['artifact'],
            'operator missing authenticated retained artifact')
    artifact = root / validated['executions'][0]['artifact']
    record = check.authenticate_artifact(artifact)
    invocation = advertised(result, artifact, check.parity.digest(check.parity.canonical(record)))
    raw = (artifact / 'output/stdout').read_bytes()
    require(len(raw) <= 8 << 20, 'operator guest output exceeds original read cap')
    for marker in (b'intentional failure after captured input "immutable operator input"',
                   b'deterministic failure after reading "pending"'):
        require(marker in raw, 'operator failed to exercise captured and unchanged fixture')
    inputs = check.canonical_file(artifact / 'evidence/replay.json')
    require(record['profile'] == profile, 'incorrect retained replay profile')
    manifest = check.canonical_file(artifact / 'manifest.json')
    details = check.canonical_file(artifact / 'evidence/inspection.json') if 'evidence/inspection.json' in manifest['files'] else None
    if profile == 'gomad.wasi-cooperative/v1':
        require(details is not None and details['choices'] is not None and details['choices']['profile'] == 'gomad-choice-trace/v4',
                'cooperative artifact lost exact choice profile')
    else:
        require(details is None or details['choices'] is None, 'stock artifact mislabeled exact choices')
    run.inspect(artifact)
    # This corpus always has a frozen source manifest, so the original conditional branch is mandatory.
    qualified = operator_qualification(run)
    run.detach()
    detached_qualification_report(run, qualified)
    run.command('discover-source-less', ['discover'])
    run.command('verify-only', ['replay', '--verify-only', artifact])
    run.replay(artifact, 1, typed=False)
    advertised_output, _ = run.command('advertised-replay', invocation, 1, installation=False)
    require(b'reproduced=true' in advertised_output, 'advertised retained replay did not reproduce')
    cache = Path(run.config['compile_cache'])
    entries = list(cache.glob('*.gomad-cache-v3'))
    require(len(entries) == 1, 'same retained module must publish one compiled-code identity')
    entry = entries[0]
    require(entry.is_file() and not entry.is_symlink() and stat.S_IMODE(entry.stat().st_mode) == 0o600,
            'compiled code is not private regular storage')
    original = entry.read_bytes()
    require(original, 'empty compiled-code cache entry')
    poisoned = bytearray(original)
    poisoned[-1] ^= 1
    try:
        entry.write_bytes(poisoned)
        run.command('cache-poison', ['replay', artifact], 3)
    finally:
        entry.write_bytes(original)
    held = entry.with_name(entry.name + '.held')
    entry.rename(held)
    try:
        entry.symlink_to(held)
        run.command('cache-symlink', ['replay', artifact], 3)
    finally:
        entry.unlink()
        held.rename(entry)
    run.replay(artifact, 1, typed=False)


def external_cluster(run):
    artifact = Path(run.config['external_artifact'])
    record = check.authenticate_artifact(artifact)
    cluster(artifact)
    run.replay(artifact, 0 if record['outcome']['class'] == 'success' else 1)
    run.command('external-missing-installation', ['--installation', '/missing/detached-cluster-installation',
                'replay', artifact], 3, installation=False)
    for option in ('--verify-only', '--observed=' + str(run.root / 'observed'), '--toolchain-root=/missing/native'):
        run.command('external-invalid-option', ['--installation', '/missing/detached-cluster-installation',
                    'replay', option, artifact], 2, installation=False)
    target = run.root / 'external-corrupt'
    shutil.copytree(artifact, target)
    check.authenticate_artifact(target)
    path = target / 'evidence/cluster-replay.json'
    raw = bytearray(path.read_bytes())
    require(raw, 'empty retained cluster payload')
    raw[-1] ^= 1
    path.write_bytes(raw)
    run.command('external-corrupt', ['replay', target], 3)


def guest_base(guest, profile, typed):
    if typed:
        expected = driver.read(HERE.parents[1] / 'internal/backends/wasm/command/testdata/scenario/config.json')
        expected['Environment'].append('GOMAD_REQUIRE_CAPTURED_INPUT=1')
    else:
        expected = dict(Profile=profile, Environment=['PWD=/workspace'], WorkingDirectory='/workspace',
            WritableDirectories=['/workspace', '/tmp'], Clock=dict(EpochNanos=946684800000000000,
            ReadStepNanos=0 if profile == 'gomad.wasi-cooperative/v1' else 1000),
            Limits=dict(OutputBytes=8 << 20, TranscriptBytes=64 << 20, Calls=1000000, Descriptors=64,
                        Files=1000, FilesystemBytes=64 << 20, PendingEvents=4096))
    optional = dict(Args=None, Stdin=None, EntropyKey=[0] * 32, storage=None, network_v2=None)
    actual = {key: value for key, value in guest.items() if key != 'CapturedInputs'}
    for key, default in optional.items():
        if key not in expected and actual.get(key) == default:
            actual.pop(key, None)
    require(actual == expected, 'original guest configuration/limits/environment changed')


def captured_input(guest, expected):
    installation_check.authenticate_captured(guest)
    captured = guest['CapturedInputs']
    descriptor = check.parity.decode(base64.b64decode(captured['Descriptor'], validate=True))
    require(descriptor['mappings'] == ['/inputs'] and descriptor.get('not_exist', []) == []
            and [entry['path'] for entry in descriptor['entries']] == ['/inputs', '/inputs/control'],
            'captured input has extra/missing original paths')
    files = [entry for entry in descriptor['entries'] if entry['kind'] == 'file']
    require(len(files) == 1 and base64.b64decode(captured['Payloads'][files[0]['payload']], validate=True) == expected,
            'captured original control bytes changed')
    require(captured['Manifest']['Limits'] == dict(PathBytes=4096, Requests=100000, Files=10000,
            DirectoryEntries=100000, SingleFileBytes=16 << 20, TotalBytes=64 << 20),
            'captured input original byte/path/request limits changed')


def source_inputs(config, family):
    declaration = driver.read(HERE / 'installed-lifecycle-corpus.json')
    source = Path(config['source'])
    require(source.is_absolute() and source.is_dir() and not source.is_symlink(), 'missing private source fixture')
    if family == 'kill-12':
        expected = {name: raw.encode() for name, raw in declaration['kill_fixture'].items()}
        require(sorted(path.name for path in source.iterdir()) == sorted(expected), 'unexpected source fixture files')
        for name, raw in expected.items():
            path = source / name
            require(path.is_file() and not path.is_symlink() and path.read_bytes() == raw, 'changed source fixture ' + name)
        return
    if family == 'operator':
        expected = {'io_failure_test.go': 'tools/gomad/internal/backends/wasm/testdata/io_failure_module/io_failure/io_failure_test.go',
                    'captured_test.go': 'tools/gomad/internal/backends/wasm/command/testdata/captured_test.go'}
        module = b'module gomad.wasm.operator.fixture\n\ngo 1.27.1\n'
    else:
        expected = {'main.go': 'tools/gomad/internal/backends/wasm/command/testdata/scenario/main.go'}
        module = b'module gomad.wasm.typedoperator\n\ngo 1.27.1\n'
    require(sorted(path.name for path in source.iterdir()) == sorted(['go.mod', *expected]), 'unexpected source fixture files')
    require((source / 'go.mod').is_file() and (source / 'go.mod').read_bytes() == module, 'changed source fixture module')
    for name, original in expected.items():
        path = source / name
        require(path.is_file() and not path.is_symlink(), 'missing source fixture ' + name)
        require(dict(bytes=path.stat().st_size, sha256=driver.sha(path)) == declaration['fixtures'][original],
                'changed source fixture ' + name)


def admit(config, family, work):
    require(config['schema'] == 'gomad.installed-lifecycle-inputs/v1', 'unsupported lifecycle config')
    driver.check_files(config['files'])
    for path in (Path(config['binary']), Path(config['installation']), Path(__file__),
                 HERE / 'installed-lifecycle-corpus.json', Path(config['qualification_config'])):
        require(path.is_absolute() and config['files'].get(str(path)) == driver.sha(path), 'unsealed lifecycle input')
    frozen = driver.configured(Path(config['qualification_config']))
    frozen['_config_path'] = config['qualification_config']
    frozen['_config_sha256'] = driver.sha(Path(config['qualification_config']))
    driver.guard(frozen)
    role = live.role(frozen, 'release')
    require(config['binary'] == role['binary'] and config['binary'] == frozen['binary'],
            'lifecycle binary does not match frozen native candidate')
    require(not any(name.startswith(('GOMAD', 'QUALDRV')) for name in config.get('environment', {})),
            'lifecycle caller cannot inject private fixture controls')
    require(config['binary_role'] == 'release', 'installed lifecycle requires ordinary public release binary')
    require(config['repo'] == frozen['repo'], 'wrong native candidate checkout')
    root = Path(config['private_root'])
    require(root.is_absolute() and root.is_dir() and not root.is_symlink() and work.is_relative_to(root),
            'lifecycle work requires an owned private root')
    required = {'source', 'inputs', 'compiler_root', 'guest_config', 'preparation_cache'}
    if family != 'operator' or config['profile'] == 'gomad.wasi-cooperative/v1':
        required.add('runtime_root')
    if family not in ('operator', 'external-cluster'):
        required.add('compile_cache')
    if family != 'external-cluster':
        source_inputs(config, family)
        require(set(config['detach']) == {config[name] for name in required}, 'incomplete original unavailable inputs')
        for name in required:
            path = Path(config[name])
            require(path.is_relative_to(root) and path.exists() and not path.is_symlink(), 'unowned lifecycle input ' + name)
            if path.is_file():
                require(config['files'].get(str(path)) == driver.sha(path), 'unsealed private input ' + name)
            else:
                for child in path.rglob('*'):
                    require(not child.is_symlink(), 'symlink private input ' + name)
                    if child.is_file():
                        require(config['files'].get(str(child)) == driver.sha(child), 'unsealed private tree input ' + name)
    else:
        require(config['external_artifact'] in config['trees'] and
                config['trees'][config['external_artifact']] == tree(Path(config['external_artifact'])),
                'unsealed external retained cluster')
    installation = driver.read(config['installation'])
    if family not in ('external-cluster', 'kill-12'):
        guest = driver.read(config['guest_config'])
        require(installation['admission']['guest_configuration'] == guest, 'installation guest config differs from caller source')
        guest_base(guest, installation['profile'], family != 'operator')
        captured_input(guest, b'immutable operator input' if family == 'operator' else b'retained typed input')
    if family != 'external-cluster':
        require(installation['fuel'] == 4000000000 and installation['memory_bytes'] == 256 << 20,
                'original installed execution fuel/memory bounds changed')
        cache = Path(config['compile_cache'])
        require(cache.is_absolute() and cache.is_relative_to(root) and not cache.is_symlink(),
                'compiled cache must remain inside the owned private root')
    require(installation['host_sha256'] == frozen['identity']['host'] and
            installation['helper_sha256'] == frozen['identity']['helper'], 'installation candidate binding mismatch')
    if family != 'external-cluster':
        expected_input = b'immutable operator input' if family == 'operator' else b'retained typed input'
        if family not in ('kill-12', 'operator'):
            require(installation['admission']['environment'].get('GOMAD_REQUIRE_CAPTURED_INPUT') == '1',
                    'typed installation did not require original captured input')
        if family != 'kill-12':
            entries = installation['namespace']['entries']
            require(any(bytes(entry['path']) == b'/inputs/control' and bytes(entry['data']) == expected_input
                        and entry['readonly'] and not entry['directory'] for entry in entries),
                    'installation lost original captured control bytes')
        require(str(Path(installation['compiler_path']).parent.parent) == config['compiler_root']
                and installation['runtime_root'] == (config['runtime_root'] if 'runtime_root' in required else '')
                and installation['cache_root'] == config['preparation_cache'], 'installation private paths differ from caller inputs')
    for original, expected in driver.read(HERE / 'installed-lifecycle-corpus.json')['fixtures'].items():
        path = HERE.parents[3] / original
        require(path.is_file() and dict(bytes=path.stat().st_size, sha256=driver.sha(path)) == expected,
                'original retained fixture changed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--work', type=Path, required=True)
    parser.add_argument('--family', choices=('prompt', 'typed-detached', 'kill-12', 'kill-24', 'operator', 'external-cluster'), required=True)
    args = parser.parse_args()
    config = driver.read(args.config)
    args.work = args.work.resolve()
    admit(config, args.family, args.work)
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    run = Run(config, args.work)
    initial = check.parity.decode(run.command('discover', ['discover'])[0])
    if args.family != 'operator' or config['profile'] == 'gomad.wasi-cooperative/v1':
        discovery(initial)
    if args.family == 'prompt':
        run.prompt_cases()
    elif args.family == 'typed-detached':
        run.detached()
    elif args.family.startswith('kill-'):
        kill_resume(run, int(args.family.split('-')[1]))
    elif args.family == 'operator':
        operator(run)
    else:
        external_cluster(run)
    driver.write(args.work / 'result.json', dict(schema='gomad.installed-lifecycle-result/v1',
        family=args.family, config_sha256=driver.sha(args.config), actual_commands=run.index,
        evidence=tree(args.work)))
    print('installed lifecycle family completed:', args.family)


if __name__ == '__main__':
    main()
