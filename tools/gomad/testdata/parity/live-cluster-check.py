#!/usr/bin/env python3
"""Execute transferred cluster service lifecycles with retained source-less replay.

--input is an authenticated gomad.cluster-fixture-input/v1 for the unchanged
cluster/testdata/service guest, produced from its preparation inputs. Each cell
copies the executable into its own private input and removes that input before
replaying the retained artifact in a fresh process with no compiler on PATH.
"""
import argparse
from copy import deepcopy
import hashlib
import gzip
import struct
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
import shutil
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def provider_source():
    raw = (CORPUS / 'reference/live-broker-provider.go.txt').read_bytes()
    assert digest(raw) == 'bce7c6fe9e573eeea49be57d0882a8ad1da60bd23dbc7499d32c6253de7126c4', 'retained original provider source changed'
    return raw


def provider_reference(root):
    root = root.resolve()
    index_raw = (root / 'observation-index.json').read_bytes()
    index = json.loads(index_raw)
    assert index['schema'] == 'gomad.live-broker-observation-index/v1'
    derivation_raw = (root / 'derivation.json').read_bytes()
    assert index['derivation_sha256'] == digest(derivation_raw)
    derivation = json.loads(derivation_raw)
    assert derivation['roots'] == [19]
    assert derivation['oracle_role'] == 'original-private-go-broker-reference-and-original-one-helper-provider'
    assert digest(provider_source()) == derivation['original_test_sha256']
    for filename, key in (('test-only-helper.go', 'helper_sha256'), ('test-only.diff', 'diff_sha256')):
        assert digest((root / filename).read_bytes()) == derivation[key]
    events_raw = (root / 'go-test.jsonl').read_bytes()
    assert index['go_test_sha256'] == digest(events_raw)
    assert json.loads((root / 'command.json').read_bytes())['exit'] == 0
    name = 'TestProviderClusterStartsOnlyRequestedIncarnations'
    statuses = [event['Action'] for line in events_raw.splitlines()
                if (event := json.loads(line)).get('Test') == name
                and event['Action'] in ('pass', 'fail', 'skip')]
    assert statuses == ['pass'], 'exact Go provider assertion did not execute and pass'
    rows = []
    for filename, identity in index['files'].items():
        assert Path(filename).name == filename
        raw = (root / 'observations' / filename).read_bytes()
        assert len(raw) == identity['bytes'] and digest(raw) == identity['sha256']
        if filename.startswith('provider-') and filename.endswith('.json'):
            row = json.loads(raw)
            assert row['test'] == name
            rows.append(row)
    assert sorted(row['ordinal'] for row in rows) == [1, 2, 3, 4]
    opens = [row for row in rows if row['kind'] == 'open']
    assert len(opens) == 3
    reference = [row for row in opens if row['role'] == 'original-private-go-broker-reference']
    assert len(reference) == 1
    assert reference[0]['go_broker_reference'] and reference[0]['admitted'] and not reference[0]['error']
    assert reference[0]['spec']['fidelity'] == 'hard_isolation'
    ordinary = [row for row in opens if row['role'] == 'original-one-helper-provider' and row['admitted']]
    refused = [row for row in opens if row['role'] == 'original-one-helper-provider' and not row['admitted']]
    assert len(ordinary) == len(refused) == 1
    assert refused[0]['spec']['fidelity'] == 'hard_isolation' and 'hard isolation' in refused[0]['error']
    assert not ordinary[0]['error'] and not ordinary[0]['helper_started_at_open']
    assert ordinary[0]['lazy_cluster_request_at_open']
    assert [node['id'] for node in ordinary[0]['spec']['nodes']] == ['a', 'b' * 128]
    for row in opens:
        for request in row['boots'].values():
            raw = Path(request['ModulePath']).read_bytes()
            assert digest(raw) == request['ModuleSHA256'] and len(raw) <= 8 << 20
    finished = [row for row in rows if row['kind'] == 'finish']
    assert len(finished) == 1 and not finished[0]['error']
    incarnations = finished[0]['evidence']['Execution']['incarnations']
    assert len(incarnations) == 1 and incarnations[0]['handle'] == handle('a')
    assert finished[0]['frames']['Results'] == 1 and finished[0]['frames']['GuestCallbacks'] == 0
    return dict(role=derivation['oracle_role'], index_sha256=digest(index_raw),
                derivation_sha256=index['derivation_sha256'], command=str(root / 'command.json'),
                assertions=dict(private_reference_hard_isolation_admitted=True,
                    single_helper_hard_isolation_refused=True, ordinary_open_lazy=True,
                    only_requested_incarnation=True, results=1, go_guest_callbacks=0))


