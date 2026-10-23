#!/usr/bin/env python3
"""Read the one frozen final-cycle stress matrix; never start a second matrix."""
import argparse
import base64
import stat
import copy
import hashlib
import json
import math
import os
from pathlib import Path
import re
import subprocess
import sys
import time

sys.dont_write_bytecode = True
AXES = ('cpu', 'concurrent', 'layout', 'handoff')
GATE = 'stress/41-cases-four-axes-original-repetitions'
REPO = Path(__file__).resolve().parents[4]


def require(condition, message):
    if not condition:
        raise ValueError(message)


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), allow_nan=False).encode()


def decode(raw):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            require(key not in result, 'duplicate JSON member: ' + key)
            result[key] = value
        return result
    return json.loads(raw, object_pairs_hook=unique,
        parse_constant=lambda value: (_ for _ in ()).throw(ValueError('nonfinite JSON: ' + value)))


def sha(path):
    with Path(path).open('rb') as handle:
        return hashlib.file_digest(handle, 'sha256').hexdigest()


def regular(path):
    path = Path(path)
    require(path.is_absolute() and '..' not in path.parts and path.is_file() and not path.is_symlink(),
            'missing/nonregular input: ' + str(path))
    require(path.resolve() == path, 'symlink input ancestor: ' + str(path))
    return path


def read(path):
    return decode(regular(path).read_bytes())


def pin(path, expected):
    path = regular(path)
    require(type(expected) is str and re.fullmatch('[0-9a-f]{64}', expected) and sha(path) == expected,
            'changed authenticated input: ' + str(path))
    return path


def sealed(config, path):
    return pin(path, config['files'].get(str(path)))


def within(root, path):
    path = regular(path)
    require(path.is_relative_to(root), 'raw evidence outside frozen attempt: ' + str(path))
    return path


def cases():
    return ([('runtime', index * 1000003) for index in range(32)] +
        [('two-cluster', seed) for seed in (7, 19, 42)] +
        [(name, seed) for name in ('workflow', 'activity') for seed in (0, 1000003, 2000006)])


def matrix(report, spec):
    require(report['schema'] == 'gomad.rust-stress/v1' and report['passed'] is True and report['rejected'] == [],
            'UNEXECUTED: nonpassing stress matrix')
    wanted = set(cases())
    refs = [(row['workload'], row['seed']) for row in report['references']]
    require(len(refs) == 41 and set(refs) == wanted, 'missing/duplicate original stress references')
    observed = [(row['workload'], row['seed'], row['axis']) for row in report['cells']]
    require(len(observed) == 164 and set(observed) == {(name, seed, axis) for name, seed in wanted for axis in AXES},
            'missing/duplicate/changed stress workload, seed or axis')
    positive = 0
    for cell in report['cells']:
        fanout = spec['parallel'] if cell['axis'] == 'concurrent' else 1
        count = spec['repetitions'] * fanout
        require(all(type(cell[key]) is int for key in ('rounds', 'fanout', 'requested', 'completed')) and
                cell['rounds'] == spec['repetitions'] and cell['fanout'] == fanout and cell['requested'] == count,
                'stress original repetitions/cohort changed')
        if cell['axis'] == 'handoff' and cell['workload'] != 'two-cluster':
            require(cell['status'] == 'skipped' and cell['completed'] == 0 and cell['runs'] == [] and
                    cell['comparisons'] == [] and cell['reasons'] ==
                    ['inapplicable: no cluster baton or remote peer replies in a standalone guest'],
                    'invalid standalone handoff disposition')
        else:
            require(cell['status'] == 'pass' and cell['completed'] == count and len(cell['runs']) == count and
                    len(cell['comparisons']) == count and cell['reasons'] == [], 'UNEXECUTED: short/failed stress cell')
            require(all(row == dict(schema='gomad.wasm-first-divergence/v1', status='identical')
                        for row in cell['comparisons']), 'divergent/undiagnosable stress comparison')
            positive += 1
    return 41, 164, positive


def event_order(raw):
    require(raw and len(raw) <= 32 << 20 and raw.endswith(b'\n'), 'missing complete event-order diagnostic')
    events, boundaries, counted, complete = [], [], 0, False
    header = False
    for line in raw.splitlines(keepends=True):
        require(not complete and line.startswith(b'gomad-wasm-event-order-v1 '), 'invalid event-order framing')
        frame = decode(line[len(b'gomad-wasm-event-order-v1 '):])
        kind = frame.get('type')
        for field in ('order', 'incarnation', 'ordinal', 'trace', 'decision', 'inspection_order', 'events', 'bytes'):
            if field in frame:
                require(type(frame[field]) is int and 0 <= frame[field] < 1 << 64, 'invalid event-order unsigned field')
        if 'time_nanos' in frame:
            require(type(frame['time_nanos']) is int and -(1 << 63) <= frame['time_nanos'] < 1 << 63, 'invalid event-order time')
        if kind == 'header':
            require(not header and frame == dict(type='header', schema='gomad.wasm-event-order/v1'), 'invalid event-order header')
            header = True
        elif kind == 'event':
            require(header and set(frame) == {'type', 'order', 'kind', 'instance', 'incarnation', 'ordinal', 'trace', 'decision', 'time_nanos', 'value'},
                    'invalid event-order event fields')
            require(frame['order'] == len(events) + 1 and len(events) < 131072 and frame['instance'] and
                    (frame['incarnation'] != 0 or frame['instance'] == 'controller') and frame['kind'] in
                    ('choice', 'draw-counter', 'clock', 'entropy', 'wasi-op', 'model-transition', 'output', 'world-order'),
                    'invalid event-order event')
            events.append(frame)
        elif kind == 'boundary':
            require(header and set(frame) == {'type', 'order', 'instance', 'incarnation', 'trace', 'decision', 'time_nanos', 'inspection_order'} and
                    frame['order'] == len(events) and frame['instance'] and frame['incarnation'] > 0 and len(boundaries) < 131072 and
                    (not boundaries or boundaries[-1]['inspection_order'] < frame['inspection_order'] and boundaries[-1]['order'] <= frame['order']),
                    'invalid event-order boundary')
            boundaries.append(frame)
        elif kind == 'complete':
            require(header and frame == dict(type='complete', events=len(events), bytes=counted), 'invalid event-order completion')
            complete = True
            continue
        else:
            raise ValueError('unknown event-order frame')
        counted += len(line)
    require(complete and events, 'missing complete event-order events')
    return events


