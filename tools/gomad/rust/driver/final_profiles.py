#!/usr/bin/env python3
"""Independent readers for final terminal and fn-185 R4 preservation evidence."""
import base64
import copy
import hashlib
import struct
from pathlib import Path

from driver import read, require

OWNED = (
    'terminal/current-host-admission-and-linked-attribution',
    'fn185-R4/runtime-cost-equivalent-release-behavior',
    'fn185-R4/qualified-workload-choice-wasi-network-stdout-stderr-preservation',
)
WORKFLOW = '^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$'
ACTIVITY = '^TestActivityTestSuite$'
STREAMS = ('stdout', 'stderr', 'wasi', 'choice', 'network', 'terminal')


def host_admission(goos, goarch, raw):
    require(goos == 'linux' and goarch == 'arm64', 'current host must be Linux/arm64')
    require(isinstance(raw, bytes) and 0 < len(raw) <= 1 << 20, 'CPU witness exceeds original 1MiB bound or is empty')
    architecture, asimd = False, False
    for line in raw.decode('utf-8', errors='replace').split('\n'):
        key, delimiter, value = line.partition(':')
        if delimiter and key.strip() == 'CPU architecture':
            architecture |= value.strip() == '8'
        if delimiter and key.strip() == 'Features':
            asimd |= 'asimd' in value.split()
    require(architecture and asimd, 'current CPU lacks original architecture/asimd admission')
    return hashlib.sha256(raw).hexdigest()


def bounded_path(root, name):
    path = Path(name)
    require(path.is_absolute() and path.is_relative_to(root) and path.resolve() == path
            and path.is_file() and not path.is_symlink(), 'missing/foreign raw profile output')
    return path


def evidence(root, run):
    require(not run.get('synthetic', False) and run['reaped'] is True,
            'synthetic/unreaped execution')
    streams = {}
    require(set(run['streams']) == set(STREAMS), 'missing/extra canonical stream')
    for name, entry in run['streams'].items():
        if name == 'terminal' and isinstance(entry.get('native_bytes'), bytes):
            raw = entry['native_bytes']
        else:
            path = bounded_path(root, entry['path'])
            raw = path.read_bytes()
        require(entry['sha256'] == hashlib.sha256(raw).hexdigest(), 'changed canonical stream: ' + name)
        streams[name] = raw
    require(len(streams['terminal']) == 16, 'missing original runtime finish terminal')
    return streams


def same_evidence(root, before, after, *, qualified):
    left, right = evidence(root, before), evidence(root, after)
    for name in STREAMS:
        require(left[name] == right[name], 'profiling changed canonical ' + name)
    for field in ('termination', 'exit_code'):
        require(before[field] == after[field], 'profiling changed canonical ' + field)
    if qualified:
        for field in ('calls', 'time_advances', 'environment_identity', 'fuel_remaining'):
            require(before[field] == after[field], 'profiling changed canonical ' + field)
    require(before['termination'] == 'exit' and before['exit_code'] == 0,
            'original successful workload did not exit zero')
    return left


def exact_bounds(spec, *, runtime=False):
    require(spec['profile'] == 'gomad.wasi-cooperative/v1' and spec['clock_read_step_nanos'] == 0,
            'changed original cooperative profile/clock')
    require(spec['fuel'] == (1 << 40 if runtime else 1 << 50)
            and spec['memory_bytes'] == (256 << 20 if runtime else 4 << 30),
            'changed original fuel/memory bounds')
    require(spec['wall_nanos'] == (180_000_000_000 if runtime else 1_800_000_000_000),
            'changed original workload deadline')
    if runtime:
        require(spec['seed'] == 7 and spec['package'] == './internal/model/wasi/testdata/runtime_cost',
                'changed original runtime seed/package')
        require(spec['calls'] == 1 << 20 and spec['transcript_bytes'] == 64 << 20
                and spec['choice_bytes'] == 64 << 20 and spec['runtime_diagnostics'] is False,
                'changed original runtime-cost bounds')
    else:
        require(spec['package'] == './tests' and spec['working_directory'] == '/workspace/tests'
                and spec['selection'] in (WORKFLOW, ACTIVITY), 'substituted original Temporal selection')
        require(spec['arguments'] == ['-test.run=' + spec['selection'], '-persistenceType=sql',
                '-persistenceDriver=sqlite'], 'changed original Temporal arguments')


