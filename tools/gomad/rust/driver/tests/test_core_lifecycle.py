"""Pure adversarial lifecycle preimages; no guest, Cargo, Go or target preparation."""
import base64
import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_lifecycle as gate
from driver import Refusal


def store(node='a', terminal='exit', code=0, output=b'mark'):
    return dict(handle=gate.handle(node), terminal=terminal, exit_code=code, message='',
                authorization_to_terminal_nanos=10, interrupted=False, stdout=list(output))


def activity(count):
    return dict(launched_stores=count, remote_callback_attempts=0)


def row(root, variant, observed, inputs=None):
    value=dict(schema=gate.SCHEMA, original_root=root, variant=variant, observed=observed)
    if inputs is not None:value['inputs']=inputs
    return json.dumps(value)


def source_fixture(name):
    if name == gate.TEST:
        return 'fn store_result_fact() { "authorization_to_terminal_nanos":item.execution.authorization_to_terminal_nanos }'
    if name == gate.SOURCES[1]:
        return ('CLUSTER_STOP_GRACE: Duration = Duration::from_secs(5); watcher.join()\n'
            'fn drain_running() { match joined(pending) {} }\nfn caller() { '
            'let join_error = drain_running(running, &mut completed, broker, driven.is_ok()); }')
    if name.endswith('cluster/handoff/broker.rs'):
        return '\n    '.join('pub fn '+method+'() { self.native_control_attempts.fetch_add(1, std::sync::atomic::Ordering::Relaxed); }'
                             for method in ('activate', 'run', 'command'))
    return 'fixture source'


