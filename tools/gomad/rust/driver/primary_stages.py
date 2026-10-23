#!/usr/bin/env python3
"""Closed primary recipes and independent raw readers for original live stage labels."""
import argparse
import copy
import hashlib
import json
import os
import stat
from pathlib import Path
import re
import sys

from driver import (HERE, Refusal, busy_host, command, configured, coverage,
                    guard, read, require, sha, subgates, write)
import live

OWNED = {'retained-durable-restart', 'busy-grouped', 'busy-boundaries', 'network-streams',
         'storage', 'network', 'temporal-inspection', 'artifact', 'operator', 'conformance-helper-build', 'terminal-controls', 'wasmmeasure-build', 'helper-build', 'ordinary-helper-publication'}
WORKLOAD_TOOLS = HERE.parent / 'workload_tools'
TIMEOUTS = {'retained-durable-restart': 900, 'busy-grouped': 1800, 'busy-boundaries': 1800,
            'network-streams': 1800, 'storage': 3600, 'network': 3600, 'temporal-inspection': 3600,
            'artifact': 900, 'operator': 1800, 'conformance-helper-build': 3600, 'terminal-controls': 900, 'wasmmeasure-build': 3600, 'helper-build': 3600, 'ordinary-helper-publication': 900}
MISSING = {
    'ordinary': 'complete original make test-wasm callable/subtest map, including duplicate selections',
    'cluster-provider': 'complete original 78 selected roots; decoder/writer/preflight/search/cleanup/model predicates beyond transferred roots18..28',
    'cluster-functional': 'complete original 46 selected roots; lifecycle/protocol/storage-context/codec/identity predicates beyond roots33..35 and private35',
    'retained-cluster': 'all original 24 Retained|Replay roots; independent replay/codec/reader/softassert/stress-reopen predicates beyond roots18,22,24,25,27',
}


def native_bootstrap(config):
    import native_roles
    binding = config.get('native_measurement', {})
    require(set(binding) == {'admission'}, 'sealed native measurement bootstrap admission required')
    live.sealed_input(config, binding['admission'])
    admission = native_roles.validate_admission(config, binding['admission'], dict(kind='release'))
    identity = admission['identity']
    require(identity['host_implementation_sha256'] == config['identity']['host']
            and identity['linked_implementation_sha256'] == config['identity']['helper'], 'native bootstrap differs from current candidate')
    return admission['executable']


def managed_native_run(root, response, producer, filename):
    require(set(response) == {'result', 'requested_root'} and response['requested_root'] == str(root / 'primary'),
            'native producer changed requested output root')
    path = Path(response['result'])
    command_receipt = read(root / 'primary-final.command.json')
    pid = command_receipt['pid']
    run = path.parent.parent
    require(path.name == filename and run.parent == root / 'primary/.gomad-measure' / producer,
            'native producer borrowed another managed store')
    match = re.fullmatch('run-' + str(pid) + r'-([0-9]+)-[0-9]+', run.name)
    require(match and command_receipt['start_ns'] <= int(match[1]) <= command_receipt['end_ns'],
            'native managed run has foreign PID or creation interval')
    require(run.resolve() == run and run.is_dir() and stat.S_IMODE(run.stat().st_mode) == 0o700
            and run.stat().st_uid == os.geteuid(), 'native managed run is not its private actual directory')
    for name, contents in (('.pin', str(run).encode()), ('.complete', b'')):
        item = run / name
        require(item.is_file() and not item.is_symlink() and item.read_bytes() == contents, 'native managed pin/completion changed')
    require(read(run.parent / 'latest.json') == run.name and (run / '.lease').is_file()
            and not (run / '.lease').is_symlink(), 'native completed managed run is not published')
    return path, run, pid


def native_build(config, root):
    import native_roles
    response = json.loads((root / 'primary-final.stdout').read_text())
    path, _, pid = managed_native_run(root, response, 'terminal-exit-helper-build', 'helper-build.json')
    require(re.fullmatch('release-native-' + str(pid) + r'-[0-9]+', path.parent.name), 'native build has foreign actual process owner')
    receipt = native_roles.validate_build(config, path, dict(kind='release'))
    require(receipt['identity']['host_implementation_sha256'] == config['identity']['host']
            and receipt['identity']['linked_implementation_sha256'] == config['identity']['helper'], 'fresh native build differs from current source')
    admissions = list(path.parent.glob('admitted-native-*/helper-admission.json'))
    require(len(admissions) == 1, 'missing/duplicate fresh native immutable admission')
    admission = native_roles.validate_admission(config, admissions[0], dict(kind='release'))
    require(admission['build_receipt'] == str(path), 'fresh native admission borrowed another build')
    return receipt, admission


