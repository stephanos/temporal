#!/usr/bin/env python3
"""Exercise public choice exploration with real compiled guests and durable rounds."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import runpy

MODULE = Path(__file__).resolve().parents[2]
rounds = runpy.run_path(str(Path(__file__).with_name('round-check.py')))


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-compiler', 'input', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--go-baseline', type=Path)
    parser.add_argument('--go-source', type=Path)
    parser.add_argument('--divergence', action='store_true')
    parser.add_argument('--divergence-only', action='store_true')
    parser.add_argument('--ordinary-binary', type=Path)
    parser.add_argument('--site-scope-only', action='store_true')
    parser.add_argument('--go-cache', type=Path, default=Path(os.environ.get('GOCACHE', '/home/agent/.cache/go-build-local')))
    args = parser.parse_args()
    assert bool(args.go_baseline) == bool(args.go_source), 'provide both Go baseline and frozen source'
    args.binary, args.go_compiler, args.input, args.work = (
        path.resolve() for path in (args.binary, args.go_compiler, args.input, args.work))
    args.work.mkdir(parents=True, exist_ok=False)
    environment = {name: value for name, value in os.environ.items()
                   if not name.startswith('GOMAD_') and name != 'GOROOT'}
    receipts = []

    def run(name, command, statuses=(0,), extra=None, cwd=None):
        result = subprocess.run(list(map(str, command)), env=dict(environment, **(extra or {})),
                                capture_output=True, timeout=900, cwd=cwd)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                             cwd=str(cwd) if cwd else None, environment_overrides=extra or {},
                             stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(encode(receipts))
        assert result.returncode in statuses, (name, result.returncode, result.stdout.decode(errors='replace'),
                                               result.stderr.decode(errors='replace'))
        return result.stdout

    identity = args.work / 'identity'
    identity_command = [args.binary, 'fixture-replay-input', '--root', identity, '--input', args.input]
    if args.divergence or args.divergence_only:
        identity_command.extend(['--role', 'determinism-fixture'])
    bundle = json.loads(run('identity', identity_command))
    cooperative = next(identity / row['artifact'] for row in bundle['records']
                       if json.loads((identity / row['artifact'] / 'record.json').read_bytes())['profile'] == 'gomad.wasi-cooperative/v1')
    replay = json.loads((cooperative / 'evidence/replay.json').read_bytes())
    configuration = replay['configuration']
    model = configuration['model']
    model['mode'], model['read_step_nanos'] = 1, 0
    model.pop('seed', None)
    host_environment = dict(GOENV='off', GOFLAGS='', GOTOOLCHAIN='local', GOWORK='off',
        GOOS='', GOARCH='', CGO_ENABLED='0', GOCACHE=str(args.go_cache.resolve()))
    stock_root = run('compiler-root', [args.go_compiler, 'env', 'GOROOT'], extra=host_environment).decode().strip()
    current_overlay = json.loads(run('current-runtime-overlay', [args.go_compiler, 'run',
        './internal/backends/wasm/toolchain/cmd/overlay', stock_root, MODULE / 'internal/model/runtimesource',
        args.work / 'preparation/runtime-overlay'], extra=host_environment, cwd=MODULE))
    installation = dict(schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
        compiler_sha256=digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
        runtime_root=str(MODULE / 'internal/model/runtimesource'), cache_root=str(args.work / 'preparation'),
        profile='gomad.wasi-cooperative/v1', fuel=1 << 60, memory_bytes=256 << 20,
        helper_sha256=replay['helper_sha256'], host_sha256=replay['host_sha256'],
        runtime_implementation_sha256=current_overlay['implementation_sha256'], model=model,
        namespace=configuration['namespace'], storage=None, network=None, build_tags=[])
    manifest = args.work / 'installation.json'
    manifest.write_bytes(encode(installation))
    prefix = [args.binary, '--installation', manifest]

    def command(destination, executions=8, capacity='128MiB', depth=2, scope=None):
        invocation = [*prefix, 'explore', '--json', '--strategy=choice-exploration', '--choices',
            '--on-failure=all', '--parallel=1', '--seeds=89', '--max-executions=' + str(executions),
            '--max-choice-depth=' + str(depth), '--max-exploration-bytes=' + capacity,
            '--keep-successes=all', '--success-limit=32', '--success-bytes=512MiB',
            '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', destination,
            '--working-dir', MODULE, 'go-run', './internal/backends/wasm/toolchain/testdata/choices']
        if scope is not None:
            invocation.insert(invocation.index('--choices'), '--site-scope=' + scope)
        return invocation

    def locate(destination):
        roots = list((destination / 'campaigns').glob('campaign-*'))
        assert len(roots) == 1, roots
        return roots[0]

    def events(root):
        return rounds['authenticate'](root)['events']

    def check_site_scope():
        prepared_target = None

        def scoped_command(destination, **bounds):
            invocation = command(destination, **bounds)
            target = invocation.index('go-run')
            if prepared_target is None:
                invocation[target + 1] = './testdata/parity/site_scope'
            else:
                invocation[target:] = ['exec', '--provenance', prepared_target / 'target/provenance.json',
                                      '--', prepared_target / 'target/module.wasm']
            return invocation

        for index, (strategy, pattern) in enumerate([
                ('seed', 'main'), ('simulation-exploration', 'main'),
                ('choice-exploration', ''), ('choice-exploration', 'main,'),
                ('choice-exploration', 'main/ child'), ('choice-exploration', 'main/*')]):
            destination = args.work / ('scope-invalid-' + str(index))
            invocation = scoped_command(destination, scope=pattern)
            invocation[invocation.index('--strategy=choice-exploration')] = '--strategy=' + strategy
            error = json.loads(run('scope-invalid-' + str(index), invocation, (2,)))
            assert error['classification'] == 'invalid_input' and 'site-scope' in error['message'], error
            assert not destination.exists(), destination

        observations = {}
        for name, pattern in [('unscoped', None), ('main', 'main'), ('empty', 'gomad.invalid/no-match/...')]:
            destination = args.work / ('scope-' + name)
            invocation = scoped_command(destination, executions=3, depth=1024, scope=pattern)
            if name == 'empty':
                invocation[invocation.index('--keep-successes=all')] = '--keep-successes=none'
                invocation.remove('--success-limit=32')
                invocation.remove('--success-bytes=512MiB')
            raw = run('scope-' + name, invocation, (0, 1))
            root = locate(destination)
            checked = rounds['authenticate'](root)
            envelope = json.loads(raw.splitlines()[-1])['envelope']
            inspected = json.loads(run('scope-' + name + '-inspect', [*prefix, 'inspect', '--json', '--choices', root]))
            observations[name] = dict(envelope=envelope)
            if pattern is None:
                prepared_target = root / checked['plan']['prepared']['prepared_artifact']
                assert 'site_scope' not in checked['plan']['strategy']['config']
                assert all('site_scope' not in event['segment'] and 'scope_counts' not in event['segment']
                           for event in checked['events'])
            else:
                observations[name]['semantics'] = rounds['scope_semantics'](root, inspected, envelope)
                before = rounds['snapshot'](root)
                refusal = json.loads(run('scope-' + name + '-changed-resume',
                    [*prefix, 'resume', '--json', '--site-scope=net/http', root], (2,)))
                assert 'exploration plan identity or bounds changed' in refusal['message'], refusal
                assert rounds['snapshot'](root) == before
                same = json.loads(run('scope-' + name + '-same-resume', [*prefix, 'resume', '--json', '--site-scope=' + pattern, root], (2,)))
                assert same['classification'] == 'invalid_input' and 'complete' in same['message'], same
                assert rounds['snapshot'](root) == before
                if name == 'empty':
                    assert envelope['stop_reason'] == 'site_scope_empty' and envelope['logical_executions'] == 1, envelope
                    assert envelope['site_scope']['resolved_count'] == 0 and checked['events'][0]['executions'][0]['artifact']
                else:
                    assert observations[name]['semantics']['admitted'] > 0, observations[name]
                    assert observations[name]['semantics']['skipped'] > 0, observations[name]
                    assert observations[name]['semantics']['runnable_in_scope'] > 0, observations[name]
                    assert observations[name]['semantics']['runnable_out_of_scope'] > 0, observations[name]
                    assert observations[name]['semantics']['mixed_runnable'] > 0, observations[name]
                    scoped_root = root
        assert observations['main']['envelope']['deepest_prefix'] > observations['unscoped']['envelope']['deepest_prefix'], observations

        interrupted_destination = args.work / 'scope-interrupted'
        run('scope-interrupt', scoped_command(interrupted_destination, executions=3, depth=1024, scope='main'),
            (86,), {'GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER': '0'})
        interrupted = locate(interrupted_destination)
        assert len(events(interrupted)) == 1
        before = rounds['snapshot'](interrupted)
        run('scope-interrupted-changed-resume', [*prefix, 'resume', '--json', '--site-scope=net/http', interrupted], (2,))
        assert rounds['snapshot'](interrupted) == before
        run('scope-resume', [*prefix, 'resume', '--json', '--site-scope=main', interrupted], (0, 1))
        assert events(interrupted) == events(scoped_root)

        # Retain scope on a real failure, then discard its campaign before minimizing.
        failure_destination = args.work / 'scope-failure'
        invocation = command(failure_destination, executions=8, depth=1024, scope='main')
        failure_directory = MODULE / 'testdata/parity/live_backend/minimization'
        assert (failure_directory / 'go.mod').is_file()
        invocation[invocation.index('--working-dir') + 1] = failure_directory
        invocation[invocation.index('go-run') + 1] = '.'
        run('scope-failure', invocation, (1,))
        failure_root = locate(failure_destination)
        failure_checked = rounds['authenticate'](failure_root)
        failure_prepared = failure_root / failure_checked['plan']['prepared']['prepared_artifact']
        failure_provenance = json.loads((failure_prepared / 'target/provenance.json').read_bytes())
        snapshots = []
        detached = args.work / 'scope-failure-detached'
        for ordinal, event in enumerate(failure_checked['events']):
            expected_resolution = failure_checked['plan']['strategy']['config']['site_scope'] if ordinal == 0 else previous_resolution
            for execution in event['executions']:
                if not execution.get('artifact'):
                    continue
                artifact = failure_root / execution['artifact']
                manifest = json.loads((artifact / 'manifest.json').read_bytes())
                context_raw = (artifact / 'provenance/search-context.json').read_bytes()
                entry = manifest['files']['provenance/search-context.json']
                assert (len(context_raw), digest(context_raw)) == (entry['bytes'], entry['sha256'])
                context = json.loads(context_raw)
                assert context['plan_sha256'] == digest(encode(failure_checked['plan']))
                assert context['round'] == ordinal and context['scope'] == expected_resolution
                assert 'provenance/search-context.json' not in manifest['record']['streams']
                snapshots.append(dict(round=ordinal, payload_sha256=digest(context_raw), scope=context['scope']))
                if not detached.exists() and manifest['record']['outcome']['class'] == 'target_failure':
                    shutil.copytree(artifact, detached)
            previous_resolution = dict(patterns=['main'], resolved_sha256=event['segment']['site_scope']['resolved_sha256'], resolved_count=event['segment']['site_scope']['resolved_count'])
        assert detached.exists() and len(snapshots) >= 2, snapshots
        shutil.rmtree(failure_destination)
        minimized_root = args.work / 'scope-failure-minimized'
        minimize_command = [args.binary, 'minimize', '--json', '--artifacts', minimized_root, detached]
        minimize_controls = args.work / 'scope-failure-minimize-controls.jsonl'
        minimized = json.loads(run('scope-failure-minimize', minimize_command,
            extra={'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(minimize_controls)}))
        assert minimized['site_scope']['patterns'] == ['main'] and minimized['changed'], minimized
        assert all(row['removed_in_scope'] + row['removed_out_of_scope'] == len(row['removed']) for row in minimized['accepted'])
        assert json.loads(run('scope-failure-minimize-resume', [*minimize_command[:3], '--resume', *minimize_command[3:]])) == minimized
        run('scope-failure-minimize-changed-scope', [args.binary, 'minimize', '--resume', '--site-scope=runtime', '--artifacts', minimized_root, detached], (3,))
        override = json.loads(run('scope-failure-minimize-override', [args.binary, 'minimize', '--json', '--site-scope=runtime', '--artifacts', args.work / 'scope-failure-override', detached]))
        assert override['site_scope']['patterns'] == ['runtime']
        parent_before = rounds['snapshot'](detached)
        for index, pattern in enumerate(('', 'main,', 'main/ child', 'main/*')):
            name = 'scope-minimize-invalid-' + str(index)
            destination = args.work / name
            controls = args.work / (name + '-controls.jsonl')
            run(name, [args.binary, 'minimize', '--site-scope=' + pattern,
                '--artifacts', destination, detached], (2,),
                {'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(controls)})
            assert not destination.exists() and not controls.exists(), name
        empty_minimum = json.loads(run('scope-minimize-empty', [args.binary, 'minimize', '--json',
            '--site-scope=gomad.invalid/no-match/...', '--artifacts', args.work / 'scope-minimize-empty', detached]))
        assert empty_minimum['changed'] and empty_minimum['accepted'], empty_minimum
        assert empty_minimum['site_scope']['resolved_count'] == 0, empty_minimum
        assert all(row['removed_in_scope'] == 0 and row['removed_out_of_scope'] == len(row['removed'])
                   for row in empty_minimum['accepted']), empty_minimum
        recovery = {}
        for mode in ('after accepted', 'after publication'):
            name = 'scope-minimize-' + mode.replace(' ', '-')
            destination = args.work / name
            controls = args.work / (name + '-controls.jsonl')
            invocation = [args.binary, 'minimize', '--json', '--artifacts', destination, detached]
            run(name, invocation, (3,), {'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(controls),
                'GOMAD_FIXTURE_MINIMIZE_INTERRUPT': mode})
            assert ('interrupted ' + mode).encode() in (args.work / (name + '.stderr')).read_bytes()
            resumed = json.loads(run(name + '-resume', [*invocation[:3], '--resume', *invocation[3:]],
                extra={'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(controls)}))
            assert all(resumed[key] == minimized[key] for key in ('attempts', 'record_hash', 'accepted')), resumed
            if mode == 'after accepted':
                assert controls.read_bytes() == minimize_controls.read_bytes(), controls
            recovery[mode] = resumed
        changed_context = args.work / 'scope-failure-changed-context'
        shutil.copytree(detached, changed_context)
        changed_path = changed_context / 'provenance/search-context.json'
        changed = json.loads(changed_path.read_bytes())
        changed['round'] += 1
        changed_path.write_bytes(encode(changed))
        changed_manifest_path = changed_context / 'manifest.json'
        changed_manifest = json.loads(changed_manifest_path.read_bytes())
        changed_manifest['files']['provenance/search-context.json'].update(
            bytes=changed_path.stat().st_size, sha256=digest(changed_path.read_bytes()))
        changed_manifest_path.write_bytes(encode(changed_manifest))
        assert (changed_context / 'record.json').read_bytes() == (detached / 'record.json').read_bytes()
        checkpoint_before = rounds['snapshot'](minimized_root)
        run('scope-minimize-changed-context', [args.binary, 'minimize', '--resume',
            '--artifacts', minimized_root, changed_context], (3,))
        assert b'checkpoint binding changed' in (args.work / 'scope-minimize-changed-context.stderr').read_bytes()
        assert rounds['snapshot'](minimized_root) == checkpoint_before
        assert rounds['snapshot'](detached) == parent_before
        raw_before = (Path(minimized['artifact_path']) / 'evidence/inspection.json').read_bytes()
        inspected_minimum = json.loads(run('scope-failure-minimum-inspect', [args.binary, 'inspect', '--json', minimized['artifact_path']]))['artifact']['minimization']
        assert sum(group['count'] for group in inspected_minimum['final_forced_by_symbol']) == inspected_minimum['final_forced_decisions']
        assert all('symbol' in decision for reduction in inspected_minimum['accepted'] for decision in reduction['removed'])
        assert (Path(minimized['artifact_path']) / 'evidence/inspection.json').read_bytes() == raw_before
        display = runpy.run_path(str(Path(__file__).with_name('site-minimizer-display.py')))
        (args.work / 'scope-minimizer-display.json').write_bytes(
            display['check'](minimized, inspected_minimum, Path(minimized['artifact_path'])))
        replayed = run('scope-failure-minimum-replay', [args.binary, 'replay', minimized['artifact_path']], (1,))
        assert b'reproduced=true' in replayed
        malformed = args.work / 'scope-failure-invalid-context'
        shutil.copytree(detached, malformed)
        context_path = malformed / 'provenance/search-context.json'
        context = json.loads(context_path.read_bytes())
        context['round'] = -1
        context_path.write_bytes(encode(context))
        manifest_path = malformed / 'manifest.json'
        manifest = json.loads(manifest_path.read_bytes())
        manifest['files']['provenance/search-context.json'].update(bytes=context_path.stat().st_size, sha256=digest(context_path.read_bytes()))
        manifest_path.write_bytes(encode(manifest))
        invalid_output = args.work / 'scope-failure-invalid-output'
        run('scope-failure-invalid-context-minimize', [args.binary, 'minimize', '--site-scope=main', '--artifacts', invalid_output, malformed], (2,))
        assert str(malformed).encode() in (args.work / 'scope-failure-invalid-context-minimize.stderr').read_bytes()
        assert not invalid_output.exists()
        observations['minimize'] = dict(snapshots=snapshots, result=minimized, override=override,
            empty=empty_minimum, recovery=recovery, inspection=inspected_minimum)

        rounds['scope_digest_refusal'](run, prefix, scoped_root, args.work / 'scope-changed-digest')
        rounds['scope_block_refusals'](run, prefix, scoped_root, args.work)

        prepared = rounds['authenticate'](scoped_root)['plan']['prepared']
        provenance = json.loads((scoped_root / prepared['prepared_artifact'] / 'target/provenance.json').read_bytes())

        def build_stripped(name, original_provenance):
            directory = Path(original_provenance['inputs']['directory'])
            package = original_provenance['inputs']['source']
            stripped = args.work / (name + '.wasm')
            build_environment = dict(original_provenance['inputs']['environment'], GOCACHE=str(args.go_cache.resolve()))
            run(name + '-build', [args.go_compiler, 'build', '-mod=readonly', '-pgo=off',
                '-overlay', current_overlay['path'], '-trimpath', '-buildvcs=false', '-ldflags=-s -w',
                '-o', stripped, package], extra=build_environment, cwd=directory)
            stripped_provenance = copy.deepcopy(original_provenance)
            stripped_provenance['inputs']['build_flags'].append('-ldflags=-s -w')
            stripped_provenance['build_key'] = digest(encode(stripped_provenance['inputs']))
            stripped_provenance['module_sha256'] = digest(stripped.read_bytes())
            stripped_provenance['choice_identity']['TargetSHA256'] = list(bytes.fromhex(stripped_provenance['module_sha256']))
            stripped_provenance['choice_identity']['ToolchainBuildKey'] = stripped_provenance['build_key']
            provenance_path = args.work / (name + '-provenance.json')
            provenance_path.write_bytes(encode(stripped_provenance))
            return stripped, provenance_path

        stripped, provenance_path = build_stripped('stripped', provenance)
        destination = args.work / 'scope-no-symbols'
        invocation = scoped_command(destination, scope='main')
        invocation[invocation.index('exec'):] = ['exec', '--provenance', provenance_path, '--', stripped]
        error = json.loads(run('scope-no-symbols', invocation, (2,)))
        assert 'symbol information' in error['message'] and 'no_symbols' in error['message'], error
        assert not destination.exists(), destination
        observations['no_symbols'] = dict(module_sha256=digest(stripped.read_bytes()), error=error)
        stripped_failure, stripped_failure_provenance = build_stripped('stripped-failure', failure_provenance)
        stripped_destination = args.work / 'scope-stripped-failure'
        invocation = command(stripped_destination, executions=8, depth=1024)
        target = invocation.index('go-run')
        invocation[target:] = ['exec', '--provenance', stripped_failure_provenance, '--', stripped_failure]
        run('scope-stripped-failure', invocation, (1,))
        stripped_root = locate(stripped_destination)
        stripped_checked = rounds['authenticate'](stripped_root)
        assert 'site_scope' not in stripped_checked['plan']['strategy']['config']
        retained_failure = next(stripped_root / execution['artifact']
            for event in stripped_checked['events'] for execution in event['executions']
            if execution.get('artifact') and json.loads((stripped_root / execution['artifact'] / 'record.json').read_bytes())['outcome']['class'] == 'target_failure')
        assert (retained_failure / 'target/module.wasm').read_bytes() == stripped_failure.read_bytes()
        stripped_detached = args.work / 'scope-stripped-failure-detached'
        shutil.copytree(retained_failure, stripped_detached)
        stripped_before = rounds['snapshot'](stripped_detached)
        stripped_output = args.work / 'scope-stripped-minimize'
        stripped_controls = args.work / 'scope-stripped-minimize-controls.jsonl'
        run('scope-stripped-minimize', [args.binary, 'minimize', '--site-scope=main',
            '--artifacts', stripped_output, stripped_detached], (2,),
            {'GOMAD_FIXTURE_MINIMIZE_CONTROLS': str(stripped_controls)})
        stripped_error = (args.work / 'scope-stripped-minimize.stderr').read_bytes()
        assert b'no_symbols' in stripped_error and b'symbol information' in stripped_error, stripped_error
        assert str(stripped_detached).encode() in stripped_error, stripped_error
        assert not stripped_output.exists() and not stripped_controls.exists()
        assert rounds['snapshot'](stripped_detached) == stripped_before
        observations['no_symbols']['minimize'] = dict(module_sha256=digest(stripped_failure.read_bytes()),
            parent_record_sha256=digest((stripped_detached / 'record.json').read_bytes()), status=2)
        (args.work / 'site-scope-observations.json').write_bytes(encode(dict(
            schema='gomad.site-scope-public-check/v1', binary_sha256=digest(args.binary.read_bytes()),
            driver_sha256=digest(Path(__file__).read_bytes()), observations=observations)))

    def check_divergences():
        observations = {}
        ordinary_manifest = args.work / 'ordinary-installation.json'
        if args.ordinary_binary:
            ordinary_identity = args.work / 'ordinary-identity'
            ordinary_bundle = json.loads(run('ordinary-identity', [args.ordinary_binary.resolve(),
                'fixture-replay-input', '--root', ordinary_identity, '--input', args.input]))
            ordinary_artifact = next(ordinary_identity / row['artifact'] for row in ordinary_bundle['records']
                if json.loads((ordinary_identity / row['artifact'] / 'record.json').read_bytes())['profile'] == 'gomad.wasi-cooperative/v1')
            ordinary_replay = json.loads((ordinary_artifact / 'evidence/replay.json').read_bytes())
            ordinary_installation = dict(installation, cache_root=str(args.work / 'ordinary-preparation'))
            for field in ('helper_sha256', 'host_sha256'):
                ordinary_installation[field] = ordinary_replay[field]
            ordinary_manifest.write_bytes(encode(ordinary_installation))
        for case in ('sites', 'expected-missing', 'observed-missing'):
            destination = args.work / ('divergence-' + case)
            invocation = command(destination)
            invocation[-1] = './internal/backends/wasm/toolchain/testdata/choice_divergence'
            invocation.extend(['--', case])
            run(case + '-explore', invocation, (0, 1))
            campaign = locate(destination)

            def snapshot():
                return {str(path.relative_to(campaign)): digest(path.read_bytes())
                        for path in campaign.rglob('*') if path.is_file()}

            before = snapshot()
            inspected = run(case + '-inspect', [*prefix, 'inspect', '--json', '--choices', campaign])
            repeated = run(case + '-inspect-repeat', [*prefix, 'inspect', '--json', '--choices', campaign])
            assert inspected == repeated and snapshot() == before, case
            report = json.loads(inspected)
            replay_symbols = {}
            for execution in report['campaign']['executions']:
                for site in execution.get('choices', {}).get('sites', []):
                    assert 'symbol' in site, (case, site)
                for decision in execution.get('choices', {}).get('replay_decisions', []):
                    assert 'symbol' in decision, (case, decision)
                    replay_symbols[(decision['site_offset'], decision['site_missing'])] = decision['symbol']
            divergences = [execution['divergence'] for execution in report['campaign']['executions']
                           if execution['divergence']]
            authenticated = [execution['divergence'] for event in events(campaign)
                             for execution in event['executions'] if execution['divergence']]
            assert divergences, ('no actual helper divergence', case)
            assert len(divergences) == len(authenticated), case
            for divergence, recorded in zip(divergences, authenticated):
                assert divergence['reason'] == ('site' if case == 'sites' else 'kind'), divergence
                assert divergence['ordinal'] == recorded['Ordinal'], divergence
                assert divergence['tape_records'] == recorded['TapeRecords'], divergence
                assert divergence['tape_records'] > divergence['ordinal'], divergence
                for side in ('expected', 'observed'):
                    decision = divergence[side]
                    assert decision is not None, (case, side, divergence)
                    assert {key: value for key, value in decision.items() if key != 'symbol'} == recorded[side.title()]
                    missing = case == side + '-missing'
                    assert decision['SiteMissing'] == missing, decision
                    if missing:
                        assert decision['symbol'] == {'unavailable': 'site_missing'}, decision
                    else:
                        function = 'recordedSelect' if side == 'expected' else 'replayedSelect'
                        assert decision['symbol']['function'] == function, decision
                        assert decision['symbol']['package'] == 'main', decision
                    if side == 'expected':
                        assert decision['symbol'] == replay_symbols[(decision['SiteOffset'], missing)], decision
            observations[case] = divergences
            if args.ordinary_binary:
                ordinary = invocation.copy()
                ordinary[0] = args.ordinary_binary.resolve()
                ordinary[2] = ordinary_manifest
                ordinary[ordinary.index('--artifacts') + 1] = args.work / ('ordinary-' + case)
                run(case + '-ordinary-refusal', ordinary, (2, 3))
                ordinary_roots = list((args.work / ('ordinary-' + case) / 'campaigns').glob('campaign-*'))
                for root in ordinary_roots:
                    assert not events(root), root
                    assert not (root / 'evidence').exists(), root
        (args.work / 'divergence-observations.json').write_bytes(encode(dict(
            schema='gomad.choice-divergence-symbol-check/v1',
            scope='explicit helper determinism fixture for authenticated inspect projection; not ordinary qualification',
            binary_sha256=digest(args.binary.read_bytes()), observations=observations)))

    if args.divergence_only:
        check_divergences()
        return

    check_site_scope()
    if args.site_scope_only:
        return

    destination = args.work / 'artifacts'
    report = run('explore', command(destination), (0, 1))
    campaign = locate(destination)
    run('inspect', [*prefix, 'inspect', '--json', campaign])
    before = rounds['snapshot'](campaign)
    complete = json.loads(run('resume-complete', [*prefix, 'resume', '--json', campaign], (2,)))
    assert complete['classification'] == 'invalid_input' and 'complete' in complete['message'], complete
    assert rounds['snapshot'](campaign) == before
    baseline = events(campaign)
    authenticated_campaign = rounds['authenticate'](campaign)
    rounds['negative_checks'](campaign, args.work / 'independent-negatives')
    rounds['plan_refusals'](run, prefix, campaign, args.work)
    assert len(baseline) > 1, 'no actual forced prefix rounds'
    envelope = json.loads(report.splitlines()[-1])['envelope']
    assert envelope['logical_executions'] == len(baseline) <= 8, envelope
    assert envelope['deepest_prefix'] <= 2 and envelope['pending_bytes'] <= 128 << 20, envelope
    assert envelope['stop_reason'], envelope
    executions = [execution for event in baseline for execution in event['executions']]
    assert all(execution['record']['outcome']['class'] != 'runner_failure' for execution in executions)
    assert all(result['divergence'] is None for event in baseline for result in event['segment']['results'])

    interrupted_destination = args.work / 'interrupted'
    run('interrupt', command(interrupted_destination), (86,), {'GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER': '0'})
    interrupted = locate(interrupted_destination)
    assert len(events(interrupted)) == 1
    run('inspect-prefix', [*prefix, 'inspect', '--json', interrupted])
    run('resume', [*prefix, 'resume', '--json', interrupted], (0, 1))
    assert [event['segment'] for event in events(interrupted)] == [event['segment'] for event in baseline]
    assert rounds['authenticate'](interrupted)['artifacts'] == authenticated_campaign['artifacts']

    torn = args.work / 'torn'
    shutil.copytree(campaign, torn)
    journal = torn / 'journal.jsonl'
    journal.write_bytes(journal.read_bytes() + b'{"event":')
    before = journal.read_bytes()
    run('inspect-torn-refusal', [*prefix, 'inspect', '--json', torn], (2,))
    assert journal.read_bytes() == before
    complete = json.loads(run('resume-torn', [*prefix, 'resume', '--json', torn], (2,)))
    assert complete['classification'] == 'invalid_input' and 'complete' in complete['message'], complete
    assert events(torn) == baseline

    malformed = args.work / 'malformed-tail'
    shutil.copytree(campaign, malformed)
    malformed_journal = malformed / 'journal.jsonl'
    malformed_journal.write_bytes(malformed_journal.read_bytes() + b'not-json')
    before = malformed_journal.read_bytes()
    run('resume-malformed-refusal', [*prefix, 'resume', '--json', malformed], (2,))
    assert malformed_journal.read_bytes() == before

    altered = args.work / 'altered-envelope'
    shutil.copytree(campaign, altered)
    altered_journal = altered / 'journal.jsonl'
    lines = altered_journal.read_bytes().splitlines()
    first = json.loads(lines[0])
    first['ordinal'] += 1
    altered_journal.write_bytes(b'\n'.join([encode(first), *lines[1:]]) + b'\n')
    before = altered_journal.read_bytes()
    run('inspect-envelope-refusal', [*prefix, 'inspect', '--json', altered], (2,))
    run('resume-envelope-refusal', [*prefix, 'resume', '--json', altered], (2,))
    assert altered_journal.read_bytes() == before

    run('capacity-refusal', command(args.work / 'capacity', capacity='1'), (2,))
    bounded = args.work / 'execution-bound'
    run('execution-bound', command(bounded, executions=1), (0, 1))
    assert len(events(locate(bounded))) == 1
    if args.divergence:
        check_divergences()
    go_observations = None
    if args.go_baseline:
        baseline_root = args.go_baseline.resolve()
        for line in (baseline_root / 'inputs.sha256').read_text().splitlines():
            expected, relative = line.split()
            assert digest((baseline_root / relative).read_bytes()) == expected, relative
        go_binary = baseline_root / 'bin/gomad'
        go_module = args.go_source.resolve() / 'tools/gomad'
        target = 'internal/backends/wasm/toolchain/testdata/choices/main.go'
        assert (go_module / target).read_bytes() == (MODULE / target).read_bytes()
        go_manifest = args.work / 'go-installation.json'
        run('go-install', [go_binary, 'install', '--compiler', args.go_compiler,
            '--helper', baseline_root / 'bin/gomad-wasmhost', '--profile', 'gomad.wasi-cooperative/v1',
            '--runtime-root', go_module / 'internal/model/runtimesource',
            '--cache', args.work / 'go-preparation', '--fuel', str(1 << 60), '--output', go_manifest])
        go_command = command(args.work / 'go-artifacts')
        go_command[0], go_command[2] = go_binary, go_manifest
        go_command[go_command.index('--working-dir') + 1] = go_module
        go_reports = [json.loads(line) for line in run('go-explore', go_command, (0, 1)).splitlines()]
        go_report = next(row for row in go_reports if row['type'] == 'result')
        for field in ('logical_executions', 'committed_rounds', 'deepest_prefix', 'pending',
                      'omitted_by_depth', 'omitted_by_execution_bound', 'stop_reason', 'bounded_complete'):
            assert go_report['exploration'][field] == envelope[field], (field, go_report, envelope)
        go_outputs = []
        for index, row in enumerate(row for row in go_reports if row['type'] == 'artifact'):
            artifact = Path(row['path'])
            run('go-inspect-' + str(index), [go_binary, '--installation', go_manifest, 'inspect', '--json', artifact])
            go_outputs.append((artifact / 'stdout').read_text())
        rust_outputs = [(campaign / execution['artifact'] / 'output/stdout').read_text() for execution in executions]
        assert sorted(go_outputs) == sorted(rust_outputs), (go_outputs, rust_outputs)
        assert go_report['succeeded'] == len(executions), go_report
        go_observations = dict(binary_sha256=digest(go_binary.read_bytes()), report=go_report,
                               guest_outputs=sorted(go_outputs))
    (args.work / 'observations.json').write_bytes(encode(dict(schema='gomad.rust-choice-live-check/v1',
        binary_sha256=digest(args.binary.read_bytes()), driver_sha256=digest(Path(__file__).read_bytes()),
        rounds=len(baseline), executions=len(executions), envelope=envelope,
        segments=[event['segment'] for event in baseline], go_reference=go_observations)))


if __name__ == '__main__':
    main()
