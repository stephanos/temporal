#!/usr/bin/env python3
"""Prepare actual public Go cluster replay evidence and linked-host fixture inputs."""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess
import gzip
import struct
import sys


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def linked_input(reference):
    """Translate authenticated public boot inputs, never retained execution outputs."""
    root = reference.parent
    receipt = json.loads(reference.read_bytes())
    retained = json.loads((root / 'retained.json').read_bytes())
    if receipt['schema'] == 'gomad.cluster-go-api-reference/v1':
        module_path = Path(receipt['module_path'])
    else:
        manifest = json.loads((root / 'manifest.json').read_bytes())
        artifact = Path(receipt['artifact'])
        for entry in manifest['files']:
            raw = (artifact / entry['path']).read_bytes()
            assert len(raw) == int(entry['size']) and sha(raw) == entry['sha256'].removeprefix('sha256:')
        module_path = artifact / manifest['target']['file']
    module = module_path.read_bytes()
    assert sha(module) == receipt['module_sha256']
    assert sha(canonical(retained)) == receipt['retained_sha256']
    spec = retained['spec']
    boots = {boot['id']: boot['request'] for boot in retained['boots']}
    nodes = {node['id']: node for node in spec['nodes']}
    overrides = {(item['handle']['node'], item['handle']['incarnation']): item['boot']
                 for item in retained['incarnation_boots'] or []}
    incarnations = []
    for step in retained['lifecycle']:
        if step['action'] not in ('start', 'restart'):
            continue
        handle = step['handle']
        boot = boots[overrides.get((handle['node'], handle['incarnation']), nodes[handle['node']]['boot'])]
        config = boot['Config']
        limits = config['Limits']
        identity = boot['Runtime']['Choice']['ExecutionIdentity']
        # The linked aggregate bound is independent of the retained per-stream choice bound.
        bound = dict(fuel=boot['Fuel'], memory_bytes=boot['MemoryBytes'], wall_millis=boot['Timeout'] // 1000000,
                     output_bytes=limits['OutputBytes'], evidence_bytes=1 << 30)
        storage = config.get('storage')
        if storage is not None:
            go_model = {'volumes': [{'ID': v['ID'], 'Path': v['Path'], 'CapacityBytes': v['CapacityBytes']}
                                    for v in storage['volumes']],
                        'limits': {'PendingOperations': storage['limits']['PendingOperations'],
                                   'Transitions': storage['limits']['Transitions']}}
            model_raw = json.dumps(go_model, separators=(',', ':')).encode()
            storage = {'descriptor': {'model': storage, 'model_identity': 'sha256:' + sha(model_raw),
                                      'return_image': False}, 'image': None, 'context': None}
        incarnations.append({
            'handle': handle, 'module': {'file': 'target/module.wasm', 'bytes': len(module),
                                        'sha256': receipt['module_sha256'], 'limit': 512 << 20},
            'limits': bound, 'arguments': config['Args'],
            'environment': dict(item.split('=', 1) for item in config['Environment']),
            'runtime_sha256': bytes(identity['ImplementationSHA256']).hex(),
            'choice_identity': identity, 'network': config.get('network_v2'), 'storage': storage,
            'model': {'entropy_key': config['EntropyKey'], 'mode': 1, 'limit': boot['Runtime']['Choice']['Limit'],
                      'diagnostics': False, 'epoch_nanos': config['Clock']['EpochNanos'],
                      'read_step_nanos': config['Clock']['ReadStepNanos'], 'now': 1,
                      'working_directory_sha256': sha(config['WorkingDirectory'].encode()),
                      'arguments': {'bytes': 0, 'count': 0, 'chunks': 0},
                      'environment': {'bytes': 0, 'count': 0, 'chunks': 0},
                      'limits': {'calls': limits['Calls'], 'transcript_bytes': limits['TranscriptBytes'],
                                 'pending_events': limits['PendingEvents']}},
            'namespace': {'entries': [], 'writable_directories': [list(v.encode()) for v in config['WritableDirectories']],
                          'preopens': [list(v.encode()) for v in sorted(set(config['WritableDirectories']))],
                          'working_directory': list(config['WorkingDirectory'].encode()),
                          'stdin': list(base64.b64decode(config['Stdin'] or '', validate=True)),
                          'limits': {'files': limits['Files'], 'descriptors': limits['Descriptors'],
                                     'filesystem_bytes': limits['FilesystemBytes'], 'output_bytes': limits['OutputBytes']},
                          'epoch_nanos': config['Clock']['EpochNanos']}})
    transitions = spec['limits']['network_transitions']
    configuration = {'engine': 'cranelift-fuel-nan-canonical-nounwind-epoch-v1',
                     'profile': 'gomad.wasi-cooperative/v1', 'limits': incarnations[0]['limits'],
                     'broker': {'seed': spec['seed'], 'pause_on_settle': False,
                                'limits': {'max_requests': transitions * 2, 'max_events': transitions * 2,
                                           'max_queued_events': min(65536, transitions * 2),
                                           'max_transitions': transitions * 8,
                                           'max_payload_bytes': spec['limits']['observation_bytes'], 'max_string_bytes': 1024}},
                     'network': {'model': incarnations[0]['network'],
                                 'addresses': {node['id']: node['address'] for node in spec['nodes']},
                                 'links': spec.get('links') or [], 'strict_topology': True, 'faults': []},
                     'incarnations': incarnations,
                     'steps': [{'action': step['action'], 'handle': step['handle']} for step in retained['lifecycle']],
                     'boundaries': [{'action': action, 'handle': item['handle'],
                                     'marker': list(base64.b64decode(item['marker'], validate=True))}
                                    for key, action in [('output_crashes', 'crash'), ('output_stops', 'stop')]
                                    for item in retained[key] or []], 'crash_groups': []}
    return {'schema': 'gomad.cluster-fixture-input/v1',
            'toolchain_sha256': incarnations[0]['choice_identity']['ToolchainBuildKey'],
            'configuration': configuration,
            'modules': {'target/module.wasm': str(module_path)}}


