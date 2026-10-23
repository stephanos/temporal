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
        preparation_proof(config, spec, guest, dynamic_root=dynamic_root)
        require((dynamic_root is not None or guest['module'] == spec['module'])
                and guest['module_sha256'] == spec['module_sha256'],
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
                     observations=str(module_root / 'preparation-observations'))
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


def native_canonical_streams(root, outcome):
    from driver import sha
    native = read(root / 'streams.json')
    receipt = read(root / 'canonical-streams.json')
    require(receipt['schema'] == 'gomad.native-canonical-streams/v1'
            and receipt['source'] == 'finalized linked Execution.evidence from closed Sink::into_streams'
            and receipt['transformation'] == 'identity bytes; no content normalization'
            and receipt['native_streams_sha256'] == sha(root / 'streams.json')
            and receipt['outcome_sha256'] == sha(root / 'outcome.json')
            and set(receipt['streams']) == set(STREAMS), 'missing/foreign native canonical stream receipt')
    raw_native = {}
    for name, entry in native.items():
        require(Path(name).name == name and name not in ('.', '..'), 'foreign native stream name')
        raw = bounded_path(root, str(root / name)).read_bytes()
        require(len(raw) == entry['bytes'] and hashlib.sha256(raw).hexdigest() == entry['sha256'],
                'changed native evidence stream')
        raw_native[name] = raw
    streams = {}
    for name, entry in receipt['streams'].items():
        require(entry['file'] == name, 'foreign native canonical file')
        path = bounded_path(root, str(root / name))
        raw = path.read_bytes()
        require(len(raw) == entry['bytes'] and sha(path) == entry['sha256'], 'changed native canonical stream')
        if name in ('stdout', 'stderr', 'network', 'choice'):
            if name in raw_native:
                require(entry['source'] == 'native:' + name and raw == raw_native[name], 'changed canonical native source')
            else:
                require(name in ('stdout', 'stderr', 'network') and not raw
                        and entry['source'] == 'closed-sink-empty:' + name, 'unobserved native canonical stream')
        elif name == 'wasi':
            require(entry['source'] == 'binary-transcript-alias:transcript' and 'transcript' in raw_native
                    and raw == raw_native['transcript']
                    and ('wasi' not in raw_native or raw == raw_native['wasi']), 'changed binary WASI alias')
        else:
            require(entry['source'] == 'outcome:local_model.terminal_base64' and 'terminal' not in raw_native
                    and raw == base64.b64decode(outcome['local_model']['terminal_base64'], validate=True)
                    and len(raw) == 16, 'changed actual native model terminal')
        streams[name] = dict(path=str(path), sha256=entry['sha256'])
    return streams


def native_run(root):
    report = read(root / 'report.json')
    require(report['schema'] == 'gomad.linked-profile-run/v1' and report['execution'].get('Ok'),
            'missing/failed native linked execution')
    outcome = read(root / 'outcome.json')
    request = read(root / 'request.json')
    streams = native_canonical_streams(root, outcome)
    model = outcome['local_model']
    polls = select_polls((root / 'choice').read_bytes())
    result = dict(streams=streams, select_poll=polls, reaped=True, termination=outcome['termination'],
                  exit_code=outcome.get('exit_code'), calls=model['calls'], time_advances=model['advances'],
                  environment_identity=request['guest']['model'], fuel_remaining=outcome['fuel_remaining'],
                  authorization_to_terminal_nanos=report['authorization_to_terminal_nanos'])
    return report, result


