#!/usr/bin/env python3
"""Offline regressions for corpus coverage and supplied-input binding; no guests."""
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
from pathlib import Path
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


if __name__ == '__main__':
    unittest.main()
