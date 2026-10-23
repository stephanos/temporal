#!/usr/bin/env python3
"""Offline Make policy tests; fake Go output is never runtime/golden evidence."""
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
loader = SourceFileLoader('interception_make', str(HERE / 'interception-make-check.py'))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)
MAKEFILE = HERE.parents[3] / 'Makefile'


class MakePolicy(unittest.TestCase):
    def execute(self, race, stale=False):
        with tempfile.TemporaryDirectory(prefix='interception-make-offline-') as directory:
            root = Path(directory)
            bindir = root / 'bin'
            bindir.mkdir()
            compiler = bindir / 'go'
            shutil.copyfile(HERE / 'interception-make-go.py', compiler)
            compiler.chmod(0o700)
            shutil.copyfile(MAKEFILE, root / 'Makefile')
            (root / 'go.mod').write_text('module offline.fixture\n')
            if stale:
                (root / 'test.log').write_text('ok\tstale-package\t(cached)\n')
            capture = root / 'capture.jsonl'
            env = dict(os.environ, PATH=str(bindir) + os.pathsep + os.environ['PATH'],
                       INTERCEPTION_MAKE_CAPTURE=str(capture))
            before = hashlib.sha256(MAKEFILE.read_bytes()).hexdigest()
            result = subprocess.run(check.make_args(race) + ['ROOT=' + str(root)],
                                    cwd=root, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            rows = [json.loads(line) for line in capture.read_text().splitlines()]
            self.assertEqual(hashlib.sha256(MAKEFILE.read_bytes()).hexdigest(), before)
            return result, [row for row in rows if row['argv'][0] == 'test']

    def test_default_and_explicit_race_refuse_three_commands_and_make_fails_with_fresh_log(self):
        for race in (None, 'on'):
            with self.subTest(race=race):
                result, rows = self.execute(race)
                self.assertEqual(result.returncode, 2)
                self.assertEqual(result.stdout.count(b'unsupported guest option: -race'), 3)
                self.assertEqual(len(rows), 3)
                self.assertTrue(all('-race' in row['argv'] for row in rows))

    def test_stale_success_log_can_mask_all_three_go_refusals(self):
        result, rows = self.execute(None, stale=True)
        self.assertEqual(result.returncode, 0)
        self.assertEqual(result.stdout.count(b'unsupported guest option: -race'), 3)
        self.assertEqual(len(rows), 3)

    def test_positive_recipe_expansion_preserves_selector_flags_and_three_same_package_calls(self):
        result, rows = self.execute('off')
        self.assertEqual(result.returncode, 0)
        self.assertEqual(len(rows), 3)
        self.assertEqual(len({tuple(row['argv']) for row in rows}), 1)
        for row in rows:
            self.assertEqual(row['cgo'], '0')
            self.assertEqual(row['argv'][1], './tests/gomadfunctional')
            for argument in ('-run=^TestFrontendSystemInfo$', '-v', '-count=1',
                             '-persistenceType=sql', '-persistenceDriver=sqlite', '-timeout=5m'):
                self.assertIn(argument, row['argv'])
            self.assertNotIn('-race', row['argv'])
            self.assertNotIn('-shuffle', row['argv'])
            self.assertIn('test_dep', row['argv'][row['argv'].index('-tags') + 1])

    def test_usage_normalization_changes_only_usage_executable(self):
        self.assertEqual(check.usage_bytes(b'Usage of /tmp/go-build123/exe/lintcode:\n  -base string\n'),
                         b'Usage of $HOST_TOOL:\n  -base string\n')
        self.assertNotEqual(check.usage_bytes(b'Usage of /x:\nchanged option\n'),
                            check.usage_bytes(b'Usage of /y:\noriginal option\n'))


if __name__ == '__main__':
    unittest.main()