def sealed_role(config, entry, expected):
    import json
    import live
    from driver import sha
    for field in ('binary', 'build_receipt'):
        live.sealed_input(config, entry[field])
    build = read(entry['build_receipt'])
    binary = entry['binary']
    if build.get('schema') == 'gomad.native-measurement-build/v1':
        require(build['executable'] == binary and build['executable_sha256'] == sha(binary)
                and build['role'] == (dict(kind='diagnostic', features=expected) if expected else dict(kind='release')),
                'foreign native role build receipt')
        for path, digest in build['inputs'].items():
            live.sealed_input(config, path)
            require(sha(path) == digest, 'native role build input changed')
    else:
        require(build['profile'] == 'release' and build['exit'] == 0
                and build['source_sha256'] == config['source_sha256']
                and build['binary_sha256'] == sha(binary) and build['helper_features'] == expected,
                'foreign/nonmatching native producer build')
    identities = config.setdefault('_final_profile_role_identities', {})
    if binary not in identities:
        raw, _ = invoke(config, Path(config['root']), 'final-profile-role-' + str(len(identities)), [binary, '--identity'])
        identity = json.loads(raw)
        require(identity['schema'] == 'gomad.native-measurement-identity/v1'
                and identity['executable_sha256'] == sha(binary) and identity['features'] == expected
                and identity['role'] == ('diagnostic-profile' if expected else 'ordinary'), 'foreign actual native role identity')
        if not expected:
            require(identity['host_implementation_sha256'] == config['identity']['host']
                    and identity['linked_implementation_sha256'] == config['identity']['helper'],
                    'ordinary measurement implementation differs from frozen release')
            config['_final_profile_release_identity'] = identity
        else:
            release = config['_final_profile_release_identity']
            require(identity['source_inputs'] == release['source_inputs']
                    and identity['runtime_implementation_sha256'] == release['runtime_implementation_sha256']
                    and identity['configuration'] == release['configuration'] + '-diagnostic-profile',
                    'diagnostic role source/runtime differs from ordinary release')
        if build.get('schema') == 'gomad.native-measurement-build/v1':
            require(identity == build['identity'], 'native build identity differs from actual executable')
        identities[binary] = identity
    require(identities[binary]['features'] == expected, 'native executable borrowed a different role')
    return binary


def invoke(config, root, name, argv):
    import live
    from driver import command
    raw = command(config, root, name, argv)
    receipt, process = read(root / (name + '.command.json')), read(root / (name + '.process.json'))
    live.completed_command(receipt, process, argv, config['repo'])
    return raw, process


def native_child(config, attempt_root, native_root, binary, features, exit_code=0):
    from driver import sha
    report = read(bounded_path(attempt_root, str(native_root / 'report.json')))
    journal = read(bounded_path(attempt_root, str(native_root.parent / (native_root.name + '-driver.json'))))
    input_path = native_root.parent / (native_root.name + '.input.json')
    require(journal['schema'] == 'gomad.native-process-observation/v1' and not report.get('synthetic', False)
            and journal['command'] == [binary, '--linked-run', str(input_path), str(native_root)]
            and journal['directory'] == config['repo'] and journal['exit_code'] == exit_code
            and journal['signal'] is None and journal['cleanup_errors'] == []
            and journal['termination'] == ('exit' if exit_code == 0 else 'exit_failure')
            and type(journal['pid']) is int and journal['pid'] > 0
            and report['fresh_store_id'] == journal['pid'], 'foreign/incomplete actual native child')
    require(report['host_sha256'] == sha(binary) and report['features'] == features
            and report['linked_implementation_sha256'] == config['_final_profile_role_identities'][binary]['linked_implementation_sha256']
            and report['host_implementation_sha256'] == config['_final_profile_role_identities'][binary]['host_implementation_sha256']
            and report['request_sha256'] == sha(input_path) == sha(native_root / 'request.json'),
            'native child role/request identity changed')
    for name in ('stdout', 'stderr'):
        stream = journal[name]
        path = bounded_path(attempt_root, stream['path'])
        require(stream['error'] is None and stream['truncated'] is False
                and stream['bytes'] == stream['captured_bytes'] == path.stat().st_size
                and stream['sha256'] == sha(path) == journal[name + '_sha256'],
                'native child lost actual command stream')
    return report, journal


def terminal_clock(report, journal):
    terminal, reaped = report['terminal_monotonic_nanos'], journal['reaped_monotonic_nanos']
    require(type(terminal) is int and type(reaped) is int and 0 < terminal <= reaped
            and 0 < reaped - terminal <= 1_000_000_000,
            'original terminal-to-child-reap bound missing/exceeded')
    require(report['terminal_clock'] == 'CLOCK_MONOTONIC', 'foreign native terminal clock')
    linked = report['terminal_to_return_nanos']
    if linked is None:
        linked = report['admission_refusal']['terminal_to_return_nanos']
    require(type(linked) is int and 0 < linked <= reaped - terminal,
            'linked return is not bounded by actual terminal/reap observations')
    return reaped - terminal


def process_exit_observation(observed, report, journal, native_root, elapsed):
    require(observed['schema'] == 'gomad.native-terminal-process-exit/v1'
            and observed['sample'] == native_root.name and observed['clock'] == 'CLOCK_MONOTONIC'
            and observed['pid'] == journal['pid']
            and observed['driver_observation'] == str(native_root.parent / (native_root.name + '-driver.json'))
            and observed['terminal_monotonic_nanos'] == report['terminal_monotonic_nanos']
            and observed['reaped_monotonic_nanos'] == journal['reaped_monotonic_nanos']
            and observed['terminal_to_reap_nanos'] == elapsed and observed['bound_nanos'] == 1_000_000_000,
            'terminal-to-process-exit differs from actual clocks/child')


