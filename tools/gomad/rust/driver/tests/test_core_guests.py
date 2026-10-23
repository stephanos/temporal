"""Pure adversarial raw guest fixtures; no compiler, Cargo or guest execution."""
import base64
import copy
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import core_guests as gate
from driver import Refusal
from native_observations import canonical,digest
import core_world


def config(case):
    root,_,seed,policy,maximum=gate.CASES[case]
    owners=[gate.handle('a')] if root==gate.CAPACITY else [gate.handle('a'),gate.handle('b')]
    if root in (gate.PLAIN,gate.COMPRESSED,gate.GROUPED):owners += [gate.handle('a',2),gate.handle('b',2)]
    limits=dict(memory_bytes=256<<20,fuel=4000000000,wall_millis=300000 if root==gate.SOFT else 60000,
                output_bytes=4 if root==gate.CAPACITY else 1<<20,evidence_bytes=64<<20)
    model=dict(mode=1,limit=8<<20,diagnostics=True,epoch_nanos=946684800000000000,read_step_nanos=0,
               entropy_key=[1,2,3]+[0]*29,limits=dict(calls=100000,transcript_bytes=32<<20,pending_events=4096))
    namespace=dict(working_directory=list(b'/workspace'),writable_directories=[list(b'/workspace'),list(b'/tmp')],
                   epoch_nanos=946684800000000000,limits=dict(files=1024,descriptors=128,filesystem_bytes=8<<20,output_bytes=limits['output_bytes']))
    network=dict(profile='gomad.simulation-network-stream/v3',faults=[],limits=dict(listeners=128,connections=128,bytes=8<<20,connection_bytes=4<<20,stall_nanos=10_000_000_000,transitions=10000))
    c=dict(network=dict(model=network,addresses={'a':'10.0.0.1'} if root==gate.CAPACITY else {'a':'10.0.0.1','b':'10.0.0.2'},links=[],strict_topology=True,faults=[]),nodes=['a'] if root==gate.CAPACITY else ['a','b'],steps=gate.expected_steps(root),boundaries=[],crash_groups=[],
           oracle_plan=None,world_encoding='',broker=dict(seed=seed,pause_on_settle=False,
           limits=dict(max_requests=2<<20,max_events=2<<20,max_queued_events=65536,max_transitions=8<<20,max_payload_bytes=64<<20,max_string_bytes=1024)),incarnations=[])
    if root==gate.COMPRESSED:
        c['world_encoding']=c['world_profile']='gomad/world/compressed-snapshot/v2'
        c['broker']['limits'].update(max_queued_events=250000,max_string_bytes=(1<<32)-1)
    if root in (gate.PLAIN,gate.COMPRESSED):c['network']['links']=[dict(**{'from':a,'to':b},enabled=True,delay_nanos=0) for a,b in (('a','b'),('b','a'))]
    if root==gate.GROUPED:c['crash_groups']=[dict(nodes=['a','b'],marker=list(b'committed durable-a'))]
    if root==gate.CANARY:c['oracle_plan']=dict(schema='gomad.wasm.history-observation/v1',prefix='GOMAD_HISTORY_OBSERVATION ',maximum_bytes=4096,soft_assertions=[],histories=[dict(name='fixture.recovered',expected=gate.handle('a'),actual=gate.handle('b'),expected_events=['1','2'])],convergence=[dict(name='fixture.converged',handles=[gate.handle('a'),gate.handle('b')])])
    for owner in owners:
        args=['soft-assertion','-test.v','-assertion-mode=late'] if root==gate.SOFT else ['capacity','-test.run=TestOutputCapacity']
        if root in (gate.PLAIN,gate.COMPRESSED):args=['service']+(['server'] if owner['node']=='a' else ['client','10.0.0.1'])
        elif root==gate.CANARY:args=['service','observation']+(['canary'] if owner['node']=='b' and case.endswith('true') else [])
        elif root==gate.GROUPED:args=['service','durable' if owner['incarnation']==1 else 'recover']
        storage=None
        if root==gate.GROUPED:
            from core_codec import struct_json
            go_model=dict(volumes=[dict(ID=owner['node'],Path='/workspace',CapacityBytes=8<<20)],limits=dict(PendingOperations=1024,Transitions=10000))
            model_spec=dict(go_model,faults=None,sqlite_lock_databases=None)
            storage=dict(descriptor=dict(model=model_spec,model_identity='sha256:'+digest(struct_json(go_model))))
        c['incarnations'].append(dict(module=dict(file='target/module.wasm',bytes=8,sha256=digest(b'\0asm\x01\0\0\0'),limit=512<<20),network=network,handle=owner,limits=copy.deepcopy(limits),model=copy.deepcopy(model),namespace=copy.deepcopy(namespace),
            environment=dict(PWD='/workspace',TZ='UTC'),arguments=args,workload_kind='go-test' if root in (gate.SOFT,gate.CAPACITY) else 'go-run',storage=storage))
    if policy:c['oracle_plan']=dict(schema='gomad.wasm.soft-assertion-observation/v1',maximum_bytes=maximum,prefix='',histories=[],convergence=[],
        soft_assertions=[dict(handle=owner,report_only=True) for owner in owners])
    return c


