"""Actual public Rust CLI adapters and independent receiver checks."""
import hashlib
import json
import os
from pathlib import Path
import re
import struct
import time

from driver import (HERE, REPO, Refusal, Unexecuted, bind_identity, check_files, command, coverage,
                    mapping, read, require, sha, subgates, write, busy_host)

CELLS = {}
for workload, profile, identity in (
        ('fixture', 'controlled', 'ready-select-timer'),
        ('fixture', 'cooperative', 'ready-select-timer'),
        ('sqlite', 'cooperative', 'sqlite-durable-recovery'),
        ('onebox', 'cooperative', 'unchanged-workflow-type-smoke'),
        ('multi-instance', 'cooperative', 'independent-temporal-replication-durable-restart')):
    for cell in ('success', 'canary'):
        CELLS[f'workload-{workload}-{profile}-{cell}'] = dict(workload=workload, profile=profile, id=identity, cell=cell)

# Static checks can retain Go compiler/model/oracle commands. Live host qualification cannot.
STATIC = set(list(mapping())[:24])


def passed(label):
    return [dict(id=name, status='PASS', executed=True, exit=0) for name in subgates(label)]


def public(config, installation=None):
    argv = [config['binary']]
    if installation:
        argv.extend(['--installation', installation])
    return argv


def admission(config, root):
    require(sha(config['binary']) == config['binary_sha256'], 'release executable changed')
    require(config.get('features') == [], 'release feature set must be empty')
    require(config['build_receipt'] in config['files'], 'missing frozen release feature build receipt')
    build = read(config['build_receipt'])
    require(build['binary_sha256'] == config['binary_sha256'] and build['features'] == []
            and build['exit'] == 0 and build['profile'] == 'release', 'unadmitted release build')
    # Build receipt is reviewed input, not a substitute for release CLI admission in qualify.
    for name, installation in config['installations'].items():
        require(installation in config['files'], 'unsealed installation: ' + name)
        observed = read(installation)
        actual = {key: observed[key + '_sha256'] for key in config['identity']}
        bind_identity('release', config['identity'], build['role'], actual)
        require(observed['admission']['host_executable_sha256'] == config['binary_sha256'], 'installation binary mismatch')
        command(config, root, 'discover-' + name, public(config, installation) + ['discover'])
    # Fixture-only controls must be absent from the admitted release parser.
    for index, control in enumerate(('fixture-execution-bundle', 'fixture-replay-input', 'fixture-qualify-accounting')):
        name = 'release-refusal-' + str(index)
        output = command(config, root, name, public(config) + [control], accepted=(2,))
        require(output == b'' and (root / (name + '.stderr')).read_bytes() == b'gomad: unsupported command\n',
                'fixture grammar is present in qualification binary')


def go_roots(expected, text, repeats):
    for name in expected:
        escaped = re.escape(name)
        require(len(re.findall(r'^\s*--- PASS: ' + escaped + r' \(', text, re.M)) == repeats,
                'missing/duplicate PASS subgate: ' + name)
        require(not re.search(r'^\s*--- (SKIP|FAIL): ' + escaped + r' ', text, re.M), 'skipped/failed subgate: ' + name)


