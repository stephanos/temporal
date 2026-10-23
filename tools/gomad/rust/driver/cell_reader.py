#!/usr/bin/env python3
"""Authenticate a mandatory durable cell's raw execution, replay and domain oracles."""
import argparse
import base64
import copy
import hashlib
import json
from pathlib import Path
import sys

from driver import HERE, Refusal, command, configured, coverage, guard, read, require, sha, subgates
import adapters
import configure
import receiving
import sqlite

OWNED = {label for label, wanted in adapters.CELLS.items() if wanted['workload'] in ('sqlite', 'multi-instance')}


def normalized_scenario(target):
    # Mirrors campaign/scenario.rs serde defaults for these two fixed requests.
    value = json.loads(base64.b64decode(target['Scenario'], validate=True))
    for key in ('incarnation_boots', 'output_crashes', 'output_stops', 'network_faults'):
        value[key] = value.get(key) or []
    spec = value['spec']
    spec.setdefault('seed', 0)
    for key in ('links', 'volumes'):
        spec[key] = spec.get(key) or []
    for key in ('faults', 'exploration', 'scenario_choice_plan', 'replay'):
        spec.setdefault(key, None)
    for node in spec['nodes']:
        node.setdefault('config', None)
        node['volumes'] = node.get('volumes') or []
    for link in spec['links']:
        link.setdefault('enabled', False)
        link.setdefault('delay_nanos', 0)
    for boot in value['boots']:
        boot.setdefault('args', None)
        boot.setdefault('storage', None)
        if boot['storage'] is not None:
            for key in ('faults', 'sqlite_lock_databases'):
                if not boot['storage'].get(key):
                    boot['storage'].pop(key, None)
            for fault in boot['storage'].get('faults', []):
                if not fault.get('bytes'):
                    fault.pop('bytes', None)
    for fault in value['network_faults']:
        for key, default in (('AtNanos', 0), ('DelayNanos', 0), ('Symmetric', False)):
            fault.setdefault(key, default)
    return value


def prepared_configuration(specification, installation, runtime_sha256):
    # qualify/cells.rs options followed by campaign/prepare.rs configuration().
    campaign, target = specification['Campaign'], specification['Campaign']['Target']
    environment = dict(installation['admission']['environment'])
    environment.update(entry.split('=', 1) for entry in campaign['Environment'])
    working = bytes(installation['namespace']['working_directory'])
    environment.update(PWD=working.decode(), TZ='UTC', GOMAD_IO_PROFILE='wasm', GOMAD_CHOICE_PROFILE='gomad-choice-trace/v4')
    choice = max(1 << 20, min(campaign['ChoiceTraceLimit'], 64 << 20))
    model = copy.deepcopy(installation['model'])
    model.pop('seed', None)
    model.update(mode=1, limit=choice, diagnostics=False, working_directory_sha256=receiving.digest(working))
    namespace = copy.deepcopy(installation['namespace'])
    namespace['limits']['output_bytes'] = campaign['OutputLimit']
    nanos = campaign['ExecutionTimeout']
    result = dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1', profile=installation['profile'],
        limits=dict(fuel=installation['fuel'], memory_bytes=installation['memory_bytes'], wall_millis=(nanos+999999)//1000000,
                    output_bytes=campaign['OutputLimit'], evidence_bytes=choice+2*campaign['OutputLimit']+(128<<20)),
        arguments=['gomad-target', *target.get('Args', [])], wall_nanos=nanos, world_transition_bytes=campaign['WorldTransitionLimit'],
        environment=environment, derive_seed_environment=True, runtime_sha256=runtime_sha256, model=model, namespace=namespace,
        storage=installation.get('storage'))
    if installation.get('network') is not None:
        result['network'] = installation['network']
    return result


def requested_target(result, specification, installation):
    target = specification['Campaign']['Target']
    provenance = result['provenance']
    inputs = provenance['inputs']
    require(inputs['kind'] == target['Kind'] and inputs['source'] == target['Source']
            and inputs['directory'] == target['WorkingDir'] and inputs['arguments'] == target.get('Args', [])
            and inputs['tags'] == sorted(set(target.get('BuildTags', []) + installation.get('build_tags', [])))
            and inputs['compiler_path'] == installation['compiler_path']
            and inputs['runtime_implementation'] == installation['runtime_implementation_sha256'],
            'actual durable preparation differs from requested target/toolchain')
    scenario = normalized_scenario(target)
    raw = bytes(provenance['scenario'])
    require(json.loads(raw) == scenario and inputs['scenario'] == provenance['scenario']
            and result['manifest']['scenario_sha256'] == receiving.digest(raw),
            'actual durable scenario differs from original lifecycle/network/storage request')
    return scenario