def snapshot(root):
    return {str(path.relative_to(root)): digest(path.read_bytes())
            for path in root.rglob('*') if path.is_file()}


def handle(node, incarnation=1):
    return dict(node=node, incarnation=incarnation)


def step(action, node, incarnation=1):
    return dict(action=action, handle=handle(node, incarnation))


def storage(node):
    model = dict(volumes=[dict(ID=node + '-store', Path='/workspace', CapacityBytes=8 << 20)],
                 limits=dict(PendingOperations=1024, Transitions=10000))
    return dict(descriptor=dict(model=model, model_identity='sha256:' + digest(
        json.dumps(model, separators=(',', ':')).encode()), return_image=False), image=None, context=None)


def configuration(template, cell):
    compressed = cell == 'compressed-lifecycle'
    if compressed:
        cell = 'lifecycle'
    config = deepcopy(template['configuration'])
    base = config['incarnations'][0]
    base['limits'] = dict(fuel=4_000_000_000, memory_bytes=256 << 20,
                          wall_millis=60_000, output_bytes=1 << 20, evidence_bytes=1 << 30)
    base['namespace'].update(working_directory=list(b'/workspace'),
        writable_directories=[list(b'/workspace'), list(b'/tmp')],
        preopens=[list(b'/tmp'), list(b'/workspace')], epoch_nanos=946684800000000000)
    base['namespace']['limits'] = dict(files=1024, descriptors=128,
        filesystem_bytes=8 << 20, output_bytes=1 << 20)
    base['environment'] = dict(PWD='/workspace', TZ='UTC')
    base['model'].update(mode=1, limit=8 << 20, diagnostics=True, now=1,
        epoch_nanos=946684800000000000, read_step_nanos=0,
        entropy_key=[1, 2, 3] + [0] * 29,
        working_directory_sha256=digest(b'/workspace'),
        limits=dict(calls=100000, transcript_bytes=32 << 20, pending_events=4096))
    base['model'].pop('seed', None)
    network = dict(profile='gomad.simulation-network-stream/v3',
        limits=dict(listeners=128, connections=128, bytes=8 << 20,
                    connection_bytes=4 << 20, stall_nanos=10_000_000_000, transitions=10000), faults=[])
    base['network'] = network
    config.update(original_limits=dict(network_transitions=1 << 20, observation_bytes=64 << 20), world_profile='', world_encoding='', limits=deepcopy(base['limits']), nodes=['a', 'b'],
        broker=dict(seed=7, pause_on_settle=False, limits=dict(max_requests=2 << 20,
            max_events=2 << 20, max_queued_events=65536, max_transitions=8 << 20,
            max_payload_bytes=64 << 20, max_string_bytes=1024)),
        network=dict(model=network, addresses={'a': '10.0.0.1', 'b': '10.0.0.2'},
            links=[],
            strict_topology=True, faults=[]), boundaries=[], crash_groups=[])
    config['network']['links'] = [{'from': a, 'to': b, 'enabled': True}
                                  for a, b in [('a', 'b'), ('b', 'a')]]

    def incarnation(node, number, mode, durable=False):
        item = deepcopy(base)
        item.update(handle=handle(node, number), arguments=['service', mode],
                    storage=storage(node) if durable else None)
        if mode == 'client':
            item['arguments'].append('10.0.0.1')
        return item

    if cell == 'lifecycle':
        config['incarnations'] = [incarnation(node, number, mode)
            for number in (1, 2) for node, mode in [('a', 'server'), ('b', 'client')]]
        config['steps'] = [step(action, node, number) for number in (1, 2)
            for action, node in [('start' if number == 1 else 'restart', 'a'),
                                 ('start' if number == 1 else 'restart', 'b'), ('wait', 'b'), ('wait', 'a')]]
    elif cell == 'grouped':
        config['incarnations'] = [incarnation(node, number, 'durable' if number == 1 else 'recover', True)
                                  for number in (1, 2) for node in ('a', 'b')]
        config['steps'] = [step(action, node, number) for action, number in
                           [('start', 1), ('crash', 1), ('restart', 2), ('wait', 2)] for node in ('a', 'b')]
        config['crash_groups'] = [dict(nodes=['a', 'b'], marker=list(b'committed durable-a'))]
    elif cell == 'durable':
        config['incarnations'] = [incarnation('a', 1, 'durable', True),
                                  incarnation('a', 2, 'recover', True), incarnation('b', 1, 'park')]
        config['steps'] = [step('start', 'a'), step('crash', 'a'), step('restart', 'a', 2),
                           step('start', 'b'), step('wait', 'a', 2), step('wait', 'b')]
        config['boundaries'] = [dict(action='crash', handle=handle('a'), marker=list(b'committed durable-a'))]
    elif cell.startswith('cleanup-'):
        config['nodes'] = ['a', 'b', 'dormant']
        config['network']['addresses']['dormant'] = '10.0.0.3'
        config['network']['links'] = []
        config['incarnations'] = [incarnation(node, 1, 'durable') for node in ('a', 'b')]
        config['steps'] = [step('start', 'a'), step('start', 'b')]
        if cell == 'cleanup-action-capacity':
            config['steps'].append(step('crash', 'a'))
    elif cell.startswith('unbound-'):
        config['nodes'] = ['a', 'dormant']
        config['network']['addresses'] = {'a': '10.0.0.1', 'dormant': '10.0.0.2'}
        config['network']['links'] = []
        config['incarnations'] = [incarnation('a', 1, 'durable', True), incarnation('a', 2, 'recover', True)]
        config['steps'] = [step('start', 'a'), step(cell.removeprefix('unbound-'), 'a'),
                           step('restart', 'a', 2), step('wait', 'a', 2)]
    elif cell == 'stop':
        config['incarnations'] = [incarnation('a', 1, 'durable'), incarnation('b', 1, 'park')]
        config['steps'] = [step('start', 'a'), step('start', 'b'), step('stop', 'a'), step('wait', 'b')]
        config['boundaries'] = [dict(action='stop', handle=handle('a'), marker=list(b'committed durable-a'))]
    elif cell == 'lazy':
        config['nodes'] = ['a', 'b' * 128]
        config['network'] = None
        item = incarnation('a', 1, 'observation')
        item['network'] = None
        config['incarnations'] = [item]
        config['steps'] = [step('start', 'a'), step('wait', 'a')]
    else:
        raise ValueError(cell)
    if compressed:
        config['world_encoding'] = 'gomad/world/compressed-snapshot/v2'
        config['world_profile'] = config['world_encoding']
        config['broker']['limits'] = dict(max_requests=2 << 20, max_events=2 << 20,
            max_queued_events=250000, max_transitions=8 << 20,
            max_payload_bytes=64 << 20, max_string_bytes=(1 << 32) - 1)
    return config


