#!/usr/bin/env python3
"""Source-only negative tests for the all-public-verb gate; no guest execution."""
import ast
import importlib.machinery
import importlib.util
from pathlib import Path
import tempfile
import sys
import textwrap
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True

SCRIPT = Path(__file__).resolve().parents[2] / 'scripts/all-verb-parity'


class GateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        loader = importlib.machinery.SourceFileLoader('all_verb_parity', str(SCRIPT))
        cls.gate = importlib.util.module_from_spec(importlib.util.spec_from_loader(loader.name, loader))
        loader.exec_module(cls.gate)

    def test_duplicate_json_member_refused(self):
        with self.assertRaises(ValueError):
            self.gate.decode(b'{"schema":1,"schema":1}')

    def test_inventory_escape_and_stale_bytes_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'input').write_bytes(b'original')
            manifest = root / 'inputs.sha256'
            for raw in (self.gate.digest(b'original') + '  ../input\n', '0' * 64 + '  input\n'):
                manifest.write_text(raw)
                binding = dict(path=str(root), inventory=str(manifest), sha256=self.gate.digest(manifest.read_bytes()))
                with self.assertRaises(ValueError):
                    self.gate.authenticate_tree(binding, root)

    def test_empty_inventory_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = root / 'inputs.sha256'
            manifest.write_bytes(b'')
            with self.assertRaises(ValueError):
                self.gate.authenticate_tree(dict(path=str(root), inventory=str(manifest),
                    sha256=self.gate.digest(b'')), root)

    def test_inventory_symlink_escape_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            root = base / 'root'
            root.mkdir()
            (base / 'outside').write_bytes(b'outside')
            (root / 'escape').symlink_to(base / 'outside')
            manifest = base / 'inventory'
            manifest.write_text(self.gate.digest(b'outside') + '  escape\n')
            with self.assertRaises(ValueError):
                self.gate.authenticate_tree(dict(path=str(root), inventory=str(manifest),
                    sha256=self.gate.digest(manifest.read_bytes())), base)

    def test_golden_regeneration_refused_before_inputs_or_execution(self):
        with patch.dict(self.gate.os.environ, {'GOMAD_GOLDEN_REGENERATE': '1'}):
            with self.assertRaisesRegex(ValueError, 'regeneration forbidden'):
                self.gate.run_all(Path('/missing'), '0' * 64, Path('/missing-output'))

    def test_workflow_provisioning_python_compiles(self):
        workflow = SCRIPT.parents[3] / '.github/workflows/gomad-behavioral-parity.yml'
        text = workflow.read_text()
        block = text.split("python3 - <<'PY'\n", 1)[1].split('          PY\n', 1)[0]
        ast.parse(textwrap.dedent(block))
        self.assertIn('  full-parity:\n', text)
        self.assertNotIn('continue-on-error:', text)

    def test_fixed_coverage_includes_new_and_refused_verbs(self):
        required = {'inspect', 'replay', 'explore', 'resume', 'execute-shard', 'merge', 'minimize',
                    'qualify', 'report', 'qualify-set', 'merge-set', 'compare-support', 'install',
                    'doctor', 'discover', 'plan', 'recover', 'analyze', 'stress'}
        self.assertEqual(set(self.gate.PUBLIC_VERBS), required)
        with self.assertRaises(ValueError):
            self.gate.require_fields(dict(commands=[]), {'schema', 'files', 'trees'}, 'bindings')

    def test_zero_exit_without_mandatory_receipt_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(ValueError):
                self.gate.require_receipt(Path(directory), 'missing.json', 'expected/v1')

    def test_skipped_receipt_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'result.json').write_text('{"schema":"expected/v1","status":"skipped"}')
            with self.assertRaises(ValueError):
                self.gate.require_receipt(root, 'result.json', 'expected/v1')

    def test_receipt_stream_hash_substitution_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'run.stdout').write_bytes(b'changed')
            (root / 'run.stderr').write_bytes(b'')
            (root / 'commands.json').write_text(self.gate.encode([dict(name='run', status=0,
                expected=0, stdout_sha256='0' * 64, stderr_sha256=self.gate.digest(b''))]).decode())
            with self.assertRaises(ValueError):
                self.gate.authenticate_commands(root)

    def test_missing_expected_status_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'commands.json').write_text('[{"name":"run","status":0}]')
            with self.assertRaises(ValueError):
                self.gate.authenticate_commands(root)


if __name__ == '__main__':
    unittest.main()
