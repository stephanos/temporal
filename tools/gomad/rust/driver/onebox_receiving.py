"""Independent retained fn222 and revised onebox-inspection process admission."""
import os
import base64
import subprocess
from pathlib import Path
import sys

from driver import Refusal, Unexecuted, load_module, read, require, sha

CELLS = [('smoke', 'onebox/unchanged-workflow-type-smoke'), ('frontend', 'onebox/frontend-system-info')]
GATES = ['unchanged-source-preparation', 'measured-pass-replay-success',
         'measured-pass-replay-canary', 'storage-transport-inspection']


def binding(config):
    value = config.get('fn222')
    if value is None:
        raise Unexecuted('fn222 ordered support producer binding is missing')
    argv = value['argv']
    script = str(Path(config['repo']) / 'tools/gomad/rust/workload_tools/temporal-support.py')
    require(len(argv) > 2 and argv[1] == script and argv.count('--root') == 1
            and '--prepare-only' not in argv, 'fn222 requires the exact complete original producer route')
    for flag, expected in [('--binary', config['binary']), ('--repo', config['repo']),
                           ('--source-manifest', config['source_manifest']),
                           ('--private-observation-config', config['_config_path'])]:
        require(argv.count(flag) == 1 and argv[argv.index(flag) + 1] == expected,
                'fn222 differs from frozen private input: ' + flag)
    root = Path(argv[argv.index('--root') + 1])
    require(root.is_absolute() and root.is_relative_to(Path(config['root'])) and root != Path(config['root']),
            'fn222 producer root outside frozen attempt')
    require(argv.count('--support') == 1, 'fn222 linked receiver must be explicit and frozen')
    support = argv[argv.index('--support') + 1]
    for path in [argv[0], script, support]:
        require(path in config['files'] and sha(path) == config['files'][path], 'unsealed fn222 command input')
    return argv, root, support


def command(reader, path, expected=None, accepted=(0,), native=False):
    path = reader.contained(path)
    row = read(path)
    registration = read(path.with_name(path.name.replace('.command.json', '.process.json')))
    require(registration['active'] is False and registration['observation'] == 'completed'
            and registration['pid'] == row['operator_process_id'] and registration['argv'] == row['command']
            and registration['exit'] == row['exit'] and registration['start_ns'] == row['started_nanos']
            and registration['start_ns'] <= registration['end_ns'] <= row['ended_nanos'],
            'fn222 command/process registration differs')
    require(row['schema'] == 'gomad.rust-onebox-command/v1', 'foreign fn222 process schema')
    argv = row['command']
    require(expected is None or argv == list(map(str, expected)), 'fn222 actual command differs')
    require(row['cwd'] == reader.config['repo'] and type(row['operator_process_id']) is int
            and row['operator_process_id'] > 0 and type(row['exit']) is int and row['exit'] in accepted,
            'fn222 command process/exit/cwd mismatch')
    require(type(row['started_nanos']) is int and type(row['ended_nanos']) is int
            and row['ended_nanos'] >= row['started_nanos'] >= reader.config['sent_epoch'] * 1_000_000_000,
            'fn222 command outside frozen send interval')
    require(row['elapsed_nanos'] <= 3600 * 1_000_000_000, 'fn222 command exceeded original outer bound')
    for flag in ['GOMAD_WASM_QUALIFICATION_GUEST_PROFILE', 'GOMAD_WASM_TEMPORAL_BUILD_CACHE',
                 'GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES']:
        require(row['environment'].get(flag) == reader.config.get('environment', {}).get(flag),
                'fn222 private environment control changed: ' + flag)
    require(row['environment']['GOMAD_WASM_QUALIFY_WORKLOADS'] == ('1' if row['mandatory_workloads'] else None),
            'fn222 actual environment differs from declared mandatory intent')
    executable = row['executable']
    require(argv and str(Path(argv[0]).resolve()) == executable['path']
            and reader.config['files'].get(executable['path']) == executable['sha256']
            and sha(executable['path']) == executable['sha256'], 'fn222 command executable is foreign')
    streams = {}
    for name in ['stdout', 'stderr']:
        stream = row[name]
        source = reader.contained(stream['path'])
        require(source.stat().st_size == stream['bytes'] <= 64 << 20
                and sha(source) == stream['sha256'], 'fn222 raw output changed')
        streams[name] = source.read_bytes()
    owners = [parent / 'producer-root.json' for parent in (path.parent, *path.parents)
              if parent.is_relative_to(reader.root) and (parent / 'producer-root.json').is_file()]
    if owners:
        require(registration['stage_pid'] == read(owners[0])['pid'], 'fn222 process not owned by producer root')
    if native:
        require(executable['path'] == reader.config['binary'] and 'private_observations' in row,
                'fn222 native preparation lacks its own private observation seal')
        index = 3 if len(argv) > 1 and argv[1] == '--installation' else 1
        if argv[index] == 'qualify':
            require(len(argv) == index + 3 and argv[index + 1] == '--spec', 'fn222 measured spec argv changed')
            specification = row.get('qualification_spec')
            require(isinstance(specification, dict) and specification['path'] == argv[index + 2],
                    'fn222 measured command omitted its exact spec witness')
            raw = base64.b64decode(specification['bytes_base64'], validate=True)
            from receiving import digest
            require(0 < len(raw) <= 16 << 20 and digest(raw) == specification['sha256'],
                    'fn222 retained spec bytes/digest changed')
        observations = load_module('fn222_native_observations', Path(__file__).with_name('native_observations.py'))
        observations.verify_snapshot(reader.config, row['operator_process_id'], row['private_observations'])
    return row, streams['stdout']


