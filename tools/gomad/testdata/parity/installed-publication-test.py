#!/usr/bin/env python3
"""Real native make publication, with the original controlled Cargo publisher; no compiler/guest execution."""
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
import tomllib
import unittest

MODULE = Path(__file__).resolve().parents[2]
WANT = b'#!/bin/sh\nexit 0\n'


class PublicationTests(unittest.TestCase):
    def test_public_build_recipes_publish_independent_executable_in_all_original_cases(self):
        for target, binary, package, features in (
                ('build', 'gomad', 'gomad', []),
                ('wasm-test-helper', 'gomad-wasmhost', 'gomad-wasmhost', ['reference-helper'])):
            for name in ('default-target', 'custom-target', 'custom-target-matching-output'):
                with self.subTest(target=target, case=name), tempfile.TemporaryDirectory(prefix='gomad publication ') as private:
                    root = Path(private)
                    (root / 'Makefile').write_bytes((MODULE / 'Makefile').read_bytes())
                    script = MODULE / 'rust/publish-executable.sh'
                    if script.exists():
                        (root / 'rust').mkdir()
                        shutil.copyfile(script, root / 'rust/publish-executable.sh')
                    fixture = root / 'fixture'
                    fixture.write_bytes(WANT)
                    fixture.chmod(0o700)
                    output = root / 'rust/target/release' / binary
                    if name == 'custom-target-matching-output':
                        output.parent.mkdir(parents=True)
                        os.link(fixture, output)
                    tools = root / 'bin'
                    tools.mkdir()
                    cargo = tools / 'cargo'
                    cargo.write_text('#!' + sys.executable + '\n' + '''import json,os,sys
from pathlib import Path
args=sys.argv[1:]
Path(os.environ['CARGO_ARGS']).write_text(json.dumps(args))
package=args[args.index('-p')+1]
output=Path(os.environ.get('CARGO_TARGET_DIR','rust/target'))/'release'/('gomad' if package=='gomad' else 'gomad-wasmhost')
output.parent.mkdir(parents=True,exist_ok=True)
if not output.exists(): os.link(os.environ['FIXTURE'],output)
''')
                    cargo.chmod(0o700)
                    env = {key: value for key, value in os.environ.items()
                           if not key.startswith(('CARGO_', 'GOMAD', 'QUALDRV', 'MAKE'))
                           and key not in ('PATH', 'FIXTURE')}
                    env.update(PATH=str(tools) + os.pathsep + os.environ['PATH'], FIXTURE=str(fixture),
                               CARGO_ARGS=str(root / 'cargo-args.json'))
                    if name != 'default-target':
                        env['CARGO_TARGET_DIR'] = str(root / 'custom target')
                    completed = subprocess.run([shutil.which('make'), target], cwd=root, env=env,
                                               capture_output=True, timeout=30, check=False)
                    self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)
                    self.assertEqual(output.read_bytes(), WANT)
                    self.assertFalse(output.is_symlink())
                    self.assertEqual(stat.S_IMODE(output.stat().st_mode), 0o755)
                    self.assertEqual(output.stat().st_nlink, 1)
                    self.assertNotEqual((fixture.stat().st_dev, fixture.stat().st_ino),
                                        (output.stat().st_dev, output.stat().st_ino))
                    self.assertEqual(stat.S_IMODE(fixture.stat().st_mode), 0o700)
                    args = json.loads((root / 'cargo-args.json').read_bytes())
                    expected = ['+1.99.1', 'build', '--manifest-path', 'rust/Cargo.toml', '--release', '--locked', '-p', package]
                    if features:
                        expected += ['--features', ','.join(features)]
                    self.assertEqual(args, expected)
                    if target == 'build':
                        self.assertFalse((output.parent / 'gomad-wasmhost').exists())
                    fixture.write_bytes(b'changed Cargo inode')
                    self.assertEqual(output.read_bytes(), WANT)

    def test_standalone_transport_remains_explicit_feature_only(self):
        manifest = tomllib.loads((MODULE / 'rust/wasmhost/Cargo.toml').read_text())
        transport = next(binary for binary in manifest['bin'] if binary['name'] == 'gomad-wasmhost')
        self.assertEqual(transport['required-features'], ['reference-helper'])
        self.assertFalse(manifest['features'].get('default', []))

    def test_invalid_publication_source_preserves_existing_output(self):
        with tempfile.TemporaryDirectory() as private:
            root = Path(private)
            fixture, output, link = root / 'fixture', root / 'output', root / 'link'
            fixture.write_bytes(WANT)
            output.write_bytes(b'previous independent publication')
            link.symlink_to(fixture)
            for source in (link, root / 'missing', root):
                result = subprocess.run(['sh', str(MODULE / 'rust/publish-executable.sh'), str(source), str(output)],
                                        capture_output=True, timeout=30, check=False)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(output.read_bytes(), b'previous independent publication')
                self.assertEqual(fixture.read_bytes(), WANT)


if __name__ == '__main__':
    unittest.main()
