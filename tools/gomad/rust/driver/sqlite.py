"""Four actual SQLite selections, immutable primary pairs and independent catalogue admission."""
import base64
import copy
import json
import hashlib
import re
from pathlib import Path

from driver import Unexecuted, command, read, require, sha, write


def catalogue(result):
    if 'native_crash_catalogues' not in result:
        raise Unexecuted('actual Rust Result has no native crash catalogue projection')
    rows = result['native_crash_catalogues']
    require(len(rows) == 1, 'exactly one deduplicated native SQLite catalogue required')
    row = rows[0]
    require(row['handle'] == dict(node='sqlite', incarnation=2) and row['volume'] == 'sqlite', 'wrong native catalogue owner')
    enumeration, context = row['enumeration'], row['context']
    require(enumeration['Complete'] is True and not enumeration['Capacity'] and len(enumeration['States']) == 4,
            'actual reviewed four-state complete enumeration required')
    states = enumeration['States']
    require(len({state['Identity'] for state in states}) == 4, 'duplicate native crash state')
    require(len(context['Operations']) <= 8, 'changed native pending-operation bound')
    require(all(state['Volume'] == 'sqlite' and state['PendingSHA256'] == context['PendingSHA256']
                and len(state['SelectedOperations']) <= 8 for state in states), 'native state/context mismatch')
    return row


def selection_spec(template, primary_path, primary_sha256, state, context, destination):
    specification = copy.deepcopy(template)
    identity = state['Identity']
    require(identity.startswith('sha256:') and len(identity) == 71, 'native Go state identity encoding changed')
    require(specification['Manifest']['ID'] == 'sqlite-durable-recovery', 'selection requires the registered primary SQLite manifest identity')
    specification['Manifest']['Cell'] = 'success'
    specification['Manifest']['Seed'] = 7
    specification['Manifest']['Repeat'] = 2
    scenario = json.loads(base64.b64decode(specification['Campaign']['Target']['Scenario'], validate=True))
    require(scenario['workload'] == dict(kind='sqlite-durable-recovery', selection=None), 'not a primary actual SQLite scenario')
    require(Path(primary_path).is_absolute(), 'absolute actual primary Result path required')
    require('RuntimeWorkload' not in template, 'primary already has runtime selection intent')
    specification['RuntimeWorkload'] = dict(kind='sqlite-durable-selection', primary_result=primary_path,
        selection=dict(primary_result_sha256=primary_sha256, state=state, context=context))
    specification['ArtifactRoot'] = destination
    specification['Campaign']['Artifacts'] = str(Path(destination) / 'artifacts')
    specification['Campaign']['Target']['PreparationRoot'] = str(Path(destination) / 'preparation')
    return specification


def matched_inputs(primary, selected, runtime_workload):
    for field in ('workload_source_sha256', 'integrated_source_sha256', 'host_os', 'host_arch', 'fuel',
                  'memory_bytes', 'seed', 'repeat', 'compiler_sha256', 'helper_sha256', 'engine_sha256',
                  'runtime_sha256', 'model_sha256', 'profile', 'host_sha256', 'host_config_sha256', 'module_sha256'):
        require(field in primary['manifest'] and field in selected['manifest']
                and selected['manifest'][field] == primary['manifest'][field], 'selected/primary input mismatch: ' + field)
    require('runtime_workload' not in primary['provenance'], 'primary already has runtime intent')
    observed = copy.deepcopy(selected['provenance'])
    require(observed.pop('runtime_workload', None) == runtime_workload,
            'selected provenance lacks exact authorized runtime intent')
    require(observed == primary['provenance'], 'selected/primary preparation provenance differs')


def select(config, root, label, entry):
    from adapters import public, result_report, passed
    index = int(label.rsplit('-', 1)[1])
    primary_entry = config['stages']['workload-sqlite-cooperative-success']
    require(entry['installation'] == primary_entry['installation'], 'selection must use the primary installation')
    primary_stage = Path(config['root']) / 'stages/workload-sqlite-cooperative-success'
    reference = read(primary_stage / 'result.json')
    primary_path = reference['path']
    require(sha(primary_path) == reference['sha256'], 'primary Result changed before selection')
    primary = result_report(config, root, 'primary-report', primary_path)
    native = catalogue(primary)
    state, context = native['enumeration']['States'][index], native['context']
    source = read(config['stages']['workload-sqlite-cooperative-success']['spec'])
    specification = selection_spec(source, primary_path, reference['sha256'], state, context, str(root / 'selected'))
    spec_path = root / 'selection-spec.json'
    write(spec_path, specification)
    installation = config['installations'][entry['installation']]
    response = json.loads(command(config, root, 'qualify', public(config, installation) + ['qualify', '--spec', str(spec_path)],
                                  private_observations=config))
    selected_path = response['result_path']
    result = result_report(config, root, 'selected-report', selected_path)
    require(result['manifest']['id'] == specification['Manifest']['ID'], 'wrong selected state ID')
    expected = dict(primary_result_sha256=reference['sha256'], state=state, context=context)
    require(result.get('native_crash_selection') == expected, 'selected Result lacks exact actual state/context/primary binding')
    require(catalogue(result) == native, 'selected execution does not reproduce complete primary catalogue')
    require(result['observations']['operator_process_id'] == read(root / 'qualify.command.json')['pid']
            and result['observations']['operator_process_id'] != reference['pid'], 'selection borrowed producer/preparation measurement')
    matched_inputs(primary, result, specification['RuntimeWorkload'])
    require(sha(primary_path) == reference['sha256'], 'selection changed primary Result')
    write(root / 'selection.json', dict(index=index, path=selected_path, sha256=sha(selected_path),
          pid=result['observations']['operator_process_id'], primary_sha256=reference['sha256'], selection=expected))
    return passed(label)


