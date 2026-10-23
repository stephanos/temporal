"""Recorded live-corpus endpoints and a read-only, candidate-bound final reader."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import sys

import reach

from driver import (HERE, Refusal, Unexecuted, command, coverage, guard, read, require,
                    sha, subgates, write)

SLIM_SOURCES = {
    'common/archiver/provider/provider_cloud_gomad_test.go': 'TestCloudArchiversRefuseConfiguredSchemesWithoutFallback',
    'temporal/membership_test.go': 'TestDefaultMembershipLinksRingpop',
    'tests/testcore/flag_sql_wasm_slim_test.go': 'TestSlimGuestRefusesUnlinkedSQLDriversWithoutFallback',
}
SLIM_IDS = {'go/go.temporal.io/server/' + path.rsplit('/', 1)[0] + '/' + name for path, name in SLIM_SOURCES.items()}

PARITY = HERE.parents[1] / 'testdata/parity'
FAMILIES = {
    'inspection': ('live-inspection-check.py', (1, 2, 3)),
    'observed': ('live-backend-observed.py', (4, 5, 6)),
    'backend-runtime': ('live-backend-runtime.py', (7,)),
    'choice': ('live-backend-choice.py', (8, 9, 10, 11)),
    'minimize': ('live-backend-minimize.py', (12,)),
    'markers': ('live-backend-markers.py', (13, 14, 15)),
    'softassert': ('live-backend-softassert.py', (16, 17)),
    'oracles': ('live-cluster-oracles.py', (18, 27, 28)),
    'cluster': ('live-cluster-check.py', tuple(range(19, 27))),
    'runtime': ('live-runtime-check.py', (29, 30, 31)),
    'broker': ('live-broker-cluster-check.py', (32, 33, 34, 35, 36, 39)),
    'owners': ('live-broker-owner-check.py', (37, 38)),
    'private35': (None, ()),
    'codec-oracles': (None, ()),
    'prepared-metadata': (None, ()),
}
CELLS = ('lazy', 'stop', 'grouped', 'durable', 'lifecycle', 'compressed-lifecycle',
         'cleanup-scenario-error', 'cleanup-action-capacity', 'cleanup-context-cancellation',
         'unbound-crash', 'unbound-stop')
PRIVATE35 = 'linked_cluster::live_controller_tests::shutdown_reinsertion_preserves_actual_live_guest_world'


AUXILIARY = {
    'codec-oracles': ['world::deflate::tests::' + name for name in (
        'default_compression_matches_pinned_go', 'decode_all_go_block_types_with_bounded_output',
        'accepts_independent_go_compression_levels', 'actual_world_snapshots_match_raw_gzip',
        'mutated_stream_admission_matches_go', 'original_full_profile_bounds',
        'changing_distributions_and_lazy_matches_agree_with_go')],
    'prepared-metadata': ['campaign::prepare::driver::retained_input_tests::retained_actual_wasm_metadata_matches_only_its_original_recipe'],
}


def load(name):
    spec = importlib.util.spec_from_file_location('live_final_' + name.replace('-', '_'), PARITY / (name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def roots():
    names = [item for item in read(HERE / 'required-final.json')['subgates'] if item.startswith('transferred-live/')]
    require(len(names) == len(set(names)) == 39, 'changed original transferred-root contract')
    return {index + 1: name for index, name in enumerate(names)}


def required():
    names = read(HERE / 'required-final.json')['subgates']
    result = [name for name in names if name.startswith(('transferred-live/', 'cluster-admission/'))
              or name == 'test-wasm/complete-behavioral-reach']
    require(len(result) == len(set(result)) == 124, 'changed live/admission/reach obligations')
    return result


def tampers():
    names = [name.split('/')[1] for name in required() if name.startswith('cluster-admission/')]
    result = list(dict.fromkeys(names))
    require(len(result) == 21, 'changed original tamper contract')
    return result


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def inventory(root):
    require(root.is_dir() and not root.is_symlink(), 'missing/nonregular actual evidence root')
    result = {}
    for path in sorted(root.rglob('*')):
        require(not path.is_symlink(), 'symlink in live evidence: ' + str(path))
        if path.is_file():
            result[str(path.relative_to(root))] = dict(bytes=path.stat().st_size, sha256=sha(path))
        else:
            require(path.is_dir(), 'nonregular live evidence')
    require(result, 'empty actual evidence')
    return result


def completed_command(receipt, process, argv, cwd):
    require(isinstance(receipt, dict) and isinstance(process, dict), 'missing actual command/process journal')
    for item in (receipt, process):
        require(item.get('argv') == argv and type(item.get('pid')) is int and item['pid'] > 0,
                'wrong live command/PID')
        require(item.get('exit') == 0 and item.get('observation') == 'completed', 'nonpassing/inconclusive live producer')
        require(type(item.get('start_ns')) is int and type(item.get('end_ns')) is int
                and 0 < item['start_ns'] < item['end_ns'], 'unexecuted live producer interval')
    require(receipt.get('cwd') == cwd and receipt['pid'] == process['pid']
            and receipt['start_ns'] == process['start_ns'] and process.get('active') is False
            and not process.get('cleanup_signal'), 'live process was not cleanly reaped')


def sealed_input(config, path):
    path = Path(path)
    require(path.is_absolute() and not path.is_symlink(), 'unsealed live input path')
    if path.is_file():
        require(config['files'].get(str(path)) == sha(path), 'unsealed/changed live input: ' + str(path))
    else:
        require(path.is_dir(), 'missing live input: ' + str(path))
        files = inventory(path)
        for name, entry in files.items():
            require(config['files'].get(str(path / name)) == entry['sha256'], 'incomplete live input directory inventory')


def role(config, name):
    entry = config['live_corpus']['roles'][name]
    for field in ('binary', 'build_receipt'):
        sealed_input(config, entry[field])
    built = read(entry['build_receipt'])
    require(built['exit'] == 0 and built['profile'] == 'release' and built['source_sha256'] == config['source_sha256'],
            'live binary is not the frozen built candidate')
    require(built['binary_sha256'] == sha(entry['binary']) and built['helper_sha256'] == config['identity']['helper']
            and built['helper_features'] == [], 'foreign/nonordinary live helper implementation')
    require(built['role'] == name, 'live binary role differs from actual build receipt')
    require(built['features'] == ([] if name in ('release', 'ordinary-helper-test', 'ordinary-host-test') else ['determinism-fixture']),
            'unexpected live build feature set')
    require(name in ('release', 'host-fixture', 'ordinary-helper-test', 'ordinary-host-test'), 'unsupported live role')
    if name == 'release':
        require(entry['binary'] == config['binary'] and built['binary_sha256'] == config['binary_sha256'],
                'live release executable differs from admitted CLI')
    return entry


def option(argv, flag, default=None):
    values = [argv[n + 1] for n, item in enumerate(argv[:-1]) if item == flag]
    require(len(values) <= 1 and flag not in argv[-1:], 'duplicate/incomplete live option: ' + flag)
    return values[0] if values else default


def planned(config, family):
    require(family in FAMILIES, 'unknown live family')
    live = config.get('live_corpus')
    require(isinstance(live, dict) and set(live.get('families', {})) == set(FAMILIES),
            'missing/unexpected complete live family plan')
    entry = live['families'][family]
    stage = entry['stage']
    require(stage in config['stages'] and config['stages'][stage].get('live_families', []).count(family) == 1,
            'live endpoint lacks its actual owning stage')
    assigned = [name for item in config['stages'].values() for name in item.get('live_families', [])]
    require(len(assigned) == len(set(assigned)) and set(assigned) == set(FAMILIES), 'duplicate/missing live stage assignment')
    base = Path(config['root']) / 'stages' / stage
    work = base / ('live-' + family)
    argv = entry['argv']
    require(isinstance(argv, list) and argv and all(isinstance(value, str) for value in argv), 'missing live endpoint argv')
    executable = role(config, entry['role'])['binary']
    script = FAMILIES[family][0]
    if script:
        require(len(argv) > 2 and argv[1] == str(PARITY / script), 'wrong original assertion endpoint')
        sealed_input(config, argv[0]); sealed_input(config, argv[1])
        require(option(argv, '--binary') == executable and option(argv, '--work') == str(work), 'wrong live binary/output binding')
        require(entry['role'] in ('release', 'host-fixture'), 'invalid live controller role')
        if family == 'broker': require(option(argv, '--roots', '32,33,34,35,36,39') == '32,33,34,35,36,39', 'partial broker corpus')
        if family == 'runtime': require(option(argv, '--root', 'all') == 'all', 'partial runtime corpus')
        if family == 'cluster':
            require('--admissions' in argv and '--cell' not in argv and option(argv, '--provider-reference') and option(argv, '--input'),
                    'complete lifecycle/admission/root19 reference and sealed template required')
        for index, value in enumerate(argv[2:], 2):
            if value.startswith('/') and value not in (executable, str(work)):
                if argv[index - 1] == '--unavailable': require(not Path(value).exists(), 'detached input still exists')
                else: sealed_input(config, value)
        controller = option(argv, '--controller-binary')
        if controller: require(controller == role(config, 'host-fixture')['binary'], 'foreign controller')
        require(not entry.get('environment'), 'unexpected per-family live environment')
    elif family == 'private35':
        require(entry['role'] == 'ordinary-helper-test' and argv[0] == executable
                and len(argv) == 5 and set(argv[1:]) == {'--ignored', '--exact', '--nocapture', PRIVATE35},
                'private35 must execute the actual exact ordinary test')
        environment = entry.get('environment', {})
        require(set(environment) == {'GOMAD_LIVE_CLUSTER_SHUTDOWN_INPUT', 'GOMAD_LIVE_CLUSTER_SHUTDOWN_OUTPUT'}
                and environment['GOMAD_LIVE_CLUSTER_SHUTDOWN_OUTPUT'] == str(work), 'private35 original input/output controls')
        sealed_input(config, environment['GOMAD_LIVE_CLUSTER_SHUTDOWN_INPUT'])
        private35_input(environment['GOMAD_LIVE_CLUSTER_SHUTDOWN_INPUT'])
    else:
        expected = AUXILIARY[family]
        require(entry['role'] == ('ordinary-helper-test' if family == 'codec-oracles' else 'ordinary-host-test')
                and argv[0] == executable and len(argv) == len(expected) + 5
                and set(argv[1:]) == {'--ignored', '--exact', '--nocapture', '--test-threads=1', *expected},
                'actual exact ignored receiving tests required')
        environment = entry.get('environment', {})
        if family == 'codec-oracles':
            require(set(environment) == {'GOMAD_GO_GZIP_ORACLE', 'GOMAD_CODEC_EVIDENCE_DIR'}
                    and environment['GOMAD_CODEC_EVIDENCE_DIR'] == str(work), 'missing actual bounded codec oracle evidence')
            sealed_input(config, environment['GOMAD_GO_GZIP_ORACLE'])
        else:
            require(set(environment) == {'GOMAD_TEST_UNACCEPTED_MODULE', 'GOMAD_TEST_UNACCEPTED_RECIPE', 'GOMAD_TEST_UNACCEPTED_SHA256'},
                    'missing original prepared metadata inputs')
            for key in ('GOMAD_TEST_UNACCEPTED_MODULE', 'GOMAD_TEST_UNACCEPTED_RECIPE'): sealed_input(config, environment[key])
            require(sha(environment['GOMAD_TEST_UNACCEPTED_MODULE']) == environment['GOMAD_TEST_UNACCEPTED_SHA256'],
                    'changed actual retained unaccepted module')
    return entry, base, work


def admission_rows(rows):
    expected = {(variant, operation) for variant in tampers() for operation in ('verify', 'storage-selections', 'reopen', 'replay')}
    require(isinstance(rows, list) and len(rows) == 84, 'incomplete original admission matrix')
    actual = [(item['variant'], item['operation']) for item in rows]
    require(len(set(actual)) == len(actual) and set(actual) == expected, 'missing/duplicate/unknown tamper entrypoint')
    require(all(item['status'] == 2 and item['executable_writes'] == 0 for item in rows), 'admission executed or accepted tampered input')


def private35_input(path):
    actual = read(path)
    config = actual['configuration']
    expected = dict(memory_bytes=256 << 20, fuel=4000000000, wall_millis=60000,
                    output_bytes=1 << 20, evidence_bytes=64 << 20)
    require(config['limits'] == expected and len(config['incarnations']) == 2
            and all(item['limits'] == expected for item in config['incarnations']), 'changed original private35 execution bounds')
    require(config['steps'] == [dict(action='start', handle=dict(node=node, incarnation=1)) for node in ('a', 'b')],
            'private35 must reconstruct the original live pair after Start')


def private35(work, stdout):
    require(len(re.findall(r'^test result: ok\. 1 passed; 0 failed; 0 ignored;', stdout, re.M)) == 1
            and PRIVATE35 + ' ... ok' in stdout, 'private35 exact ignored test did not actually pass')
    report = read(work / 'result.json')
    require(report['schema'] == 'gomad.live-cluster-shutdown-reinsertion/v1' and report['conditions'] == 'ordinary'
            and report['unchanged_guest'] is True, 'nonordinary/private35 fixture declaration')
    rows = report['iterations']
    require(len(rows) == 16 and [row['iteration'] for row in rows] == list(range(16)), 'private35 missing/duplicate iteration')
    baseline = None
    for index, row in enumerate(rows):
        require(row['live_verified'] is True and row['completed'] == 2 and row['reverse_insertion'] == bool(index % 2),
                'private35 did not mutate the actual still-live map')
        pair = []
        for name in ('world', 'network'):
            path = work / f'{name}-{index}.bin'
            require(path.stat().st_size == row[name + '_bytes'] > 0, 'private35 missing raw evidence')
            pair.append(sha(path))
        if baseline is None: baseline = pair
        require(pair == baseline, 'private35 raw evidence depends on insertion order')


def raw_commands(config, family, work):
    rows = read(work / 'commands.json')
    require(isinstance(rows, list) and rows, 'missing actual live child commands')
    allowed = {item['binary'] for item in config['live_corpus']['roles'].values()}
    for row in rows:
        argv = row.get('argv', row.get('command'))
        require(isinstance(argv, list) and argv, 'missing actual child argv')
        # pprof is the original independent parser, never a Go host producer.
        parser = family == 'runtime' and row.get('name') == 'heap-pprof'
        require(argv[0] in allowed or parser, 'foreign live executable')
        if parser: sealed_input(config, argv[0])
        value = row.get('status', row.get('exit'))
        require(type(value) is int and not row.get('timed_out', False), 'unexecuted/timed-out child')
        if family == 'inspection' and row.get('cancelled'):
            require(value != 0 and row['early_cancel'] is False and 1 <= row['elapsed_seconds'] < 10,
                    'original cancelled dump did not observe its bounded nonzero disposition')
        elif 'expected_statuses' in row: require(value in row['expected_statuses'], 'unexpected live application/refusal disposition')
        elif 'expected' in row: require(value == row['expected'], 'unexpected live application/refusal disposition')
        elif family in ('broker', 'owners', 'runtime'): require(value == 0, 'live fixture controller failed')
        else: raise Refusal('live child has no exact expected application/refusal disposition')
        if 'stdout_sha256' in row:
            for name in ('stdout', 'stderr'):
                require(sha(work / (row['name'] + '.' + name)) == row[name + '_sha256'], 'raw command output changed')
    return rows


def backend(config, work, family):
    report = read(work / 'roots.json')
    require(report['schema'] == 'gomad.live-backend-corpus/v1', 'wrong backend corpus schema')
    expected = list(FAMILIES[family][1])
    observed = [row['root'] for row in report['roots']]
    if family == 'markers':
        require([(row['root'], row['variant']) for row in report['roots']] == [(13, 'crash'), (13, 'stop'), (14, 'crash'), (14, 'stop'), (15, 'wait')], 'missing original marker variant')
    else: require(observed == expected, 'missing/duplicate/renamed backend root')
    for row in report['roots']: require('transferred-live/' + row['original'] == roots()[row['root']], 'renamed transferred root')
    if family == 'backend-runtime': require(report['roots'][0]['selected_seeds'] == [index * 1000003 for index in range(32)], 'changed original representative seeds')
    if family == 'minimize': require(report['roots'][0]['variants'] == ['repeat', 'after accepted', 'after publication'], 'missing minimization resume variant')
    if family == 'softassert':
        require(report['roots'][0]['cells'] == [dict(mode=mode, report_only=flag, seeds=[7, 19]) for mode in ('own', 'late') for flag in (False, True)], 'missing soft assertion policy/seed variant')
        require(report['roots'][1]['reports'] > 0, 'no committed soft assertion report')
    common = load('live-backend-common')
    campaigns = list(work.glob('**/campaigns/campaign-*'))
    require(campaigns, 'root declarations have no raw campaigns')
    for path in campaigns:
        plan = common.seed['canonical_file'](path / 'plan.json')
        if 'strategy' in plan: common.rounds['authenticate'](path)
        else: common.seed['authenticate_campaign'](path, complete=False)
        prepared = plan.get('prepared', plan)
        built = read(role(config, config['live_corpus']['families'][family]['role'])['build_receipt'])
        require(prepared['helper_sha256'] == config['identity']['helper'] and prepared['host_sha256'] == built['host_sha256'],
                'retained campaign implementation differs from frozen live role')
        for manifest in path.rglob('manifest.json'): common.seed['authenticate_artifact'](manifest.parent)


def admissions(work, commands):
    check = load('live-cluster-check')
    for cell in ('lifecycle', 'compressed-lifecycle'):
        base = work / cell / 'admission-matrix'
        rows = read(base / 'admissions.json'); admission_rows(rows)
        for row in rows:
            name = row['variant'] + '-' + row['operation']
            request = read(base / (name + '.json'))
            require(request['operation'] == row['operation'], 'wrong actual admission entrypoint')
            private = base / (name + '-private')
            require(private.is_dir() and not list(private.iterdir()), 'admission wrote executable inputs')
            matching = [item for item in commands if item['name'] == 'admit-' + name and item['argv'][-1] == str(base / (name + '.json'))]
            require(len(matching) == 1 and matching[0]['status'] == matching[0]['expected'] == 2
                    and matching[0]['stdout'] == '' and matching[0]['stderr'] and not matching[0]['engine_entered'],
                    'tamper refusal was not observed before guest execution')


def auxiliary(work, family, stdout):
    expected = AUXILIARY[family]
    rows = re.findall(r'^test (\S+) \.\.\. (ok|ignored|FAILED)', stdout, re.M)
    require(len(rows) == len(expected) and {name for name, _ in rows} == set(expected)
            and all(status == 'ok' for _, status in rows), 'missing/duplicate/skipped actual ignored receiving test')
    summary = f'test result: ok. {len(expected)} passed; 0 failed; 0 ignored;'
    require(stdout.count(summary) == 1, 'wrong actual ignored-test selected count')
    if family == 'codec-oracles':
        rust, go = (work / name for name in ('world-codec-max-rust.gz', 'world-codec-max-go.gz'))
        require(0 < rust.stat().st_size <= 64 << 20 and sha(rust) == sha(go), 'missing/different original bounded raw gzip')
        require('expanded_bytes=402653184 compressed_bytes=' + str(rust.stat().st_size) in stdout,
                'original384MiB boundary did not execute')
    else:
        require('unaccepted input metadata only: module_bytes=' in stdout, 'actual metadata input never decoded')
    return {'rust-test/' + name for name in expected}


def runtime_boundaries(work, commands):
    """Expand the original all-decision obligation from actual retained wire bytes."""
    import copy
    base = work / 'inspection-record'
    metadata = read(base / 'result.json')['inspection']
    require(metadata['complete'] is True and len(metadata['incarnations']) == 1
            and metadata['incarnations'][0]['registry']['closed'] is True, 'incomplete original inspection registry')
    raw = (base / 'streams/choice').read_bytes()
    require(raw and len(raw) % 96 == 0, 'incomplete original choice records')
    wire = [raw[index:index + 96] for index in range(0, len(raw), 96)]
    decisions = [index for index, record in enumerate(wire) if record[9] & 1 and record[8] != 4]
    boundaries = metadata['boundaries']
    require(boundaries and len(boundaries) == len(decisions)
            and any(item['trace'] > item['decision'] for item in boundaries), 'inspection omitted actual decision boundaries')
    source = read(work / 'inspection-record.input.json')
    require(all(source[field] == limit for field, limit in dict(memory_bytes=256 << 20, fuel=4000000000,
            wall_millis=60000, output_bytes=1 << 20, evidence_bytes=128 << 20).items()), 'changed original inspection execution bounds')
    expected, cases, kinds = [], set(), set()
    for ordinal, boundary in enumerate(boundaries):
        index = boundary['trace']
        require(type(index) is int and 0 <= index < len(wire) and type(boundary['decision']) is int and boundary['decision'] == ordinal,
                'invalid original inspection decision coordinate')
        while index < len(wire) and wire[index][8] == 4: index += 1
        require(index == decisions[ordinal], 'inspection decision/trace mapping differs from actual choices')
        kind = wire[index][8]; kinds.add(kind)
        name = f'inspection-kind{kind}-{ordinal}'; expected.append(name)
        request = copy.deepcopy(source)
        request['model']['mode'] = 2
        request['replay_trace_path'] = str(base / 'streams/choice')
        request['inspection'] = dict(metadata_path=str(base / 'streams/inspection'), decision=ordinal,
                                     dumps=['stacks', 'heap'], probes=[])
        require(read(work / (name + '.input.json')) == request, 'stopped request changed the original exact decision/dumps/bounds')
        result = read(work / name / 'result.json')
        require(result['prefix_verified'] is True and result['debug']['stacks']
                and (work / name / 'heap.pprof').stat().st_size > 0, 'actual stopped stack/heap/prefix proof absent')
        cases.add('live-case/TestCooperativeInspectionEveryDecision/' + f'kind{kind}-{ordinal}')
    actual = [item['name'] for item in commands if item['name'].startswith('inspection-kind')]
    require(actual == expected and {1, 2} <= kinds, 'missing/duplicate/extra all-decision stopped executions')
    return cases


def read_family(config, family, work, stdout):
    if family == 'private35':
        private35(work, stdout); return {'private35/16-live-reinsertions'}
    if family in AUXILIARY: return auxiliary(work, family, stdout)
    commands = raw_commands(config, family, work)
    for report_name in ('roots.json', 'result.json', 'observations.json'):
        if (work / report_name).is_file():
            actual = read(work / report_name)
            if 'binary_sha256' in actual:
                expected_binary = role(config, config['live_corpus']['families'][family]['role'])['binary']
                require(actual['binary_sha256'] == sha(expected_binary), 'raw family result names a different executable')
    script = FAMILIES[family][0]
    if script.startswith('live-backend-'):
        backend(config, work, family)
        if family == 'backend-runtime':
            aliases = read(work / 'retired-preparation-aliases.json')
            require(aliases['schema'] == 'gomad.retired-preparation-aliases/v1'
                    and aliases['after'] == 'runtime-detached-exact-replay'
                    and aliases['aliases'] and aliases['aliases'][0] == dict(path='compiler-link', target=str(work / 'go')),
                    'root7 missing original post-replay alias retirement')
            paths = [row['path'] for row in aliases['aliases']]
            require(len(paths) == len(set(paths)) and all(path == 'compiler-link' or (path.startswith('go-unavailable/')
                    and '..' not in Path(path).parts) for path in paths), 'unexpected retired preparation alias')
            require(all(not (work / path).exists() and not (work / path).is_symlink() for path in paths),
                    'preparation aliases were not retired after original assertions')
        report = read(work / 'roots.json')
        require(report['driver_sha256'] == sha(PARITY / script)
                and report['shared_driver_sha256'] == sha(PARITY / 'live-backend-common.py'), 'backend assertion producer changed')
        return
    if family == 'inspection':
        report = read(work / 'result.json')
        require(report['schema'] == 'gomad.transferred-live-inspection/v1' and report['roots'] == [roots()[n].split('/', 1)[1] for n in (1, 2, 3)]
                and report['commands'] == len(commands), 'incomplete retained inspection sweep')
        require(all((work / (item['name'] + '.stdout')).is_file() and (work / (item['name'] + '.stderr')).is_file() for item in commands), 'missing actual inspection output')
    elif family == 'oracles':
        require(read(work / 'passed.json') == dict(roots=[18, 27, 28], commands=len(commands)), 'incomplete typed cluster oracle cells')
        require(list(work.glob('**/evidence/cluster_oracles')), 'no actual retained oracle bytes')
    elif family == 'runtime':
        require(read(work / 'passed.json') == dict(roots='all', commands=len(commands)), 'incomplete runtime/all-decision sweep')
        require(any(item['name'] == 'heap-pprof' for item in commands), 'original heap parser unexecuted')
        for item in commands:
            if item['name'] == 'heap-pprof': continue
            output = Path(item['argv'][item['argv'].index('--root') + 1])
            report = read(output / 'result.json')
            require(report['execution_returned'] is True and report['outcome']['termination']
                    and report['outcome']['fuel_remaining'] <= read(work / (item['name'] + '.input.json'))['fuel'], 'missing actual runtime terminal')
        return runtime_boundaries(work, commands)
    elif family == 'cluster':
        report = read(work / 'observations.json')
        require(report['schema'] == 'gomad.live-cluster-service-cells/v1' and [item['cell'] for item in report['cells']] == list(CELLS), 'missing/duplicate lifecycle/cleanup variant')
        check = load('live-cluster-check')
        entry = config['live_corpus']['families'][family]
        reference = check.provider_reference(Path(option(entry['argv'], '--provider-reference')))
        require(reference == report['provider_reference'], 'root19 reference declaration differs from actual original observation')
        require(reference['assertions']['private_reference_hard_isolation_admitted']
                and reference['assertions']['single_helper_hard_isolation_refused'], 'root19 architecture split missing')
        lazy = read(work / 'lazy/observed/result.json')
        require(lazy['activity_before_execution'] == dict(launched_stores=0, remote_callback_attempts=0)
                and lazy['activity_after_execution'] == dict(launched_stores=1, remote_callback_attempts=0)
                and len(lazy['incarnations']) == 1, 'root19 actual lazy ordinary stores differ')
        template = read(option(entry['argv'], '--input'))
        original_module = template['configuration']['incarnations'][0]['module']
        for item in report['cells']:
            cell = item['cell']; expected = check.configuration(template, cell)
            for incarnation in expected['incarnations']:
                incarnation['module'] = dict(file='target/module.wasm', bytes=original_module['bytes'],
                                            sha256=original_module['sha256'], limit=512 << 20)
            artifact = Path(item['artifact'])
            if cell == 'lazy' or cell.startswith(('cleanup-', 'unbound-')):
                check.check_raw(artifact, cell, expected)
            else:
                common = load('live-backend-common')
                common.seed['authenticate_artifact'](artifact)
                check.check_artifact(artifact, cell, expected)
                retained = read(artifact / 'evidence/cluster-replay.json')
                require(retained['helper_sha256'] == config['identity']['helper'], 'foreign retained cluster helper')
        admissions(work, commands)
    elif family in ('broker', 'owners'):
        check = load('live-broker-cluster-check' if family == 'broker' else 'live-broker-owner-check')
        entry = config['live_corpus']['families'][family]
        reference = Path(option(entry['argv'], '--observation'))
        index = read(reference / 'observation-index.json'); derivation = read(reference / 'derivation.json')
        observe = load('live-broker-observe'); observe.validate_versions(index, derivation)
        require(sha(reference / 'derivation.json') == index['derivation_sha256'] and sha(reference / 'go-test.jsonl') == index['go_test_sha256'], 'foreign original broker observations')
        rows, brokers, runs = [], [], []
        for name, descriptor in index['files'].items():
            path = reference / 'observations' / name
            require(Path(name).name == name and path.stat().st_size == descriptor['bytes'] and sha(path) == descriptor['sha256'], 'changed original owner payload')
            if name.endswith('.json'): observe.validate_row(read(path), index)
            if name.startswith('cluster-'): rows.append(read(path))
            if name.startswith('broker-'): brokers.append(read(path))
            if name.startswith('observation-'): runs.append(read(path))
        observe.validate_events((reference / 'go-test.jsonl').read_bytes(), list(FAMILIES[family][1]))
        result = read(work / 'result.json')
        require(result['executions'] == len(commands) == (77 if family == 'broker' else 16), 'incomplete original broker/owner cell matrix')
        if family == 'broker':
            rows = sorted(rows + check.broker_rows(brokers, runs, [32, 39]), key=lambda row: row['ordinal'])
            require(len(rows) == 77, 'wrong original broker matrix')
            for index, (row, item) in enumerate(zip(rows, commands)):
                require(row['test'] == item['test'] and 'comparison' in item, 'missing actual semantic comparison')
                actual_input = read(work / f'cell-{index}.json')
                expected_input = check.linked_input(row)
                require(actual_input['configuration'] == expected_input['configuration']
                        and actual_input['modules'] == expected_input['modules'], 'changed original broker module/resource/lifecycle input')
                states = {(state['handle']['ID'], state['handle']['Incarnation']): state for state in row['states']}
                forced = {}
                for step in row['steps']:
                    if step['action'] not in ('start', 'restart') or step['runtime']['Choice']['Mode'] == 1: continue
                    original = states[step['node'], step['incarnation']]['result']; choice = step['runtime']['Choice']
                    raw = check.b64(choice.get('Trace') or (original['InterruptedChoiceBytes'] if original['RuntimeInterrupted'] else original['ChoiceTrace']['Bytes']))
                    key = step['node'] + '/' + str(step['incarnation']); forced[key] = raw
                    actual = actual_input['forcing'][key]
                    require(actual['mode'] == (3 if original['RuntimeInterrupted'] else choice['Mode'])
                            and actual['identity'] == choice['ExecutionIdentity'] and Path(actual['trace_path']).read_bytes() == raw,
                            'missing/changed actual forced original guest trace')
                require(set(actual_input['forcing']) == set(forced), 'missing/extra original replay control')
                require(check.compare(work / f'cell-{index}', row, process_exit=item['exit']) == item['comparison'], 'raw broker evidence changed')
            diversity = {}
            for index, row in enumerate(rows):
                if not row['test'].startswith(check.observe.ROOTS[36] + '/') or not check.independent_stream_recording(row): continue
                current = work / f'cell-{index}' / 'incarnations/a/1'
                stdout = (current / 'stdout').read_bytes()
                witness = re.search(rb'(?m)^runnable order(?: [0-3]){4}$', stdout)
                require(witness is not None and b'fresh heap 1' in stdout, 'missing actual runnable/heap witness')
                fixture = row['test'].split('/')[-1]
                require((b'stall expired' if fixture == 'stall' else b'ordered bytes ordered-byte') in stdout,
                        'missing actual independent guest stream witness')
                diversity.setdefault(fixture, []).append([sha(current / 'choice'), witness.group().decode()])
            require(result['diversity'] == diversity, 'declared diversity differs from actual raw guest evidence')
            require(set(result['diversity']) == {'partition-heal-delay', 'stall', 'live-link'}, 'missing original diversity fixture')
            for samples in result['diversity'].values():
                require(len(samples) == 3 and len({item[0] for item in samples}) >= 2 and len({item[1] for item in samples}) >= 2, 'original diversity bounds not met')
        else:
            originals = [item for item in runs if item['remote_model'] and item['test'].split('/')[0] in (observe.ROOTS[37], observe.ROOTS[38])]
            require(len(originals) == 6, 'missing original owner configuration')
            for item in commands:
                matched = [row for row in originals if digest(json.dumps(row, sort_keys=True).encode()) == item['oracle_sha256']]
                require(len(matched) == 1, 'ambiguous/missing actual owner oracle')
                original = matched[0]
                output = Path(item['command'][item['command'].index('--root') + 1])
                check.compare(output, original['result'], original['request']['Runtime']['Choice']['Limit'])

    if family == 'broker': return {'live-case/' + row['test'] for row in rows}


def execute_families(config, stage_root, label):
    for family in config['stages'][label].get('live_families', []):
        entry, base, work = planned(config, family)
        require(base == stage_root and not work.exists(), 'live evidence must be fresh and stage-owned')
        observed = dict(config)
        observed['environment'] = dict(config.get('environment', {}), **entry.get('environment', {}))
        name = 'live-' + family
        if family in AUXILIARY: work.mkdir(mode=0o700)
        raw = command(observed, base, name, entry['argv'])
        if family in AUXILIARY: (work / 'unit.stdout').write_bytes(raw)
        read_family(config, family, work, raw.decode())
        write(base / (name + '.proof.json'), dict(schema='gomad.rust-live-proof/v1', family=family,
              source_sha256=config['source_sha256'], identity=config['identity'], config_sha256=config['_config_sha256'],
              files=inventory(work), command_sha256=sha(base / (name + '.command.json')),
              process_sha256=sha(base / (name + '.process.json')), stdout_sha256=sha(base / (name + '.stdout')),
              stderr_sha256=sha(base / (name + '.stderr'))))


def read_proof(config, family):
    entry, base, work = planned(config, family)
    stage = read(base / 'receipt.json')
    require(stage['stage'] == entry['stage'] and stage['exit'] == 0 and stage['status'] == 'PASS'
            and stage['source'] == config['source_sha256'] and stage['identity'] == config['identity']
            and stage['config_sha256'] == config['_config_sha256'], 'stale/nonpassing live owning stage')
    coverage(subgates(entry['stage']), stage['subgates'])
    name = 'live-' + family
    proof = read(base / (name + '.proof.json'))
    require(proof['schema'] == 'gomad.rust-live-proof/v1' and proof['family'] == family
            and proof['source_sha256'] == config['source_sha256'] and proof['identity'] == config['identity']
            and proof['config_sha256'] == config['_config_sha256'], 'foreign live candidate proof')
    require(proof['files'] == inventory(work), 'raw live evidence missing/changed after producer')
    for suffix in ('command.json', 'process.json', 'stdout', 'stderr'):
        key = suffix.split('.')[0] + '_sha256'
        require(proof[key] == sha(base / (name + '.' + suffix)), 'raw live execution receipt changed')
    completed_command(read(base / (name + '.command.json')), read(base / (name + '.process.json')), entry['argv'], config['repo'])
    actual = read_family(config, family, work, (base / (name + '.stdout')).read_text())
    return dict(proof, actual_cases=sorted(actual or ()))


def reach_coverage(original, entries, actual):
    require(original and set(entries) == set(original), 'missing/unexpected original test-wasm callable/subvariant mapping')
    commands = {}
    for name, entry in entries.items():
        evidence = entry['evidence']
        require(evidence and len(evidence) == len(set(evidence)) and set(evidence) <= actual,
                'unexecuted/duplicate test-wasm receiving obligation: ' + name)
        if '/original-make/' in name:
            root = name.split('/original-make/', 1)[0]
            require(all('@stage/' in item and '@run/' in item for item in evidence),
                    'original repeated Make selection needs actual stage/command testcase proof')
            previous = commands.setdefault(root, set())
            identities = {item.split('@stage/', 1)[1] for item in evidence}
            require(not previous.intersection(identities), 'one execution substituted for repeated original Make selections')
            previous.update(identities)


def original_callables(config, mapping, actual=()):
    commit = mapping['reference_commit']
    require(commit == config['live_corpus']['original_reference_commit'] and re.fullmatch('[0-9a-f]{40}', commit),
            'wrong original test-wasm source commitment')
    inventory = reach.inventory(config['repo'], commit)
    require(mapping['sources'] == inventory['sources'] and mapping['makefiles'] == inventory['makefiles'],
            'incomplete/changed original Make and test-wasm source universe')
    original = set(inventory['callables'])
    for key, detail in inventory['callables'].items():
        if len(detail['original_runs']) > 1:
            original.update(key + '/original-make/' + run for run in detail['original_runs'])
    sites = inventory['subtest_sites']
    require(set(mapping['subtests']) == set(sites), 'unknown/omitted original helper or root subtest site')
    for site, detail in sites.items():
        expansion = mapping['subtests'][site]
        if isinstance(expansion, dict):
            require(expansion == dict(derive='runtime-inspection-boundaries')
                    and site.split('@', 1)[0].endswith('#TestCooperativeInspectionEveryDecision')
                    and detail['expression'] == 'fmt.Sprintf("kind%d-%d", record.Kind, boundary.Decision)',
                    'unsupported original subvariant derivation')
            prefix = 'live-case/TestCooperativeInspectionEveryDecision/'
            expansion = sorted(case.removeprefix(prefix) for case in actual if case.startswith(prefix + 'kind'))
        require(isinstance(expansion, list) and expansion and len(expansion) == len(set(expansion))
                and all(isinstance(case, str) and case for case in expansion),
                'unmapped original literal/dynamic test-wasm subtest site: ' + site)
        if detail['expansions']:
            require(expansion == detail['expansions'], 'original literal subtest was renamed')
        original.update(site + '/' + case for case in expansion)
    require({path + '#' + test for path, test in SLIM_SOURCES.items()} <= original,
            'original Make slim-refusal callable set omitted')
    return original


def slim_command(argv):
    require(len(argv) > 2 and argv[1] == 'test', 'slim proof is not an actual Go test run')
    flags, packages = {}, []
    cursor = 2
    while cursor < len(argv):
        token = argv[cursor]
        if not token.startswith('-'):
            packages.append(token); cursor += 1; continue
        require(not packages, 'Go flags after package selection are not the original slim command')
        name, separator, value = token.partition('=')
        require(name in ('-json', '-v', '-tags', '-timeout', '-run', '-count') and name not in flags,
                'unknown/duplicate original slim flag: ' + name)
        if name in ('-json', '-v'):
            require(not separator, 'changed original slim boolean flag')
            value = True
        elif not separator:
            cursor += 1
            require(cursor < len(argv) and not argv[cursor].startswith('-'), 'missing original slim flag value')
            value = argv[cursor]
        flags[name] = value
        cursor += 1
    require(flags.get('-json') is True and flags.get('-tags') == 'gomad_wasm_slim,test_dep'
            and flags.get('-timeout') == '15m' and flags.get('-count') == '1',
            'original slim tags/15minute bound/fresh count changed')
    pattern = '|'.join('^' + name + '$' for name in SLIM_SOURCES.values())
    require(flags.get('-run') == pattern and packages == ['./common/archiver/provider', './temporal', './tests/testcore'],
            'original three-root slim selector/package role changed')


def receiving_report(config, endpoint):
    stage = endpoint['stage']; base = Path(config['root']) / 'stages' / stage
    stage_receipt = read(base / 'receipt.json')
    require(stage_receipt['stage'] == stage and stage_receipt['status'] == 'PASS' and stage_receipt['exit'] == 0
            and stage_receipt['source'] == config['source_sha256'] and stage_receipt['identity'] == config['identity']
            and stage_receipt['config_sha256'] == config['_config_sha256'], 'unexecuted/foreign test-wasm receiving stage')
    coverage(subgates(stage), stage_receipt['subgates'])
    argv = config['stages'][stage]['checker']
    require(len(argv) >= 2 and argv[1].endswith('.py'), 'test-wasm receiving reader must be pinned code')
    for path in argv[:2]: sealed_input(config, path)
    completed_command(read(base / 'reader.command.json'), read(base / 'reader.process.json'), argv, config['repo'])
    report = read(base / 'reader.stdout')
    require(report['schema'] == 'gomad.rust-driver-check/v1' and report['stage'] == stage
            and report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity']
            and report['config_sha256'] == config['_config_sha256'], 'test-wasm receiving reader binding mismatch')
    require(argv[1] == str(HERE / 'static_stages.py') and option(argv, '--stage') == stage,
            'test-wasm receiving report must use the actual retained/static reader')
    plan = option(argv, '--plan'); require(plan is not None, 'missing actual static plan')
    sealed_input(config, plan)
    spec = importlib.util.spec_from_file_location('live_static_receiving', HERE / 'static_stages.py')
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    computed = module.authenticate_result(Path(plan), stage)
    require(computed == report, 'receiving report differs from actual raw test reports')
    native_ids = [item['id'] for item in report['testcases']]
    executions = [dict(item, id=item['id'] + '@' + item['raw']) for item in report['testcases']]
    coverage([item['id'] for item in executions], executions)
    require(endpoint['required'] and set(endpoint['required']) <= set(native_ids), 'missing exact receiving testcase')
    for item in report['testcases']:
        if item['id'] in SLIM_IDS:
            require(re.fullmatch(r'command-[0-9]{4}', item['raw']), 'slim testcase has no actual raw command reference')
            base = Path(report['static_execution']['path']).parent
            raw = base / (item['raw'] + '.stdout')
            receipt = base / (item['raw'] + '.command.json')
            bound = {row['path'] for row in report['raw_reports']}
            require(str(raw) in bound and str(receipt) in bound, 'slim run is outside authenticated raw reports')
            slim_command(read(receipt)['argv'])
    for raw in report['raw_reports']:
        require(sha(raw['path']) == raw['sha256'], 'actual receiving testcase raw report changed')
    return {item['id'] for item in report['testcases']} | {
        item['id'] + '@stage/' + stage + '@run/' + item['raw'] for item in report['testcases']}


def test_wasm_reach(config, actual):
    entry = config['live_corpus'].get('test_wasm_reach')
    if not isinstance(entry, dict): raise Unexecuted('complete original test-wasm reach map is not integrated')
    sealed_input(config, entry['map'])
    mapping = read(entry['map'])
    require(mapping['schema'] == 'gomad.original-test-wasm-reach/v1', 'wrong original full test-wasm map')
    original = original_callables(config, mapping, actual)
    reports = set(actual)
    for endpoint in entry.get('reports', []): reports.update(receiving_report(config, endpoint))
    require(SLIM_IDS <= reports, 'three original slim-refusal executions are missing')
    reach_coverage(original, mapping['entries'], reports)


def read_corpus(config, root):
    if not isinstance(config.get('live_corpus'), dict): raise Unexecuted('actual live corpus endpoints are not integrated')
    require(isinstance(config.get('root'), str) and root == Path(config['root']), 'wrong final evidence root')
    proofs = {family: read_proof(config, family) for family in FAMILIES}
    rust_admission = config['live_corpus'].get('root19_rust_admission')
    require(isinstance(rust_admission, dict), 'missing actual Rust hard-isolation refusal proof')
    admission_case = 'rust/gomad/campaign::scenario::fidelity_tests::single_linked_helper_refuses_hard_isolation'
    require(admission_case in receiving_report(config, rust_admission), 'root19 Rust refusal testcase unexecuted')
    actual = set(required()) - {'test-wasm/complete-behavioral-reach'}
    actual.update(case for proof in proofs.values() for case in proof['actual_cases'])
    test_wasm_reach(config, actual)
    return [dict(id=name, status='PASS', executed=True, exit=0) for name in required()]