def preflight(config, label, root):
    if label == 'ordinary-helper-publication':
        native_bootstrap(config)
        require(config['stages']['helper-build']['adapter'] == config['stages']['conformance-helper-build']['adapter'] == 'primary',
                'ordinary publication requires distinct actual helper/conformance producers')
        live.sealed_input(config, config['installations']['cooperative-runtime'])
    else:
        recipes(config, label, root)


def publish_native_files(config, root):
    import os
    original_root = Path(config['root']) / 'stages/conformance-helper-build'
    source, _ = conformance(config, original_root)
    destination = root / 'primary/conformance/gomad-wasmhost'
    destination.parent.mkdir(mode=0o700)
    with destination.open('xb') as output:
        os.chmod(destination, 0o755)
        output.write(Path(source).read_bytes())
    host = root / 'primary/ordinary-host/gomad'
    host.parent.mkdir(mode=0o700)
    with host.open('xb') as output:
        os.chmod(host, 0o755)
        output.write(Path(config['binary']).read_bytes())


def published_host(config, root):
    path = root / 'primary/ordinary-host/gomad'
    require(path.is_file() and not path.is_symlink() and path.resolve() == path and path.stat().st_nlink == 1
            and path.stat().st_mode & 0o777 == 0o755 and sha(path) == config['binary_sha256'],
            'ordinary installed host publication changed current executable bytes')
    return path


def published_discovery_command(config, reader, root, item):
    import receiving
    path = published_host(config, root)
    require(item['argv'][0] == str(path) and item['name'] == 'primary-discover', 'foreign derived discovery executable')
    # This local derived admission is never serialized into the frozen input table.
    derived = dict(config, files=dict(config['files'], **{str(path): config['binary_sha256']}))
    verifier = receiving.Retained(derived)
    verifier.command(derived, root, item['name'], item['argv'], timeout=TIMEOUTS['ordinary-helper-publication'])
    reader.commands.update(verifier.commands)


def native_publication(config, root):
    import native_roles
    _, original = native_build(config, Path(config['root']) / 'stages/helper-build')
    response = json.loads((root / 'primary-final.stdout').read_text())
    path, run, pid = managed_native_run(root, response, 'terminal-exit-helper-stage', 'helper-admission.json')
    require(re.fullmatch('admitted-native-' + str(pid) + r'-[0-9]+', path.parent.name), 'ordinary publication borrowed another producer')
    admission = native_roles.validate_admission(config, path, dict(kind='release'))
    require(admission['build_receipt'] == original['build_receipt'] and admission['executable'] != original['executable'],
            'ordinary publication did not copy the original fresh admitted helper')
    source_run = Path(original['build_receipt']).parent.parent
    pin = run / '.pins' / hashlib.sha256(str(source_run).encode()).hexdigest()
    require(pin.is_file() and not pin.is_symlink() and pin.samefile(source_run / '.pin'),
            'ordinary publication lost its actual managed source input pin')
    read_stage(config, 'conformance-helper-build', Path(config['root']) / 'stages/conformance-helper-build')
    source, digest = conformance(config, Path(config['root']) / 'stages/conformance-helper-build')
    copied = root / 'primary/conformance/gomad-wasmhost'
    ordinary = Path(admission['executable'])
    require(copied.is_file() and not copied.is_symlink() and sha(copied) == digest and copied.stat().st_mode & 0o777 == 0o755
            and copied.stat().st_nlink == ordinary.stat().st_nlink == 1
            and (copied.stat().st_dev, copied.stat().st_ino) != (ordinary.stat().st_dev, ordinary.stat().st_ino)
            and ordinary != Path(source) and sha(ordinary) != sha(copied), 'ordinary/conformance publication bytes or role separation changed')
    published_host(config, root)
    installation = config['installations']['cooperative-runtime']
    discovered = json.loads((root / 'primary-discover.stdout').read_text())
    require(set(discovered) == {'path', 'source', 'sha256', 'manifest', 'capabilities'}
            and discovered['path'] == installation and discovered['manifest'] == read(installation)
            and discovered['sha256'].removeprefix('sha256:') == sha(installation)
            and discovered['manifest']['helper_sha256'] == admission['identity']['linked_implementation_sha256']
            and discovered['manifest']['host_sha256'] == config['identity']['host'], 'installed ordinary discovery differs from actual published native role')