def validate(reader, arguments):
    _, _, executable = binding(reader.config)
    result = subprocess.run([executable, *map(str, arguments)], capture_output=True, timeout=60,
                            cwd=reader.config['repo'], env={'PATH': ''})
    require(result.returncode == 0 and len(result.stdout) <= 32 << 20 and len(result.stderr) <= 1 << 20,
            'independent linked fn222 retained validator refused evidence')


def gates(request):
    expected = [(cell, gate) for _, cell in CELLS for gate in GATES]
    require([(row['cell'], row['name']) for row in request['gates']] == expected
            and all(type(row['exit']) is int and row['exit'] == 0 for row in request['gates']),
            'fn222 missing, duplicate, foreign or failed gate')


def support(reader):
    from receiving import decoded, digest
    argv, root, executable = binding(reader.config)
    outer = reader.root / 'stages/temporal-inspection'
    reader.command(reader.config, outer, 'fn222', argv, timeout=3600)
    owner = reader.owner(root, argv)
    outer_command = read(outer / 'fn222.command.json')
    require(owner['pid'] == outer_command['pid']
            and outer_command['start_ns'] <= owner['start_ns'] <= outer_command['end_ns'],
            'fn222 producer root/process interval mismatch')
    request = read(root / 'support-request.json')
    validate(reader, ['validate', root / 'support-list.json'])
    listing = read(root / 'support-list.json')
    require(listing['schema'] == 'gomad.rust-wasm-support-list/v1'
            and listing['candidate']['integrated_source_sha256'] == reader.config['source_sha256']
            and listing['candidate']['executable'] == request['executable']
            and request['executable'] == dict(path=reader.config['binary'], sha256=reader.config['binary_sha256']),
            'fn222 list candidate differs from frozen release')
    gates(request)
    expected_results, pids = [], set()
    for short, cell in CELLS:
        rows = [row for row in listing['rows'] if row['cell'] == cell]
        require(len(rows) == 1 and rows[0]['gates'] == [gate for gate in request['gates'] if gate['cell'] == cell],
                'fn222 row omitted or changed its ordered gates')
        for mode in ['success', 'canary']:
            receipt, stdout = command(reader, root / f'{short}-{mode}.command.json', accepted=(1,) if mode == 'canary' else (0,), native=True)
            response = decoded(stdout)
            expected_spec = __import__('receiving').workloads.support_spec(Path(reader.config['repo']),
                root / (short + '-results') / mode, short, mode == 'canary',
                source_manifest=Path(reader.config['source_manifest']))
            require(decoded(base64.b64decode(receipt['qualification_spec']['bytes_base64'], validate=True)) == expected_spec, 'fn222 changed original selection, argv, policy or bounds')
            require(receipt['mandatory_workloads'] == (short == 'smoke'), 'fn222 changed measured workload intent')
            path = Path(response['result_path'])
            require(path.is_relative_to(root / (short + '-results') / mode), 'fn222 Result borrowed from another row')
            expected_results.append(str(path))
            report_command = [reader.config['binary'], 'report', '--result', str(path)]
            _, output = command(reader, root / f'{short}-{mode}-report.command.json', report_command)
            reported = decoded(output)
            result = reader.result(path, reported)
            require(result['observations']['operator_process_id'] == receipt['operator_process_id']
                    and receipt['operator_process_id'] not in pids, 'fn222 repeated or foreign operator PID')
            pids.add(receipt['operator_process_id'])
            reference = rows[0][mode]
            require(reference['path'] == str(path) and reference['result_sha256'] == sha(path)
                    and reference['raw_manifest'] == result['manifest'], 'fn222 edited raw Result reference')
            command(reader, root / f'{short}-{mode}-admission.command.json', [executable, 'check-result', str(path)])
        preparation, _ = command(reader, root / (short + '-plan.command.json'), native=True)
        require(preparation['mandatory_workloads'] is False, 'fn222 preparation incorrectly enabled mandatory intent')
    require(len(listing['rows']) == 2 and request['results'] == expected_results, 'fn222 requires exactly both support pairs')
    for gate in request['gates']:
        log = reader.contained(gate['log'])
        require(sha(log) == gate['log_sha256'], 'fn222 stage log changed')
        value = read(log)
        require(value['schema'] == 'gomad.rust-onebox-stage/v1' and value['exit'] == 0
                and value['command'] == gate['command'], 'fn222 wrapper log differs from gate')
    command(reader, root / 'support-publication.command.json', [executable, 'publish', str(root / 'support-request.json'), str(root / 'support-list')])
    command(reader, root / 'support-validation.command.json', [executable, 'validate', str(root / 'support-list.json')])
    inspection(reader, root / 'inspection')