def inputs(case):
    root=gate.CASES[case][0]
    selected=[key for key in gate.PINS if ('softassert' in key if root==gate.SOFT else 'service/main.go' in key or root==gate.CAPACITY and 'output_capacity' in key)]
    return dict(original_commit='1d695061b6382009d0659238c652a5a765b1d5f5',sources=[dict(path=key,sha256=gate.PINS[key]) for key in selected],
                module_sha256=digest(b'\0asm\x01\0\0\0'),toolchain_sha256=digest(b'compiler'),prepared_provenance_sha256=digest(b'prepared'),configuration=config(case))


def capacity(case):
    root,_,_,policy,maximum=gate.CASES[case]
    result=gate.report([gate.soft_oracle('a/1',b'PASS',b'',policy,False)])
    detail='capacity: resource=oracle_evidence_bytes' if maximum==1 else 'choice trace is not complete'
    assessment=dict(oracles=[] if maximum==1 else result['oracles'],oracle_error=detail if maximum==1 else '',
        capacity_failure=dict(termination='capacity',message='output capacity',soft_assertions=dict(messages=[],report_only=policy,inconclusive=True),detail=detail))
    return dict(schema='gomad.retained-reader-observation/v1',original_root=root,case=case,inputs=inputs(case),observed=dict(
        outcome=dict(termination='capacity',message='output capacity',output_bytes=4),stdout_base64='UEFTUw==',choice_bytes=0,
        assessment=assessment,publication_error='capacity: output capacity; '+detail))


def world(c):
    u=core_world.u;configuration=dict(seed=c['broker']['seed'],**c['broker']['limits'])
    declared=b''.join(u(configuration[key],4 if key=='max_string_bytes' else 8) for key in core_world.CONFIG)
    prefix=b'gomad/world/state/v1\0'+u(1,4)+declared
    counters=b''.join(u(value) for value in (0,1,1,1,0))
    empty=u(0)*3
    state=digest(prefix+counters+empty+core_world.frame(digest(b'')))
    return b'GMWSNAP1'+u(1,4)+declared+counters+u(0)*2+bytes.fromhex(digest(b''))+bytes.fromhex(state)+empty