def canonical_argv(config, label, root, mode):
    require(mode in ('run', 'read') and label in OWNED, 'unknown primary recipe/mode')
    return [config['python'], str(HERE / 'primary_stages.py'), mode, '--config', config['_config_path'],
            '--stage', label, '--root', str(root)]


def recipes(config, label, root):
    require(label in OWNED, 'unknown primary stage: ' + label)
    require(root == Path(config['root']) / 'stages' / label, 'foreign primary stage root')
    python = config['python']
    live.sealed_input(config, python)
    work = root / 'primary'
    def recipe(name, script, flags):
        live.sealed_input(config, str(script))
        return dict(name=name, argv=[python, str(script), *flags])
    if label == 'ordinary-helper-publication':
        binary = native_bootstrap(config)
        build, _ = native_build(config, Path(config['root']) / 'stages/helper-build')
        receipt_path = Path(build['environment']['CARGO_TARGET_DIR']).parent / 'helper-build.json'
        return [dict(name='primary-final', argv=[binary, 'terminal-exit-helper-stage', '--repo', config['repo'],
                    '--terminal-exit-helper-stage-root', str(work), '--terminal-exit-helper-build-receipt', str(receipt_path)]),
                dict(name='primary-discover', argv=[str(work / 'ordinary-host/gomad'), '--installation', config['installations']['cooperative-runtime'], 'discover'])]
    if label in ('wasmmeasure-build', 'helper-build'):
        binary = native_bootstrap(config)
        return [dict(name='primary-final', argv=[binary, 'terminal-exit-helper-build', '--repo', config['repo'],
                                                '--terminal-exit-helper-build-root', str(work)])]
    if label == 'terminal-controls':
        return [recipe('primary-final', HERE / 'terminal_controls.py',
                       ['produce', '--config', config['_config_path'], '--root', str(work)])]
    if label == 'conformance-helper-build':
        plan_path = live.option(config['stages']['validate']['argv'], '--plan')
        require(plan_path, 'reviewed static tool plan required for conformance build')
        live.sealed_input(config, plan_path)
        plan = read(plan_path)
        require(plan['schema'] == 'gomad.driver-static-plan/v1' and plan['repo'] == config['repo']
                and plan['driver_config'] == config['_config_path'], 'foreign conformance tool plan')
        cargo, rustc = plan['tools']['cargo'], plan['tools']['rustc']
        live.sealed_input(config, cargo)
        live.sealed_input(config, rustc)
        target = work / 'target'
        return [dict(name='primary-final', argv=[cargo, 'build', '--manifest-path',
            str(Path(config['repo']) / 'tools/gomad/rust/wasmhost/Cargo.toml'), '--release', '--locked', '--offline',
            '--no-default-features', '--features', 'conformance', '--target-dir', str(target), '--message-format=json'],
            environment=dict(RUSTC=rustc, CARGO_HOME=plan['caches']['cargo_home'], CARGO_TARGET_DIR=str(target),
                             CARGO_BUILD_JOBS='1'))]
    if label in ('artifact', 'operator'):
        return [recipe('primary-final', HERE / 'final_retained.py', ['--config', config['_config_path'],
                       '--section', 'retained' if label == 'artifact' else 'operator', '--root', config['root']])]
    if label in ('retained-durable-restart', 'busy-grouped'):
        entry = config['live_corpus']['families']['cluster']
        binary = live.role(config, entry['role'])['binary']
        template = live.option(entry['argv'], '--input')
        require(template, 'actual unchanged cluster service template required')
        live.sealed_input(config, template)
        for path in read(template)['modules'].values():
            live.sealed_input(config, path)
        cell = 'durable' if label == 'retained-durable-restart' else 'grouped'
        count = 10 if label == 'busy-grouped' else 1
        return [recipe('primary-' + str(index), live.PARITY / 'live-cluster-check.py',
                ['--binary', binary, '--input', template, '--work', str(work / str(index)), '--cell', cell])
                for index in range(count)]
    if label in ('busy-boundaries', 'network-streams'):
        entry = config['live_corpus']['families']['broker']
        binary = live.role(config, entry['role'])['binary']
        reference = live.option(entry['argv'], '--observation')
        require(reference, 'actual authenticated original broker observations required')
        live.sealed_input(config, reference)
        roots = '33,34' if label == 'busy-boundaries' else '36'
        return [recipe('primary-final', live.PARITY / 'live-broker-cluster-check.py',
                ['--binary', binary, '--observation', reference, '--work', str(work / 'broker'), '--roots', roots])]
    inputs = config['primary']
    require(set(inputs) == {'compiler', 'runtime_root', 'cache', 'captured_config'}, 'unknown/missing primary workload inputs')
    for key in ('compiler', 'runtime_root', 'captured_config'):
        live.sealed_input(config, inputs[key])
    require(Path(inputs['cache']).is_absolute() and Path(inputs['cache']).is_dir()
            and not Path(inputs['cache']).is_relative_to(Path(config['repo'])), 'primary compiler cache must be external and assigned')
    common = ['--binary', config['binary'], '--repo', config['repo'], '--compiler', inputs['compiler'],
              '--runtime-root', inputs['runtime_root'], '--cache', inputs['cache'],
              '--captured-config', inputs['captured_config'], '--private-observation-config', config['_config_path']]
    if label == 'temporal-inspection':
        return [recipe('primary-final', WORKLOAD_TOOLS / 'temporal-inspection.py',
                       ['--owner', 'all', *common, '--root', str(work / 'inspection')])]
    workload = 'recovery' if label == 'storage' else 'retry'
    primary = work / workload
    first = recipe('primary-workload', WORKLOAD_TOOLS / 'producer.py',
                   [*common, '--root', str(primary), '--workload', workload, '--mode', 'success',
                    '--source-manifest', config['source_manifest']])
    if label == 'network':
        first['name'] = 'primary-final'
        return [first]
    return [first, recipe('primary-final', WORKLOAD_TOOLS / 'temporal-recovery.py',
                         ['--binary', config['binary'], '--repo', config['repo'], '--private-observation-config', config['_config_path'],
                          '--primary', str(primary), '--root', str(work / 'recovery-selected')])]