def bound_native_request(config, path, spec, *, runtime=False, dynamic_root=None):
    import live
    from driver import sha
    if dynamic_root is None:
        live.sealed_input(config, path)
    else:
        bounded_path(dynamic_root, str(path))
    request = read(path)
    if dynamic_root is None:
        require(request == spec['native_input'], 'actual native request differs from frozen original recipe')
    require(request['kind'] == 'standalone', 'original standalone recipe replaced')
    guest = request['guest']
    if dynamic_root is not None:
        expected = copy.deepcopy(spec['native_input'])
        expected['guest']['module'] = guest['module']
        expected['guest']['module_sha256'] = guest['module_sha256']
        if 'compile_cache' in request:
            cache = Path(request['compile_cache'])
            require(cache.is_absolute() and cache.resolve() == cache and cache.is_relative_to(dynamic_root),
                    'foreign dynamic runtime compile cache')
            expected['compile_cache'] = str(cache)
        require(request == expected, 'dynamic runtime request changed original modeled semantics')
    if dynamic_root is None:
        live.sealed_input(config, guest['module'])
    else:
        bounded_path(dynamic_root, guest['module'])
    require(sha(guest['module']) == guest['module_sha256'], 'original module changed')
    limits = guest['limits']
    require(limits['fuel'] == spec['fuel'] and limits['memory_bytes'] == spec['memory_bytes']
            and limits['wall_millis'] * 1_000_000 == spec['wall_nanos'], 'native bounds differ from original recipe')
    model = guest['model']
    require(model['configuration']['read_step_nanos'] == spec['clock_read_step_nanos'], 'native original clock changed')
    require(model['configuration']['seed'] == spec['seed'], 'native original seed changed')
    if runtime:
        require(model['configuration']['limits']['calls'] == spec['calls']
                and model['configuration']['limits']['transcript_bytes'] == spec['transcript_bytes']
                and model['configuration']['limit'] == spec['choice_bytes']
                and model['configuration']['diagnostics'] == spec['runtime_diagnostics'], 'native original runtime bounds changed')
        preparation_proof(config, spec, guest, runtime=True, dynamic_root=dynamic_root)
    else:
        require(model['arguments'][1:] == spec['arguments'], 'native original Temporal arguments changed')
        cwd = spec['working_directory'].encode()
        require(model['configuration']['working_directory_sha256'] == hashlib.sha256(cwd).hexdigest()
                and guest['namespace']['working_directory'] == list(cwd), 'native original Temporal working directory changed')
        preparation_proof(config, spec, guest)
        require(guest['module'] == spec['module'] and guest['module_sha256'] == spec['module_sha256'],
                'native original Temporal module changed')
    return request


def preparation_proof(config, spec, guest, *, runtime=False, dynamic_root=None):
    import live
    import preparation
    if dynamic_root is None:
        proof = spec['preparation']
        for field in ('provenance', 'installation', 'observations'):
            live.sealed_input(config, proof[field])
    else:
        module_root = Path(guest['module']).parent
        proof = dict(provenance=str(bounded_path(dynamic_root, str(module_root / 'module-provenance.json'))),
                     installation=str(bounded_path(dynamic_root, str(module_root / 'installation.json'))),
                     observations=str(module_root))
    provenance, installation = read(proof['provenance']), read(proof['installation'])
    inputs = provenance['inputs']
    require(provenance['module_sha256'] == guest['module_sha256']
            and inputs['kind'] == ('go-run' if runtime else 'go-test')
            and inputs['source'] == ('./internal/model/wasi/testdata/runtime_cost' if runtime else './tests')
            and inputs['directory'] == (str(Path(config['repo']) / 'tools/gomad') if runtime else config['repo']),
            'module was not prepared from original unchanged source selection')
    require(inputs['sources'] and all(config['files'].get(path) == value for path, value in inputs['sources'].items()),
            'qualified original source closure is not frozen')
    preparation.Inventory(config).project(proof['observations'], installation, inputs)