def soft(case='soft/seed=7/report-only=false',message='invalid fixture state'):
    i=inputs(case);c=i['configuration'];policy=gate.CASES[case][3]
    payloads={'target/module.wasm':b'\0asm\x01\0\0\0','evidence/world':world(c),'evidence/network':b''}
    retained=[];summary=[];oracles=[]
    for item in c['incarnations']:
        owner=item['handle']['node']+'/1';stdout=b'PASS\n';stderr=canonical(dict(msg='failed assertion: '+message,**{'failed-assertion':True}))+b'\n'
        evidence={}
        for key,raw in [('stdout',stdout),('stderr',stderr)]:
            path='evidence/cluster/'+owner+'/'+key;payloads[path]=raw
            evidence[key]=dict(file=path,bytes=len(raw),sha256=digest(raw),limit=64<<20)
        retained.append(dict(handle=item['handle'],interrupted=False,evidence=evidence))
        summary.append(dict(handle=item['handle'],termination='exit',exit_code=0,interrupted=False,
            offered_output_bytes=len(stdout)+len(stderr),streams={key:dict(bytes=entry['bytes'],sha256=entry['sha256']) for key,entry in evidence.items()}))
        oracles.append(gate.soft_oracle(owner,stdout,stderr,policy))
    report=gate.report(oracles)
    retained_inputs=dict(schema='gomad.cluster-replay-inputs/v1',configuration=c,retained=retained)
    payloads[gate.FILE]=canonical(retained_inputs)
    payloads['evidence/cluster_summary']=canonical(dict(schema='gomad.cluster-observation/v1',incarnations=summary,
        world_sha256=digest(payloads['evidence/world']),network_sha256=digest(b'')))
    payloads['evidence/cluster_oracles']=canonical(report)
    identity=dict(target=i['module_sha256'],toolchain=i['toolchain_sha256'],configuration=digest(canonical(c)))
    status=0 if policy else 1
    record=dict(identity=identity,execution_identity=digest(canonical(identity)),seed=gate.CASES[case][2],outcome=dict(class_='success' if policy else 'target_failure'),
        streams={key:dict(bytes=len(raw),sha256=digest(raw),records=0) for key,raw in payloads.items() if key.startswith('evidence/')})
    record['outcome']['class']=record['outcome'].pop('class_')
    payloads['record.json']=canonical(record)
    manifest=dict(schema='gomad.artifact-manifest/v2',record=record,record_sha256=digest(payloads['record.json']),
        files={key:dict(bytes=len(raw),sha256=digest(raw),module=key=='target/module.wasm') for key,raw in payloads.items()})
    artifact=gate.artifact_bytes(canonical(manifest),payloads['record.json'],payloads)
    requested=gate.CASES[case][2]==7
    replay=dict(manifest=manifest,record_sha256=digest(payloads['record.json']),retained_inputs=retained_inputs,original_module_absent=True,installation_absent=True,
        public_replay_exit=status if requested else None,public_replay_output=f'gomad: reproduced=true diagnostic=false result={"success" if policy else "target_failure"} choice-replay=exact\n' if requested else '')
    row=dict(schema='gomad.retained-reader-observation/v1',original_root=gate.SOFT,case=case,inputs=i,observed=dict(report=report,replay=replay))
    return row,artifact


