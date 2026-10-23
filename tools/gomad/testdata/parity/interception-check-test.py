#!/usr/bin/env python3
"""Offline regressions for corpus coverage and supplied-input binding; no guests."""
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
from pathlib import Path
import tempfile
import unittest

loader = SourceFileLoader('interception_check', str(Path(__file__).with_name('interception-check.py')))
spec = spec_from_loader(loader.name, loader)
check = module_from_spec(spec)
loader.exec_module(check)


def case(name, seeds='7,8', status=0, json_output=False, outcomes=None):
    return dict(name=name, seeds=seeds, status=status, json=json_output,
                module='/actual/module.wasm', receipt='/actual/receipt.json', receipt_sha256='a' * 64,
                replayable=False, outcomes=outcomes or [dict(ordinal=i, seed=seed, **{'class':'success'}, termination='exit', status=0)
                                                       for i, seed in enumerate(check.selected_seeds(seeds))])


def complete_inputs():
    failure = dict(ordinal=0, seed=7, **{'class':'target_failure'}, termination='exit', status=17)
    first = case('first-failure', status=17, outcomes=[failure])
    first.update(later_would_pass_seed=8, later_outcome=dict(**{'class':'success'}, termination='exit'))
    replay = case('failure-with-replay', status=17, outcomes=[failure])
    replay['replayable'] = True
    unprepared = dict(name='unprepared-module', module='/unprepared/module.wasm', seeds='1',
                      status=2, json=False, replayable=False, outcomes=[])
    infra = case('infrastructure', status=3, outcomes=[dict(ordinal=0, seed=7, **{'class':'runner_failure'}, termination='fuel', status=3)])
    cases = [case('success'), first, replay, unprepared, infra, case('standalone-json', json_output=True)]
    return dict(schema='gomad.interception-corpus-inputs/v1', cases=cases)


def seal_artifact(root, record, payloads):
    payloads = dict(payloads, **{'record.json': check.parity.canonical(record)})
    manifest = dict(schema='gomad.artifact-manifest/v2', record=record,
                    record_sha256=check.parity.digest(payloads['record.json']),
                    files={name: dict(bytes=len(raw), sha256=check.parity.digest(raw),
                                      module=name == 'target/module.wasm') for name, raw in payloads.items()})
    for name, raw in dict(payloads, **{'manifest.json':check.parity.canonical(manifest),
                                     'storage.json':check.parity.canonical(dict(format='gomad.artifact-storage/v2'))}).items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(raw)
    return sum(path.stat().st_size for path in root.rglob('*') if path.is_file())


def sealed_fixture(root, work='/short', pid=7, launch_id='c' * 64):
    module = b'\0asm\x01\0\0\0'
    limits = dict(fuel=1000, memory_bytes=1 << 20, wall_millis=1000, output_bytes=1 << 20, evidence_bytes=16 << 20)
    configuration = dict(limits=limits)
    provenance = dict(schema='gomad.rust-wasm-provenance/v1', inputs={}, build_key='4' * 64,
                      module_sha256=check.parity.digest(module), configuration=configuration,
                      choice_identity=None, scenario=None, network=None)
    invocation = dict(schema='gomad.intercepted-invocation/v1', wrapper_run_id='a' * 64, invocation_id='b' * 64,
                      original_argv=['676f'], driver_argv=['676f'], compiler_sha256='1' * 64,
                      installation_sha256='2' * 64, recipe_sha256='3' * 64, recipe_file='driver-recipe.json')
    origin = dict(schema='gomad.intercepted-origin/v1', origin='intercepted-go', invocation=invocation,
                  module_sha256=check.parity.digest(module), preparation_sha256=check.parity.digest(check.parity.canonical(provenance)),
                  launch=dict(invocation_id=launch_id, argv=[(work + '/module.wasm').encode().hex()],
                              process_id=pid, parent_process_id=pid + 1))
    provenance['origin'] = origin
    origin_bytes = check.parity.canonical(origin)
    identity = dict(target=check.parity.digest(module), toolchain='4' * 64, platform='5' * 64,
                    implementation='6' * 64, configuration=check.parity.digest(check.parity.canonical(configuration)))
    record = dict(schema='gomad.execution-record/v2', identity=identity,
                  execution_identity=check.parity.digest(check.parity.canonical(identity)), profile='gomad.wasi-stock-exploratory/v1',
                  seed=7, arguments=['gomad-target'], environment={}, limits=limits, virtual_time_nanos=0,
                  outcome=dict(**{'class':'success'}, termination='exit', exit_code=0, reason=''),
                  streams={'target/intercepted-origin.json':dict(bytes=len(origin_bytes), sha256=check.parity.digest(origin_bytes), records=1)})
    payloads = {'target/module.wasm':module, 'target/provenance.json':check.parity.canonical(provenance),
                'target/intercepted-origin.json':origin_bytes}
    stored = seal_artifact(root, record, payloads)
    return origin, record, payloads, stored


