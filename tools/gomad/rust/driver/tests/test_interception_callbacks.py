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
            inputs=str(root/'inputs.json'), streaming_root=str(root/'streaming'), closed_root=str(root/'closed'))

    def test_missing_or_substituted_proof_bindings_refuse(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            value = self.fixture(root)
            (root/'streaming').mkdir()
            recipe = {key:value[key] for key in ('main_build','harness_build','installation')}
            driver.write(root/'streaming/recipe.json', recipe)
            self.assertEqual(gate.bindings(value), recipe)
            for mutation in [dict(streaming_root='relative'), dict(closed_root=value['streaming_root']),
                             dict(main_build=str(root/'other.json')), dict(schema='offline-proof')]:
                with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal):
                    gate.bindings(dict(value, **mutation))
            for key in value:
                with self.subTest(missing=key), self.assertRaises(driver.Refusal):
                    gate.bindings({name:item for name,item in value.items() if name != key})

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
            config = dict(files={value['installation']:'installation-hash'},identity=dict(host='host',helper='helper'))
            main = dict(binary='/actual/gomad',source_sha256='source')
            checker = Mock()
            checker.main.return_value = dict(artifact='actual raw artifact',
                origin=dict(invocation=dict(installation_sha256='installation-hash')))
            checker_path = gate.HERE.parents[1]/'testdata/parity/interception-callback-check.py'
            config['files'][str(checker_path)] = driver.sha(checker_path)
            with contextlib.ExitStack() as stack:
                stack.enter_context(patch.object(gate,'bindings',return_value={}))
                stack.enter_context(patch.object(gate.streaming,'context',return_value=(config,main,{}, {}, {})))
                controlled = stack.enter_context(patch.object(gate.streaming,'evaluate',return_value={'raw':'controlled'}))
                stack.enter_context(patch.object(gate,'load_module',return_value=checker))
                candidate = stack.enter_context(patch.object(gate,'closed_candidate'))
                stack.enter_context(patch.object(gate,'check_files'))
                roles = stack.enter_context(patch.object(gate.main_cli_build,'read_build'))
                result = gate.evaluate(value)
                self.assertEqual(result['required'],list(gate.REQUIRED))
                controlled.assert_called_once_with(root/'streaming')
                self.assertIn('--read-only',checker.main.call_args.args[0])
                candidate.assert_called_once()
                self.assertEqual([call.kwargs['expected_role'] for call in roles.call_args_list],
                                 ['release','interception-harness'])
                controlled.side_effect = driver.Refusal('missing raw controlled proof')
                with self.assertRaises(driver.Refusal):gate.evaluate(value)
                controlled.side_effect = None
                checker.main.side_effect = ValueError('missing raw closed proof')
                with self.assertRaises(ValueError):gate.evaluate(value)


if __name__ == '__main__':unittest.main()