def endpoint(config, label):
    root = Path(config['root']) / 'stages' / label
    item = recipes(config, label, root)[-1]
    argv = item['argv']
    output = '--work' if '--work' in argv else '--root'
    return root, item, Path(live.option(argv, output))


def run(config, label, root):
    guard(config)
    require(not (root / 'primary').exists(), 'primary stage outputs must be fresh')
    (root / 'primary').mkdir(mode=0o700)
    if label == 'ordinary-helper-publication':
        publish_native_files(config, root)
    with busy_host(label):
        for item in recipes(config, label, root):
            observed = dict(config, environment=dict(config.get('environment', {}), **item.get('environment', {})))
            command(observed, root, item['name'], item['argv'], timeout=TIMEOUTS[label])
    guard(config)


def actual_commands(config, label, root):
    import receiving
    reader = receiving.Retained(config)
    reader.command(config, root, 'producer', canonical_argv(config, label, root, 'run'), timeout=TIMEOUTS[label])
    pids = []
    for item in recipes(config, label, root):
        if label == 'ordinary-helper-publication' and item['name'] == 'primary-discover':
            published_discovery_command(config, reader, root, item)
        else:
            reader.command(config, root, item['name'], item['argv'], timeout=TIMEOUTS[label])
        receipt = read(root / (item['name'] + '.command.json'))
        require(receipt['environment'] == dict(config.get('environment', {}), **item.get('environment', {})),
                'changed primary command environment')
        pids.append(receipt['pid'])
    require(len(pids) == len(set(pids)), 'primary repetitions borrowed producer PID')
    return reader