def admission_attribution(profile):
    require(profile['schema'] == 'gomad.wasm-profile/v2' and profile['diagnostic_only'] is True,
            'missing actual admission profile')
    for field in ('abi_overlap_errors', 'span_errors', 'unattributed_imports', 'unframed_imports',
                  'threads_omitted', 'report_rows_dropped'):
        require(type(profile[field]) is int and profile[field] == 0, 'incomplete admission attribution')
    for phase in ('instantiate', 'execute'):
        require(profile['phases'][phase]['count'] == profile['phases'][phase]['wall_nanos'] == 0,
                'admission refusal entered guest execution')
    require(not profile['abi_ops'] and not profile['owners'] and not profile['threads']
            and all(bucket['count'] == 0 for bucket in profile['abi'].values())
            and profile['abi_excluded']['count'] == 0, 'admission refusal executed guest imports')
    row = profile['reconciliation']
    require(all(row[field] == 0 for field in ('execute_wall_nanos', 'attributed_host_wall_nanos',
            'guest_residual_wall_nanos', 'outer_host_wall_nanos', 'difference_nanos'))
            and row['tolerance_nanos'] == 50000, 'admission attribution reconciliation differs from actual empty execution')


def native_terminal(config, root, work, entry, terminal_path, ordinary, diagnostic):
    from driver import Unexecuted, sha
    terminal = read(terminal_path)
    if terminal['schema'] != 'gomad.native-terminal-exit/v2':
        raise Unexecuted('terminal producer lacks actual pre-engine/loaded-module admission observations')
    base = terminal_path.parent
    host = read(base / 'host.json')
    require(host['schema'] == 'gomad.native-measurement-host/v1' and host['host_os'] == 'linux'
            and host['host_arch'] == 'aarch64', 'terminal current host must be Linux/arm64')
    cpu_path = bounded_path(base, str(base / 'host/host-cpuinfo.txt'))
    cpu = cpu_path.read_bytes()
    require(host_admission('linux', 'arm64', cpu) == host['inputs'][str(cpu_path)]
            and host['witness']['cpuinfo'].encode() == cpu, 'current raw CPU capture changed')
    require(terminal['helper_admission'] == read(entry['terminal_helper_receipt'])
            and terminal['diagnostic_control_admission'] == read(entry['terminal_diagnostic_helper_receipt']),
            'foreign terminal release/diagnostic admission')
    require(len(terminal['observations']) == 6, 'missing exact terminal success observations')
    recipe = read(entry['qualified']['workflow']['recipe'])
    actual, pids = [], set()
    require(len(terminal['process_exit_observations']) == 6, 'missing actual terminal-to-reap publications')
    for index, (name, inline) in enumerate(zip(['prime'] + ['terminal-' + str(index) for index in range(5)], terminal['observations'])):
        native_root = base / 'evidence' / name
        report, journal = native_child(config, base, native_root, ordinary, [])
        elapsed = terminal_clock(report, journal)
        process_exit_observation(terminal['process_exit_observations'][index], report, journal, native_root, elapsed)
        require(journal['pid'] not in pids, 'borrowed terminal success process')
        pids.add(journal['pid'])
        require(inline['report'] == report and inline['request'] == read(native_root / 'request.json')
                and inline['root']['outcome'] == read(native_root / 'outcome.json'), 'inline terminal observation differs from raw execution')
        bound_native_request(config, native_root / 'request.json', recipe, dynamic_root=base)
        _, run = native_run(native_root)
        streams = evidence(work, run)
        require(run['termination'] == 'exit' and run['exit_code'] == 0
                and b'--- PASS: TestWorkflowTypeEncodingSuite' in streams['stdout']
                and b'TestPlainASCII/Succeeds' in streams['stdout'], 'actual terminal workflow did not pass')
        actual.append(run)
    for run in actual[1:]:
        same_evidence(work, actual[0], run, qualified=True)
    rows = terminal['admission_refusals']
    require([row['stage'] for row in rows] == ['module-digest', 'loaded-module-descriptor'],
            'missing actual original admission controls')
    baseline = read(base / 'evidence/prime/request.json')
    for row in rows:
        stage = row['stage']
        native_root = Path(row['native_root'])
        report, journal = native_child(config, base, native_root, diagnostic, ['diagnostic-profile'],
                                       exit_code=1 if stage == 'module-digest' else 0)
        elapsed = terminal_clock(report, journal)
        process_exit_observation(row['process_exit'], report, journal, native_root, elapsed)
        require(journal['pid'] not in pids, 'borrowed terminal refusal process')
        pids.add(journal['pid'])
        request = read(native_root / 'request.json')
        expected = copy.deepcopy(baseline)
        expected.pop('compile_cache', None)
        if stage == 'module-digest':
            expected['guest']['module_sha256'] = '1' * 64 if baseline['guest']['module_sha256'] == '0' * 64 else '0' * 64
        else:
            expected['guest']['limits']['memory_bytes'] = 65536
        require(request == expected and row['request_sha256'] == sha(native_root / 'request.json'),
                'terminal admission control changed undeclared original inputs')
        refusal = report['admission_refusal']
        profile = read(native_root / 'profile.json')
        require(row['refusal'] == refusal and row['profile'] == profile and profile['schema'] == 'gomad.wasm-profile/v2'
                and report['diagnostic_only'] is True and refusal['schema'] == 'gomad.native-admission-refusal/v1'
                and refusal['clock'] == 'monotonic' and row['terminal_to_return_nanos'] == refusal['terminal_to_return_nanos']
                and type(refusal['terminal_to_return_nanos']) is int and 0 < refusal['terminal_to_return_nanos'] <= elapsed,
                'foreign actual refusal timing/profile')
        admission_attribution(profile)
        require(report['attribution'] == dict(Ok=None), 'native admission attribution did not validate')
        if stage == 'module-digest':
            require(refusal['stage'] == 'request-admission' and refusal['error'] == 'module SHA-256 mismatch'
                    and report['execution'] == dict(Err='module SHA-256 mismatch')
                    and not (native_root / 'outcome.json').exists(), 'module digest control was not refused before engine')
            for phase in ('engine_init', 'compile', 'deserialize'):
                require(profile['phases'][phase]['count'] == 0, 'digest refusal created an engine')
        else:
            outcome = read(native_root / 'outcome.json')
            metrics = outcome['metrics']
            require(refusal['stage'] == 'loaded-module-descriptor' and refusal['outcome'] == outcome
                    and outcome['termination'] == 'capacity' and outcome['message'] == 'initial guest memory exceeds limit'
                    and metrics['prepare_nanos'] > 0 and metrics['compile_nanos'] > 0
                    and metrics['instantiate_nanos'] == metrics['execute_nanos'] == 0
                    and profile['phases']['compile']['count'] == 1, 'loaded-module descriptor control did not refuse before instantiation')


