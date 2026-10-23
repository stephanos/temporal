#!/usr/bin/env python3
"""One granted public-CLI shim case with raw authentication before golden projection.

Synthetic compiler captures prove routing/refusal only. Real guest/cache cases
require explicit reviewed input bindings; no installation or module is fabricated.
"""
import argparse
import base64
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
import re
from pathlib import Path
import signal
import shlex
import subprocess
import sys
import threading
from types import SimpleNamespace

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
SELF = Path(__file__).resolve()
CORPUS = json.loads((HERE / 'interception-shim-corpus.json').read_text())
CASES = {case['name']: case for case in CORPUS['cases']}
RAW_ARGS = [b'', b'space value', b"'quoted'", b'$dollar`literal`', b'\xff\x80non-utf8']


def load(name, path):
    loader = SourceFileLoader(name, str(path))
    module = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(module)
    return module


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=True).encode()


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def file_sha(path):
    result = hashlib.sha256()
    with Path(path).open('rb') as source:
        for chunk in iter(lambda: source.read(1 << 20), b''):
            result.update(chunk)
    return result.hexdigest()


def save(path, value):
    path.write_bytes(encode(value) + b'\n')


def bytes_environment():
    return {key.hex(): value.hex() for key, value in sorted(os.environb.items())}


def fixture():
    """Internal fixture modes are explicitly outside installed/compiler admission."""
    mode, *args = sys.argv[2:]
    root = Path(os.environ['INTERCEPTION_CAPTURE_ROOT'])
    if mode == 'compiler':
        save(root / ('compiler-' + str(os.getpid()) + '.json'),
             dict(argv=[os.fsencode(arg).hex() for arg in args], environment=bytes_environment(),
                  executable=str(Path(globals().get('FIXTURE_EXECUTABLE', sys.argv[0])).resolve()), pid=os.getpid()))
        if args == ['env', 'GOOS']:
            print(os.uname().sysname.lower())
        return 0
    if mode == 'host-child':
        save(root / 'inherited.json', dict(environment=bytes_environment()))
        return subprocess.call([b'go', *[os.fsencode(arg) for arg in args]], env=os.environb)
    if mode == 'named-child':
        save(root / 'named.json', dict(argv=[os.fsencode(arg).hex() for arg in args]))
        return 0
    if mode == 'concurrent-child':
        commands = json.loads(Path(args[0]).read_bytes())
        barrier = threading.Barrier(len(commands))
        statuses = [None] * len(commands)
        def execute(index, command):
            barrier.wait()
            statuses[index] = subprocess.call(['go', *command])
        threads = [threading.Thread(target=execute, args=(index, command))
                   for index, command in enumerate(commands)]
        for thread in threads:
            thread.start()
        for thread in threads:
            thread.join()
        save(root / 'concurrent-statuses.json', statuses)
        return 0 if all(status == 0 for status in statuses) else 1
    if mode == 'external-launcher':
        module = Path(args[0])
        with module.open('rb') as source:
            equal(source.read(8), b'\0asm\x01\0\0\0', 'external.actual_wasm_module')
        invocation = Path(os.environ['GOMAD_SHIM_INVOCATION'])
        raw = invocation.read_bytes()
        if sha(raw) != os.environ['GOMAD_SHIM_INVOCATION_SHA256']:
            raise ValueError('external launcher invocation digest mismatch')
        metadata = json.loads(raw)
        recipe = invocation.parent / metadata['recipe_file']
        recipe_raw = recipe.read_bytes()
        if sha(recipe_raw) != metadata['recipe_sha256']:
            raise ValueError('external launcher recipe digest mismatch')
        shared = json.loads(recipe_raw)
        for prepared in shared['packages']:
            for key, value in dict(GOOS='wasip1', GOARCH='wasm', CGO_ENABLED='0', GOTOOLCHAIN='local').items():
                equal(prepared['environment'][key], value, 'external.prepared_guest_environment.' + key)
        inventories = []
        for prepared in shared['packages']:
            observed = {}
            for kind in ('compiler','sources'):
                observed[kind] = {path: file_sha(path) for path in prepared[kind]}
                equal(observed[kind], prepared[kind], 'external.current_' + kind + '_inventory')
            inventories.append(observed)
        save(root / ('external-' + str(os.getpid()) + '.json'),
             dict(module_sha256=file_sha(module), module_bytes=module.stat().st_size,
                  argv=[os.fsencode(arg).hex() for arg in args], environment=bytes_environment(),
                  invocation=metadata, invocation_sha256=sha(raw), invocation_base64=base64.b64encode(raw).decode(),
                  inventories=inventories,
                  recipe_base64=base64.b64encode(recipe_raw).decode(), pid=os.getpid()))
        return 0
    raise ValueError('unknown internal fixture mode')


def program(path, mode):
    path.write_text('#!' + sys.executable + '\nimport runpy,sys\nsys.argv[1:1]=["--fixture",' +
                    repr(mode) + ']\nrunpy.run_path(' + repr(str(SELF)) + ',init_globals={"FIXTURE_EXECUTABLE":sys.argv[0]},run_name="__main__")\n')
    path.chmod(0o700)


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(label + ': actual ' + repr(actual) + ', expected ' + repr(expected))


def expand(arguments, bindings):
    result = []
    for argument in arguments:
        if argument == '$RAW_ARGS':
            result.extend(RAW_ARGS)
        else:
            for key, value in bindings.items():
                argument = argument.replace('$' + key, str(value))
            result.append(os.fsencode(argument))
    return result


def run(argv, cwd, env, output, timeout):
    save(output / 'command.json', dict(argv=[os.fsencode(arg).hex() for arg in argv], cwd=str(cwd),
                                     environment={key.hex(): value.hex() for key, value in sorted(env.items())}))
    with (output / 'stdout').open('wb') as stdout, (output / 'stderr').open('wb') as stderr:
        child = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stdout, stderr=stderr, start_new_session=True)
        save(output / 'started.json', dict(pid=child.pid))
        try:
            status = child.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            os.killpg(child.pid, signal.SIGTERM)
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                child.wait()
            save(output / 'result.json', dict(timed_out=True, status=child.returncode))
            raise ValueError('external deadline expired; no retry')
    result = dict(status=status, timed_out=False, stdout_sha256=file_sha(output / 'stdout'),
                  stderr_sha256=file_sha(output / 'stderr'))
    save(output / 'result.json', result)
    return result


