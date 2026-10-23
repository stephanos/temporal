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


def encoded_image(image):
    import struct
    def number(value):return struct.pack('<Q',value)
    def blob(raw):return number(len(raw))+raw
    def text(value):return blob(value.encode())
    def numbers(values):return number(len(values))+b''.join(number(value) for value in values)
    def obj(value):return b''.join(number(value[field]) for field in ('inode','mode','kind','mod_time'))+blob(value['data'])
    def operations(values):
        raw=number(len(values))
        for value in values:
            effect=value['effect']
            raw+=number(value['id'])+text(value['kind'])+numbers(value['deps'])+number(len(value['stores']))+b''.join(text(store) for store in value['stores'])
            raw+=text(effect['kind'])+number(effect['inode'])+number(effect['parent'])+numbers(effect['parents'])
            raw+=b''.join(number(effect[field]) for field in ('offset','size','mode','mod_time'))+blob(effect['data'])
            raw+=number(effect['object'] is not None)+(obj(effect['object']) if effect['object'] is not None else b'')
            raw+=number(len(effect['namespace']))+b''.join(text(part['path'])+number(part['inode']) for part in effect['namespace'])
        return raw
    sections=[text(image['model']),number(len(image['occurrences']))+b''.join(text(key)+number(value) for key,value in sorted(image['occurrences'].items()))]
    raw=number(image['next_inode'])+number(image['generation'])+number(len(image['volumes']))
    for name,volume in sorted(image['volumes'].items()):
        raw+=text(name)+text(volume['mount'])+number(volume['capacity'])+number(len(volume['objects']))+b''.join(obj(value) for _,value in sorted(volume['objects'].items()))
        raw+=number(len(volume['paths']))+b''.join(text(path)+number(inode) for path,inode in sorted(volume['paths'].items()))+numbers(volume['owned'])
        raw+=operations(volume['pending'])+operations(volume['history'])+number(volume['next_operation'])
        raw+=number(len(volume['last']))+b''.join(text(key)+number(value) for key,value in sorted(volume['last'].items()))
    sections.append(raw)
    return b'GMVOLIMG'+struct.pack('<II',1,3)+b''.join(struct.pack('<I',index)+blob(value) for index,value in enumerate(sections,1))


def selected_fixture():
    import json
    import base64
    import struct
    root=Path(__file__).resolve().parents[5]
    corpus=json.loads((root/'tools/gomad/internal/model/wasi/testdata/volume-conformance/vectors/core.json').read_bytes())
    scenario=next(value for value in corpus['scenarios'] if value['name']=='captured-readonly-descendant-refuses-directory-rename')
    step=next(value for value in scenario['steps'] if value['op']=='api-mkdir')
    image=gate.recovery_image(base64.b64decode(step['image_base64']))
    model=descriptor_model(1000,10000);image['model']=model['model_identity'];volume=image['volumes']['data']
    volume.update(mount='/workspace',capacity=1<<20,pending=[],last={})
    volume['history'][1]['effect']['namespace'][0]['path']='/db'
    for op in volume['history']:
        op['stores']=[store.replace('/mutable','/db') for store in op['stores']]
    raw=encoded_image(image)
    context={'data':dict(PendingSHA256=gate.pending_identity('data',volume['history']),Operations=[gate.observed_operation(op) for op in volume['history']])}
    states=[]
    for chosen in ([],[1,2]):
        changed=copy.deepcopy(volume)
        for op in volume['history']:
            if op['id'] in chosen:gate.apply_effect(changed,op['effect'])
        projected=gate.entries(changed,True)
        states.append(dict(Volume='data',PendingSHA256=context['data']['PendingSHA256'],SelectedOperations=chosen,Entries=projected,Identity=gate.crash_identity('data',projected)))
    snapshot=gate.recovered_snapshots(image)[0];volatile=copy.deepcopy(volume)
    for op in volume['history']:gate.apply_effect(volatile,op['effect'])
    snapshot.update(Volatile=gate.entries(volatile),PendingOperations=2,PendingSHA256=context['data']['PendingSHA256'])
    snapshot['Identity']=gate.snapshot_identity(snapshot)
    pages={'data':dict(States=states,ChoiceExploration=None,Complete=True,Capacity='')}
    log=b'actual retained log'
    parts=[gate.struct_json([snapshot]),gate.struct_json({'data':context['data']['Operations']}),gate.struct_json(pages),
        gate.digest(raw).encode(),struct.pack('<Q',len(log))+bytes.fromhex(gate.digest(log)),gate.struct_json(context)]
    state=gate.state_envelope(3,parts)
    row=dict(schema=gate.SCHEMA,case='storage-selected-baseline',inputs=dict(context=context,descriptor=model),
        observed=dict(states=states,image_bytes=list(raw),state_bytes=list(state),log_bytes=list(log)))
    return row


