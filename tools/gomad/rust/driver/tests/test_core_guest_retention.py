"""Same-artifact Go lifecycle retention; no subprocess, compiler, or guest execution."""
import copy
from pathlib import Path
import sys
import unittest
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import core_guest_retention as gate
import test_core_retained_matrix as matrix_tests
from driver import Refusal


class ExactGuestMutations(unittest.TestCase):
    def test_changed_import_and_runtime_preserve_original_field_plus_suffix(self):
        baseline=dict(payloads={gate.FILE:gate.canonical(dict(configuration=dict(incarnations=[dict(imports=[dict(name='fd_write')],choice_identity=dict(ToolchainBuildKey='original-key'))])))},
            record=dict(streams={gate.FILE:dict(bytes=0,sha256='',records=0)},identity={},execution_identity=''),
            manifest=dict(files={gate.FILE:dict(module=False),'record.json':dict(module=False)},record={},record_sha256=''))
        for case,field in (('changed-imports','imports'),('changed-runtime-binding','choice_identity')):
            payloads,_=gate.expected_mutation(baseline,case)
            changed=gate.decoded(payloads[gate.FILE])['configuration']['incarnations'][0]
            self.assertEqual(changed['imports'][0]['name'] if field=='imports' else changed['choice_identity']['ToolchainBuildKey'],
                             'fd_write-changed' if field=='imports' else 'original-key-changed')
            self.assertEqual(gate.decoded(baseline['payloads'][gate.FILE])['configuration']['incarnations'][0]['imports'][0]['name'],'fd_write')

    def test_plain_and_compressed_each_require_the_full_same_artifact_matrix(self):
        self.assertEqual(len(gate.VARIANTS),27)
        self.assertEqual(len(set(gate.VARIANTS)),27)
        self.assertIn('primary-empty',gate.VARIANTS)
        self.assertIn('module-independent',gate.VARIANTS)
        with self.assertRaises(Refusal):gate.receipts('',{})


    def raw(self, before, payloads=None, manifest=None):
        payloads = dict(before['payloads'] if payloads is None else payloads)
        manifest = before['manifest'] if manifest is None else manifest
        module = payloads.pop('target/module.wasm', None)
        references = {} if module is None else {'target/module.wasm':dict(bytes=len(module),sha256=gate.digest(module),source='original-artifact')}
        return dict(manifest_base64=matrix_tests.encoded(gate.canonical(manifest)),
            record_base64=matrix_tests.encoded(payloads['record.json']),
            payloads_base64={name:matrix_tests.encoded(raw) for name,raw in payloads.items()},module_preimages=references)

    def test_actual_module_reference_is_bound_to_original_bytes_and_descriptor(self):
        before = gate.matrix.artifact(matrix_tests.fixture())
        raw = self.raw(before)
        self.assertEqual(gate.artifact(raw,before)['payloads'],before['payloads'])
        for forgery in ('hash','bytes','source','duplicate','missing','metadata'):
            changed=copy.deepcopy(raw)
            if forgery in ('hash','bytes','source'):
                key={'hash':'sha256','bytes':'bytes','source':'source'}[forgery]
                changed['module_preimages']['target/module.wasm'][key] = {'hash':'0'*64,'bytes':9,'source':'another-artifact'}[forgery]
            elif forgery=='duplicate':changed['payloads_base64']['target/module.wasm']=matrix_tests.encoded(before['payloads']['target/module.wasm'])
            elif forgery=='missing':changed['module_preimages']={}
            else:changed['payloads_base64'][gate.FILE]=matrix_tests.encoded(b'{}')
            with self.subTest(forgery=forgery),self.assertRaises(Refusal):gate.artifact(changed,before)

    def test_native_empty_and_foreign_primary_mutate_exact_identity_and_record_only(self):
        before=gate.matrix.artifact(matrix_tests.fixture())
        for case,target in (('primary-empty',''),('primary-foreign','0'*64)):
            payloads,manifest=gate.expected_mutation(before,case)
            after=gate.artifact(self.raw(before,payloads,manifest),before)
            self.assertEqual(after['record']['identity']['target'],target)
            self.assertEqual(after['record']['execution_identity'],gate.digest(gate.canonical(after['record']['identity'])))
            self.assertEqual({key:raw for key,raw in payloads.items() if key!='record.json'},
                             {key:raw for key,raw in before['payloads'].items() if key!='record.json'})
            self.assertEqual(manifest['record_sha256'],gate.digest(payloads['record.json']))
            self.assertEqual(before['record']['identity']['target'],gate.digest(before['payloads']['target/module.wasm']))

    def test_module_independence_requires_unchanged_metadata_and_actual_original_bound(self):
        before=gate.matrix.artifact(matrix_tests.fixture())
        payloads={name:raw for name,raw in before['payloads'].items() if name!='target/module.wasm'}
        manifest=copy.deepcopy(before['manifest']);manifest['files'].pop('target/module.wasm')
        receipt=dict(artifact=self.raw(before,payloads,manifest),read_inputs_base64=matrix_tests.encoded(payloads[gate.FILE]),
                     bound_bytes=len(payloads['record.json'])*4//3+(64<<10),original_module_absent=True,original_installation_absent=True)
        self.assertEqual(gate.unavailable(receipt,before)['removed_module_bytes'],8)
        for forgery in ('read','bound','module-present','producer-present','installation-present'):
            changed=copy.deepcopy(receipt)
            if forgery=='read':changed['read_inputs_base64']=matrix_tests.encoded(payloads[gate.FILE]+b' ')
            elif forgery=='bound':changed['bound_bytes']+=1
            elif forgery=='module-present':changed['artifact']=self.raw(before)
            elif forgery=='producer-present':changed['original_module_absent']=False
            else:changed['original_installation_absent']=False
            with self.subTest(forgery=forgery),self.assertRaises(Refusal):gate.unavailable(changed,before)

    def test_complete_catalog_cannot_borrow_another_profile_or_original_artifact(self):
        rows=[dict(case=case,variant=variant) for case in gate.ROOTS for variant in gate.VARIANTS]
        for changed in (rows[:-1],rows+[rows[0]],[rows[1],rows[0],*rows[2:]]):
            stdout='\n'.join(gate.MARKER+gate.canonical(row).decode() for row in changed)
            with self.assertRaises(Refusal):gate.receipts(stdout,{})
        before=gate.matrix.artifact(matrix_tests.fixture())
        originals={case:dict(path='/actual/'+case,artifact=before) for case in gate.ROOTS}
        first=rows[0]
        first.update(schema=gate.SCHEMA,original_root=gate.ROOTS[first['case']],original_artifact='/foreign/artifact',
                     record_sha256=gate.digest(before['payloads']['record.json']),module_sha256=gate.digest(before['payloads']['target/module.wasm']),observed={})
        stdout='\n'.join(gate.MARKER+gate.canonical(row).decode() for row in rows)
        with self.assertRaisesRegex(Refusal,'borrowed another'):gate.receipts(stdout,originals)


if __name__=='__main__':unittest.main()