def fake_case(case, binary, output, timeout):
    module, bindir, capture = [output / name for name in ('module', 'bin', 'capture')]
    for path in (module, bindir, capture, module / 'runtime'):
        path.mkdir(mode=0o700, parents=True, exist_ok=True)
    (module / 'go.mod').write_text('module interception.offline\n')
    (module / 'gomad.toml').write_text(case.get('config', 'runtime_root="runtime"\nderive="default"\n'))
    (module / 'escape').symlink_to(output.parent, target_is_directory=True)
    program(bindir / 'go', 'compiler')
    program(bindir / 'replay', 'named-child')
    child = bindir / 'host-child'
    program(child, 'host-child')
    env = {os.fsencode(key): os.fsencode(value) for key, value in
           dict(PATH=str(bindir) + os.pathsep + os.environ.get('PATH', '/usr/bin:/bin'),
                XDG_CACHE_HOME=str(output / 'cache'), HOME=str(output / 'home'),
                LC_ALL='C.UTF-8', PYTHONCOERCECLOCALE='0',
                INTERCEPTION_CAPTURE_ROOT=str(capture),
                INTERCEPTION_RAW_ENV=os.fsdecode(b'\xff\x80env value')).items()}
    for key, value in case.get('env', {}).items():
        env[os.fsencode(key)] = os.fsencode(value)
    prefix = [os.fsencode(binary), b'--artifacts', os.fsencode(output / 'artifacts')]
    argv = expand(case['argv'], {})
    if case['stage'] == 'host':
        argv = prefix + [b'--', os.fsencode(child)] + argv
    elif case['stage'] == 'refusal':
        argv = prefix + [b'--', b'go'] + argv
    elif case['stage'] == 'config':
        argv = prefix + [b'--', b'true']
    else:
        if case['name'] == 'wrapper-filesystem':
            (output / 'cache').write_bytes(b'not a directory')
        argv = [os.fsencode(binary)] + argv
    result = run(argv, module, env, output, timeout)
    equal(result['status'], case['status'], case['name'] + '.status')
    calls = [json.loads(path.read_bytes()) for path in sorted(capture.glob('compiler-*.json'))]
    semantic = dict(case=case['name'], status=result['status'], compiler_calls=len(calls),
                    synthetic_compiler=True, installed_admission=False)
    if case['stage'] == 'host':
        equal(len(calls), 1, 'host.exact_compiler_call')
        inherited = json.loads((capture / 'inherited.json').read_bytes())
        equal(calls[0]['environment'], inherited['environment'], 'host.byte_exact_environment')
        equal(calls[0]['argv'], [arg.hex() for arg in expand(case['argv'], {})], 'host.byte_exact_argv')
        equal(calls[0]['executable'], str(bindir / 'go'), 'host.exact_compiler_path')
        if case['name'] == 'host-env-goos':
            equal((output / 'stdout').read_bytes(), (os.uname().sysname.lower() + '\n').encode(), 'host.goos')
        semantic.update(argv=calls[0]['argv'], environment_exact=True, compiler_path_exact=True)
    elif case['stage'] in ('refusal', 'config'):
        equal(calls, [], 'refusal.before_any_compiler_call')
        equal((output / 'stdout').read_bytes(), b'', 'refusal.no_stdout')
        stderr = (output / 'stderr').read_bytes()
        equal(len(stderr.splitlines()), 1, 'refusal.one_stderr_line')
        if case['stage'] == 'refusal' and case['diagnostic'].encode() not in stderr:
            raise ValueError('missing exact refused option diagnostic')
        semantic['diagnostic'] = stderr.decode()
    elif case['name'] == 'wrapper-named-replay':
        equal(json.loads((capture / 'named.json').read_bytes())['argv'],
              [os.fsencode(value).hex() for value in case['argv'][2:]], 'wrapper.named_child_arguments')
    elif case['name'] == 'wrapper-direct-go':
        equal(len(calls), 1, 'wrapper.direct_go_reaches_shim')
        equal(calls[0]['argv'], [b'version'.hex()], 'wrapper.direct_go_arguments')
    else:
        equal(calls, [], 'wrapper.no_compiler_call')
    equal(list((output / 'artifacts').rglob('plan.json')), [], 'synthetic.no_guest_campaign')
    # Error text can name the fresh fixture root; preserve raw stderr above.
    if 'diagnostic' in semantic:
        semantic['diagnostic'] = semantic['diagnostic'].replace(str(output), '$OUTPUT')
    return semantic


def schema_inventory(checkout):
    root = checkout / 'schema/sqlite'
    return {str(path.relative_to(root)): file_sha(path) for path in sorted(root.rglob('*')) if path.is_file()}


def authenticate_cache(cache, binary, compiler, checkout, previous_binary=None):
    checker = load('shim_installation', HERE / 'installation-lane.py')
    result = {}
    for path in sorted(cache.rglob('derived-installations/*.json')):
        raw = path.read_bytes()
        manifest = json.loads(raw)
        equal(checker.encode(manifest), raw, 'cache.canonical_manifest_bytes')
        owner = binary
        if manifest['admission']['host_executable_sha256'] != file_sha(binary):
            if previous_binary is None or file_sha(previous_binary) != manifest['admission']['host_executable_sha256']:
                raise ValueError('unbound prior executable in derived cache')
            owner = previous_binary
        checker.project_manifest(manifest, cache, SimpleNamespace(binary=owner, go_compiler=compiler,
                                 go_source=checkout, preparation_root=cache))
        checker.authenticate_captured(manifest['admission']['guest_configuration'])
        captured = manifest['admission']['guest_configuration']['CapturedInputs']
        descriptor = json.loads(base64.b64decode(captured['Descriptor'], validate=True))
        schema = {entry['path'].removeprefix('/workspace/schema/sqlite/'): entry['sha256'].removeprefix('sha256:')
                  for entry in descriptor['entries'] if entry.get('payload') and
                  entry['path'].startswith('/workspace/schema/sqlite/')}
        result[str(path)] = dict(sha256=sha(raw), configuration_sha256=manifest['admission']['configuration_sha256'],
                                host_executable_sha256=manifest['admission']['host_executable_sha256'], schema_inventory=schema)
    return result