def storage_identity(storage):
    # scenario.rs Storage serializes in this field order, with Go HTML escaping.
    ordered = dict(volumes=[{key: volume[key] for key in ('ID', 'Path', 'CapacityBytes')} for volume in storage['volumes']],
                   limits={key: storage['limits'][key] for key in ('PendingOperations', 'Transitions')})
    if storage.get('faults'):
        ordered['faults'] = [{key: fault[key] for key in ('name', 'resource', 'operation', 'occurrence', 'action', 'bytes')
                              if key in fault and (key != 'bytes' or fault[key])} for fault in storage['faults']]
    if storage.get('sqlite_lock_databases'):
        ordered['sqlite_lock_databases'] = storage['sqlite_lock_databases']
    raw = json.dumps(ordered, separators=(',', ':'), ensure_ascii=False)
    for char, escaped in (('&', r'\u0026'), ('<', r'\u003c'), ('>', r'\u003e'), ('\u2028', r'\u2028'), ('\u2029', r'\u2029')):
        raw = raw.replace(char, escaped)
    return 'sha256:' + receiving.digest(raw.encode())


def replay_scenario(result, scenario):
    replication = scenario['workload']['kind'] == 'independent-temporal-replication-durable-restart'
    base = result['provenance']['configuration']
    expected_handles = [step['handle'] for step in scenario['actions'] if step['action'] in ('start', 'restart')]
    for execution in result['report']['executions']:
        replay = read(Path(execution['artifact_path']) / 'evidence/cluster-replay.json')
        actual = replay['configuration']
        require(receiving.digest(receiving.canonical(actual)) == execution['record']['identity']['configuration']
                and actual['steps'] == scenario['actions'] and actual['workload'] == scenario['workload']
                and actual['nodes'] == [node['id'] for node in scenario['spec']['nodes']]
                and [item['handle'] for item in actual['incarnations']] == expected_handles,
                'actual durable replay changed original lifecycle/owners')
        expected_limits = dict(base['limits'])
        incarnations = sum(1 for step in scenario['actions'] if step['action'] in ('start', 'restart'))
        expected_limits['evidence_bytes'] = base['limits']['evidence_bytes'] * max(incarnations, 1)
        require(actual['engine'] == base['engine'] and actual['profile'] == base['profile']
                and actual['limits'] == expected_limits and actual.get('wall_nanos') == base.get('wall_nanos')
                and not any(key in actual for key in ('original_limits', 'oracle_plan', 'simulation_original', 'world_encoding', 'world_profile')),
                'actual durable replay changed engine/profile/limits/world policy')
        limits = scenario['spec']['limits']
        broker = dict(seed=7, pause_on_settle=False, limits=dict(
            max_requests=limits['network_transitions']*2, max_events=limits['network_transitions']*2,
            max_queued_events=min(65536, limits['network_transitions']*2), max_transitions=limits['network_transitions']*8,
            max_payload_bytes=limits['observation_bytes'], max_string_bytes=1024))
        if base.get('world_transition_bytes') is not None:
            broker['transition_bytes'] = base['world_transition_bytes']
        if replication:
            broker['limits'] = dict(max_requests=2<<20, max_events=2<<20, max_queued_events=250000,
                max_transitions=8<<20, max_payload_bytes=64<<20, max_string_bytes=(1<<32)-1)
        require(actual['broker'] == broker, 'actual durable replay changed original broker seed/bounds')
        expected_network = None
        if result['provenance']['network'] is not None:
            expected_network = dict(model=result['provenance']['network'], strict_topology=True,
                addresses={node['id']: node['address'] for node in scenario['spec']['nodes']}, links=scenario['spec']['links'],
                faults=[{key: fault[name] for key, name in (('name', 'Name'), ('action', 'Action'), ('from', 'From'),
                    ('to', 'To'), ('at_nanos', 'AtNanos'), ('delay_nanos', 'DelayNanos'), ('symmetric', 'Symmetric'))}
                    for fault in scenario['network_faults']])
        require(actual['network'] == expected_network, 'actual durable replay changed partition/heal/topology policy')
        boundaries = [dict(handle=item['handle'], action=action,
                           marker=list(base64.b64decode(item['marker'], validate=True)) if isinstance(item['marker'], str) else item['marker'])
                      for field, action in (('output_crashes', 'crash'), ('output_stops', 'stop')) for item in scenario[field]]
        require(actual['boundaries'] == ([] if replication else boundaries)
                and actual['crash_groups'] == ([dict(nodes=['active', 'standby'], marker=list(b'GOMAD_TEMPORAL_OBSERVATION '))] if replication else []),
                'actual durable replay changed original output-triggered grouped crash')
        module = Path(execution['artifact_path']) / 'target/module.wasm'
        expected_module = dict(file='target/module.wasm', bytes=module.stat().st_size,
            sha256=result['provenance']['module_sha256'], limit=512<<20)
        for item in actual['incarnations']:
            require(item['module'] == expected_module and item['imports'] == result['provenance']['module_metadata']['imports']
                    and item['runtime_sha256'] == base['runtime_sha256'] and item['choice_identity'] == result['provenance']['choice_identity']
                    and item.get('wall_nanos') == base.get('wall_nanos')
                    and not any(key in item for key in ('workload_kind', 'storage_crash_selection')),
                    'actual durable incarnation borrowed a foreign module/runtime/choice owner')
            handle = item['handle']
            node = next(node for node in scenario['spec']['nodes'] if node['id'] == handle['node'])
            boot_id = next((binding['boot'] for binding in scenario['incarnation_boots'] if binding['handle'] == handle), node['boot'])
            boot = next(boot for boot in scenario['boots'] if boot['id'] == boot_id)
            expected_model = copy.deepcopy(base['model'])
            environment = dict(base['environment'])
            if base.get('derive_seed_environment', False):
                environment['GOMADSEED'] = '7'
            if replication:
                role = handle['node']
                peer = '10.0.0.2' if role == 'active' else '10.0.0.1'
                expected_model.update(limit=64 << 20,
                    entropy_key=list(hashlib.sha256(('gomad-wasm-temporal-replication/v1/' + role).encode()).digest()))
                environment.update(GOMAD_TEMPORAL_CLUSTER=role,
                    GOMAD_TEMPORAL_PHASE='commit' if handle['incarnation'] == 1 else 'reopen',
                    GOMAD_TEMPORAL_REMOTE_FRONTEND=peer + ':7233', GOMAD_TEMPORAL_REMOTE_HTTP=peer + ':7243',
                    GOMAD_TEMPORAL_REMOTE_HISTORY=peer + ':7234')
            require(item['model'] == expected_model and item['environment'] == environment,
                    'actual durable incarnation model/environment differs from original policy')
            require(item['arguments'] == ([base['arguments'][0]] + boot['args'] if boot['args'] is not None else base['arguments'])
                    and item['namespace'] == base['namespace'] and item['limits'] == base['limits']
                    and item['network'] == result['provenance']['network'], 'actual durable incarnation input/policy differs')
            storage = copy.deepcopy(item['storage']['descriptor']['model'])
            for key in ('faults', 'sqlite_lock_databases'):
                if not storage.get(key):
                    storage.pop(key, None)
            for fault in storage.get('faults', []):
                if not fault.get('bytes'):
                    fault.pop('bytes', None)
            require(storage == boot['storage'], 'actual durable incarnation volume/fault model differs')
            descriptor = item['storage']['descriptor']
            controls = dict(model_identity=storage_identity(boot['storage']), return_image=True,
                crash_limits=None if replication else dict(States=32, Operations=8, Depth=8, Bytes=4<<20, Work=1<<16),
                crash_selection=None, crash_context=None, image=None)
            require(item['storage'].get('image') is None and item['storage'].get('context') is None
                    and set(item['storage']) == {'descriptor', 'image', 'context'}
                    and {key: value for key, value in descriptor.items() if key != 'model'} == controls,
                    'actual durable incarnation initial image/context/crash controls differ')