def checked_command(config, root, label, entry):
    argv = entry['argv']
    require(argv and argv[0] in config['files'], 'unsealed command executable')
    if label not in STATIC:
        require(Path(argv[0]).name not in ('go', 'wasmmeasure', 'bash', 'sh', 'make'),
                'old Go qualification producer/shell entry is forbidden')
    with busy_host(label):
        raw = command(config, root, 'producer', argv, timeout=entry['timeout_s'])
    if entry.get('reader') == 'static-exit':
        require(label in STATIC, 'exit-only observation cannot qualify a live gate')
        require(not any(token in argv for token in ('test', 'test-harness')), 'tests require executed root/count reader')
    elif entry.get('reader') == 'rust-tests':
        require(label in STATIC, 'Rust unit tests cannot qualify live gates')
        summaries = re.findall(r'test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored;', raw.decode())
        require(summaries and sum(int(row[0]) for row in summaries) > 0
                and all(int(row[1]) == 0 and int(row[2]) == 0 for row in summaries), 'empty/failed/skipped Rust suite')
    elif entry.get('reader') == 'go-roots':
        require(label in STATIC, 'Go live qualification roots are forbidden')
        go_roots(mapping()[label]['required_subgates'], raw.decode(), mapping()[label]['repetitions'])
    else:
        checker = entry.get('checker')
        if not checker:
            raise Unexecuted(label + ': actual independent reader is not implemented')
        require(checker[0] in config['files'], 'unsealed reader executable')
        raw = command(config, root, 'reader', checker, timeout=entry['timeout_s'])
        report = json.loads(raw)
        require(report['schema'] == 'gomad.rust-driver-check/v1' and report['stage'] == label,
                'wrong reader/stage schema')
        require(report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity'],
                'reader candidate mismatch')
        coverage(subgates(label), report['subgates'])
        return report['subgates']
    return passed(label)


def validate_cell_spec(specification, wanted):
    manifest, target = specification['Manifest'], specification['Campaign']['Target']
    require(manifest.get('Provisional', False) is False, 'provisional Result cannot qualify')
    for key, value in dict(ID=wanted['id'], Workload=wanted['workload'], Profile=wanted['profile'],
                           Cell=wanted['cell'], Seed=7, Repeat=2).items():
        require(manifest.get(key) == value, 'wrong mandatory cell ' + key)
    if wanted['workload'] == 'onebox':
        require(target['Kind'] == 'go-test' and target['Source'] == './tests', 'unchanged ./tests smoke required')
        selected = '[' if wanted['cell'] == 'canary' else '^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$'
        require(target['Args'] == ['-test.v', '-test.count=1', '-test.timeout=72h', '-test.run=' + selected,
                                   '-persistenceType=sql', '-persistenceDriver=sqlite'], 'wrong unchanged workflow argv')
    if wanted['workload'] == 'fixture':
        require(target['Kind'] == 'go-run' and target['Source'] == '.'
                and target['Args'] == (['canary'] if wanted['cell'] == 'canary' else []), 'wrong original fixture target')
    if wanted['workload'] in ('sqlite', 'multi-instance'):
        require(bool(target.get('Scenario')), 'actual durable incarnations/oracle scenario required')


def validate_budget(specification, installation, wanted):
    campaign = specification['Campaign']
    onebox = wanted['workload'] == 'onebox'
    replication = wanted['workload'] == 'multi-instance'
    temporal = onebox or replication
    require(installation['fuel'] == (1 << 50 if temporal else 4_000_000_000), 'changed fuel bound')
    require(installation['memory_bytes'] == (4 << 30 if temporal else 256 << 20), 'changed guest memory bound')
    if wanted['workload'] == 'fixture':
        exact = dict(ExecutionTimeout=60_000_000_000, OverallTimeout=180_000_000_000, TerminateGrace=100_000_000,
                     OutputLimit=1 << 20, WorldTransitionLimit=1 << 20)
        for field, value in exact.items():
            require(campaign[field] == value, 'changed fixture bound ' + field)
    if onebox:
        exact = dict(ExecutionTimeout=900_000_000_000, OverallTimeout=1_800_000_000_000,
                     TerminateGrace=1_000_000_000, OutputLimit=256 << 20, WorldTransitionLimit=32 << 20,
                     ChoiceTraceLimit=64 << 20, SuccessBytesLimit=2 << 30)
        for field, value in exact.items():
            require(campaign[field] == value, 'changed onebox bound ' + field)
    require(installation['model']['limits']['calls'] == (1 << 28 if onebox else 4_000_000 if replication else 100_000 if wanted['workload'] == 'sqlite' else 10_000),
            'changed WASI call bound')
    require(installation['model']['limits']['transcript_bytes'] == (1 << 30 if onebox else 128 << 20 if replication else 32 << 20 if wanted['workload'] == 'sqlite' else 8 << 20),
            'changed WASI transcript bound')
    namespace = (dict(descriptors=4096, files=100000, filesystem_bytes=512 << 20, output_bytes=256 << 20) if onebox else
                 dict(descriptors=4096, files=10000, filesystem_bytes=512 << 20, output_bytes=8 << 20) if replication else
                 dict(descriptors=128, files=1024, filesystem_bytes=8 << 20, output_bytes=1 << 20) if wanted['workload'] == 'sqlite' else
                 dict(descriptors=64, files=1000, filesystem_bytes=8 << 20, output_bytes=1 << 20))
    require(installation['namespace']['limits'] == namespace, 'changed original namespace limits')
    require(installation['model']['limits']['pending_events'] == 4096, 'changed original model event bound')
    require(installation['model']['epoch_nanos'] == (1700000000000000000 if replication else 946684800000000000),
            'changed workload clock epoch')
    require(installation['model']['read_step_nanos'] == (1000 if wanted['profile'] == 'controlled' else 0),
            'changed workload clock step')