class StoreOwnership(unittest.TestCase):
    def test_missing_duplicate_or_forged_drop_is_refused(self):
        good=dict(declared_nodes=['a','b'],incarnations=['a/1'],activity=activity(1),store_results=[store()])
        gate.validate_case(gate.LAZY,'native-lazy-store',good)
        for edit in ('drop','boolean','duplicate','owner','count','callback'):
            bad=copy.deepcopy(good)
            if edit=='drop':bad['store_results'][0]['authorization_to_terminal_nanos']=None
            elif edit=='boolean':bad['store_results'][0]['authorization_to_terminal_nanos']=True
            elif edit=='duplicate':bad['store_results'].append(store())
            elif edit=='owner':bad['store_results'][0]['handle']['node']='b'
            elif edit=='count':bad['activity']['launched_stores']=0
            else:bad['activity']['remote_callback_attempts']=1
            with self.subTest(edit=edit),self.assertRaises(Refusal):gate.validate_case(gate.LAZY,'native-lazy-store',bad)

    def test_failed_wait_cannot_reorder_invent_or_continue_actions(self):
        actual=store(terminal='exit',code=17)
        good=dict(activity=activity(1),exit_code=17,incarnations=['a/1'],
                  steps=[gate.step('start','a'),gate.step('wait','a')],terminal='exit',store_results=[actual])
        gate.validate_case(gate.OPERATOR,'exit',good)
        for steps in ([gate.step('wait','a'),gate.step('start','a')],good['steps']+[gate.step('start','b')],[gate.step('start','a'),gate.step('stop','a')]):
            bad=copy.deepcopy(good);bad['steps']=steps
            with self.assertRaises(Refusal):gate.validate_case(gate.OPERATOR,'exit',bad)
        bad=copy.deepcopy(good);bad['steps'][0]['handle']['incarnation']=True
        with self.assertRaises(Refusal):gate.validate_case(gate.OPERATOR,'exit',bad)
        bad=copy.deepcopy(good);bad['exit_code']=0
        with self.assertRaises(Refusal):gate.validate_case(gate.OPERATOR,'exit',bad)

    def test_held_peer_must_block_until_release_and_cannot_attempt_controls(self):
        def event(tag, node):
            encoded=node.encode()
            return bytes([tag])+len(encoded).to_bytes(4,'little')+encoded+(1).to_bytes(8,'little')
        lifecycle=b'GMWIL\x01\0\0'+b''.join(event(tag,node) for tag,node in ((1,'b'),(3,'b'),(1,'a'),(3,'a')))
        peer=store('b');failed=store('a',terminal='trap',code=None);failed['message']='actual trap'
        good=dict(terminal='trap',exit_code=None,message='actual trap',
            failed_module_bytes=list(gate.HELD_FAILED_MODULE),failed_module_sha256=gate.hashlib.sha256(gate.HELD_FAILED_MODULE).hexdigest(),
            peer_module_bytes=list(gate.HELD_PEER_MODULE),peer_module_sha256=gate.hashlib.sha256(gate.HELD_PEER_MODULE).hexdigest(),
            held_peer_result=peer,store_results=[failed,peer],activity=activity(2),
            probe_millis=50,probe_outcome='timeout',probe_elapsed_nanos=50_100_000,
            release_to_return_nanos=1000,join_error=None,native_control_entry_points=['activate','run','command'],
            before_control_attempts=4,held_control_attempts=4,after_control_attempts=4,
            before_lifecycle_bytes=list(lifecycle),held_lifecycle_bytes=list(lifecycle),after_lifecycle_bytes=list(lifecycle))
        gate.validate_case(gate.DRAIN,'held-peer-release',good)
        for key,value in [('probe_outcome','returned'),('probe_elapsed_nanos',49_999_999),
            ('held_control_attempts',5),('after_control_attempts',5),('join_error','store failed'),
            ('held_lifecycle_bytes',list(lifecycle)+[0])]:
            bad=copy.deepcopy(good);bad[key]=value
            with self.subTest(key=key),self.assertRaises(Refusal):gate.validate_case(gate.DRAIN,'held-peer-release',bad)
        bad=copy.deepcopy(good);bad['store_results'][1]['authorization_to_terminal_nanos']=None
        with self.assertRaises(Refusal):gate.validate_case(gate.DRAIN,'held-peer-release',bad)
        bad=copy.deepcopy(good);bad['peer_module_bytes']=list(gate.HELD_FAILED_MODULE)
        bad['peer_module_sha256']=gate.hashlib.sha256(gate.HELD_FAILED_MODULE).hexdigest()
        with self.assertRaises(Refusal):gate.validate_case(gate.DRAIN,'held-peer-release',bad)

    def test_cleanup_requires_real_peer_output_drop_and_no_dormant_start(self):
        actual=[store('a',terminal='stop',code=None),store('b',terminal='stop',code=None)]
        for item in actual:item['interrupted']=True
        good=dict(terminal='capacity',message='scenario action capacity limit=2 used=2 delta=1',
            steps=[gate.step('start','b'),gate.step('start','a')],joined_incarnations=2,
            acknowledged_handle=gate.handle('b'),acknowledged_stdout=list(b'mark'),
            all_stores_dropped_before_return=True,dormant_started=False,activity=activity(2),store_results=actual)
        gate.validate_case(gate.CLEANUP,'action-limit',good)
        for key,value in [('steps',good['steps']+[gate.step('stop','a')]),('dormant_started',True),('joined_incarnations',1)]:
            bad=copy.deepcopy(good);bad[key]=value
            with self.assertRaises(Refusal):gate.validate_case(gate.CLEANUP,'action-limit',bad)
        bad=copy.deepcopy(good);bad['store_results'][1]['stdout']=[]
        with self.assertRaises(Refusal):gate.validate_case(gate.CLEANUP,'action-limit',bad)

    def test_actual_five_second_bound_rejects_shortened_and_extended_windows(self):
        good=dict(execution_deadline_millis=900000,cleanup_grace_millis=5000,elapsed_nanos=5_010_000_000,
            cleanup_error='actual scenario failure; cluster cleanup deadline exceeded',acknowledged_stdout=list(b'mark'),
            authored_steps=[gate.step('start','a')],joined_incarnations=1,all_stores_dropped_before_return=True,
            activity=activity(1),store_results=[store(terminal='watchdog_timeout',code=None)])
        gate.validate_case(gate.GRACE,'live-noncooperative-actual-five-second-grace',good)
        for value in (50_000_000,10_000_000_000,True):
            bad=copy.deepcopy(good);bad['elapsed_nanos']=value
            with self.assertRaises(Refusal):gate.validate_case(gate.GRACE,'live-noncooperative-actual-five-second-grace',bad)


