#!/usr/bin/env python3
"""Run observed original broker cluster cells through the linked Rust fixture seam."""
import argparse
import base64
from copy import deepcopy
import hashlib
import importlib.util
import json
import re
from pathlib import Path
import subprocess
import struct
import sys

sys.dont_write_bytecode = True


def load(name):
    path = Path(__file__).with_name(name + '.py')
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


observe = load('live-broker-observe')
owner = load('live-broker-owner-check')
storage_model = load('storage-model')


def canonical(value):
    return json.dumps(value, separators=(',', ':'), ensure_ascii=False).encode()


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def b64(raw):
    return base64.b64decode(raw or '', validate=True)


def storage(request):
    model = request['Config'].get('storage')
    if model is None:
        return None
    descriptor = dict(model=model, model_identity='sha256:' + sha(storage_model.encode(model)), return_image=False)
    if request.get('StorageCrashLimits'):
        descriptor['crash_limits'] = request['StorageCrashLimits']
    if request.get('StorageCrashSelection') is not None:
        descriptor['crash_selection'] = request['StorageCrashSelection']
    context = request.get('StorageCrashContext')
    raw = canonical(context) if context is not None else None
    if raw is not None:
        descriptor['crash_context'] = dict(bytes=len(raw), chunks=(len(raw) + 65535) // 65536, sha256=sha(raw))
    return dict(descriptor=descriptor, image=None, context=list(raw) if raw is not None else None)


def linked_input(row):
    original = row['configuration']
    broker = original['Broker']
    boots = {item['ID']: item['Request'] for item in original['Instances']}
    boots.update(original.get('Boots') or {})
    incarnations, modules = [], {}
    for step in row['steps']:
        if step['action'] not in ('start', 'restart'):
            continue
        request = deepcopy(boots[step['boot']])
        request['Runtime'] = step['runtime']
        config = owner.configuration(request)
        module = Path(request['ModulePath'])
        raw = module.read_bytes()
        assert sha(raw) == request['ModuleSHA256'] and len(raw) <= 512 << 20
        if row['test'] == observe.ROOTS[39]:
            assert len(raw) <= 8 << 20
        name = 'target/' + request['ModuleSHA256'] + '.wasm'
        modules[name] = str(module)
        config['model']['now'] = 1
        config['namespace']['preopens'].sort()
        incarnations.append(dict(handle=dict(node=step['node'], incarnation=step['incarnation']),
             module=dict(file=name, bytes=len(raw), sha256=sha(raw), limit=512 << 20), limits=config['limits'],
             arguments=config['arguments'], environment=config['environment'], runtime_sha256=config['runtime_sha256'],
             model=config['model'], namespace=config['namespace'], storage=storage(request),
             network=request['Config'].get('network_v2'), choice_identity=request['Runtime']['Choice']['ExecutionIdentity']))
    network = None
    if broker['Network'] is not None:
        network = dict(model=broker['Network'], addresses=broker['Addresses'] or {}, strict_topology=broker['StrictTopology'],
            links=[],
            faults=[dict(name=fault['Name'], action=fault['Action'], **{'from': fault['From'], 'to': fault['To']},
                         at_nanos=fault['AtNanos'], delay_nanos=fault['DelayNanos'], symmetric=fault['Symmetric'])
                    for fault in broker['Faults'] or []])
        network['links'] = [dict(**{'from': item['From'], 'to': item['To']}, enabled=item['Enabled'], delay_nanos=item['DelayNanos'])
                            for item in broker['Links'] or []]
    limits = broker['Limits']
    boundaries = [dict(handle=dict(node=item['node'], incarnation=item['incarnation']), action=action,
                       marker=list(b64(item['marker_base64'])))
                  for field, action in (('OutputCrashes', 'crash'), ('OutputStops', 'stop')) for item in original[field] or []]
    groups = original.get('OutputCrashGroup')
    groups = [dict(nodes=groups['nodes'], marker=list(b64(groups['marker_base64'])))] if groups else []
    steps = []
    for step in row['steps']:
        if step['action'] == 'link':
            continue
        steps.append(dict(action=step['action'], handle=dict(node=step['node'], incarnation=step['incarnation'])))
    configuration = dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1', profile='gomad.wasi-cooperative/v1',
        limits=incarnations[0]['limits'], nodes=[node['ID'] for node in broker['Nodes']], world_profile=broker.get('WorldProfile', ''),
        broker=dict(seed=broker['Seed'], pause_on_settle=False,
                    limits=dict(max_requests=limits['maxRequests'], max_events=limits['maxEvents'], max_queued_events=limits['maxQueuedEvents'],
                                max_transitions=limits['maxTransitions'], max_payload_bytes=limits['maxPayloadBytes'], max_string_bytes=limits['maxStringBytes'])),
        network=network, incarnations=incarnations, steps=steps, boundaries=boundaries, crash_groups=groups)
    return dict(schema='gomad.cluster-fixture-input/v1', configuration=configuration, modules=modules,
                toolchain_sha256=incarnations[0]['choice_identity']['ToolchainBuildKey'])