def authenticate_build_marker(provenance, origin, record):
    inputs = provenance['inputs']
    marker = inputs['build_launcher']
    expected = dict(schema='gomad.build-launcher/v1', module_sha256=record['identity']['target'],
                    invocation_sha256=sha(encode(origin['invocation'])),
                    catalog_sha256=origin['invocation']['installation_sha256'],
                    recipe_sha256=origin['invocation']['recipe_sha256'])
    equal(set(marker), {*expected, 'installation_sha256'}, 'build_launcher.marker_fields')
    for key, value in expected.items():
        equal(marker[key], value, 'build_launcher.' + key)
    bindings = inputs['shared_driver_installations']
    if len(bindings) > 65536:
        raise ValueError('build_launcher catalog exceeds bound')
    authenticate_installations(provenance, origin, bindings)
    selected = [inputs['go_build_info']['path'], marker['installation_sha256']]
    if selected not in bindings:
        raise ValueError('build_launcher selected installation differs from retained catalog')
    registration = inputs['build_driver_registration']
    equal(set(registration), {'schema','process_id','invocation_sha256','compiler_sha256',
                              'driver_argv_sha256','module_root'}, 'build_launcher.registration_fields')
    equal(registration['schema'], 'gomad.driver-registration/v1', 'build_launcher.registration_schema')
    equal(registration['invocation_sha256'], marker['invocation_sha256'], 'build_launcher.registered_invocation')
    equal(registration['compiler_sha256'], origin['invocation']['compiler_sha256'], 'build_launcher.registered_compiler')
    equal(registration['driver_argv_sha256'], sha(encode(origin['invocation']['driver_argv'])), 'build_launcher.registered_driver_argv')
    if type(registration['process_id']) is not int or not 0 < registration['process_id'] < 1 << 32:
        raise ValueError('build_launcher invalid original Go PID')
    if not isinstance(registration['module_root'], str) or not Path(registration['module_root']).is_absolute():
        raise ValueError('build_launcher original module root is not absolute')
    driver = [bytes.fromhex(value) for value in origin['invocation']['driver_argv']]
    index = 1
    while index < len(driver):
        if driver[index] == b'-C':
            index += 2
        elif driver[index].startswith(b'-C='):
            index += 1
        else:
            break
    if index >= len(driver) or driver[index] != b'build':
        raise ValueError('build_launcher original driver is not go build')


def authenticated_plans(root):
    checker = load('shim_plan_authentication', HERE / 'interception-check.py')
    campaigns, publications = {}, []
    for path in sorted(root.rglob('plan.json')):
        if path.is_symlink() or not path.resolve().is_relative_to(root.resolve()):
            raise ValueError('unsafe retained plan path')
        if not (path.parent / 'storage.json').exists():
            publications.append(path)
            continue
        plan = checker.campaign.canonical_file(path)
        complete = 'launcher' in plan
        if complete:
            if type(plan['launcher']) is not bool or not plan['launcher']:
                raise ValueError('campaign.launcher_role must be true')
        validated = checker.campaign.authenticate_campaign(path.parent, complete=complete)
        artifact = path.parent / plan['prepared_artifact']
        record = checker.campaign.authenticate_artifact(artifact)
        origin = checker.authenticate_origin(artifact, record, plan['origin'])
        provenance = checker.campaign.canonical_file(artifact / 'target/provenance.json')
        if 'build_launcher' in provenance['inputs']:
            authenticate_build_marker(provenance, origin, record)
        if not complete:
            equal((validated['journal'], validated['executions']), ([], []), 'preparation.zero_guest_journal')
            equal((record['outcome']['class'], record['outcome']['termination']),
                  ('prepared','prepared'), 'preparation.record_role')
        campaigns[path.parent.resolve()] = dict(root=path.parent, validated=validated, prepared=record,
                                                artifact=artifact, origin=origin, complete=complete)
    admitted = set()
    for path in publications:
        relative = path.relative_to(root)
        if len(relative.parts) != 4 or relative.parts[0] not in ('admissions','build-launchers'):
            raise ValueError('unknown noncampaign plan publication')
        receipt_path = path.with_name(path.name + '.preparation.json')
        if not receipt_path.is_file() or receipt_path.is_symlink():
            raise ValueError('admission publication missing preparation receipt')
        receipt = checker.campaign.canonical_file(receipt_path)
        equal((receipt['schema'], receipt['guest_executions'], receipt['exit_code']),
              ('gomad.rust-preparation-receipt/v1', 0, 0), 'receipt.zero_guest_admission')
        target = Path(receipt['campaign']).resolve()
        if target not in campaigns or campaigns[target]['complete'] or target in admitted:
            raise ValueError('admission receipt references unknown, completed or duplicate preparation')
        prepared = campaigns[target]
        if relative.parts[0] == 'build-launchers':
            origin = prepared['origin']
            equal(relative.parts[1], origin['invocation']['invocation_id'], 'build_launcher.publication_invocation')
            if not re.fullmatch('[0-9a-f]{64}', relative.parts[2]):
                raise ValueError('build_launcher invalid publication identity')
            provenance = checker.campaign.canonical_file(prepared['artifact'] / 'target/provenance.json')
            authenticate_build_marker(provenance, origin, prepared['prepared'])
            equal(receipt['command'], ['os-hex:' + value for value in origin['invocation']['original_argv']],
                  'build_launcher.exact_original_command')
        equal(Path(receipt['plan']), path.resolve(), 'receipt.published_plan_path')
        equal(checker.parity.digest(path.read_bytes()), receipt['plan_sha256'], 'receipt.plan_digest')
        equal(path.read_bytes(), (target / 'plan.json').read_bytes(), 'receipt.published_plan_bytes')
        equal(receipt['prepared_artifact'], prepared['validated']['plan']['prepared_artifact'], 'receipt.prepared_artifact')
        manifest = checker.campaign.canonical_file(prepared['artifact'] / 'manifest.json')
        equal(receipt['record_sha256'], manifest['record_sha256'], 'receipt.record_digest')
        for prefix, filename in [('module','target/module.wasm'), ('provenance','target/provenance.json')]:
            equal(receipt[prefix + '_file'], filename, 'receipt.payload_path')
            raw = (prepared['artifact'] / filename).read_bytes()
            equal((len(raw), checker.parity.digest(raw)),
                  (receipt[prefix + '_bytes'], receipt[prefix + '_sha256']), 'receipt.payload_binding')
        admitted.add(target)
    preparations = {path for path, value in campaigns.items() if not value['complete']}
    equal(admitted, preparations, 'preparation.exact_admission_receipts')
    origins = [value['origin'] for value in campaigns.values() if not value['complete']]
    for value in campaigns.values():
        if value['complete'] and value['origin'] not in origins:
            raise ValueError('launcher campaign has no matching authenticated preparation')
    return list(campaigns.values())


