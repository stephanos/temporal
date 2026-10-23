import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import driver
import preparation
import test_runtime_binding as runtime_fixture
from preparation_fixture import complete,outer_seal


class PreparationClosure(unittest.TestCase):
    def setUp(self):
        self.binding=runtime_fixture.RuntimePreparationBinding()
        self.binding.setUp()
        self.addCleanup(self.binding.doCleanups)
        for name in ("sidecar","config","installation","provenance","root","reseal"):
            setattr(self,name,getattr(self.binding,name))

    def test_exact_complete_native_compiler_and_transitive_source_projection(self):
        compiler,sources=preparation.Inventory(self.config).project(self.sidecar,self.installation,self.provenance['inputs'])
        self.assertGreater(len(compiler),20)
        self.assertGreater(len(sources),8)
        self.assertNotIn(self.installation['compiler_path'],compiler)
        self.assertIn(str(self.root/'sdk/pkg/include/funcdata.h'),compiler)
        self.assertIn(str(self.root/'sdk/src/runtime/common.h'),sources)
        self.assertNotIn(str(self.root/'sdk/src/runtime/new_choice.go'),sources)

    def test_resealed_omitted_tool_header_runtime_and_transitive_source_refuse(self):
        original=copy.deepcopy(self.provenance)
        metadata_path=self.sidecar/preparation.PREFIX/'compile/metadata.json'
        original_metadata=metadata_path.read_bytes()
        for kind,suffix in [('compiler','/compile'),('compiler','/link'),('compiler','/funcdata.h'),('compiler','/gomad_choicewire_generated.go.txt'),
                            ('sources','/print.go'),('sources','/common.h'),('sources','/textflag.h'),('sources','/go.sum')]:
            self.provenance=copy.deepcopy(original)
            selected=[path for path in self.provenance['inputs'][kind] if path.endswith(suffix)]
            self.assertEqual(len(selected),1)
            del self.provenance['inputs'][kind][selected[0]]
            self.binding.provenance=self.provenance
            self.reseal()
            # Reseal the module-key projection too: omission must fail the independently
            # reconstructed inventory, rather than only a now-stale object-cache path.
            metadata=json.loads(original_metadata)
            fields=('compiler','compiler_path','compiler_sha256','go_version','sources','runtime','runtime_implementation','kind','source','directory','tags','build_flags','environment')
            key=preparation.digest(preparation.canonical(dict(schema='gomad.native-compilation-inputs/v1',profile=self.installation['profile'],**{key:self.provenance['inputs'][key] for key in fields})))
            next(row for row in metadata['commands'] if row['name']=='compile-module')['environment']['GOCACHE']=str(Path(self.installation['cache_root'])/'compilation-v1'/key/'objects')
            metadata_path.write_bytes(preparation.canonical(metadata))
            with self.subTest(kind=kind,suffix=suffix),self.assertRaises(driver.Refusal):
                preparation.Inventory(self.config).project(self.sidecar,self.installation,self.provenance['inputs'])

    def test_missing_command_raw_stream_snapshot_duplicate_pid_and_changed_env_refuse(self):
        metadata_path=self.sidecar/preparation.PREFIX/'compile/metadata.json'
        original=json.loads(metadata_path.read_bytes())
        edits=[lambda value:value['commands'].pop(),
               lambda value:value['commands'][1].update(pid=value['commands'][0]['pid']),
               lambda value:value['commands'][1].update(exit=1),
               lambda value:value['commands'][1].update(start_ns=0),
               lambda value:value['commands'][1].update(stdout='f'*64),
               lambda value:value['commands'][1]['environment'].update(SECRET_TOKEN='forged'),
               lambda value:value['commands'][1]['environment'].update(GOROOT='/foreign'),
               lambda value:value['snapshots'].pop('compiler-base-paths')]
        for edit in edits:
            value=copy.deepcopy(original);edit(value);metadata_path.write_bytes(preparation.canonical(value))
            with self.subTest(edit=edit),self.assertRaises((driver.Refusal,KeyError)):
                preparation.Inventory(self.config).project(self.sidecar,self.installation,self.provenance['inputs'])

    def test_outer_actual_command_seal_binds_exact_sidecar_set_owner_and_prepared_artifact(self):
        import receiving
        row=outer_seal(self.binding)
        reader=receiving.Retained(self.config)
        reader.capture_preparations(row)
        campaign=Path(row['private_observations']['campaigns'][0]['path'])
        sidecar,owner=reader.prepared_observation(campaign,self.provenance)
        self.assertEqual(owner,1234)
        self.assertEqual(sidecar,self.binding.sidecar)
        mutations=[lambda value:value['private_observations'].update(binary_sha256='f'*64),
                   lambda value:value.update(pid=5678),
                   lambda value:value['private_observations']['campaigns'].append(copy.deepcopy(value['private_observations']['campaigns'][0])),
                   lambda value:value['private_observations']['campaigns'][0].update(receipt_sha256='f'*64),
                   lambda value:value['private_observations']['campaigns'][0]['files'].pop(next(iter(value['private_observations']['campaigns'][0]['files'])))]
        for edit in mutations:
            changed=copy.deepcopy(row);edit(changed)
            with self.subTest(edit=edit),self.assertRaises(driver.Refusal):
                receiving.Retained(self.config).capture_preparations(changed)
        extra=self.binding.sidecar/'extra';extra.write_bytes(b'unsealed')
        with self.assertRaises(driver.Refusal):receiving.Retained(self.config).capture_preparations(row)

    def test_runtime_receiver_rejects_complete_inventory_omissions_after_all_identity_resealing(self):
        import receiving
        original=copy.deepcopy(self.binding.provenance)
        metadata_path=self.sidecar/preparation.PREFIX/'compile/metadata.json'
        original_metadata=metadata_path.read_bytes()
        fields=('compiler','compiler_path','compiler_sha256','go_version','sources','runtime','runtime_implementation','kind','source','directory','tags','build_flags','environment')
        for kind,suffix in [('compiler','/compile'),('compiler','/link'),('compiler','/funcdata.h'),('compiler','/gomad_choicewire_generated.go.txt'),('sources','/print.go'),('sources','/common.h'),('sources','/textflag.h'),('sources','/go.sum')]:
            self.binding.provenance=copy.deepcopy(original)
            inputs=self.binding.provenance['inputs']
            selected=next(path for path in inputs[kind] if path.endswith(suffix))
            del inputs[kind][selected];self.binding.reseal()
            key=preparation.digest(preparation.canonical(dict(schema='gomad.native-compilation-inputs/v1',profile=self.installation['profile'],**{key:inputs[key] for key in fields})))
            metadata=json.loads(original_metadata)
            next(row for row in metadata['commands'] if row['name']=='compile-module')['environment']['GOCACHE']=str(Path(self.installation['cache_root'])/'compilation-v1'/key/'objects')
            metadata_path.write_bytes(preparation.canonical(metadata))
            with self.subTest(kind=kind,suffix=suffix),self.assertRaises(driver.Refusal):
                receiving.Retained(self.config).runtime_preparation(self.installation,self.binding.provenance,self.binding.record,self.binding.module,3,self.sidecar,1234)

    def test_final_endpoint_requires_current_producer_and_complete_campaign_preparation(self):
        import receiving
        row=outer_seal(self.binding)
        campaign=Path(row['private_observations']['campaigns'][0]['path'])
        reader=receiving.Retained(self.config)
        pending=(campaign,self.provenance,1234,frozenset([str(campaign)]))
        reader.pending_preparations=[pending]
        with self.assertRaises(driver.Unexecuted): reader.verify_pending_preparations()
        reader.capture_preparations(row)
        for owner,campaigns in [(5678,pending[3]),(1234,frozenset([str(campaign),str(campaign)+'-missing']))]:
            reader.pending_preparations=[(campaign,self.provenance,owner,campaigns)]
            with self.subTest(owner=owner,campaigns=campaigns),self.assertRaises(driver.Refusal):
                reader.verify_pending_preparations()
        reader.pending_preparations=[pending]
        reader.verify_pending_preparations()
        self.assertEqual(reader.pending_preparations,[])

    def test_observation_metadata_symlink_and_sparse_oversize_refuse_before_read(self):
        path=self.sidecar/preparation.PREFIX/'compile/metadata.json'
        original=path.read_bytes()
        target=path.with_name('alias.json');target.write_bytes(original)
        path.unlink();path.symlink_to(target.name)
        with self.assertRaises(driver.Refusal): preparation.Inventory(self.config).observed(self.sidecar,'compile')
        path.unlink()
        with path.open('wb') as stream: stream.truncate((16<<20)+1)
        with self.assertRaises(driver.Refusal): preparation.Inventory(self.config).observed(self.sidecar,'compile')