def inspect_cluster(args, reference, artifact, run):
    receipt = json.loads(reference.read_bytes())
    api = receipt['schema'] == 'gomad.cluster-go-api-reference/v1'
    go_artifact = None if api else Path(receipt['artifact'])
    retained = json.loads((reference.parent / 'retained.json').read_bytes())
    record = json.loads(base64.b64decode(retained['record'], validate=True))
    metadata = json.loads((artifact / 'evidence/inspection').read_bytes())
    assert metadata == record['wasm_execution']['inspection'], 'same-target cluster inspection differs'
    assert b'reproduced=true' in run('rust-replay', [args.binary, 'replay', artifact], 1)
    run('rust-target-required', [args.binary, 'replay', '--stop=choice=0', artifact], 2)
    frames = base64.b64decode(metadata['boundary_frames'], validate=True)
    assert frames[:8] == b'GMWIB\x01\0\0'
    count, = struct.unpack_from('<I', frames, 8)
    boundaries, offset = {}, 12
    for _ in range(count):
        length, = struct.unpack_from('<I', frames, offset)
        frame = frames[offset + 4:offset + 4 + length]
        size, = struct.unpack_from('<I', frame)
        node = frame[4:4 + size].decode()
        incarnation, decision, trace, order, time, _, _ = struct.unpack_from('<7Q', frame, 4 + size)
        boundaries.setdefault((node, incarnation), []).append(dict(choice=decision, trace=trace, order=order, time_nanos=time))
        offset += 4 + length
    assert offset == len(frames)
    if not boundaries:
        flags = ['--instance=a', '--incarnation=1', '--stop=choice=0']
        rust = run('rust-no-boundary', [args.binary, 'replay', *flags, artifact], 2)
        go = run('go-no-boundary', [args.baseline / 'bin/gomad', '--installation', receipt['installation'],
                                   'replay', *flags, go_artifact], 2)
        assert not rust and not go
        return False
    assert len(boundaries) == 3
    for (node, incarnation), choices in boundaries.items():
        for label, boundary in [('first', choices[0]), ('last', choices[-1])]:
            name = f'{node}-{incarnation}-{label}'
            flags = [f'--instance={node}', f'--incarnation={incarnation}', f'--stop=choice={boundary["choice"]}']
            rust_dir, go_dir = args.work / ('rust-' + name), args.work / ('go-' + name)
            rust = json.loads(run('rust-inspect-' + name, [args.binary, 'replay', *flags, '--debug-output', rust_dir, artifact]))
            if api:
                go_dir = reference.parent
                go_debug = json.loads((go_dir / (name + '.debug.json')).read_bytes())
                go_debug['truncated'] = False
                go_heap = go_dir / (name + '.heap.pprof')
            else:
                go = json.loads(run('go-inspect-' + name, [args.baseline / 'bin/gomad', '--installation', receipt['installation'],
                                                          'replay', *flags, '--debug-output', go_dir, go_artifact]))
                for field in ('status', 'prefix_verified', 'witness_match', 'resolved', 'prefix_cuts', 'truncated'):
                    assert rust[field] == go[field], (name, field, rust[field], go[field])
                go_debug = json.loads((go_dir / 'debug.json').read_bytes())
                go_heap = go_dir / 'heap.pprof'
            assert rust['resolved'] == dict(instance=node, incarnation=incarnation, **boundary), rust
            assert rust['prefix_verified'] and rust['witness_match'] and rust['status'] == 'inspected_prefix'
            assert json.loads((rust_dir / 'debug.json').read_bytes()) == go_debug, name
            assert gzip.decompress((rust_dir / 'heap.pprof').read_bytes()) == gzip.decompress(go_heap.read_bytes()), name
    return True