def authenticate_campaigns(root, expected, compiler_sha256):
    checker = load('shim_interception', HERE / 'interception-check.py')
    observations = []
    for retained in authenticated_plans(root):
        campaign = retained['validated']
        plan = campaign['plan']
        prepared_path = retained['artifact']
        prepared = retained['prepared']
        origin = retained['origin']
        if not campaign['executions']:
            continue
        equal(origin['invocation']['compiler_sha256'], compiler_sha256, 'real.compiler_binding')
        original = [bytes.fromhex(value) for value in origin['invocation']['original_argv']]
        commands = expected.get('commands', [expected['argv']])
        if original[1:] not in [[os.fsencode(value) for value in command] for command in commands]:
            raise ValueError('real original argv differs from every accepted command')
        if expected.get('installation_bindings'):
            bindings = []
            for binding in expected['installation_bindings']:
                equal(file_sha(binding['path']), binding['sha256'], 'real.installation_binding_bytes')
                bindings.append([binding['import_path'], binding['sha256']])
            provenance = checker.campaign.canonical_file(prepared_path / 'target/provenance.json')
            authenticate_installations(provenance, origin, bindings)
        rewritten = [bytes.fromhex(value).decode('utf-8') for value in origin['invocation']['driver_argv']]
        effective = driver_flags(rewritten[2:])
        if original[1] != b'build' and 'run-guest' not in effective.get('exec', ''):
            raise ValueError('authenticated real driver did not select Gomad run-guest')
        if original[1] == b'test':
            selected_count = driver_flags([os.fsdecode(value) for value in original[2:]]).get('count', '1')
            equal(effective.get('count'), selected_count, 'real.effective_caller_count')
        outcomes = []
        for execution in campaign['executions']:
            if execution['artifact'] is None:
                raise ValueError('real execution missing retained artifact')
            artifact = retained['root'] / execution['artifact']
            record = checker.campaign.authenticate_artifact(artifact)
            equal(checker.authenticate_origin(artifact, record, plan['origin']), origin, 'real.execution_origin')
            for key, expected_limit in expected['limits'].items():
                equal(record['limits'][key], expected_limit, 'real.accepted_limit.' + key)
            outcomes.append(dict(seed=record['seed'], outcome=record['outcome']))
        observations.append(dict(origin=origin, outcomes=outcomes,
                                 configuration_sha256=prepared['identity']['configuration']))
    equal(len(observations), expected['campaigns'], 'real.campaign_count')
    expected_outcomes = expected.get('outcome_classes', ['success'])
    equal(sorted(item['outcome']['class'] for row in observations for item in row['outcomes']),
          sorted(expected_outcomes), 'real.actual_outcomes')
    return observations


def driver_flags(arguments):
    result = {}
    index = 0
    while index < len(arguments):
        value = arguments[index]
        if value in ('-args','--args','--'):
            break
        if value.startswith('-'):
            name, separator, content = value.lstrip('-').partition('=')
            if name in ('exec','count'):
                if not separator:
                    index += 1
                    if index >= len(arguments):
                        raise ValueError('driver control flag lacks value')
                    content = arguments[index]
                result[name] = content
        index += 1
    return result