def result_report(config, root, name, result_path):
    # The public reader authenticates raw canonical Result/receipt/records and recomputes costs.
    raw = command(config, root, name, public(config) + ['report', '--result', str(result_path)])
    report = json.loads(raw)
    result = report['result']
    require(report['publication']['status'] == 'within' and report['receipt']['operator_peak_rss_bytes'] is not None
            and result['observations']['operator_peak_rss_bytes'] is not None, 'unattributed/exceeded terminal cost')
    result['_driver_publication'] = report['publication']
    require(result['schema'] == 'gomad.rust-qualification-result/v1', 'non-Rust Result lineage')
    manifest = result['manifest']
    require(not manifest['provisional'] and manifest['seed'] == 7 and manifest['repeat'] == 2, 'nonfinal Result')
    require(manifest['integrated_source_sha256'] == config['source_sha256'], 'wrong Result source')
    for key, value in config['identity'].items():
        require(manifest[key + '_sha256'] == value, 'Result identity mismatch: ' + key)
    require(result['cost_failure'] is None and result['budget_failure'] is None
            and result['budget']['within_budget'] is True and not result['budget']['pending'], 'incomplete cost/budget attribution')
    execution = result['report']
    require(execution['deterministic'] and execution['failure'] is None and len(execution['executions']) == 2,
            'incomplete deterministic repetitions')
    require(all(row['artifact_path'] and row['replay'] and row['replay']['attempted'] and row['replay']['matched']
                for row in execution['executions']), 'missing retained exact replay')
    cell = manifest['cell']
    expected_class = 'qualified' if cell == 'success' else 'target_failure'
    require(report['classification'] == expected_class, 'wrong success/canary classification')
    require(execution['target_success'] == (cell == 'success'), 'wrong success/canary outcome')
    if cell == 'canary':
        require(result['cost']['distinct_outcomes'] == 1 and result['cost']['replayable_bugs'] == 1,
                'canary must have exactly one replayable distinct bug')
    return result


def private_capture(config, entry):
    enabled = entry.get('private_observations', False)
    require(type(enabled) is bool, 'private observation opt-in must be boolean')
    return {'private_observations': config} if enabled else {}


