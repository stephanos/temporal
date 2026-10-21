#!/usr/bin/env python3
"""Exercise public choice exploration with real compiled guests and durable rounds."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess

MODULE = Path(__file__).resolve().parents[2]


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
    args = parser.parse_args()
    assert bool(args.go_baseline) == bool(args.go_source), 'provide both Go baseline and frozen source'
    args.binary, args.go_compiler, args.input, args.work = (
        path.resolve() for path in (args.binary, args.go_compiler, args.input, args.work))
    args.work.mkdir(parents=True, exist_ok=False)
    environment = {name: value for name, value in os.environ.items() if not name.startswith('GOMAD_')}
    receipts = []

    def run(name, command, statuses=(0,), extra=None):
        result = subprocess.run(list(map(str, command)), env=dict(environment, **(extra or {})),
                                capture_output=True, timeout=900)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
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
    installation = dict(schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
        compiler_sha256=digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
        runtime_root=str(MODULE / 'internal/model/runtimesource'), cache_root=str(args.work / 'preparation'),
        profile='gomad.wasi-cooperative/v1', fuel=1 << 60, memory_bytes=256 << 20,
        helper_sha256=replay['helper_sha256'], host_sha256=replay['host_sha256'],
        runtime_implementation_sha256=replay['runtime_implementation_sha256'], model=model,
        namespace=configuration['namespace'], storage=None, network=None, build_tags=[])
    manifest = args.work / 'installation.json'
    manifest.write_bytes(encode(installation))
    prefix = [args.binary, '--installation', manifest]

    def command(destination, executions=8, capacity='128MiB'):
        return [*prefix, 'explore', '--json', '--strategy=choice-exploration', '--choices',
            '--on-failure=all', '--parallel=1', '--seeds=89', '--max-executions=' + str(executions),
            '--max-choice-depth=2', '--max-exploration-bytes=' + capacity,
            '--keep-successes=all', '--success-limit=32', '--success-bytes=512MiB',
            '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', destination,
            '--working-dir', MODULE, 'go-run', './internal/backends/wasm/toolchain/testdata/choices']

    def locate(destination):
        roots = list((destination / 'campaigns').glob('campaign-*'))
        assert len(roots) == 1, roots
        return roots[0]

    def events(root):
        return [json.loads(line)['event'] for line in (root / 'journal.jsonl').read_bytes().splitlines()]

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
            for field in ('helper_sha256', 'host_sha256', 'runtime_implementation_sha256'):
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

    destination = args.work / 'artifacts'
    report = run('explore', command(destination), (0, 1))
    campaign = locate(destination)
    run('inspect', [*prefix, 'inspect', '--json', campaign])
    run('resume-complete', [*prefix, 'resume', '--json', campaign], (0, 1))
    baseline = events(campaign)
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

    torn = args.work / 'torn'
    shutil.copytree(campaign, torn)
    journal = torn / 'journal.jsonl'
    journal.write_bytes(journal.read_bytes() + b'{"event":')
    before = journal.read_bytes()
    run('inspect-torn-refusal', [*prefix, 'inspect', '--json', torn], (2,))
    assert journal.read_bytes() == before
    run('resume-torn', [*prefix, 'resume', '--json', torn], (0, 1))
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
