"""Synthetic raw fixtures exercise custody refusals; never qualification evidence."""
import base64
import io
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import original_ringpop as original
import receiving


def encoded(value):
    return base64.b64encode(value).decode()


class OriginalRingpop(unittest.TestCase):
    def test_capture_patch_is_exact_original_source_after_successful_decoder(self):
        raw = subprocess.check_output(['git', 'show', original.REFERENCE + ':' + original.CAPTURE_SOURCE],
                                      cwd=driver.HERE.parents[3])
        modified = original.capture_patch(raw)
        self.assertEqual(modified.count(b'os.Getenv("GOMAD_ORIGINAL_RINGPOP_NETWORK")'), 1)
        self.assertLess(modified.index(b'if err != nil {\n\t\t\treturn err\n\t\t}', modified.index(b'DecodeClusterNetworkEvidence(session.networkWire')),
                        modified.index(b'os.Getenv("GOMAD_ORIGINAL_RINGPOP_NETWORK")'))
        self.assertLess(modified.index(b'os.Getenv("GOMAD_ORIGINAL_RINGPOP_NETWORK")'),
                        modified.index(b'session.result.Network = evidence.Record'))
        self.assertIn(b'os.O_WRONLY|os.O_CREATE|os.O_EXCL', modified)
        self.assertIn(b'errors.Join(writeErr, file.Sync(), file.Close())', modified)
        for value in (raw.replace(b'\t"io"\n', b''), modified):
            with self.assertRaises(driver.Refusal): original.capture_patch(value)

    def test_archive_inventory_rejects_escape_and_custodies_symlinks(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / 'source.tar'
            with tarfile.open(archive, 'w') as handle:
                info = tarfile.TarInfo('file')
                info.size = 4
                handle.addfile(info, io.BytesIO(b'data'))
                link = tarfile.TarInfo('link')
                link.type, link.linkname = tarfile.SYMTYPE, 'file'
                handle.addfile(link)
            source = root / 'source'
            source.mkdir()
            with tarfile.open(archive) as handle: handle.extractall(source, filter='data')
            self.assertEqual(original.archive_inventory(archive), original.inventory(source))
            (source / 'link').unlink()
            (source / 'link').symlink_to('/dev/null')
            with self.assertRaises(driver.Refusal): original.inventory(source)
            with tarfile.open(archive, 'w') as handle:
                handle.addfile(tarfile.TarInfo('../escape'))
            with self.assertRaises(driver.Refusal): original.archive_inventory(archive)

    def test_real_original_archive_preserves_every_file_and_literal_symlink(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive, source = root / 'source.tar', root / 'source'
            subprocess.run(['git', 'archive', '--format=tar', '--output=' + str(archive), original.REFERENCE],
                           cwd=driver.HERE.parents[3], check=True)
            source.mkdir()
            actual = original.extract_archive(archive, source)
            expected = original.archive_inventory(archive)
            self.assertEqual(actual, expected)
            self.assertEqual(len(actual['files']), 9603)
            self.assertEqual(os.readlink(source / 'schema/elasticsearch/visibility/index_template_v7.json'),
                             './versioned/v15/index_template_v7.json')
            self.assertEqual({name: os.readlink(source / name) for name in expected['symlinks']}, expected['symlinks'])

    def test_source_extraction_refuses_escaping_absolute_and_parent_link_targets(self):
        for target in ('../outside', '/outside', 'inside/../data', './inside/../../outside'):
            with self.subTest(target=target), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                archive, source = root / 'source.tar', root / 'source'
                source.mkdir()
                with tarfile.open(archive, 'w') as handle:
                    body = tarfile.TarInfo('data')
                    body.size = 4
                    handle.addfile(body, io.BytesIO(b'data'))
                    link = tarfile.TarInfo('link')
                    link.type, link.linkname = tarfile.SYMTYPE, target
                    handle.addfile(link)
                with self.assertRaises(driver.Refusal):
                    original.extract_archive(archive, source)
                self.assertFalse((root / 'outside').exists())

    def fixture(self, root, mutation=None):
        artifact = root / 'artifacts/success'
        (artifact / 'backend').mkdir(parents=True)
        module = b'fixture-module'
        module_hash = receiving.digest(module)
        helper_hash = 'a' * 64
        outputs = []
        owners = []
        for number, node in enumerate(('a', 'b', 'c'), 1):
            value = b''.join(marker + ('10.0.0.%d:7946\n' % number).encode()
                             for marker in (b'bootstrapped ', b'partitioned view ', b'converged '))
            outputs.append(dict(handle=dict(node=node, incarnation=1), stream='stdout', bytes=encoded(value),
                full_sha256='sha256:' + receiving.digest(value), total_bytes=len(value), retained_bytes=len(value),
                discarded_bytes=0, truncated=False))
            owners.append(dict(handle=dict(node=node, incarnation=1), interrupted=False, module_sha256=module_hash,
                               helper_sha256=helper_hash, choice_trace=encoded(bytes([number]) * 96)))
        record = dict(outcome='completed', seed=7, outputs=outputs,
                      wasm_execution=dict(virtual_elapsed_nanos=120000000001, incarnations=owners, world=encoded(b'world')))
        if mutation == 'two-node': record['wasm_execution']['incarnations'].pop()
        if mutation == 'helper': owners[-1]['helper_sha256'] = 'b' * 64
        if mutation == 'module': owners[-1]['module_sha256'] = 'b' * 64
        if mutation == 'choice-tape': owners[-1]['choice_trace'] = encoded(bytes(264))
        if mutation == 'phase': outputs.pop()
        if mutation == 'duration': record['wasm_execution']['virtual_elapsed_nanos'] -= 1
        if mutation == 'outcome': record['outcome'] = 'interrupted'
        driver.write(artifact / 'backend/evidence.bin', dict(schema='gomad.wasm-simulation-retained/v2',
            record=encoded(driver.json.dumps(record).encode())))
        (artifact / 'target').write_bytes(module)
        files = [dict(path=name, size=str((artifact / name).stat().st_size), sha256='sha256:' + driver.sha(artifact / name))
                 for name in ('target', 'backend/evidence.bin')]
        target = dict(file='target', sha256=files[0]['sha256'], backend=dict(evidence=dict(
            file='backend/evidence.bin', sha256=files[1]['sha256'])))
        driver.write(artifact / 'manifest.json', dict(artifact_kind='gomad.success/v1', seed='7', files=files, target=target))
        (root / 'network.bin').write_bytes(b'GMNETEV1' + bytes(32))
        if mutation == 'network': (root / 'network.bin').write_bytes(b'legacy network JSON is not raw helper wire')
        if mutation == 'raw-tamper': (artifact / 'backend/evidence.bin').write_bytes(b'changed')
        return helper_hash, module_hash

    def test_exact_three_owner_raw_reference_and_negative_causality(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            helper, module = self.fixture(root)
            artifact, actual, raw = original.extract(root, helper)
            self.assertEqual(actual, module)
            self.assertEqual(set(raw), {'world', 'network', 'choice/a', 'choice/b', 'choice/c'})
            self.assertEqual(raw['choice/c'], bytes([3]) * 96)
        for mutation in ('two-node', 'helper', 'module', 'choice-tape', 'phase', 'duration', 'outcome', 'network', 'raw-tamper'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                helper, _ = self.fixture(root, mutation)
                with self.assertRaises(driver.Refusal): original.extract(root, helper)

    def test_frozen_original_archive_patch_and_exact_build_receipt_custody(self):
        for mutation in (None, 'argv', 'environment', 'source', 'capture'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                source = root / 'source'
                capture = source / original.CAPTURE_SOURCE
                capture.parent.mkdir(parents=True)
                raw = b'package wasi\nimport (\n\t"io"\n)\n\t\tsession.result.Network = evidence.Record\n'
                capture.write_bytes(raw)
                archive = root / 'source.tar'
                with tarfile.open(archive, 'w') as handle: handle.add(capture, arcname=original.CAPTURE_SOURCE)
                before = original.inventory(source)
                capture.write_bytes(original.capture_patch(raw))
                after = original.inventory(source)
                tools = {name: str(root / name) for name in ('git', 'compiler', 'cargo')}
                for value in tools.values(): Path(value).write_bytes(b'pinned tool')
                plan = dict(schema='gomad.original-ringpop-build-plan/v1', reference_commit=original.REFERENCE,
                            repo='/original/repository', cargo_target='/exclusive/target', **tools,
                            files={value: driver.sha(value) for value in tools.values()})
                argv = original.build_argv(plan, root)
                seals = {}
                for number, (name, arguments) in enumerate(argv.items(), 1):
                    row = dict(argv=arguments, pid=number, exit=0, observation='completed', start_ns=10, end_ns=20,
                               cwd=plan['repo'] if name == 'archive' else str(source / 'tools/gomad'))
                    environment = {} if name == 'archive' else dict(GOCACHE=str(root / 'go-cache'),
                        CARGO_TARGET_DIR=plan['cargo_target'], CARGO_INCREMENTAL='0')
                    driver.write(root / (name + '.command.json'), dict(row, environment=environment, temporary_paths={}))
                    driver.write(root / (name + '.process.json'), dict(row, active=False))
                    for suffix in ('stdout', 'stderr'): (root / (name + '.' + suffix)).write_bytes(b'')
                    seals[name] = {suffix.split('.')[0] + '_sha256': driver.sha(root / (name + '.' + suffix))
                                   for suffix in ('command.json', 'process.json', 'stdout', 'stderr')}
                binary, helper = root / 'gomad-original', root / 'gomad-wasmhost-original'
                binary.write_bytes(b'original host')
                helper.write_bytes(b'original helper')
                inputs = dict(schema='gomad.original-ringpop-inputs/v1', reference_commit=original.REFERENCE,
                    build_plan=plan, source_root=str(source), source_archive=str(archive), source_archive_sha256=driver.sha(archive),
                    original_source_inventory=before, source_inventory=after, binary=str(binary), binary_sha256=driver.sha(binary),
                    helper=str(helper), helper_sha256=driver.sha(helper), compiler=tools['compiler'], compiler_sha256=driver.sha(tools['compiler']),
                    capture=dict(path=original.CAPTURE_SOURCE, original_sha256=receiving.digest(raw), sha256=driver.sha(capture),
                        environment_variable=original.CAPTURE_VARIABLE, phase='after original network decoder accepted complete helper bytes'),
                    build_commands=seals)
                if mutation in ('argv', 'environment'):
                    path = root / 'helper-build.command.json'
                    row = driver.read(path)
                    if mutation == 'argv': row['argv'] = row['argv'] + ['--features', 'unrelated']
                    else: row['environment']['RUSTFLAGS'] = '-Cdebug-assertions=yes'
                    path.unlink()
                    driver.write(path, row)
                    seals['helper-build']['command_sha256'] = driver.sha(path)
                if mutation == 'source': capture.write_bytes(capture.read_bytes() + b'changed')
                if mutation == 'capture': inputs['capture']['phase'] = 'before original decoder'
                driver.write(root / 'inputs.json', inputs)
                config = dict(final_retained=dict(ringpop_original=dict(inputs=str(root / 'inputs.json'))),
                              files={str(path): driver.sha(path) for path in root.rglob('*') if path.is_file()})
                if mutation is None: self.assertEqual(original.sealed(config), inputs)
                else:
                    with self.assertRaises(driver.Refusal): original.sealed(config)

    def test_read_only_missing_reference_never_launches_a_producer(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(original, 'sealed', return_value={}), \
                patch.object(original, 'command') as command:
            with self.assertRaisesRegex(driver.Refusal, 'UNEXECUTED'):
                original.collect({}, Path(directory), read_only=True)
            command.assert_not_called()

    def test_build_recipe_is_original_source_and_pinned_profile(self):
        plan = dict(git='/pinned/git', compiler='/pinned/go', cargo='/pinned/cargo', repo='/source/repo')
        argv = original.build_argv(plan, Path('/reference'))
        self.assertEqual(argv['archive'][-1], original.REFERENCE)
        self.assertEqual(argv['host-build'][-1], './cmd/gomad')
        self.assertIn('-trimpath', argv['host-build'])
        self.assertEqual(argv['helper-build'][1:7], ['+1.99.1', 'build', '--release', '--locked', '--offline', '-j1'])
        self.assertNotIn('--features', argv['helper-build'])


if __name__ == '__main__': unittest.main()
