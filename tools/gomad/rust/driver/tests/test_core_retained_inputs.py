"""Independent retained metadata mutations and typed input refusals; no native build."""
import copy
from pathlib import Path
import sys
import unittest
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import core_retained_inputs as gate
from driver import Refusal


class StrictInputMutation(unittest.TestCase):
    def test_raw_mutations_preserve_actual_syntax_and_closed_native_fields(self):
        raw=gate.canonical(dict(schema=gate.SCHEMA,configuration=dict(model={},incarnations=[dict(module={})],environment={}),retained=[]))
        for case in gate.DECODE_CASES:
            with self.subTest(case=case):
                changed=gate.decoder_mutation(raw,case)
                self.assertNotEqual(changed,raw)
        self.assertIn(b'"engine":"duplicate"',gate.decoder_mutation(raw,'duplicate-nested'))
        self.assertEqual(gate.decoder_mutation(raw,'truncated'),raw[:-1])
        self.assertEqual(len(gate.decoder_mutation(raw,'oversized')),(16<<20)+1)

    def test_history_case_catalog_has_exact_raw_ambiguous_variants(self):
        cases=gate.history_cases()
        self.assertEqual(len(cases),8)
        self.assertEqual(gate.history(cases['valid'])['history'],'YQ==')
        for case,raw in cases.items():
            if case=='valid':continue
            with self.subTest(case=case),self.assertRaises(Refusal):gate.history(raw)

    def test_record_bindings_reject_resealed_configuration_foreign_primary(self):
        baseline=dict(schema=gate.SCHEMA,helper_sha256='a'*64,host_sha256='b'*64,runtime_implementation_sha256='c'*64,
            toolchain_sha256='d'*64,configuration=dict(engine='compiled',profile='gomad.wasi-cooperative/v1',limits={},broker=dict(seed=0),
            incarnations=[dict(handle=dict(node='sqlite',incarnation=2),module=dict(file='target/module.wasm',sha256='e'*64))]),retained=[dict(handle=dict(node='sqlite',incarnation=2))])
        changed=gate.binding_mutation(baseline,'primary-path')
        self.assertEqual(changed['configuration']['incarnations'][0]['module']['file'],'target/foreign.wasm')
        self.assertEqual(baseline['configuration']['incarnations'][0]['module']['file'],'target/module.wasm')
        self.assertNotEqual(gate.binding_mutation(baseline,'toolchain')['toolchain_sha256'],baseline['toolchain_sha256'])




class RetainedAuthentication(unittest.TestCase):
    def test_module_variants_require_actual_changed_bytes_filetype_and_link_target(self):
        import base64
        raw=b'\0asm\1\0\0\0'
        for case in ('tampered-bytes','truncated','missing-source','symbolic-link-source'):
            source=(b'foreign!' if case=='tampered-bytes' else raw[:-1]) if case in ('tampered-bytes','truncated') else None
            kind='file' if source is not None else 'symlink' if case=='symbolic-link-source' else None
            row=dict(inputs=dict(original_base64=base64.b64encode(raw).decode(),file='target/module.wasm',
                descriptor=dict(bytes=8,sha256=gate.digest(raw),module=True),source_kind=kind,
                source_bytes_base64=None if source is None else base64.b64encode(source).decode(),link_target='/actual/genuine-module' if kind=='symlink' else None),
                observed=dict(error='artifact payload identity mismatch' if case=='tampered-bytes' else 'artifact payload size mismatch' if case=='truncated' else 'missing or symbolic path refused',compiled_cache_exists=False))
            gate.module_row(case,row)
            bad=copy.deepcopy(row);bad['inputs']['source_kind']='file' if kind!='file' else 'symlink'
            with self.subTest(case=case),self.assertRaises(Refusal):gate.module_row(case,bad)

    def test_raw_descriptor_cases_recompute_the_same_authentication_preimage(self):
        import base64
        raw=b'actual retained bytes'
        for case in ('digest','bytes','path','limit'):
            descriptor=dict(file=gate.FILE,bytes=len(raw),sha256=gate.digest(raw),limit=16<<20)
            if case=='digest':descriptor['sha256']=gate.digest(b'foreign bytes')
            if case=='bytes':descriptor['bytes']+=1
            if case=='path':descriptor['file']='evidence/foreign.json'
            if case=='limit':descriptor['limit']=32<<20
            refusal='missing retained cluster payload' if case=='path' else 'cluster payload exceeds caller bound or has invalid identity' if case=='limit' else 'retained cluster payload identity mismatch'
            row=dict(inputs=dict(payload_base64=base64.b64encode(raw).decode(),descriptor=descriptor),observed=dict(error=refusal))
            gate.authentication_row(case,row,raw)
            row['inputs']['payload_base64']=base64.b64encode(raw+b'changed').decode()
            with self.subTest(case=case),self.assertRaises(Refusal):gate.authentication_row(case,row,raw)

    def test_recursive_directory_inventory_proves_no_write_without_counter_alias(self):
        snapshot={'target':dict(kind='directory'),'target/module.wasm':dict(kind='file',bytes=8,sha256=gate.digest(b'\0asm\1\0\0\0')),
                  'record.json':dict(kind='file',bytes=2,sha256=gate.digest(b'{}'))}
        observed=dict(directory_before=snapshot,directory_after=copy.deepcopy(snapshot),entries_before=2,entries_after=2,compiled_cache_exists=False)
        gate.directory_unchanged(observed)
        observed['directory_after']['target/module.wasm']['sha256']='0'*64
        with self.assertRaises(Refusal):gate.directory_unchanged(observed)
        observed['directory_after']=copy.deepcopy(snapshot);observed['entries_after']=True
        with self.assertRaises(Refusal):gate.directory_unchanged(observed)

    def test_all_eighty_seven_cases_are_required_even_when_unit_test_passes(self):
        self.assertEqual(len(gate.expected_cases()),87)
        with self.assertRaises(Refusal):gate.observations({},'test all ... ok\n')