def retained_configuration(expected):
    """Apply the exact serde defaults of the retained Configuration and StorageDescriptor."""
    result = copy.deepcopy(expected)
    if result.get('world_encoding') == '':
        result.pop('world_encoding')
    for key in ('original_limits', 'oracle_plan', 'workload', 'simulation_original', 'world_profile', 'wall_nanos'):
        if key in result and result[key] is None:
            result.pop(key)
    for incarnation in result['incarnations']:
        if incarnation.get('workload_kind') == '':
            incarnation.pop('workload_kind')
        for key in ('imports', 'storage_crash_selection', 'wall_nanos'):
            if key in incarnation and incarnation[key] is None:
                incarnation.pop(key)
        storage = incarnation.get('storage')
        if storage:
            for key in ('image', 'context'):
                storage.setdefault(key, None)
            descriptor = storage['descriptor']
            for key in ('crash_limits', 'crash_selection', 'crash_context', 'image'):
                descriptor.setdefault(key, None)
            for key in ('faults', 'sqlite_lock_databases'):
                descriptor['model'].setdefault(key, None)
    if result.get('network'):
        for link in result['network']['links']:
            link.setdefault('delay_nanos', 0)
    return result


def cluster(config, label, root):
    check = live.load('live-cluster-check')
    common = live.load('live-backend-common')
    cell = 'durable' if label == 'retained-durable-restart' else 'grouped'
    for item in recipes(config, label, root):
        argv = item['argv']
        work = Path(live.option(argv, '--work'))
        template = read(live.option(argv, '--input'))
        binary = live.option(argv, '--binary')
        report = read(work / 'observations.json')
        require(report['schema'] == 'gomad.live-cluster-service-cells/v1' and report['binary_sha256'] == sha(binary)
                and len(report['cells']) == 1 and report['cells'][0]['cell'] == cell, 'missing/foreign primary cluster cell')
        expected = check.configuration(template, cell)
        module = template['configuration']['incarnations'][0]['module']
        import receiving
        module_path = template['modules'][module['file']]
        raw_module = Path(module_path).read_bytes()
        require(len(raw_module) == module['bytes'] and sha(module_path) == module['sha256'], 'changed sealed cluster service module')
        imports = receiving.module_metadata(raw_module)['imports']
        for incarnation in expected['incarnations']:
            incarnation['module'] = dict(file='target/module.wasm', bytes=module['bytes'], sha256=module['sha256'], limit=512 << 20)
            incarnation['imports'] = imports
        artifact = Path(report['cells'][0]['artifact'])
        require(artifact.is_relative_to(work), 'foreign primary cluster artifact')
        common.seed['authenticate_artifact'](artifact)
        record = check.check_artifact(artifact, cell, expected)
        retained = read(artifact / 'evidence/cluster-replay.json')
        require(retained['helper_sha256'] == config['identity']['helper'], 'foreign ordinary primary cluster helper')
        require(retained['configuration'] == retained_configuration(expected),
                'primary cluster changed original configuration or resource bounds')
        commands = read(work / 'commands.json')
        require([row['name'] for row in commands] == [cell + '-record', cell + '-detached-replay'], 'missing/duplicate source-less cluster execution')
        for row in commands:
            require(row['status'] == row['expected'] == 0 and row['engine_entered'] is True
                    and row['argv'][0] == binary, 'unexecuted primary cluster command')
        replay = commands[1]
        require(replay['detached'] is True and replay['argv'] == [binary, 'fixture-live-cluster', '--root', str(work / cell / 'replay-private'),
                '--input', str(work / cell / 'replay-admission.json')], 'primary replay must use detached original admission')
        actual = json.loads(replay['stdout'])
        require(actual['reproduced'] is True and actual['wall_elapsed_nanos'] > 0
                and actual['record_sha256'] == sha(artifact / 'record.json')
                and len(actual['completed']) == len(expected['incarnations']), 'fresh detached primary cluster replay differs')
        require(list((work / cell / 'replay-private').iterdir()) == []
                and not (work / cell / 'service.wasm').exists() and not (work / cell / 'input.json').exists(),
                'primary retained replay retained executable inputs')
        require(report['cells'][0]['record_sha256'] == check.digest(check.canonical(record)), 'primary cluster record changed')


