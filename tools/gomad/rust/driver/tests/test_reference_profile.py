"""Invocation/custody controls; synthetic inputs are not runtime qualification."""
import base64
import copy
import hashlib
import io
import os
import tarfile
from pathlib import Path
import sys
import subprocess
import tempfile
import unittest
from unittest import mock

HERE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(HERE))
import reference_profile as reference


def native_input():
    paths = ['/tmp', '/workspace']
    limits = dict(files=1000, descriptors=64, filesystem_bytes=8 << 20, output_bytes=1 << 20)
    return dict(kind='standalone', network=None, guest=dict(module='/module.wasm', module_sha256='a' * 64,
        limits=dict(fuel=1 << 40, memory_bytes=256 << 20, output_bytes=1 << 20, evidence_bytes=256 << 20, wall_millis=180000),
        storage=None, namespace=dict(entries=[dict(path=list(name.encode()), directory=True, readonly=False, data=[]) for name in paths],
            writable_directories=[list(name.encode()) for name in ['/workspace', '/tmp']], preopens=[list(name.encode()) for name in paths],
            working_directory=list(b'/workspace'), stdin=[], limits=limits, epoch_nanos=946684800000000000),
        model=dict(arguments=['guest', 'literal $arg'], environment=['ONLY=captured', 'PWD=/workspace', 'TZ=UTC'], replay_trace=None,
            configuration=dict(entropy_key=[1, 2, 3] + [0] * 29, working_directory_sha256=hashlib.sha256(b'/workspace').hexdigest(),
                arguments=dict(bytes=0, count=0, chunks=0), environment=dict(bytes=0, count=0, chunks=0), seed=7, mode=1,
                limit=64 << 20, diagnostics=False, epoch_nanos=946684800000000000, read_step_nanos=0, now=1,
                limits=dict(calls=1 << 20, transcript_bytes=64 << 20, pending_events=4096)))))


def mapped(native):
    return reference.original_request(native, '/pinned/module.wasm', '/helper', 'b' * 64, 'c' * 64, dict(TargetSHA256=[170] * 32, ToolchainBuildKey='c' * 64, GOOS='wasip1', GOARCH='wasm', ImplementationSHA256=[19] * 32), ('linux', 'aarch64'))


