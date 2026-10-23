#!/usr/bin/env python3
"""Fresh fn233 receiving proofs; separate from the immutable 81-case corpus."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import shlex
import subprocess
import sys
import time

sys.dont_write_bytecode = True
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader

HERE = Path(__file__).resolve().parent
loader = SourceFileLoader('interception_receiving_shim', str(HERE / 'interception-shim-check.py'))
shim = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(shim)
check = shim.load('interception_receiving_origin', HERE / 'interception-check.py')
equal = shim.equal
LIMITS = dict(evidence_bytes=159383552, fuel=1 << 50, memory_bytes=4 << 30,
              output_bytes=8 << 20, wall_millis=300000)
FRONTEND = './tests/gomadfunctional'
WORKER = './common/tasktoken'
SELECTOR = '^TestFrontendSystemInfo$'
PARALLEL_SELECTOR = '^(TestFrontendSystemInfo|TestWorkerTaskID)$'
EXACT_REPLAY = b'gomad: reproduced=true diagnostic=false result=success choice-replay=exact\n'
CASES = ('auto-first', 'auto-reuse', 'manual', 'parallel')


def sha(path):
    return shim.file_sha(path)


def read(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > 32 << 20:
        raise ValueError('missing, unsafe or oversized JSON: ' + str(path))
    return json.loads(path.read_bytes())


def write(path, value):
    with path.open('x', encoding='utf-8') as output:
        json.dump(value, output, sort_keys=True, separators=(',', ':'))
        output.write('\n')
    path.chmod(0o600)


def encoded(argv):
    return [os.fsencode(value).hex() for value in argv]


def decoded(argv):
    return [os.fsdecode(bytes.fromhex(value)) for value in argv]


def pin_inputs(path):
    if not path.is_absolute():
        raise ValueError('absolute receiving inputs path required')
    inputs = read(path)
    equal(inputs['schema'], 'gomad.interception-receive-inputs/v1', 'receiving.inputs_schema')
    for key in ('binary', 'fixture_binary', 'compiler', 'checkout', 'cache', 'installation'):
        if not Path(inputs[key]).is_absolute():
            raise ValueError('absolute ' + key + ' required')
    pins = inputs['files']
    required = [inputs[key] for key in ('binary', 'fixture_binary', 'compiler', 'installation')]
    required += [str(Path(inputs['checkout']) / name) for name in ('go.mod', 'gomad.toml')]
    if inputs['binary'] == inputs['fixture_binary'] or pins[inputs['binary']] == pins[inputs['fixture_binary']]:
        raise ValueError('ordinary and fixture executable roles must be distinct')
    selections = inputs['accepted_selections']
    equal([(value['package'], value['selector']) for value in selections],
          [(FRONTEND, SELECTOR), (WORKER, '^TestWorkerTaskID$')], 'receiving.exact_selections')
    required += [value['receipt'] for value in selections]
    if not all(name in pins for name in required):
        raise ValueError('missing static input pins')
    for name, digest in pins.items():
        source = Path(name)
        if not source.is_absolute() or source.is_symlink() or not source.is_file():
            raise ValueError('unsafe pinned input: ' + name)
        equal(sha(source), digest, 'receiving.current_input.' + name)
    for selection in selections:
        equal(pins[selection['receipt']], selection['receipt_sha256'], 'receiving.selection_custody')
    # Overrides must not redirect the pinned compiler or the common preparation cache.
    forbidden = {'PATH', 'GOROOT', 'GOOS', 'GOARCH', 'GOFLAGS', 'CGO_ENABLED', 'XDG_CACHE_HOME',
                 'GOENV', 'GOTOOLCHAIN', 'GOWORK'}
    if any(key in forbidden or key.startswith('GOMAD_') for key in inputs['environment']):
        raise ValueError('uncontrolled compiler/cache environment override')
    return inputs


def environment(inputs):
    result = {key: value for key, value in os.environb.items()
              if not key.startswith(b'GOMAD_') and key not in
              (b'GOOS', b'GOARCH', b'GOROOT', b'GOFLAGS', b'CGO_ENABLED')}
    result.update({b'PATH': os.fsencode(str(Path(inputs['compiler']).parent)) + b':' + result.get(b'PATH', b''),
                   b'GOENV': b'off', b'GOTOOLCHAIN': b'local', b'GOWORK': b'off',
                   b'XDG_CACHE_HOME': os.fsencode(inputs['cache'])})
    result.update({os.fsencode(key): os.fsencode(value) for key, value in inputs['environment'].items()})
    return result


def cache_state(inputs):
    return {str(path): dict(sha256=sha(path), mtime_ns=path.stat().st_mtime_ns)
            for path in sorted(Path(inputs['cache']).rglob('derived-installations/*.json'))}


def command(inputs, case, root):
    binary = inputs['fixture_binary'] if case in ('manual', 'parallel') else inputs['binary']
    if case.startswith('auto-'):
        return [binary, '--seeds', '1', '--artifacts', str(root / 'artifacts'), '--', 'go', 'test',
                '-mod=readonly', '-pgo=off', '-trimpath', '-buildvcs=false', '-tags=test_dep',
                '-count=1', '-v', '-run=' + SELECTOR, FRONTEND]
    argv = [binary, 'fixture-driver-prepare', '--output', str(root / 'invocation'),
            '--installation', inputs['installation']]
    if case == 'parallel':
        argv += ['--package', WORKER]
    return argv + ['--choices', '--seeds', '1', '--execution-timeout', '5m', '--overall-timeout', '10m',
                   '--artifacts', str(root / 'artifacts'), '--working-dir', inputs['checkout'],
                   '--build-tag', 'test_dep', 'go-test', FRONTEND, '--',
                   '-test.run=' + (PARALLEL_SELECTOR if case == 'parallel' else SELECTOR), '-test.v=true']


def observe(pid, compiler=None, executable_cache=None):
    children = set()
    for task in Path('/proc', str(pid), 'task').glob('*/children'):
        try:
            children.update(int(value) for value in task.read_text().split())
        except FileNotFoundError:
            continue
    launches, drivers = [], []
    for child in sorted(children):
        try:
            argv = Path('/proc', str(child), 'cmdline').read_bytes().rstrip(b'\0').split(b'\0')
            if compiler is not None:
                executable = Path('/proc', str(child), 'exe')
                if os.readlink(executable) == compiler:
                    if child not in executable_cache:
                        executable_cache[child] = sha(executable)
                    drivers.append(dict(pid=child, argv=[value.hex() for value in argv],
                                        executable=os.fsencode(compiler).hex(), executable_sha256=executable_cache[child]))
            if len(argv) >= 2 and argv[1] == b'run-guest':
                launches.append(dict(pid=child, argv=[value.hex() for value in argv]))
        except FileNotFoundError:
            continue
    return dict(monotonic_ns=time.monotonic_ns(), go_pid=pid, launchers=launches, drivers=drivers)


def run(argv, inputs, root, deadline_ns, observing=False):
    root.mkdir(mode=0o700)
    env = environment(inputs)
    write(root / 'command.json', dict(argv=encoded(argv), cwd=inputs['checkout'],
                                     environment={key.hex(): value.hex() for key, value in sorted(env.items())},
                                     deadline_ns=deadline_ns))
    started = time.monotonic_ns()
    if started >= deadline_ns:
        raise ValueError('receiving deadline expired before spawn; no retry')
    samples = []
    executable_cache = {}
    expired = False
    with (root / 'stdout').open('xb') as stdout, (root / 'stderr').open('xb') as stderr:
        os.chmod(root / 'stdout', 0o600)
        os.chmod(root / 'stderr', 0o600)
        child = subprocess.Popen(argv, cwd=inputs['checkout'], env=env, stdin=subprocess.DEVNULL, stdout=stdout, stderr=stderr,
                                 start_new_session=True)
        write(root / 'started.json', dict(pid=child.pid, monotonic_ns=started, unix_ns=time.time_ns()))
        try:
            while child.poll() is None:
                if observing:
                    samples.append(observe(child.pid, inputs['compiler'] if observing == 'wrapper' else None, executable_cache))
                if time.monotonic_ns() >= deadline_ns:
                    expired = True
                    os.killpg(child.pid, signal.SIGTERM)
                    try:
                        child.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        os.killpg(child.pid, signal.SIGKILL)
                        child.wait()
                    break
                time.sleep(0.01)
        finally:
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGKILL)
                child.wait()
            write(root / 'observation.json', dict(samples=samples))
            write(root / 'result.json', dict(status=child.returncode, timed_out=expired,
                                             monotonic_ns=time.monotonic_ns(),
                                             stdout_sha256=sha(root / 'stdout'), stderr_sha256=sha(root / 'stderr')))
    if expired:
        raise ValueError('receiving deadline expired; no retry')
    return child.returncode


def authenticate_command(root, argv, inputs, deadline_ns, status=0):
    receipt, start, result = (read(root / name) for name in ('command.json', 'started.json', 'result.json'))
    equal(receipt['argv'], encoded(argv), 'receiving.exact_command')
    equal(receipt['cwd'], inputs['checkout'], 'receiving.exact_cwd')
    equal(receipt['deadline_ns'], deadline_ns, 'receiving.shared_deadline')
    env = {bytes.fromhex(key): bytes.fromhex(value) for key, value in receipt['environment'].items()}
    for key, value in environment(inputs).items():
        if key in (b'PATH', b'GOENV', b'GOTOOLCHAIN', b'GOWORK', b'XDG_CACHE_HOME') or os.fsdecode(key) in inputs['environment']:
            equal(env.get(key), value, 'receiving.admitted_environment.' + os.fsdecode(key))
    if any(key.startswith(b'GOMAD_') or key in (b'GOROOT', b'GOOS', b'GOARCH', b'GOFLAGS', b'CGO_ENABLED') for key in env):
        raise ValueError('unbound inherited Gomad environment')
    equal((result['status'], result['timed_out']), (status, False), 'receiving.actual_status')
    if type(start['pid']) is not int or start['pid'] <= 0:
        raise ValueError('missing actual process identity')
    if not start['monotonic_ns'] < result['monotonic_ns'] <= deadline_ns:
        raise ValueError('command outside admitted deadline')
    for stream in ('stdout', 'stderr'):
        equal(sha(root / stream), result[stream + '_sha256'], 'receiving.raw_stream_digest')
    return start['pid']


def authenticate_wrapper(samples, origins, wrapper_pid, inputs):
    observed = {}
    for sample in samples:
        equal(sample['go_pid'], wrapper_pid, 'receiving.observed_actual_wrapper')
        for driver in sample['drivers']:
            equal(driver['executable'], os.fsencode(inputs['compiler']).hex(), 'receiving.observed_compiler_path')
            equal(driver['executable_sha256'], inputs['files'][inputs['compiler']], 'receiving.observed_compiler_bytes')
            observed[driver['pid']] = driver['argv']
    equal(len(observed), 1, 'receiving.one_actual_go_child')
    for origin in origins:
        parent = origin['launch']['parent_process_id']
        if parent == wrapper_pid or parent not in observed:
            raise ValueError('launcher parent is not the observed actual Go child')
        equal(observed[parent], origin['invocation']['driver_argv'], 'receiving.observed_exact_go_driver')


def authenticate_overlap(samples, origins, go_pid):
    known = {row['launch']['process_id']: row['launch']['argv'] for row in origins}
    equal(len(known), 2, 'receiving.distinct_launcher_pids')
    overlapped = False
    previous = 0
    for sample in samples:
        equal(sample['go_pid'], go_pid, 'receiving.observed_go_pid')
        if sample['monotonic_ns'] < previous:
            raise ValueError('nonmonotonic process observation')
        previous = sample['monotonic_ns']
        seen = set()
        for launch in sample['launchers']:
            pid = launch['pid']
            if pid not in known or pid in seen:
                raise ValueError('unbound or duplicate observed launcher')
            equal(launch['argv'], known[pid], 'receiving.observed_actual_argv')
            seen.add(pid)
        overlapped |= len(seen) == 2
    if not overlapped:
        raise ValueError('no observed concurrent distinct launchers')


def authenticate_output(root, packages):
    expected = b''
    for package, name, stdout, stderr in packages:
        equal(stdout, ('=== RUN   ' + name + '\n--- PASS: ' + name + ' (0.00s)\nPASS\n').encode(),
              'receiving.exact_named_guest_stdout')
        first, rest = stdout.split(b'\n', 1)
        outer = (root / 'stdout').read_bytes()
        summaries = re.findall(rb'^ok  \t' + re.escape(package.encode()) + rb'\t[0-9.]+s\n', outer, re.M)
        equal(len(summaries), 1, 'receiving.one_go_package_summary')
        expected += first + b'\n' + stderr + rest + summaries[0]
    equal((root / 'stdout').read_bytes(), expected, 'receiving.exact_forwarded_stdout')
    equal((root / 'stderr').read_bytes(), b'', 'receiving.no_extra_driver_stderr')


def authenticate_execution(inputs, case, root, binary, go_pid=None):
    rows = shim.authenticated_plans(root / 'artifacts')
    complete = [row for row in rows if row['complete']]
    count = 2 if case == 'parallel' else 1
    equal((len(rows), len(complete)), (2 * count, count), 'receiving.exact_campaign_roles')
    origins, artifacts, packages = [], [], []
    for retained in complete:
        origin = retained['origin']
        inv, launch = origin['invocation'], origin['launch']
        driver = decoded(inv['driver_argv'])
        equal(driver[0], inputs['compiler'], 'receiving.pinned_driver_compiler')
        equal(driver[1], 'test', 'receiving.real_go_subcommand')
        effective = shim.driver_flags(driver[2:])
        equal(shlex.split(effective.get('exec', '')), [binary, 'run-guest'], 'receiving.actual_exec_launcher')
        if case in ('manual', 'parallel'):
            tail = (['-p=2', FRONTEND, WORKER] if case == 'parallel' else [FRONTEND])
            tail += ['-test.run=' + (PARALLEL_SELECTOR if case == 'parallel' else SELECTOR), '-test.v=true']
            equal(driver[-len(tail):], tail, 'receiving.exact_manual_driver_tail')
        equal(inv['compiler_sha256'], inputs['files'][inputs['compiler']], 'receiving.compiler_hash')
        equal(decoded(launch['argv'])[:2], [binary, 'run-guest'], 'receiving.actual_launcher')
        equal(shim.driver_flags(driver[2:]).get('count'), '1', 'receiving.no_test_cache')
        if go_pid is not None:
            equal(launch['parent_process_id'], go_pid, 'receiving.actual_go_parent')
        plan = retained['validated']['plan']
        equal((plan['overall_millis'], plan['overall_nanos'], plan['parallel'], plan['selection_count']),
              (600000, 600000000000, 1, 1), 'receiving.original_overall_bounds')
        equal(len(retained['validated']['executions']), 1, 'receiving.one_seed')
        artifact = retained['root'] / retained['validated']['executions'][0]['artifact']
        record = check.campaign.authenticate_artifact(artifact)
        equal(check.authenticate_origin(artifact, record, plan['origin']), origin, 'receiving.execution_origin')
        equal(record['limits'], LIMITS, 'receiving.original_limits')
        equal(record['seed'], 1, 'receiving.original_seed')
        equal(record['outcome'], dict(**{'class': 'success'}, exit_code=0, reason='', termination='exit'),
              'receiving.actual_success')
        argv = decoded(launch['argv'])
        equal(record['arguments'], ['gomad-target', *argv[3:]], 'receiving.guest_arguments')
        if Path(argv[2]).exists():
            raise ValueError('original Go temporary module still present')
        provenance = check.campaign.canonical_file(artifact / 'target/provenance.json')
        captured = provenance['inputs']
        equal(captured['kind'], 'go-test', 'receiving.real_go_test')
        equal(captured['compiler_sha256'], inputs['files'][inputs['compiler']], 'receiving.module_compiler')
        equal(captured['go_version'], 'go1.27.1', 'receiving.pinned_go_version')
        package = captured['source']
        names = {FRONTEND: ('go.temporal.io/server/tests/gomadfunctional', 'TestFrontendSystemInfo'),
                 WORKER: ('go.temporal.io/server/common/tasktoken', 'TestWorkerTaskID')}
        if case.startswith('auto-'):
            names = {value[0]: value for value in names.values()}
        selected = names.get(package)
        name = selected[1] if selected else None
        if name is None or (case != 'parallel' and name != 'TestFrontendSystemInfo'):
            raise ValueError('unexpected executed Go package')
        equal(captured['go_build_info']['path'], selected[0] + '.test', 'receiving.module_build_package')
        equal(captured['go_build_info']['compiler_version'], 'go1.27.1', 'receiving.module_build_compiler')
        for key, value in {'CGO_ENABLED': '0', 'GOARCH': 'wasm', 'GOOS': 'wasip1', '-trimpath': 'true'}.items():
            equal(captured['go_build_info']['settings'][key], value, 'receiving.actual_module_setting.' + key)
        selector = PARALLEL_SELECTOR if case == 'parallel' else SELECTOR
        if '-test.run=' + selector not in argv[3:]:
            raise ValueError('wrong actual guest selector')
        for kind in ('compiler', 'sources'):
            if not captured[kind]:
                raise ValueError('missing original ' + kind + ' inventory')
            for path, digest in captured[kind].items():
                equal(sha(path), digest, 'receiving.current_' + kind + '_bytes')
        if case in ('manual', 'parallel'):
            equal(inv['installation_sha256'], inputs['files'][inputs['installation']], 'receiving.fixture_installation')
            installed = read(Path(inputs['installation']))
            equal(installed['admission']['host_executable_sha256'], inputs['files'][binary], 'receiving.installation_executable')
            recipes = read(root / 'invocation/driver-recipe.json')
            recipes = recipes['packages'] if 'packages' in recipes else [recipes]
            matches = [value for value in recipes if value['options']['source'] == package]
            equal(len(matches), 1, 'receiving.exact_package_recipe')
            recipe = matches[0]
            equal(recipe['installed'], installed, 'receiving.recipe_installation')
            for kind in ('compiler', 'sources'):
                equal(recipe[kind], captured[kind], 'receiving.original_recipe_' + kind)
                equal(recipe['inputs'][kind], captured[kind], 'receiving.original_input_' + kind)
        packages.append((selected[0], name, (artifact / 'output/stdout').read_bytes(),
                         (artifact / 'output/stderr').read_bytes() if (artifact / 'output/stderr').exists() else b''))
        origins.append(origin)
        artifacts.append(str(artifact))
    equal(len({value['invocation']['wrapper_run_id'] for value in origins}), 1, 'receiving.shared_wrapper')
    equal(len({value['invocation']['invocation_id'] for value in origins}), 1, 'receiving.shared_driver')
    equal(len({value['launch']['invocation_id'] for value in origins}), count, 'receiving.distinct_launches')
    if count == 2:
        equal(len({tuple(value['launch']['argv']) for value in origins}), 2, 'receiving.distinct_guest_argv')
    packages.sort(key=lambda row: row[0] != 'go.temporal.io/server/tests/gomadfunctional')
    authenticate_output(root / 'run', packages)
    ordered = sorted(zip(origins, artifacts), key=lambda row: decoded(row[0]['launch']['argv'])[2])
    return [row[0] for row in ordered], [row[1] for row in ordered]


def authenticate_cache_pair(before, after, first_after=None):
    if first_after is None:
        if before or not after:
            raise ValueError('auto-first requires an empty derived cache and a real new installation')
    else:
        equal(before, first_after, 'receiving.same_first_cache')
        equal(after, before, 'receiving.reuse_no_manifest_rewrite')


def collect(inputs_path, case, root, readers=True):
    inputs = pin_inputs(inputs_path)
    admission = read(root / 'admission.json')
    equal(admission['inputs_sha256'], sha(inputs_path), 'receiving.input_custody')
    equal(admission['case'], case, 'receiving.case')
    equal(admission['root'], str(root), 'receiving.caller_bound_root')
    deadline = admission['deadline_ns']
    if not 0 < deadline - admission['started_ns'] <= 600_000_000_000:
        raise ValueError('receiving budget exceeds original 600 seconds')
    authenticate_roles(inputs, root, deadline)
    equal(read(root / 'schema-before.json'), shim.schema_inventory(Path(inputs['checkout'])),
          'receiving.current_schema_matches_admission')
    argv = command(inputs, case, root)
    binary = inputs['binary'] if case.startswith('auto-') else inputs['fixture_binary']
    if case.startswith('auto-'):
        pid = authenticate_command(root / 'run', argv, inputs, deadline)
        origins, artifacts = authenticate_execution(inputs, case, root, binary)
        observation = read(root / 'run/observation.json')['samples']
        started, ended = read(root / 'run/started.json'), read(root / 'run/result.json')
        if any(not started['monotonic_ns'] <= sample['monotonic_ns'] <= ended['monotonic_ns'] for sample in observation):
            raise ValueError('wrapper observation outside actual process interval')
        authenticate_wrapper(observation, origins, pid, inputs)
        equal(decoded(origins[0]['invocation']['original_argv']), argv[argv.index('--') + 1:],
              'receiving.exact_original_wrapper_argv')
        current = shim.authenticate_cache(Path(inputs['cache']), Path(binary), Path(inputs['compiler']), Path(inputs['checkout']))
        after = read(root / 'cache-after.json')
        if case == 'auto-reuse':
            equal(cache_state(inputs), after, 'receiving.current_reuse_cache')
        equal({key: row['sha256'] for key, row in current.items()},
              {key: row['sha256'] for key, row in after.items()}, 'receiving.cache_payloads')
        for artifact in artifacts:
            provenance = check.campaign.canonical_file(Path(artifact) / 'target/provenance.json')
            bindings = provenance['inputs']['shared_driver_installations']
            if len(bindings) != 1 or bindings[0][1] not in {value['sha256'] for value in current.values()}:
                raise ValueError('actual auto execution is not bound to the authenticated derived installation')
            shim.authenticate_installations(provenance, origins[0], [[provenance['inputs']['source'], bindings[0][1]]])
        for value in current.values():
            equal(value['schema_inventory'], shim.schema_inventory(Path(inputs['checkout'])), 'receiving.actual_schema')
        baseline = None
        if case == 'auto-reuse':
            first = Path(admission['baseline'])
            equal(first, root.parent / 'auto-first', 'receiving.same_root_baseline')
            collect(inputs_path, 'auto-first', first)
            baseline = read(first / 'cache-after.json')
        authenticate_cache_pair(read(root / 'cache-before.json'), after, baseline)
    else:
        authenticate_command(root / 'prepare', argv, inputs, deadline)
        exported = read(root / 'prepare/stdout')
        equal(exported['guest_executions'], 0, 'receiving.zero_guest_export')
        invocation_path = root / 'invocation/invocation.json'
        equal(exported['invocation'], str(invocation_path), 'receiving.export_path')
        equal(exported['invocation_sha256'], sha(invocation_path), 'receiving.export_hash')
        invocation = check.campaign.canonical_file(invocation_path)
        equal(decoded(invocation['driver_argv']), exported['driver_argv'], 'receiving.export_argv')
        equal(invocation['original_argv'], invocation['driver_argv'], 'receiving.manual_fixture_driver')
        equal(invocation['recipe_sha256'], sha(root / 'invocation/driver-recipe.json'), 'receiving.recipe_bytes')
        run_argv = [binary, 'fixture-driver-run', '--invocation', str(invocation_path), '--sha256', sha(invocation_path)]
        pid = authenticate_command(root / 'run', run_argv, inputs, deadline)
        registration = check.campaign.canonical_file(root / 'invocation/driver-registration.json')
        equal(registration['schema'], 'gomad.driver-registration/v1', 'receiving.registration_schema')
        equal(registration['process_id'], pid, 'receiving.registered_go_pid')
        equal(registration['compiler_sha256'], inputs['files'][inputs['compiler']], 'receiving.registered_compiler')
        equal(registration['driver_argv_sha256'], check.parity.digest(check.parity.canonical(invocation['driver_argv'])),
              'receiving.registered_exact_argv')
        equal(registration['invocation_sha256'], sha(invocation_path), 'receiving.registration_invocation')
        origins, artifacts = authenticate_execution(inputs, case, root, binary, pid)
        for origin in origins:
            equal(origin['invocation'], invocation, 'receiving.manual_registered_origin')
        if case == 'parallel':
            observation = read(root / 'run/observation.json')['samples']
            started, ended = read(root / 'run/started.json'), read(root / 'run/result.json')
            if any(not started['monotonic_ns'] <= sample['monotonic_ns'] <= ended['monotonic_ns'] for sample in observation):
                raise ValueError('parallel observation outside actual process interval')
            authenticate_overlap(observation, origins, pid)
    if readers:
        authenticate_readers(inputs, root, artifacts, binary, deadline)
    return dict(schema='gomad.interception-receiving-proof/v1', case=case, accepted=True,
                binary_sha256=inputs['files'][binary], inputs_sha256=sha(inputs_path), artifacts=artifacts,
                origins=origins, ordinary_auto_derived=case.startswith('auto-'),
                fixture_manual=case in ('manual', 'parallel'), final_qualification=False)


def produce(inputs_path, case, root):
    inputs = pin_inputs(inputs_path)
    root.mkdir(mode=0o700)
    start = time.monotonic_ns()
    admission = dict(inputs_sha256=sha(inputs_path), case=case, root=str(root), started_ns=start,
                     deadline_ns=start + 600_000_000_000)
    if case == 'auto-reuse':
        admission['baseline'] = str(root.parent / 'auto-first')
        collect(inputs_path, 'auto-first', Path(admission['baseline']))
    write(root / 'admission.json', admission)
    write(root / 'schema-before.json', shim.schema_inventory(Path(inputs['checkout'])))
    for name, argv, _ in role_commands(inputs):
        run(argv, inputs, root / name, admission['deadline_ns'])
    authenticate_roles(inputs, root, admission['deadline_ns'])
    if case == 'auto-first' and cache_state(inputs):
        raise ValueError('auto-first requires a fresh owned cache; no cleanup is performed')
    if case.startswith('auto-'):
        write(root / 'cache-before.json', cache_state(inputs))
        run(command(inputs, case, root), inputs, root / 'run', admission['deadline_ns'], 'wrapper')
        write(root / 'cache-after.json', cache_state(inputs))
    else:
        status = run(command(inputs, case, root), inputs, root / 'prepare', admission['deadline_ns'])
        equal(status, 0, 'receiving.actual_prepare_status')
        exported = read(root / 'prepare/stdout')
        invocation = root / 'invocation/invocation.json'
        equal(exported['invocation_sha256'], sha(invocation), 'receiving.export_before_execute')
        run([inputs['fixture_binary'], 'fixture-driver-run', '--invocation', str(invocation),
             '--sha256', sha(invocation)], inputs, root / 'run', admission['deadline_ns'], case == 'parallel')
    provisional = collect(inputs_path, case, root, readers=False)
    produce_readers(inputs, root, provisional['artifacts'],
                    inputs['binary'] if case.startswith('auto-') else inputs['fixture_binary'],
                    admission['deadline_ns'])
    result = collect(inputs_path, case, root)
    write(root / 'proof.json', result)
    return result


def role_commands(inputs):
    return [('ordinary-role', [inputs['binary'], 'fixture-driver-prepare'],
             b'gomad: unsupported command; use gomad -- COMMAND [ARG ...] to run a child\n'),
            ('fixture-role', [inputs['fixture_binary'], 'fixture-driver-prepare'],
             b'gomad: fixture-driver-prepare requires --output DIR --installation FILE [PREPARATION OPTIONS] TARGET\n')]


def authenticate_roles(inputs, root, deadline):
    for name, argv, stderr in role_commands(inputs):
        authenticate_command(root / name, argv, inputs, deadline, 2)
        equal((root / name / 'stdout').read_bytes(), b'', 'receiving.role_probe_stdout')
        equal((root / name / 'stderr').read_bytes(), stderr, 'receiving.actual_binary_role')


def authenticate_inspect(stdout, artifact):
    inspected = json.loads(stdout)
    equal(inspected['schema'], 'gomad.inspect/v9', 'receiving.inspect_schema')
    equal((inspected['kind'], inspected['path']), ('artifact', str(artifact)), 'receiving.inspect_actual_path')
    record = check.campaign.authenticate_artifact(artifact)
    equal(inspected['artifact']['record'], record, 'receiving.inspect_exact_record')
    equal(inspected['artifact']['record_sha256'], sha(artifact / 'record.json'), 'receiving.inspect_record_hash')


def reader_commands(inputs, root, artifacts, binary):
    result = []
    for index, artifact in enumerate(artifacts):
        prefix = 'artifact-' + str(index)
        result += [(prefix + '-inspect', [binary, 'inspect', '--json', artifact], 0),
                   (prefix + '-replay', [binary, 'replay', artifact], 0),
                   (prefix + '-minimize-refusal', [binary, 'minimize', '--json', '--attempt-budget=1',
                    '--artifacts', str(root / (prefix + '-reductions')), artifact], 3)]
    result.append(('qualify-source-refusal', [inputs['binary'], 'qualify', '--spec',
                                             str(root / 'qualification-source-refusal.json')], 3))
    return result


def qualification_spec(root, artifact):
    return dict(Manifest=dict(Provisional=True, IntegratedSourceSHA256='sha256:' + '0' * 64),
                Campaign=dict(Target=dict(Backend='wasm', Kind='exec', Source=artifact)),
                ArtifactRoot=str(root / 'qualification'))


def produce_readers(inputs, root, artifacts, binary, deadline):
    write(root / 'qualification-source-refusal.json', qualification_spec(root, artifacts[0]))
    for name, argv, status in reader_commands(inputs, root, artifacts, binary):
        actual = run(argv, inputs, root / name, deadline)
        equal(actual, status, 'receiving.actual_reader_status.' + name)


def authenticate_readers(inputs, root, artifacts, binary, deadline):
    equal(read(root / 'qualification-source-refusal.json'), qualification_spec(root, artifacts[0]),
          'receiving.actual_qualification_spec')
    for name, argv, status in reader_commands(inputs, root, artifacts, binary):
        authenticate_command(root / name, argv, inputs, deadline, status)
        stdout, stderr = ((root / name / stream).read_bytes() for stream in ('stdout', 'stderr'))
        if name.endswith('-inspect'):
            authenticate_inspect(stdout, Path(argv[-1]))
            equal(stderr, b'', 'receiving.inspect_stderr')
        elif name.endswith('-replay'):
            equal((stdout, stderr), (EXACT_REPLAY, b''), 'receiving.exact_retained_replay')
        elif name.endswith('-minimize-refusal'):
            equal((stdout, stderr), (b'', b'gomad: minimization requires a replayable target failure\n'),
                  'receiving.success_is_not_minimized')
        elif name == 'qualify-source-refusal':
            equal(stdout, b'', 'receiving.qualify_refusal_stdout')
            if b'unsupported WASM source preparation request' not in stderr:
                raise ValueError('qualify did not reject the actual intercepted source request')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inputs', required=True, type=Path)
    parser.add_argument('--case', required=True, choices=CASES)
    parser.add_argument('--root', required=True, type=Path)
    parser.add_argument('--read-only', action='store_true')
    args = parser.parse_args()
    if not args.root.is_absolute():
        parser.error('--root must be absolute')
    result = collect(args.inputs, args.case, args.root) if args.read_only else produce(args.inputs, args.case, args.root)
    print(json.dumps(result, sort_keys=True))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError, TypeError) as error:
        print('interception receiving refused: ' + str(error), file=sys.stderr)
        raise SystemExit(1)