def sqlite_oracles(reader, result, success):
    for execution in result['report']['executions']:
        artifact = Path(execution['artifact_path'])
        ack = receiving.line((artifact / 'evidence/cluster/sqlite/2/stdout').read_bytes(), 'ACK ')
        recovered = (artifact / 'evidence/cluster/sqlite/3/stdout').read_bytes()
        integrity, raw_history = receiving.line(recovered, 'INTEGRITY '), receiving.line(recovered, 'HISTORY ')
        require(ack.isdecimal() and str(int(ack)) == ack and int(ack) > 0, 'invalid actual SQLite acknowledgment')
        rows = [row.split(':') for row in raw_history.split(',')]
        require(all(len(row) == 2 and row[0].isdecimal() and str(int(row[0])) == row[0]
                    and int(row[0]) > 0 and row[1] for row in rows), 'invalid actual SQLite raw history')
        require(len(ack.encode()) + len(b'committed') + len(recovered) <= 1 << 20, 'original SQLite oracle byte bound exceeded')
        checks = {'acknowledged-durable-recovery.events': sorted(row[0] for row in rows) == ['1', '2'],
                  'acknowledged-durable-recovery.integrity': integrity == 'ok',
                  'acknowledged-durable-recovery.acknowledged': ack == '2' and all(row[1] == 'committed' for row in rows)}
        require(read(artifact / 'evidence/workload-oracles.json') == dict(schema='gomad.rust-workload-oracles/v1', checks=checks)
                and all(checks.values()) == success, 'SQLite actual raw durable oracle differs from original cell')
    if success:
        native = sqlite.catalogue(result)
        reader.native_catalogue(result, native)
        for state in native['enumeration']['States']:
            receiving.native_state(state, native['context'], 'sqlite', 8)