def qualify_cell(config, root, label, entry):
    wanted = CELLS[label]
    specification = read(entry['spec'])
    validate_cell_spec(specification, wanted)
    require(entry['spec'] in config['files'], 'unsealed qualification spec')
    require(specification['Manifest']['IntegratedSourceSHA256'] == config['source_sha256'], 'wrong spec source')
    installation = config['installations'][entry['installation']]
    validate_budget(specification, read(installation), wanted)
    if wanted['workload'] in ('sqlite', 'multi-instance') and not entry.get('checker'):
        raise Unexecuted(label + ': actual durable/history/catalogue reader required')
    response = json.loads(command(config, root, 'qualify', public(config, installation) + ['qualify', '--spec', entry['spec']],
                                  accepted=(0,) if wanted['cell'] == 'success' else (1,), **private_capture(config, entry)))
    result_path = Path(response['result_path'])
    result = result_report(config, root, 'report', result_path)
    manifest = result['manifest']
    require(result['observations']['operator_process_id'] == read(root / 'qualify.command.json')['pid'],
            'qualification producer PID differs from actual fresh command')
    for key, value in wanted.items():
        require(manifest[key] == value, 'published mandatory cell differs: ' + key)
    if wanted['workload'] == 'onebox' and wanted['cell'] == 'success':
        for index, execution in enumerate(result['report']['executions']):
            artifact = execution['artifact_path']
            inspected = json.loads(command(config, root, f'smoke-inspect-{index}', public(config) + ['inspect', '--json', artifact]))
            stdout = payload(artifact, inspected, 'output/stdout').decode()
            go_roots(['TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds'], stdout, 1)
    if entry.get('checker'):
        rows = json.loads(command(config, root, 'actual-domain-reader', entry['checker'] + ['--result', str(result_path)]))['subgates']
        coverage(subgates(label), rows)
    write(root / 'result.json', dict(path=str(result_path), sha256=sha(result_path),
          pid=result['observations']['operator_process_id'], workload=wanted, identity=config['identity']))
    return passed(label)


def corpus(config, root, label):
    pids, paths, observed = set(), set(), set()
    for cell, wanted in CELLS.items():
        reference = read(Path(config['root']) / 'stages' / cell / 'result.json')
        path = reference['path']
        require(sha(path) == reference['sha256'], 'primary Result changed')
        result = result_report(config, root, 'report-' + cell, path)
        key = tuple(result['manifest'][field] for field in ('workload', 'profile', 'id', 'cell'))
        require(key == tuple(wanted[field] for field in ('workload', 'profile', 'id', 'cell')), 'wrong corpus cell')
        require(key not in observed and path not in paths and reference['pid'] not in pids, 'borrowed/duplicate corpus producer')
        observed.add(key); paths.add(path); pids.add(reference['pid'])
    require(len(observed) == len(paths) == len(pids) == 10, 'exact ten-cell corpus required')
    files = {}
    for path in paths:
        result_path = Path(path)
        pairs = list(result_path.parent.glob(result_path.name + '.publication-*.json'))
        require(len(pairs) == 1, 'mandatory Result has missing/duplicate receipt')
        files[str(result_path)] = sha(result_path)
        files[str(pairs[0])] = sha(pairs[0])
    if label == 'workload-corpus-after-relocation':
        require(files == read(Path(config['root']) / 'stages/workload-corpus/corpus.json')['files'],
                'primary ten Result/receipt pairs changed during selected-state publication')
    write(root / 'corpus.json', dict(results=sorted(paths), producer_pids=sorted(pids), files=files))
    return passed(label)


def plan(config, root, label, entry):
    installation = config['installations'][entry['installation']]
    result = json.loads(command(config, root, 'plan', public(config, installation) + ['plan', '--json',
            '--output', str(root / 'plan.json'), '--artifacts', str(root / 'artifacts'), *entry['arguments']], **private_capture(config, entry)))
    require(result['schema'] == 'gomad.plan-result/v1' and result['guest_executions'] == 0, 'wrong preparation response')
    require(sha(result['plan']) == result['plan_sha256'], 'changed plan')
    require(sha(result['preparation_receipt']) == result['preparation_receipt_sha256'], 'changed preparation receipt')
    receipt = read(result['preparation_receipt'])
    require(receipt['schema'] == 'gomad.rust-preparation-receipt/v1' and receipt['exit_code'] == 0,
            'unexecuted preparation receipt')
    require(receipt['module_sha256'] == result['module_sha256'], 'preparation module binding')
    prepared = Path(result['campaign']) / receipt['prepared_artifact']
    inspected = json.loads(command(config, root, 'prepared-inspect', public(config) + ['inspect', '--json', str(prepared)]))
    require(inspected['artifact']['record_sha256'] == receipt['record_sha256']
            and inspected['artifact']['files'][receipt['module_file']]['sha256'] == receipt['module_sha256']
            and inspected['artifact']['files'][receipt['provenance_file']]['sha256'] == receipt['provenance_sha256'],
            'preparation receipt/retained artifact mismatch')
    if not entry.get('checker'):
        raise Unexecuted(label + ': original unchanged module/preparation/host admission reader required')
    rows = json.loads(command(config, root, 'preparation-reader', entry['checker']))['subgates']
    coverage(subgates(label), rows)
    write(root / 'preparation.json', result)
    return passed(label)