def sealed_campaign(root, work='/short', pid=7):
    origin, record, _, stored = sealed_fixture(root / 'execution', work, pid)
    sealed_fixture(root / 'prepared', work, pid)
    configuration = dict(limits=record['limits'])
    plan = dict(schema='gomad.seed-campaign-plan/v2', selection='7', selection_count=1, parallel=1,
                shard=None, journal_bytes=1 << 20, prepared_artifact='prepared', module_sha256=record['identity']['target'],
                toolchain=record['identity']['toolchain'], configuration=configuration, origin=origin, scenario=None)
    raw = check.parity.canonical(plan)
    (root / 'plan.json').write_bytes(raw)
    (root / 'storage.json').write_bytes(check.parity.canonical(dict(format='gomad.campaign-storage/v2',
                                                                 plan_sha256=check.parity.digest(raw), max_bytes=1 << 20)))
    execution = dict(ordinal=0, seed=7, artifact='execution', record=record, stored_bytes=stored)
    events = [dict(kind='admit', jobs=[dict(ordinal=0, seed=7)]), dict(kind='receipt', execution=execution),
              dict(kind='outcome', executions=[execution], admitted=[], stop_reason=None), dict(kind='complete')]
    previous, journal = check.parity.digest(raw), []
    for ordinal, event in enumerate(events):
        row = dict(ordinal=ordinal, previous_sha256=previous, event=event)
        previous = check.parity.digest(check.parity.canonical(row))
        row['sha256'] = previous
        journal.append(check.parity.canonical(row))
    (root / 'journal.jsonl').write_bytes(b'\n'.join(journal) + b'\n')
    return origin, execution