def broker_rows(rows, runs, selected_roots):
    result = []
    for index, broker in enumerate(sorted(rows, key=lambda row: row['ordinal'])):
        root = broker['test'].split('/')[0]
        if root not in {observe.ROOTS[number] for number in selected_roots}:
            continue
        following = min([row['ordinal'] for row in rows if row['ordinal'] > broker['ordinal']], default=1 << 60)
        owners = [row for row in runs if broker['ordinal'] < row['ordinal'] < following and row['test'] == broker['test']]
        assert owners, (root, broker['ordinal'])
        states, instances, boots, starts, waits = [], [], {}, [], []
        for row in owners:
            node, incarnation = row['node']['ID'], row['node']['Incarnation']
            request = row['request']
            if root == observe.ROOTS[32] and node == 'a':
                calls = row['result']['GoFilesystemCalls']
                if row['remote_model']:
                    assert calls.get('path_open', 0) > 0
                else:
                    assert calls and set(calls) == {'fd_write'}, 'split owner served non-output filesystem call'
            boot = node if incarnation == 1 else 'recover'
            if incarnation == 1:
                instances.append(dict(ID=node, Request=request))
            else:
                boots[boot] = request
            starts.append(dict(action='start' if incarnation == 1 else 'restart', node=node, incarnation=incarnation,
                               boot=boot, runtime=request['Runtime']))
            waits.append(dict(action='wait', node=node, incarnation=incarnation))
            oracle = row['result']
            diagnostic = b64(oracle['InterruptedDiagnosticBytes']) if oracle['RuntimeInterrupted'] else b64(oracle['DiagnosticTrace']['Bytes'])[64:]
            states.append(dict(handle=row['node'], result=oracle, network_wire='',
                               streams=dict(diagnostic=base64.b64encode(diagnostic).decode()),
                               storage_image_identity=row['storage_image_identity']))
        starts.sort(key=lambda step: (step['incarnation'], step['node']))
        if root == observe.ROOTS[32]:
            assert len(starts) == 3
            steps = [starts[0], starts[1], dict(action='wait',node='a',incarnation=1), starts[2],
                     dict(action='wait',node='a',incarnation=2),dict(action='wait',node='b',incarnation=1)]
            marker = b'committed durable-a'
        else:
            assert len(starts) == 1
            steps = starts + waits
            marker = b'timer'
        result.append(dict(test=broker['test'], ordinal=broker['ordinal'], states=states, steps=steps,
             configuration=dict(Broker=broker['configuration'], Instances=instances, Boots=boots,
                OutputCrashes=[dict(node='a',incarnation=1,marker_base64=base64.b64encode(marker).decode())],
                OutputStops=[], OutputCrashGroup=None),
             result=dict(World=broker['world']), network_wire='', synthetic_owner=True))
    return result


def compare_storage_state(raw, state):
    assert raw[:8] == b'GMSTORE\x01'
    body = raw[:-40]
    assert struct.unpack_from('<Q', raw, len(body))[0] == len(body)
    assert hashlib.sha256(body).digest() == raw[-32:]
    version, count = struct.unpack_from('<II', body, 8)
    assert (version, count) in ((1, 5), (2, 6), (3, 6))
    offset, sections = 16, []
    for tag in range(1, count + 1):
        actual, length = struct.unpack_from('<IQ', body, offset)
        assert actual == tag
        offset += 12
        sections.append(body[offset:offset + length])
        offset += length
    assert offset == len(body)
    oracle = state['result']
    assert json.loads(sections[0]) == oracle['StorageSnapshots']
    assert json.loads(sections[1]) == oracle['StorageOperations']
    assert json.loads(sections[2]) == oracle['StorageCrashEnumeration']
    assert sections[3].decode() == state['storage_image_identity']


