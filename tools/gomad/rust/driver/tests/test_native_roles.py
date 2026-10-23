"""Synthetic native receipt mutations; no compiler or native process is executed."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import native_roles as native
import core_identity
from driver import Refusal, sha


class NativeRoles(unittest.TestCase):
    def fixture(self, root):
        repo = root / 'source'
        for kind, package in (('host', 'gomad'), ('helper', 'wasmhost')):
            base = repo / 'tools/gomad/rust' / package
            names = set(native.COMMON) | {'src/bin/measure.rs'}
            if kind == 'helper':
                names.update(core_identity.EXTRA)
                names.update(core_identity.REQUIRED)
            if kind == 'host':
                names.update(native.HOST_EXTRA)
                names.update('../../internal/model/' + name for name in native.CHOICE)
                names.update('../../internal/backends/wasm/toolchain/' + name for name in native.RUNTIME)
            for name in names:
                path = (base / name).resolve()
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(b'synthetic source: ' + str(path).encode())
        tools = root / 'tools'
        tools.mkdir()
        for name in ('cargo', 'rustc'):
            (tools / name).write_bytes(b'synthetic compiler ' + name.encode())
        attempt = root / 'attempt'
        logs = attempt / 'release-native-123-0'
        executable = logs / 'target/release/gomad-measure'
        executable.parent.mkdir(parents=True)
        executable.write_bytes(b'synthetic native executable')
        executable.chmod(0o755)
        config = dict(repo=str(repo), root=str(attempt), files={str(path):sha(path) for path in root.rglob('*') if path.is_file() and not path.is_relative_to(attempt)})
        role = dict(kind='release')
        version = 'rustc 1.99.1 (synthetic)\nhost: aarch64-unknown-linux-gnu\n'
        inventories = {kind:native.source_inventory(config, kind)[1] for kind in ('host', 'helper')}
        runtime = native.hashlib.sha256(b'gomad-wasm-runtime-implementation/v1\0')
        for name in native.RUNTIME:
            raw = inventories['host']['../../internal/backends/wasm/toolchain/' + name]
            runtime.update(len(raw).to_bytes(8, 'big') + raw)
        identity = dict(schema='gomad.native-measurement-identity/v1', role='ordinary', features=[],
            host_implementation_sha256=native.implementation(inventories['host'], b'gomad.host-implementation/v1\0', [], version.splitlines()[0]+'\n', 'aarch64-unknown-linux-gnu'),
            linked_implementation_sha256=native.implementation(inventories['helper'], b'gomad.linked-implementation/v1\0', [], version.splitlines()[0]+'\n', 'aarch64-unknown-linux-gnu'),
            runtime_implementation_sha256=runtime.hexdigest(), configuration=native.BASE_CONFIGURATION,
            source_inputs={kind:native.source_manifest(files) for kind,files in inventories.items()}, executable_sha256=sha(executable))
        semantic = native.validate_identity(config, identity, role, sha(executable), version)
        environment = dict(PATH='/tools', CARGO_TARGET_DIR=str(logs/'target'), RUSTC=str(tools/'rustc'), RUSTC_WRAPPER='', RUSTC_WORKSPACE_WRAPPER='')
        crate = str(repo/'tools/gomad/rust/gomad')
        command = native.cargo_command(str(tools/'cargo'), crate, role)
        frames = [dict(reason='compiler-artifact', target=dict(name='gomad-measure', kind=['bin'], src_path=crate+'/src/bin/measure.rs'),
                       executable=str(executable), features=[], profile=dict(opt_level='3', debug_assertions=False, test=False)),
                  dict(reason='build-finished', success=True)]
        commands = {'rustc-path':['rustup','which','--toolchain','1.99.1','rustc'], 'cargo-path':['rustup','which','--toolchain','1.99.1','cargo'],
                    'rustc-version':[str(tools/'rustc'),'-vV'], 'cargo-version':[str(tools/'cargo'),'--version'],
                    'cargo-build':command, 'artifact-identity':[str(executable),'--identity']}
        outputs = {'rustc-path':str(tools/'rustc')+'\n', 'cargo-path':str(tools/'cargo')+'\n', 'rustc-version':version,
                   'cargo-version':'cargo 1.99.1 (synthetic)\n', 'cargo-build':'\n'.join(json.dumps(frame) for frame in frames),
                   'artifact-identity':json.dumps(identity)}
        for index, (name, argv) in enumerate(commands.items()):
            actual_env = environment if name in ('cargo-build','artifact-identity') else {key:value for key,value in environment.items() if not key.startswith('RUSTC')}
            value = dict(schema='gomad.native-process-observation/v1', command=argv, directory=str(logs) if name=='artifact-identity' else crate,
                         environment=actual_env, pid=100+index, parent_pid=99, clock='CLOCK_MONOTONIC', started_monotonic_nanos=900, reaped_monotonic_nanos=1000+index, exit_code=0, signal=None,
                         duration_nanos=1, termination='exit', cleanup_errors=[], error=None, resource_usage=None)
            for stream in ('stdout','stderr'):
                path = logs/(name+'.'+stream)
                path.write_text(outputs[name] if stream=='stdout' else '')
                size = path.stat().st_size
                value[stream] = dict(path=str(path), bytes=size, captured_bytes=size, sha256=sha(path), truncated=False, error=None)
                value[stream+'_sha256'] = sha(path)
            (logs/(name+'.json')).write_text(json.dumps(value))
        inputs = dict(semantic)
        inputs.update({str(path):sha(path) for path in tools.iterdir()})
        inputs.update({str(path):sha(path) for path in logs.iterdir() if path.is_file()})
        receipt = dict(schema='gomad.native-measurement-build/v1', crate_root=crate, executable=str(executable), executable_sha256=sha(executable),
                       rustc=str(tools/'rustc'), cargo=str(tools/'cargo'), inputs=inputs, command=command, environment=environment,
                       role=role, rustc_version=version, cargo_version=outputs['cargo-version'].strip()+'\n', identity=identity)
        path = logs/'helper-build.json'
        path.write_text(json.dumps(receipt))
        return config, path, receipt, role

    def test_full_synthetic_build_and_mutated_receipts(self):
        with tempfile.TemporaryDirectory() as temporary:
            config, path, original, role = self.fixture(Path(temporary))
            native.validate_build(config, path, role)
            for field in ('command','environment','role','identity','inputs'):
                changed = copy.deepcopy(original)
                if field == 'command': changed[field][1] = 'check'
                if field == 'environment': changed[field]['RUSTC_WRAPPER'] = '/unsealed-wrapper'
                if field == 'role': changed[field] = dict(kind='warm')
                if field == 'identity': changed[field]['host_implementation_sha256'] = '0'*64
                if field == 'inputs': changed[field].pop(str(path.parent/'cargo-build.json'))
                path.write_text(json.dumps(changed))
                with self.subTest(field=field), self.assertRaises(Refusal):
                    native.validate_build(config, path, role)

    def test_native_command_lifecycle_requires_actual_clock_start_and_same_parent(self):
        for field,bad in (('clock','CLOCK_REALTIME'),('started_monotonic_nanos',None),
                          ('started_monotonic_nanos',2000),('parent_pid',0)):
            with self.subTest(field=field,bad=bad),tempfile.TemporaryDirectory() as temporary:
                config,path,receipt,role=self.fixture(Path(temporary))
                raw=path.parent/'cargo-build.json';row=json.loads(raw.read_text());row[field]=bad
                raw.write_text(json.dumps(row))
                with self.assertRaises(Refusal):native.observation(raw,row['command'],row['directory'],row['environment'])
        with tempfile.TemporaryDirectory() as temporary:
            config,path,receipt,role=self.fixture(Path(temporary))
            raw=path.parent/'cargo-build.json';row=json.loads(raw.read_text());row['parent_pid']=777
            raw.write_text(json.dumps(row));receipt['inputs'][str(raw)]=sha(raw);path.write_text(json.dumps(receipt))
            config['files'][str(raw)]=sha(raw);config['files'][str(path)]=sha(path)
            with self.assertRaises(Refusal):native.validate_build(config,path,role)

    def test_missing_native_lifecycle_fields_refuse_without_key_errors(self):
        for field in ('clock','parent_pid','started_monotonic_nanos','reaped_monotonic_nanos'):
            with self.subTest(field=field),tempfile.TemporaryDirectory() as temporary:
                config,path,receipt,role=self.fixture(Path(temporary))
                raw=path.parent/'cargo-build.json';row=json.loads(raw.read_text());row.pop(field)
                raw.write_text(json.dumps(row))
                with self.assertRaises(Refusal):native.observation(raw,row['command'],row['directory'],row['environment'])

    def test_admission_reopens_copy_identity_and_refuses_changed_publication(self):
        with tempfile.TemporaryDirectory() as temporary:
            config, path, receipt, role = self.fixture(Path(temporary))
            destination = path.parent / 'admitted-native-123-1'
            destination.mkdir()
            executable = destination / 'gomad-measure'
            executable.write_bytes(Path(receipt['executable']).read_bytes())
            executable.chmod(0o755)
            for name, binary in (('artifact-identity', receipt['executable']), ('staged-identity', str(executable))):
                observation = native.read(path.parent / 'artifact-identity.json')
                observation.update(command=[binary, '--identity'], directory=str(destination))
                for stream in ('stdout', 'stderr'):
                    stream_path = destination / (name + '.' + stream)
                    stream_path.write_bytes((path.parent / ('artifact-identity.' + stream)).read_bytes())
                    observation[stream]['path'] = str(stream_path)
                (destination / (name + '.json')).write_text(json.dumps(observation))
            admission = dict(schema='gomad.native-measurement-admission/v1', build_receipt=str(path), build_receipt_sha256=sha(path),
                cargo_artifact=receipt['executable'], cargo_artifact_sha256=receipt['executable_sha256'], executable=str(executable),
                executable_sha256=receipt['executable_sha256'], identity=receipt['identity'])
            published = destination / 'helper-admission.json'
            published.write_text(json.dumps(admission))
            native.validate_admission(config, published, role)
            for field in ('build_receipt_sha256', 'cargo_artifact_sha256', 'executable_sha256', 'identity'):
                changed = copy.deepcopy(admission)
                changed[field] = {} if field == 'identity' else '0'*64
                published.write_text(json.dumps(changed))
                with self.subTest(field=field), self.assertRaises(Refusal):
                    native.validate_admission(config, published, role)
            published.write_text(json.dumps(admission))
            executable.chmod(0o777)
            with self.assertRaises(Refusal):
                native.validate_admission(config, published, role)

    def test_later_failed_or_duplicate_cargo_completion_is_refused(self):
        with tempfile.TemporaryDirectory() as temporary:
            config, path, receipt, role = self.fixture(Path(temporary))
            output = path.parent / 'cargo-build.stdout'
            initial = output.read_text()
            for success in (False, True):
                output.write_text(initial + '\n' + json.dumps(dict(reason='build-finished', success=success)))
                with self.subTest(success=success), self.assertRaises(Refusal):
                    native.cargo_artifact(output, role, receipt['executable'], receipt['crate_root'])

    def test_forbidden_feature_combinations_and_duplicate_fields(self):
        for values in (['stress-jitter','diagnostic-profile'], ['guest-profile'], ['diagnostic-profile','diagnostic-profile']):
            with self.subTest(values=values), self.assertRaises(Refusal):
                native.features(dict(kind='diagnostic', features=values))
        with self.assertRaises(Refusal):
            native.decode('{"role":"ordinary","role":"diagnostic-profile"}')


if __name__ == '__main__':
    unittest.main()