class LiteralCapacity(unittest.TestCase):
    def test_all_terminal_policy_budget_combinations_are_recomputed_from_raw_output(self):
        import base64
        handle=dict(node='sqlite',incarnation=2)
        for termination,message in (('infrastructure','output capacity'),('capacity','transcript capacity'),('exit',''),('capacity','output capacity')):
            for flag in (False,True):
                for maximum in (1,4096):
                    case=f'{termination}/{message}/report-only={str(flag).lower()}/maximum={maximum}'
                    complete=termination!='capacity'
                    report=gate.oracle('soft-assertion.instance-'+gate.digest(b'sqlite/2'),complete and flag,
                        [('instance',b'sqlite/2'),('messages',gate.canonical(['invalid {state}'])),('status',b'observed' if complete else b'inconclusive')],
                        soft=True,report_only=flag,inconclusive=not complete)
                    observed=dict(oracles=[] if maximum==1 else [report],oracle_error='capacity: resource=oracle_evidence_bytes' if maximum==1 else '')
                    if termination=='capacity' and message=='output capacity':observed['capacity_failure']=dict(termination='capacity',message='output capacity',
                        soft_assertions=dict(messages=['invalid {state}'],report_only=flag,inconclusive=True),detail=observed['oracle_error'] or 'choice trace is not complete')
                    row=dict(inputs=dict(handle=handle,stdout_base64=base64.b64encode(b'PASS').decode(),stderr_base64=base64.b64encode(b'failed assertion: invalid {state}').decode(),
                        termination=termination,message=message,offered_bytes=len(b'PASSfailed assertion: invalid {state}'),oracle_plan=dict(schema='gomad.wasm.soft-assertion-observation/v1',prefix='',maximum_bytes=maximum,histories=[],convergence=[],soft_assertions=[dict(handle=handle,report_only=flag)])),observed=observed)
                    gate.capacity_row(case,row)
                    bad=copy.deepcopy(row);bad['inputs']['stderr_base64']=base64.b64encode(b'failed assertion: forged').decode()
                    with self.subTest(case=case),self.assertRaises(Refusal):gate.capacity_row(case,bad)


def metadata_fixture():
    module=b'\0asm\1\0\0\0';handle=dict(node='sqlite',incarnation=2)
    toolchain=gate.digest(b'retained-reader-test-toolchain')
    item=dict(handle=handle,module=dict(file='target/module.wasm',bytes=8,sha256=gate.digest(module),limit=512<<20),
        limits={},arguments=[],environment={},runtime_sha256='b'*64,model={},namespace={},storage=None,network=None,
        choice_identity=dict(TargetSHA256=list(bytes.fromhex(gate.digest(module))),ToolchainBuildKey=toolchain))
    return dict(schema=gate.SCHEMA,helper_sha256='a'*64,host_sha256='b'*64,runtime_implementation_sha256='c'*64,toolchain_sha256=toolchain,
        configuration=dict(engine='compiled',profile='gomad.wasi-cooperative/v1',limits={},broker=dict(seed=0),network=None,incarnations=[item],steps=[],boundaries=[],crash_groups=[]),
        retained=[dict(handle=handle,interrupted=False,evidence={})])