def replication_oracles(result, success):
    for execution in result['report']['executions']:
        artifact = Path(execution['artifact_path'])
        checks, histories = {}, []
        for role in ('active', 'standby'):
            expected = receiving.history((artifact / f'evidence/cluster/{role}/1/stdout').read_bytes())
            actual = receiving.history((artifact / f'evidence/cluster/{role}/2/stdout').read_bytes())
            checks[role + '-durable-recovery.history'] = expected['history'] == actual['history']
            checks[role + '-durable-recovery.events'] = success and sorted(actual['event_ids']) == [1, 2, 3, 4, 5]
            checks[role + '-durable-recovery.identity'] = expected['cluster_id'] == actual['cluster_id']
            histories.append(actual['history'])
        checks['independent-replication-convergence'] = histories[0] == histories[1]
        require(read(artifact / 'evidence/workload-oracles.json') == dict(schema='gomad.rust-workload-oracles/v1', checks=checks)
                and all(checks.values()) == success, 'replication actual raw histories/oracle differ from original cell')


def collect(config, label, result_path):
    require(label in OWNED, 'unowned mandatory domain cell')
    guard(config)
    entry = config['stages'][label]
    root = Path(config['root']) / 'stages' / label
    wanted = adapters.CELLS[label]
    success = wanted['cell'] == 'success'
    reader = receiving.Retained(config)
    specification = read(entry['spec'])
    expected = configure.cell_spec(label, str(Path(config['root']) / 'cells' / label), config['source_manifest'],
                                   config['source_sha256'], bindings=config)
    require(specification == expected and config['files'].get(entry['spec']) == sha(entry['spec']), 'foreign mandatory cell request')
    reader.prepare_cell_source(config, root, label, specification)
    installation = config['installations'][entry['installation']]
    adapters.validate_budget(specification, read(installation), wanted)
    response = json.loads(reader.command(config, root, 'qualify', adapters.public(config, installation) + ['qualify', '--spec', entry['spec']],
                                         accepted=(0,) if success else (1,), private_observations=config))
    require(response['result_path'] == str(result_path), 'mandatory checker borrowed another actual Result')
    result = reader.adapter.result_report(config, root, 'report', str(result_path))
    manifest = result['manifest']
    for key, value in wanted.items():
        require(manifest[key] == value, 'wrong original mandatory cell identity: ' + key)
    require(result['observations']['operator_process_id'] == read(root / 'qualify.command.json')['pid'], 'mandatory checker borrowed producer PID')
    installed = read(installation)
    scenario = requested_target(result, specification, installed)
    require(result['provenance']['configuration'] == prepared_configuration(specification, installed,
            result['provenance']['configuration']['runtime_sha256']), 'actual durable preparation changed original model/limits/environment')
    replay_scenario(result, scenario)
    if wanted['workload'] == 'sqlite':
        sqlite_oracles(reader, result, success)
    else:
        replication_oracles(result, success)
    reader.verify_pending_preparations()
    rows = [dict(id=name, status='PASS', executed=True, exit=0) for name in subgates(label)]
    coverage(subgates(label), rows)
    guard(config)
    return dict(schema='gomad.rust-driver-check/v1', stage=label, source_sha256=config['source_sha256'], identity=config['identity'], subgates=rows)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--stage', choices=sorted(OWNED), required=True)
    parser.add_argument('--result', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config.update(_config_path=str(args.config.resolve()), _config_sha256=sha(args.config))
        print(json.dumps(collect(config, args.stage, args.result.resolve()), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, AssertionError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
