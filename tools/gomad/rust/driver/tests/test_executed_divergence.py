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


if __name__ == '__main__':unittest.main()