def check_artifact(artifact, cell, expected):
    compressed = cell == 'compressed-lifecycle'
    if compressed:
        cell = 'lifecycle'
    record = json.loads((artifact / 'record.json').read_bytes())
    assert record['outcome']['class'] == 'success', record['outcome']
    retained = json.loads((artifact / 'evidence/cluster-replay.json').read_bytes())
    assert retained['configuration']['steps'] == expected['steps']
    encoding = retained['configuration'].get('world_encoding', '')
    assert encoding == ('gomad/world/compressed-snapshot/v2' if compressed else ''), encoding
    world = (artifact / 'evidence/world').read_bytes()
    if expected.get('world_profile') is not None:
        assert world[:8] == b'GMWORLD2' and world[8:16] == struct.pack('>II', 64, 0)
        expanded_bytes, gzip_bytes = struct.unpack('>QQ', world[16:32])
        assert expanded_bytes <= 384 << 20 and gzip_bytes == len(world) - 64 <= 64 << 20
        expanded = gzip.decompress(world[64:])
        assert len(expanded) == expanded_bytes and digest(expanded) == world[32:64].hex()
        assert expanded[:8] == b'GMWSNAP1'
    if cell == 'lifecycle':
        assert len((artifact / 'evidence/cluster-replay.json').read_bytes()) <= len((artifact / 'record.json').read_bytes()) * 4 // 3 + (64 << 10), 'retention embeds executable bytes'
    summaries = json.loads((artifact / 'evidence/cluster_summary').read_bytes())['incarnations']
    assert len(summaries) == len(expected['incarnations']), summaries
    actual = {(row['handle']['node'], row['handle']['incarnation']): row for row in summaries}
    assert set(actual) == {(item['handle']['node'], item['handle']['incarnation'])
                           for item in expected['incarnations']}
    interrupted = {(row['handle']['node'], row['handle']['incarnation']): row['interrupted']
                   for row in retained['retained']}
    assert len(interrupted) == len(actual)
    for (node, number), row in actual.items():
        stdout = (artifact / f'evidence/cluster/{node}/{number}/stdout').read_bytes()
        assert b'fresh heap 1\n' in stdout, (cell, node, number, stdout)
        if cell in ('durable', 'grouped') and number == 2:
            assert b'fresh heap 1\nrecovered durable-a\n' in stdout, stdout
        if cell in ('durable', 'stop') and node == 'b':
            assert b'independent peer\n' in stdout, stdout
        if cell == 'lifecycle':
            assert b'ordered bytes ordered-byte\n' in stdout, stdout
        revoked = (cell in ('durable', 'stop') and node == 'a' and number == 1
                   or cell == 'grouped' and number == 1)
        assert interrupted[node, number] == revoked, (cell, row, interrupted)
        if revoked:
            assert b'committed durable-a\n' in stdout, stdout
            assert row['termination'] in ('stop', 'crash'), row
        else:
            assert row['termination'] == 'exit' and row['exit_code'] == 0, row
    return record