def linked_attribution(config, root):
    import adapters
    import receiving
    import receiving_cost
    from driver import coverage, sha, subgates
    receiving.read_receiving(config)
    stage = root / 'stages/workload-corpus'
    receipt = read(stage / 'receipt.json')
    require(receipt['stage'] == 'workload-corpus' and receipt['status'] == 'PASS' and receipt['exit'] == 0
            and receipt['source'] == config['source_sha256'] and receipt['identity'] == config['identity']
            and receipt['config_sha256'] == config['_config_sha256'], 'foreign/unexecuted attribution corpus')
    coverage(subgates('workload-corpus'), receipt['subgates'])
    corpus = read(stage / 'corpus.json')
    require(len(corpus['results']) == len(set(corpus['results'])) == 10, 'missing exact ten-cell attribution corpus')
    for path, digest in corpus['files'].items():
        require(bounded_path(root, path).is_file() and sha(path) == digest, 'changed same-cycle attribution input')
    observed, pids = set(), set()
    for index, path in enumerate(corpus['results']):
        result = adapters.result_report(config, root, 'final-profile-attribution-' + str(index), path)
        raw = read(path)
        require(raw == {key: value for key, value in result.items() if key != '_driver_publication'},
                'attribution public report differs from actual Result')
        receiving_cost.validate(raw)
        manifest = raw['manifest']
        key = tuple(manifest[field] for field in ('workload', 'profile', 'id', 'cell'))
        observed.add(key)
        pid = raw['observations']['operator_process_id']
        require(pid not in pids, 'borrowed attribution operator')
        pids.add(pid)
        publications = list(Path(path).parent.glob(Path(path).name + '.publication-*.json'))
        require(len(publications) == 1 and str(publications[0]) in corpus['files'], 'unbound attribution publication')
        publication = read(publications[0])
        require(publication['result_sha256'] == sha(path) and publication['operator_process_id'] == pid,
                'foreign attribution publication operator')
        executed = raw['observations']['executed']['linked']
        after, peak = publication['linked_after'], publication['operator_peak_rss_bytes']
        require(all(type(after[field]) is int and after[field] >= executed[field]
                    for field in ('user_nanos', 'system_nanos', 'peak_rss_bytes'))
                and type(peak) is int and 0 < raw['observations']['operator_peak_rss_bytes'] <= peak <= 2 << 30
                and after['peak_rss_bytes'] <= max(peak, executed['peak_rss_bytes']),
                'publication tail is not attributed to actual operator')
        require(result['_driver_publication'] == dict(cpu_nanos=after['user_nanos'] + after['system_nanos']
                - executed['user_nanos'] - executed['system_nanos'], peak_rss_bytes=peak, status='within'),
                'publication tail differs from raw actual counters')
    expected = {tuple(wanted[field] for field in ('workload', 'profile', 'id', 'cell'))
                for wanted in adapters.CELLS.values()}
    require(observed == expected and set(corpus['producer_pids']) == pids,
            'missing/foreign original attributed workload cell')


