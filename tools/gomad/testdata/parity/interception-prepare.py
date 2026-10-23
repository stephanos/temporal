#!/usr/bin/env python3
"""Print shared preparation/driver commands or collect genuine retained corpus inputs.

This tool launches no commands. Run each emitted command only in its granted
resource window and save exporter stdout at the declared export_stdout path.
"""
import argparse
from pathlib import Path

from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader

loader = SourceFileLoader('interception_prepare_check', str(Path(__file__).with_name('interception-check.py')))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)


def plan(binary, installation, source, work):
    waves = []
    for name, target_args, limit in [('success', [], None), ('failure', ['seed-failure'], None),
                                     ('infrastructure', [], '1'), ('callback', ['stream'], None)]:
        root = work / name
        argv = [binary, 'fixture-driver-prepare', '--output', root / 'invocation',
                '--installation', installation, '--choices', '--seeds', '1',
                '--execution-timeout', '5m', '--overall-timeout', '10m',
                '--artifacts', root / 'artifacts', '--working-dir', source / 'tools/gomad',
                '--build-tag', 'test_dep']
        if limit is not None:
            argv += ['--output-limit', limit]
        argv += ['go-run', './cmd/gomad/testdata/seedfree', '--', *target_args]
        waves.append(dict(name=name, export_argv=list(map(str, argv)),
                          export_stdout=str(root / 'export.stdout'), artifacts=str(root / 'artifacts')))
    return waves


def driver_command(binary, wave):
    exported = check.document(Path(wave['export_stdout']))
    check.equal(exported['guest_executions'], 0, 'export.zero_guests')
    invocation = Path(exported['invocation'])
    check.equal(invocation, Path(wave['export_stdout']).parent / 'invocation/invocation.json', 'export.invocation_path')
    raw = invocation.read_bytes()
    check.equal(check.parity.digest(raw), exported['invocation_sha256'], 'export.invocation_digest')
    bound = check.campaign.canonical_file(invocation)
    check.equal([bytes.fromhex(value).decode() for value in bound['driver_argv']],
                exported['driver_argv'], 'export.exact_driver_argv')
    return list(map(str, [binary, 'fixture-driver-run', '--invocation', invocation,
                          '--sha256', exported['invocation_sha256']]))


def retained_input(root):
    receipts = list((root / 'admissions').glob('*/*/plan.json.preparation.json'))
    check.equal(len(receipts), 1, 'one_actual_admission_receipt')
    path = receipts[0].resolve()
    receipt = check.campaign.canonical_file(path)
    check.equal((receipt['schema'], receipt['guest_executions'], receipt['exit_code']),
                ('gomad.rust-preparation-receipt/v1', 0, 0), 'receipt.zero_guest_admission')
    prepared = check.campaign.authenticate_campaign(Path(receipt['campaign']), complete=False)
    check.equal((prepared['journal'], prepared['executions']), ([], []), 'receipt.zero_guest_campaign')
    check.equal(check.parity.digest(Path(receipt['plan']).read_bytes()), receipt['plan_sha256'], 'receipt.plan_digest')
    published = check.campaign.canonical_file(Path(receipt['plan']))
    check.equal(published, prepared['plan'], 'receipt.published_plan')
    check.equal(receipt['prepared_artifact'], prepared['plan']['prepared_artifact'], 'receipt.prepared_artifact')
    artifact = Path(receipt['campaign']) / receipt['prepared_artifact']
    record = check.campaign.authenticate_artifact(artifact)
    manifest = check.campaign.canonical_file(artifact / 'manifest.json')
    check.equal(manifest['record_sha256'], receipt['record_sha256'], 'receipt.record_digest')
    check.authenticate_origin(artifact, record, prepared['plan']['origin'])
    for prefix, filename in [('module', 'target/module.wasm'), ('provenance', 'target/provenance.json')]:
        check.equal(receipt[prefix + '_file'], filename, 'receipt.payload_path')
        raw = (artifact / filename).read_bytes()
        check.equal((len(raw), check.parity.digest(raw)),
                    (receipt[prefix + '_bytes'], receipt[prefix + '_sha256']), 'receipt.payload_binding')
    return dict(module=str(artifact / receipt['module_file']), receipt=str(path),
                receipt_sha256=check.parity.digest(path.read_bytes()), args=record['arguments'][1:])