def check_raw(root, cell, expected):
    report = json.loads((root / 'result.json').read_bytes())
    assert report['schema'] == 'gomad.live-cluster-observation/v1'
    assert report['execution_returned'] and report['wall_elapsed_nanos'] > 0
    expected_steps = expected['steps'][:2] if cell == 'cleanup-action-capacity' else expected['steps']
    assert report['executed_steps'] == expected_steps, 'cleanup fabricated authored lifecycle actions'
    rows = report['incarnations']
    if cell == 'lazy':
        assert report['outcome']['termination'] == 'exit' and report['outcome']['exit_code'] == 0
        assert len(rows) == 1 and rows[0]['handle'] == handle('a'), rows
        assert rows[0]['outcome']['termination'] == 'exit' and rows[0]['outcome']['exit_code'] == 0
        assert report['activity_before_execution'] == dict(launched_stores=0, remote_callback_attempts=0)
        assert report['activity_after_execution'] == dict(launched_stores=1, remote_callback_attempts=0)
    elif cell.startswith('cleanup-'):
        assert len(rows) == 2 and {row['handle']['node'] for row in rows} == {'a', 'b'}
        if cell != 'cleanup-context-cancellation':
            assert all(row['interrupted'] for row in rows), rows
        assert report['outcome']['termination'] == {
            'cleanup-scenario-error': 'scenario_failure', 'cleanup-action-capacity': 'capacity',
            'cleanup-context-cancellation': 'cancelled',
        }[cell], report['outcome']
        stdout = (root / 'incarnations/b/1/stdout').read_bytes()
        assert b'fresh heap 1' in stdout, 'cleanup preceded actual acknowledgment'
    else:
        assert report['outcome']['termination'] == 'exit' and report['outcome']['exit_code'] == 0, report
        assert len(rows) == 2 and all(row['handle']['node'] == 'a' for row in rows), rows
        first, second = rows
        assert first['interrupted'] and first['outcome']['termination'] == cell.removeprefix('unbound-'), first
        assert second['handle']['incarnation'] == 2 and second['outcome']['exit_code'] == 0, second
        assert b'committed durable-a' in (root / 'incarnations/a/1/stdout').read_bytes()
        assert b'fresh heap 1\nrecovered durable-a' in (root / 'incarnations/a/2/stdout').read_bytes()
    return report