def reference_result(output, item, owner, original):
    from driver import sha
    run = read(item['path'])
    require(run['schema'] == 'gomad.go-reference-run/v1' and run['original_commit'] == original
            and run['owner'] == owner and run['error'] == '' and run['cancelled'] is False
            and run['watchdog_timeout'] is False, 'original reference failed/cancelled')
    raw_path = bounded_path(output, run['raw_result']['path'])
    require(sha(raw_path) == run['raw_result']['sha256']
            and raw_path.read_bytes() == (Path(item['path']).parent / 'invoke.stdout').read_bytes(),
            'original raw result differs from actual child stdout')
    raw = read(raw_path)
    result = raw['result']
    for stream, payload in dict(stdout=result['Stdout'], stderr=result['Stderr'], wasi=result['Transcript'],
            choice=result['ChoiceTrace']['Bytes'], network=result['NetworkTranscript'], terminal=raw['terminal']).items():
        value = base64.b64decode(payload or '', validate=True)
        require(bounded_path(output, run['streams'][stream]['path']).read_bytes() == value,
                'original reference stream differs from actual raw Result')
    fields = dict(termination='Termination', exit_code='ExitCode', reaped='Reaped', cancelled='Cancelled',
                  watchdog_timeout='WatchdogTimeout', calls='Calls',
                  time_advances='TimeAdvances', environment_identity='EnvironmentIdentity',
                  fuel_remaining='FuelRemaining', authorization_to_terminal_nanos='AuthorizationToTerminalNanos')
    require(run['transcript_sha256'] == bytes(result['TranscriptSHA256']).hex(), 'original WASI digest changed')
    require(all(run[field] == result[key] for field, key in fields.items())
            and run['select_poll'] == result['ChoiceTrace']['Summary']['SelectPoll']
            and raw['error'] == '', 'original reference metadata differs from raw Result')
    return run


def reference_helper_custody(config, entry):
    import json
    import live
    from driver import sha
    receipt = read(entry['helper_receipt'])
    binary = receipt['binary']
    artifact = receipt['artifact_executable']
    require(receipt['schema'] == 'gomad.reference-helper-build/v1'
            and binary['path'] == entry['helper'] and binary['sha256'] == sha(entry['helper'])
            and receipt['copy'] == dict(method='copyfile', source=artifact, destination=binary)
            and artifact['path'] == receipt['cargo_artifact']['executable']
            and {key: value for key, value in artifact.items() if key != 'path'}
                == {key: value for key, value in binary.items() if key != 'path'},
            'reference helper is not the actual copied Cargo artifact')
    for binding in (artifact, receipt['cargo_stdout'], receipt['command_file'], receipt['process_file']):
        path = binding['path']
        live.sealed_input(config, path)
        require(sha(path) == binding['sha256'] and Path(path).stat().st_size == binding['bytes'],
                'changed reference helper build custody')
    source = receipt['source']
    require(source['source_root'] == config['repo'] and source['source_commit'] == config['head'],
            'reference bridge helper was not built from frozen candidate')
    package = Path(config['repo']) / 'tools/gomad/rust/wasmhost'
    expected = {str(path) for path in package.joinpath('src').rglob('*.rs')}
    expected.update(str(path) for path in (package / 'build.rs', package / 'Cargo.toml', package.parent / 'Cargo.toml',
                                         package.parent / 'Cargo.lock', package.parent / 'rust-toolchain.toml'))
    require(set(source['files']) == expected, 'reference helper source closure is incomplete')
    for path, identity in source['files'].items():
        live.sealed_input(config, path)
        raw = Path(path).read_bytes()
        blob = b'blob ' + str(len(raw)).encode() + b'\0' + raw
        require(identity['relative'] == str(Path(path).relative_to(config['repo']))
                and len(raw) == identity['bytes'] and sha(path) == identity['sha256']
                and hashlib.sha1(blob, usedforsecurity=False).hexdigest() == identity['git_blob'],
                'reference helper source differs from frozen Git bytes')
    command = read(receipt['command_file']['path'])
    process = read(receipt['process_file']['path'])
    cargo = command['argv'][0]
    rustc = command['environment']['RUSTC']
    argv = [cargo, 'build', '--offline', '--locked', '--manifest-path', str(package.parent / 'Cargo.toml'),
            '-p', 'gomad-wasmhost', '--features', 'reference-helper', '--bin', 'gomad-wasmhost',
            '--message-format=json-render-diagnostics']
    require(command == receipt['command'] and cargo in receipt['toolchain'] and rustc in receipt['toolchain'],
            'reference helper build toolchain binding changed')
    live.completed_command(command, process, argv, config['repo'])
    for path, identity in receipt['toolchain'].items():
        live.sealed_input(config, path)
        require(sha(path) == identity['sha256'] and Path(path).stat().st_size == identity['bytes'],
                'reference helper compiler changed')
    artifacts = [value for line in Path(receipt['cargo_stdout']['path']).read_bytes().splitlines()
                 if (value := json.loads(line)).get('reason') == 'compiler-artifact'
                 and value.get('target', {}).get('name') == 'gomad-wasmhost' and value.get('executable')]
    require(artifacts == [receipt['cargo_artifact']] and artifacts[0]['features'] == ['reference-helper']
            and artifacts[0]['target']['required-features'] == ['reference-helper'],
            'reference helper has foreign Cargo feature/target output')