def retained(config, root, label, entry):
    expected_implementation = hashlib.sha256(json.dumps(config['identity'], sort_keys=True, separators=(',', ':')).encode()).hexdigest()
    for index, artifact in enumerate(entry['artifacts']):
        inspection = json.loads(command(config, root, f'inspect-{index}', public(config) + ['inspect', '--json', '--choices', artifact]))
        record = inspection['artifact']['record']
        require(record['identity']['implementation'] == expected_implementation, 'retained implementation identity mismatch')
        replay = command(config, root, f'replay-{index}', public(config) + ['replay', artifact],
                         accepted=(0,) if record['outcome']['class'] == 'success' else (1,))
        require(b'reproduced=true' in replay and b'diagnostic=false' in replay, 'retained full replay mismatch')
    require(entry['artifacts'], 'empty retained selection')
    # This adapter verifies retained replay only; domain assertions still require the exact checker.
    if not entry.get('checker'):
        raise Unexecuted(label + ': retained domain/subgate reader required')
    raw = command(config, root, 'retained-reader', entry['checker'])
    rows = json.loads(raw)['subgates']
    coverage(subgates(label), rows)
    return rows


def payload(artifact, inspection, name):
    entry = inspection['artifact']['files'].get(name)
    if entry is None:
        raise Unexecuted('required retained payload absent: ' + name)
    path = Path(artifact) / name
    require(not Path(name).is_absolute() and '..' not in Path(name).parts, 'invalid retained payload path')
    require(all(not parent.is_symlink() for parent in [path, *path.parents]), 'retained payload symlink')
    raw = path.read_bytes()
    require(len(raw) == entry['bytes'] and hashlib.sha256(raw).hexdigest() == entry['sha256'],
            'retained payload changed after authentication: ' + name)
    return raw


def runtime_installation(installation):
    require(installation['profile'] == 'gomad.wasi-cooperative/v1', 'runtime cooperative profile required')
    require(installation['fuel'] == 4_000_000_000 and installation['memory_bytes'] == 256 << 20,
            'changed runtime fuel/memory bound')
    model, namespace = installation['model'], installation['namespace']
    require(model['epoch_nanos'] == 946684800000000000 and model['read_step_nanos'] == 1000,
            'changed runtime clock')
    require(model['entropy_key'] == [1, 2, 3] + [0]*29, 'changed runtime entropy key')
    require(model['limits'] == dict(calls=10000, transcript_bytes=8 << 20, pending_events=4096),
            'changed runtime WASI limits')
    require(namespace['limits'] == dict(descriptors=64, files=1000, filesystem_bytes=8 << 20, output_bytes=1 << 20),
            'changed runtime namespace limits')
    require(bytes(namespace['working_directory']) == b'/workspace', 'changed runtime working directory')