def collect_wave(wave):
    root = Path(wave['artifacts'])
    result = retained_input(root)
    executed, prepared = check.campaign_roles([(path, check.campaign.canonical_file(path / 'plan.json'))
                                               for path in (root / 'campaigns').glob('campaign-*')])
    observed = check.campaign.authenticate_campaign(executed)
    check.equal((observed['plan']['selection'], len(observed['executions'])), ('1', 1), 'driver.single_probe')
    execution = observed['executions'][0]
    artifact = executed / execution['artifact']
    record = check.campaign.authenticate_artifact(artifact)
    sealed = check.authenticate_origin(artifact, record, observed['plan']['origin'])
    for campaign_root in [prepared, executed]:
        plan = check.campaign.canonical_file(campaign_root / 'plan.json')
        payload = campaign_root / plan['prepared_artifact']
        bound = check.campaign.authenticate_artifact(payload)
        check.equal(check.authenticate_origin(payload, bound, plan['origin']), sealed, 'driver.shared_origin')
    provenance = check.campaign.canonical_file(artifact / 'target/provenance.json')
    check.equal((provenance['inputs']['kind'], provenance['inputs']['source']),
                ('go-run', './cmd/gomad/testdata/seedfree'), 'driver.registered_source')
    check.equal((artifact / 'target/module.wasm').read_bytes(), Path(result['module']).read_bytes(), 'driver.actual_module')
    expected = ('runner_failure', 'capacity', 3) if wave['name'] == 'infrastructure' else ('success', 'exit', 0)
    check.equal((record['outcome']['class'], record['outcome']['termination'], check.mapped_status(record['outcome'])),
                expected, 'driver.actual_probe_outcome')
    check.equal(record['limits']['output_bytes'], 1 if wave['name'] == 'infrastructure' else 8 << 20,
                'driver.original_output_limit')
    return result


def corpus_inputs(prepared):
    def outcome(seed, classification='success', termination='exit', status=0):
        return dict(ordinal=0, seed=seed, **{'class':classification}, termination=termination, status=status)

    def case(name, recipe, seeds, rows, json=False, replayable=True):
        return dict(prepared[recipe], name=name, seeds=seeds, outcomes=rows,
                    status=rows[-1]['status'], json=json, replayable=replayable)

    success = [outcome(1), dict(outcome(3), ordinal=1)]
    failure = [outcome(2, 'target_failure', status=17)]
    first = case('first-failure', 'failure', '2,1', failure)
    first.update(later_would_pass_seed=1, later_outcome=dict(**{'class':'success'}, termination='exit'))
    inputs = dict(schema='gomad.interception-corpus-inputs/v1', cases=[
        case('success', 'success', '1,3', success), first,
        case('failure-with-replay', 'failure', '2', failure),
        dict(name='unprepared-module', module=prepared['success']['module'], args=[], seeds='1',
             status=2, json=False, replayable=False, outcomes=[]),
        case('infrastructure', 'infrastructure', '1,3', [outcome(1, 'runner_failure', 'capacity', 3)], replayable=False),
        case('standalone-json', 'success', '1,3', success, json=True)], callback=prepared['callback'])
    check.validate_cases(inputs, check.document(check.HERE / 'interception-corpus.json'))
    return inputs


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('stage', choices=['plan', 'initialize', 'drivers', 'collect'])
    for name in ['binary', 'installation', 'source', 'work']:
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    waves = plan(*(getattr(args, name).resolve() for name in ['binary', 'installation', 'source', 'work']))
    if args.stage == 'initialize':
        args.work.resolve().mkdir(mode=0o700, parents=True, exist_ok=False)
        for wave in waves:
            Path(wave['export_stdout']).parent.mkdir(mode=0o700)
    if args.stage in ('plan', 'initialize'):
        result = dict(schema='gomad.interception-preparation-commands/v1', waves=waves,
                      environment=dict(GOMAXPROCS='2'), execution_grant_required=True)
    elif args.stage == 'drivers':
        result = dict(schema='gomad.interception-driver-commands/v1', waves=[
            dict(name=wave['name'], argv=driver_command(args.binary.resolve(), wave),
                 expected_status=1 if wave['name'] == 'infrastructure' else 0,
                 timeout_seconds=600, cleanup_seconds=5) for wave in waves])
    else:
        result = corpus_inputs({wave['name']: collect_wave(wave) for wave in waves})
    print(check.parity.canonical(result).decode())


if __name__ == '__main__':
    main()
