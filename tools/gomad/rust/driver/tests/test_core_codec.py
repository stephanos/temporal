"""Adversarial pure codec facts; no native builds or target preparation."""
import copy
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_codec as gate
from driver import Refusal


def operation(identity):
    return dict(ID=identity, Kind='write', Dependencies=None, Inode=0, Offset=0,
                Bytes=0, PayloadSHA256='', EffectSHA256='', Path='', Destination='')


def descriptor_model(pending,transitions,volumes=None):
    volumes=volumes or [dict(ID='data',Path='/workspace',CapacityBytes=1<<20)]
    model=dict(volumes=volumes,limits=dict(PendingOperations=pending,Transitions=transitions),faults=None,sqlite_lock_databases=None)
    return dict(model=model,model_identity='sha256:'+gate.digest(gate.struct_json(dict(volumes=volumes,limits=model['limits']))))


def context():
    second = operation(2); second['Dependencies'] = [1]
    return {'data':dict(PendingSHA256='sha256:'+'a'*64, Operations=[operation(1), second])}


class ContextAdmission(unittest.TestCase):
    def test_selection_uses_dependency_closure_and_prior_observation(self):
        self.assertIsNone(gate.context_refusal(['data'], 100, {'data':[1,2]}, context()))
        self.assertEqual(gate.context_refusal(['data'], 100, {'data':[2]}, context()),
                         'WASI invalid: storage crash selection is not dependency-closed')
        self.assertEqual(gate.context_refusal(['data'], 100, {'data':[1]}, None),
                         'WASI invalid: storage crash selection requires prior observation')

    def test_selection_precedes_invalid_inventory_and_sorted_volumes(self):
        raw = context(); raw['data']['PendingSHA256'] = 'forged'
        self.assertEqual(gate.context_refusal(['data'], 100, {'data':[3]}, raw),
                         'WASI invalid: storage crash selection unknown operation')
        self.assertEqual(gate.context_refusal(['a','b'], 100, {'b':[1,1],'a':[0]}, None),
                         'WASI invalid: storage crash selection zero operation')

    def test_null_inventory_and_bad_dependency_refuse(self):
        raw = context(); raw['data']['Operations'] = None
        self.assertEqual(gate.context_refusal(['data'], 100, {}, raw),
                         'WASI invalid: storage crash context inventory')
        raw = context(); raw['data']['Operations'][0]['Dependencies'] = [2]
        self.assertEqual(gate.context_refusal(['data'], 100, {}, raw),
                         'WASI invalid: storage crash context dependency')

    def test_typed_integer_and_unknown_fields_fail_closed(self):
        for field,value in [('ID',True),('Dependencies',[False]),('Extra','unknown')]:
            raw=context();raw['data']['Operations'][0][field]=value
            with self.subTest(field=field),self.assertRaises(Refusal):
                gate.context_refusal(['data'],100,{},raw)

    def test_transport_authenticates_exact_wire_and_sequence(self):
        raw=b'{"data":{"Operations":[]}}'
        frames=gate.context_frames(raw)
        declared=dict(bytes=len(raw),chunks=1,sha256=gate.digest(raw))
        self.assertEqual(gate.context_transport(frames,declared),raw)
        for changed in (frames.replace(b'"sequence":0',b'"sequence":1'),frames+b'{}\n',frames[:-1]):
            with self.assertRaises(Refusal):gate.context_transport(changed,declared)
        declared['sha256']='0'*64
        with self.assertRaises(Refusal):gate.context_transport(frames,declared)


if __name__=='__main__':unittest.main()