def admission_matrix(artifact, root, run):
    manifest = json.loads((artifact / 'manifest.json').read_bytes())
    retained = json.loads((artifact / 'evidence/cluster-replay.json').read_bytes())
    helper, primary = retained['helper_sha256'], manifest['record']['identity']['target']
    variants = [
        'missing-action', 'mutated-action', 'missing-module', 'foreign-module', 'boot-module-differs',
        'changed-limits', 'changed-environment', 'changed-helper-budget', 'changed-choice-budget',
        'malformed-storage-selection', 'changed-imports', 'missing-model-implementation',
        'changed-model-implementation', 'changed-world-profile', 'changed-world-encoding',
        'trailing-world-byte', 'invalid-world-header', 'corrupt-world-digest',
        'changed-runtime-binding', 'changed-original-record', 'changed-root-identity',
    ]

    def seal(path, payloads, change=None):
        current = json.loads((path / 'manifest.json').read_bytes())
        for name, raw in payloads.items():
            (path / name).write_bytes(raw)
            descriptor = dict(bytes=len(raw), sha256=digest(raw))
            current['files'][name].update(descriptor)
            if name in current['record']['streams']:
                current['record']['streams'][name].update(descriptor)
        if change:
            change(current)
        raw = canonical(current['record'])
        if 'record.json' in payloads:
            raw = payloads['record.json']
        (path / 'record.json').write_bytes(raw)
        current['record_sha256'] = digest(raw)
        current['files']['record.json'].update(bytes=len(raw), sha256=digest(raw))
        (path / 'manifest.json').write_bytes(canonical(current))

    observations = []
    for variant in variants:
        path = root / variant
        shutil.copytree(artifact, path)
        inputs = deepcopy(retained)
        config = inputs['configuration']
        boot = config['incarnations'][0]
        payloads = {}
        if variant == 'missing-action':
            config['steps'].pop()
        elif variant == 'mutated-action':
            config['steps'][0]['handle']['incarnation'] += 1
        elif variant == 'missing-module':
            inputs['configuration']['incarnations'][0]['module']['sha256'] = ''
        elif variant == 'foreign-module':
            boot['module']['sha256'] = '0' * 64
        elif variant == 'boot-module-differs':
            config['incarnations'][1]['module']['sha256'] = '0' * 64
        elif variant == 'changed-limits':
            config['original_limits']['network_transitions'] += 1
        elif variant == 'changed-environment':
            boot['namespace']['epoch_nanos'] += 1
            boot['model']['epoch_nanos'] += 1
        elif variant == 'changed-helper-budget':
            boot['wall_nanos'] = boot.get('wall_nanos', boot['limits']['wall_millis'] * 1_000_000) + 1
            boot['limits']['wall_millis'] = (boot['wall_nanos'] - 1) // 1_000_000 + 1
        elif variant == 'changed-choice-budget':
            boot['model']['limit'] += 1
        elif variant == 'malformed-storage-selection':
            boot['storage_crash_selection'] = {'unknown': [0, 0]}
        elif variant == 'changed-imports':
            boot['imports'][0]['name'] += '-changed'
        elif variant == 'missing-model-implementation':
            inputs['helper_sha256'] = ''
        elif variant == 'changed-model-implementation':
            inputs['helper_sha256'] = '0' * 64
        elif variant == 'changed-world-profile':
            config['world_profile'] = 'unknown'
        elif variant == 'changed-world-encoding':
            config['world_encoding'] = 'unknown'
        elif variant in ('trailing-world-byte', 'invalid-world-header', 'corrupt-world-digest'):
            world = bytearray((path / 'evidence/world').read_bytes())
            if variant == 'trailing-world-byte':
                world.append(0)
            else:
                world[0 if variant == 'invalid-world-header' else 32] ^= 1
            payloads['evidence/world'] = bytes(world)
        elif variant == 'changed-runtime-binding':
            boot['choice_identity']['ToolchainBuildKey'] += '-changed'
        elif variant == 'changed-original-record':
            payloads['record.json'] = (path / 'record.json').read_bytes() + b'\n'
        elif variant == 'changed-root-identity':
            pass
        payloads['evidence/cluster-replay.json'] = canonical(inputs)
        def change(current):
            if variant in ('changed-limits', 'malformed-storage-selection', 'changed-imports', 'changed-world-profile', 'changed-world-encoding', 'missing-action', 'mutated-action'):
                current['record']['identity']['configuration'] = digest(canonical(config))
                current['record']['execution_identity'] = digest(canonical(current['record']['identity']))
            if variant == 'changed-root-identity':
                current['record']['execution_identity'] = '0' * 64
        seal(path, payloads, change)
        before = snapshot(path)
        for operation in ('verify', 'storage-selections', 'reopen', 'replay'):
            empty = root / (variant + '-' + operation + '-private')
            empty.mkdir(mode=0o700)
            request = root / (variant + '-' + operation + '.json')
            request.write_bytes(canonical(dict(schema='gomad.live-cluster-admission/v1', artifact=str(path),
                operation=operation, helper_sha256=helper, primary_sha256=primary)))
            result = run('admit-' + variant + '-' + operation,
                         ['fixture-live-cluster', '--root', empty, '--input', request], status=2, detached=True)
            assert result.stdout == b'' and result.stderr, (variant, operation, result.stdout, result.stderr)
            assert list(empty.iterdir()) == [], 'rejection wrote executable/helper inputs'
            assert snapshot(path) == before, 'admission modified retained inputs'
            observations.append(dict(variant=variant, operation=operation, status=2, executable_writes=0))
    for label, changed in [('foreign-helper', dict(helper_sha256='f' * 64)),
                            ('missing-primary', dict(primary_sha256='')),
                            ('foreign-primary', dict(primary_sha256='0' * 64))]:
        empty = root / label
        empty.mkdir(mode=0o700)
        request = root / (label + '.json')
        value = dict(schema='gomad.live-cluster-admission/v1', artifact=str(artifact),
                     operation='replay', helper_sha256=helper, primary_sha256=primary)
        value.update(changed)
        request.write_bytes(canonical(value))
        run('admit-' + label, ['fixture-live-cluster', '--root', empty, '--input', request], status=2, detached=True)
        assert list(empty.iterdir()) == []
    assert len(observations) == 21 * 4
    (root / 'admissions.json').write_bytes(canonical(observations))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument('--input', type=Path)
    source.add_argument('--module', type=Path)
    parser.add_argument('--runtime-identity')
    parser.add_argument('--controller-binary', type=Path)
    parser.add_argument('--provider-reference', type=Path)
    parser.add_argument('--admissions', action='store_true')
    parser.add_argument('--cell', choices=('lazy', 'stop', 'grouped', 'durable', 'lifecycle', 'compressed-lifecycle', 'cleanup-scenario-error', 'cleanup-action-capacity', 'cleanup-context-cancellation', 'unbound-crash', 'unbound-stop'), action='append')
    args = parser.parse_args()
    args.binary, args.work = (path.resolve() for path in (args.binary, args.work))
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    reference = provider_reference(args.provider_reference) if args.provider_reference else None
    if args.input:
        template = json.loads(args.input.read_bytes())
    else:
        identity = args.runtime_identity
        assert identity and len(identity) == 64 and int(identity, 16), 'actual BuildOverlay identity required'
        module = args.module.resolve()
        raw = module.read_bytes()
        sha = digest(raw)
        runtime_sha = digest(identity.encode())
        template = dict(schema='gomad.cluster-fixture-input/v1', toolchain_sha256=identity,
            modules={'target/module.wasm': str(module)}, configuration=dict(
                engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',
                profile='gomad.wasi-cooperative/v1', incarnations=[dict(
                    module=dict(file='target/module.wasm', bytes=len(raw), sha256=sha, limit=512 << 20),
                    runtime_sha256=runtime_sha, choice_identity=dict(TargetSHA256=list(bytes.fromhex(sha)),
                        ToolchainBuildKey=identity, GOOS='wasip1', GOARCH='wasm',
                        ImplementationSHA256=list(bytes.fromhex(runtime_sha))),
                    namespace=dict(entries=[], stdin=[], limits={}),
                    model=dict(arguments=dict(bytes=0, count=0, chunks=0),
                               environment=dict(bytes=0, count=0, chunks=0)))]))
    assert template['schema'] == 'gomad.cluster-fixture-input/v1'
    loader = SourceFileLoader('live_cluster_authentication', str(CORPUS.parents[1] / 'scripts/behavioral-parity'))
    parity = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(parity)
    receipts = []
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}

    def run(name, arguments, status=0, detached=False, controller=False):
        marker = args.work / (name + '-guest-start')
        env = dict(environment, GOMAD_FIXTURE_GUEST_START_FILE=str(marker))
        if detached:
            env.update(PATH='/compiler/unavailable', GOMAD_STOCK_GO='/compiler/unavailable',
                       GOROOT='/runtime/unavailable', GOTOOLCHAIN='local')
        binary = args.controller_binary.resolve() if controller and args.controller_binary else args.binary
        command = [str(binary), *map(str, arguments)]
        result = subprocess.run(command, capture_output=True, timeout=180, env=env)
        receipts.append(dict(name=name, argv=command, status=result.returncode, expected=status,
            stdout=result.stdout.decode(errors='replace'), stderr=result.stderr.decode(errors='replace'),
            engine_entered=marker.exists(), detached=detached))
        (args.work / 'commands.json').write_bytes(canonical(receipts))
        assert result.returncode == status, receipts[-1]
        if status == 2:
            assert not marker.exists(), 'refusal entered guest execution'
        elif name.endswith(('-record', '-observe', '-detached-replay')):
            assert marker.read_bytes() == b'linked execution entered\n', 'missing positive engine-entry control'
        return result

    observations = []
    for cell in args.cell or ('lazy', 'stop', 'grouped', 'durable', 'lifecycle', 'compressed-lifecycle', 'cleanup-scenario-error', 'cleanup-action-capacity', 'cleanup-context-cancellation', 'unbound-crash', 'unbound-stop'):
        root = args.work / cell
        root.mkdir(mode=0o700)
        config = configuration(template, cell)
        executable = root / 'service.wasm'
        original = template['configuration']['incarnations'][0]['module']
        raw = Path(template['modules'][original['file']]).read_bytes()
        assert digest(raw) == original['sha256'] and len(raw) == original['bytes']
        assert len(raw) <= 8 << 20, 'original service module admission bound'
        executable.write_bytes(raw)
        executable.chmod(0o600)
        for item in config['incarnations']:
            item['module'] = dict(file='target/module.wasm', bytes=len(raw), sha256=digest(raw), limit=512 << 20)
            item['choice_identity']['TargetSHA256'] = list(bytes.fromhex(digest(raw)))
        request = dict(schema=template['schema'], toolchain_sha256=template['toolchain_sha256'],
            configuration=config, modules={'target/module.wasm': str(executable)})
        input_path = root / 'input.json'
        if cell == 'lazy' or cell.startswith(('cleanup-', 'unbound-')):
            controls = {}
            if cell.startswith('cleanup-'):
                controls['cleanup'] = dict(mode=cell.removeprefix('cleanup-').replace('-', '_'),
                    after=dict(handle=handle('b'), marker=list(b'fresh heap 1')))
                if cell == 'cleanup-action-capacity':
                    controls['action_limit'] = 2
            elif cell.startswith('unbound-'):
                controls['observations'] = [dict(handle=handle('a'), marker=list(b'committed durable-a'))]
            raw_request = dict(schema='gomad.live-cluster-observation/v1', configuration=config,
                               modules=request['modules'], controls=controls)
            input_path.write_bytes(canonical(raw_request))
            destination = root / 'observed'
            run(cell + '-observe', ['fixture-live-cluster', '--root', destination, '--input', input_path], controller=True)
            report = check_raw(destination, cell, config)
            executable.unlink()
            input_path.unlink()
            observations.append(dict(cell=cell, artifact=str(destination),
                incarnations=len(report['incarnations']), steps=len(report['executed_steps']), detached=False,
                activity_before_execution=report['activity_before_execution'],
                activity_after_execution=report['activity_after_execution']))
            continue
        input_path.write_bytes(canonical(request))
        destination = root / 'recording'
        run(cell + '-record', ['fixture-replay-cluster-input', '--root', destination, '--input', input_path])
        bundle = json.loads((destination / 'foundation-result.json').read_bytes())
        parity.authenticate(destination, bundle)
        assert len(bundle['records']) == 1
        artifact = destination / bundle['records'][0]['artifact']
        record = check_artifact(artifact, cell, config)
        before = snapshot(artifact)
        executable.unlink()
        input_path.unlink()
        assert not executable.exists() and not input_path.exists()
        retained = json.loads((artifact / 'evidence/cluster-replay.json').read_bytes())
        replay_private = root / 'replay-private'
        replay_private.mkdir(mode=0o700)
        replay_input = root / 'replay-admission.json'
        replay_input.write_bytes(canonical(dict(schema='gomad.live-cluster-admission/v1', artifact=str(artifact),
            operation='replay', helper_sha256=retained['helper_sha256'], primary_sha256=record['identity']['target'])))
        result = run(cell + '-detached-replay', ['fixture-live-cluster', '--root', replay_private,
                     '--input', replay_input], detached=True)
        replayed = json.loads(result.stdout)
        assert replayed['reproduced'] and replayed['wall_elapsed_nanos'] > 0, replayed
        assert replayed['record_sha256'] == digest((artifact / 'record.json').read_bytes()), replayed
        assert len(replayed['completed']) == len(config['incarnations']), replayed
        if cell == 'stop':
            stopped = next(row for row in replayed['completed'] if row['handle'] == handle('a'))
            assert stopped['interrupted'] and stopped['termination'] == 'stop', stopped
            assert stopped['metrics']['execute_nanos'] > 0, stopped
        assert list(replay_private.iterdir()) == [], 'linked replay wrote executable inputs'
        assert snapshot(artifact) == before, 'replay changed retained canonical record or payload'
        if cell in ('lifecycle', 'compressed-lifecycle'):
            missing_engine = deepcopy(config)
            missing_engine['engine'] = ''
            missing_engine_input = root / 'missing-engine.json'
            missing_engine_input.write_bytes(canonical(dict(schema='gomad.live-cluster-observation/v1',
                configuration=missing_engine, modules={'target/module.wasm': template['modules'][original['file']]})))
            absent_output = root / 'missing-engine-output'
            run(cell + '-missing-engine', ['fixture-live-cluster', '--root', absent_output,
                 '--input', missing_engine_input], status=2, detached=True)
            assert not absent_output.exists(), 'missing-engine refusal wrote executable output'
        if cell in ('lifecycle', 'compressed-lifecycle') and args.admissions:
            admission_root = root / 'admission-matrix'
            admission_root.mkdir(mode=0o700)
            admission_matrix(artifact, admission_root, run)
        observations.append(dict(cell=cell, artifact=str(artifact), record_sha256=digest(canonical(record)),
                                 incarnations=len(config['incarnations']), steps=len(config['steps']), detached=True))
    (args.work / 'observations.json').write_bytes(canonical(dict(
        schema='gomad.live-cluster-service-cells/v1', binary_sha256=digest(args.binary.read_bytes()),
        provider_reference=reference, cells=observations)))
    print('actual cluster service cells: ' + ', '.join(row['cell'] for row in observations))


if __name__ == '__main__':
    main()