def broker(config, label, root):
    check = live.load('live-broker-cluster-check')
    observe = check.observe
    item = recipes(config, label, root)[0]
    argv = item['argv']
    work, reference = Path(live.option(argv, '--work')), Path(live.option(argv, '--observation'))
    roots = [33, 34] if label == 'busy-boundaries' else [36]
    index, derivation = read(reference / 'observation-index.json'), read(reference / 'derivation.json')
    observe.validate_versions(index, derivation)
    require(sha(reference / 'derivation.json') == index['derivation_sha256']
            and sha(reference / 'go-test.jsonl') == index['go_test_sha256']
            and sha(reference / 'test-only-helper.go') == derivation['helper_sha256']
            and sha(reference / 'test-only.diff') == derivation['diff_sha256'], 'changed original broker observation derivation')
    observe.validate_events((reference / 'go-test.jsonl').read_bytes(), roots)
    rows = []
    for name, descriptor in index['files'].items():
        require(Path(name).name == name, 'unsafe broker observation member')
        path = reference / 'observations' / name
        require(path.stat().st_size == descriptor['bytes'] and sha(path) == descriptor['sha256'], 'changed original broker observation')
        if name.endswith('.json'):
            value = read(path)
            observe.validate_row(value, index)
            if name.startswith('cluster-') and value['test'].split('/')[0] in {observe.ROOTS[number] for number in roots}:
                rows.append(value)
    rows.sort(key=lambda row: row['ordinal'])
    for number, count in ((33, 8), (34, 27), (36, 18)):
        if number in roots:
            require(sum(row['test'].split('/')[0] == observe.ROOTS[number] for row in rows) == count, 'incomplete original broker primary matrix')
    commands = read(work / 'commands.json')
    report = read(work / 'result.json')
    require(report['schema'] == 'gomad.live-broker-cluster-check/v1' and report['roots'] == roots
            and report['binary_sha256'] == sha(live.option(argv, '--binary'))
            and report['executions'] == len(commands) == len(rows), 'missing/foreign broker primary results')
    diversity = {}
    for index, (original, actual) in enumerate(zip(rows, commands)):
        output = work / f'cell-{index}'
        request = read(work / f'cell-{index}.json')
        expected = check.linked_input(original)
        require(request['configuration'] == expected['configuration'] and request['modules'] == expected['modules'], 'changed primary broker lifecycle/module/bounds')
        controls = dict(initial_links=[dict(action=step['link_action'], **{'from': step['from'], 'to': step['to']},
                                           delay=step['delay'], symmetric=step['symmetric'])
                                      for step in original['steps'] if step['action'] == 'link'])
        require(set(request) == {'schema', 'configuration', 'modules', 'controls', 'forcing'}
                and request['schema'] == 'gomad.live-cluster-observation/v1' and request['controls'] == controls,
                'changed original primary broker controls')
        states = {(state['handle']['ID'], state['handle']['Incarnation']): state for state in original['states']}
        forced = {}
        for step in original['steps']:
            if step['action'] not in ('start', 'restart') or step['runtime']['Choice']['Mode'] == 1:
                continue
            previous = states[step['node'], step['incarnation']]['result']
            choice = step['runtime']['Choice']
            trace = check.b64(choice.get('Trace') or (previous['InterruptedChoiceBytes'] if previous['RuntimeInterrupted'] else previous['ChoiceTrace']['Bytes']))
            key = step['node'] + '/' + str(step['incarnation'])
            forced[key] = trace
            actual_forcing = request['forcing'][key]
            require(actual_forcing['mode'] == (3 if previous['RuntimeInterrupted'] else choice['Mode'])
                    and actual_forcing['identity'] == choice['ExecutionIdentity']
                    and Path(actual_forcing['trace_path']) == work / f"forcing-{index}-{step['node']}-{step['incarnation']}.choices"
                    and Path(actual_forcing['trace_path']).read_bytes() == trace, 'changed original primary forcing trace')
        require(set(request['forcing']) == set(forced), 'missing/extra original primary replay control')
        require(actual['test'] == original['test'] and actual['exit'] == 0 and actual['command'] ==
                [live.option(argv, '--binary'), 'fixture-live-cluster', '--root', str(output), '--input', str(work / f'cell-{index}.json')],
                'wrong primary broker executable/cell')
        require(check.compare(output, original, process_exit=actual['exit']) == actual['comparison'], 'primary broker raw semantic comparison changed')
        if 36 in roots and check.independent_stream_recording(original):
            current = output / 'incarnations/a/1'
            stdout = (current / 'stdout').read_bytes()
            order = re.search(rb'(?m)^runnable order(?: [0-3]){4}$', stdout)
            fixture = original['test'].split('/')[-1]
            require(order and b'fresh heap 1' in stdout and
                    (b'stall expired' if fixture == 'stall' else b'ordered bytes ordered-byte') in stdout, 'missing independent stream witness')
            diversity.setdefault(fixture, []).append([sha(current / 'choice'), order.group().decode()])
    require(report['diversity'] == diversity, 'changed primary stream diversity')
    if 36 in roots:
        require(set(diversity) == {'partition-heal-delay', 'stall', 'live-link'}, 'missing original stream fixture')
        for samples in diversity.values():
            require(len(samples) == 3 and len({item[0] for item in samples}) >= 2 and len({item[1] for item in samples}) >= 2,
                    'original stream seeds lack actual choice/order variation')