def runtime_pair(root, spec, release, diagnostic):
    exact_bounds(spec, runtime=True)
    streams = same_evidence(root, release, diagnostic, qualified=False)
    require(b'runtime-cost 100000' in streams['stdout'], 'missing original runtime-cost loop witness')
    for run in (release, diagnostic):
        require(type(run['authorization_to_terminal_nanos']) is int and run['authorization_to_terminal_nanos'] > 0
                and type(run['select_poll']) is int and run['select_poll'] >= 100000,
                'missing original runtime-cost timing/select witness')


def select_polls(raw):
    require(raw and len(raw) % 96 == 0 and len(raw) <= 64 << 20,
            'malformed original runtime-cost choice framing')
    polls = 0
    for ordinal in range(len(raw) // 96):
        position, kind, flags, _, alternatives, selected = struct.unpack_from('>QBBHII', raw, ordinal * 96)
        require(position == ordinal and 1 <= kind <= 4 and flags & ~7 == 0 and alternatives > selected,
                'malformed original runtime-cost choice record')
        polls += kind == 2
    return polls


def attribution(profile):
    require(profile['schema'] == 'gomad.wasm-profile/v2' and profile['diagnostic_only'] is True,
            'missing actual linked attribution profile')
    for field in ('abi_overlap_errors', 'span_errors', 'unattributed_imports', 'unframed_imports',
                  'threads_omitted', 'report_rows_dropped'):
        require(type(profile[field]) is int and profile[field] == 0, 'incomplete linked attribution: ' + field)
    def number(value):
        require(type(value) is int and 0 <= value < 1 << 64, 'invalid linked attribution counter')
        return value
    def row(value):
        execute, attributed, residual, outer = (number(value[field]) for field in
            ('execute_wall_nanos', 'attributed_host_wall_nanos', 'guest_residual_wall_nanos', 'outer_host_wall_nanos'))
        difference, tolerance = abs(execute - attributed - residual), max(50000, execute // 200)
        require(residual + outer == execute and value['difference_nanos'] == difference
                and value['tolerance_nanos'] == tolerance and difference <= tolerance,
                'linked attribution reconciliation miss')
        return execute, attributed, residual, outer
    totals = row(profile['reconciliation'])
    require(totals[0] > 0 and profile['phases']['execute']['wall_nanos'] == totals[0], 'unmeasured linked execution')
    owners, ops = profile['owners'], profile['abi_ops']
    require(owners and ops, 'missing linked import owner tables')
    for name, operation in ops.items():
        require(operation['served_by'] and set(operation['served_by']) <= set(owners)
                and sum(number(count) for count in operation['served_by'].values()) == number(operation['count']),
                'unattributed linked import: ' + name)
    attributed = sum(number(owner['execute_exclusive_wall_nanos']) for owner in owners.values())
    glue = owners.get('instrumentation', {}).get('execute_exclusive_wall_nanos', 0)
    require(attributed == totals[1] and attributed - glue == sum(number(op['execute_wall_nanos']) for op in ops.values()),
            'linked owner/import wall totals differ')
    require(profile['reconciliation']['distribution_relative_error_ppm'] == 62500, 'changed bounded distribution precision')
    if profile['threads']:
        rows = [row(thread) for thread in profile['threads']]
        require(tuple(sum(values) for values in zip(*rows)) == totals, 'linked store thread totals differ')


def native_run(root):
    report = read(root / 'report.json')
    require(report['schema'] == 'gomad.linked-profile-run/v1' and report['execution'].get('Ok'),
            'missing/failed native linked execution')
    outcome = read(root / 'outcome.json')
    request = read(root / 'request.json')
    catalog = read(root / 'streams.json')
    streams = {}
    for name in STREAMS[:-1]:
        require(name in catalog, 'native producer has not retained original canonical ' + name)
        streams[name] = dict(path=str(root / name), sha256=catalog[name]['sha256'])
    model = outcome['local_model']
    terminal = base64.b64decode(model['terminal_base64'], validate=True)
    streams['terminal'] = dict(native_bytes=terminal, sha256=hashlib.sha256(terminal).hexdigest())
    choice = (root / 'choice').read_bytes()
    polls = select_polls(choice)
    result = dict(streams=streams, select_poll=polls, reaped=True, termination=outcome['termination'],
                  exit_code=outcome.get('exit_code'), calls=model['calls'], time_advances=model['advances'],
                  environment_identity=request['guest']['model'], fuel_remaining=outcome['fuel_remaining'],
                  authorization_to_terminal_nanos=report['authorization_to_terminal_nanos'])
    return report, result


def sealed_role(config, entry, expected):
    import live
    for field in ('binary', 'build_receipt'):
        live.sealed_input(config, entry[field])
    build = read(entry['build_receipt'])
    from driver import sha
    require(build['profile'] == 'release' and build['exit'] == 0
            and build['source_sha256'] == config['source_sha256']
            and build['binary_sha256'] == sha(entry['binary'])
            and build['helper_sha256'] == config['identity']['helper']
            and build['helper_features'] == expected, 'foreign/nonmatching native producer role')
    return entry['binary']


def invoke(config, root, name, argv):
    import live
    from driver import command
    raw = command(config, root, name, argv)
    receipt, process = read(root / (name + '.command.json')), read(root / (name + '.process.json'))
    live.completed_command(receipt, process, argv, config['repo'])
    return raw, process


def collect(config, root):
    import json
    import live
    from driver import Unexecuted, guard, sha
    guard(config)
    require(root == Path(config['root']), 'foreign final profile cycle root')
    entry = config.get('final_profiles')
    if not entry:
        raise Unexecuted('missing final terminal/runtime/preservation producers')
    measure = sealed_role(config, entry['measure'], [])
    ordinary = sealed_role(config, entry['ordinary'], [])
    diagnostic = sealed_role(config, entry['diagnostic'], ['diagnostic-profile'])
    for field in ('stock_go', 'terminal_module', 'terminal_module_receipt', 'terminal_helper_receipt'):
        live.sealed_input(config, entry[field])
    work = root / 'final-profiles'
    require(not work.exists(), 'final profile outputs must be fresh')
    work.mkdir(mode=0o700)
    runtime_root, terminal_root = work / 'runtime', work / 'terminal'
    common = ['--repo', config['repo'], '--stock-go', entry['stock_go'], '--test-helper', ordinary]
    runtime_argv = [measure, 'runtime-cost-profile', *common, '--profile-root', str(runtime_root)]
    raw, _ = invoke(config, root, 'final-profile-runtime', runtime_argv)
    envelope = json.loads(raw)
    runtime_report = read(bounded_path(runtime_root, envelope['result']))
    require(runtime_report['schema'] == 'gomad.native-named-measurement/v1'
            and runtime_report['producer'] == 'runtime-cost-profile'
            and runtime_report['failure'] is None and runtime_report['accepted'] is True,
            'native runtime-cost producer failed')
    runs = runtime_report['runs']
    require(len(runs) == 2 and all(row['status'] == 'executed' for row in runs),
            'missing runtime original owner pair')
    # An ordinary/ordinary repetition is not the original owner comparison.
    if [row['sample'].get('owner') for row in runs] != ['remote-model', 'helper-model']:
        raise Unexecuted('runtime-cost original remote-model/helper-model comparison is unexecuted; ordinary/ordinary repeats supply no receiving proof')
    observed = [native_run(bounded_path(runtime_root, str(Path(row['native_root']) / 'report.json')).parent) for row in runs]
    live.sealed_input(config, entry['runtime_recipe'])
    runtime_spec = read(entry['runtime_recipe'])
    for row, (report, _) in zip(runs, observed):
        native_root = Path(row['native_root'])
        request_path = native_root / 'request.json'
        bound_native_request(config, request_path, runtime_spec, runtime=True, dynamic_root=runtime_root)
        require(report['request_sha256'] == sha(request_path)
                and report['host_sha256'] == sha(ordinary)
                and report['linked_implementation_sha256'] == config['identity']['helper']
                and report['features'] == [], 'foreign runtime original native request/role')
    runtime_pair(work, runtime_spec, observed[0][1], observed[1][1])
    terminal_argv = [measure, 'terminal-exit', *common, '--terminal-exit-root', str(terminal_root),
        '--terminal-exit-module', entry['terminal_module'], '--terminal-exit-module-receipt',
        entry['terminal_module_receipt'], '--terminal-exit-helper-receipt', entry['terminal_helper_receipt']]
    raw, process = invoke(config, root, 'final-profile-terminal', terminal_argv)
    terminal = read(bounded_path(terminal_root, json.loads(raw)['result']))
    require(terminal['schema'] == 'gomad.native-terminal-exit/v1', 'foreign native terminal report')
    if len(terminal['observations']) != 2 or any('root' in item for item in terminal['observations']):
        raise Unexecuted('native terminal producer lacks original post-load authorization-mismatch observation')
    cpu = bounded_path(terminal_root, terminal['cpuinfo_path']).read_bytes()
    require(host_admission(terminal['host_os'], terminal['host_arch'], cpu) == terminal['cpuinfo_sha256'],
            'changed current host CPU capture')
    require(terminal['operator_process_id'] == process['pid'], 'foreign terminal observation PID')
    terminal_observations(terminal['observations'])
    pairs = entry['qualified']
    require(set(pairs) == {'workflow', 'activity'}, 'missing original qualified workload')
    for name, pair in pairs.items():
        live.sealed_input(config, pair['recipe'])
        recipe = read(pair['recipe'])
        exact_bounds(recipe)
        require(recipe['selection'] == (WORKFLOW if name == 'workflow' else ACTIVITY),
                'wrong qualified workload selection')
        actual = []
        for role, binary in (('release', ordinary), ('diagnostic', diagnostic)):
            live.sealed_input(config, pair[role + '_input'])
            output = work / (name + '-' + role)
            _, process = invoke(config, root, 'final-profile-' + name + '-' + role,
                                ([binary, '--linked-run', pair[role + '_input'], str(output)] if role == 'release'
                                 else [binary, pair[role + '_input'], str(output)]))
            report, run = native_run(output)
            require(read(output / 'request.json') == bound_native_request(config, pair[role + '_input'], recipe),
                    'retained native request differs from actual original input')
            require(report['fresh_store_id'] == process['pid']
                    and report['request_sha256'] == sha(pair[role + '_input'])
                    and report['host_sha256'] == sha(binary)
                    and report['linked_implementation_sha256'] == config['identity']['helper']
                    and report['features'] == ([] if role == 'release' else ['diagnostic-profile']),
                    'foreign qualified native execution')
            if role == 'diagnostic':
                attribution(read(output / 'profile.json'))
            actual.append(run)
        same_evidence(work, *actual, qualified=True)
    guard(config)
    return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                identity=config['identity'], subgates=[dict(id=name, status='PASS', executed=True, exit=0) for name in OWNED])


def terminal_observations(observations):
    require(len(observations) == 2, 'missing original success/authorization-mismatch terminal pair')
    success, mismatch = observations
    for item in observations:
        require(item['reaped'] is True and type(item['terminal_to_reap_nanos']) is int
                and 0 < item['terminal_to_reap_nanos'] <= 1_000_000_000,
                'original terminal-to-reap one-second bound exceeded/unmeasured')
    require(success['termination'] == 'exit' and success['exit_code'] == 0
            and '--- PASS: TestWorkflowTypeEncodingSuite' in success['stdout']
            and 'TestPlainASCII/Succeeds' in success['stdout'], 'original terminal workflow did not pass')
    require(mismatch['authorization_changed'] is True and mismatch['termination'] == 'invalid'
            and mismatch['message'] == 'execution authorization identity mismatch'
            and mismatch['started'] is not None and mismatch['instantiate_nanos'] == 0
            and mismatch['execute_nanos'] == 0, 'original post-load authorization refusal changed')


def main():
    import argparse
    import json
    import sys
    from driver import Refusal, configured, sha
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config['_config_path'] = str(args.config.resolve())
        config['_config_sha256'] = sha(args.config)
        print(json.dumps(collect(config, args.root.resolve()), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, TypeError, AssertionError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