class SelectedPersistence(unittest.TestCase):
    def test_selected_baseline_replays_distinct_states_and_snapshot_identities(self):
        row=selected_fixture();facts=gate.selected_baseline(row)
        self.assertEqual(len(facts['states']),2)
        for mutation in ('missing-state','forged-state','forged-context','forged-snapshot'):
            bad=copy.deepcopy(row)
            if mutation=='missing-state':bad['observed']['states'].pop()
            if mutation=='forged-state':bad['observed']['states'][0]['Identity']='sha256:'+'0'*64
            if mutation=='forged-context':bad['inputs']['context']['data']['Operations'][0]['Kind']='forged'
            if mutation=='forged-snapshot':
                raw=bytes(bad['observed']['state_bytes']);version,parts=gate.state_raw_sections(raw)
                snapshot=gate.decoded(parts[0]);snapshot[0]['Identity']='sha256:'+'0'*64;parts[0]=gate.canonical(snapshot)
                bad['observed']['state_bytes']=list(gate.state_envelope(version,parts))
            with self.subTest(mutation=mutation),self.assertRaises(Refusal):gate.selected_baseline(bad)


def evidence_fixture(baseline):
    import base64
    _,parts=gate.state_raw_sections(bytes(baseline['observed']['state_bytes']))
    descriptor=copy.deepcopy(baseline['inputs']['descriptor']);descriptor['crash_limits']=dict(States=16,Operations=8,Depth=8,Bytes=1<<20,Work=256)
    storage=dict(descriptor=descriptor)
    rows=[dict(schema=gate.SCHEMA,case='storage.native-original-state',inputs=dict(storage=storage),
        observed={name:baseline['observed'][name] for name in ('state_bytes','log_bytes','image_bytes')})]
    log=bytes(baseline['observed']['log_bytes'])
    def append(case,selector,variant,sections,error,version=3,request=None,observed_log=None):
        rows.append(dict(schema=gate.SCHEMA,case=case,inputs={selector:variant,
            'state_base64':base64.b64encode(gate.state_envelope(version,sections)).decode(),
            'log_base64':base64.b64encode(log if observed_log is None else observed_log).decode(),'storage':storage if request is None else request},observed=dict(error=error)))
    for edit in ('valid','snapshot-hash','operation-history','missing-inventory'):
        changed=parts.copy();context=gate.decoded(parts[5])
        if edit=='snapshot-hash':context['data']['PendingSHA256']='sha256:'+'b'*64
        if edit=='operation-history':context['data']['Operations'][0]['Kind']='forged'
        if edit=='missing-inventory':context['data']['Operations']=None
        changed[5]=gate.canonical(context)
        append('storage.native-state-context','edit',edit,changed,None if edit=='valid' else 'retained storage evidence: crash context')
    append('storage.native-state-context-presence','edit','v1-with-limits',parts[:5],'retained storage evidence: crash context',version=1)
    request=copy.deepcopy(storage);request['descriptor']['crash_limits']=None
    append('storage.native-state-context-presence','edit','v3-without-limits',parts,'retained storage evidence: crash context',request=request)
    append('storage.native-state-context-presence','edit','forged-log',parts,'retained storage log binding',observed_log=b'forged')
    for stage in (1,2,3):
        changed=parts.copy();changed[3]=b'';changed[5]=bytes([stage])
        append('storage.native-partial-state','stage',stage,changed,'retained storage failed stage' if stage==3 else None,version=2)
    variants={}
    value=gate.decoded(parts[0]);value[0]['Persisted']=[None];variants['snapshot-null-entry']=(0,gate.canonical(value),'invalid type: null')
    value=gate.decoded(parts[0]);value[0]['Identity']='x';variants['snapshot-identity']=(0,gate.canonical(value),'retained storage snapshot inventory')
    value=gate.decoded(parts[0]);value[0]['unexpected']=1;variants['snapshot-unknown-field']=(0,gate.canonical(value),'unknown field')
    variants['snapshot-duplicate-field']=(0,parts[0].replace(b'"ID":"data"',b'"ID":"data","ID":"data"',1),'duplicate JSON key')
    value=gate.decoded(parts[1]);value['data'][0]['unexpected']=1;variants['history-unknown-field']=(1,gate.canonical(value),'unknown field')
    variants['enumeration-json']=(2,b'{','EOF while parsing an object')
    value=gate.decoded(parts[2]);value['data']['States'][0]['Identity']='x';variants['enumeration-identity']=(2,gate.canonical(value),'retained storage crash state identity')
    value=gate.decoded(parts[2]);value['data']['unexpected']=1;variants['enumeration-unknown-field']=(2,gate.canonical(value),'unknown field')
    for name,section in (('history-duplicate-key',1),('enumeration-duplicate-key',2),('context-duplicate-key',5)):
        value=gate.canonical(gate.decoded(parts[section])['data']);variants[name]=(section,b'{"data":'+value+b',"data":'+value+b'}','duplicate JSON key')
    variants['image-identity']=(3,b'garbage','retained storage image identity');variants['image-uppercase']=(3,b'A'*64,'retained storage image identity')
    for variant,(section,raw,error) in variants.items():
        changed=parts.copy();changed[section]=raw;append('storage.native-resealed-section','mutation',variant,changed,error)
    return rows


