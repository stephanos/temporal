#!/usr/bin/env python3
"""Authenticate public preparation plans without counting retention as execution."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import runpy
import shutil
import subprocess


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def read(path):
    raw = path.read_bytes()
    value = json.loads(raw)
    if json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode() != raw:
        raise ValueError('noncanonical JSON: ' + str(path))
    return value, raw


def check_resources(receipt):
    start, end = receipt['preparation_start'], receipt['preparation_end']
    cpu = None
    if start is not None and end is not None:
        for owner in ('linked', 'children'):
            for field in ('user_nanos', 'system_nanos', 'peak_rss_bytes'):
                assert end[owner][field] >= start[owner][field]
        cpu = sum(end['children'][field] - start['children'][field]
                  for field in ('user_nanos', 'system_nanos'))
    peak = end['children']['peak_rss_bytes'] if end is not None else None
    assert receipt['compiler_cpu_nanos'] == cpu
    assert receipt['compiler_lifetime_peak_rss_bytes'] == peak
    assert receipt['elapsed_nanos'] > 0 and receipt['process_id'] > 0
    assert receipt['exit_code'] == 0 and receipt['command']


def check_retained(binary, installation, root, working_dir, environment):
    root.mkdir()
    installation_value = json.loads(installation.read_bytes())
    commands = []

    def run(name, vector, expected=0, cwd=None, extra=None):
        result = subprocess.run(list(map(str, vector)), capture_output=True, timeout=600,
            env=dict(environment, **(extra or {})), cwd=cwd)
        (root / (name + '.stdout')).write_bytes(result.stdout)
        (root / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, command=list(map(str, vector)), exit=result.returncode,
            stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (root / 'commands.json').write_text(json.dumps(commands, sort_keys=True, separators=(',', ':')))
        if result.returncode != expected:
            raise ValueError(f'{name}: {result.returncode}: {result.stdout!r} {result.stderr!r}')
        return result.stdout

    compiler = installation_value['compiler_path']
    original_root = Path(run('compiler-root', [compiler, 'env', 'GOROOT'], extra={'GOENV':'off', 'GOTOOLCHAIN':'local', 'GOWORK':'off'}).decode().strip())
    input_root = Path(installation_value['cache_root']) / 'plan-portable-inputs'
    input_root.mkdir(exist_ok=True)
    alias = input_root / 'private-compiler'
    (alias / 'bin').mkdir(parents=True)
    shutil.copy2(compiler, alias / 'bin/go')
    for entry in original_root.iterdir():
        if entry.name != 'bin':
            if entry.is_dir():
                (alias / entry.name).symlink_to(entry, target_is_directory=True)
            else:
                shutil.copy2(entry, alias / entry.name)
    source = input_root / 'private-source'
    source.mkdir()
    for filename in ('go.mod', 'main.go'):
        shutil.copy2(working_dir / 'internal/backends/wasm/qualification/testdata/fixture' / filename, source / filename)
    private_installation = root / 'installation.json'
    run('install-private-inputs', [binary, 'install', '--compiler', alias / 'bin/go',
        '--runtime-root', installation_value['runtime_root'], '--profile', installation_value['profile'],
        '--cache', input_root / 'preparation', '--output', private_installation])
    vector = [binary, '--installation', private_installation, 'plan', '--json', '--choices', '--seeds=7',
        '--output', root / 'plan.json', '--artifacts', root / 'artifacts', '--working-dir', source, 'go-run', '.']
    response = json.loads(run('prepare-private-inputs', vector))
    receipt_path = Path(response['preparation_receipt'])
    receipt, receipt_bytes = read(receipt_path)
    check_resources(receipt)
    assert receipt['command'] == list(map(str, vector))
    opener = [binary, 'fixture-preparation-open', '--receipt', receipt_path, '--sha256', response['preparation_receipt_sha256']]
    admitted = json.loads(run('open-before-removal', opener))
    shutil.rmtree(alias)
    shutil.rmtree(source)
    assert not alias.exists() and not source.exists()
    error = json.loads(run('existing-plan-before-missing-tools', vector, 2))
    assert 'output already exists' in error['message']
    receipt_collision = root / 'receipt-collision.json.preparation.json'
    receipt_collision.write_bytes(b'existing receipt')
    collision_vector = list(vector)
    collision_vector[collision_vector.index('--output') + 1] = root / 'receipt-collision.json'
    error = json.loads(run('existing-receipt-before-missing-tools', collision_vector, 2))
    assert 'output already exists' in error['message']
    assert receipt_collision.read_bytes() == b'existing receipt'
    assert not (root / 'receipt-collision.json').exists()
    assert len(list((root / 'artifacts/campaigns').iterdir())) == 1
    assert json.loads(run('open-after-removal', opener)) == admitted
    assert admitted['guest_executions'] == 0 and admitted['module_sha256'] == response['module_sha256']
    wrong_hash = [*opener[:-1], '0' * 64]
    run('refuse-wrong-receipt-digest', wrong_hash, 2)
    plan_path = Path(receipt['plan'])
    original_plan = plan_path.read_bytes()
    changed = json.loads(original_plan)
    changed['configuration']['arguments'] = ['changed']
    encode = lambda value: json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    plan_path.write_bytes(encode(changed))
    run('refuse-changed-plan', opener, 2)
    plan_path.write_bytes(original_plan)
    changed_receipt = dict(receipt, unexpected=True)
    changed_bytes = encode(changed_receipt)
    receipt_path.write_bytes(changed_bytes)
    run('refuse-unknown-receipt-field', [*opener[:-1], digest(changed_bytes)], 2)
    changed_receipt = dict(receipt, compiler_cpu_nanos=(receipt['compiler_cpu_nanos'] or 0) + 1)
    changed_bytes = encode(changed_receipt)
    receipt_path.write_bytes(changed_bytes)
    run('refuse-changed-resource-observation', [*opener[:-1], digest(changed_bytes)], 2)
    receipt_path.write_bytes(receipt_bytes)
    module = Path(receipt['campaign']) / receipt['prepared_artifact'] / receipt['module_file']
    module_bytes = module.read_bytes()
    module.write_bytes(b'changed module')
    run('refuse-changed-module', opener, 2)
    module.write_bytes(module_bytes)
    artifact = Path(receipt['campaign']) / receipt['prepared_artifact']
    provenance_path = artifact / receipt['provenance_file']
    provenance, original_provenance = read(provenance_path)
    artifact_manifest_path = artifact / 'manifest.json'
    artifact_manifest, original_artifact_manifest = read(artifact_manifest_path)
    actual_metadata = provenance['module_metadata']
    assert actual_metadata == provenance['inputs']['module_metadata']
    assert actual_metadata['imports'] and isinstance(actual_metadata['initial_memory_pages'], int)
    authenticate = runpy.run_path(str(Path(__file__).with_name('campaign-check.py')))['authenticate_artifact']
    for name in ('imports', 'memory'):
        label = 'refuse-resealed-module-metadata-' + name
        changed_provenance = copy.deepcopy(provenance)
        if name == 'imports':
            changed_provenance['module_metadata']['imports'] = []
        else:
            changed_provenance['module_metadata']['initial_memory_pages'] += 1
        changed_raw = encode(changed_provenance)
        changed_manifest = copy.deepcopy(artifact_manifest)
        changed_manifest['files'][receipt['provenance_file']].update(bytes=len(changed_raw), sha256=digest(changed_raw))
        changed_manifest_raw = encode(changed_manifest)
        changed_receipt_raw = encode(dict(receipt, provenance_bytes=len(changed_raw), provenance_sha256=digest(changed_raw)))
        (root / (label + '-provenance.json')).write_bytes(changed_raw)
        (root / (label + '-manifest.json')).write_bytes(changed_manifest_raw)
        (root / (label + '-receipt.json')).write_bytes(changed_receipt_raw)
        try:
            provenance_path.write_bytes(changed_raw)
            artifact_manifest_path.write_bytes(changed_manifest_raw)
            receipt_path.write_bytes(changed_receipt_raw)
            authenticate(artifact)
            run(label, [*opener[:-1], digest(changed_receipt_raw)], 2)
            assert b'prepared module metadata' in (root / (label + '.stderr')).read_bytes()
        finally:
            provenance_path.write_bytes(original_provenance)
            artifact_manifest_path.write_bytes(original_artifact_manifest)
            receipt_path.write_bytes(receipt_bytes)
        assert json.loads(run('open-restored-after-' + name, opener)) == admitted
    assert json.loads(run('open-restored', opener)) == admitted
    return dict(opened=admitted, removed_source_and_compiler=True,
        refusals=['wrong_receipt_digest', 'changed_plan', 'unknown_receipt_field', 'changed_resource_observation', 'changed_module', 'resealed_metadata_imports', 'resealed_metadata_memory'])


def check_scope(binary, installation, root, working_dir, environment):
    root.mkdir()
    commands = []

    def run(name, vector, expected=0, extra=None, cwd=None):
        result = subprocess.run(list(map(str, vector)), capture_output=True, timeout=600,
            env=dict(environment, **(extra or {})), cwd=cwd)
        (root / (name + '.stdout')).write_bytes(result.stdout)
        (root / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, command=list(map(str, vector)), exit=result.returncode,
            stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (root / 'commands.json').write_text(json.dumps(commands, sort_keys=True, separators=(',', ':')))
        if result.returncode != expected:
            raise ValueError(f'{name}: {result.returncode}: {result.stdout!r} {result.stderr!r}')
        return result.stdout

    prefix = [binary, '--installation', installation, 'plan', '--json']
    prepared_target = None

    def command(name, pattern=None, strategy='choice-exploration'):
        vector = [*prefix, '--output', root / (name + '.json'), '--strategy=' + strategy]
        if pattern is not None:
            vector.append('--site-scope=' + pattern)
        vector += ['--choices', '--seeds=7', '--on-failure=all', '--max-executions=4',
            '--max-choice-depth=1024', '--max-exploration-bytes=1MiB',
            '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', root / (name + '-artifacts'),
            '--working-dir', working_dir]
        if prepared_target is None:
            vector += ['go-run', './testdata/parity/site_scope']
        else:
            vector += ['exec', '--provenance', prepared_target / 'target/provenance.json', '--', prepared_target / 'target/module.wasm']
        return vector

    invalid = []
    for index, (strategy, pattern) in enumerate([
        ('seed', 'main'), ('simulation-exploration', 'main'),
        ('choice-exploration', ''), ('choice-exploration', 'main,'),
        ('choice-exploration', 'main/ child'), ('choice-exploration', 'main/*'),
    ]):
        name = 'invalid-' + str(index)
        error = json.loads(run(name, command(name, pattern, strategy), 2))
        assert error['classification'] == 'invalid_input' and 'site-scope' in error['message'], error
        assert not (root / (name + '-artifacts')).exists()
        invalid.append(error)

    observations = {}
    for name, pattern in [('unscoped', None), ('matching', 'main'), ('empty', 'gomad.invalid/no-match/...')]:
        response = json.loads(run(name, command(name, pattern)))
        plan, plan_raw = read(Path(response['plan']))
        receipt, receipt_raw = read(Path(response['preparation_receipt']))
        check_resources(receipt)
        assert digest(plan_raw) == receipt['plan_sha256'] == response['plan_sha256']
        assert digest(receipt_raw) == response['preparation_receipt_sha256']
        campaign = Path(receipt['campaign'])
        artifact = campaign / plan['prepared']['prepared_artifact']
        authenticate = runpy.run_path(str(Path(__file__).with_name('campaign-check.py')))['authenticate_artifact']
        authenticate(artifact)
        config = plan['strategy']['config']
        events = [json.loads(line)['event'] for line in (campaign / 'journal.jsonl').read_bytes().splitlines()]
        if pattern is None:
            assert receipt['guest_executions'] == response['guest_executions'] == 0 and events == []
            assert 'site_scope' not in config
            prepared_target = artifact
        else:
            assert receipt['guest_executions'] == response['guest_executions'] == 1 and len(events) == 1
            assert len(events[0]['executions']) == 1
            scope = config['site_scope']
            assert scope['patterns'] == [pattern]
            assert scope['resolved_sha256'].startswith('sha256:') and len(scope['resolved_sha256']) == 71
            assert events[0]['segment']['site_scope'] == scope
            base = events[0]['executions'][0]
            assert base['artifact']
            authenticate(campaign / base['artifact'])
            if name == 'empty':
                assert scope['resolved_count'] == 0
                assert events[0]['segment']['scope_counts']['forced_in_scope'] == 0
            else:
                assert scope['resolved_count'] > 0
            envelope = response['envelope']
            assert envelope['logical_executions'] == 1
            assert envelope['site_scope'] == scope
            if name == 'empty':
                assert envelope['stop_reason'] == 'site_scope_empty'
        observations[name] = dict(plan_sha256=response['plan_sha256'], guest_executions=response['guest_executions'],
            site_scope=config.get('site_scope'), module_sha256=response['module_sha256'], envelope=response['envelope'])

    provenance, _ = read(prepared_target / 'target/provenance.json')
    manifest = json.loads(installation.read_bytes())
    overlay = None
    for filename in provenance['inputs']['compiler']:
        if filename.endswith('.json'):
            candidate = json.loads(Path(filename).read_bytes())
            if isinstance(candidate, dict) and 'Replace' in candidate:
                overlay = filename
                break
    assert overlay is not None
    stripped = root / 'stripped.wasm'
    build_environment = dict(provenance['inputs']['environment'], GOCACHE=str(root / 'go-cache'))
    run('build-stripped', [manifest['compiler_path'], 'build', '-mod=readonly', '-pgo=off', '-overlay', overlay,
        '-trimpath', '-buildvcs=false', '-ldflags=-s -w', '-o', stripped, './testdata/parity/site_scope'],
        extra=build_environment, cwd=working_dir)
    stripped_provenance = copy.deepcopy(provenance)
    stripped_provenance['inputs']['build_flags'].append('-ldflags=-s -w')
    encode = lambda value: json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    stripped_provenance['build_key'] = digest(encode(stripped_provenance['inputs']))
    stripped_provenance['module_sha256'] = digest(stripped.read_bytes())
    stripped_provenance['choice_identity']['TargetSHA256'] = list(bytes.fromhex(stripped_provenance['module_sha256']))
    stripped_provenance['choice_identity']['ToolchainBuildKey'] = stripped_provenance['build_key']
    implementation = json.loads(run('stripped-choice-implementation',
        [binary, 'fixture-choice-implementation', '--build-key', stripped_provenance['build_key']]))
    stripped_provenance['choice_identity']['ImplementationSHA256'] = implementation['implementation_sha256']
    stripped_provenance['configuration']['runtime_sha256'] = implementation['runtime_sha256']
    stripped_path = root / 'stripped-provenance.json'
    stripped_path.write_bytes(encode(stripped_provenance))
    invocation = command('stripped', 'main')
    invocation[invocation.index('exec'):] = ['exec', '--provenance', stripped_path, '--', stripped]
    error = json.loads(run('stripped', invocation, 2))
    assert 'symbol information' in error['message'] and 'no_symbols' in error['message'], error
    assert not (root / 'stripped-artifacts').exists()
    observations['invalid'] = invalid
    observations['no_symbols'] = dict(error=error, provenance_origin='fixture-derived from actual stripped module and production choice implementation')
    return observations


def check(binary, installation, root, working_dir, package='./...', go_binary=None, go_installation=None, go_working_dir=None, kind='go-run'):
    root.mkdir(parents=True)
    output = root / 'plan.json'
    profile = json.loads(installation.read_bytes())['profile']
    trace_flags = ['--choices'] if profile == 'gomad.wasi-cooperative/v1' else []
    command = [str(binary), '--installation', str(installation), 'plan', '--json',
        '--output', str(output), '--artifacts', str(root / 'artifacts'),
        '--working-dir', str(working_dir), '--seeds=7', '--on-failure=all',
        *trace_flags, kind, package, '--', '-test.run=^TestPrepareOnlyNeverExecuted$']
    environment = {key: value for key, value in os.environ.items()
        if not key.startswith('GOMAD_') and key not in ('GOROOT', 'GOTOOLCHAIN', 'GOWORK')}
    environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', GOPROXY='off', GOSUMDB='off', GOPRIVATE='', GONOPROXY='none', GOVCS='*:off')
    result = subprocess.run(command, capture_output=True, timeout=600, env=environment)
    (root / 'exit.txt').write_text(str(result.returncode) + '\n')
    (root / 'stdout.json').write_bytes(result.stdout)
    (root / 'stderr.txt').write_bytes(result.stderr)
    if result.returncode != 0:
        raise ValueError(f'public plan failed {result.returncode}: {result.stderr!r} {result.stdout!r}')
    response = json.loads(result.stdout)
    plan, plan_raw = read(output)
    receipt_path = Path(response['preparation_receipt'])
    receipt, receipt_raw = read(receipt_path)
    check_resources(receipt)
    assert receipt['command'] == command
    assert response['guest_executions'] == receipt['guest_executions'] == 0
    assert response['plan_sha256'] == receipt['plan_sha256'] == digest(plan_raw)
    assert response['preparation_receipt_sha256'] == digest(receipt_raw)
    campaign = Path(receipt['campaign'])
    assert (campaign / 'journal.jsonl').read_bytes() == b''
    artifact = campaign / receipt['prepared_artifact']
    auth = runpy.run_path(str(Path(__file__).with_name('campaign-check.py')))['authenticate_artifact']
    record = auth(artifact)
    assert record['outcome']['class'] == record['outcome']['termination'] == 'prepared'
    assert receipt['record_sha256'] == digest((artifact / 'record.json').read_bytes())
    module = (artifact / receipt['module_file']).read_bytes()
    assert receipt['module_sha256'] == response['module_sha256'] == digest(module)
    assert receipt['module_bytes'] == len(module)
    provenance, provenance_raw = read(artifact / receipt['provenance_file'])
    assert receipt['provenance_sha256'] == digest(provenance_raw)
    assert receipt['provenance_bytes'] == len(provenance_raw)
    assert provenance['inputs']['kind'] == kind
    assert provenance['inputs']['source'] == package
    assert provenance['inputs']['arguments'] == ['-test.run=^TestPrepareOnlyNeverExecuted$']
    assert provenance['build_key'] == digest(json.dumps(provenance['inputs'], sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode())
    assert plan['configuration']['arguments'] == ['gomad-target', '-test.run=^TestPrepareOnlyNeverExecuted$']
    prior = output.read_bytes(), receipt_path.read_bytes()
    repeated = subprocess.run(command, capture_output=True, timeout=600, env=environment)
    (root / 'collision-exit.txt').write_text(str(repeated.returncode) + '\n')
    (root / 'collision-stdout.json').write_bytes(repeated.stdout)
    (root / 'collision-stderr.txt').write_bytes(repeated.stderr)
    assert repeated.returncode != 0
    assert prior == (output.read_bytes(), receipt_path.read_bytes())
    assert output.stat().st_mode & 0o777 == receipt_path.stat().st_mode & 0o777 == 0o600
    retained_view = check_retained(binary, installation, root / 'retained-open', working_dir, environment) if trace_flags else None
    scope_view = check_scope(binary, installation, root / 'site-scope', working_dir, environment) if trace_flags else None
    go_view = None
    if go_binary is not None:
        if go_installation is None:
            raise ValueError('Go reference installation required')
        go_root = root / 'go-explore'
        flags = ['--json', '--seeds=7', '--parallel=1', '--on-failure=all',
            '--working-dir', str(go_working_dir or working_dir), '--keep-successes=all',
            '--success-limit=1', '--success-bytes=256MiB', '--artifacts', str(go_root)]
        flags += trace_flags
        target = [kind, package, '--', '-test.run=^TestPrepareOnlyNeverExecuted$']
        prefix = [str(go_binary), '--installation', str(go_installation)]
        for name, vector, expected in [
            ('go-plan-refusal', [*prefix, 'plan', '--output', str(root / 'go-plan.json'), *flags, *target], 2),
            ('go-explore', [*prefix, 'explore', *flags, *target], 0),
        ]:
            observed = subprocess.run(vector, capture_output=True, timeout=600, env=environment)
            (root / (name + '-exit.txt')).write_text(str(observed.returncode) + '\n')
            (root / (name + '-stdout')).write_bytes(observed.stdout)
            (root / (name + '-stderr')).write_bytes(observed.stderr)
            if observed.returncode != expected:
                raise ValueError(f'{name} status {observed.returncode}: {observed.stderr!r} {observed.stdout!r}')
        reference = runpy.run_path(str(Path(__file__).with_name('campaign-reference-check.py')))
        roots = [path.parent.parent for path in go_root.rglob('store/log')]
        assert len(roots) == 1
        explored = reference['authenticate_campaign'](roots[0])
        assert len(explored['records']) == 1
        execution = explored['records'][0]
        assert int(execution['seed']) == 7
        assert plan['selection'] == '7' and plan['selection_count'] == 1
        artifact_reference = execution.get('success_artifact') or execution.get('artifact')
        assert artifact_reference
        go_artifact = Path(artifact_reference)
        if not go_artifact.is_absolute():
            go_artifact = roots[0] / go_artifact
        go_manifest = reference['authenticate_artifact'](go_artifact)
        go_target = reference['contained'](go_artifact, go_manifest['target']['file']).read_bytes()
        assert digest(go_target) == receipt['module_sha256']
        assert go_manifest['target']['argv'] == plan['configuration']['arguments']
        go_view = {'public_plan_status': 2, 'public_explore_status': 0,
            'selection': plan['selection'], 'selection_count': plan['selection_count'],
            'module_sha256': digest(go_target), 'argv': go_manifest['target']['argv'], 'executed_seed': int(execution['seed'])}
    return {'schema': response['schema'], 'guest_executions': receipt['guest_executions'],
        'prepared_record': record, 'prepared_record_sha256': receipt['record_sha256'],
        'module_sha256': receipt['module_sha256'], 'module_bytes': receipt['module_bytes'],
        'provenance_sha256': receipt['provenance_sha256'], 'provenance_bytes': receipt['provenance_bytes'],
        'build_key': provenance['build_key'], 'plan_sha256': receipt['plan_sha256'],
        'receipt_schema': receipt['schema'], 'publication_collision_refused': True,
        'module_authenticated': True, 'provenance_authenticated': True, 'go_reference': go_view, 'site_scope': scope_view, 'retained_open': retained_view}



if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    for flag in ('binary', 'installation', 'work', 'working-dir'):
        parser.add_argument('--' + flag, required=True, type=Path)
    parser.add_argument('--package', default='./...')
    parser.add_argument('--kind', choices=('go-run', 'go-test'), default='go-run')
    for flag in ('go-binary', 'go-installation', 'go-working-dir'):
        parser.add_argument('--' + flag, type=Path)
    args = parser.parse_args()
    print(json.dumps(check(args.binary, args.installation, args.work, args.working_dir, args.package, args.go_binary, args.go_installation, args.go_working_dir, args.kind), sort_keys=True))