def original_source(config, root, entry, label, execute=invoke):
    import live
    from driver import sha
    live.sealed_input(config, entry['git'])
    derived = Path(entry['derived'])
    source_path = derived / 'source.json'
    live.sealed_input(config, str(source_path))
    source = read(source_path)
    original = '1d695061b6382009d0659238c652a5a765b1d5f5'
    require(source['schema'] == 'gomad.derived-go-reference/v1' and source['original_commit'] == original,
            'foreign original reference source')
    import tarfile
    raw_tree, _ = execute(config, root, label,
                         [entry['git'], '-C', config['repo'], 'ls-tree', '-r', '-z', original])
    tree = {}
    for member in raw_tree.split(b'\0'):
        if not member:
            continue
        header, name = member.split(b'\t', 1)
        mode, kind, identity = header.decode().split(' ')
        if kind == 'commit' and mode == '160000':
            # Git archives retain an empty submodule directory, without its external tree.
            continue
        require(kind == 'blob' and mode in ('100644', '100755', '120000'), 'unsupported immutable original tree entry')
        tree[name.decode()] = (mode, identity)
    archive_path = derived / 'original.tar'
    live.sealed_input(config, str(archive_path))
    original_files = {}
    with tarfile.open(archive_path) as archive:
        require(archive.pax_headers.get('comment') == original, 'archive does not identify immutable original commit')
        for member in archive:
            if member.isdir():
                continue
            require(not Path(member.name).is_absolute() and '..' not in Path(member.name).parts,
                    'foreign original archive member')
            require(member.name in tree, 'archive contains foreign original source')
            mode, identity = tree[member.name]
            if member.issym():
                require(mode == '120000', 'original archive symlink mode differs')
                body = member.linkname.encode()
                original_files[member.name] = dict(kind='symlink', target=member.linkname)
            else:
                require(member.isfile(), 'unsupported original archive member')
                body = archive.extractfile(member).read()
                require(mode in ('100644', '100755') and bool(member.mode & 0o111) == (mode == '100755'),
                        'original archive file mode differs')
                original_files[member.name] = dict(kind='file', bytes=len(body), sha256=hashlib.sha256(body).hexdigest())
            blob = b'blob ' + str(len(body)).encode() + b'\0' + body
            require(hashlib.sha1(blob, usedforsecurity=False).hexdigest() == identity, 'archive differs from immutable original Git object')
    require(set(original_files) == set(tree), 'archive omits immutable original source')
    require(original_files == source['original_files'], 'original source table differs from immutable archive')
    require(set(source['glue']) == {'tools/gomad/cmd/gomad-reference/main.go'}, 'reference source has undeclared glue')
    actual_names = {str(path.relative_to(derived / 'source')) for path in (derived / 'source').rglob('*')
                    if path.is_file() or path.is_symlink()}
    require(actual_names == set(original_files) | set(source['glue']), 'undeclared original reference source addition')
    template = str(Path(__file__).parent / 'reference/main.go.txt')
    live.sealed_input(config, template)
    for name, digest in source['glue'].items():
        require(sha(derived / 'source' / name) == digest == sha(template), 'changed invocation-only reference glue')
    for name, item in source['original_files'].items():
        path = derived / 'source' / name
        require(path.is_relative_to(derived / 'source') and '..' not in Path(name).parts,
                'foreign original source member')
        if item['kind'] == 'symlink':
            import os
            require(path.is_symlink() and os.readlink(path) == item['target'], 'changed original source symlink')
        else:
            require(path.is_file() and not path.is_symlink() and path.stat().st_size == item['bytes']
                    and sha(path) == item['sha256'], 'changed immutable original source')
    return derived, source