class RestartOwnership(unittest.TestCase):
    def test_recovery_image_and_refused_storage_are_actual_preimages(self):
        import struct
        import core_codec
        def number(value): return struct.pack('<Q', value)
        def blob(value): return number(len(value)) + value
        volumes=[dict(ID='data', Path='/workspace', CapacityBytes=4096)]
        model=dict(volumes=volumes, limits=dict(PendingOperations=100, Transitions=1000),faults=None,sqlite_lock_databases=None)
        identity='sha256:'+core_codec.digest(core_codec.struct_json(dict(volumes=volumes,limits=model['limits'])))
        obj=number(3)+number(0o700)+number(2)+number(0)+blob(b'')
        volume=blob(b'data')+blob(b'/workspace')+number(4096)+number(1)+obj+number(1)+blob(b'/')+number(3)
        volume+=number(1)+number(3)+number(0)+number(0)+number(1)+number(0)
        sections=[blob(identity.encode()),number(0),number(4)+number(1)+number(1)+volume]
        image=b'GMVOLIMG'+struct.pack('<II',1,3)+b''.join(struct.pack('<I',index)+blob(section) for index,section in enumerate(sections,1))
        descriptor=dict(model=model,model_identity=identity,image=None)
        before=dict(descriptor=descriptor,image=None,context=None)
        retry=copy.deepcopy(before);retry['image']=list(image)
        retry['descriptor']['image']=dict(bytes=len(image),chunks=1,sha256=core_codec.digest(image))
        good=dict(refusal='cluster restart lacks recovery image',before_incarnation=1,before_slot=1,
            after_refusal_incarnation=1,after_refusal_slot=1,peer_incarnation=1,peer_slot=2,
            refused_storage_unchanged=True,retry_incarnation=2,retry_slot=3,recovery_input_unchanged=True,
            storage_before=before,storage_after_refusal=copy.deepcopy(before),retry_storage=retry,
            recovery_image=list(image),activity=activity(0))
        gate.validate_case(gate.RESTART,'missing-storage-image',good)
        for edit in ('consumed','storage','image','slot'):
            bad=copy.deepcopy(good)
            if edit=='consumed':bad['after_refusal_incarnation']=2
            elif edit=='storage':bad['storage_after_refusal']['image']=[]
            elif edit=='image':bad['recovery_image'][0]^=1
            else:bad['retry_slot']=4
            with self.subTest(edit=edit),self.assertRaises(Refusal):gate.validate_case(gate.RESTART,'missing-storage-image',bad)

    def test_changed_source_fails_before_case_admission(self):
        import tempfile
        import hashlib
        with tempfile.TemporaryDirectory() as temporary:
            repo=Path(temporary)
            config=dict(repo=str(repo),files={})
            for name in gate.SOURCES:
                path=repo/name;path.parent.mkdir(parents=True,exist_ok=True)
                source=source_fixture(name)
                path.write_text(source);config['files'][str(path.resolve())]=hashlib.sha256(path.read_bytes()).hexdigest()
            with self.assertRaisesRegex(Refusal,'omitted'):gate.observations(config,'')
            (repo/gate.TEST).write_text('substituted collector')
            with self.assertRaisesRegex(Refusal,'not frozen'):gate.observations(config,'')


