"""Synthetic retained bytes exercise the pure reader; no compiler or guest executes."""
import copy
import json
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import native_observations as native
import preparation
import receiving
import terminal_module_receiving as terminal
from preparation_fixture import complete
import test_runtime_binding as runtime_fixture


class TerminalModuleReader(unittest.TestCase):
    def setUp(self):
        fixture = runtime_fixture.RuntimePreparationBinding()
        fixture.setUp()
        self.addCleanup(fixture.doCleanups)
        self.config, self.root = fixture.config, fixture.root
        self.stage = self.root / 'stages/terminal-module'
        self.output = self.stage / 'primary/module'
        self.output.mkdir(parents=True, mode=0o700)
        self.output.chmod(0o700)
        self.config['sent_epoch'] = 0
        self.config['stages'] = {'terminal-module': {'adapter': 'primary'}}
        self.binary = self.root / 'measure'
        self.binary.write_bytes(b'explicit synthetic ordinary role, never executed')
        self.config['binary'] = str(self.binary)
        self.config['binary_sha256'] = native.digest(self.binary.read_bytes())
        source = self.root / 'frozen-source-manifest'
        source.write_bytes(b'explicit synthetic source manifest')
        self.config.update(source_manifest=str(source), source_sha256=native.digest(source.read_bytes()))
        for path in (self.binary, source):
            self.config['files'][str(path)] = native.digest(path.read_bytes())
        self.spec = dict(schema='gomad.native-terminal-module-preparation-input/v1', repo=str(self.root),
                         stock_go=fixture.installation['compiler_path'], toolchain_cache=str(self.root/'cache'))
        spec_path = self.root / 'terminal-spec.json'
        spec_path.write_bytes(native.canonical(self.spec))
        self.config['terminal_module_spec'] = str(spec_path)
        self.config['files'][str(spec_path)] = native.digest(spec_path.read_bytes())
        self.provenance = fixture.provenance
        self.installation = fixture.installation
        self.provenance['inputs'].update(kind='go-test', source='./tests', directory=str(self.root), arguments=[],
             tags=['disable_grpc_modules','sqlite3_dotlk','test_dep'])
        self.provenance['inputs']['build_flags'].append('-p=1')
        self.provenance['inputs']['environment']['GOMAXPROCS'] = '1'
        # Complete the existing independently projected compiler fixture, then replace its selected source stream.
        old = fixture.sidecar
        import shutil
        shutil.rmtree(old)
        sidecar = complete(fixture)
        sidecar.rename(self.output/'preparation-observations')
        self.sidecar = self.output/'preparation-observations'
        frozen_installation=Path(self.config['installations']['runtime'])
        self.config['files'][str(frozen_installation)]=native.digest(frozen_installation.read_bytes())
        runtime_identity=terminal.runtime_identity(self.config)
        self.installation['runtime_implementation_sha256']=runtime_identity
        self.provenance['inputs']['runtime_implementation']=runtime_identity
        selected = self.root/'tests/main.go'
        selected.parent.mkdir()
        selected.write_bytes(b'package main\n')
        self.config['files'][str(selected)] = native.digest(selected.read_bytes())
        self.provenance['inputs']['sources'][str(selected)] = native.digest(selected.read_bytes())
        for phase in ('compile', 'admission'):
            path = self.sidecar/preparation.PREFIX/phase/'metadata.json'
            data = json.loads(path.read_bytes())
            for command in data['commands']:
                name = command['name']
                command['cwd'] = str(self.root/'tools/gomad') if name == 'runtime-overlay' else str(self.root)
                if name in ('selected-package','selected-source-closure','compile-module'):
                    flags = ['-mod=readonly','-pgo=off','-p=1','-overlay',str(self.root/'cache/runtime-overlay'/('e'*64)/'overlay.json'),
                             '-tags','disable_grpc_modules,sqlite3_dotlk,test_dep']
                    prefix = [self.spec['stock_go'],'test','-c'] if name == 'compile-module' else [self.spec['stock_go'],'list']
                    command['argv'] = prefix+flags+(['-trimpath','-buildvcs=false','-o',str(self.root/'cache/.rust-preparation-1234-0/module.wasm')] if name == 'compile-module'
                        else ['-deps','-json','-test'] if name == 'selected-source-closure' else [])+['./tests']
                if name == 'runtime-overlay':
                    raw = (self.sidecar/data['blobs'][command['stdout']]['path']).read_bytes()
                    value=json.loads(raw);value['implementation_sha256']=runtime_identity
                    raw=native.canonical(value);identity=native.digest(raw)
                    relative=f'{preparation.PREFIX}/{phase}/{identity}.bin'
                    (self.sidecar/relative).write_bytes(raw)
                    data['blobs'][identity]=dict(path=relative,bytes=len(raw),sha256=identity)
                    command['stdout']=identity
                if name == 'selected-source-closure':
                    old_raw = (self.sidecar/data['blobs'][command['stdout']]['path']).read_bytes()
                    packets = preparation.documents(old_raw)
                    packets.append(dict(Dir=str(selected.parent), GoFiles=['main.go'], ImportPath='synthetic/tests'))
                    raw = b'\n'.join(native.canonical(packet) for packet in packets)+b'\n'
                    identity = native.digest(raw)
                    relative = f'{preparation.PREFIX}/{phase}/{identity}.bin'
                    (self.sidecar/relative).write_bytes(raw)
                    data['blobs'][identity] = dict(path=relative, bytes=len(raw), sha256=identity)
                    command['stdout'] = identity
                if name == 'compile-module':
                    fields=('compiler','compiler_path','compiler_sha256','go_version','sources','runtime','runtime_implementation','kind','source','directory','tags','build_flags','environment')
                    key=native.digest(native.canonical(dict(schema='gomad.native-compilation-inputs/v1',profile=self.installation['profile'],**{key:self.provenance['inputs'][key] for key in fields})))
                    command['environment']['GOCACHE']=str(self.root/'cache/compilation-v1'/key/'objects')
            used = set(data['snapshots'].values()) | {command[stream] for command in data['commands'] for stream in ('stdout','stderr')}
            for identity in set(data['blobs'])-used:
                (self.sidecar/data['blobs'].pop(identity)['path']).unlink()
            path.write_bytes(native.canonical(data))
        self.installation['admission']['guest_configuration'] = terminal.original_configuration()
        self.installation.update(fuel=1<<50, memory_bytes=4<<30, build_tags=[])
        policy = terminal.original_configuration()
        policy.pop('CapturedInputs')
        model, namespace, environment, _ = receiving.inspection_policy_projection(policy)
        self.installation.update(model=model, namespace=namespace, storage=None, network=None)
        self.installation['admission'].update(environment=environment, arguments=[], explicit_helper=None, compile_cache_root=None,
             guest_configuration=terminal.original_configuration(),configuration_sha256=native.digest(native.canonical(terminal.original_configuration())),
             host_executable_sha256=self.config['binary_sha256'])
        self.provenance['configuration'] = terminal.normalized_configuration(self.installation, '0'*64)
        self.provenance['inputs']['configuration'] = dict(self.provenance['configuration'], runtime_sha256=None)
        imports = b''.join(bytes([13])+b'gomad_wasm_v1'+bytes([len(name)])+name+b'\0\0' for name in (b'config',b'decision',b'observation',b'finish',b'idle'))
        self.module = b'\0asm\1\0\0\0\1\7\1\x60\2\x7f\x7f\1\x7f'+bytes([2,len(imports)+1,5])+imports+b'\5\3\1\0\1'
        # Exact small module above uses a multi-byte section length when required.
        if len(imports)+1 >= 128:
            self.module = b'\0asm\1\0\0\0\1\7\1\x60\2\x7f\x7f\1\x7f'+bytes([2,(len(imports)+1)|128,1,5])+imports+b'\5\3\1\0\1'
        self.provenance['module_metadata'] = receiving.module_metadata(self.module)
        self.provenance['inputs']['module_metadata'] = self.provenance['module_metadata']
        self.provenance['module_sha256'] = native.digest(self.module)
        self.reseal()
        self.argv = [str(self.binary),'--prepare-terminal-module',str(spec_path),str(self.output)]
        self.stdout = native.canonical(dict(result=str(self.output/'terminal-module.json')))+b'\n'
        self.bootstrap = patch('primary_stages.native_bootstrap', return_value=str(self.binary))
        self.bootstrap.start()
        self.addCleanup(self.bootstrap.stop)

    def reseal(self):
        p=self.provenance
        p['build_key']=native.digest(native.canonical(p['inputs']))
        source=self.config['repo']+'/tools/gomad/internal/model/'
        hasher=__import__('hashlib').sha256(b'gomad-choice-implementation-source-v3')
        for name in ('choice/schema/choicewire.json','choice/schema/choicewire.go.tmpl','choice/schema/choicewire_runtime.go.tmpl','choice/schema/choicewire_test.go.tmpl','runtimesource/gomad_semantics.go.txt','choice/trace.go','choice/tape.go'):
            raw=Path(source+name).read_bytes();hasher.update(len(raw).to_bytes(8,'big'));hasher.update(raw)
        implementation=__import__('hashlib').sha256(b'gomad-choice-implementation-v2\0'+hasher.digest()+bytes.fromhex(p['build_key'])).hexdigest()
        p['configuration']['runtime_sha256']=implementation
        p['choice_identity'].update(ImplementationSHA256=list(bytes.fromhex(implementation)),ToolchainBuildKey=p['build_key'],TargetSHA256=list(bytes.fromhex(p['module_sha256'])))
        for name,raw in [('module.wasm',self.module),('module-provenance.json',native.canonical(p)),('installation.json',native.canonical(self.installation))]:
            path=self.output/name;path.write_bytes(raw);path.chmod(0o600)
        for path in self.sidecar.rglob('*'):
            path.chmod(0o700 if path.is_dir() else 0o600)
        self.sidecar.chmod(0o700)
        files={str(path.relative_to(self.sidecar)):native.snapshot(path,native.RAW_LIMIT,0o600) for path in self.sidecar.rglob('*') if path.is_file() and path.name!='receipt.json'}
        receipt=dict(schema='gomad.native-preparation-observation-receipt/v1',owner_pid=1234,host_sha256=self.config['identity']['host'],helper_sha256=self.config['identity']['helper'],prepared_record_sha256=native.digest(native.canonical(p)),module_sha256=p['module_sha256'],provenance_sha256=native.digest(native.canonical(p)),build_key=p['build_key'],compiler_sha256=native.digest(native.canonical(p['inputs']['compiler'])),sources_sha256=native.digest(native.canonical(p['inputs']['sources'])),files=files)
        (self.sidecar/'receipt.json').write_bytes(native.canonical(receipt));(self.sidecar/'receipt.json').chmod(0o600)
        observations=dict(schema='gomad.native-measurement-preparation-observations/v1',artifact_root=str(self.sidecar),record='module-provenance.json',record_sha256=receipt['prepared_record_sha256'],receipt='preparation-observations/receipt.json')
        path=self.output/'preparation-observations.json';path.write_bytes(native.canonical(observations));path.chmod(0o600)
        self.descriptor=dict(schema='gomad.native-terminal-module/v1',recipe=dict(kind='go-test',source='./tests',directory=str(self.root),build_tags=['disable_grpc_modules','test_dep','sqlite3_dotlk'],target_arguments=[]),configuration=terminal.original_configuration(),memory_bytes=4<<30,fuel=1<<50)
        for key,name in [('module','module.wasm'),('provenance','module-provenance.json'),('installation','installation.json'),('preparation_observations','preparation-observations.json'),('preparation_receipt','preparation-observations/receipt.json')]:
            self.descriptor[key]=str(self.output/name);self.descriptor[key+'_sha256']=native.digest((self.output/name).read_bytes())
        self.publish_descriptor()

    def publish_descriptor(self):
        path=self.output/'terminal-module.json';path.write_bytes(native.canonical(self.descriptor));path.chmod(0o600)

    def test_exact_module_captures_and_reads_actual_closed_outer_context(self):
        seal=terminal.capture(self.config,1234,self.argv,self.stdout)
        driver.write(self.stage/'primary-final.command.json',dict(argv=self.argv,cwd=str(self.root),pid=1234,exit=0,observation='completed',start_ns=1,end_ns=100000,environment={},temporary_paths={},private_observations=seal))
        driver.write(self.stage/'primary-final.process.json',dict(argv=self.argv,pid=1234,active=False,exit=0,observation='completed',stage_pid=5678,start_ns=1,end_ns=100000))
        (self.stage/'primary-final.stdout').write_bytes(self.stdout)
        self.assertEqual(terminal.read_module(self.config,self.stage,self.output/'terminal-module.json'),self.descriptor)

    def test_stale_attempt_process_times_and_postcapture_descriptor_swap_refuse(self):
        self.test_exact_module_captures_and_reads_actual_closed_outer_context()
        self.config['sent_epoch']=1
        with self.assertRaises(driver.Refusal):terminal.read_module(self.config,self.stage,self.output/'terminal-module.json')
        self.config['sent_epoch']=0
        path=self.stage/'primary-final.process.json'
        process=json.loads(path.read_bytes());process['start_ns']=2;path.write_bytes(native.canonical(process))
        with self.assertRaises(driver.Refusal):terminal.read_module(self.config,self.stage,self.output/'terminal-module.json')
        process['start_ns']=1;path.write_bytes(native.canonical(process))
    def test_postcapture_descriptor_swap_cannot_return_unvalidated_paths(self):
        self.test_exact_module_captures_and_reads_actual_closed_outer_context()
        original=terminal.capture
        def swapped(*args):
            seal=original(*args)
            self.descriptor['module']='/borrowed/after-capture.wasm'
            self.publish_descriptor()
            return seal
        with patch.object(terminal,'capture',side_effect=swapped),self.assertRaisesRegex(driver.Refusal,'changed after authenticated'):
            terminal.read_module(self.config,self.stage,self.output/'terminal-module.json')

    def test_original_tuple_rejects_slim_selection_bounds_and_unknown_fields(self):
        original=copy.deepcopy(self.descriptor)
        edits=[lambda d:d['recipe']['build_tags'].append('gomad_wasm_slim'),lambda d:d['recipe']['target_arguments'].append('-test.run=^Selected$'),lambda d:d['configuration']['Limits'].update(OutputBytes=256<<20),lambda d:d.update(fuel=1<<40),lambda d:d.update(unknown=True),lambda d:d.update(module='/borrowed/module.wasm')]
        for edit in edits:
            self.descriptor=copy.deepcopy(original);edit(self.descriptor);self.publish_descriptor()
            with self.subTest(edit=edit),self.assertRaises(driver.Refusal):terminal.capture(self.config,1234,self.argv,self.stdout)

    def test_raw_custody_rejects_extra_entries_links_permissions_owner_and_mutation(self):
        extra=self.sidecar/'extra';extra.write_bytes(b'unlisted')
        with self.assertRaises((driver.Refusal,ValueError)):terminal.capture(self.config,1234,self.argv,self.stdout)
        extra.unlink()
        blob=next(path for path in self.sidecar.rglob('*.bin'))
        blob.chmod(0o644)
        with self.assertRaises((driver.Refusal,ValueError)):terminal.capture(self.config,1234,self.argv,self.stdout)
        blob.chmod(0o600)
        alias=self.root/'alias';os.link(blob,alias)
        with self.assertRaises((driver.Refusal,ValueError)):terminal.capture(self.config,1234,self.argv,self.stdout)
        alias.unlink()
        with self.assertRaises((driver.Refusal,ValueError)):terminal.capture(self.config,5678,self.argv,self.stdout)
        blob.write_bytes(b'changed')
        with self.assertRaises((driver.Refusal,ValueError)):terminal.capture(self.config,1234,self.argv,self.stdout)

    def test_outer_selector_and_sealed_context_refuse_foreign_spec_argv_and_pid(self):
        for argv in ([str(self.binary),'--prepare-terminal-module'],
                     [str(self.binary),'--prepare-terminal-module',self.config['terminal_module_spec'],str(self.root/'foreign')]):
            with self.subTest(argv=argv),self.assertRaises(driver.Refusal):
                native.from_command(self.config,1234,argv,self.stdout)
        seal=native.from_command(self.config,1234,self.argv,self.stdout)
        driver.write(self.stage/'primary-final.command.json',dict(argv=self.argv,cwd=str(self.root),pid=1234,exit=0,observation='completed',start_ns=1,end_ns=100000,environment={},temporary_paths={},private_observations=seal))
        driver.write(self.stage/'primary-final.process.json',dict(argv=self.argv,pid=5678,active=False,exit=0,observation='completed',stage_pid=5678,start_ns=1,end_ns=100000))
        (self.stage/'primary-final.stdout').write_bytes(self.stdout)
        with self.assertRaises(driver.Refusal):terminal.read_module(self.config,self.stage,self.output/'terminal-module.json')

    def test_resealed_runtime_policy_and_unknown_installation_fields_refuse(self):
        old=copy.deepcopy(self.installation)
        self.installation['unexpected']=True
        self.reseal()
        with self.assertRaises(driver.Refusal):terminal.capture(self.config,1234,self.argv,self.stdout)
        self.installation=old
        self.provenance['configuration']['model']['diagnostics']=True
        self.provenance['inputs']['configuration']=dict(self.provenance['configuration'],runtime_sha256=None)
        self.reseal()
        with self.assertRaises(driver.Refusal):terminal.capture(self.config,1234,self.argv,self.stdout)

    def test_unfrozen_compiler_auxiliary_is_refused_instead_of_adopted(self):
        overlay=next(path for path in self.installation['admission']['compiler_inputs'] if path.endswith('overlay.json'))
        del self.config['files'][overlay]
        with self.assertRaisesRegex(driver.Refusal,'frozen'):terminal.capture(self.config,1234,self.argv,self.stdout)

    def test_frozen_compiler_membership_and_missing_cache_witness_refuse(self):
        self.installation['admission']['compiler_inputs'].pop(next(iter(self.installation['admission']['compiler_inputs'])))
        self.installation['admission']['inputs_sha256']=native.digest(native.canonical(self.installation['admission']['compiler_inputs']))
        self.reseal()
        with self.assertRaisesRegex(driver.Refusal,'differs from frozen installation'):
            terminal.capture(self.config,1234,self.argv,self.stdout)
        self.config['installations']={}
        with self.assertRaisesRegex(driver.Refusal,'no frozen current installation witness'):
            terminal.capture(self.config,1234,self.argv,self.stdout)