class SealedOriginAndGolden(unittest.TestCase):
    def test_valid_sealed_artifact_cannot_borrow_another_plan_origin(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            origin, record, _, _ = sealed_fixture(root / 'first')
            other, other_record, _, _ = sealed_fixture(root / 'other', launch_id='d' * 64)
            self.assertEqual(check.campaign.authenticate_artifact(root / 'other'), other_record)
            self.assertEqual(check.authenticate_origin(root / 'first', record, origin), origin)
            with self.assertRaisesRegex(ValueError, 'plan_origin'):
                check.authenticate_origin(root / 'other', other_record, origin)

    def test_unbound_origin_provenance_and_missing_launch_are_refused(self):
        for mutation in ['stream', 'provenance', 'launch']:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                origin, record, payloads, _ = sealed_fixture(root)
                if mutation == 'stream':
                    record['streams'].clear()
                elif mutation == 'provenance':
                    provenance = check.parity.decode(payloads['target/provenance.json'])
                    provenance['build_key'] = '9' * 64
                    payloads['target/provenance.json'] = check.parity.canonical(provenance)
                else:
                    del origin['launch']
                    raw = check.parity.canonical(origin)
                    payloads['target/intercepted-origin.json'] = raw
                    provenance = check.parity.decode(payloads['target/provenance.json'])
                    provenance['origin'] = origin
                    payloads['target/provenance.json'] = check.parity.canonical(provenance)
                    record['streams']['target/intercepted-origin.json'].update(bytes=len(raw), sha256=check.parity.digest(raw))
                seal_artifact(root, record, payloads)
                check.campaign.authenticate_artifact(root)
                with self.assertRaises((ValueError, check.parity.Refusal)):
                    check.authenticate_origin(root, record, origin)

    def test_golden_ignores_authenticated_path_pid_width_accounting_only(self):
        with tempfile.TemporaryDirectory() as temporary:
            views, raw_bytes = [], []
            for name, work, pid in [('short', '/x', 7), ('long', '/a/much/longer/absolute/work/root', 123456789)]:
                root = Path(temporary) / name
                _, execution = sealed_campaign(root, work, pid)
                checked = check.campaign.authenticate_campaign(root)
                self.assertEqual(checked['executions'][0]['stored_bytes'], execution['stored_bytes'])
                raw_bytes.append(execution['stored_bytes'])
                manifest = check.campaign.canonical_file(root / 'execution/manifest.json')
                row = dict(type='outcome', ordinal=0, seed=7, status=0, stored_bytes=execution['stored_bytes'],
                           record_sha256=manifest['record_sha256'], origin_sha256=execution['record']['streams']['target/intercepted-origin.json']['sha256'])
                views.append(check.stable_rows([row], {}))
            self.assertNotEqual(*raw_bytes)
            self.assertEqual(*views)
            root = Path(temporary) / 'short'
            previous = check.parity.digest((root / 'plan.json').read_bytes())
            journal = []
            for line in (root / 'journal.jsonl').read_bytes().splitlines():
                row = check.parity.decode(line)
                event = row['event']
                if event['kind'] == 'receipt':
                    event['execution']['stored_bytes'] += 1
                elif event['kind'] == 'outcome':
                    event['executions'][0]['stored_bytes'] += 1
                row['previous_sha256'] = previous
                del row['sha256']
                previous = check.parity.digest(check.parity.canonical(row))
                row['sha256'] = previous
                journal.append(check.parity.canonical(row))
            (root / 'journal.jsonl').write_bytes(b'\n'.join(journal) + b'\n')
            with self.assertRaisesRegex(ValueError, 'retained.logical_byte_accounting'):
                check.campaign.authenticate_campaign(root)


class CoverageAndBinding(unittest.TestCase):
    def validate(self, inputs):
        declaration = check.document(Path(__file__).with_name('interception-corpus.json'))
        check.validate_cases(inputs, declaration)

    def test_complete_distinct_semantics_are_accepted_without_running_commands(self):
        self.validate(complete_inputs())

    def test_six_successful_aliases_cannot_claim_the_corpus(self):
        for name, mutate in [
            ('standalone-json', lambda value: value.__setitem__('json', False)),
            ('infrastructure', lambda value: value.update(status=0, outcomes=case('success')['outcomes'])),
            ('first-failure', lambda value: value.pop('later_would_pass_seed')),
            ('first-failure', lambda value: value.__setitem__('later_would_pass_seed', 7)),
            ('first-failure', lambda value: value.__setitem__('later_outcome', dict(**{'class':'target_failure'}, termination='exit'))),
            ('failure-with-replay', lambda value: value.__setitem__('replayable', False)),
        ]:
            inputs = complete_inputs()
            mutate(next(value for value in inputs['cases'] if value['name'] == name))
            with self.subTest(name=name, mutation=mutate), self.assertRaises(ValueError):
                self.validate(inputs)

    def test_gaps_and_executions_after_failure_are_refused(self):
        inputs = complete_inputs()
        first = next(value for value in inputs['cases'] if value['name'] == 'first-failure')
        first['outcomes'].append(dict(ordinal=1, seed=8, **{'class':'success'}, termination='exit', status=0))
        with self.assertRaises(ValueError):
            self.validate(inputs)

    def test_a_valid_other_artifact_does_not_bind_supplied_module(self):
        module = b'supplied actual module bytes'
        sha = check.parity.digest(module)
        record = dict(identity=dict(target=sha))
        manifest = dict(files={'target/module.wasm':dict(sha256=sha, bytes=len(module))})
        check.bind_module_input(record, manifest, sha, len(module))
        for changed_record, changed_manifest in [
            (dict(identity=dict(target='b' * 64)), manifest),
            (record, dict(files={'target/module.wasm':dict(sha256='b' * 64, bytes=len(module))})),
            (record, dict(files={'target/module.wasm':dict(sha256=sha, bytes=len(module) + 1)})),
        ]:
            with self.subTest(), self.assertRaises(ValueError):
                check.bind_module_input(changed_record, changed_manifest, sha, len(module))

    def test_unbounded_or_duplicate_selections_are_refused(self):
        for value in ['1-18446744073709551615', '7,7', '-1', '8-7', '7,,8']:
            with self.subTest(value=value), self.assertRaises(ValueError):
                check.selected_seeds(value)

    def test_preparation_and_execution_are_distinct_authenticated_roles(self):
        prepared, executed = Path('/prepared'), Path('/execution')
        self.assertEqual(check.campaign_roles([(prepared, {}), (executed, dict(launcher=True))]),
                         (executed, prepared))
        for plans in [[(executed, dict(launcher=True))], [(prepared, {}), (executed, {})],
                      [(prepared, {}), (executed, dict(launcher=True)), (Path('/other'), {})],
                      [(prepared, {}), (executed, dict(launcher=1))]]:
            with self.subTest(plans=plans), self.assertRaises(ValueError):
                check.campaign_roles(plans)


if __name__ == '__main__':
    unittest.main()
