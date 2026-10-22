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

    def test_reference_wrappers_mutate_only_authenticated_derived_copies(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            trees = {}
            bindings = {}
            for name in ('source', 'cluster-source'):
                root = base / name
                (root / 'tools/gomad/cmd').mkdir(parents=True)
                (root / 'tools/gomad/go.mod').write_bytes(b'module reference\n')
                manifest = base / (name + '.sha256')
                manifest.write_text(self.gate.digest(b'module reference\n') + '  tools/gomad/go.mod\n')
                bindings[name] = dict(path=str(root), inventory=str(manifest),
                                      sha256=self.gate.digest(manifest.read_bytes()))
                trees[name], _ = self.gate.authenticate_tree(bindings[name], base)
            original = {name: self.gate.source_snapshot(root) for name, root in trees.items()}
            work = base / 'work'
            work.mkdir()
            derived = self.gate.derive_sources(trees, work)
            for name in trees:
                wrapper = derived[name] / 'tools/gomad/cmd/reference-wrapper'
                wrapper.mkdir()
                (wrapper / 'main.go').write_bytes(b'package main\n')
                self.gate.authenticate_tree(bindings[name], base)
                self.assertEqual(self.gate.source_snapshot(trees[name]), original[name])
                self.assertNotEqual(self.gate.source_snapshot(derived[name]), original[name])
            origins = self.gate.decode((work / 'derived-source-inputs.json').read_bytes())
            self.assertEqual(origins['schema'], 'gomad.parity-derived-source-inputs/v1')
            self.assertEqual(origins['origins']['source']['files'], original['source'])

    def test_derived_copy_changed_bytes_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            source, work = base / 'source', base / 'work'
            source.mkdir()
            work.mkdir()
            (source / 'input').write_bytes(b'original')
            copy = self.gate.shutil.copytree
            def corrupt(origin, destination, **kwargs):
                result = copy(origin, destination, **kwargs)
                (destination / 'input').write_bytes(b'corrupted')
                return result
            with patch.object(self.gate.shutil, 'copytree', side_effect=corrupt):
                with self.assertRaisesRegex(ValueError, 'copy changed bytes'):
                    self.gate.derive_sources({'source': source, 'cluster-source': source}, work)

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