def native_input(artifact, inspected, native):
    from adapters import payload
    state_name = 'evidence/cluster/sqlite/2/storage_state'
    log_name = 'evidence/cluster/sqlite/2/storage'
    image_name = 'evidence/cluster/sqlite/2/storage_precrash_image'
    state = payload(artifact, inspected, state_name)
    log = payload(artifact, inspected, log_name)
    image = payload(artifact, inspected, image_name)
    require(len(image) <= 9 << 20, 'changed original SQLite filesystem-derived image bound')
    require(hashlib.sha256(state).hexdigest() == native['storage_state_sha256'], 'catalogue storage-state binding')
    retained = json.loads(payload(artifact, inspected, 'evidence/cluster-replay.json'))
    configuration = retained['configuration']
    encoded = json.dumps(configuration, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    require(hashlib.sha256(encoded).hexdigest() == inspected['artifact']['record']['identity']['configuration'],
            'native oracle configuration binding')
    owners = [row for row in configuration['incarnations'] if row['handle'] == dict(node='sqlite', incarnation=2)]
    require(len(owners) == 1, 'native oracle requires exactly the actual SQLite crash owner')
    descriptor = (owners[0].get('storage') or {}).get('descriptor') or {}
    model_identity = descriptor.get('model_identity')
    require(isinstance(model_identity, str) and re.fullmatch(r'sha256:[0-9a-f]{64}', model_identity) is not None,
            'native oracle requires authenticated actual descriptor model identity')
    return dict(storage_state_path=str(Path(artifact) / state_name), storage_state_sha256=hashlib.sha256(state).hexdigest(),
                storage_log_path=str(Path(artifact) / log_name), storage_log_sha256=hashlib.sha256(log).hexdigest(),
                precrash_image_path=str(Path(artifact) / image_name), precrash_image_sha256=hashlib.sha256(image).hexdigest(),
                model_identity=model_identity, limits=dict(states=32, operations=8, depth=8, bytes=4 << 20, work=1 << 16))


def aggregate(config, root, label, entry):
    from adapters import reader_usage
    before = reader_usage()
    import sqlite_oracle
    require("oracle" not in entry, "caller SQLite oracle argv is forbidden; authentic fixed Go build receipt required")
    sqlite_oracle.validate_build(config)
    from adapters import public, result_report, passed
    reference = read(Path(config['root']) / 'stages/workload-sqlite-cooperative-success/result.json')
    require(sha(reference['path']) == reference['sha256'], 'primary Result changed before catalogue')
    primary = result_report(config, root, 'primary-report', reference['path'])
    native = catalogue(primary)
    artifact = primary['report']['executions'][0]['artifact_path']
    inspected = json.loads(command(config, root, 'primary-inspect', public(config) + ['inspect', '--json', artifact]))
    request = root / 'oracle-input.json'
    write(request, native_input(artifact, inspected, native))
    actual = sqlite_oracle.invoke(config, root, request)
    require(actual['enumeration'] == native['enumeration'] and actual['context'] == native['context'],
            'actual native Go catalogue/context differs from Rust owner')
    seen, processes, cpu, terminal_cpu = set(), {reference['pid']}, 0, 0
    for index in range(4):
        selected = read(Path(config['root']) / 'stages' / f'sqlite-selection-{index}' / 'selection.json')
        require(selected['index'] == index and selected['primary_sha256'] == reference['sha256']
                and sha(selected['path']) == selected['sha256'], 'missing/stale native selected Result')
        result = result_report(config, root, f'selected-report-{index}', selected['path'])
        expected = dict(primary_result_sha256=reference['sha256'], state=native['enumeration']['States'][index], context=native['context'])
        matched_inputs(primary, result, dict(kind='sqlite-durable-selection',
                       primary_result=reference['path'], selection=expected))
        require(result['native_crash_selection'] == selected['selection'] == expected and catalogue(result) == native,
                'selected state/context or complete enumeration differs')
        require(selected['sha256'] not in seen and selected['pid'] not in processes, 'duplicate selected Result or producer')
        require(result['observations']['operator_process_id'] == selected['pid'], 'selected operator PID binding')
        seen.add(selected['sha256']); processes.add(selected['pid'])
        cpu += result['cost']['cpu_nanos']
        terminal_cpu += result['_driver_publication']['cpu_nanos']
    require(len(seen) == 4, 'all four actual selected states required')
    from adapters import reader_cost
    reader_cpu, reader_peak = reader_cost(before, 'SQLite catalogue reader self metric unavailable/nonmonotone/over2GiB')
    write(root / 'catalogue.json', dict(primary=reference['sha256'], selected=sorted(seen), producer_pids=sorted(processes),
          selected_execution_cpu_nanos=cpu, selected_terminal_cpu_nanos=terminal_cpu,
          selected_total_cpu_nanos=cpu+terminal_cpu, primary_cpu_counted=False,
          reader_cpu_nanos=reader_cpu, reader_peak_rss_bytes=reader_peak,
          supplementary_total_cpu_nanos=cpu+terminal_cpu+reader_cpu))
    return passed(label)