def snapshot():
    return dict(directory_before={'target':dict(kind='directory'),'target/module.wasm':dict(kind='file',bytes=8,sha256=gate.digest(b'\0asm\1\0\0\0'))},
        directory_after={'target':dict(kind='directory'),'target/module.wasm':dict(kind='file',bytes=8,sha256=gate.digest(b'\0asm\1\0\0\0'))},entries_before=1,entries_after=1)


class ClosedDecodeMatrix(unittest.TestCase):
    def test_each_strict_mutation_has_matching_public_refusal_and_preserves_raw_directory(self):
        import base64
        baseline=metadata_fixture();raw=gate.canonical(baseline);self.assertEqual(gate.metadata(raw),baseline)
        fragments={'foreign-schema':'unsupported retained cluster replay input schema','legacy-schema':'unsupported retained cluster replay input schema',
            'duplicate-top':'duplicate object key','duplicate-nested':'duplicate object key','duplicate-model-value':'duplicate object key','duplicate-environment-map':'duplicate object key',
            'excessive-nesting':'nesting','oversized':'artifact payload exceeds caller bound','unknown-top':'unknown field `unknown`','unknown-nested':'unknown field `unknown`',
            'missing-module':'missing field `module`','wrong-module-type':'invalid type: integer','trailing-value':'trailing data','truncated':'invalid syntax','invalid-utf8':'unicode'}
        unknown={'host-faults':'faults','host-scenarios':'scenarios','host-history':'history','host-observations':'observations','host-oracles':'oracles','go-served-calls':'go_filesystem_calls','guest-callbacks':'guest_callbacks'}
        for case in gate.DECODE_CASES:
            refusal=fragments[case] if case in fragments else 'unknown field `'+unknown[case]+'`'
            row=dict(inputs=dict(payload_base64=base64.b64encode(gate.decoder_mutation(raw,case)).decode()),
                observed=dict(snapshot(),error=refusal,prepare_error=refusal,public_status=2,public_error='replay: '+refusal))
            gate.decode_row(case,row,raw)
            bad=copy.deepcopy(row);bad['inputs']['payload_base64']=base64.b64encode(raw).decode()
            with self.subTest(case=case),self.assertRaises(Refusal):gate.decode_row(case,bad,raw)

    def test_every_resealed_binding_variant_keeps_original_identity_and_first_guard(self):
        baseline=metadata_fixture()
        for case in gate.BIND_CASES:
            changed=gate.binding_mutation(baseline,case)
            identity=dict(target=gate.digest(b'\0asm\1\0\0\0'),toolchain=baseline['toolchain_sha256'],platform='f'*64,implementation='a'*64,configuration=gate.digest(gate.canonical(changed['configuration'])))
            record=dict(schema='gomad.execution-record/v2',identity=identity,execution_identity=gate.digest(gate.canonical(identity)),profile=baseline['configuration']['profile'],limits={},seed=0)
            refusal='retained cluster primary module identity mismatch' if case in ('primary-path','primary-digest') else 'retained cluster incarnation count mismatch' if case in ('missing-incarnations','missing-retained','duplicate-incarnation') else 'retained cluster handle mismatch' if case=='foreign-retained' else 'WASM retained cluster implementation/configuration identity mismatch'
            row=dict(inputs=dict(retained_inputs=changed,record=record),observed=dict(snapshot(),compiled_cache_exists=False,error=refusal,prepare_error=refusal))
            gate.binding_row(case,row,baseline)
            row['inputs']['record']['identity']['configuration']='0'*64
            with self.subTest(case=case),self.assertRaises(Refusal):gate.binding_row(case,row,baseline)


