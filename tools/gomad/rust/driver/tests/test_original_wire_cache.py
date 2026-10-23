"""Synthetic cache/source mutation controls, never original Go execution evidence."""
import base64
import hashlib
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import installed_cluster_cache as cache
import original_wire_cache as original
from driver import Refusal, sha
from tools.gomad.rust.driver.tests import test_installed_cluster_cache as fixtures


def fixture(root):
    module_cache = root / 'module-cache'
    lines = []
    for module, version in original.MODULES:
        _, _, sums = fixtures.CacheInputs().fixture(module_cache, module, version)
        lines.extend(sums.read_text().splitlines())
    sums.unlink()
    sums = root / 'source/tools/gomad/go.sum'
    sums.parent.mkdir(parents=True)
    sums.write_text('\n'.join(lines) + '\n')
    return module_cache, sums


class OriginalWireCache(unittest.TestCase):
    def test_exact_six_cache_modules_and_static_inputs_bind_all_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            module_cache, sums = fixture(root)
            files = original.verify(module_cache, sums)
            entry = dict(module_cache=str(module_cache), derived=str(root))
            self.assertEqual(original.input_paths(entry), {str(module_cache / name) for name in files})
            config = dict(files={str(module_cache / name): value['sha256'] for name, value in files.items()})
            binding = original.binding(config, entry)
            self.assertEqual(binding['root'], str(module_cache))
            self.assertEqual(binding['go_sum_sha256'], sha(sums))
            self.assertEqual(binding['file_count'], len(files))
            config['files'].pop(next(iter(config['files'])))
            with self.assertRaises(Refusal):
                original.binding(config, entry)

    def test_missing_extra_alias_and_wrong_original_sum_refuse(self):
        for mutation in ('missing', 'extra', 'alias', 'sum'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                module_cache, sums = fixture(root)
                module, version = original.MODULES[0]
                source, metadata = cache.paths(module_cache, module, version)
                if mutation == 'missing': (metadata / (version + '.zip')).unlink()
                if mutation == 'extra': (module_cache / 'ambient-input').write_bytes(b'extra')
                if mutation == 'alias': (source / 'alias').symlink_to(source / 'go.mod')
                if mutation == 'sum': sums.write_text('')
                with self.assertRaises((Refusal, OSError)):
                    original.verify(module_cache, sums)

    def test_offline_readonly_environment_overrides_ambient_module_and_object_caches(self):
        with patch.dict(os.environ, GOMODCACHE='/ambient/modules', GOCACHE='/ambient/objects',
                        GOPROXY='https://invalid.example', GOPRIVATE='*', GONOPROXY='*', GOFLAGS='-mod=mod'):
            environment = original.environment(Path('/pinned/go'), Path('/private/scratch'), '/pinned/modules')
            self.assertEqual(environment['GOMODCACHE'], '/pinned/modules')
            self.assertEqual(environment['GOCACHE'], '/private/scratch/go-cache')
            self.assertEqual(environment['GOFLAGS'], '-p=1 -mod=readonly')
            self.assertEqual(environment['GOPROXY'], 'off')
            self.assertEqual(environment['GOVCS'], '*:off')
            self.assertEqual(environment['GOPRIVATE'], '')
            self.assertEqual(environment['GONOPROXY'], '')
            self.assertEqual(environment['GO111MODULE'], 'on')

    def test_generated_legacy_mod_metadata_is_checked_by_its_original_go_mod_sum(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            module, version = original.MODULES[0]
            source, metadata, sums = fixtures.CacheInputs().fixture(root, module, version)
            (source / 'go.mod').unlink()
            name = module + '@' + version + '/unix/file.go'
            raw = (source / 'unix/file.go').read_bytes()
            with zipfile.ZipFile(metadata / (version + '.zip'), 'w') as archive:
                archive.writestr(name, raw)
            digest = 'h1:' + base64.b64encode(hashlib.sha256(
                (hashlib.sha256(raw).hexdigest() + '  ' + name + '\n').encode()).digest()).decode()
            (metadata / (version + '.ziphash')).write_text(digest)
            sums.write_text(module + ' ' + version + ' ' + digest + '\n' + sums.read_text().splitlines()[1] + '\n')
            cache.verify(root, sums, module, version)
            (metadata / (version + '.mod')).write_bytes(b'module attacker.invalid\n')
            with self.assertRaises(Refusal):
                cache.verify(root, sums, module, version)

    def test_multi_module_snapshot_preserves_source_and_rejects_overwrite(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            module_cache, sums = fixture(root)
            expected = original.verify(module_cache, sums)
            destination = root / 'private'
            cache.snapshot_modules(module_cache, destination, sums, original.MODULES)
            self.assertEqual(original.verify(destination, sums), expected)
            self.assertEqual(original.verify(module_cache, sums), expected)
            with self.assertRaises(Refusal):
                cache.snapshot_modules(module_cache, destination, sums, original.MODULES)


if __name__ == '__main__':
    unittest.main()