def raw_input(row, payload, work, index):
    controls = dict(initial_links=[dict(action=step['link_action'], **{'from': step['from'], 'to': step['to']},
                                       delay=step['delay'], symmetric=step['symmetric'])
                                  for step in row['steps'] if step['action'] == 'link'])
    if row['test'] == observe.ROOTS[35]:
        controls['cleanup'] = dict(mode='shutdown', reverse_insertion=row['reverse_insertion'])
    forcing = {}
    states = {(state['handle']['ID'], state['handle']['Incarnation']): state for state in row['states']}
    for step in row['steps']:
        if step['action'] not in ('start', 'restart') or step['runtime']['Choice']['Mode'] == 1:
            continue
        state = states[step['node'], step['incarnation']]
        result = state['result']
        source = step['runtime']['Choice'].get('Trace')
        trace = b64(source) if source else b64(result['InterruptedChoiceBytes'] if result['RuntimeInterrupted'] else result['ChoiceTrace']['Bytes'])
        path = work / f"forcing-{index}-{step['node']}-{step['incarnation']}.choices"
        path.write_bytes(trace)
        forcing[step['node'] + '/' + str(step['incarnation'])] = dict(trace_path=str(path),
            mode=3 if result['RuntimeInterrupted'] else step['runtime']['Choice']['Mode'],
            identity=step['runtime']['Choice']['ExecutionIdentity'])
    return dict(schema='gomad.live-cluster-observation/v1', configuration=payload['configuration'],
                modules=payload['modules'], controls=controls, forcing=forcing)


def forged_storage_cleanup(row):
    absent = [state for state in row['states']
              if state.get('collection', {}).get('status') == 'host_cleanup_without_result']
    if not absent:
        return None
    assert row['test'] in {observe.ROOTS[34] + f'/storage-crash/seed-{seed}' for seed in (7, 19, 42)}
    expected = {(state['handle']['ID'], state['handle']['Incarnation']): state for state in row['states']}
    assert len(row['states']) == len(expected) == 2 and set(expected) == {('a', 1), ('b', 1)}
    assert len(absent) == 1 and absent[0] is expected['b', 1]
    for state in row['states']:
        observe.validate_collection(state, row['result']['GroupGone'])
    assert expected['a', 1]['collection']['status'] == 'collected'
    original = expected['a', 1]['result']
    assert original['Termination'] == 'invalid' and original['Message'] == 'storage crash context diverged from prior observation'
    assert original['StorageImage'] is None
    assert row['result']['GroupGone'] is True and row['result']['Frames']['Results'] == 1
    assert [(step['action'], step['node'], step['incarnation']) for step in row['steps']] == [
        ('start', 'a', 1), ('start', 'b', 1), ('wait', 'a', 1)]
    instances = {item['ID']: item['Request'] for item in row['configuration']['Instances']}
    assert len(row['configuration']['Instances']) == 2 and set(instances) == {'a', 'b'}
    context = instances['a']['StorageCrashContext']
    assert set(context) == {'a-store'} and context['a-store']['PendingSHA256'] == 'sha256:' + '0' * 64
    return 'b', 1