def runtime_witness(record, inspection, stdout, diagnostic=None, terminal=None):
    if diagnostic is None or terminal is None:
        raise Unexecuted('runtime requires retained diagnostic bytes and exact 16-byte runtime-terminal')
    require(record['outcome']['termination'] == 'exit' and record['outcome']['exit_code'] in (0, 17), 'runtime infrastructure termination')
    for witness in (b'entropy ', b'map ', b'runq ', b'timer callbacks ', b'timer parking', b'note negative park wake',
                    b'collector automatic finalizer cleanup 1 1', b'canary '):
        require(witness in stdout, 'missing runtime witness: ' + repr(witness))
    choice = inspection['artifact']['choices']
    require(choice['terminal_state'] == 'complete' and all(choice[key] > 0 for key in ('runnable', 'select_poll', 'select_result')),
            'incomplete runtime hook witnesses')
    # Linked owner streams hold raw 96-byte wire records, without a transport header.
    require(len(diagnostic) == choice['records'] * 96 and diagnostic, 'diagnostic/choice count mismatch')
    previous = None
    for index in range(len(diagnostic) // 96):
        raw = diagnostic[index*96:(index+1)*96]
        require(struct.unpack_from('>Q', raw)[0] == index and raw[28] <= 2 and not any(raw[29:32] + raw[36:40]),
                'malformed runtime diagnostic ordinal/reserved/GC phase')
        counters = [struct.unpack_from(fmt, raw, offset)[0] for fmt, offset in
                    [('>q', 8), ('>Q', 16), ('>I', 24)] + [('>Q', offset) for offset in range(40, 96, 8)]]
        require(previous is None or all(a >= b for a, b in zip(counters, previous)), 'runtime diagnostic counters regressed')
        previous = counters
    for offset, fmt in ((16, '>Q'), (24, '>I'), (40, '>Q'), (56, '>Q'), (64, '>Q'), (80, '>Q')):
        require(struct.unpack_from(fmt, diagnostic, len(diagnostic)-96+offset)[0] > 0, 'missing allocation/GC/seeded draw witness')
    require(len(terminal) == 16 and struct.unpack_from('>I', terminal)[0] == 1
            and struct.unpack_from('>I', terminal, 4)[0] > 0, 'incomplete runtime terminal/peak witness')
    return dict(exit=record['outcome']['exit_code'], choice=record['streams']['evidence/choice']['sha256'],
                stdout=hashlib.sha256(stdout).hexdigest(), summary={key: choice[key] for key in
                ('records', 'runnable', 'select_poll', 'select_result')})


def runtime(config, root, label, entry):
    import runtime_gate
    return runtime_gate.runtime(config, root, label, entry, globals())


def runtime_aggregate(config, root, label):
    import runtime_gate
    return runtime_gate.runtime_aggregate(config, root, label, globals())



def execute_primary(config, root, label):
    entry = config['stages'][label]
    adapter = entry['adapter']
    expected = ('qualify' if label in CELLS else 'runtime' if re.fullmatch(r'runtime-\d+', label) else
                'runtime-aggregate' if label == 'runtime-aggregate' else
                'sqlite-selection' if re.fullmatch(r'sqlite-selection-[0-3]', label) else
                'sqlite-catalogue' if label == 'sqlite-catalogue' else
                'corpus' if label in ('workload-corpus', 'workload-corpus-after-relocation') else
                'plan' if label == 'terminal-module' else
                'retained' if label in ('artifact', 'operator', 'retained-cluster', 'retained-durable-restart') else 'checked-command')
    require(adapter in ('pending', expected), 'adapter does not preserve stage semantics: ' + label)
    if adapter == 'pending':
        raise Unexecuted(label + ': ' + entry.get('reason', 'actual Rust endpoint unavailable'))
    if adapter == 'checked-command':
        result = checked_command(config, root, label, entry)
        if label == 'temporal-inspection' and config.get('fn222'):
            from onebox_receiving import binding
            argv, _, _ = binding(config)
            command(config, root, 'fn222', argv, timeout=3600)
        return result
    if adapter == 'sqlite-selection':
        import sqlite
        return sqlite.select(config, root, label, entry)
    if adapter == 'sqlite-catalogue':
        import sqlite
        return sqlite.aggregate(config, root, label, entry)
    if adapter == 'qualify': return qualify_cell(config, root, label, entry)
    if adapter == 'corpus': return corpus(config, root, label)
    if adapter == 'plan': return plan(config, root, label, entry)
    if adapter == 'retained': return retained(config, root, label, entry)
    if adapter == 'runtime': return runtime(config, root, label, entry)
    if adapter == 'runtime-aggregate': return runtime_aggregate(config, root, label)
    raise Unexecuted(label + ': unknown actual Rust adapter ' + adapter)


def execute(config, root, label):
    observed = execute_primary(config, root, label)
    if config['stages'][label].get('live_families'):
        import live
        live.execute_families(config, root, label)
    return observed
