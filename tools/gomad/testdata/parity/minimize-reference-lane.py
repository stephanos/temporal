#!/usr/bin/env python3
"""Compare fresh public Go/Rust minimizers on identical cooperative failures."""
import argparse
import base64
import fcntl
import hashlib
import json
import os
from pathlib import Path
import runpy
import shutil
import subprocess
import sys
import tarfile

sys.dont_write_bytecode = True


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f'{label}: got {actual!r}, expected {expected!r}')


def reductions(row):
    return [{'kind': reduction['kind'], 'removed': [
        {'dimension': item['dimension'], 'ordinal': int(item['ordinal']),
         'identity': item['identity'].removeprefix('sha256:')}
        for item in reduction['removed']]} for reduction in row['accepted']]


def rust_case(artifact, module_path):
    manifest = json.loads((artifact / 'manifest.json').read_bytes())
    for entry in manifest['files']:
        raw = (artifact / entry['path']).read_bytes()
        equal((sha(raw), len(raw)), (entry['sha256'].removeprefix('sha256:'), int(entry['size'])), entry['path'])
    provenance = json.loads((artifact / manifest['target']['backend']['provenance']['file']).read_bytes())
    config = provenance['Config']
    equal((config['CapturedInputs']['Descriptor'], config['CapturedInputs']['Payloads']), (None, None), 'captured inputs')
    equal((config.get('storage'), config.get('network_v2')), (None, None), 'model scope')
    module = (artifact / manifest['target']['file']).read_bytes()
    module_path.write_bytes(module)
    module_path.chmod(0o600)
    implementation = manifest['choice_profile']['implementation_sha256'].removeprefix('sha256:')
    choice_limit = int(manifest['choice_profile']['trace']['limit'])
    limits = config['Limits']
    environment = {entry['name']: entry['value'] for entry in manifest['environment']}
    environment['PWD'] = config['WorkingDirectory']
    return {'name': 'seed' + manifest['seed'], 'seed': int(manifest['seed']),
            'module_path': str(module_path), 'module_sha256': sha(module),
            'toolchain_build_key': provenance['BuildKey'],
            'choice_identity': {'TargetSHA256': list(bytes.fromhex(sha(module))),
                                'ToolchainBuildKey': provenance['BuildKey'], 'GOOS': 'wasip1', 'GOARCH': 'wasm',
                                'ImplementationSHA256': list(bytes.fromhex(implementation))},
            'configuration': {'engine': 'cranelift-fuel-nan-canonical-nounwind-epoch-v1',
                'profile': config['Profile'], 'runtime_sha256': implementation,
                'limits': {'fuel': provenance['Fuel'], 'memory_bytes': provenance['MemoryBytes'],
                           'wall_millis': int(manifest['limits']['execution_timeout_nanos']) // 1000000,
                           'output_bytes': limits['OutputBytes'],
                           'evidence_bytes': limits['TranscriptBytes'] + choice_limit + 2 * limits['OutputBytes'] + (16 << 20)},
                'arguments': manifest['target']['argv'], 'environment': environment,
                'model': {'entropy_key': config['EntropyKey'], 'mode': 1, 'limit': choice_limit,
                          'diagnostics': False, 'epoch_nanos': config['Clock']['EpochNanos'],
                          'read_step_nanos': config['Clock']['ReadStepNanos'],
                          'now': 1 if config['Clock']['ReadStepNanos'] == 0 else 0,
                          'working_directory_sha256': sha(config['WorkingDirectory'].encode()),
                          'arguments': {'bytes': 0, 'count': 0, 'chunks': 0},
                          'environment': {'bytes': 0, 'count': 0, 'chunks': 0},
                          'limits': {'calls': limits['Calls'], 'transcript_bytes': limits['TranscriptBytes'],
                                     'pending_events': limits['PendingEvents']}},
                'namespace': {'entries': [], 'writable_directories': [list(path.encode()) for path in config['WritableDirectories']],
                              'preopens': [list(path.encode()) for path in sorted(set(config['WritableDirectories']))],
                              'working_directory': list(config['WorkingDirectory'].encode()),
                              'stdin': list(base64.b64decode(config['Stdin'] or '', validate=True)),
                              'limits': {'files': limits['Files'], 'descriptors': limits['Descriptors'],
                                         'filesystem_bytes': limits['FilesystemBytes'], 'output_bytes': limits['OutputBytes']},
                              'epoch_nanos': config['Clock']['EpochNanos']}}}