def compare(output, row, process_exit=None):
    summary = json.loads((output / 'result.json').read_bytes())
    assert summary['schema'] == 'gomad.live-cluster-observation/v1' and summary['execution_returned']
    expected = {(item['handle']['ID'], item['handle']['Incarnation']): item for item in row['states']}
    uncollected = forged_storage_cleanup(row)
    if uncollected is None:
        assert len(summary['incarnations']) == len(expected)
    else:
        assert process_exit == 0, 'original host cleanup requires actual completed CLI process'
        assert summary['outcome']['termination'] == 'invalid'
        assert summary['outcome']['message'] == expected['a', 1]['result']['Message']
        assert [(step['action'], step['handle']['node'], step['handle']['incarnation']) for step in summary['executed_steps']] == [
            ('start', 'a', 1), ('start', 'b', 1), ('wait', 'a', 1)]
        handles = [(item['handle']['node'], item['handle']['incarnation']) for item in summary['incarnations']]
        assert len(handles) == len(set(handles)) and ('a', 1) in handles and set(handles) <= set(expected)
        image = output / 'incarnations/a/1/storage_image'
        assert not image.exists() or not image.read_bytes(), 'forged context produced a storage image'
    for item in summary['incarnations']:
        handle, actual = item['handle'], item['outcome']
        state = expected[handle['node'], handle['incarnation']]
        if (handle['node'], handle['incarnation']) == uncollected:
            assert actual['termination'], 'uncollected original peer cannot supply a phantom Rust result'
            continue
        result = state['result']
        assert actual['termination'] == result['Termination'], (row['test'], actual, result['Termination'])
        assert actual.get('exit_code') == result['ExitCode']
        assert actual['message'] == result['Message']
        assert item['interrupted'] == result['RuntimeInterrupted']
        assert result['Reaped']
        root = output / 'incarnations' / handle['node'] / str(handle['incarnation'])
        streams = dict(stdout=result['Stdout'], stderr=result['Stderr'], transcript=result['Transcript'], storage=result['StorageTranscript'],
                       choice=result['InterruptedChoiceBytes'] if result['RuntimeInterrupted'] else result['ChoiceTrace']['Bytes'],
                       network=state['network_wire'])
        streams.update(state['streams'])
        for stream, encoded in streams.items():
            raw = (root / stream).read_bytes() if (root / stream).exists() else b''
            assert raw == b64(encoded), (row['test'], handle, stream)
        if state.get('storage_image_identity'):
            compare_storage_state((root / 'storage_state').read_bytes(), state)
        if b'continuation executed' in b64(result['Stdout']) + b64(result['Stderr']):
            raise AssertionError('revoked guest continuation executed')
        if row['test'] == observe.ROOTS[39]:
            assert item['interrupted'] and item['choice_bytes'] > 0
            assert actual['peak_memory_bytes'] > 0 and actual['fuel_remaining'] < 4000000000
            metrics = actual['metrics']
            assert metrics['execute_nanos'] > 0 and metrics['resource_usage_available'] and metrics['peak_rss_bytes'] > 0
            assert summary['wall_elapsed_nanos'] > 0
            assert item['authorization_to_terminal_nanos'] > 0
    if uncollected is not None:
        return dict(kind='original-forged-storage-negative', uncollected_go_peer=dict(node='b', incarnation=1),
                    go_collected_results=1, rust_published_results=len(summary['incarnations']),
                    cleanup='actual CLI exit0 after linked execution returned', collected_network_compared=True,
                    unpublished_original_aggregate='no original final World/network assertion in this negative case')
    assert (output / 'streams/world').read_bytes() == b64(row['result']['World'])
    assert (output / 'streams/network').read_bytes() == b64(row['network_wire'])
    if 'Frames' in row['result']:
        assert row['result']['Frames']['GuestCallbacks'] == 0