class StorageDetachment(unittest.TestCase):
    def test_native_projection_preserves_raw_source_when_each_detached_layer_changes(self):
        import base64
        context=gate.canonical({'volume':{'PendingSHA256':'a'*64,'Operations':[{'ID':2,'Dependencies':[1]}]}})
        storage=dict(descriptor=dict(crash_context=dict(chunks=1,bytes=len(context),sha256=gate.digest(context)),
            crash_limits=dict(States=32,Operations=8,Depth=8,Bytes=4096,Work=65536)),context=list(context))
        item=dict(storage=storage,storage_crash_selection=dict(volume=[1,2],empty=[]));raw=gate.canonical(item)
        projected=copy.deepcopy(storage);projected['descriptor']['crash_selection']=item['storage_crash_selection']
        detached=copy.deepcopy(projected);detached['descriptor']['crash_selection']['volume'][0]=99;detached['descriptor']['crash_selection'].pop('empty');detached['descriptor']['crash_limits']['States']=1;detached['context'][0]=99
        dependency=copy.deepcopy(projected);value=gate.decoded(context);value['volume']['Operations'][0]['Dependencies'][0]=99;dependency['context']=list(gate.canonical(value))
        row=dict(inputs=dict(original_incarnation_base64=base64.b64encode(raw).decode()),observed=dict(projected=projected,detached=detached,dependency_detached=dependency,reprojected=projected,original_after_base64=base64.b64encode(raw).decode()))
        gate.storage_row('native-deep-detachment',row)
        row['observed']['dependency_detached']=projected
        with self.assertRaises(Refusal):gate.storage_row('native-deep-detachment',row)


class WorkloadOwnerIntent(unittest.TestCase):
    def test_all_policy_and_sqlite_caller_intent_variants_are_exact(self):
        for case in ('','go-run','go-test','unknown','unknown-handle','duplicate-policy'):
            if case in ('unknown-handle','duplicate-policy'):
                policy=dict(handle=dict(node='foreign' if case=='unknown-handle' else 'sqlite',incarnation=2),report_only=True)
                inputs=dict(oracle_plan=dict(schema='gomad.wasm.soft-assertion-observation/v1',prefix='',maximum_bytes=4096,histories=[],convergence=[],soft_assertions=[policy]*(2 if case=='duplicate-policy' else 1)))
            else:inputs=dict(workload_kind=case)
            refusal=None if case=='go-test' else 'invalid recorded cluster workload kind' if case=='unknown' else 'soft assertion policy requires a recorded Go-test workload'
            row=dict(inputs=inputs,observed=dict(error=refusal));gate.policy_row(case,row)
            row['observed']['error']=None if refusal else 'refused'
            with self.subTest(case=case),self.assertRaises(Refusal):gate.policy_row(case,row)
        baseline=dict(kind='sqlite-durable-recovery',selection=None)
        gate.sqlite_row('native-valid',dict(inputs=baseline,observed=dict(workload=baseline)))
        for case,(name,value) in gate.SQLITE_MUTATIONS.items():
            row=dict(inputs=dict(baseline,**{name:value}),observed=dict(error='unknown field `'+name+'`'))
            gate.sqlite_row(case,row)
            row['inputs'][name]='substituted'
            with self.subTest(case=case),self.assertRaises(Refusal):gate.sqlite_row(case,row)

    def test_actual_replay_divergence_is_recomputed_from_record_stream_bytes(self):
        import base64
        inputs=metadata_fixture();identity=dict(configuration=gate.digest(gate.canonical(inputs['configuration'])))
        raw=b'actual fresh native execution evidence'
        record=dict(schema='gomad.execution-record/v2',identity=identity,execution_identity=gate.digest(gate.canonical(identity)),
            outcome=dict(termination='exit',exit_code=0,reason=''),virtual_time_nanos=0,
            streams={'evidence/cluster_summary':dict(sha256=gate.digest(raw))})
        encoded=base64.b64encode(gate.canonical(inputs)).decode()
        actual=dict(outcome=dict(termination='exit',exit_code=0,message=''),evidence=dict(cluster_summary=list(raw)))
        row=dict(inputs=dict(retained_inputs_base64=encoded,record=record,actual_execution=actual),
            observed=dict(snapshot(),divergence=None,termination='exit',exit_code=0,reopened_inputs_base64=encoded,compiled_cache_exists=False))
        gate.admission_row(row)
        row['inputs']['actual_execution']['evidence']['cluster_summary'][0]^=1
        with self.assertRaises(Refusal):gate.admission_row(row)


if __name__=='__main__':unittest.main()