def original_bounds(owner, workload):
    runtime, cluster = workload == 'runtime', workload == 'two-cluster'
    output = 1 << 20 if runtime else 8 << 20 if cluster else 256 << 20
    limits = owner['limits']
    namespace = owner['namespace']['limits']
    model = owner['model']
    require(limits['memory_bytes'] == (256 << 20 if runtime else 4 << 30) and
            limits['fuel'] == (4_000_000_000 if runtime else 1 << 50) and limits['output_bytes'] == output and
            owner['wall_nanos'] == (60_000_000_000 if runtime else 900_000_000_000), 'stress original guest bounds changed')
    require(namespace['output_bytes'] == output and namespace['files'] == (1000 if runtime else 10000 if cluster else 100000) and
            namespace['descriptors'] == (64 if runtime else 4096) and
            namespace['filesystem_bytes'] == (8 << 20 if runtime else 512 << 20), 'stress original namespace bounds changed')
    require(model['mode'] == 1 and 'seed' not in model and model.get('replay') is None and model.get('tape') is None and
            model['diagnostics'] is True and model.get('digest_only', False) is False and
            model['limits']['transcript_bytes'] == (64 << 20 if runtime else 128 << 20 if cluster else 1 << 30) and
            model['limits']['calls'] == (10000 if runtime else 4000000 if cluster else 1 << 28) and
            model['limits']['pending_events'] == 4096 and model['limit'] == (1 << 20 if runtime else 32 << 20) and
            (runtime or model['read_step_nanos'] == 0), 'stress original model bounds or fresh diagnostic mode changed')