def independent_stream_recording(row):
    starts = [step for step in row['steps'] if step['action'] in ('start', 'restart')]
    assert [(step['node'], step['incarnation']) for step in starts] == [('a', 1), ('b', 1)], 'root36 requires unique original Start a1/b1 controls'
    modes = []
    for step in starts:
        assert step['action'] == 'start', 'root36 requires initial guest starts'
        runtime = step.get('runtime')
        assert isinstance(runtime, dict), 'root36 requires explicit original runtime'
        choice = runtime.get('Choice')
        assert isinstance(choice, dict), 'root36 requires explicit original Choice'
        mode = choice.get('Mode')
        assert type(mode) is int and mode in (1, 2), 'root36 requires record or forced replay mode'
        modes.append(mode)
    assert modes[0] == modes[1], 'root36 guest replay modes differ'
    return modes[0] == 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'observation', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--roots', default='32,33,34,35,36,39')
    args = parser.parse_args()
    roots = [int(value) for value in args.roots.split(',')]
    assert roots and set(roots) <= {32, 33, 34, 35, 36, 39}
    work, observations = args.work.resolve(), args.observation.resolve()
    work.mkdir(mode=0o700, parents=True, exist_ok=False)
    observe.validate_events((observations / 'go-test.jsonl').read_bytes(), roots)
    derivation = json.loads((observations / 'derivation.json').read_bytes())
    assert sha((observations / 'test-only-helper.go').read_bytes()) == derivation['helper_sha256']
    assert sha((observations / 'test-only.diff').read_bytes()) == derivation['diff_sha256']
    index = json.loads((observations / 'observation-index.json').read_bytes())
    observe.validate_versions(index, derivation)
    assert index['derivation_sha256'] == sha((observations / 'derivation.json').read_bytes())
    assert index['go_test_sha256'] == sha((observations / 'go-test.jsonl').read_bytes())
    rows, brokers, runs = [], [], []
    for name, descriptor in index['files'].items():
        assert Path(name).name == name
        raw = (observations / 'observations' / name).read_bytes()
        assert len(raw) == descriptor['bytes'] and sha(raw) == descriptor['sha256']
        if name.endswith('.json'):
            observe.validate_row(json.loads(raw), index)
        if name.startswith('broker-'):
            brokers.append(json.loads(raw))
        if name.startswith('observation-'):
            runs.append(json.loads(raw))
        if name.startswith('cluster-'):
            row = json.loads(raw)
            if row['test'].split('/')[0] in {observe.ROOTS[root] for root in roots}:
                rows.append(row)
    rows.extend(broker_rows(brokers, runs, [root for root in roots if root in (32, 39)]))
    expected_counts = {32: 4, 33: 8, 34: 27, 35: 16, 36: 18, 39: 4}
    for root in roots:
        assert len([row for row in rows if row['test'].split('/')[0] == observe.ROOTS[root]]) == expected_counts[root]
    receipts, diversity = [], {}
    for index, row in enumerate(sorted(rows, key=lambda row: row['ordinal'])):
        payload = raw_input(row, linked_input(row), work, index)
        input_path = work / f'cell-{index}.json'
        input_path.write_bytes(canonical(payload))
        output = work / f'cell-{index}'
        command = [str(args.binary.resolve()), 'fixture-live-cluster', '--root', str(output), '--input', str(input_path)]
        result = subprocess.run(command, capture_output=True, timeout=180)
        (work / f'cell-{index}.stderr').write_bytes(result.stderr)
        receipts.append(dict(test=row['test'], command=command, exit=result.returncode))
        (work / 'commands.json').write_bytes(canonical(receipts))
        assert result.returncode == 0, (row['test'], result.stderr)
        receipts[-1]['comparison'] = compare(output, row, process_exit=result.returncode)
        (work / 'commands.json').write_bytes(canonical(receipts))
        if row['test'].startswith(observe.ROOTS[36] + '/') and independent_stream_recording(row):
            fixture = row['test'].split('/')[-1]
            raw = (output / 'incarnations/a/1/choice').read_bytes()
            stdout = (output / 'incarnations/a/1/stdout').read_bytes()
            order = re.search(rb'(?m)^runnable order(?: [0-3]){4}$', stdout)
            assert order is not None, 'linked guest omitted runnable-order witness'
            assert b'fresh heap 1' in stdout
            assert (b'stall expired' if fixture == 'stall' else b'ordered bytes ordered-byte') in stdout
            diversity.setdefault(fixture, []).append((sha(raw), order.group().decode()))
    if 36 in roots:
        assert set(diversity) == {'partition-heal-delay', 'stall', 'live-link'}
        for fixture, samples in diversity.items():
            assert len(samples) == 3 and len({sample[0] for sample in samples}) >= 2
            assert len({sample[1] for sample in samples}) >= 2
    (work / 'result.json').write_bytes(canonical(dict(schema='gomad.live-broker-cluster-check/v1', roots=roots,
        executions=len(receipts), diversity=diversity, binary_sha256=sha(args.binary.read_bytes()),
        root35_post_start_reinsertion='pending-private-live-unit' if 35 in roots else None,
        qualification=False)))


if __name__ == '__main__':
    main()