def changed_rust_signature(source, destination):
    shutil.copytree(source, destination)
    manifest = json.loads((destination / 'manifest.json').read_bytes())

    def payload(name, raw):
        (destination / name).write_bytes(raw)
        for entry in (manifest['files'][name], manifest['record']['streams'][name]):
            entry.update(bytes=len(raw), sha256=sha(raw))

    raw = bytearray((destination / 'output/stdout').read_bytes())
    equal(bool(raw), True, 'signature fixture nonempty stdout')
    raw[0] ^= 1
    payload('output/stdout', raw)
    details = json.loads((destination / 'evidence/inspection.json').read_bytes())
    if details.get('outputs'):
        details['outputs']['stdout']['full_sha256'] = sha(raw)
        payload('evidence/inspection.json', canonical(details))
    raw = canonical(manifest['record'])
    (destination / 'record.json').write_bytes(raw)
    manifest['record_sha256'] = sha(raw)
    manifest['files']['record.json'].update(bytes=len(raw), sha256=sha(raw))
    (destination / 'manifest.json').write_bytes(canonical(manifest))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('baseline', 'source', 'go-compiler', 'rust-binary', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--go-cache', type=Path)
    parser.add_argument('--preparation-root', type=Path)
    parser.add_argument('--golden', type=Path, help='reviewed semantic projection; regeneration requires GOMAD_GOLDEN_REGENERATE=1')
    args = parser.parse_args()
    for name in ('baseline', 'source', 'go_compiler', 'rust_binary', 'output'):
        setattr(args, name, getattr(args, name).resolve())
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    corpus = Path(__file__).resolve().parent
    inventory = (args.baseline / 'inputs.sha256').read_bytes()
    for line in inventory.decode().splitlines():
        expected, relative = line.split()
        equal(sha((args.baseline / relative).read_bytes()), expected, relative)
    equal(sha(args.go_compiler.read_bytes()), '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64',
          'frozen Go compiler identity')
    with tarfile.open(args.baseline / 'source-1d7b37f00a.tar.gz') as archive:
        for member in archive.getmembers():
            parts = Path(member.name).parts
            if member.isfile() and parts[1:3] == ('tools', 'gomad'):
                equal(sha((args.source / Path(*parts[1:])).read_bytes()), sha(archive.extractfile(member).read()), member.name)
    environment = {key: value for key, value in os.environ.items()
                   if key not in ('GOROOT', 'GOFLAGS', 'GOMOD', 'GOPATH', 'GOEXPERIMENT') and not key.startswith('GOMAD_')}
    environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='0', GOPROXY='off', GOSUMDB='off',
                       GOCACHE=str((args.go_cache or args.output / 'go-cache').resolve()))
    commands = []

    def run(name, command, status=0, stdin=None, cwd=None):
        result = subprocess.run(list(map(str, command)), input=stdin, capture_output=True,
                                cwd=cwd, env=environment, timeout=600)
        (args.output / (name + '.stdout')).write_bytes(result.stdout)
        (args.output / (name + '.stderr')).write_bytes(result.stderr)
        commands.append({'name': name, 'command': list(map(str, command)), 'status': result.returncode,
                         'expected_status': status, 'stdout_sha256': sha(result.stdout), 'stderr_sha256': sha(result.stderr)})
        (args.output / 'commands.json').write_bytes(canonical(commands))
        equal(result.returncode, status, name + '.status')
        return result.stdout

    go = args.baseline / 'bin/gomad'
    binary_hash = sha(args.rust_binary.read_bytes())
    for name, flags in [('json-bool', ['--json=maybe', '/failure']), ('byte-zero', ['--max-bytes=0', '/failure']),
                        ('unknown-flag', ['--unknown', '/failure']), ('missing-artifact', [])]:
        for lineage, binary in [('go', go), ('rust', args.rust_binary)]:
            run('parser-' + name + '-' + lineage, [binary, 'minimize', *flags], 2)
    module = args.source / 'tools/gomad'
    wrapper = module / ('internal/runner/fn232-minimize-reference-' + sha(str(args.output).encode())[:12])
    wrapper.mkdir(mode=0o700)
    template = corpus / 'reference/minimize.go.txt'
    (wrapper / 'main.go').write_bytes(template.read_bytes())
    executable = args.output / 'go-minimize-reference'
    try:
        run('go-build', [args.go_compiler, 'build', '-trimpath', '-o', executable,
                         './' + str(wrapper.relative_to(module))], cwd=module)
    finally:
        shutil.rmtree(wrapper)
    config = {'execution_sha256': 'sha256:' + 'a' * 64, 'controller_sha256': '', 'base_seed': 7,
              'parallel': 1, 'max_executions': 100, 'max_forced_decisions': 10,
              'max_exploration_bytes': 100000, 'max_result_bytes': 100000, 'failure_budget': 10,
              'dimension_limits': {name: 10 for name in ('runtime', 'scenario', 'network', 'storage', 'fault', 'crash')}}
    overrides = [{'dimension': dimension, 'ordinal': index, 'site_sha256': 'sha256:' + 'b' * 64,
                  'alternatives': 2, 'alternative_set_sha256': 'sha256:' + 'c' * 64, 'selected': 1,
                  'selected_sha256': 'sha256:' + 'd' * 64, 'identity': '',
                  **({'control': 'AQ=='} if dimension == 'runtime' else {})}
                 for index, dimension in enumerate(('runtime', 'runtime', 'scenario', 'network', 'storage', 'fault', 'crash'))]

    def controller_view(row):
        def reduction(value):
            return {**reductions({'accepted': [value]})[0],
                    'before_sha256': value['before_sha256'].removeprefix('sha256:'),
                    'after_sha256': value['after_sha256'].removeprefix('sha256:')}
        return {'proposals': [{'index': value['index'], 'reduction': reduction(value['reduction']),
                               'candidate_sha256': value['candidate']['sha256'].removeprefix('sha256:')}
                              for value in row['proposals']],
                'accepted': [reduction(value) for value in row['accepted']],
                'attempts': row['attempts'], 'stop_reason': row['stop_reason']}

    controllers = []
    for name, accepted, budget, decisions in [('reject-all', [], 100, overrides), ('accept-first', [0], 100, overrides),
                                             ('accept-after-reject', [3, 7], 100, overrides), ('budget', [0], 1, overrides),
                                             ('empty', [], 4, [])]:
        request = {'Operation': 'proposals', 'config': config, 'original': {'overrides': decisions},
                   'attempt_budget': budget, 'accepted_indices': accepted}
        expected = None
        for repeat in range(2):
            prefix = f'controller-{name}-{repeat}'
            go_row = json.loads(run(prefix + '-go', [executable], stdin=canonical(request)))
            rust_input = {key: value for key, value in request.items() if key != 'Operation'}
            rust_input.update(config=go_row['config'], original=go_row['original'])
            path = args.output / (prefix + '.json')
            path.write_bytes(canonical(rust_input))
            rust_row = json.loads(run(prefix + '-rust', [args.rust_binary, 'fixture-minimize-controller', '--input', path]))
            actual = controller_view(rust_row)
            equal(actual, controller_view(go_row), prefix + '.controller parity')
            if expected is not None:
                equal(actual, expected, prefix + '.repeated controller')
            expected = actual
        controllers.append({'name': name, **expected})
    scoped_controllers = []
    runtime_request = {'Operation': 'canonical', 'config': config, 'original': {'overrides': [
        dict(overrides[0], ordinal=index) for index in range(3)]}}
    runtime = json.loads(run('scoped-controller-canonical', [executable], stdin=canonical(runtime_request)))
    decision = lambda forced: {key: forced[key] for key in ('dimension', 'ordinal', 'identity')}
    runtime_decisions = [decision(forced) for forced in runtime['original']['overrides']]
    predicate = [runtime_decisions[0], runtime_decisions[2]]
    for name, inside, attempts in [('unscoped', None, 2), ('same-minimum', [0, 2], 3), ('distinct-minimum', [1, 2], 4)]:
        repeated = []
        for repeat in range(2):
            request = dict(runtime, attempt_budget=100, accepted_indices=[], accept_when_remaining_any=predicate)
            if inside is not None:
                request['ordering'] = dict(key={'algorithm': 'fixture/site-ordering/v1'},
                    in_scope=[runtime_decisions[index] for index in inside])
            path = args.output / f'scoped-controller-{name}-{repeat}.json'
            path.write_bytes(canonical(request))
            row = json.loads(run(f'scoped-controller-{name}-{repeat}', [args.rust_binary, 'fixture-minimize-controller', '--input', path]))
            equal(row['attempts'], attempts, name + '.content predicate attempt count')
            final_sha = row['accepted'][-1]['after_sha256']
            final = next(proposal['candidate'] for proposal in row['proposals'] if proposal['candidate']['sha256'] == final_sha)
            equal(any(decision(forced) in predicate for forced in final['overrides']), True, name + '.same content predicate')
            repeated.append(controller_view(row))
        equal(repeated[0], repeated[1], name + '.repeated scoped controller')
        scoped_controllers.append(dict(name=name, **repeated[0]))
    run('controller-nil-go-refusal', [executable], 3, stdin=canonical({'Operation': 'proposals', 'config': config,
        'original': {'overrides': None}, 'attempt_budget': 4, 'accepted_indices': []}))
    nil = json.loads(run('controller-nil-canonical', [executable], stdin=canonical({'Operation': 'canonical', 'config': config,
                       'original': {'overrides': None}})))
    nil.update(attempt_budget=4, accepted_indices=[])
    nil_path = args.output / 'controller-nil.json'
    nil_path.write_bytes(canonical(nil))
    installation = args.output / 'go-installation.json'
    run('go-install', [go, 'install', '--compiler', args.go_compiler, '--helper', args.baseline / 'bin/gomad-wasmhost',
                      '--profile', 'gomad.wasi-cooperative/v1', '--runtime-root', module / 'internal/model/runtimesource',
                      '--cache', (args.preparation_root or args.output) / 'minimize-preparation', '--output', installation])
    corpus_sources = (args.preparation_root or args.output) / 'minimize-corpus-sources'
    guest = corpus_sources / 'constant-failure'
    guest.mkdir(mode=0o700, parents=True, exist_ok=True)
    (guest / 'go.mod').write_text('module gomad.minimize-fixture\n\ngo 1.27.1\n')
    (guest / 'failure_test.go').write_text('''package minimizefixture
import ("runtime"; "testing")
func TestFailure(t *testing.T) {
    done := make(chan struct{}, 2)
    for range 2 { go func() { runtime.Gosched(); done <- struct{}{} }() }
    <-done
    <-done
    t.Fatal("deterministic minimizer failure")
}
''')
    run('go-explore', [go, '--installation', installation, 'explore', '--json', '--choices', '--seeds=7,8', '--parallel=1',
                       '--on-failure=all', '--working-dir', guest,
                       '--build-tag=gomad_fixture', '--build-tag=test_dep', '--execution-timeout=30s', '--overall-timeout=5m',
                       '--artifacts', args.output / 'go-parents', 'go-test', '.', '--', '-test.run=^TestFailure$', '-test.v'], 1)
    parents = sorted((path.parent for path in (args.output / 'go-parents').rglob('manifest.json')
                      if json.loads(path.read_bytes()).get('artifact_kind') == 'gomad.target-failure/v1'),
                     key=lambda path: int(json.loads((path / 'manifest.json').read_bytes())['seed']))
    equal(len(parents), 2, 'fresh Go parents')
    witness = runpy.run_path(str(corpus / 'minimize-counterfactual-probe.py'))['prepare'](
        go, installation, args.output / 'counterfactual', run, corpus_sources / 'counterfactual')
    parents.append(Path(witness['artifact']))
    cases = [rust_case(parent, args.output / f'seed{index}.wasm') for index, parent in enumerate(parents)]
    cases[-1].update(name=f'counterfactual-seed{witness["seed"]}',
                     forced_prefix={'trace_path': witness['trace_path'], 'cutoff': witness['decisions']})
    baseline = Path(witness['baseline'])
    equal(json.loads((baseline / 'manifest.json').read_bytes())['artifact_kind'],
          'gomad.success/v1', 'Go counterfactual default succeeds')
    baseline_case = rust_case(baseline, args.output / 'counterfactual-default.wasm')
    baseline_case['name'] = f'counterfactual-default-seed{witness["seed"]}'
    equal((baseline_case['seed'], baseline_case['module_sha256'], baseline_case['configuration']),
          (cases[-1]['seed'], cases[-1]['module_sha256'], cases[-1]['configuration']),
          'counterfactual default and forced executions have identical base inputs')
    input_path = args.output / 'rust-input.json'
    input_path.write_bytes(canonical({'schema': 'gomad.replay-fixture-input/v1', 'cases': [*cases, baseline_case]}))
    rust_root = args.output / 'rust-parents'
    bundle = json.loads(run('rust-produce', [args.rust_binary, 'fixture-replay-input', '--root', rust_root, '--input', input_path]))
    equal(len(bundle['records']), len(parents) + 1, 'fresh Rust baseline and parents')
    baseline_record = json.loads((rust_root / bundle['records'][-1]['artifact'] / 'record.json').read_bytes())
    equal(baseline_record['outcome']['class'], 'success', 'Rust counterfactual default succeeds')
    parent_records = bundle['records'][:-1]
    cluster_work = args.output / 'cluster-refusal'
    cluster_driver = corpus / 'minimize-cluster-refusal.py'
    run('cluster-refusal', [sys.executable, cluster_driver, '--binary', args.rust_binary,
        '--baseline', args.baseline, '--source', corpus.parents[3], '--go-compiler', args.go_compiler,
        '--work', cluster_work, '--input', input_path, '--guest', guest,
        '--preparation-root', (args.preparation_root or args.output) / 'cluster-refusal-preparation'])
    cluster_report = json.loads((cluster_work / 'minimize-cluster-refusal-result.json').read_bytes())
    equal(cluster_report['schema'], 'gomad.minimize-cluster-refusal/v1', 'cluster refusal report schema')
    equal(cluster_report['driver_sha256'], sha(cluster_driver.read_bytes()), 'cluster refusal driver identity')
    equal(cluster_report['rust_binary_sha256'], binary_hash, 'cluster refusal Rust binary identity')
    cluster_projection = []
    for row in cluster_report['results']:
        lineage = row['lineage']
        replay = (cluster_work / (lineage + '-replay.stdout')).read_bytes()
        projection = {'lineage': lineage, 'replay_status': row['replay_status'],
                      'reproduced': b'reproduced=true' in replay, 'minimize_status': row['minimize_status']}
        equal(projection, {'lineage': lineage, 'replay_status': 1, 'reproduced': True, 'minimize_status': 3},
              lineage + '.cluster exact replay and minimization refusal')
        cluster_projection.append(projection)
    equal([row['lineage'] for row in cluster_projection], ['go', 'rust'], 'cluster refusal lineages')
    shared_bound = max(len((rust_root / entry['artifact'] / 'manifest.json').read_bytes()) +
                       sum(item['bytes'] for item in json.loads((rust_root / entry['artifact'] / 'manifest.json').read_bytes())['files'].values())
                       for entry in parent_records) + (1 << 20)
    comparisons, signatures, repeat_records, scoped_reports = [], [], [], []
    for index, (parent, entry) in enumerate(zip(parents, parent_records)):
        kind = 'counterfactual' if index == 2 else 'constant-failure'
        rust_parent = rust_root / entry['artifact']
        go_manifest = json.loads((parent / 'manifest.json').read_bytes())
        rust_record = json.loads((rust_parent / 'record.json').read_bytes())
        equal(rust_record['outcome']['class'], 'target_failure', 'Rust parent outcome')
        equal((rust_parent / 'evidence/choice').read_bytes(),
              (parent / go_manifest['choice_profile']['trace']['file']).read_bytes(), 'identical original choices')
        shared = json.loads(run(f'{index}-shared-output', [args.rust_binary, 'minimize', '--json', '--max-bytes', shared_bound,
                                '--artifacts', args.output / 'shared-output', rust_parent]))
        equal(shared['changed'], True, 'shared output accepts each independently bounded artifact')
        projection = dict(rust_record, seed=0)
        signatures.append((go_manifest['outcome']['failure_signature'], sha(canonical(projection))))
        pair, raw_records = [], []
        for repeat in range(2):
            prefix = f'{index}-{repeat}'
            go_result = json.loads(run(prefix + '-go-minimize', [go, '--installation', installation, 'minimize', '--json',
                                  '--artifacts', args.output / (prefix + '-go'), parent]))
            rust_output = args.output / (prefix + '-rust')
            rust_result = json.loads(run(prefix + '-rust-minimize', [args.rust_binary, 'minimize', '--json',
                                    '--artifacts', rust_output, rust_parent]))
            minimized = Path(rust_result['artifact_path'])
            equal(sha((minimized / 'record.json').read_bytes()), rust_result['record_hash'], prefix + '.published record hash')
            equal((minimized / 'minimization/original-record.json').read_bytes(), (rust_parent / 'record.json').read_bytes(),
                  prefix + '.immutable original predicate')
            metadata = json.loads((minimized / 'evidence/inspection.json').read_bytes())['minimization']
            equal(metadata['predicate']['failure_signature'], signatures[-1][1], prefix + '.predicate signature')
            equal(metadata['parent_record_hash'], entry['record_sha256'], prefix + '.parent record identity')
            inspected = json.loads(run(prefix + '-rust-inspect',
                                       [args.rust_binary, 'inspect', '--json', minimized]))['artifact']
            equal('site_scope' in rust_result, False, prefix + '.legacy absence remains unscoped')
            for reduction in rust_result['accepted']:
                equal((reduction['removed_in_scope'], reduction['removed_out_of_scope']),
                      (0, len(reduction['removed'])), prefix + '.unscoped removal counts')
            manifest = json.loads((minimized / 'manifest.json').read_bytes())
            for name in ('minimization/source-choice.bin', 'minimization/source-choice.json'):
                raw = (minimized / name).read_bytes()
                equal((len(raw), sha(raw)), (manifest['files'][name]['bytes'], manifest['files'][name]['sha256']), prefix + '.' + name)
                equal(name in manifest['record']['streams'], False, prefix + '.source provenance is manifest-only')
            equal((minimized / 'minimization/source-choice.bin').read_bytes(),
                  (rust_parent / 'evidence/choice').read_bytes(), prefix + '.source is original parent tape')
            equal(sum(group['count'] for group in inspected['minimization']['final_forced_by_symbol']),
                  metadata['final_forced_decisions'], prefix + '.symbol group count')
            equal(all('symbol' in decision for reduction in inspected['minimization']['accepted']
                      for decision in reduction['removed']), True, prefix + '.removed decisions have symbol reports')
            equal(json.loads((minimized / 'evidence/inspection.json').read_bytes())['minimization'], metadata,
                  prefix + '.inspection leaves raw evidence unchanged')
            detached = args.output / (prefix + '-detached-minimum')
            shutil.copytree(minimized, detached)
            detached_report = json.loads(run(prefix + '-rust-detached-inspect',
                [args.rust_binary, 'inspect', '--json', detached]))['artifact']['minimization']
            equal(detached_report, inspected['minimization'], prefix + '.detached symbol report')
            explicit_output = args.output / (prefix + '-rust-explicit-scope')
            explicit = json.loads(run(prefix + '-rust-explicit-scope', [args.rust_binary, 'minimize', '--json',
                '--site-scope=runtime/...', '--artifacts', explicit_output, rust_parent]))
            equal({'changed': explicit['changed'], 'attempts': int(explicit['attempts']),
                  'stop_reason': explicit['stop_reason'], 'accepted': reductions(explicit)},
                  {'changed': rust_result['changed'], 'attempts': int(rust_result['attempts']),
                   'stop_reason': rust_result['stop_reason'], 'accepted': reductions(rust_result)}, prefix + '.public prefix order exception')
            equal(explicit['site_scope']['patterns'], ['runtime/...'], prefix + '.explicit scope report')
            equal(explicit['record_hash'], rust_result['record_hash'], prefix + '.scope reporting leaves fixed prefix record unchanged')
            for reduction in explicit['accepted']:
                equal(reduction['removed_in_scope'] + reduction['removed_out_of_scope'], len(reduction['removed']), prefix + '.scoped removal counts')
            scoped_reports.append(dict(kind=kind, seed=rust_record['seed'], site_scope=explicit['site_scope'],
                accepted=[dict(reduction, removed_in_scope=raw['removed_in_scope'], removed_out_of_scope=raw['removed_out_of_scope'])
                          for reduction, raw in zip(reductions(explicit), explicit['accepted'])],
                attempts=explicit['attempts'], final_forced_by_symbol=inspected['minimization']['final_forced_by_symbol']))
            run(prefix + '-rust-scope-resume-refusal', [args.rust_binary, 'minimize', '--resume', '--site-scope=testing',
                '--artifacts', explicit_output, rust_parent], 3)
            full_signature = sha(canonical(dict(json.loads((minimized / 'record.json').read_bytes()), seed=0)))
            equal(inspected['failure_signature'], full_signature, prefix + '.independent decorated failure signature')
            equal(full_signature != metadata['predicate']['failure_signature'], True,
                  prefix + '.decoration changes full signature without changing immutable predicate')
            go_path = Path(go_result['artifact_path'])
            go_minimized = json.loads((go_path / 'manifest.json').read_bytes())
            go_payloads = {item['path']: (go_path / item['path']).read_bytes() for item in go_minimized['files']}
            for item in go_minimized['files']:
                equal((len(go_payloads[item['path']]), sha(go_payloads[item['path']])),
                      (int(item['size']), item['sha256'].removeprefix('sha256:')), prefix + '.Go retained payload')
            raw_records.append({'go_record_hash': go_result['record_hash'], 'go_payloads': go_payloads,
                                'rust': (minimized / 'record.json').read_bytes()})
            view = lambda row: {'changed': row['changed'], 'attempts': int(row['attempts']),
                                'stop_reason': row['stop_reason'], 'accepted': reductions(row)}
            equal(view(rust_result), view(go_result), prefix + '.semantic parity')
            equal(bool(reductions(go_result)), True, prefix + '.nonvacuous reductions')
            if kind == 'counterfactual':
                equal(rust_result['changed'], True, prefix + '.counterfactual reduction changes artifact')
                equal(sum(len(item['removed']) for item in rust_result['accepted']) > 0,
                      True, prefix + '.counterfactual removes actual controls')
                equal(metadata['final_forced_decisions'] > 0, True,
                      prefix + '.counterfactual retains controls required for failure')
            for label, binary, flags, result in [('go', go, ['--installation', installation], go_result),
                                                ('rust', args.rust_binary, [], rust_result)]:
                replay = run(prefix + '-' + label + '-replay', [binary, *flags, 'replay', result['artifact_path']], 1)
                equal(b'reproduced=true' in replay and b'choice-replay=exact' in replay, True, prefix + '.exact replay')
            equal(json.loads(run(prefix + '-rust-resume', [args.rust_binary, 'minimize', '--json', '--resume',
                             '--artifacts', rust_output, rust_parent])), rust_result, prefix + '.resume')
            pair.append(view(rust_result))
            if index == 0 and repeat == 0:
                capped_output = args.output / 'fresh-capped-output'
                run('fresh-byte-bound', [args.rust_binary, 'minimize', '--max-bytes=1', '--artifacts', capped_output, rust_parent], 3)
                equal([path.name for path in (capped_output / 'minimized').iterdir()
                       if not path.name.startswith('.') and (path / 'manifest.json').exists()], [],
                      'byte-bound failure publishes no final artifact')
                run('changed-bounds', [args.rust_binary, 'minimize', '--resume', '--attempt-budget=65', '--artifacts', rust_output, rust_parent], 3)
                run('changed-byte-bound', [args.rust_binary, 'minimize', '--resume', '--max-bytes=1', '--artifacts', rust_output, rust_parent], 3)
                run('existing-workspace', [args.rust_binary, 'minimize', '--artifacts', rust_output, rust_parent], 3)
                checkpoint = next(rust_output.rglob('checkpoint.json'))
                with (checkpoint.parent / '.lock').open('rb') as lock:
                    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    run('concurrent-writer', [args.rust_binary, 'minimize', '--resume', '--artifacts', rust_output, rust_parent], 3)
                for mutation in ('corrupt', 'implementation', 'parent', 'predicate', 'state-budget'):
                    target = args.output / ('refusal-' + mutation)
                    shutil.copytree(rust_output, target)
                    path = next(target.rglob('checkpoint.json'))
                    state = json.loads(path.read_bytes())
                    if mutation == 'corrupt':
                        path.write_bytes(b'{')
                    else:
                        if mutation == 'state-budget':
                            state['state']['attempt_budget'] += 1
                            state['published'] = None
                            state['complete'] = False
                            state['state']['sha256'] = 'sha256:' + sha(b'gomad-rust-minimizer-state/v1\0' +
                                                                     canonical(dict(state['state'], sha256='')))
                        elif mutation == 'predicate':
                            state['binding']['original_predicate_identity']['outcome']['reason'] += '-changed'
                        else:
                            field = 'implementation_sha256' if mutation == 'implementation' else 'parent_record_hash'
                            state['binding'][field] = 'f' * 64
                        state['sha256'] = sha(canonical({key: value for key, value in state.items() if key != 'sha256'}))
                        path.write_bytes(canonical(state))
                    run(mutation + '-refusal', [args.rust_binary, 'minimize', '--resume', '--artifacts', target, rust_parent], 3)
                corrupted = args.output / 'refusal-accepted-payload'
                shutil.copytree(rust_output, corrupted)
                path = next(corrupted.rglob('checkpoint.json'))
                state = json.loads(path.read_bytes())
                payload = path.parent / 'accepted' / state['accepted']['directory'] / 'output/stdout'
                raw = bytearray(payload.read_bytes())
                equal(bool(raw), True, 'accepted artifact has output payload')
                raw[0] ^= 1
                payload.write_bytes(raw)
                run('corrupt-accepted-payload-refusal', [args.rust_binary, 'minimize', '--resume',
                     '--artifacts', corrupted, rust_parent], 3)
                substituted = args.output / 'refusal-publication'
                shutil.copytree(rust_output, substituted)
                path = next(substituted.rglob('checkpoint.json'))
                state = json.loads(path.read_bytes())
                manifest = json.loads((rust_parent / 'manifest.json').read_bytes())
                shutil.copytree(rust_parent, substituted / 'minimized/unrelated')
                state['published'] = {'directory': 'unrelated', 'record_hash': manifest['record_sha256']}
                state['sha256'] = sha(canonical({key: value for key, value in state.items() if key != 'sha256'}))
                path.write_bytes(canonical(state))
                run('substituted-publication-refusal', [args.rust_binary, 'minimize', '--resume',
                     '--artifacts', substituted, rust_parent], 3)
        equal(pair[0], pair[1], f'{index}.repeated semantics')
        equal(raw_records[0], raw_records[1], f'{index}.byte-identical repeated Go/Rust records')
        repeat_records.append({'kind': kind, 'seed': rust_record['seed'], 'go_record_hash': raw_records[0]['go_record_hash'],
                               'go_payloads': {name: sha(raw) for name, raw in raw_records[0]['go_payloads'].items()},
                               'rust_record_sha256': sha(raw_records[0]['rust'])})
        comparisons.append({'kind': kind, 'seed': rust_record['seed'], **pair[0]})
        if index == 0:
            reference = json.loads(run('go-runner-api', [executable], stdin=canonical({'Operation': 'minimize',
                                  'Installation': str(installation), 'Artifact': str(parent),
                                  'Output': str(args.output / 'go-runner-api'), 'attempt_budget': 64})))
            equal(view(reference), pair[0], 'actual runner.Minimize/public CLI agreement')
            altered = json.loads(run('go-change-signature', [executable], stdin=canonical({'Operation': 'changed-signature',
                                 'Artifact': str(parent), 'Output': str(args.output / 'go-altered')})))
            equal(altered['failure_signature'] != go_manifest['outcome']['failure_signature'], True, 'Go changed signature')
            rust_altered = args.output / 'rust-altered'
            changed_rust_signature(rust_parent, rust_altered)
            altered_record = json.loads((rust_altered / 'record.json').read_bytes())
            equal(sha(canonical(dict(altered_record, seed=0))) != signatures[-1][1], True, 'Rust changed signature')
            for label, binary, flags, changed in [('go', go, ['--installation', installation], Path(altered['artifact'])),
                                                 ('rust', args.rust_binary, [], rust_altered)]:
                run(label + '-altered-inspect', [binary, 'inspect', '--json', changed])
                row = json.loads(run(label + '-signature-rejection', [binary, *flags, 'minimize', '--json', '--attempt-budget=1',
                                     '--artifacts', args.output / (label + '-signature-rejection'), changed]))
                equal((row['changed'], int(row['attempts']), reductions(row)), (False, 1, []), label + '.signature rejection')
    shared_sizes = []
    for path in (args.output / 'shared-output/minimized').iterdir():
        if not path.name.startswith('.') and (path / 'manifest.json').exists():
            raw = (path / 'manifest.json').read_bytes()
            shared_sizes.append(len(raw) + sum(item['bytes'] for item in json.loads(raw)['files'].values()))
    equal(len(shared_sizes), 3, 'shared output retains all three parent results')
    equal(all(size <= shared_bound for size in shared_sizes) and sum(shared_sizes) > shared_bound, True,
          'byte bound applies per artifact rather than to aggregate output')
    failure_groups = []
    for left in range(len(signatures)):
        for right in range(left + 1, len(signatures)):
            same = signatures[left][0] == signatures[right][0]
            equal(same, signatures[left][1] == signatures[right][1], f'{left}/{right}.failure-group parity')
            failure_groups.append({'left': left, 'right': right, 'same': same})
    run('controller-nil-rust-refusal', [args.rust_binary, 'fixture-minimize-controller', '--input', nil_path], 2)
    equal(sha(args.rust_binary.read_bytes()), binary_hash, 'Rust binary unchanged')
    report = {'schema': 'gomad.minimize-reference-parity/v1', 'driver_sha256': sha(Path(__file__).read_bytes()),
              'baseline_inputs_sha256': sha(inventory), 'go_binary_sha256': sha(go.read_bytes()),
              'go_compiler_sha256': sha(args.go_compiler.read_bytes()), 'go_wrapper_sha256': sha(executable.read_bytes()),
              'template_sha256': sha(template.read_bytes()), 'rust_binary_sha256': binary_hash,
              'counterfactual_driver_sha256': sha((corpus / 'minimize-counterfactual-probe.py').read_bytes()),
              'cluster_driver_sha256': cluster_report['driver_sha256'], 'cluster_refusals': cluster_projection,
              'input_sha256': sha(input_path.read_bytes()), 'comparisons': comparisons,
              'counterfactual_witness': witness, 'failure_groups': failure_groups,
              'controllers': controllers, 'scoped_controllers': scoped_controllers, 'scoped_reports': scoped_reports,
              'byte_identical_repeated_records': repeat_records,
              'go_nil_controller_candidate': 'frozen Go refuses nil overrides after canonical clone; excluded from admitted parity cases',
              'nil_controller_wrapper_status': {'go': 3, 'rust': 2},
              'parser_refusals': ['json-bool', 'byte-zero', 'unknown-flag', 'missing-artifact'],
              'scope': 'standalone cooperative choice-prefix; simulation reductions require separate corpus',
              'refusals': ['fresh-byte-bound', 'changed-bounds', 'changed-byte-bound', 'corrupt', 'concurrent-writer',
                           'implementation', 'parent', 'predicate', 'state-budget', 'corrupt-accepted-payload', 'substituted-publication'],
              'signature_changing_candidate': 'rejected by both fresh executions'}
    if args.golden:
        raw = canonical({'comparisons': comparisons, 'controllers': controllers,
                         'scoped_controllers': scoped_controllers, 'scoped_reports': scoped_reports,
                         'failure_groups': failure_groups,
                         'cluster_refusals': cluster_projection,
                         'refusals': [{'name': row['name'], 'status': row['status']} for row in commands
                                      if row['expected_status'] in (2, 3)]}) + b'\n'
        if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
            args.golden.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            args.golden.write_bytes(raw)
        else:
            equal(args.golden.read_bytes(), raw, 'reviewed minimizer semantic golden')
    (args.output / 'minimize-reference-result.json').write_bytes(canonical(report))
    print('fresh Go/Rust minimizer parity: five controller cases, three failures including a default-success counterfactual, repeated removed-decision agreement, exact replay and refusals')


if __name__ == '__main__':
    main()