class StateEvidenceVariants(unittest.TestCase):
    def test_all_context_presence_partial_and_resealed_variants_require_raw_preimages(self):
        baseline=selected_fixture();rows=evidence_fixture(baseline)
        facts=gate.evidence_observations(rows,[baseline])
        self.assertEqual(len(facts['TestStorageCrashContextEvidenceRefusal']['cases']),24)
        for variant in ('omit','duplicate','alias-pass','substitute-state','substitute-log'):
            changed=copy.deepcopy(rows)
            if variant=='omit':changed.pop()
            if variant=='duplicate':changed.append(copy.deepcopy(changed[-1]))
            if variant=='alias-pass':changed[2]['observed']['error']=None
            if variant=='substitute-state':changed[2]['inputs']['state_base64']=changed[1]['inputs']['state_base64']
            if variant=='substitute-log':changed[1]['inputs']['log_base64']='Zm9yZ2Vk'
            with self.subTest(variant=variant),self.assertRaises(Refusal):gate.evidence_observations(changed,[baseline])




def selected_rows(baseline):
    raw=bytes(baseline['observed']['image_bytes']);image=gate.recovery_image(raw);volume=image['volumes']['data']
    context=baseline['inputs']['context'];rows=[baseline]
    for state in baseline['observed']['states']:
        recovered=copy.deepcopy(image)
        for op in volume['history']:
            if op['id'] in state['SelectedOperations']:gate.apply_effect(recovered['volumes']['data'],op['effect'])
        recovered['volumes']['data']['owned']=sorted(recovered['volumes']['data']['objects'])
        raw=encoded_image(recovered)
        rows.append(dict(schema=gate.SCHEMA,case='storage-selected-state-recovery',inputs=dict(context=context,state=state),
            observed=dict(image_bytes=list(raw),recovery_image_sha256=gate.digest(raw),db_present=any(entry['Path']=='/db' for entry in state['Entries']),snapshots=gate.recovered_snapshots(recovered))))
    state_raw=bytes(baseline['observed']['state_bytes']);log=bytes(baseline['observed']['log_bytes']);_,parts=gate.state_raw_sections(state_raw)
    partial=gate.state_envelope(2,parts[:3]+[b'',parts[4],b'\x02'])
    for variant in ('hash','history','missing'):
        forged=copy.deepcopy(context)
        if variant=='hash':forged['data']['PendingSHA256']='sha256:'+'0'*64
        elif variant=='history':forged['data']['Operations'][0]['Kind']='forged'
        else:forged['data']['Operations']=[]
        rows.append(dict(schema=gate.SCHEMA,case='storage-context-divergence',inputs=dict(variant=variant,context=forged),
            observed=dict(failure=['invalid','storage crash context diverged from prior observation'],image=None,state_bytes=list(partial),log_bytes=list(log),
                original_state_bytes=list(state_raw),original_log_bytes=list(log),state_sha256=gate.digest(partial))))
    large=copy.deepcopy(image);large_volume=large['volumes']['data'];history=[]
    for index in range(50):
        allocation=copy.deepcopy(volume['history'][0]);namespace=copy.deepcopy(volume['history'][1]);inode=4+index
        allocation.update(id=2*index+1,deps=[],stores=[f'alloc:{inode}'])
        allocation['effect'].update(inode=inode)
        allocation['effect']['object'].update(inode=inode,kind=1,mode=0o600)
        namespace.update(id=2*index+2,deps=[allocation['id']],stores=[f'namespace:{index}'])
        namespace['effect']['namespace']=[dict(path='/'+'\x01'*3998+f'{index:02d}',inode=inode)]
        history.extend([allocation,namespace]);gate.apply_effect(large_volume,allocation['effect']);gate.apply_effect(large_volume,namespace['effect'])
    large_volume.update(history=history,next_operation=101,owned=sorted(large_volume['objects']))
    large['next_inode']=55
    raw=encoded_image(large);snapshots=gate.recovered_snapshots(large)
    large_context={'data':dict(PendingSHA256=gate.pending_identity('data',history),Operations=[gate.observed_operation(op) for op in history])}
    rows.append(dict(schema=gate.SCHEMA,case='storage-large-observed-selection',
        inputs=dict(files=50,path_bytes=4000,context=large_context,context_sha256=gate.digest(gate.canonical(large_context)),selection={'data':list(range(1,101))}),
        observed=dict(image_bytes=list(raw),snapshots=snapshots,image_sha256=gate.digest(raw),snapshots_sha256=gate.digest(gate.struct_json(snapshots)))))
    return rows