def authenticate_installations(provenance, origin, expected_bindings):
    bindings = provenance['inputs']['shared_driver_installations']
    if not isinstance(bindings, list) or not bindings:
        raise ValueError('missing retained shared installation bindings')
    names = set()
    for binding in bindings:
        if not isinstance(binding, list) or len(binding) != 2:
            raise ValueError('invalid shared installation binding shape')
        name, digest = binding
        if not isinstance(name, str) or not name or name in names or not isinstance(digest, str):
            raise ValueError('duplicate or invalid shared installation package')
        if len(digest) != 64 or any(value not in '0123456789abcdef' for value in digest):
            raise ValueError('invalid shared installation digest')
        names.add(name)
    equal(bindings, expected_bindings, 'retained.exact_installation_mapping')
    # Match the Rust canonical JSON encoding, which does not escape ordinary Unicode.
    raw = json.dumps(bindings, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    equal(sha(raw), origin['invocation']['installation_sha256'], 'retained.installation_commitment')


def validate_real(case, inputs):
    name = case['name']
    for key, ceiling in dict(memory_bytes=4 << 30, fuel=1 << 50, wall_millis=300_000).items():
        value = inputs['limits'][key]
        if type(value) is not int or not 0 < value <= ceiling:
            raise ValueError('real limit exceeds original accepted ceiling: ' + key)
    argv = inputs['argv']
    if not isinstance(argv, list) or not argv or not all(isinstance(value, str) for value in argv):
        raise ValueError('explicit real argv strings required')
    selections = inputs['accepted_selections']
    if not isinstance(selections, list) or not selections:
        raise ValueError('reviewed accepted selections required')
    for selection in selections:
        for key in ('package','selector','receipt','receipt_sha256'):
            if not isinstance(selection.get(key), str) or not selection[key]:
                raise ValueError('accepted selection missing ' + key)
        equal(file_sha(selection['receipt']), selection['receipt_sha256'], 'accepted.selection_receipt')
    zero = name.startswith('auto-') or name in ('external-launcher','exec-true-count0','explicit-excluded-tag')
    count = 0 if zero else (2 if name in ('multi-package-json','concurrent-identities') else 1)
    equal(inputs['campaigns'], count, 'real.required_campaign_count')
    if name not in ('guest-lookalikes','auto-schema-missing','explicit-excluded-tag'):
        equal(inputs['status'], 0, 'real.required_success')
    elif name in ('auto-schema-missing','explicit-excluded-tag'):
        equal(inputs['status'], 2, 'real.required_input_refusal')
    if name.startswith('auto-'):
        if inputs.get('installation'):
            raise ValueError('auto case cannot select an explicit installation')
        if '-exec=true' not in argv or '-count=0' not in argv:
            raise ValueError('auto cache probes must remain explicit compile-only')
    elif not inputs.get('installation'):
        raise ValueError('real branch requires the supplied genuine explicit installation')
    commands = inputs.get('commands', [argv])
    for selection in selections:
        if not any(selection['package'] in command for command in commands):
            raise ValueError('accepted concrete package missing from actual command')
        if argv[0] == 'test' and not any(
                value in ('-run=' + selection['selector'], '-test.run=' + selection['selector'])
                for command in commands for value in command):
            raise ValueError('exact accepted selector missing from actual command')
    if name == 'caller-count' and '-count=2' not in argv:
        raise ValueError('caller count2 probe required')
    for label, token in (('boundary-args','-args'),('boundary-double-args','--args'),
                         ('boundary-terminator','--')):
        if name == label and token not in argv:
            raise ValueError('missing exact boundary token ' + token)
    if name == 'guest-lookalikes':
        if '-args' not in argv or not {'-tags=guest-only','-exec=guest-only','-count=guest-only'} <= set(argv[argv.index('-args') + 1:]):
            raise ValueError('all guest flag lookalikes must follow -args')
        equal(inputs['status'], 1, 'real.guest_unknown_flag_failure')
        equal(inputs.get('outcome_classes'), ['target_failure'], 'real.guest_lookalike_outcome')
    if name == 'goflags-quoted' and not any(quote in inputs['environment'].get('GOFLAGS','') for quote in ('"', "'")):
        raise ValueError('quoted effective GOFLAGS required')
    if name == 'repeated-effective-flags' and not {'-race=true','-race=false'} <= set(argv):
        raise ValueError('effective false repeated race flags required')
    if name == 'empty-effective-exec' and not {'-exec=false','-exec='} <= set(argv):
        raise ValueError('overridden nonempty exec and final empty exec required')
    if name == 'external-launcher':
        if not any('$EXTERNAL_LAUNCHER' in value for value in argv):
            raise ValueError('actual external fixture launcher required')
        if not isinstance(inputs.get('launcher_arguments'), list) or not inputs.get('installation_bindings'):
            raise ValueError('external exact launcher arguments and installation mapping required')
    if name == 'exec-true-count0' and not {'-exec=true','-count=0'} <= set(argv):
        raise ValueError('literal caller-owned true/count0 case required')
    if name == 'multi-package-json':
        if not {'-json','-v','-p=2'} <= set(argv) or len(selections) != 2:
            raise ValueError('actual two-package -json/-v/-p=2 selection required')
    if name == 'concurrent-identities':
        commands = inputs.get('commands', [])
        if len(commands) != 2 or commands[0] == commands[1] or any(command[0] != 'test' for command in commands):
            raise ValueError('two distinct real Go test invocations required')
    if name in ('guest-run','guest-build') and argv[0] != name.removeprefix('guest-'):
        raise ValueError('matching guest main subcommand required')
    if name == 'guest-build' and ('$BUILD_OUTPUT' not in argv or '-o' not in argv):
        raise ValueError('fresh durable build launcher output required')
    if name == 'explicit-excluded-tag':
        manifest = json.loads(Path(inputs['installation']).read_bytes())
        excluded = manifest.get('excluded_build_tags', [])
        if not excluded or not any(value.startswith('-tags=') and any(tag in value.split('=',1)[1].split(',') for tag in excluded) for value in argv):
            raise ValueError('genuine explicit exclusion policy and excluded caller tag required')
    if inputs['campaigns'] and name != 'guest-lookalikes':
        equal(inputs.get('outcome_classes'), ['success'] * count, 'real.required_actual_successes')
    if inputs['campaigns'] and not inputs.get('installation_bindings'):
        raise ValueError('exact retained installation mapping expectations required')


def authenticate_external(captures, expected, compiler):
    equal(len(captures), 1, 'external.exactly_one_capture')
    captured = captures[0]
    raw = base64.b64decode(captured['invocation_base64'], validate=True)
    invocation = json.loads(raw)
    canonical = lambda value: json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    equal(canonical(invocation), raw, 'external.canonical_invocation')
    equal(invocation, captured['invocation'], 'external.captured_invocation_object')
    equal(sha(raw), captured['invocation_sha256'], 'external.invocation_commitment')
    equal(invocation['schema'], 'gomad.intercepted-invocation/v1', 'external.invocation_schema')
    equal(invocation['recipe_file'], 'driver-recipe.json', 'external.exact_recipe_file')
    for key in ('wrapper_run_id','invocation_id','compiler_sha256','installation_sha256','recipe_sha256'):
        value = invocation[key]
        if not isinstance(value, str) or not re.fullmatch('[0-9a-f]{64}', value):
            raise ValueError('external invalid invocation digest: ' + key)
    if invocation['wrapper_run_id'] == invocation['invocation_id']:
        raise ValueError('external driver aliases wrapper identity')
    equal(invocation['original_argv'], [os.fsencode(value).hex() for value in ['go', *expected['argv']]],
          'external.exact_original_caller_argv')
    equal(invocation['compiler_sha256'], expected['compiler_sha256'], 'external.expected_compiler')
    equal(file_sha(compiler), expected['compiler_sha256'], 'external.actual_compiler_bytes')
    env = {bytes.fromhex(key): bytes.fromhex(value) for key, value in captured['environment'].items()}
    equal(env[b'GOMAD_SHIM_INVOCATION_SHA256'].decode(), sha(raw), 'external.inherited_invocation_commitment')
    equal(env[b'GOMAD_SHIM_RUN'].decode(), invocation['wrapper_run_id'], 'external.inherited_wrapper_identity')
    driver = [bytes.fromhex(value).decode('utf-8') for value in invocation['driver_argv']]
    equal(driver[:2], [str(compiler), 'test'], 'external.exact_driver_compiler_subcommand')
    equal(shlex.split(driver_flags(driver[2:])['exec']), [expected['external_launcher']], 'external.effective_caller_launcher')
    received = [bytes.fromhex(value) for value in captured['argv']]
    if not received or not Path(os.fsdecode(received[0])).is_absolute():
        raise ValueError('external missing actual Go module operand')
    equal(received[1:], [os.fsencode(value) for value in expected['launcher_arguments']], 'external.exact_launcher_arguments')
    if type(captured['module_bytes']) is not int or captured['module_bytes'] < 8 or not re.fullmatch('[0-9a-f]{64}', captured['module_sha256']):
        raise ValueError('external invalid actual module commitment')
    recipe_raw = base64.b64decode(captured['recipe_base64'], validate=True)
    shared = json.loads(recipe_raw)
    equal(canonical(shared), recipe_raw, 'external.canonical_recipe')
    equal(sha(recipe_raw), invocation['recipe_sha256'], 'external.recipe_commitment')
    equal(shared['schema'], 'gomad.shim-driver/v1', 'external.recipe_schema')
    equal(shared['driver_argv'], invocation['driver_argv'], 'external.exact_driver_argv')
    bindings = expected['installation_bindings']
    equal(len(shared['packages']), len(bindings), 'external.exact_package_count')
    equal(shared['installations'], [binding['sha256'] for binding in bindings], 'external.exact_installation_digests')
    ordered = [[binding['import_path'], binding['sha256']] for binding in bindings]
    if len({binding[0] for binding in ordered}) != len(ordered):
        raise ValueError('external duplicate installation package')
    equal(sha(canonical(ordered)), invocation['installation_sha256'], 'external.installation_mapping_commitment')
    equal(len(captured['inventories']), len(bindings), 'external.inventory_capture_count')
    for prepared, binding, observed in zip(shared['packages'], bindings, captured['inventories']):
        equal(prepared['schema'], 'gomad.driver-recipe/v1', 'external.package_recipe_schema')
        equal(prepared['import_path'], binding['import_path'], 'external.exact_package_order')
        equal(prepared['options']['target_kind'], 'go-test', 'external.prepared_target_kind')
        equal(prepared['options']['source'], binding['import_path'], 'external.prepared_source')
        for key, value in [('source', prepared['options']['source']), ('kind', prepared['options']['target_kind']),
                           ('directory', prepared['directory']), ('configuration', prepared['configuration'])]:
            equal(prepared['inputs'][key], value, 'external.recipe_input.' + key)
        equal(file_sha(binding['path']), binding['sha256'], 'external.actual_installation_bytes')
        equal(prepared['installed'], json.loads(Path(binding['path']).read_bytes()), 'external.exact_installed_manifest')
        equal(prepared['installed']['compiler_path'], str(compiler), 'external.recipe_compiler_path')
        equal(prepared['installed']['compiler_sha256'], expected['compiler_sha256'], 'external.recipe_compiler_identity')
        equal(prepared['options']['installation'], binding['path'], 'external.recipe_installation_selection')
        for kind in ('compiler','sources'):
            if not prepared[kind]:
                raise ValueError('external empty ' + kind + ' inventory')
            equal(prepared['inputs'][kind], prepared[kind], 'external.recipe_' + kind + '_commitment')
            equal(observed[kind], prepared[kind], 'external.observed_' + kind + '_inventory')
        equal(prepared['inputs']['compiler_path'], str(compiler), 'external.recipe_input.compiler_path')
        equal(prepared['inputs']['compiler_sha256'], expected['compiler_sha256'], 'external.recipe_input.compiler_sha256')
        for key, value in dict(GOOS='wasip1', GOARCH='wasm', CGO_ENABLED='0', GOTOOLCHAIN='local').items():
            equal(prepared['environment'][key], value, 'external.guest_environment.' + key)
    return captured


def real_case(case, binary, output, timeout, inputs):
    required = ('schema', 'case', 'compiler', 'compiler_sha256', 'checkout', 'argv', 'status',
                'campaigns', 'accepted_selections', 'cache', 'environment', 'limits')
    if any(key not in inputs for key in required):
        raise ValueError('incomplete reviewed real-case input declaration')
    equal(inputs['schema'], 'gomad.interception-shim-real/v1', 'real.inputs_schema')
    equal(inputs['case'], case['name'], 'real.case')
    validate_real(case, inputs)
    compiler, checkout, cache = (Path(inputs[key]) for key in ('compiler', 'checkout', 'cache'))
    if any(not path.is_absolute() for path in (compiler, checkout, cache)):
        raise ValueError('absolute real paths required')
    equal(file_sha(compiler), inputs['compiler_sha256'], 'real.pinned_compiler')
    if not inputs['accepted_selections']:
        raise ValueError('explicit accepted package/selector declarations required')
    if not (checkout / 'go.mod').is_file() or not (checkout / 'gomad.toml').is_file():
        raise ValueError('real module/config inputs missing')
    env = {os.fsencode(key): os.fsencode(value) for key, value in os.environ.items()
           if not key.startswith('GOMAD_') and key not in ('GOOS','GOARCH','GOROOT','GOFLAGS','CGO_ENABLED')}
    env.update({b'PATH': os.fsencode(str(compiler.parent) + os.pathsep + os.environ.get('PATH', '')),
                b'GOENV':b'off', b'GOTOOLCHAIN':b'local', b'GOWORK':b'off', b'XDG_CACHE_HOME':os.fsencode(cache),
                b'INTERCEPTION_CAPTURE_ROOT':os.fsencode(output)})
    env.update({os.fsencode(key): os.fsencode(value) for key, value in inputs['environment'].items()})
    artifacts = output / 'artifacts'
    command = [os.fsencode(binary), b'--seeds', b'0', b'--artifacts', os.fsencode(artifacts)]
    installation = inputs.get('installation')
    if installation:
        equal(file_sha(installation), inputs['installation_sha256'], 'real.installation_bytes')
        command.extend([b'--installation', os.fsencode(installation)])
    launcher = output / 'external-launcher'
    program(launcher, 'external-launcher')
    argv = [os.fsdecode(value) for value in expand(inputs['argv'], {'EXTERNAL_LAUNCHER':launcher, 'BUILD_OUTPUT':output / 'published-main'})]
    actual_expected = dict(inputs, argv=argv, external_launcher=str(launcher))
    before = schema_inventory(checkout)
    save(output / 'schema-before.json', before)
    cache_before = {str(path): dict(sha256=file_sha(path), mtime_ns=path.stat().st_mtime_ns)
                    for path in cache.rglob('derived-installations/*.json')}
    save(output / 'cache-before.json', cache_before)
    if case['name'] == 'concurrent-identities':
        child = output / 'concurrent-child'
        program(child, 'concurrent-child')
        save(output / 'concurrent-commands.json', inputs['commands'])
        child_argv = [b'--', os.fsencode(child), os.fsencode(output / 'concurrent-commands.json')]
    else:
        child_argv = [b'--', b'go'] + [os.fsencode(value) for value in argv]
    result = run(command + child_argv, checkout, env, output, timeout)
    equal(result['status'], inputs['status'], 'real.actual_driver_status')
    if case['name'] == 'guest-build':
        published = output / 'published-main'
        module = output / 'published-main.wasm'
        if not published.is_file() or not os.access(published, os.X_OK) or not module.read_bytes().startswith(b'\0asm'):
            raise ValueError('guest build failed to publish durable module and executable launcher')
        save(output / 'published.json', dict(launcher_sha256=file_sha(published), module_sha256=file_sha(module)))
        replay_output = output / 'published-execution'
        replay_output.mkdir(mode=0o700)
        launched = run([os.fsencode(published)], checkout, env, replay_output, timeout)
        equal(launched['status'], 0, 'real.durable_launcher_runs_after_wrapper_cleanup')
    equal(schema_inventory(checkout), before, 'real.stage_did_not_mutate_schema')
    observations = authenticate_campaigns(artifacts, actual_expected, inputs['compiler_sha256'])
    go_events = []
    if case['name'] == 'multi-package-json':
        events = [json.loads(line) for line in (output / 'stdout').read_bytes().splitlines()]
        save(output / 'go-events.json', events)
        packages = {event['Package'] for event in events if event.get('Action') == 'pass' and not event.get('Test')}
        equal(len(packages), 2, 'real.go_json_package_passes')
        for selection in inputs['accepted_selections']:
            if not any(event.get('Action') == 'pass' and re.fullmatch(selection['selector'], event.get('Test','')) for event in events):
                raise ValueError('Go JSON lacks accepted selected test PASS')
        go_events = sorted((event['Action'], event.get('Package',''), event.get('Test','')) for event in events
                           if event.get('Action') in ('start','run','pass','fail','skip'))
    external = [json.loads(path.read_bytes()) for path in sorted(output.glob('external-*.json'))]
    if case['name'] == 'external-launcher':
        authenticate_external(external, actual_expected, compiler)
        equal(observations, [], 'real.external_launcher_has_no_gomad_execution_claim')
    if case['name'] == 'exec-true-count0':
        equal((external, observations), ([], []), 'real.compile_only_no_guest')
    cache_after = authenticate_cache(cache, binary, compiler, checkout,
                                     Path(inputs['previous_binary']) if inputs.get('previous_binary') else None) if case['name'].startswith('auto-') else {}
    save(output / 'cache-authenticated.json', cache_after)
    baseline = None
    if case['name'].startswith('auto-') and case['name'] != 'auto-prime':
        baseline_path = Path(inputs['baseline'])
        baseline = json.loads(baseline_path.read_bytes())
        equal(file_sha(baseline_path), inputs['baseline_sha256'], 'auto.baseline_receipt_bytes')
        equal(baseline['accepted'], True, 'auto.baseline_accepted')
        old = baseline['cache']
        if case['name'] == 'auto-reuse':
            equal({key: value['sha256'] for key, value in cache_after.items()},
                  {key: value['sha256'] for key, value in old.items()}, 'auto.same_entries')
            equal({str(path): dict(sha256=file_sha(path), mtime_ns=path.stat().st_mtime_ns)
                   for path in cache.rglob('derived-installations/*.json')}, cache_before, 'auto.no_manifest_rewrite')
        elif case['name'] == 'auto-schema-missing':
            equal(before, {}, 'auto.schema_unavailable')
            equal(result['status'], 2, 'auto.missing_schema_refusal')
            equal(cache_after, old, 'auto.missing_schema_no_fallback')
        elif case['name'] in ('auto-schema-mutation','auto-changed-executable'):
            added = set(cache_after) - set(old)
            if not added:
                raise ValueError('auto changed input reused stale manifest')
            if not any(cache_after[key]['schema_inventory'] == before and
                       cache_after[key]['host_executable_sha256'] == file_sha(binary) for key in added):
                raise ValueError('new cache entry does not bind actual current schema and executable')
            if case['name'] == 'auto-schema-mutation':
                if before == baseline['schema_inventory']:
                    raise ValueError('schema mutation case did not change any schema bytes')
            elif file_sha(binary) == baseline['binary_sha256']:
                raise ValueError('changed executable case supplied the same executable bytes')
    if case['name'] in ('auto-prime','auto-reuse') and not any(
            value['schema_inventory'] == before and value['host_executable_sha256'] == file_sha(binary)
            for value in cache_after.values()):
        raise ValueError('cache case lacks an authenticated manifest for current inputs')
    save(output / 'real-authenticated.json', dict(accepted=True, binary_sha256=file_sha(binary), cache=cache_after,
                                               schema_inventory=before, campaigns=observations, external=external))
    # Dynamic identities remain in the raw authenticated receipt. Golden views retain
    # their equality/distinctness relationships and exact semantic arguments instead.
    if case['name'] == 'concurrent-identities':
        equal(json.loads((output / 'concurrent-statuses.json').read_bytes()), [0, 0], 'real.concurrent_driver_statuses')
    wrappers = {row['origin']['invocation']['wrapper_run_id'] for row in observations}
    drivers = {row['origin']['invocation']['invocation_id'] for row in observations}
    launches = {row['origin']['launch']['invocation_id'] for row in observations}
    if observations:
        equal(len(wrappers), 1, 'real.one_wrapper_identity')
        expected_drivers = 2 if case['name'] == 'concurrent-identities' else 1
        equal(len(drivers), expected_drivers, 'real.driver_isolation')
        equal(len(launches), len(observations), 'real.launch_isolation')
    return dict(case=case['name'], status=result['status'], real_compiler=True,
                campaigns=len(observations), wrapper_identities=len(wrappers), driver_identities=len(drivers),
                launch_identities=len(launches), configuration_sha256=sorted(row['configuration_sha256'] for row in observations),
                outcome_classes=sorted(item['outcome']['class'] for row in observations for item in row['outcomes']),
                argv=[value.replace(str(launcher), '$EXTERNAL_LAUNCHER') for value in argv],
                cache_entries=len(cache_after), external_launches=len(external), go_events=go_events)


def compare(first, second, golden):
    generations = []
    for root in (first, second):
        generation = {}
        for name in CASES:
            stage = root / name
            receipt = json.loads((stage / 'receipt.json').read_bytes())
            equal(receipt['accepted'], True, name + '.accepted')
            for relative, digest in receipt['raw_files'].items():
                path = stage / relative
                if not path.resolve().is_relative_to(stage.resolve()):
                    raise ValueError('raw receipt path escapes stage')
                equal(file_sha(path), digest, name + '.raw_integrity.' + relative)
            equal(file_sha(stage / 'semantic.json'), receipt['semantic_sha256'], name + '.semantic_integrity')
            if (stage / 'real-authenticated.json').exists():
                authenticated = json.loads((stage / 'real-authenticated.json').read_bytes())
                for path, retained in authenticated['cache'].items():
                    equal(file_sha(path), retained['sha256'], name + '.cache_raw_integrity')
            generation[name] = json.loads((stage / 'semantic.json').read_bytes())
        generations.append(generation)
    for name in CASES:
        equal((first / name / 'semantic.json').read_bytes(), (second / name / 'semantic.json').read_bytes(),
              name + '.byte_identical_semantic_file')
    equal(generations[0], generations[1], 'byte_identical_second_semantic_regeneration')
    if golden.exists():
        raise ValueError('golden destination must be new; never overwrite failed evidence')
    save(golden, dict(schema=CORPUS['schema'], cases=generations[0]))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--binary-sha256', required=True)
    parser.add_argument('--case', choices=CASES, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--timeout', type=int, required=True)
    parser.add_argument('--real-inputs', type=Path)
    args = parser.parse_args()
    if not args.binary.is_absolute() or args.binary.is_symlink() or not args.binary.is_file():
        raise ValueError('absolute regular immutable binary required')
    equal(file_sha(args.binary), args.binary_sha256, 'binary.exact_supplied_bytes')
    if not args.output.is_absolute() or args.timeout <= 0:
        raise ValueError('fresh absolute output and granted positive deadline required')
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    case = CASES[args.case]
    save(args.output / 'inputs.json', dict(binary=str(args.binary), binary_sha256=args.binary_sha256,
                                         corpus_sha256=file_sha(HERE / 'interception-shim-corpus.json'), case=case))
    try:
        if case['stage'] == 'real':
            if args.real_inputs is None:
                raise ValueError('real case requires reviewed --real-inputs; no substitute fixture')
            raw_inputs = args.real_inputs.read_bytes()
            (args.output / 'real-inputs.json').write_bytes(raw_inputs)
            inputs = json.loads(raw_inputs)
            equal(inputs['binary_sha256'], args.binary_sha256, 'real.accepted_binary_binding')
            semantic = real_case(case, args.binary, args.output, args.timeout, inputs)
        else:
            if args.real_inputs is not None:
                raise ValueError('synthetic routing/refusal case cannot claim real admission')
            semantic = fake_case(case, args.binary, args.output, args.timeout)
        semantic['binary_sha256'] = args.binary_sha256
        semantic['corpus_sha256'] = file_sha(HERE / 'interception-shim-corpus.json')
        equal(file_sha(args.binary), args.binary_sha256, 'binary.unchanged_after_stage')
        save(args.output / 'semantic.json', semantic)
        raw_files = {str(path.relative_to(args.output)): file_sha(path) for path in args.output.rglob('*')
                     if path.is_file() and not path.is_symlink()}
        save(args.output / 'receipt.json', dict(accepted=True, semantic_sha256=file_sha(args.output / 'semantic.json'),
                                               raw_files=raw_files))
    except BaseException as error:
        save(args.output / 'receipt.json', dict(accepted=False, error=str(error)))
        raise


if __name__ == '__main__':
    if len(sys.argv) > 1 and sys.argv[1] == '--fixture':
        raise SystemExit(fixture())
    if len(sys.argv) > 1 and sys.argv[1] == '--compare':
        if len(sys.argv) != 5:
            raise SystemExit('usage: interception-shim-check.py --compare FIRST SECOND NEW_GOLDEN')
        compare(*[Path(value) for value in sys.argv[2:]])
    else:
        main()