class GuestReader(unittest.TestCase):
    def test_all_eleven_variant_configurations_keep_original_owners_bounds_and_intent(self):
        for case in gate.CASES:
            original=config(case)
            gate.configuration(case,original)
            for edit in ('actions','broker','network','arguments','fuel'):
                bad=copy.deepcopy(original)
                if edit=='actions':bad['steps']=bad['steps'][:-1]
                elif edit=='broker':bad['broker']['limits']['max_transitions']+=1
                elif edit=='network':bad['network']['model']['limits']['connection_bytes']+=1
                elif edit=='arguments':bad['incarnations'][0]['arguments'].append('synthetic')
                else:bad['incarnations'][0]['limits']['fuel']+=1
                with self.subTest(case=case,edit=edit),self.assertRaises(Refusal):gate.configuration(case,bad)

    def test_capacity_all_original_policies_and_budget(self):
        for case in gate.CASES:
            if case.startswith('capacity/'):
                self.assertEqual(gate.validate_case(capacity(case))['original_root'],gate.CAPACITY)

    def test_capacity_never_accepts_forged_counters_choice_policy_or_class(self):
        row=capacity('capacity/report-only=false/maximum=4096')
        for edit in ('stdout','offered','choice','terminal','message','complete','policy','budget','publication','bool'):
            bad=copy.deepcopy(row);o=bad['observed']
            if edit=='stdout':o['stdout_base64']='UEFTUwo='
            elif edit=='offered':o['outcome']['output_bytes']=5
            elif edit=='choice':o['choice_bytes']=1
            elif edit=='terminal':o['outcome']['termination']='infrastructure'
            elif edit=='message':o['outcome']['message']='transcript capacity'
            elif edit=='complete':o['assessment']['oracles'][0]['inconclusive']=False
            elif edit=='policy':o['assessment']['capacity_failure']['soft_assertions']['report_only']=True
            elif edit=='budget':bad['inputs']['configuration']['incarnations'][0]['limits']['output_bytes']=5
            elif edit=='publication':o['publication_error']='PASS'
            else:o['choice_bytes']=False
            with self.subTest(edit=edit),self.assertRaises(Refusal):gate.validate_case(bad)

    def test_soft_seed_and_policy_reports_rederive_from_raw_bytes(self):
        for case in gate.CASES:
            if case.startswith('soft/'):
                row,artifact=soft(case)
                self.assertEqual(gate.validate_case(row,artifact)['case'],case)
        row,artifact=soft(message='forged invariant')
        with self.assertRaises(Refusal):gate.validate_case(row,artifact)

    def test_soft_cannot_relabel_replay_or_remove_original_raw_retention(self):
        row,artifact=soft()
        for edit in ('label','status','report','manifest','config','seed','module','missing','evidence','alias'):
            bad=copy.deepcopy(row);raw=copy.deepcopy(artifact)
            if edit=='label':bad['original_root']=gate.CANARY
            elif edit=='status':bad['observed']['replay']['public_replay_exit']=0
            elif edit=='report':bad['observed']['report']['oracles'][0]['passed']=True
            elif edit=='manifest':bad['observed']['replay']['manifest']['record_sha256']='0'*64
            elif edit=='config':bad['inputs']['configuration']['steps'].reverse()
            elif edit=='seed':bad['inputs']['configuration']['broker']['seed']=19
            elif edit=='module':bad['inputs']['module_sha256']='0'*64
            elif edit=='missing':raw=None
            elif edit=='evidence':raw['payloads']['evidence/cluster/a/1/stderr']=b''
            else:bad['observed']['replay']['public_replay_output']='PASS reproduced=true choice-replay=exact'
            with self.subTest(edit=edit),self.assertRaises(Refusal):gate.validate_case(bad,raw)

    def test_raw_artifact_resealing_does_not_hide_identity_or_payload_mutations(self):
        _,a=soft();p=a['payloads'];m=a['manifest']
        for edit in ('record','manifest','payload','missing','descriptor','identity','stream'):
            payloads=copy.deepcopy(p);manifest=copy.deepcopy(m)
            if edit=='record':payloads['record.json']+=b'\n'
            elif edit=='manifest':manifest['record_sha256']='0'*64
            elif edit=='payload':payloads['target/module.wasm']+=b'\0'
            elif edit=='missing':del payloads[gate.FILE]
            elif edit=='descriptor':manifest['files']['target/module.wasm']['module']=False
            elif edit=='identity':manifest['record']['execution_identity']='0'*64
            else:manifest['record']['streams']['evidence/world']['bytes']+=1
            with self.subTest(edit=edit),self.assertRaises(Refusal):gate.artifact_bytes(canonical(manifest),payloads['record.json'],payloads)

    def test_history_canary_independently_preserves_four_oracles(self):
        for canary in (False,True):
            values={}
            for owner in ('a/1','b/1'):
                history=b'different history' if canary and owner=='b/1' else b'committed history'
                value=dict(cluster_id='independent-oracle-reference',history=base64.b64encode(history).decode(),event_ids=[1,2],types=['started','completed'])
                values[owner]=dict(stdout=b'fresh heap 1\nGOMAD_HISTORY_OBSERVATION '+canonical(value)+b'\n')
            result=gate.history_report(values,canary)
            self.assertEqual(result['outcome'],'oracle_failed' if canary else 'completed')
            self.assertEqual([item['passed'] for item in result['oracles']],[not canary,True,True,not canary])
            bad=copy.deepcopy(values);bad['a/1']['stdout']*=2
            with self.assertRaises(Refusal):gate.history_report(bad,canary)
            with self.assertRaises(Refusal):gate.history_report(values,not canary)

    def test_closed_variant_inventory_refuses_omitted_duplicate_unknown(self):
        rows=[dict(case=case) for case in gate.CASES]
        text='\n'.join(gate.MARKER+canonical(row).decode() for row in rows)
        self.assertEqual(len(gate.parse_rows(text)),11)
        for value in (text.split('\n',1)[1],text+'\n'+text.splitlines()[0],text+'\n'+gate.MARKER+'{"case":"PASS"}'):
            with self.assertRaises(Refusal):gate.parse_rows(value)
        with self.assertRaises(Refusal):gate.observations({},dict(schema='PASS'))

    def test_raw_custody_refuses_symlink_escape_and_changes(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory).resolve();path=root/'raw';path.write_bytes(b'original')
            files=gate.RawFiles(root)
            self.assertEqual(files.read(path),b'original')
            files.finish();path.write_bytes(b'changed')
            with self.assertRaises(Refusal):files.finish()
            link=root/'alias';link.symlink_to(path)
            with self.assertRaises(Refusal):gate.RawFiles(root).read(link)
            with self.assertRaises(Refusal):gate.RawFiles(root/'nested').read(path)


if __name__=='__main__':unittest.main()