class RequestMapping(unittest.TestCase):
    def test_original_archive_keeps_exact_relative_symlink_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / 'original.tar'
            with tarfile.open(archive, 'w') as bundle:
                data = tarfile.TarInfo('versioned/value')
                data.size = 1
                bundle.addfile(data, io.BytesIO(b'x'))
                link = tarfile.TarInfo('value')
                link.type = tarfile.SYMTYPE
                link.linkname = './versioned/value'
                bundle.addfile(link)
            source = root / 'source'
            source.mkdir()
            with tarfile.open(archive) as bundle:
                reference.extract_original(bundle, source)
            self.assertEqual(os.readlink(source / 'value'), './versioned/value')
            self.assertEqual((source / 'value').read_bytes(), b'x')

    def test_build_cli_dispatches_to_builder_without_starting_go(self):
        with tempfile.TemporaryDirectory(dir='/home/agent/wt', prefix='fn232-reference-cli-') as temporary:
            files = Path(temporary) / 'files.json'
            files.write_bytes(b'{}')
            args = ['reference_profile.py', 'build', '--derived', '/derived', '--stock-go', '/stock-go', '--output', '/output', '--files', str(files)]
            with mock.patch.object(sys, 'argv', args), mock.patch.object(reference, 'build') as builder:
                reference.main()
                builder.assert_called_once_with('/derived', '/stock-go', '/output', {})

    def test_original_runtime_recipe_preserves_each_guest_control(self):
        native = native_input()
        value = mapped(native)
        self.assertEqual(value['Fuel'], 1 << 40)
        self.assertEqual(value['MemoryBytes'], 256 << 20)
        self.assertEqual(value['Timeout'], 180000000000)
        self.assertEqual(value['Config']['Args'], native['guest']['model']['arguments'])
        self.assertEqual(value['Config']['Environment'], native['guest']['model']['environment'])
        self.assertEqual(value['Config']['EntropyKey'], [1, 2, 3] + [0] * 29)
        self.assertEqual(value['Config']['Limits'], dict(OutputBytes=1 << 20, TranscriptBytes=64 << 20, Calls=1 << 20,
            PendingEvents=4096, Descriptors=64, Files=1000, FilesystemBytes=8 << 20))
        self.assertEqual(value['Config']['Clock'], dict(EpochNanos=946684800000000000, ReadStepNanos=0))
        self.assertEqual(value['Runtime']['Seed'], 7)
        self.assertFalse(value['Runtime']['Diagnostics'])
        self.assertEqual(value['Runtime']['Choice']['Limit'], 64 << 20)
        self.assertEqual(value['Runtime']['Choice']['ExecutionIdentity']['ImplementationSHA256'], [19] * 32)
        self.assertEqual(value['ModulePath'], '/pinned/module.wasm')
        self.assertEqual(value['Engine']['configuration'], reference.ENGINE)

    def test_guest_input_is_never_silently_ignored(self):
        for mutation in range(11):
            native = native_input()
            guest = native['guest']
            if mutation == 0:
                native['network'] = {}
            elif mutation == 1:
                guest['storage'] = {}
            elif mutation == 2:
                guest['namespace']['entries'][0]['data'] = [1]
            elif mutation == 3:
                guest['model']['replay_trace'] = [1]
            elif mutation == 4:
                guest['model']['configuration']['split'] = True
            elif mutation == 5:
                guest['namespace']['limits']['output_bytes'] += 1
            elif mutation == 6:
                guest['model']['configuration']['working_directory_sha256'] = 'b' * 64
            elif mutation == 7:
                guest['model']['configuration']['inspection'] = {}
            elif mutation == 8:
                guest['limits']['unmapped_bound'] = 1
            elif mutation == 9:
                guest['model']['configuration']['read_step_nanos'] = 1000
            else:
                guest['limits']['evidence_bytes'] = 1
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                mapped(native)

    def test_original_runtime_request_has_a_distinct_closed_preparation_projection(self):
        native = native_input()
        native['guest']['model']['environment'] = ['PWD=/workspace', 'TZ=UTC', 'ONLY=captured']
        native['guest']['model']['configuration']['digest_only'] = False
        model = copy.deepcopy(native['guest']['model']['configuration'])
        del model['seed']
        del model['digest_only']
        repo = HERE.parents[3]
        directory = repo / 'tools/gomad'
        source = directory / 'internal/model/wasi/testdata/runtime_cost/main.go'
        proof = dict(configuration=dict(model=model, arguments=['gomad-target'],
            environment=dict(ONLY='captured', PWD='/workspace', TZ='UTC',
                GOMAD_IO_PROFILE='wasm', GOMAD_CHOICE_PROFILE='gomad-choice-trace/v4'),
            derive_seed_environment=True, limits=native['guest']['limits'],
            namespace=native['guest']['namespace'], storage=None, network=None),
            inputs=dict(kind='go-run', source='./internal/model/wasi/testdata/runtime_cost',
                directory=str(directory), arguments=[], tags=[],
                sources={str(source): hashlib.sha256(source.read_bytes()).hexdigest()}))
        reference.bind_preparation(native, proof, original_runtime=True)
        for mutation in range(11):
            changed, provenance = copy.deepcopy(native), copy.deepcopy(proof)
            if mutation == 0:
                changed['guest']['model']['environment'].append('GOMADSEED=7')
            elif mutation == 1:
                changed['guest']['model']['environment'].reverse()
            elif mutation == 2:
                changed['guest']['model']['arguments'][0] = 'gomad-target'
            elif mutation == 3:
                changed['guest']['model']['configuration']['seed'] = 8
            elif mutation == 4:
                provenance['inputs']['arguments'] = ['literal $arg']
            elif mutation == 5:
                provenance['inputs']['tags'] = ['gomad_wasm_slim']
            elif mutation == 6:
                provenance['inputs']['sources'][str(source)] = '0' * 64
            elif mutation == 7:
                provenance['inputs']['source'] = './another'
            elif mutation == 8:
                provenance['configuration']['arguments'].append('literal $arg')
            elif mutation == 9:
                provenance['configuration']['model']['entropy_key'] = [0] * 32
            else:
                changed['guest']['model']['configuration']['limit'] += 1
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                reference.bind_preparation(changed, provenance, original_runtime=True)

    def test_actual_request_must_match_prepared_configuration_except_derived_seed(self):
        native = native_input()
        guest = native['guest']
        model = copy.deepcopy(guest['model']['configuration'])
        del model['seed']
        configuration = dict(model=model, arguments=guest['model']['arguments'],
            environment=dict(ONLY='captured', PWD='/workspace', TZ='UTC'), limits=guest['limits'],
            namespace=guest['namespace'], storage=None, network=None)
        proof = dict(configuration=configuration)
        reference.bind_preparation(native, proof)
        for field in ('entropy_key', 'limit', 'epoch_nanos'):
            changed = copy.deepcopy(native)
            changed['guest']['model']['configuration'][field] = [9] * 32 if field == 'entropy_key' else 9
            with self.subTest(field=field), self.assertRaisesRegex(ValueError, 'prepared configuration'):
                reference.bind_preparation(changed, proof)

    def test_original_same_evidence_checks_digest_even_when_raw_wasi_is_empty(self):
        before = dict(streams={name: dict(sha256='a' * 64) for name in ('stdout', 'stderr', 'choice', 'network', 'terminal')},
                      transcript_sha256='b' * 64, termination='exit', exit_code=0)
        reference.same_original_evidence(before, copy.deepcopy(before))
        for field in (*before['streams'], 'transcript_sha256', 'termination', 'exit_code'):
            after = copy.deepcopy(before)
            if field in after['streams']:
                after['streams'][field]['sha256'] = 'c' * 64
            else:
                after[field] = 1 if field == 'exit_code' else 'changed'
            with self.subTest(field=field), self.assertRaises(ValueError):
                reference.same_original_evidence(before, after)

    def test_raw_result_projection_retains_unmodified_original_bytes_and_failure(self):
        raw = dict(result=dict(Stdout=base64.b64encode(b'raw\x00stdout').decode(), Stderr=None, Transcript=None,
            NetworkTranscript=None, ChoiceTrace=dict(Bytes=base64.b64encode(b'choice').decode(), Summary=dict(SelectPoll=100000)),
            Termination='infrastructure', ExitCode=None, Reaped=True, Cancelled=False, WatchdogTimeout=True,
            TranscriptSHA256=[3] * 32, Calls=42, TimeAdvances=2, EnvironmentIdentity='original', FuelRemaining=99,
            AuthorizationToTerminalNanos=5), terminal=base64.b64encode(bytes(16)).decode(), error='actual failure')
        with tempfile.TemporaryDirectory(dir='/home/agent/wt', prefix='fn232-reference-fixture-') as temporary:
            directory = Path(temporary)
            encoded = reference.custody.canonical(raw)
            run = reference.raw_run(encoded, directory, 'go')
            self.assertEqual((directory / 'stdout').read_bytes(), b'raw\x00stdout')
            self.assertEqual((directory / 'wasi').read_bytes(), b'')
            self.assertEqual(run['transcript_sha256'], '03' * 32)
            self.assertEqual(run['raw_result']['sha256'], hashlib.sha256(encoded).hexdigest())
            self.assertTrue(run['watchdog_timeout'])
            self.assertEqual(run['error'], 'actual failure')


