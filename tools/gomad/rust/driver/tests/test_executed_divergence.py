"""Pure original-counterfactual receiving refusal controls; no guests or builds."""
import copy
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import executed_divergence as gate
from driver import Refusal


class OriginalExecutedDivergence(unittest.TestCase):
    def test_setup_rejects_a_guest_compiled_before_its_baseline_deadline(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'prepared').mkdir()
            cli = dict(host_sha256='host', helper_sha256='fixture-helper')
            cases = []
            for name in gate.CASES:
                installation = root / 'prepared' / (name + '-installation.json')
                cache = root / 'prepared/cache' / name
                cache.mkdir(parents=True)
                gate.write(installation, dict(cache_root=str(cache), **cli))
                cases.append(dict(name=name, campaign=dict(installation=str(installation)), original_requested_choice_bytes=0 if name=='handoff' else 4<<20))
            gate.write(root / 'prepared/producer-input.json', dict(schema='gomad.core-counterfactual-producer/v2', cases=cases))
            observed = gate.setup(root, cli, True)
            self.assertEqual(len(observed['case_caches']), 4)
            self.assertEqual(observed['guest_compilations'], 0)
            (root / 'prepared/cache/draw/module.wasm').write_bytes(b'precompiled guest')
            with self.assertRaisesRegex(Refusal,'outside its baseline deadline'):gate.setup(root, cli, True)
            self.assertEqual(observed, gate.setup(root, cli, False))

    def test_dependency_cache_is_explicit_private_and_required_before_execution(self):
        env = gate.environment(Path('/private'), Path('/compiler/bin/go'))
        self.assertEqual(env['GOMODCACHE'], '/private/module-cache')
        self.assertEqual(env['HOME'], '/private/home')
        self.assertEqual(env['GOPROXY'], 'off')
        self.assertNotIn('GOCACHE', env)
        inventory = {'dependency.go': dict(bytes=1, sha256='pinned')}
        recipe = dict(module_cache_inventory=inventory)
        with patch.object(gate.installed_cluster_cache,'verify',return_value=inventory) as verifier, patch.object(gate.live,'inventory',return_value=inventory):
            gate.dependency_cache(Path('/private'),recipe,Path('/repo'))
            verifier.assert_called_once_with(Path('/private/module-cache'),Path('/repo/tools/gomad/go.sum'))
        with patch.object(gate.installed_cluster_cache,'verify',return_value=inventory), patch.object(gate.live,'inventory',return_value={}):
            with self.assertRaises(Refusal):gate.dependency_cache(Path('/private'),recipe,Path('/repo'))

    def test_completed_original_failure_is_retained_and_live_process_substitution_refused(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            owner = dict(pid=70, repo='/repo', start_ns=99, started_monotonic_nanos=40)
            row = dict(pid=71, parent_pid=70, argv=['fixture-libtest', '--exact', 'original'], cwd='/repo',
                       start_ns=100, end_ns=200, started_monotonic_nanos=50, reaped_monotonic_nanos=150,
                       exit=101, observation='completed', clock='CLOCK_MONOTONIC', environment_mode='exact', effective_environment={'GOENV':'off'})
            process = dict(row, stage_pid=70, active=False)
            gate.write(root / 'produce.command.json', row)
            gate.write(root / 'produce.process.json', process)
            (root / 'produce.stdout').write_bytes(b'test result: FAILED. 0 passed; 1 failed; 0 ignored;\n')
            retained = gate.command_result(root, 'produce', row['argv'], {'GOENV':'off'}, owner, (0,101), set())
            self.assertEqual(retained['exit'],101)
            with self.assertRaises(Refusal):gate.command_result(root, 'produce', row['argv'], {'GOENV':'off'}, owner, (0,), set())
            (root / 'produce.process.json').unlink()
            gate.write(root / 'produce.process.json', dict(process, active=True))
            with self.assertRaises(Refusal):gate.command_result(root, 'produce', row['argv'], {'GOENV':'off'}, owner, (0,101), set())

    def test_selected_libtest_requires_one_actual_test_and_never_credits_zero_or_many(self):
        gate.selected_test(b'test result: ok. 1 passed; 0 failed; 0 ignored; 700 filtered out;\n', 0)
        gate.selected_test(b'test result: FAILED. 0 passed; 1 failed; 0 ignored; 700 filtered out;\n', 101)
        for output, status in [
            (b'test result: ok. 0 passed; 0 failed; 0 ignored;', 0),
            (b'test result: ok. 11 passed; 0 failed; 0 ignored;', 0),
            (b'test result: ok. 1 passed; 0 failed; 1 ignored;', 0),
            (b'test result: FAILED. 0 passed; 11 failed; 0 ignored;', 101),
            (b'test result: ok. 1 passed; 0 failed; 0 ignored;', 101),
        ]:
            with self.subTest(output=output), self.assertRaises(Refusal):gate.selected_test(output, status)

    def test_closed_roles_must_share_actual_source_compiler_and_both_compiled_identities(self):
        cli = dict(source_sha256='s', head='h', symlinks={}, host_sha256='host', helper_sha256='fixture-helper',
                   runtime_implementation_sha256='runtime', context={'compiler': 'pinned'}, binary='/cli', binary_sha256='cli-bytes')
        harness = dict(cli, binary='/libtest', binary_sha256='libtest-bytes')
        with patch.object(gate.main_cli_build, 'read_build', side_effect=[cli, harness]) as reader:
            gate.roles(Path('/cli-receipt'), Path('/harness-receipt'))
            self.assertEqual([call.kwargs['expected_role'] for call in reader.call_args_list],
                             ['counterfactual-fixture', 'counterfactual-harness'])
        for key in ('source_sha256', 'head', 'symlinks', 'host_sha256', 'helper_sha256', 'runtime_implementation_sha256', 'context'):
            changed = copy.deepcopy(harness)
            changed[key] = 'other'
            with self.subTest(key=key), patch.object(gate.main_cli_build, 'read_build', side_effect=[cli, changed]), self.assertRaises(Refusal):
                gate.roles('/cli-receipt', '/harness-receipt')

    def test_retained_evidence_changes_are_detected_without_sealing_reproducible_build_caches(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'prepared/cache').mkdir(parents=True)
            (root / 'prepared/cache/object').write_bytes(b'compiler cache')
            (root / 'prepared/runtime').mkdir()
            source = root / 'prepared/runtime/gomad_semantics.go.txt'
            source.write_bytes(b'actual original runtime')
            first = gate.retained_tree(root)
            self.assertNotIn('prepared/cache/object', first)
            self.assertIn('prepared/runtime/gomad_semantics.go.txt', first)
            source.write_bytes(b'changed runtime')
            self.assertNotEqual(first, gate.retained_tree(root))
            source.unlink();source.symlink_to('/outside/runtime')
            with self.assertRaises(Refusal):gate.retained_tree(root)


class NarrowAssertionLineage(unittest.TestCase):
    def report(self):
        import counterfactual_lineage as lineage
        failures = []
        for artifact in ('baseline', 'fault'):
            peer = dict(handle=dict(node='b', incarnation=1), termination='stop', exit_code=None,
                        interrupted=True, reason='WASI stop: cluster failed')
            failures.append(dict(original_root=lineage.ROOT + '/handoff', predicate=lineage.ASSERTION['predicate'],
                artifact=artifact, handle=peer['handle'], expected='exit', actual='stop', observation=peer))
        return dict(status='FAIL_ORIGINAL_PREDICATE', original_predicate_failures=failures,
            pairs=[dict(name=name, baseline=dict(artifact='baseline'), fault=dict(artifact='fault')) for name in gate.CASES])

    def test_only_exact_authorized_peer_assertion_has_superseded_applicability(self):
        import counterfactual_lineage as lineage
        report = self.report()
        disposition = lineage.reconcile(report)
        self.assertEqual(disposition['original_status'], 'FAIL_ORIGINAL_PREDICATE')
        self.assertEqual(disposition['owner_reference']['requirement'], 'R2')
        self.assertEqual(disposition['assertion']['case'], 'handoff')
        self.assertNotIn('PASS', disposition.values())
        for key, value in [('original_root', 'another-test'), ('predicate', 'missing replay'),
                           ('handle', dict(node='a', incarnation=1)), ('actual', 'watchdog')]:
            changed = copy.deepcopy(report)
            changed['original_predicate_failures'][0][key] = value
            with self.subTest(key=key), self.assertRaises(Refusal):lineage.reconcile(changed)
        for key, value in [('reason', 'watchdog'), ('interrupted', False), ('exit_code', 1)]:
            changed = copy.deepcopy(report)
            changed['original_predicate_failures'][0]['observation'][key] = value
            with self.subTest(key=key), self.assertRaises(Refusal):lineage.reconcile(changed)
        for mutation in ('original-pass', 'missing-case', 'extra-failure', 'missing-fault'):
            changed = copy.deepcopy(report)
            if mutation == 'original-pass':changed['status'] = 'PASS'
            if mutation == 'missing-case':changed['pairs'].pop()
            if mutation == 'extra-failure':changed['original_predicate_failures'].append(copy.deepcopy(changed['original_predicate_failures'][0]))
            if mutation == 'missing-fault':changed['original_predicate_failures'].pop()
            with self.subTest(mutation=mutation), self.assertRaises(Refusal):lineage.reconcile(changed)
    def test_final_map_cannot_apply_the_lineage_to_another_root_or_omit_a_pair(self):
        import counterfactual_lineage as lineage
        applicability = lineage.reconcile(self.report())
        key = lineage.SOURCE + '#' + lineage.ROOT
        entries = {key: dict(evidence=sorted('original-divergence/' + name for name in gate.CASES),
                             assertion_applicability=applicability)}
        entries.update({key + '@9714/' + case: dict(evidence=['original-divergence/' + case]) for case in gate.CASES})
        lineage.mapping(entries, applicability)
        for mutation in ('another-root', 'one-pair', 'authority', 'original-pass', 'missing-subcase'):
            changed = copy.deepcopy(entries)
            if mutation == 'another-root':changed['another-test'] = dict(assertion_applicability=applicability)
            if mutation == 'one-pair':changed[key]['evidence'].pop()
            if mutation == 'missing-subcase':del changed[key + '@9714/handoff']
            if mutation == 'authority':changed[key]['assertion_applicability']['owner_reference']['requirement'] = 'unowned'
            if mutation == 'original-pass':changed[key]['assertion_applicability']['original_status'] = 'PASS'
            with self.subTest(mutation=mutation), self.assertRaises(Refusal):lineage.mapping(changed, applicability)



class SameStageCounterfactual(unittest.TestCase):
    def test_endpoint_has_fixed_owning_stage_and_explicit_inputs(self):
        import counterfactual_receiving as receiving
        config = dict(root='/evidence', live_corpus=dict(test_wasm_reach={}))
        with self.assertRaises(Refusal):receiving.selection(config)
        selected = dict(cli_build='/cli.json', harness_build='/harness.json', compiler_root='/compiler', module_cache='/modules')
        config['live_corpus']['test_wasm_reach']['executed_divergence'] = selected
        _, stage, root = receiving.selection(config)
        self.assertEqual(stage, Path('/evidence/stages/cluster-functional'))
        self.assertEqual(root, stage / 'executed-divergence')
        for changed in (dict(selected, root='/old-result'), dict(selected, cli_build='relative')):
            config['live_corpus']['test_wasm_reach']['executed_divergence'] = changed
            with self.assertRaises(Refusal):receiving.selection(config)

    def test_absent_endpoint_is_unexecuted_without_launch_or_receipts(self):
        import counterfactual_receiving as receiving
        import broad_stages
        config = dict(root='/evidence', live_corpus=dict(test_wasm_reach={}))
        with patch.object(receiving, 'guard'), patch.object(receiving, 'inputs') as inputs, \
                patch.object(receiving, 'command') as command, patch.object(receiving, 'read') as read, \
                patch.object(broad_stages, 'read', return_value={}) as broad:
            for reader in (receiving.run, receiving.observations, receiving.final_read):
                with self.subTest(reader=reader.__name__):
                    facts = reader(config)
                    self.assertEqual(facts['status'], 'UNEXECUTED')
                    self.assertFalse(facts['executed'])
                    self.assertFalse(facts['credited'])
                    self.assertEqual(facts['actual_cases'], [])
                    self.assertIn('fn-232.14', facts['gap'])
            broad.assert_called_once()
            inputs.assert_not_called(); command.assert_not_called(); read.assert_not_called()
            for reader in (receiving.run, receiving.observations, receiving.final_read):
                with patch.object(gate.live, 'REACH_GAPS_BLOCK', True), self.assertRaises(Refusal):
                    reader(config)

    def test_declared_invalid_endpoint_never_becomes_optional_absence(self):
        import counterfactual_receiving as receiving
        import broad_stages
        selected = dict(cli_build='/cli', harness_build='/harness', compiler_root='/compiler', module_cache='/modules')
        config = dict(root='/evidence', live_corpus=dict(test_wasm_reach={}))
        with patch.object(receiving, 'guard'), patch.object(broad_stages, 'read', return_value={}):
            for invalid in (None, {}, [], 'missing', dict(selected, cli_build='relative'), dict(selected, extra='/other')):
                config['live_corpus']['test_wasm_reach']['executed_divergence'] = invalid
                for reader in (receiving.run, receiving.observations, receiving.final_read):
                    with self.subTest(invalid=invalid, reader=reader.__name__), self.assertRaises(Refusal):
                        reader(config)
            config['live_corpus']['test_wasm_reach']['executed_divergence'] = selected
            with patch.object(receiving, 'inputs', side_effect=Refusal('declared corrupt source/config/role')):
                for reader in (receiving.run, receiving.observations):
                    with self.subTest(reader=reader.__name__), self.assertRaisesRegex(Refusal, 'declared corrupt'):
                        reader(config)

    def test_child_receives_explicit_config_and_verified_loader_precedes_production(self):
        import counterfactual_receiving as receiving
        selected = dict(cli_build='/cli', harness_build='/harness', compiler_root='/compiler', module_cache='/modules')
        config = dict(python='/python', _config_path='/sealed/config.json')
        argv = receiving.argv(config, selected, Path('/out'))
        self.assertEqual(argv[argv.index('--config') + 1], config['_config_path'])
        arguments = ['executed_divergence.py', 'produce', *argv[3:]]
        loaded = dict(dry_run=True)
        with patch.object(sys, 'argv', arguments), patch.object(gate, 'configured', return_value=loaded) as loader, \
                patch.object(gate, 'sha', return_value='sealed'), patch.object(gate, 'produce') as produce:
            gate.main()
            loader.assert_called_once_with(Path('/sealed/config.json'))
            self.assertIs(produce.call_args.kwargs['frozen'], loaded)
            self.assertEqual(loaded['_config_sha256'], 'sealed')
        with patch.object(sys, 'argv', arguments), patch.object(gate, 'configured', side_effect=Refusal('invalid sealed config')), \
                patch.object(gate, 'produce') as produce:
            with self.assertRaisesRegex(Refusal, 'invalid sealed config'): gate.main()
            produce.assert_not_called()

    def test_environment_cannot_admit_dry_run_counterfactual_roles(self):
        import dry_run
        with patch.object(dry_run, 'is_record', return_value=True), \
                patch.object(gate.main_cli_build, 'read_build') as reader:
            with self.assertRaisesRegex(Refusal, 'configuration'): gate.roles('/cli', '/harness')
            reader.assert_not_called()

    def test_stronger_same_stage_drain_is_required_before_any_divergence_credit(self):
        import counterfactual_receiving as receiving
        import broad_stages
        import core_lifecycle
        config = dict(root='/evidence', live_corpus=dict(test_wasm_reach=dict(executed_divergence=
            dict(cli_build='/cli', harness_build='/harness', compiler_root='/compiler', module_cache='/modules'))))
        for variants in ([], ['trap'], ['trap', 'exit'], ['trap', 'exit', 'unknown']):
            facts = dict(readers=dict(lifecycle=dict(roots={core_lifecycle.DRAIN:
                dict(cases=[dict(variant=name) for name in variants], native_receive={'joined':True})})))
            with patch.object(broad_stages,'read',return_value=facts), patch.object(receiving,'observations') as original_reader:
                with self.subTest(variants=variants), self.assertRaises(Refusal):receiving.final_read(config)
                original_reader.assert_not_called()


if __name__ == '__main__':unittest.main()