def reference_runtime(config, root, work, native_root, runtime_spec):
    import json
    import live
    from driver import sha
    entry = config['final_profiles']['runtime_reference']
    for field in ('python', 'script', 'git', 'files', 'build_receipt', 'binary', 'helper', 'helper_receipt'):
        live.sealed_input(config, entry[field])
    derived, source = original_source(config, root, entry, 'final-profile-original-tree')
    source_path = derived / 'source.json'
    original = source['original_commit']
    build = read(entry['build_receipt'])
    require(build['schema'] == 'gomad.reference-build/v1' and build['original_commit'] == original
            and build['source_manifest']['sha256'] == sha(source_path) and build['glue'] == source['glue']
            and build['binary']['path'] == entry['binary'] and build['binary']['sha256'] == sha(entry['binary']),
            'foreign reference binary build custody')
    live.sealed_input(config, build['command']['path'])
    build_command = read(build['command']['path'])
    require(sha(build['command']['path']) == build['command']['sha256']
            and build_command['exit'] == 0 and build_command['observation'] == 'completed'
            and build_command['cwd'] == str(derived / 'source/tools/gomad')
            and build_command['argv'] == [build['stock_go']['path'], 'build', '-p=1', '-mod=readonly',
                '-tags=test_dep', '-trimpath', '-buildvcs=false', '-o', entry['binary'], './cmd/gomad-reference'],
            'reference build did not execute original invocation glue')
    live.sealed_input(config, build['stock_go']['path'])
    require(sha(build['stock_go']['path']) == build['stock_go']['sha256'], 'changed original reference compiler')
    reference_helper_custody(config, entry)
    request_path = native_root / 'request.json'
    native = read(request_path)
    provenance = Path(native['guest']['module']).parent / 'module-provenance.json'
    output = work / 'runtime-reference'
    argv = [entry['python'], entry['script'], 'run', '--derived', str(derived),
            '--native-request', str(request_path), '--request-sha256', sha(request_path),
            '--provenance', str(provenance), '--provenance-sha256', sha(provenance),
            '--files', entry['files'], '--build-receipt', entry['build_receipt'],
            '--reference-binary', entry['binary'], '--helper', entry['helper'],
            '--helper-receipt', entry['helper_receipt'], '--output', str(output)]
    invoke(config, root, 'final-profile-runtime-reference', argv)
    manifest = read(output / 'manifest.json')
    require(manifest['schema'] == 'gomad.runtime-reference-pair/v1' and manifest['original_commit'] == original
            and manifest['native_request']['path'] == str(request_path)
            and manifest['native_request']['sha256'] == sha(request_path)
            and manifest['provenance']['sha256'] == sha(provenance)
            and manifest['module']['sha256'] == native['guest']['module_sha256'], 'foreign original reference custody')
    actual = []
    for owner in ('go', 'helper'):
        directory = output / owner
        item = manifest['runs'][owner]
        require(bounded_path(output, item['path']) == directory / 'run.json'
                and item['sha256'] == sha(item['path']), 'changed original reference run')
        run = reference_result(output, item, owner, original)
        invocation_path = directory / 'invocation.json'
        invocation = read(invocation_path)
        request = invocation['request']
        require(invocation['remote_model'] is (owner == 'go')
                and request['MemoryBytes'] == runtime_spec['memory_bytes'] and request['Fuel'] == runtime_spec['fuel']
                and request['Timeout'] == runtime_spec['wall_nanos'] and request['Runtime']['Seed'] == 7
                and request['Runtime']['Diagnostics'] is False and request['Runtime']['Choice']['Limit'] == 64 << 20
                and request['Config']['Args'] == native['guest']['model']['arguments']
                and request['Config']['Environment'] == native['guest']['model']['environment']
                and request['ModuleSHA256'] == native['guest']['module_sha256']
                and request['HelperPath'] == entry['helper'] and request['HelperSHA256'] == sha(entry['helper']),
                'original reference executed a different runtime request')
        model = native['guest']['model']
        namespace = native['guest']['namespace']
        config_model, limits = model['configuration'], native['guest']['limits']
        fs = namespace['limits']
        expected_config = dict(Profile='gomad.wasi-cooperative/v1', Args=model['arguments'],
            Environment=model['environment'], WorkingDirectory=bytes(namespace['working_directory']).decode(),
            WritableDirectories=[bytes(path).decode() for path in namespace['writable_directories']],
            Stdin=base64.b64encode(bytes(namespace['stdin'])).decode(), EntropyKey=config_model['entropy_key'],
            Clock=dict(EpochNanos=config_model['epoch_nanos'], ReadStepNanos=config_model['read_step_nanos']),
            Limits=dict(OutputBytes=limits['output_bytes'], TranscriptBytes=config_model['limits']['transcript_bytes'],
                Calls=config_model['limits']['calls'], PendingEvents=config_model['limits']['pending_events'],
                Descriptors=fs['descriptors'], Files=fs['files'], FilesystemBytes=fs['filesystem_bytes']))
        require(request['Config'] == expected_config
                and request['Runtime']['Choice']['ExecutionIdentity'] == read(provenance)['choice_identity']
                and request['Runtime']['Choice']['Mode'] == 1
                and request['DigestTranscript'] == config_model.get('digest_only', False)
                and request['ModulePath'] == manifest['module']['pinned_path']
                and sha(request['ModulePath']) == native['guest']['module_sha256'],
                'original reference modeled request differs from actual native input')
        require(request['Config']['Clock']['ReadStepNanos'] == 0
                and request['Config']['Limits']['Calls'] == 1 << 20
                and request['Config']['Limits']['TranscriptBytes'] == 64 << 20,
                'original reference runtime bounds changed')
        live.completed_command(read(directory / 'invoke.command.json'), read(directory / 'invoke.process.json'),
                               [entry['binary'], str(invocation_path)], str(derived / 'source/tools/gomad'))
        actual.append(run)
    runtime_pair(work, runtime_spec, *actual)
    return actual


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
    for field in ('stock_go', 'terminal_module', 'terminal_module_receipt', 'terminal_helper_receipt', 'terminal_diagnostic_helper_receipt'):
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
    if not entry.get('runtime_reference'):
        raise Unexecuted('runtime-cost original remote-model/helper-model comparison requires the immutable original reference producer')
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
    references = reference_runtime(config, root, work, Path(runs[0]['native_root']), runtime_spec)
    same_evidence(work, references[0], observed[0][1], qualified=False)
    terminal_argv = [measure, 'terminal-exit', *common, '--terminal-exit-root', str(terminal_root),
        '--terminal-exit-module', entry['terminal_module'], '--terminal-exit-module-receipt',
        entry['terminal_module_receipt'], '--terminal-exit-helper-receipt', entry['terminal_helper_receipt'],
        '--terminal-exit-diagnostic-helper', diagnostic, '--terminal-exit-diagnostic-helper-receipt',
        entry['terminal_diagnostic_helper_receipt']]
    raw, process = invoke(config, root, 'final-profile-terminal', terminal_argv)
    native_terminal(config, root, work, entry, bounded_path(terminal_root, json.loads(raw)['result']), ordinary, diagnostic)
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
                                [binary, '--linked-run', pair[role + '_input'], str(output)])
            report, run = native_run(output)
            require(read(output / 'request.json') == bound_native_request(config, pair[role + '_input'], recipe),
                    'retained native request differs from actual original input')
            require(report['fresh_store_id'] == process['pid']
                    and report['request_sha256'] == sha(pair[role + '_input'])
                    and report['host_sha256'] == sha(binary)
                    and report['linked_implementation_sha256'] == config['_final_profile_role_identities'][binary]['linked_implementation_sha256']
                    and report['host_implementation_sha256'] == config['_final_profile_role_identities'][binary]['host_implementation_sha256']
                    and report['features'] == ([] if role == 'release' else ['diagnostic-profile']),
                    'foreign qualified native execution')
            if role == 'diagnostic':
                attribution(read(output / 'profile.json'))
            actual.append(run)
        same_evidence(work, *actual, qualified=True)
    linked_attribution(config, root)
    guard(config)
    return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                identity=config['identity'], subgates=[dict(id=name, status='PASS', executed=True, exit=0) for name in OWNED])


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