class OriginalSourceCustody(unittest.TestCase):
    def test_archive_and_single_addition_custody_refuse_source_and_glue_mutations(self):
        with tempfile.TemporaryDirectory(dir='/home/agent/wt', prefix='fn232-reference-custody-') as temporary:
            directory = Path(temporary) / 'derived'
            manifest = reference.derive(HERE.parents[3], directory)
            self.assertEqual(manifest['original_commit'], reference.ORIGINAL)
            source = directory / 'source'
            original = source / 'tools/gomad/internal/backends/wasm/wasi/process.go'
            raw = original.read_bytes()
            original.write_bytes(raw + b'\n')
            with self.assertRaisesRegex(ValueError, 'outside declared'):
                reference.verify(directory)
            original.write_bytes(raw)
            glue = source / reference.GLUE
            raw = glue.read_bytes()
            glue.write_bytes(raw + b'\n')
            with self.assertRaisesRegex(ValueError, 'outside declared'):
                reference.verify(directory)
            glue.write_bytes(raw)
            extra = source / 'undeclared.go'
            extra.write_text('package unrelated\n')
            with self.assertRaisesRegex(ValueError, 'outside declared'):
                reference.verify(directory)
            extra.unlink()
            reference.verify(directory)
            manifest_path = directory / 'source.json'
            manifest_raw = manifest_path.read_bytes()
            literal = source / 'schema/elasticsearch/visibility/index_template_v7.json'
            original_target = os.readlink(literal)
            self.assertTrue(original_target.startswith('./'))
            literal.unlink()
            literal.symlink_to(original_target[2:])
            modified = copy.deepcopy(manifest)
            modified['original_files'][str(literal.relative_to(source))]['target'] = original_target[2:]
            manifest_path.write_bytes(reference.custody.canonical(modified))
            with self.assertRaisesRegex(ValueError, 'archive'):
                reference.verify(directory)
            literal.unlink()
            literal.symlink_to(original_target)
            manifest_path.write_bytes(manifest_raw)
            original_raw = original.read_bytes()
            original.write_bytes(original_raw + b'\n')
            modified = copy.deepcopy(manifest)
            modified['original_files'][str(original.relative_to(source))] = dict(kind='file',
                **reference.custody.snapshot(original, 1 << 20))
            manifest_path.write_bytes(reference.custody.canonical(modified))
            with self.assertRaisesRegex(ValueError, 'archive'):
                reference.verify(directory)
            original.write_bytes(original_raw)
            manifest_path.write_bytes(manifest_raw)
            mode = original.stat().st_mode & 0o777
            original.chmod(mode ^ 0o100)
            with self.assertRaisesRegex(ValueError, 'mode'):
                reference.verify(directory)
            original.chmod(mode)
            extra_directory = source / 'undeclared-empty-directory'
            extra_directory.mkdir()
            with self.assertRaisesRegex(ValueError, 'archive'):
                reference.verify(directory)
            extra_directory.rmdir()
            reference.verify(directory)