def choices(raw):
    require(len(raw) % 96 == 0, 'malformed choice trace length')
    records, pending = [], []
    for ordinal in range(len(raw) // 96):
        row = raw[ordinal * 96:(ordinal + 1) * 96]
        kind, flags = row[8:10]
        readiness = int.from_bytes(row[10:12], 'big')
        alternatives, selected, data = [int.from_bytes(row[at:at + 4], 'big') for at in (12, 16, 20)]
        site, origin = int.from_bytes(row[24:32], 'big'), int.from_bytes(row[32:40], 'big')
        identity, alternative_set = row[32:64], row[64:96]
        observation, creation, missing = bool(flags & 2), kind == 4, bool(flags & 4)
        require(kind in (1, 2, 3, 4) and flags & ~7 == 0 and flags & 3 == (0 if creation else 2 if observation else 1) and
            observation == (kind == 3) and alternatives > 0 and selected < alternatives and (not missing or site == 0) and
            int.from_bytes(row[:8], 'big') == ordinal, 'malformed choice trace record')
        require(observation or creation or readiness == 0 and identity != bytes(32) and alternative_set != bytes(32), 'malformed choice decision identity')
        require(not observation or row[40:] == bytes(56) and origin <= ordinal and readiness & 1 and
            readiness >> 6 <= alternatives - int(bool(readiness & 2)), 'malformed select observation')
        require(not creation or flags & ~4 == 0 and 2 <= alternatives <= 256 and readiness == 0 and data == 0 and
            identity != bytes(32) and alternative_set == bytes(32), 'malformed creation site')
        record = dict(kind=kind, alternatives=alternatives, selected=selected, data=data, site=site, missing=missing,
                      observation=observation, origin=origin, identity=identity, alternative_set=alternative_set)
        if creation:
            require(selected == len(pending) and (not pending or alternatives == pending[0]['alternatives']), 'malformed creation site group')
            pending.append(record)
        elif pending:
            identities = [item['identity'] for item in pending]
            require(kind == 1 and len(pending) == alternatives == pending[0]['alternatives'] and
                all(a < b for a, b in zip(identities, identities[1:])) and alternative_set == hashlib.sha256(
                    b'gomad-choice-alternative-set/v1\0' + len(pending).to_bytes(8, 'big') + b''.join(identities)).digest(),
                'malformed creation alternative commitment')
            pending = []
        records.append(record)
    require(not pending, 'incomplete creation site group')
    assigned_polls = set()
    for ordinal, result in enumerate(records):
        if not result['observation'] or result['data'] < 2:
            continue
        polled = result['data'] - 1
        require(polled <= ordinal - result['origin'], 'invalid select result origin')
        for step in range(polled):
            poll_ordinal = result['origin'] + step
            require(poll_ordinal not in assigned_polls, 'overlapping select result readiness')
            assigned_polls.add(poll_ordinal)
            poll = records[poll_ordinal]
            require(poll['kind'] == 2 and not poll['observation'] and poll['site'] == result['site'] and
                poll['missing'] == result['missing'] and poll['alternatives'] == step + 2, 'invalid select result/poll relation')
    return {row['kind'] for row in records}


def runtime_oracle(payload):
    text = payload('output/stdout').read_bytes()
    for witness in (b'entropy ', b'map ', b'runq ', b'timer callbacks ', b'timer parking', b'note negative park wake',
                    b'collector automatic finalizer cleanup 1 1', b'canary '):
        require(witness in text, 'runtime stress lacks original output witness: ' + repr(witness))
    diagnostic = payload('evidence/diagnostic').read_bytes()
    trace = payload('evidence/choice').read_bytes()
    require(len(diagnostic) >= 96 and len(diagnostic) % 96 == 0 and len(trace) == len(diagnostic), 'runtime diagnostic trace incomplete')
    kinds = choices(trace)
    for ordinal in range(len(trace) // 96):
        row = diagnostic[ordinal * 96:(ordinal + 1) * 96]
        require(int.from_bytes(row[:8], 'big') == ordinal and row[28] <= 2 and row[29:32] == bytes(3) and row[36:40] == bytes(4),
                'runtime diagnostic record malformed')
    last = diagnostic[-96:]
    require({1, 2, 3} <= kinds and int.from_bytes(last[24:28], 'big') > 0 and
            all(int.from_bytes(last[at:at + 8], 'big') > 0 for at in (16, 40, 56, 64, 80)),
            'runtime lacks runnable/select, collector or seeded draw witnesses')


def cluster_oracle(configuration, payload):
    handles = {(item['handle']['node'], item['handle']['incarnation']) for item in configuration['incarnations']}
    require(len(configuration['incarnations']) == 4 and handles == {(node, incarnation) for node in ('active', 'standby') for incarnation in (1, 2)},
            'stress cluster lacks four original incarnations')
    expected_steps = [dict(action=action, handle=dict(node=node, incarnation=incarnation))
        for action, incarnation in (('start', 1), ('crash', 1), ('restart', 2), ('wait', 2)) for node in ('active', 'standby')]
    require(configuration['steps'] == expected_steps and configuration['boundaries'] == [] and configuration['crash_groups'] ==
        [dict(nodes=['active', 'standby'], marker=list(b'GOMAD_TEMPORAL_OBSERVATION '))], 'stress cluster original lifecycle changed')
    expected_network = dict(model=dict(profile='gomad.simulation-network-stream/v3',
        limits=dict(listeners=128, connections=4096, bytes=128 << 20, connection_bytes=4 << 20,
                    stall_nanos=10_000_000_000, transitions=1 << 20), faults=[]),
        addresses=dict(active='10.0.0.1', standby='10.0.0.2'), strict_topology=True,
        links=[dict(**{'from': a, 'to': b}, enabled=True, delay_nanos=0) for a, b in (('active', 'standby'), ('standby', 'active'))],
        faults=[dict(name='temporal-active-to-standby-' + name, action=action, **{'from': 'active', 'to': 'standby'},
                     at_nanos=at, delay_nanos=0, symmetric=False) for name, action, at in (('partition', 'partition', 1), ('heal', 'heal', 30_000_000_000))])
    require(configuration['network'] == expected_network, 'stress original cluster network topology/bounds changed')
    for owner in configuration['incarnations']:
        node, incarnation = owner['handle']['node'], owner['handle']['incarnation']
        environment = owner['environment']
        peer = '10.0.0.2' if node == 'active' else '10.0.0.1'
        require(all(environment[key] == value for key, value in dict(GOMAD_TEMPORAL_CLUSTER=node,
            GOMAD_TEMPORAL_PHASE='commit' if incarnation == 1 else 'reopen',
            GOMAD_TEMPORAL_REMOTE_FRONTEND=peer + ':7233', GOMAD_TEMPORAL_REMOTE_HTTP=peer + ':7243',
            GOMAD_TEMPORAL_REMOTE_HISTORY=peer + ':7234').items()), 'stress original cluster environment changed')
        storage = owner['storage']
        model = storage['descriptor']['model']
        require(model['volumes'] == [dict(ID=node, Path='/volume', CapacityBytes=256 << 20)] and
            model['limits'] == dict(PendingOperations=100000, Transitions=10000000) and
            model['sqlite_lock_databases'] == ['/volume/temporal.db'] and not model.get('faults') and
            storage.get('image') is None and storage.get('context') is None, 'stress original cluster durable inputs changed')
    summary = read(payload('evidence/cluster_summary'))
    require(summary['schema'] == 'gomad.cluster-observation/v1' and len(summary['incarnations']) == 4, 'missing stress cluster outcome facts')
    observed, histories = {}, {}
    for row in summary['incarnations']:
        node, incarnation = row['handle']['node'], row['handle']['incarnation']
        require((node, incarnation) in handles and (node, incarnation) not in observed, 'stress cluster duplicate/unknown outcome')
        prefix = 'evidence/cluster/{}/{}/'.format(node, incarnation)
        raw = payload(prefix + 'stdout').read_bytes()
        for stream, entry in row['streams'].items():
            item = payload(prefix + stream)
            require(item.stat().st_size == entry['bytes'] and sha(item) == entry['sha256'], 'stress cluster outcome stream mismatch')
        rows = [line[len(b'GOMAD_TEMPORAL_OBSERVATION '):] for line in raw.splitlines() if line.startswith(b'GOMAD_TEMPORAL_OBSERVATION ')]
        require(len(rows) == 1, 'stress cluster missing/duplicate Temporal observation')
        value = decode(rows[0])
        ids, types = value['event_ids'], value['types']
        history = base64.b64decode(value['history'], validate=True)
        require(history and value['cluster_id'] and len(ids) == len(types) and ids == list(range(1, len(ids) + 1)),
                'stress cluster lost/duplicate history events')
        require(len(encode([dict(id='committed', actor='workflow', kind='workflow-history', invocation=1, completion=2, output=value['history'])])) * 2 <= 1 << 20 and
                len(encode([str(item) for item in ids])) * 2 <= 1 << 20, 'stress cluster original history/event oracle bound exceeded')
        if incarnation == 1:
            require(row['termination'] == 'crash' and 'storage_image' in row['streams'],
                    'stress cluster commit lacks crashed durable state')
        else:
            require(row['termination'] == 'exit' and row['exit_code'] == 0, 'stress cluster reopen failed')
        observed[(node, incarnation)] = value
    for node in ('active', 'standby'):
        first, reopened = observed[(node, 1)], observed[(node, 2)]
        require(all(first[key] == reopened[key] for key in ('history', 'cluster_id', 'event_ids', 'types')), 'stress cluster history/identity changed after reopen')
        histories[node] = first
    require(histories['active']['cluster_id'] != histories['standby']['cluster_id'] and histories['active']['history'] == histories['standby']['history'],
            'stress cluster identities coincide or histories do not converge')


def artifact(root, observation, pool=None):
    path = Path(observation['artifact'])
    manifest = read(within(root, path / 'manifest.json'))
    require(within(root, path / 'storage.json').read_bytes() == b'{"format":"gomad.artifact-storage/v2"}', 'wrong stress artifact storage lineage')
    require((path / 'manifest.json').read_bytes() == encode(manifest), 'noncanonical stress manifest')
    expected = set(manifest['files']) | {'manifest.json', 'storage.json'}
    actual = set()
    for item in [path, *path.rglob('*')]:
        info = item.lstat()
        directory = stat.S_ISDIR(info.st_mode)
        require(info.st_uid == os.geteuid() and stat.S_IMODE(info.st_mode) == (0o700 if directory else 0o600) and
                (directory or stat.S_ISREG(info.st_mode)), 'stress artifact ownership/type/private mode changed')
        if not directory:
            name = str(item.relative_to(path))
            require(info.st_nlink == 1 or name == 'target/module.wasm', 'unexpected stress payload hard link')
            actual.add(name)
            if info.st_nlink > 1:
                require(pool is not None and pool.stat().st_uid == os.geteuid() and stat.S_IMODE(pool.stat().st_mode) == 0o700, 'module hard link lacks private frozen target pool')
                pooled = regular(pool / manifest['files'][name]['sha256']).stat()
                require((pooled.st_dev, pooled.st_ino) == (info.st_dev, info.st_ino), 'stress target pool inode mismatch')
    require(actual == expected, 'stress artifact inventory changed')
    record = manifest['record']
    require(set(record) == {'schema', 'identity', 'execution_identity', 'profile', 'seed', 'arguments', 'environment', 'limits', 'virtual_time_nanos', 'outcome', 'streams'},
            'missing/unknown typed execution record fields')
    require(type(record['profile']) is str and type(record['arguments']) is list and all(type(item) is str for item in record['arguments']) and
        type(record['environment']) is dict and all(type(key) is str and type(value) is str for key, value in record['environment'].items()) and
        type(record['virtual_time_nanos']) is int and 0 <= record['virtual_time_nanos'] < 1 << 64 and
        set(record['limits']) == {'fuel', 'memory_bytes', 'wall_millis', 'output_bytes', 'evidence_bytes'} and
        all(type(value) is int and 0 <= value < 1 << 64 for value in record['limits'].values()), 'malformed typed execution record')
    require(set(record['outcome']) == {'class', 'termination', 'exit_code', 'reason'} and
        all(type(record['outcome'][key]) is str for key in ('class', 'termination', 'reason')) and
        (record['outcome']['exit_code'] is None or type(record['outcome']['exit_code']) is int and 0 <= record['outcome']['exit_code'] < 1 << 32), 'malformed typed outcome')
    require(all(set(stream) == {'bytes', 'sha256', 'records'} and type(stream['sha256']) is str and re.fullmatch('[0-9a-f]{64}', stream['sha256']) and
        all(type(stream[key]) is int and 0 <= stream[key] < 1 << 64 for key in ('bytes', 'records')) for stream in record['streams'].values()), 'malformed typed record stream')
    require(record['schema'] == 'gomad.execution-record/v2' and type(record['seed']) is int and record['seed'] == observation['seed'] and
            set(record['identity']) == {'target', 'toolchain', 'platform', 'implementation', 'configuration'} and
            all(type(value) is str and re.fullmatch('[0-9a-f]{64}', value) for value in record['identity'].values()) and
            hashlib.sha256(encode(record['identity'])).hexdigest() == record['execution_identity'], 'raw stress seed/execution identity mismatch')
    require(manifest['schema'] == 'gomad.artifact-manifest/v2' and
            hashlib.sha256(encode(record)).hexdigest() == manifest['record_sha256'] == observation['record_sha256'],
            'raw stress canonical record mismatch')
    for name, entry in manifest['files'].items():
        relative = Path(name)
        require(name and not relative.is_absolute() and '..' not in relative.parts and name not in ('manifest.json', 'storage.json'),
                'unsafe stress payload path')
        payload = within(root, path / relative)
        require(type(entry['bytes']) is int and payload.stat().st_size == entry['bytes'] and sha(payload) == entry['sha256'],
                'raw stress payload changed: ' + name)
    require((path / 'record.json').read_bytes() == encode(record), 'noncanonical retained record bytes')
    require(record['streams'] == observation['streams'] and record['identity'] == observation['identities']['execution'],
            'stress observation differs from canonical artifact')
    for name, stream in record['streams'].items():
        require(name in manifest['files'] and all(stream[key] == manifest['files'][name][key] for key in ('bytes', 'sha256')),
                'stress record stream commitment changed')
    def payload(name):
        require(name in manifest['files'] and name in record['streams'], 'required stress payload is not committed: ' + name)
        return within(root, path / name)
    require(manifest['files']['target/module.wasm']['sha256'] == record['identity']['target'] and
            manifest['files']['target/module.wasm']['module'] is True and
            all(entry['module'] is (name == 'target/module.wasm') for name, entry in manifest['files'].items()), 'stress target module identity/kind changed')
    binding = read(payload('evidence/stress-inputs.json'))
    require(binding == dict(schema='gomad.stress-inputs/v1', fresh=True, seed=observation['seed'],
        workload=observation['workload'], source_sha256=observation['identities']['source']), 'stress execution binding changed')
    cluster = 'evidence/cluster-replay.json' in manifest['files']
    replay = read(payload('evidence/cluster-replay.json' if cluster else 'evidence/replay.json'))
    configuration = replay['configuration']
    require(replay['host_sha256'] == observation['identities']['host'] and
            replay['helper_sha256'] == observation['identities']['helper'] and replay['toolchain_sha256'] == record['identity']['toolchain'] and
            record['identity']['implementation'] == hashlib.sha256(encode(dict(host=replay['host_sha256'], helper=replay['helper_sha256']))).hexdigest(),
            'stress retained helper/host/toolchain implementation changed')
    require(hashlib.sha256(encode(configuration)).hexdigest() == record['identity']['configuration'], 'stress configuration identity changed')
    require(record['profile'] == configuration['profile'] and record['limits'] == configuration['limits'] and
        record['arguments'] == ([] if cluster else configuration['arguments']) and
        record['environment'] == ({} if cluster else configuration['environment']), 'stress record/configuration inputs differ')
    require(configuration['engine'] == observation['identities']['engine'] and configuration['engine'].endswith('-stress-jitter') and
            configuration['profile'] == 'gomad.wasi-cooperative/v1', 'stress artifact engine/profile role changed')
    models = configuration['incarnations'] if cluster else [configuration]
    require(models, 'missing stress models')
    for owner in models:
        original_bounds(owner, observation['workload'])
        if observation['workload'] != 'runtime':
            selection = {'workflow': '^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$', 'activity': '^TestActivityTestSuite$', 'two-cluster': '^TestIndependentClusterReplicationRecovery$'}[observation['workload']]
            require(owner['arguments'][1:] == ['-test.v', '-test.count=1', '-test.timeout=' + ('1h' if cluster else '72h'),
                '-test.run=' + selection, '-persistenceType=sql', '-persistenceDriver=sqlite'], 'stress original guest argv changed')
        model = owner['model']
        require(model['mode'] == 1 and 'seed' not in model and model.get('replay') is None and model.get('tape') is None and
                model['diagnostics'] is True, 'forced execution or missing diagnostics')
        prefix = 'evidence/cluster/{}/{}/'.format(owner['handle']['node'], owner['handle']['incarnation']) if cluster else 'evidence/'
        diagnostic = payload(prefix + 'diagnostic')
        choice = payload(prefix + 'choice')
        require(diagnostic.stat().st_size == choice.stat().st_size and diagnostic.stat().st_size % 96 == 0,
                'incomplete diagnostic trace')
    outcome = record['outcome']
    canary = observation['workload'] == 'runtime' and outcome['class'] == 'target_failure' and outcome['termination'] == 'exit' and outcome['exit_code'] == 17 and 'evidence/soft-assertions.json' not in manifest['files'] and b'canary 2\n' in payload('output/stdout').read_bytes()
    require(canary or outcome['class'] == 'success' and outcome['termination'] == 'exit' and outcome['exit_code'] == 0,
            'stress canonical outcome is unsuccessful')
    if observation['workload'] in ('workflow', 'activity'):
        text = payload('output/stdout').read_bytes()
        witness = b'--- PASS: TestWorkflowTypeEncodingSuite' if observation['workload'] == 'workflow' else b'--- PASS: TestActivityTestSuite'
        require(witness in text and b'--- SKIP:' not in text, 'stress original selection did not execute fully')
    if observation['workload'] == 'runtime':
        runtime_oracle(payload)
    if cluster:
        require(read(payload('evidence/stress-oracles.json')) == dict(passed=True, reason=None), 'stress cluster oracle failed')
        cluster_oracle(configuration, payload)
    projected = copy.deepcopy(record)
    projected['streams'].pop('evidence/event-order', None)
    return projected, event_order(payload('evidence/event-order').read_bytes())


def first_difference(expected, actual, path='$'):
    if type(expected) is not type(actual):
        return dict(path=path, expected=expected, actual=actual)
    if isinstance(expected, dict):
        for key in sorted(set(expected) | set(actual)):
            if key not in expected or key not in actual:
                return dict(path=path + '.' + key, expected=expected.get(key, '<absent>'), actual=actual.get(key, '<absent>'))
            found = first_difference(expected[key], actual[key], path + '.' + key)
            if found:
                return found
    elif isinstance(expected, (list, tuple)):
        for index in range(max(len(expected), len(actual))):
            if index >= len(expected) or index >= len(actual):
                return dict(path=path + '[' + str(index) + ']', expected=expected[index] if index < len(expected) else '<absent>',
                    actual=actual[index] if index < len(actual) else '<absent>')
            found = first_difference(expected[index], actual[index], path + '[' + str(index) + ']')
            if found:
                return found
    elif expected != actual:
        return dict(path=path, expected=expected, actual=actual)
    return None


def typed_stress_spec(spec):
    result = copy.deepcopy(spec)
    for key, value in (('only', ''), ('seed', ''), ('axis', ''), ('requested_output_root', None)):
        result.setdefault(key, value)
    for workload in result['workloads']:
        workload.setdefault('scenario', None)
    return result


def read_cycle(config, root):
    require(config['schema'] == 'gomad.stress-cycle/v1' and not config.get('synthetic', False), 'foreign/synthetic final cycle')
    fields = {'schema', 'repo', 'root', 'source_manifest', 'source_sha256', 'sent_epoch', 'deadline_s', 'identity', 'binary_sha256', 'stress', 'files'}
    require(set(config) in (fields, fields | {'dry_run'}), 'stress cycle plan has missing/unknown fields')
    require(Path(config['repo']) == REPO and Path(config['root']) == root, 'wrong frozen candidate or cycle root')
    require(type(config['deadline_s']) in (int, float) and math.isfinite(config['deadline_s']) and 0 < config['deadline_s'] <= 3600, 'changed original final-cycle deadline')
    require(type(config['sent_epoch']) in (int, float) and math.isfinite(config['sent_epoch']) and 0 < config['sent_epoch'] <= time.time(), 'invalid frozen stress send epoch')
    require(set(config['identity']) == {'host', 'helper'} and all(type(value) is str and re.fullmatch('[0-9a-f]{64}', value) for value in config['identity'].values()), 'invalid ordinary candidate identity')
    for name, expected in config['files'].items():
        pin(name, expected)
    sealed(config, config['source_manifest'])
    require(sha(config['source_manifest']) == config['source_sha256'], 'wrong frozen candidate source')
    source_paths = set()
    for line in regular(config['source_manifest']).read_text().splitlines():
        expected, name = line.split('  ', 1)
        require(Path(name).is_relative_to(REPO) and name not in source_paths, 'foreign/duplicate candidate source path')
        source_paths.add(name)
        pin(name, expected)
    require(str(Path(__file__).resolve()) in source_paths, 'candidate source omits stress reader')
    binding = config['stress']
    require(set(binding) == {'spec', 'binary', 'build_receipt', 'producer_root', 'producer_name'}, 'invalid frozen stress binding')
    for name in ('spec', 'binary', 'build_receipt'):
        sealed(config, binding[name])
    sys.path.insert(0, str(REPO / 'tools/gomad/rust/driver'))
    import dry_run
    import main_cli_build
    dry_run.enable(config)
    # The stress matrix executor is the stress-release role: only helper-stress-jitter, no fixture controls.
    build = main_cli_build.read_build(binding['build_receipt'], expected_role='stress-release')
    require((dry_run.is_record(binding['build_receipt']) or build['source_sha256'] == config['source_sha256']) and build['profile'] == 'release'
            and type(build['exit']) is int and build['exit'] == 0 and build['features'] == ['helper-stress-jitter']
            and build['helper_features'] == ['stress-jitter'] and build['binary_sha256'] == sha(binding['binary']), 'wrong stress candidate/build role')
    spec = read(binding['spec'])
    require(spec['schema'] == 'gomad.rust-stress-spec/v1' and spec['source_manifest'] == config['source_manifest'] and
            spec['source_sha256'] == config['source_sha256'] and spec['host_description'].strip() and
            all(type(spec[key]) is int and low <= spec[key] <= 256 for key, low in (('repetitions', 2), ('parallel', 2), ('cpu_workers', 1))),
            'wrong frozen stress spec or original bounds')
    require({row['name'] for row in spec['workloads']} == {'runtime', 'two-cluster', 'workflow', 'activity'} and len(spec['workloads']) == 4,
            'incomplete stress source workload binding')
    for workload in spec['workloads']:
        sealed(config, workload['installation'])
        installed = read(workload['installation'])
        require(installed['schema'] == 'gomad.rust-wasm-installation/v1' and installed['host_sha256'] == build['host_sha256'] and
                installed['helper_sha256'] == build['helper_sha256'] and installed['profile'] == 'gomad.wasi-cooperative/v1', 'stress installation uses wrong compiled role')
        sealed(config, installed['compiler_path'])
        require(sha(installed['compiler_path']) == installed['compiler_sha256'], 'stress installation compiler changed')
        if workload.get('scenario') is not None:
            sealed(config, workload['scenario'])
            scenario = read(workload['scenario'])
            require(scenario['spec']['limits'] == dict(nodes=64, directional_links=4096, volumes=64, scenario_actions=4096,
                scenario_decisions=4096, fault_actions=4096, history_operations=65536, observations=65536, oracle_results=4096,
                scenario_evidence_bytes=64 << 20, boot_config_bytes=1 << 20, volume_capacity_bytes=1 << 30, observation_bytes=64 << 20,
                network_listeners=4096, network_connections=65536, network_bytes=1 << 30, network_connection_bytes=4 << 20,
                network_stall_nanos=10_000_000_000, network_transitions=1 << 20, volume_operations=1 << 20, volume_transitions=1 << 20,
                crash_states=65536, crash_depth=256, crash_bytes=128 << 20, crash_wall_nanos=60_000_000_000), 'stress original scenario bounds changed')
    name = binding['producer_name']
    require(re.fullmatch('[a-zA-Z0-9_-]+', name), 'unsafe stress producer name')
    journal_root = Path(binding['producer_root'])
    journal = read(within(root, journal_root / (name + '.command.json')))
    process = read(within(root, journal_root / (name + '.process.json')))
    argv = [binding['binary'], 'stress', '--spec', binding['spec']]
    require(journal['argv'] == argv and journal['cwd'] == config['repo'] and type(journal['exit']) is int and journal['exit'] == 0 and
            journal['observation'] == 'completed' and type(journal['pid']) is int and journal['pid'] > 0 and
            process['active'] is False and not process.get('cleanup_signal') and type(process['stage_pid']) is int and process['stage_pid'] > 0,
            'UNEXECUTED: missing actual stress producer process')
    require(all(journal[key] == process[key] for key in ('argv', 'pid', 'start_ns', 'exit', 'observation')) and
            config['sent_epoch'] * 1e9 <= journal['start_ns'] <= process['end_ns'] <= journal['end_ns'] <=
            (config['sent_epoch'] + config['deadline_s']) * 1e9, 'stress process outside frozen cycle/bound')
    for stream in ('stdout', 'stderr'):
        within(root, journal_root / (name + '.' + stream))
    producer_stdout = decode(within(root, journal_root / (name + '.stdout')).read_bytes())
    require(set(producer_stdout) == {'report', 'passed'} and producer_stdout['passed'] is True, 'UNEXECUTED: nonpassing actual stress producer stdout')
    report_path = within(root, Path(producer_stdout['report']))
    declared_root = Path(spec['root'])
    report_root = report_path.parent
    require(report_path.name == 'report.json' and declared_root.is_relative_to(root) and
        (report_root == declared_root or report_root.parent == declared_root / '.gomad-measure/stress' and report_root.name.startswith('run-')),
        'matrix outside declared frozen stress output/retention root')
    retained_spec = read(within(root, report_root / 'spec.json'))
    require(retained_spec == typed_stress_spec(spec), 'stress ran a different spec')
    report = read(report_path)
    require(type(report['host']) is dict and report['host']['description'] == spec['host_description'] and report['host']['quiet_admission'] ==
        'operator assertion; CPU axis intentionally loads the host', 'stress quiet host admission changed')
    expected_platform = hashlib.sha256((report['host']['os'] + '/' + report['host']['arch']).encode()).hexdigest()
    require(report['source_sha256'] == config['source_sha256'] and report['binary_sha256'] == build['binary_sha256'] and
            report['command'] == argv, 'stress report belongs to wrong candidate/role')
    matrix(report, spec)
    require(decode(within(root, journal_root / (name + '.stdout')).read_bytes()) == dict(report=str(report_root / 'report.json'), passed=True),
            'stress producer stdout differs from retained report')
    pids, raw_commands = set(), [dict(argv=argv, status=journal['exit'], expected_status=0,
        stdout=str(journal_root / (name + '.stdout')), stderr=str(journal_root / (name + '.stderr')))]
    references = {(row['workload'], row['seed']): row for row in report['references']}
    authenticated = {}
    for cell in [dict(workload=row['workload'], seed=row['seed'], axis='reference', runs=[row]) for row in report['references']] + report['cells']:
        for index, run in enumerate(cell['runs']):
            require(run['fresh'] is True and all(run[key] == cell[key] for key in ('workload', 'seed', 'axis')) and
                    all(run[key] is None for key in ('oracle_error', 'event_order_error', 'perturbation_error')),
                    'forced/failed stress observation')
            prepared_root = report_root / ('prepared-' + cell['workload'])
            provenance_path = within(root, prepared_root / 'provenance.json')
            provenance = read(provenance_path)
            require(sha(provenance_path) == run['identities']['provenance'] and provenance['schema'] == 'gomad.rust-wasm-provenance/v1' and
                provenance['build_key'] == run['identities']['execution']['toolchain'] and
                provenance['module_sha256'] == run['identities']['execution']['target'] == sha(within(root, prepared_root / 'module.wasm')) and
                hashlib.sha256(encode(provenance['inputs'])).hexdigest() == provenance['build_key'] and
                provenance['inputs']['host'] == build['host_sha256'] and provenance['inputs']['helper'] == build['helper_sha256'],
                'stress actual preparation/module/toolchain identity changed')
            require(type(run['process_id']) is int and run['process_id'] > 0 and run['process_id'] not in pids,
                    'missing/reused stress process identity')
            pids.add(run['process_id'])
            identities = run['identities']
            require(identities['execution']['platform'] == expected_platform, 'stress record platform differs from actual report host')
            require(identities['source'] == config['source_sha256'] and identities['host'] == build['host_sha256'] and
                    identities['helper'] == build['helper_sha256'] and identities == references[(cell['workload'], cell['seed'])]['identities'],
                    'stress execution identity or role changed')
            executed = run['executed_command']
            require(len(executed) == 4 and executed[1:3] == ['stress', '--cell'], 'wrong stress child command')
            request_path = within(root, Path(executed[3]))
            request = read(request_path)
            run_root = request_path.parent
            require(read(within(root, run_root / 'observation.json')) == run and request['root'] == str(run_root) and
                    all(request[key] == run[key] for key in ('workload', 'seed', 'axis')) and request['source_sha256'] == config['source_sha256'],
                    'stress request/observation mismatch')
            require(sha(regular(executed[0])) == build['binary_sha256'] and run['binary'] == str(regular(executed[0])), 'wrong stress child executable')
            for stream in ('stdout', 'stderr'):
                within(root, run_root / (stream + '.log'))
            require(request['target_pool'] == str(report_root / '.targets') and request['options']['source'] == str(prepared_root / 'module.wasm') and
                    request['options']['provenance'] == str(provenance_path), 'stress child preparation owner changed')
            require(request['supervision'] == run['supervision'], 'stress request supervisor changed')
            options = request['options']
            timeout = 60 if cell['workload'] == 'runtime' else 900
            require(options['execution_timeout'] == dict(secs=timeout, nanos=0) and options['overall_timeout'] == dict(secs=timeout + 3600, nanos=0) and
                    options['choices'] is True and options['diagnostics'] is True and options['parallel'] == 1 and
                    options['soft_assertion_report_only'] is False and options['target_kind'] == 'exec', 'stress original execution options/bounds changed')
            sealed(config, options['installation'])
            conditions = run['host_conditions']
            require(conditions['executable'] == executed[0] and conditions['cwd'] == str(Path(executed[0]).parent),
                    'stress actual executable/layout cwd changed')
            if cell['axis'] == 'layout':
                repetition = run['supervision']['repetition']
                require(Path(conditions['cwd']).name == 'p' * (1 + repetition % 4 * 40) and
                        conditions['padding_bytes'] == 1 + repetition % 16 * 4096, 'stress layout perturbation changed')
            else:
                require(Path(conditions['cwd']).name == 'process' and conditions['padding_bytes'] == 0, 'unexpected stress layout condition')
            handoff = cell['axis'] == 'handoff'
            require(conditions['jitter'] is handoff and (not handoff or conditions['jitter_delay_calls'] > 0), 'handoff lacks actual host jitter delays')
            if cell['axis'] != 'reference':
                fanout = spec['parallel'] if cell['axis'] == 'concurrent' else 1
                require(run['supervision'] == dict(cpu_workers=spec['cpu_workers'] if cell['axis'] == 'cpu' else 0,
                    cohort_size=fanout, repetition=index // fanout, member=index % fanout), 'changed stress perturbation/cohort')
            view = artifact(root, run, report_root / '.targets')
            key = (cell['workload'], cell['seed'])
            if cell['axis'] == 'reference':
                authenticated[key] = view
            else:
                difference = first_difference(dict(record=authenticated[key][0], events=authenticated[key][1]), dict(record=view[0], events=view[1]))
                require(difference is None, 'first semantic stress difference: ' + json.dumps(dict(cell=[*key, cell['axis'], index], difference=difference), sort_keys=True))
            raw_commands.append(dict(argv=executed, process_id=run['process_id'], observation='completed-child',
                stdout=str(run_root / 'stdout.log'), stderr=str(run_root / 'stderr.log')))
    return report, raw_commands


def final_section(config_path, root, produce=False):
    sys.path.insert(0, str(REPO / 'tools/gomad/rust/driver'))
    import driver
    config = driver.configured(config_path)
    config['_config_path'] = str(config_path)
    config['_config_sha256'] = sha(config_path)
    driver.guard(config)
    require(root == Path(config['root']), 'wrong final attempt root')
    plan_path = sealed(config, config['stress_cycle'])
    plan = read(plan_path)
    for field in ('repo', 'root', 'source_manifest', 'source_sha256', 'sent_epoch', 'deadline_s', 'identity', 'binary_sha256', 'stress'):
        require(plan[field] == config[field], 'stress cycle plan differs from frozen config: ' + field)
    require(all(config['files'].get(name) == expected for name, expected in plan['files'].items()), 'stress cycle plan inputs differ from frozen config')
    if produce:
        binding = config['stress']
        for field in ('spec', 'binary', 'build_receipt'):
            sealed(config, binding[field])
        require(config.get('environment', {}).get('GOMAD_WASM_STRESS') == '1', 'stress opt-in must be frozen before execution')
        journal_root = Path(binding['producer_root'])
        require(journal_root.is_relative_to(root), 'stress producer outside final attempt')
        journal_root.mkdir(parents=True, exist_ok=False)
        remaining = config['sent_epoch'] + config['deadline_s'] - time.time()
        require(remaining > 0, 'original final-cycle deadline expired before stress launch')
        driver.command(config, journal_root, binding['producer_name'],
            [binding['binary'], 'stress', '--spec', binding['spec']],
            timeout=remaining)
    report, raw_commands = read_cycle(plan, root)
    actual_report = decode(regular(raw_commands[0]['stdout']).read_bytes())['report']
    driver.guard(config)
    return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
        identity=config['identity'], subgates=[dict(id=GATE, status='PASS', executed=True, exit=0,
            report=actual_report, matrix_sha256=hashlib.sha256(encode(report)).hexdigest())])


def parity(args):
    require(not sys.flags.optimize, 'Python optimization disables other required lane assertions')
    config_path = pin(args.stress_cycle, args.stress_cycle_sha256)
    config = read(config_path)
    require(sha(args.binary) == sha(config['stress']['binary']) and sha(args.ordinary_binary) == config['binary_sha256'] and
            sha(args.ordinary_binary) != sha(args.binary), 'parity binary differs from frozen matrix/ordinary candidate')
    report, commands = read_cycle(config, Path(config['root']))
    require(args.go_source.is_dir(), 'missing frozen Go source')
    for path in (args.go_binary, args.go_helper, args.go_compiler, args.input):
        regular(path)
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    retained, child_streams = [], []
    def save(name, row, stdout, stderr):
        for stream, raw in (('stdout', stdout), ('stderr', stderr)):
            relative = name + '.' + stream
            (args.work / relative).write_bytes(raw)
            row[stream] = relative
            row[stream + '_sha256'] = hashlib.sha256(raw).hexdigest()
        retained.append(row)
        (args.work / 'commands.json').write_bytes(encode(retained) + b'\n')
    for index, row in enumerate(commands):
        stdout, stderr = regular(row.pop('stdout')).read_bytes(), regular(row.pop('stderr')).read_bytes()
        if 'status' in row:
            save('matrix-' + str(index), row, stdout, stderr)
        else:
            for stream, raw in (('stdout', stdout), ('stderr', stderr)):
                relative = 'matrix-' + str(index) + '.' + stream
                (args.work / relative).write_bytes(raw)
                row[stream] = relative
                row[stream + '_sha256'] = hashlib.sha256(raw).hexdigest()
            child_streams.append(row)
    (args.work / 'matrix-streams.json').write_bytes(encode(child_streams) + b'\n')
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    environment['GOMAD_WASM_STRESS'] = '1'
    probes = [('go-availability', args.go_binary, ['stress'], (b'unknown command', b'unknown gomad command', b'unsupported command')),
              ('ordinary-exclusion', args.ordinary_binary, ['stress', '--spec', '/nonexistent-gomad-stress-spec'],
               (b'identity-bound helper-stress-jitter build',))]
    for name, binary, suffix, markers in probes:
        argv = [str(binary), *suffix]
        result = subprocess.run(argv, env=environment, capture_output=True, timeout=30, check=False)
        save(name, dict(argv=argv, status=result.returncode, expected_status=2), result.stdout, result.stderr)
        require(result.returncode == 2 and not result.stdout and any(marker in result.stderr for marker in markers),
                'stress availability/compiled role changed: ' + name)
    pin(config_path, args.stress_cycle_sha256)
    read_cycle(config, Path(config['root']))
    result = dict(schema='gomad.stress-semantic-parity/v1', status='passed',
        source_sha256=config['source_sha256'], identity=config['identity'], cycle=str(config['root']),
        stress_cycle_sha256=args.stress_cycle_sha256, matrix_sha256=hashlib.sha256(encode(report)).hexdigest(),
        availability=dict(go='unsupported', rust='compiled-stress-role'), references=41, cells=164, positive_cells=126,
        projection='canonical record excluding only evidence/event-order stream commitment; complete ordered diagnostic events compared independently',
        commands_sha256=sha(args.work / 'commands.json'), matrix_streams_sha256=sha(args.work / 'matrix-streams.json'))
    (args.work / 'stress-parity-result.json').write_bytes(encode(result) + b'\n')
    print(json.dumps(result, sort_keys=True))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--final-section', action='store_true')
    parser.add_argument('--produce', action='store_true')
    parser.add_argument('--config', type=Path)
    parser.add_argument('--root', type=Path)
    parser.add_argument('--stress-cycle', type=Path)
    parser.add_argument('--stress-cycle-sha256')
    for name in ('binary', 'ordinary-binary', 'go-binary', 'go-helper', 'go-source', 'go-compiler', 'input', 'work'):
        parser.add_argument('--' + name, type=Path)
    args = parser.parse_args()
    try:
        if args.final_section:
            require(args.config is not None and args.root is not None, 'final section requires --config and --root')
            print(json.dumps(final_section(args.config.resolve(strict=True), args.root.resolve(), args.produce), sort_keys=True))
        else:
            require(not args.produce and args.config is None and args.root is None, 'producer permitted only in final-section mode')
            require(all(value is not None for name, value in vars(args).items() if name not in
                ('produce', 'config', 'root', 'final_section')), 'UNEXECUTED: missing frozen stress cycle or parity inputs')
            parity(args)
        return 0
    except (ValueError, OSError, KeyError, TypeError, subprocess.SubprocessError) as error:
        print('stress parity refused: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