class CompleteSelectedRecovery(unittest.TestCase):
    def test_all_states_divergences_and_large_names_replay_from_raw_images(self):
        rows=selected_rows(selected_fixture());facts=gate.selected_observations(rows)
        self.assertEqual(facts['TestStorageCrashContextLargeObservedSelection']['files'],50)
        self.assertEqual(len(facts['TestStorageCrashContextSelectedOwnersAndDivergence']['states']),2)
        for mutation in ('omit-state','omit-divergence','retain-image-on-divergence','hide-enumeration','wrong-file-name'):
            changed=copy.deepcopy(rows)
            if mutation=='omit-state':changed.pop(1)
            if mutation=='omit-divergence':changed.pop(3)
            if mutation=='retain-image-on-divergence':changed[3]['observed']['image']=[]
            if mutation=='hide-enumeration':
                row=changed[3];_,parts=gate.state_raw_sections(bytes(row['observed']['state_bytes']));parts[2]=b'null'
                raw=gate.state_envelope(2,parts);row['observed']['state_bytes']=list(raw);row['observed']['state_sha256']=gate.digest(raw)
            if mutation=='wrong-file-name':
                row=changed[-1];image=gate.recovery_image(bytes(row['observed']['image_bytes']));volume=image['volumes']['data']
                path=next(path for path in volume['paths'] if path!='/');volume['paths']['/wrong']=volume['paths'].pop(path)
                raw=encoded_image(image);snapshots=gate.recovered_snapshots(image)
                row['observed'].update(image_bytes=list(raw),image_sha256=gate.digest(raw),snapshots=snapshots,snapshots_sha256=gate.digest(gate.struct_json(snapshots)))
            with self.subTest(mutation=mutation),self.assertRaises(Refusal):gate.selected_observations(changed)


if __name__=='__main__':unittest.main()