class HelperCopyCustody(unittest.TestCase):
    def test_helper_source_inventory_requires_missing_and_extra_tracked_members(self):
        with tempfile.TemporaryDirectory(dir='/home/agent/wt', prefix='fn232-helper-git-fixture-') as temporary:
            root = Path(temporary)
            package = root / 'tools/gomad/rust/wasmhost'
            (package / 'src').mkdir(parents=True)
            for name in ('src/lib.rs', 'src/conformance.rs', 'build.rs', 'build_identity.rs', 'Cargo.toml', '../Cargo.toml', '../Cargo.lock', '../rust-toolchain.toml'):
                (package / name).write_text('bounded source fixture\n')
            subprocess.run(['git', 'init', '-q', str(root)], check=True)
            subprocess.run(['git', '-C', str(root), 'add', '.'], check=True)
            subprocess.run(['git', '-C', str(root), '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture'], check=True)
            commit = subprocess.check_output(['git', '-C', str(root), 'rev-parse', 'HEAD']).decode().strip()
            measured = reference.helper_sources(root, commit)
            identity_module = package / 'build_identity.rs'
            self.assertIn(str(identity_module), measured)
            original = identity_module.read_bytes()
            identity_module.write_text('mutated implementation preimage algorithm\n')
            with self.assertRaisesRegex(ValueError, 'immutable Git input'):
                reference.helper_sources(root, commit)
            identity_module.write_bytes(original)
            identity_module.unlink()
            with self.assertRaisesRegex(ValueError, 'membership'):
                reference.helper_sources(root, commit)
            identity_module.write_bytes(original)
            extra_root = package / 'untracked_algorithm.rs'
            extra_root.write_text('undeclared implementation algorithm\n')
            with self.assertRaisesRegex(ValueError, 'membership'):
                reference.helper_sources(root, commit)
            extra_root.unlink()
            path = package / 'src/conformance.rs'
            raw = path.read_bytes()
            path.unlink()
            with self.assertRaisesRegex(ValueError, 'membership'):
                reference.helper_sources(root, commit)
            path.write_bytes(raw)
            (package / 'src/extra.rs').write_text('undeclared source\n')
            with self.assertRaisesRegex(ValueError, 'membership'):
                reference.helper_sources(root, commit)
            (package / 'src/extra.rs').unlink()
            subprocess.run(['git', '-C', str(root), 'rm', '-q', str(identity_module)], check=True)
            subprocess.run(['git', '-C', str(root), '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                            'commit', '-qm', 'historical source without identity module'], check=True)
            historical = subprocess.check_output(['git', '-C', str(root), 'rev-parse', 'HEAD']).decode().strip()
            self.assertNotIn(str(identity_module), reference.helper_sources(root, historical))

    def test_exact_copy_raw_artifact_compiler_and_process_are_required(self):
        with tempfile.TemporaryDirectory(dir='/home/agent/wt', prefix='fn232-helper-receipt-fixture-') as temporary:
            root = Path(temporary)
            def put(name, raw):
                path = root / name
                path.write_bytes(raw)
                return dict(path=str(path), **reference.custody.snapshot(path, 1 << 20))
            artifact = put('artifact', b'synthetic helper fixture')
            binary = put('copy', b'synthetic helper fixture')
            cargo, rustc = put('cargo', b'synthetic cargo'), put('rustc', b'synthetic rustc')
            source = put('source.rs', b'synthetic source')
            source_table = {source['path']: dict(relative='source.rs', git_blob='a' * 40, bytes=source['bytes'], sha256=source['sha256'])}
            argv = [cargo['path'], 'build', '--offline', '--locked', '--manifest-path', str(root / 'tools/gomad/rust/Cargo.toml'), '-p', 'gomad-wasmhost', '--features', 'reference-helper', '--bin', 'gomad-wasmhost', '--message-format=json-render-diagnostics']
            command = dict(argv=argv, environment=dict(RUSTC=rustc['path']), cwd=str(root), exit=0, observation='completed', pid=1234, start_ns=1, end_ns=3)
            process = dict(argv=argv, active=False, exit=0, observation='completed', pid=1234, start_ns=1, end_ns=2)
            cargo_artifact = dict(reason='compiler-artifact', target=dict(name='gomad-wasmhost', kind=['bin']), features=['reference-helper'], executable=artifact['path'])
            stdout = put('cargo.stdout', reference.custody.canonical(cargo_artifact) + b'\n')
            command_file = put('command.json', reference.custody.canonical(command))
            process_file = put('process.json', reference.custody.canonical(process))
            receipt = dict(schema='gomad.reference-helper-build/v1', binary=binary, artifact_executable=artifact,
                copy=dict(method='copyfile', source=artifact, destination=binary), cargo_artifact=cargo_artifact,
                cargo_stdout=stdout, command_file=command_file, process_file=process_file, command=command,
                source=dict(source_root=str(root), source_commit='b' * 40, files=source_table,
                    source_table_sha256=reference.custody.digest(reference.custody.canonical(source_table))),
                toolchain={row['path']: {key: value for key, value in row.items() if key != 'path'} for row in (cargo, rustc)})
            files = {str(path): reference.custody.digest(path.read_bytes()) for path in root.iterdir()}
            with mock.patch.object(reference, 'helper_sources', return_value=source_table):
                reference.admit_helper(receipt, Path(binary['path']), files)
                for mutation in range(5):
                    changed = copy.deepcopy(receipt)
                    if mutation == 0:
                        changed['artifact_executable']['sha256'] = 'f' * 64
                    elif mutation == 1:
                        changed['cargo_artifact']['features'] = ['reference-helper', 'stress-jitter']
                    elif mutation == 2:
                        changed['command']['argv'][0] = '/unrelated/cargo'
                    elif mutation == 3:
                        changed['copy']['method'] = 'unobserved'
                    else:
                        changed['source']['source_table_sha256'] = 'f' * 64
                    with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                        reference.admit_helper(changed, Path(binary['path']), files)


if __name__ == '__main__':
    unittest.main()