class FrozenOrderCatalog(unittest.TestCase):
    def test_frozen_catalog_derives_each_refusal_and_requires_both_orders(self):
        import tempfile
        import json
        from driver import sha
        with tempfile.TemporaryDirectory() as temporary:
            repo=Path(temporary);path=repo/'tools/gomad/testdata/parity/storage-crash-validation-order.json'
            path.parent.mkdir(parents=True)
            corpus=dict(schema='gomad.storage-crash-validation-order/v1',volumes=['a','b'],pending_operations=100,
                cases=[dict(name='zero',selection=[['b',[1,1]],['a',[0]]],context=None,
                            want='WASI invalid: storage crash selection zero operation')])
            path.write_text(json.dumps(corpus));config=dict(repo=str(repo),files={str(path):sha(path)})
            catalog=gate.order_catalog(config)
            self.assertEqual(set(catalog),{('storage-validation-order-corpus','zero',False),('storage-validation-order-corpus','zero',True)})
            rows=[]
            for reverse in (False,True):
                selection=dict(reversed(corpus['cases'][0]['selection']) if reverse else corpus['cases'][0]['selection'])
                row=dict(schema=gate.SCHEMA,case='storage-validation-order-corpus',
                    inputs=dict(name='zero',reversed=reverse,descriptor=json.dumps(dict(descriptor_model(100,100,[dict(ID=name,Path='/'+name,CapacityBytes=512) for name in ('a','b')]),crash_selection=selection,crash_context=None),separators=(',',':')),context=None),
                    observed=dict(refusal=corpus['cases'][0]['want']))
                self.assertIn(gate.storage_observation(config,row,catalog),catalog)
                bad=copy.deepcopy(row);bad['observed']['refusal']='PASS'
                with self.assertRaises(Refusal):gate.storage_observation(config,bad,catalog)
            path.write_text('{}')
            with self.assertRaises(Refusal):gate.order_catalog(config)

    def test_outer_reader_refuses_missing_cases_even_with_unit_pass(self):
        import tempfile
        import json
        from driver import sha
        with tempfile.TemporaryDirectory() as temporary:
            repo=Path(temporary);path=repo/'tools/gomad/testdata/parity/storage-crash-validation-order.json'
            path.parent.mkdir(parents=True);path.write_text(json.dumps(dict(schema='gomad.storage-crash-validation-order/v1',
                volumes=['a'],pending_operations=100,cases=[dict(name='zero',selection=[['a',[0]]],context=None,
                want='WASI invalid: storage crash selection zero operation')])))
            config=dict(repo=str(repo),files={str(path):sha(path)})
            with self.assertRaises(Refusal):gate.observations(config,'test everything ... ok\n')


class RecoveryImage(unittest.TestCase):
    def test_binary_recovery_image_rejects_truncation_alias_and_trailing(self):
        import struct
        def number(value):return struct.pack('<Q',value)
        def blob(raw):return number(len(raw))+raw
        model=b'sha256:'+b'a'*64
        # Empty valid native recovery image, including canonical root directory.
        obj=number(3)+number(0o700)+number(2)+number(0)+blob(b'')
        volume=blob(b'data')+blob(b'/workspace')+number(1024)+number(1)+obj+number(1)+blob(b'/')+number(3)
        volume+=number(1)+number(3)+number(0)+number(0)+number(1)+number(0)
        sections=[blob(model),number(0),number(4)+number(1)+number(1)+volume]
        raw=b'GMVOLIMG'+struct.pack('<II',1,3)+b''.join(struct.pack('<I',index)+blob(section) for index,section in enumerate(sections,1))
        value=gate.recovery_image(raw)
        self.assertEqual(value['volumes']['data']['paths'],{'/':3})
        for bad in (raw[:-1],raw+b'\0',raw.replace(b'GMVOLIMG',b'GMVOLIMX')):
            with self.assertRaises(Refusal):gate.recovery_image(bad)
        for length in range(len(raw)):
            with self.assertRaises(Refusal):gate.recovery_image(raw[:length])


class RetainedImageCorpus(unittest.TestCase):
    def test_every_retained_native_binary_image_decodes_with_exact_history(self):
        import json
        import base64
        root=Path(__file__).resolve().parents[5]
        corpus=json.loads((root/'tools/gomad/internal/model/wasi/testdata/volume-conformance/vectors/core.json').read_bytes())
        checked=0
        for scenario in corpus['scenarios']:
            for step in scenario['steps']:
                if 'image_base64' not in step:continue
                image=gate.recovery_image(base64.b64decode(step['image_base64'],validate=True))
                for volume in image['volumes'].values():
                    self.assertEqual(volume['next_operation'],len(volume['history'])+1)
                    self.assertTrue(all(operation in volume['history'] for operation in volume['pending']))
                checked+=1
        self.assertEqual(checked,280)