class ClosedCatalog(unittest.TestCase):
    def test_omitted_duplicate_crosslabel_and_unknown_rows_refuse(self):
        a=dict(error_kind='unsupported',message='linked cluster forbids transport callbacks',activity=dict(launched_stores=0,remote_callback_attempts=1))
        good=row(gate.CALLBACK,'call',a)
        with self.assertRaisesRegex(Refusal,'omitted'):gate.parse_rows(good)
        with self.assertRaisesRegex(Refusal,'duplicate'):gate.parse_rows(good+'\n'+good)
        with self.assertRaises(Refusal):gate.parse_rows(row(gate.MODEL,'call',a))
        with self.assertRaises(Refusal):gate.parse_rows(row('TestUnrelated','call',a))
        with self.assertRaises(Refusal):gate.parse_rows(good.replace('"variant": "call"','"variant": "call", "variant": "sync"'))

    def test_parent_deadline_is_cancelled_internal_deadline_is_watchdog(self):
        good=dict(terminal='cancelled',exit_code=None,watchdog_timeout=False,cancelled=True,bound_millis=1000,
            parent_cancel_after_millis=20,joined_incarnations=1,all_stores_dropped_before_return=True,
            activity=activity(1),store_results=[store(terminal='cancelled',code=None)])
        gate.validate_case(gate.HOST,'parent deadline',good)
        with self.assertRaises(Refusal):gate.validate_case(gate.INTERNAL,'internal deadline',good)
        good['watchdog_timeout']=True
        with self.assertRaises(Refusal):gate.validate_case(gate.HOST,'parent deadline',good)

    def test_sqlite_errors_are_rederived_from_exact_original_bytes(self):
        inputs=dict(ack_bytes=list(b'ACK 2\n'),stdout_bytes=list(b'INTEGRITY ok\nHISTORY x:committed\n'))
        gate.validate_case(gate.SQLITE,'nonnumeric-event',dict(error='invalid SQLite history'),inputs)
        with self.assertRaises(Refusal):gate.validate_case(gate.SQLITE,'nonnumeric-event',dict(error='PASS'),inputs)
        inputs['stdout_bytes']=list(b'INTEGRITY ok\nHISTORY 1:committed,2:committed\n')
        with self.assertRaises(Refusal):gate.validate_case(gate.SQLITE,'nonnumeric-event',dict(error='invalid SQLite history'),inputs)

    def test_three_global_network_cases_cannot_close_the_original_twelve_variants(self):
        import tempfile
        import hashlib
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as temporary:
            repo=Path(temporary);config=dict(repo=str(repo),files={})
            for name in gate.SOURCES:
                path=repo/name;path.parent.mkdir(parents=True,exist_ok=True)
                text=source_fixture(name)
                path.write_text(text);config['files'][str(path.resolve())]=hashlib.sha256(path.read_bytes()).hexdigest()
            facts=[dict(original_roots=[gate.BOOT_NETWORK],variant=variant) for variant in ('matching','byte-mismatch','stall-mismatch')]
            with patch.object(gate,'parse_rows',return_value=facts):result=gate.observations(config,'')
            self.assertNotIn(gate.BOOT_NETWORK,result['roots'])
            partial=result['partial_roots'][gate.BOOT_NETWORK]
            self.assertEqual(partial['covered_original_variants'],['matching'])
            missing={'network free','first byte mismatch','later byte mismatch','first stall mismatch',
                'later stall mismatch','restart byte mismatch','restart stall mismatch','unused mismatch',
                'matching explicit broker','broker byte mismatch','broker stall mismatch'}
            self.assertEqual(set(partial['original_variant_catalog'])-set(partial['covered_original_variants']),missing)
            for variant in missing:
                self.assertTrue(any('/'+variant+' ' in reason for reason in partial['remaining_original_predicates']))
            self.assertTrue(any('unchanged' in reason for reason in partial['remaining_original_predicates']))

    def test_network_limits_are_checked_for_all_actual_owners(self):
        network=dict(profile='gomad.simulation-network-stream/v3',faults=[],limits=dict(listeners=8,connections=8,bytes=1024,connection_bytes=123,stall_nanos=456,transitions=128))
        scenario=dict(spec=dict(nodes=[dict(id='a',boot='main',address='10.0.0.1'),dict(id='b',boot='main',address='10.0.0.2')],
            limits=dict(network_connection_bytes=123,network_stall_nanos=456)),
            actions=[gate.step('start','a'),gate.step('start','b'),gate.step('wait','a'),gate.step('wait','b'),
                dict(action='restart',handle=gate.handle('a',2)),dict(action='wait',handle=gate.handle('a',2))])
        inputs=dict(scenario_base64=base64.b64encode(json.dumps(scenario).encode()).decode(),network=network)
        items=[dict(handle=gate.handle('a'),network=network),dict(handle=gate.handle('b'),network=network),dict(handle=gate.handle('a',2),network=network)]
        good=dict(accepted=True,error=None,configuration=dict(incarnations=items),request_incarnations=items)
        gate.validate_case(gate.BOOT_NETWORK,'matching',good,inputs)
        for group in ('configuration','request_incarnations'):
            bad=copy.deepcopy(good)
            rows=bad[group]['incarnations'] if group=='configuration' else bad[group]
            rows[2]['network']['limits']['connection_bytes']=124
            with self.assertRaises(Refusal):gate.validate_case(gate.BOOT_NETWORK,'matching',bad,inputs)
        bad=copy.deepcopy(good);bad['request_incarnations'].pop()
        with self.assertRaises(Refusal):gate.validate_case(gate.BOOT_NETWORK,'matching',bad,inputs)


if __name__=='__main__':unittest.main()