def inspection(reader, root):
    reader.owner(root)
    from receiving import canonical, digest
    primary = read(reader.root / 'stages/workload-onebox-cooperative-success/result.json')['path']
    validate(reader, ['validate-inspection', root / 'inspection-receipt.json', sha(root / 'inspection-receipt.json'),
                      primary, reader.config['binary'], reader.config['binary_sha256']])
    receipt = read(root / 'inspection-receipt.json')
    require(receipt['schema'] == 'gomad.rust-temporal-inspection/v1' and receipt['owner'] == 'onebox'
            and receipt['binary_sha256'] == reader.config['binary_sha256'], 'foreign onebox inspection receipt')
    artifact = reader.contained(receipt['artifact'])
    require(artifact.is_relative_to(root / 'artifacts'), 'inspection borrowed another gate artifact')
    actual = reader.artifact(artifact)
    require(actual['record_sha256'] == receipt['record_sha256']
            and actual['record']['identity']['implementation'] == digest(canonical(reader.config['identity'])),
            'inspection foreign compiled implementation')
    path = Path(reader.config['repo']) / 'tools/gomad/testdata/parity/onebox-inspection-check.py'
    require(str(path) in reader.config['files'] and sha(path) == reader.config['files'][str(path)],
            'unsealed onebox inspection reader')
    checker = load_module('fn222_inspection_reader', path)
    require(checker.inventory(artifact) == receipt['artifact_sha256'], 'inspection artifact inventory changed')
    checker.authenticate_payloads(artifact)
    require(checker.validate_debug(root, receipt['report']) == receipt['diagnostic_sha256'], 'inspection diagnostic bytes changed')
    expected = dict(install=0, explore=1, **{'detached-replay': 1, 'stopped-inspection': 0, 'reopened-replay': 1})
    require(set(receipt['commands']) == set(expected), 'onebox inspection missing or foreign command gate')
    for name, status in expected.items():
        binding = receipt['commands'][name]
        command_path = reader.contained(binding['path'])
        require(command_path == root / (name + '.command.json') and sha(command_path) == binding['sha256']
                and binding['exit'] == status, 'onebox inspection command binding changed')
        row = read(command_path)
        require(row['command'][0] == reader.config['binary'] and row['pid'] > 0 and row['cwd'] == reader.config['repo']
                and row['exit'] == status and row['end_ns'] >= row['start_ns'] >= reader.config['sent_epoch'] * 1_000_000_000,
                'onebox inspection process outside candidate/send interval')
        for stream in ['stdout', 'stderr']:
            source = reader.contained(row[stream])
            require(sha(source) == row[stream + '_sha256'], 'onebox inspection raw command output changed')
        if name == 'explore':
            require('private_observations' in row, 'onebox inspection missing original native seal')
            observations = load_module('fn222_inspection_native', Path(__file__).with_name('native_observations.py'))
            observations.verify_snapshot(reader.config, row['pid'], row['private_observations'])
    return receipt