class ExactContextCases(unittest.TestCase):
    def test_all_original_dependency_hash_inventory_and_field_variants_are_exact(self):
        base=dict(PendingSHA256='sha256:'+'a'*64,Operations=[operation(2)])
        base['Operations'][0]['Dependencies']=[1]
        for variant in ('nil-dependencies','empty-dependencies','dependency','hash','nil-operations','empty-operations','field'):
            changed=copy.deepcopy(base)
            if variant in ('nil-dependencies','empty-dependencies','dependency'):
                changed['Operations'][0]['Dependencies']={'nil-dependencies':None,'empty-dependencies':[],'dependency':[3]}[variant]
            elif variant=='hash':changed['PendingSHA256']='sha256:'+'b'*64
            elif variant in ('nil-operations','empty-operations'):changed['Operations']=None if variant=='nil-operations' else []
            else:changed['Operations'][0]['EffectSHA256']='sha256:x'
            key=('storage-context-equality-original-dependency',variant)
            row=dict(schema=gate.SCHEMA,case=key[0],inputs=dict(variant=variant,base=base,changed=changed),
                     observed=dict(equal_forward=False,equal_reverse=False))
            with self.subTest(variant=variant):
                self.assertEqual(gate.storage_observation({},row,{key:None}),key)
                bad=copy.deepcopy(row);bad['observed']['equal_reverse']=0
                with self.assertRaises(Refusal):gate.storage_observation({},bad,{key:None})
                bad=copy.deepcopy(row);bad['inputs']['changed']=copy.deepcopy(base)
                with self.assertRaises(Refusal):gate.storage_observation({},bad,{key:None})

    def test_actual_5000_operation_wire_is_recomputed_before_digest_comparison(self):
        import json
        expected={'data':dict(PendingSHA256='sha256:'+'a'*64,Operations=[gate.operation(identity,'x'*256) for identity in range(1,5001)])}
        raw=gate.canonical(expected);typed=gate.struct_json(expected);wire=gate.context_frames(raw)
        descriptor=dict(descriptor_model(5000,10000),crash_selection={'data':[1]},crash_context=dict(bytes=len(raw),chunks=(len(raw)+65535)//65536,sha256=gate.digest(raw)))
        row=dict(schema=gate.SCHEMA,case='storage-large-context-transport',
            inputs=dict(descriptor=descriptor,transport_bytes=list(raw),wire_bytes=list(wire),typed_context_bytes=list(typed),
                        transport_sha256=gate.digest(raw),typed_context_sha256=gate.digest(typed)),
            observed=dict(context_sha256=gate.digest(typed),operation_ids=list(range(1,5001))))
        key=(row['case'],)
        self.assertEqual(gate.storage_observation({},row,{key:None}),key)
        row['inputs']['wire_bytes'][-2]=32
        with self.assertRaises(Refusal):gate.storage_observation({},row,{key:None})

    def test_retained_corpus_recovery_snapshot_hashes_match_independent_projection(self):
        import json
        import base64
        root=Path(__file__).resolve().parents[5]
        corpus=json.loads((root/'tools/gomad/internal/model/wasi/testdata/volume-conformance/vectors/core.json').read_bytes())
        checked=0
        for scenario in corpus['scenarios']:
            for step in scenario['steps']:
                snapshots=step.get('snapshots',[])
                if 'image_base64' not in step or not snapshots or any(snap['PendingOperations'] or snap['Persisted']!=snap['Volatile'] for snap in snapshots):continue
                image=gate.recovery_image(base64.b64decode(step['image_base64'],validate=True))
                self.assertEqual(gate.recovered_snapshots(image),snapshots,(scenario['name'],step['op']))
                checked+=1
        self.assertGreater(checked,100)
