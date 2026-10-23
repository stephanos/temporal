"""Single-route adapters preserve frozen inputs; fixtures earn no qualification."""
import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))


def module(name, filename):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(filename))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


support = module('temporal_support', 'temporal-support.py')
inspection = module('temporal_inspection', 'temporal-inspection.py')


class Delegation(unittest.TestCase):
    def test_support_forwards_all_private_controls_without_starting_second_route(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            argv = ['temporal-support.py']
            names = ['binary', 'repo', 'root', 'compiler', 'runtime-root', 'cache',
                     'captured-config', 'source-manifest', 'private-observation-config', 'support']
            for name in names:
                argv += ['--' + name, str(root / name)]
            argv.append('--prepare-only')
            observed = []
            class Runner:
                def main(self, arguments=None):
                    observed.extend(["onebox-gates.py", *arguments] if arguments is not None else sys.argv)
            with patch.object(sys, 'argv', argv), patch.object(support, 'bounded_runner', return_value=Runner()), \
                 patch.object(support.producer, 'run', side_effect=AssertionError('duplicate old route')):
                support.main()
                self.assertIs(sys.argv, argv)
            for name in names:
                self.assertEqual(observed[observed.index('--' + name) + 1], str(root / name))
            self.assertIn('--prepare-only', observed)
            self.assertEqual(Path(observed[observed.index('--inspection') + 1]).name, 'temporal-inspection.py')
            self.assertFalse((root / 'root').exists())

    def test_onebox_inspection_adapter_forwards_private_context_to_authenticated_gate(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            argv = ['temporal-inspection.py', '--owner', 'onebox']
            names = ['binary', 'repo', 'root', 'compiler', 'runtime-root', 'cache',
                     'captured-config', 'private-observation-config']
            for name in names:
                argv += ['--' + name, str(root / name)]
            observed = []
            class Runner:
                def main(self, arguments=None):
                    observed.extend(["onebox-inspection-check.py", *arguments] if arguments is not None else sys.argv)
            with patch.object(sys, 'argv', argv), patch.object(inspection, 'onebox_runner', return_value=Runner()), \
                 patch.object(inspection.producer, 'run', side_effect=AssertionError('duplicate old inspection')):
                inspection.main()
                self.assertIs(sys.argv, argv)
            for name in names:
                self.assertEqual(observed[observed.index('--' + name) + 1], str(root / name))
            self.assertFalse((root / 'root').exists())

    def test_original_helpers_preserve_exact_named_pass_and_intent(self):
        support.selected_pass(b'--- PASS: TestFrontendSystemInfo (0.01s)\n', 'frontend')
        with self.assertRaises(ValueError):
            support.selected_pass(b'PASS\n', 'frontend')
        self.assertEqual(support.gate_environment('smoke', True)['GOMAD_WASM_QUALIFY_WORKLOADS'], '1')
        self.assertNotIn('GOMAD_WASM_QUALIFY_WORKLOADS', support.gate_environment('frontend', True))


if __name__ == '__main__':
    unittest.main()
