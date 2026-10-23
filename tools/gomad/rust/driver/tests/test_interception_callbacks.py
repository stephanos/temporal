"""Adapter causality controls; no compiled-Go acceptance credit."""
import contextlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import interception_callbacks as gate


class CallbackAcceptance(unittest.TestCase):
    def fixture(self, root):
        return dict(schema='gomad.interception-callback-bindings/v1', main_build=str(root/'main.json'),
            harness_build=str(root/'harness.json'), installation=str(root/'installation.json'),
            closed_build=str(root/'fixture/main-build.json'), closed_installation=str(root/'fixture-installation.json'),
            inputs=str(root/'inputs.json'), streaming_root=str(root/'streaming'), closed_root=str(root/'closed'))

    def test_missing_or_substituted_proof_bindings_refuse(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            value = self.fixture(root)
            (root/'streaming').mkdir()
            recipe = {key:value[key] for key in ('main_build','harness_build','installation')}
            driver.write(root/'streaming/recipe.json', recipe)
            self.assertEqual(gate.bindings(value), recipe)
            for mutation in [dict(streaming_root='relative'), dict(closed_root=value['streaming_root']), dict(closed_installation=value['installation']),
                             dict(main_build=str(root/'other.json')), dict(schema='offline-proof')]:
                with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal):
                    gate.bindings(dict(value, **mutation))
            for key in value:
                with self.subTest(missing=key), self.assertRaises(driver.Refusal):
                    gate.bindings({name:item for name,item in value.items() if name != key})

    def test_distinct_fixture_role_and_installation_share_only_the_required_candidate(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            value = self.fixture(root)
            source = root/'source'
            source.write_bytes(b'frozen')
            fixture_binary = root/'fixture-gomad'
            fixture_binary.write_bytes(b'fixture')
            build_path = Path(value['closed_build'])
            build_path.parent.mkdir()
            build_path.write_bytes(b'actual typed fixture receipt')
            main = dict(head='head',source_sha256='source',symlinks={},helper_sha256='ordinary-helper',
                runtime_implementation_sha256='runtime',host_sha256='ordinary-host',binary='/ordinary/gomad')
            fixture = dict(main,host_sha256='fixture-host',binary=str(fixture_binary),
                binary_sha256=driver.sha(fixture_binary),inputs={str(source):driver.sha(source),
                    str(fixture_binary):driver.sha(fixture_binary)})
            installation = dict(compiler_path='/stock/bin/go',compiler_sha256='compiler',go_version='go1.27.1',
                profile='cooperative',runtime_implementation_sha256='runtime',admission=dict(compiler_inputs={},guest_configuration={}))
            driver.write(value['installation'],installation)
            driver.write(value['closed_installation'],dict(installation,host_sha256='fixture-host'))
            config = dict(files={str(source):driver.sha(source)})
            with patch.object(gate.main_cli_build,'read_build',return_value=fixture) as build, \
                 patch.object(gate.receiving.Retained,'installed'):
                closed, actual, _ = gate.closed_context(value,config,main)
                self.assertEqual(closed['binary'],str(fixture_binary))
                self.assertEqual(closed['identity'],dict(host='fixture-host',helper='ordinary-helper'))
                build.assert_called_once_with(value['closed_build'],expected_role='host-fixture')
                for key in ('head','source_sha256','symlinks','helper_sha256','runtime_implementation_sha256'):
                    changed = dict(fixture,**{key:'foreign'})
                    build.return_value = changed
                    with self.subTest(changed=key),self.assertRaises(driver.Refusal):
                        gate.closed_context(value,config,main)
                build.return_value = dict(fixture,host_sha256=main['host_sha256'])
                with self.assertRaises(driver.Refusal):gate.closed_context(value,config,main)
                build.return_value = fixture
                changed = dict(installation,compiler_sha256='other-compiler')
                Path(value['closed_installation']).write_text(json.dumps(changed))
                with self.assertRaises(driver.Refusal):gate.closed_context(value,config,main)

    def test_original_fixture_go_driver_preparation_lineage_is_required(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            receipt_path = root/'preparation.json'
            receipt = dict(schema='gomad.rust-preparation-receipt/v1',process_id=17,guest_executions=0,
                exit_code=0,command=['/fixture/gomad','run-guest','/compiled/module.wasm','stream'],module_sha256='module')
            driver.write(receipt_path,receipt)
            case = dict(receipt=str(receipt_path),receipt_sha256=driver.sha(receipt_path))
            inputs = root/'inputs.json'
            driver.write(inputs,dict(callback=case))
            argv = ['/stock/bin/go','run','-exec',"'/fixture/gomad' run-guest",'./cmd/gomad/testdata/seedfree','stream']
            encoded = [arg.encode().hex() for arg in argv]
            closed = dict(module_sha256='module',origin=dict(launch=dict(preparation_receipt_sha256=case['receipt_sha256']),
                invocation=dict(driver_argv=encoded,original_argv=encoded,compiler_sha256='compiler')))
            checker = Mock()
            checker.private_json.side_effect = driver.read
            arguments = (dict(inputs=str(inputs)),checker,dict(binary='/fixture/gomad'),
                dict(compiler_path='/stock/bin/go',compiler_sha256='compiler'),closed)
            gate.preparation_lineage(*arguments)
            for wrong in ['/ordinary/gomad','/other/fixture']:
                with self.subTest(launcher=wrong),self.assertRaises(driver.Refusal):
                    gate.preparation_lineage(arguments[0],checker,dict(binary=wrong),arguments[3],closed)
            for key,value in [('process_id',0),('guest_executions',1),('guest_executions',False),('exit_code',False),('command',['/ordinary/gomad','run-guest','stream'])]:
                receipt_path.write_text(json.dumps(dict(receipt,**{key:value})))
                case['receipt_sha256'] = driver.sha(receipt_path)
                inputs.write_text(json.dumps(dict(callback=case)))
                closed['origin']['launch']['preparation_receipt_sha256'] = case['receipt_sha256']
                with self.subTest(receipt=key),self.assertRaises(driver.Refusal):gate.preparation_lineage(*arguments)

    def test_closed_proof_projects_only_authenticated_origin_and_current_go_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root/'tools/gomad/cmd/gomad/testdata/seedfree/main.go'
            source.parent.mkdir(parents=True)
            source.write_bytes(b'current-source')
            compiler = root/'go/bin/go'
            compiler.parent.mkdir(parents=True)
            compiler.write_bytes(b'compiler')
            config = dict(root=str(root),repo=str(root),identity=dict(host='host',helper='helper'),
                files={str(source):driver.sha(source),str(compiler):driver.sha(compiler)})
            configuration = dict(profile='gomad.wasi-stock-exploratory/v1',runtime_sha256=None,
                model=None,engine='engine',limits=dict(fuel=1,memory_bytes=1))
            fields = ('arguments build_flags compiler compiler_path compiler_sha256 configuration directory environment '
                'go_version helper host kind module_metadata network runtime runtime_implementation scenario schema '
                'soft_assertion_report_only source sources tags').split()
            inputs = dict.fromkeys(fields)
            inputs.update(schema='gomad.rust-wasm-build-inputs/v1',kind='go-run',
                source='./cmd/gomad/testdata/seedfree',directory=str(root/'tools/gomad'),arguments=['stream'],
                go_version='go1.27.1',tags=['test_dep'],sources={str(source):driver.sha(source)},
                host='host',helper='helper',compiler_path=str(compiler),compiler_sha256=driver.sha(compiler),
                compiler={str(compiler):driver.sha(compiler)},configuration=configuration,runtime_implementation='runtime')
            module = b'wasm'
            record = dict(identity=dict(target=gate.receiving.digest(module)),streams={})
            origin = dict(schema='gomad.intercepted-origin/v1',origin='intercepted-go',invocation={},
                module_sha256=record['identity']['target'],preparation_sha256='preparation',launch={})
            provenance = dict(schema='gomad.rust-wasm-provenance/v1',inputs=inputs,configuration=configuration,
                module_metadata=None,build_key=gate.receiving.digest(gate.receiving.canonical(inputs)),
                module_sha256=record['identity']['target'],choice_identity=None,network=None,scenario=None,origin=origin)
            artifact = root/'artifact'
            (artifact/'target').mkdir(parents=True)
            (artifact/'target/module.wasm').write_bytes(module)
            driver.write(artifact/'manifest.json',dict(schema='gomad.artifact-manifest/v2',record=record,
                record_sha256=gate.receiving.digest(gate.receiving.canonical(record)),
                files={'target/module.wasm':dict(bytes=len(module),sha256=record['identity']['target'],module=True)}))
            path = artifact/'target/provenance.json'
            path.write_text(json.dumps(provenance))
            observation = dict(artifact=str(artifact),record_sha256=driver.read(artifact/'manifest.json')['record_sha256'],
                module_sha256=record['identity']['target'],origin=origin)
            installation = dict(profile=configuration['profile'],compiler_path=str(compiler),
                compiler_sha256=driver.sha(compiler),runtime_implementation_sha256='runtime',
                admission=dict(compiler_inputs={str(compiler):driver.sha(compiler)}))
            # Shared qualification remains closed to intercepted artifacts.
            with self.assertRaises(driver.Refusal):
                gate.receiving.Retained(config).provenance(dict(provenance=provenance))
            original_raw = path.read_bytes()
            gate.closed_candidate(config,installation,observation)
            self.assertEqual(path.read_bytes(),original_raw)
            with self.assertRaises(driver.Refusal):
                gate.receiving.Retained(config).provenance(dict(provenance=driver.read(path)))
            for mutation in [dict(sources={str(source):'old-source'}),dict(arguments=['success']),
                             dict(directory='/different/checkout'),dict(tags=[]),dict(compiler_path='/other/go'),
                             dict(runtime_implementation='other')]:
                changed = dict(provenance,inputs=dict(inputs,**mutation))
                changed['build_key'] = gate.receiving.digest(gate.receiving.canonical(changed['inputs']))
                path.write_text(json.dumps(changed))
                with self.subTest(mutation=mutation),self.assertRaises(driver.Refusal):
                    gate.closed_candidate(config,installation,observation)
            path.write_text(json.dumps(dict(provenance,origin={'launch':'borrowed'})))
            with self.assertRaises(driver.Refusal):gate.closed_candidate(config,installation,observation)

    def test_both_raw_readers_and_final_typed_roles_are_mandatory(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            value = self.fixture(root)
            config = dict(files={value['closed_installation']:'installation-hash'},identity=dict(host='host',helper='helper'))
            main = dict(binary='/actual/gomad',source_sha256='source')
            checker = Mock()
            checker.main.return_value = dict(artifact='actual raw artifact',
                origin=dict(invocation=dict(installation_sha256='installation-hash')))
            checker_path = gate.HERE.parents[1]/'testdata/parity/interception-callback-check.py'
            config['files'][str(checker_path)] = driver.sha(checker_path)
            with contextlib.ExitStack() as stack:
                stack.enter_context(patch.object(gate,'bindings',return_value={}))
                stack.enter_context(patch.object(gate.streaming,'context',return_value=(config,main,{}, {}, {})))
                stack.enter_context(patch.object(gate,'closed_context',return_value=(config,dict(binary='/fixture/gomad'),{})))
                controlled = stack.enter_context(patch.object(gate.streaming,'evaluate',return_value={'raw':'controlled'}))
                stack.enter_context(patch.object(gate,'load_module',return_value=checker))
                stack.enter_context(patch.object(gate,'preparation_lineage'))
                candidate = stack.enter_context(patch.object(gate,'closed_candidate'))
                stack.enter_context(patch.object(gate,'check_files'))
                roles = stack.enter_context(patch.object(gate.main_cli_build,'read_build'))
                result = gate.evaluate(value)
                self.assertEqual(result['required'],list(gate.REQUIRED))
                controlled.assert_called_once_with(root/'streaming')
                self.assertIn('--read-only',checker.main.call_args.args[0])
                self.assertIn('/fixture/gomad',checker.main.call_args.args[0])
                self.assertIn(value['closed_installation'],checker.main.call_args.args[0])
                candidate.assert_called_once()
                self.assertEqual([call.kwargs['expected_role'] for call in roles.call_args_list],
                                 ['release','interception-harness','host-fixture'])
                controlled.side_effect = driver.Refusal('missing raw controlled proof')
                with self.assertRaises(driver.Refusal):gate.evaluate(value)
                controlled.side_effect = None
                checker.main.side_effect = ValueError('missing raw closed proof')
                with self.assertRaises(ValueError):gate.evaluate(value)


if __name__ == '__main__':unittest.main()