def conformance(config, root):
    argv = recipes(config, 'conformance-helper-build', root)[0]['argv']
    rows = [json.loads(line) for line in (root / 'primary-final.stdout').read_bytes().splitlines()]
    require(rows and rows[-1].get('reason') == 'build-finished' and rows[-1].get('success') is True,
            'missing actual conformance build completion')
    executable = root / 'primary/target/release/gomad-wasmhost'
    artifacts = [row for row in rows if row.get('reason') == 'compiler-artifact' and row.get('executable') == str(executable)]
    require(len(artifacts) == 1, 'missing/duplicate actual conformance executable artifact')
    artifact = artifacts[0]
    require(artifact['target']['name'] == 'gomad-wasmhost' and artifact['target']['kind'] == ['bin']
            and artifact['target']['src_path'] == str(Path(config['repo']) / 'tools/gomad/rust/wasmhost/src/main.rs')
            and set(artifact['features']) == {'conformance', 'reference-helper'}
            and artifact['profile']['opt_level'] == '3' and artifact['profile']['debug_assertions'] is False
            and artifact['profile']['test'] is False, 'foreign conformance source/feature/profile artifact')
    require(executable.is_file() and not executable.is_symlink() and executable.stat().st_size > 0,
            'missing actual conformance executable bytes')
    require(argv[argv.index('--features') + 1] == 'conformance', 'unexpected conformance feature combination')
    return str(executable), sha(executable)


def read_stage(config, label, root):
    guard(config)
    reader = actual_commands(config, label, root)
    if label in ('wasmmeasure-build', 'helper-build'):
        native_build(config, root)
    elif label == 'ordinary-helper-publication':
        native_publication(config, root)
    elif label == 'terminal-controls':
        import terminal_controls
        report = terminal_controls.collect(config, root / 'primary', produce=False)
        require(report['schema'] == 'gomad.rust-driver-check/v1' and report['stage'] == label
                and report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity'],
                'foreign original terminal control reader')
        coverage(subgates(label), report['subgates'])
    elif label == 'conformance-helper-build':
        conformance(config, root)
    elif label in ('artifact', 'operator'):
        import final_retained
        section = 'retained' if label == 'artifact' else 'operator'
        report = final_retained.collect(config, Path(config['root']), section, read_only=True)
        require(report['schema'] == 'gomad.rust-driver-final-section/v1'
                and report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity'],
                'foreign retained/operator primary proof')
        coverage([final_retained.IDS[section]], report['subgates'])
    elif label in ('retained-durable-restart', 'busy-grouped'):
        cluster(config, label, root)
    elif label in ('busy-boundaries', 'network-streams'):
        broker(config, label, root)
    else:
        reader.temporal(label, standalone_inspection=label == 'temporal-inspection')
        reader.verify_pending_preparations()
    rows = [dict(id=name, status='PASS', executed=True, exit=0) for name in subgates(label)]
    coverage(subgates(label), rows)
    guard(config)
    return dict(schema='gomad.rust-driver-check/v1', stage=label, source_sha256=config['source_sha256'], identity=config['identity'], subgates=rows)


def execute(config, root, label):
    command(config, root, 'producer', canonical_argv(config, label, root, 'run'), timeout=TIMEOUTS[label])
    report = json.loads(command(config, root, 'reader', canonical_argv(config, label, root, 'read'), timeout=3600))
    require(report['schema'] == 'gomad.rust-driver-check/v1' and report['stage'] == label
            and report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity'], 'foreign primary reader')
    coverage(subgates(label), report['subgates'])
    return report['subgates']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('run', 'read'))
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--stage', choices=sorted(OWNED), required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config.update(_config_path=str(args.config.resolve()), _config_sha256=sha(args.config))
        if args.mode == 'run':
            run(config, args.stage, args.root.resolve())
        else:
            print(json.dumps(read_stage(config, args.stage, args.root.resolve()), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, AssertionError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