def main():
    p = argparse.ArgumentParser()
    for name in ('baseline', 'source', 'compiler', 'work'):
        p.add_argument('--' + name, type=Path, required=True)
    p.add_argument('--binary', type=Path)
    p.add_argument('--from-reference', type=Path)
    p.add_argument('--api-registered', type=Path, help='fresh registered Go reference; exported API oracle only')
    args = p.parse_args()
    for name in ('baseline', 'source', 'compiler', 'work'):
        setattr(args, name, getattr(args, name).resolve())
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    environment = {key: value for key, value in os.environ.items()
                   if not key.startswith('GOMAD_') and key not in ('GOROOT', 'GOFLAGS', 'GOMOD', 'GOEXPERIMENT')}
    environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='0', GOPROXY='off', GOSUMDB='off')
    receipts = []

    def run(name, command, status=0, cwd=None):
        result = subprocess.run(list(map(str, command)), env=environment, capture_output=True, timeout=600, cwd=cwd)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                             stdout=result.stdout.decode(), stderr=result.stderr.decode()))
        (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
        assert result.returncode == status, (name, result.returncode, result.stderr)
        return result.stdout

    if args.from_reference:
        if args.api_registered:
            retained = json.loads((args.from_reference.parent / 'retained.json').read_bytes())
            registered = json.loads(args.api_registered.read_bytes())
            guest = Path(registered['records'][0]['artifact'])
            manifest = json.loads((guest / 'manifest.json').read_bytes())
            provenance = json.loads((guest / manifest['target']['backend']['provenance']['file']).read_bytes())
            base = retained['boots'][0]['request']
            network = base['Config'].get('network_v2')
            base['Config'] = provenance['Config']
            base['Config']['Args'] = manifest['target']['argv']
            base['Config']['Environment'] = [item['name'] + '=' + item['value'] for item in manifest['environment']]
            base['Config']['Environment'].append('PWD=' + base['Config']['WorkingDirectory'])
            base['Config']['network_v2'] = network
            base['ModulePath'] = str(guest / manifest['target']['file'])
            base['ModuleSHA256'] = manifest['target']['sha256'].removeprefix('sha256:')
            base['HelperPath'] = str(args.baseline / 'bin/gomad-wasmhost')
            base['Runtime']['Choice']['ExecutionIdentity'] = {
                'TargetSHA256': list(bytes.fromhex(base['ModuleSHA256'])),
                'ToolchainBuildKey': provenance['BuildKey'], 'GOOS': 'wasip1', 'GOARCH': 'wasm',
                'ImplementationSHA256': list(bytes.fromhex(manifest['choice_profile']['implementation_sha256'].removeprefix('sha256:')))}
            scenario = json.loads((args.from_reference.parent / 'durable-canary.json').read_bytes())
            scenario['spec'].pop('volumes', None)
            scenario['boots'] = [{'id': 'registered', 'args': []}]
            for node in scenario['spec']['nodes']:
                node['boot'] = 'registered'
                node.pop('volumes', None)
            scenario.pop('incarnation_boots', None)
            scenario.pop('output_crashes', None)
            scenario['actions'] = [{'action': action, 'handle': {'node': node, 'incarnation': incarnation}}
                                   for action, node, incarnation in [('start', 'a', 1), ('wait', 'a', 1),
                                                                    ('start', 'b', 1), ('wait', 'b', 1),
                                                                    ('restart', 'a', 2), ('wait', 'a', 2)]]
            input_path = args.work / 'api-input.json'
            input_path.write_bytes(canonical({'Base': base, 'Scenario': scenario, 'Output': str(args.work / 'api')}))
            module = args.source / 'tools/gomad'
            wrapper = module / ('cmd/fn232-cluster-reference-' + sha(str(args.work).encode())[:12])
            wrapper.mkdir(mode=0o700)
            template = Path(__file__).parent / 'reference/replay-cluster.go.txt'
            (wrapper / 'main.go').write_bytes(template.read_bytes())
            executable = args.work / 'go-cluster-reference'
            run('api-build', [args.compiler, 'build', '-trimpath', '-o', executable, './' + str(wrapper.relative_to(module))], cwd=module)
            print(run('api-record-inspect', [executable, input_path]).decode())
            api_dir = args.work / 'api'
            receipt = {'schema': 'gomad.cluster-go-api-reference/v1', 'module_path': base['ModulePath'],
                       'module_sha256': base['ModuleSHA256'],
                       'retained_sha256': sha(canonical(json.loads((api_dir / 'retained.json').read_bytes()))),
                       'template_sha256': sha(template.read_bytes()), 'wrapper_sha256': sha(executable.read_bytes())}
            (api_dir / 'cluster-reference.json').write_bytes(canonical(receipt))
            return
        input_path = args.work / 'cluster-input.json'
        input_path.write_bytes(canonical(linked_input(args.from_reference.resolve())))
        assert args.binary, '--from-reference requires --binary'
        bundle = json.loads(run('rust-produce', [args.binary, 'fixture-replay-cluster-input', '--root', args.work / 'rust',
                                                '--input', input_path]))
        artifact = args.work / 'rust' / bundle['records'][0]['artifact']
        positive = inspect_cluster(args, args.from_reference.resolve(), artifact, run)
        print(json.dumps({'artifact': str(artifact), 'same_target_cluster_inspection': positive}))
        return

    module = args.source / 'tools/gomad'
    specimen = module / 'internal/backends/wasm/command/testdata/scenario'
    scenario = json.loads((specimen / 'durable.json').read_bytes())
    for boot in scenario['boots']:
        if boot['id'] == 'recover':
            boot['args'] = ['canary']
    scenario_path = args.work / 'durable-canary.json'
    scenario_path.write_bytes(canonical(scenario))
    launcher = args.baseline / 'bin/gomad'
    installation = args.work / 'installation.json'
    run('install', [launcher, 'install', '--profile', 'gomad.wasi-cooperative/v1', '--compiler', args.compiler, '--helper', args.baseline / 'bin/gomad-wasmhost',
                    '--runtime-root', module / 'internal/model/runtimesource', '--config', specimen / 'config.json',
                    '--fuel', str(1 << 60), '--cache', args.work / 'preparation', '--output', installation])
    run('explore', [launcher, '--installation', installation, 'explore', '--scenario', scenario_path,
                    '--seeds=7', '--parallel=1', '--on-failure=all', '--working-dir', module,
                    '--build-tag=gomad_fixture', '--build-tag=test_dep', '--execution-timeout=30s', '--overall-timeout=5m',
                    '--artifacts', args.work / 'artifacts', 'go-run', './internal/backends/wasm/command/testdata/scenario'], 1)
    artifacts = [path.parent for path in (args.work / 'artifacts').rglob('manifest.json')
                 if json.loads(path.read_bytes()).get('artifact_kind') == 'gomad.target-failure/v1']
    assert len(artifacts) == 1, artifacts
    artifact = artifacts[0]
    manifest = json.loads((artifact / 'manifest.json').read_bytes())
    for entry in manifest['files']:
        raw = (artifact / entry['path']).read_bytes()
        assert len(raw) == int(entry['size']) and sha(raw) == entry['sha256'].removeprefix('sha256:'), entry['path']
    reference = manifest['target']['backend']['evidence']
    retained = json.loads((artifact / reference['file']).read_bytes())
    record_raw = base64.b64decode(retained['record'], validate=True)
    record = json.loads(record_raw)
    (args.work / 'cluster-record.json').write_bytes(record_raw)
    (args.work / 'retained.json').write_bytes(canonical(retained))
    (args.work / 'manifest.json').write_bytes(canonical(manifest))
    run('verify-refused', [launcher, '--installation', installation, 'replay', '--verify-only', artifact], 2)
    replay = run('replay', [launcher, '--installation', installation, 'replay', artifact], 1)
    assert b'reproduced=true' in replay, replay
    receipt = {'schema': 'gomad.cluster-go-replay-reference/v1', 'artifact': str(artifact),
               'installation': str(installation), 'go_binary_sha256': sha(launcher.read_bytes()),
               'helper_sha256': sha((args.baseline / 'bin/gomad-wasmhost').read_bytes()),
               'guest_source_sha256': sha((specimen / 'main.go').read_bytes()),
               'scenario_sha256': sha(scenario_path.read_bytes()), 'retained_sha256': sha(canonical(retained)),
               'record_sha256': sha(record_raw), 'module_sha256': manifest['target']['sha256'].removeprefix('sha256:')}
    (args.work / 'cluster-reference.json').write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps({'reference': str(args.work / 'cluster-reference.json'), 'artifact': str(artifact),
                      'record_fields': sorted(record), 'boot_fields': sorted(retained['boots'][0]['request'])}))


if __name__ == '__main__':
    main()
