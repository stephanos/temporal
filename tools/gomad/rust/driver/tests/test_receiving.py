import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import receiving


class ReceivingContract(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.config = dict(root=str(self.root), repo=str(self.root), sent_epoch=1,
                           binary='/sealed/release', files={'/sealed/release': 'a'*64})

    def put(self, name, value):
        (self.root/name).write_bytes(receiving.canonical(value)+b'\n')

    def observation(self):
        row = dict(argv=['/sealed/release', 'discover'], cwd=str(self.root), pid=123,
                   exit=0, observation='completed', start_ns=2_000_000_000, end_ns=3_000_000_000)
        self.put('discover.command.json', row)
        self.put('discover.process.json', dict(row, active=False, stage_pid=124))
        (self.root/'discover.stdout').write_bytes(b'raw')
        (self.root/'discover.stderr').write_bytes(b'')
        return row

    def test_command_authenticates_registration_and_exact_argv(self):
        self.observation()
        reader = receiving.Retained(self.config)
        self.assertEqual(reader.command(self.config, self.root, 'discover', ['/sealed/release','discover']), b'raw')
        with self.assertRaises(driver.Refusal):
            reader.command(self.config, self.root, 'discover', ['/sealed/release','report'])

    def test_command_rejects_missing_failed_borrowed_and_incomplete_process(self):
        original = self.observation()
        for patch in [dict(pid=0), dict(exit=1), dict(observation='inconclusive'), dict(cwd='/foreign'),
                      dict(start_ns=0), dict(end_ns=1), dict(pid=True)]:
            with self.subTest(patch=patch):
                self.put('discover.command.json', dict(original, **patch))
                with self.assertRaises(driver.Refusal):
                    receiving.Retained(self.config).command(self.config,self.root,'discover',original['argv'])
        self.put('discover.command.json', original)
        (self.root/'discover.process.json').unlink()
        with self.assertRaises(driver.Refusal):
            receiving.Retained(self.config).command(self.config,self.root,'discover',original['argv'])

    def test_command_root_and_symlink_cannot_borrow_observations(self):
        row = self.observation()
        outside = self.root.parent/'foreign-receiving-root'
        with self.assertRaises(driver.Refusal):
            receiving.Retained(self.config).command(self.config,outside,'discover',row['argv'])
        (self.root/'discover.stdout').unlink()
        (self.root/'discover.stdout').symlink_to(self.root/'discover.stderr')
        with self.assertRaises(driver.Refusal):
            receiving.Retained(self.config).command(self.config,self.root,'discover',row['argv'])

    def test_read_only_recomputation_rejects_changed_or_missing_summary(self):
        reader = receiving.Retained(self.config)
        self.put('summary.json', dict(actual=1))
        reader.write(self.root/'summary.json', dict(actual=1))
        with self.assertRaises(driver.Refusal): reader.write(self.root/'summary.json', dict(actual=2))
        with self.assertRaises(driver.Refusal): reader.write(self.root/'absent.json', dict(actual=1))
        self.assertFalse((self.root/'absent.json').exists())

    def test_artifact_reopens_every_raw_payload_and_native_module(self):
        body=b'\x00asm'
        record=dict(identity=dict(target=hashlib.sha256(body).hexdigest()), streams={})
        manifest=dict(schema='gomad.artifact-manifest/v2', record=record,
                      record_sha256=hashlib.sha256(receiving.canonical(record)).hexdigest(),
                      files={'target/module.wasm':dict(bytes=len(body),sha256=hashlib.sha256(body).hexdigest(),module=True)})
        (self.root/'target').mkdir()
        (self.root/'target/module.wasm').write_bytes(body)
        self.put('manifest.json',manifest)
        receiving.Retained(self.config).artifact(self.root)
        (self.root/'target/module.wasm').write_bytes(b'changed')
        with self.assertRaises(driver.Refusal): receiving.Retained(self.config).artifact(self.root)

    def test_artifact_rejects_record_tamper_and_missing_module(self):
        manifest=dict(schema='gomad.artifact-manifest/v2',record={},record_sha256='0'*64,files={})
        self.put('manifest.json',manifest)
        with self.assertRaises(driver.Refusal): receiving.Retained(self.config).artifact(self.root)
        manifest['record_sha256']=hashlib.sha256(receiving.canonical({})).hexdigest()
        self.put('manifest.json',manifest)
        with self.assertRaises(driver.Refusal): receiving.Retained(self.config).artifact(self.root)

    def test_unbound_temporal_does_not_accept_status_only_receipt(self):
        self.put('recovery-receipt.json',dict(subgates=[dict(status='PASS',executed=True,exit=0)]))
        with self.assertRaises(driver.Unexecuted):
            receiving.Retained(self.config).temporal('storage')

    def test_history_rejects_duplicate_events_and_observation_lines(self):
        value=dict(cluster_id='active',history='eA==',event_ids=[1,2],types=['start','complete'])
        raw=b'GOMAD_TEMPORAL_OBSERVATION '+receiving.canonical(value)+b'\n'
        self.assertEqual(receiving.history(raw,sequential=True),value)
        with self.assertRaises(driver.Refusal): receiving.history(raw+raw,sequential=True)
        value['event_ids']=[1,1]
        with self.assertRaises(driver.Refusal):
            receiving.history(b'GOMAD_TEMPORAL_OBSERVATION '+receiving.canonical(value),sequential=True)

    def test_native_context_requires_dependency_closure_and_full_wire_integrity(self):
        import struct
        context=dict(PendingSHA256='sha256:'+'a'*64,Operations=[dict(ID=1,Dependencies=None),dict(ID=2,Dependencies=[1])])
        state=dict(Identity='sha256:'+'b'*64,PendingSHA256=context['PendingSHA256'],Volume='temporal',SelectedOperations=[1,2],Entries=[])
        state['Identity']=receiving.state_identity(state)
        receiving.native_state(state,context,'temporal',64)
        for selected in ([2],[1,1],[3]):
            with self.assertRaises(driver.Refusal):
                receiving.native_state(dict(state,SelectedOperations=selected),context,'temporal',64)
        body=b'GMSTORE\x01'+struct.pack('<II',3,6)
        for tag in range(1,7): body += struct.pack('<IQ',tag,1)+bytes([tag])
        raw=body+struct.pack('<Q',len(body))+hashlib.sha256(body).digest()
        self.assertEqual(receiving.storage_sections(raw)[4],b'\x04')
        with self.assertRaises(driver.Refusal): receiving.storage_sections(raw[:-1]+b'x')
        with self.assertRaises(driver.Refusal): receiving.storage_sections(raw+b'x')

    def retry_fixture(self):
        value=dict(workflow_id='workflow',run_id='run',result='durable timer survived rpc-reset',history='eA==',
                   event_ids=[1,2],types=['start','complete'],connections=[dict(local_port=10,opened_nanos=1),
                   dict(local_port=20,opened_nanos=40_000_000_001)],attempts=[
                   dict(started_nanos=1,ended_nanos=40_000_000_001,code='Unavailable'),
                   dict(started_nanos=40_000_000_002,ended_nanos=40_000_000_003,code='OK')])
        rows=[dict(at=1,request=dict(op='dial'),result=dict(state='completed',error='',handle=1,local=dict(port=10))),
              dict(at=2,request=dict(op='read',request_id=1,handle=1),result={}),
              dict(at=40_000_000_000,fault=dict(name='rpc-reset',action='reset',at_nanos=40_000_000_000),
                   request=dict(op='poll',request_id=1),result=dict(error='connection reset by peer')),
              dict(at=40_000_000_002,request=dict(op='dial'),result=dict(state='completed',error='',handle=2,local=dict(port=20)))]
        return value,dict(operations=rows)

    def test_retry_requires_actual_established_stream_fault_reconnect_and_inflight_retry(self):
        import copy
        value,network=self.retry_fixture()
        def raw(value): return b'GOMAD_NETWORK_TEMPORAL_OBSERVATION '+receiving.canonical(value)
        receiving.retry_witness(raw(value),network)
        for index in range(4):
            changed=copy.deepcopy(network)
            del changed['operations'][index]
            with self.subTest(omitted=index), self.assertRaises(driver.Refusal):
                receiving.retry_witness(raw(value),changed)
        value['attempts']=value['attempts'][:1]
        with self.assertRaises(driver.Refusal): receiving.retry_witness(raw(value),network)

    def test_nested_producer_retains_truthful_pid_time_argv_environment_and_stream_hashes(self):
        import importlib.util
        producer_path=Path(receiving.__file__).resolve().parents[1]/'workload_tools/producer.py'
        sys.path.insert(0,str(producer_path.parent))
        self.addCleanup(lambda:sys.path.remove(str(producer_path.parent)))
        spec=importlib.util.spec_from_file_location('receiving_test_producer',producer_path)
        producer=importlib.util.module_from_spec(spec)
        spec.loader.exec_module(producer)
        result=producer.run(self.root,'fixture',[sys.executable,'-c','print(123)'],0)
        self.assertEqual(result,123)
        row=driver.read(self.root/'fixture.command.json')
        self.assertGreater(row['pid'],0)
        self.assertLessEqual(row['start_ns'],row['end_ns'])
        self.assertEqual(row['stdout_sha256'],hashlib.sha256(b'123\n').hexdigest())
        self.assertEqual(row['argv_sha256'],receiving.digest(receiving.canonical(row['argv'])))
        self.config.update(repo=row['cwd'],files={sys.executable:'a'*64})
        reader=receiving.Retained(self.config)
        self.assertEqual(reader.nested(self.root,'fixture'),b'123\n')
        (self.root/'fixture.stdout').write_bytes(b'124\n')
        with self.assertRaises(driver.Refusal): reader.nested(self.root,'fixture')

    def test_journal_recomputes_plan_link_ordinal_receipt_and_completion(self):
        plan=b'{}'
        (self.root/'plan.json').write_bytes(plan)
        self.put('storage.json',dict(plan_sha256=receiving.digest(plan),max_bytes=4096))
        previous=receiving.digest(plan)
        rows=[]
        for ordinal,event in enumerate([dict(kind='receipt',execution=dict(record=dict(seed=7))),dict(kind='complete')]):
            row=dict(ordinal=ordinal,previous_sha256=previous,event=event)
            row['sha256']=receiving.digest(receiving.canonical(row))
            rows.append(row)
            previous=row['sha256']
        raw=b''.join(receiving.canonical(row)+b'\n' for row in rows)
        (self.root/'journal.jsonl').write_bytes(raw)
        receiving.Retained(self.config).journal(self.root,receiving.digest(receiving.canonical(dict(seed=7))))
        for wrong in [raw[:-1],raw.replace(b'7',b'8'),raw+raw]:
            (self.root/'journal.jsonl').write_bytes(wrong)
            with self.assertRaises(driver.Refusal): receiving.Retained(self.config).journal(self.root,receiving.digest(receiving.canonical(dict(seed=7))))

    def result_fixture(self):
        configuration=dict(profile='gomad.wasi-stock-exploratory/v1',runtime_sha256=None,
                           model=None,engine='engine',limits=dict(fuel=4_000_000_000,memory_bytes=256<<20))
        fields=('arguments','build_flags','compiler','compiler_sha256','configuration','directory','environment','go_version',
                'helper','host','kind','network','runtime','runtime_implementation','scenario','schema','soft_assertion_report_only','source','sources','tags')
        inputs={key:None for key in fields}
        inputs.update(schema='gomad.rust-wasm-build-inputs/v1',configuration=configuration,
                      compiler=dict(runtime='compiler'),compiler_sha256='c'*64,helper='d'*64,host='e'*64,sources=[])
        module=b'\x00asm'
        module_hash=receiving.digest(module)
        key=receiving.digest(receiving.canonical(inputs))
        provenance=dict(schema='gomad.rust-wasm-provenance/v1',inputs=inputs,configuration=configuration,
                        build_key=key,module_sha256=module_hash,choice_identity=None,network=None,scenario=None)
        manifest=dict(integrated_source_sha256='f'*64,workload_source_sha256=receiving.digest(receiving.canonical([])),
                      runtime_sha256=receiving.digest(receiving.canonical(inputs['compiler'])),compiler_sha256='c'*64,
                      helper_sha256='d'*64,host_sha256='e'*64,model_sha256=receiving.digest(b'null'),
                      engine_sha256=receiving.digest(b'engine'),host_config_sha256=receiving.digest(receiving.canonical(configuration)),
                      module_sha256=module_hash,toolchain_sha256=key,scenario_sha256=None,profile='controlled',
                      fuel=4_000_000_000,memory_bytes=256<<20,seed=7)
        executions=[]
        for iteration in range(2):
            artifact=self.root/f'artifact-{iteration}'
            (artifact/'target').mkdir(parents=True)
            (artifact/'target/module.wasm').write_bytes(module)
            record=dict(identity=dict(target=module_hash,toolchain=key),seed=7,streams={})
            record_hash=receiving.digest(receiving.canonical(record))
            native=dict(schema='gomad.artifact-manifest/v2',record=record,record_sha256=record_hash,
                        files={'target/module.wasm':dict(bytes=4,sha256=module_hash,module=True)})
            (artifact/'manifest.json').write_bytes(receiving.canonical(native))
            campaign=self.root/f'campaign-{iteration}'
            campaign.mkdir()
            (campaign/'plan.json').write_bytes(b'{}')
            (campaign/'storage.json').write_bytes(receiving.canonical(dict(plan_sha256=receiving.digest(b'{}'),max_bytes=4096)))
            previous=receiving.digest(b'{}')
            raw=b''
            for ordinal,event in enumerate([dict(kind='receipt',execution=dict(record=record)),dict(kind='complete')]):
                row=dict(ordinal=ordinal,previous_sha256=previous,event=event)
                row['sha256']=receiving.digest(receiving.canonical(row))
                previous=row['sha256']
                raw += receiving.canonical(row)+b'\n'
            (campaign/'journal.jsonl').write_bytes(raw)
            executions.append(dict(iteration=iteration,artifact_path=str(artifact),campaign_path=str(campaign),record=record,
                                   record_sha256=record_hash,replay=dict(attempted=True,matched=True,divergence=None,choice_replay='exact')))
        observations=dict(operator_process_id=123,operator_peak_rss_bytes=20,
                          executed=dict(linked=dict(user_nanos=10,system_nanos=10,peak_rss_bytes=20)))
        result=dict(schema='gomad.rust-qualification-result/v1',manifest=manifest,observations=observations,
                    provenance=provenance,report=dict(executions=executions))
        raw=receiving.canonical(result)+b'\n'
        path=self.root/('result-'+receiving.digest(raw)+'.json')
        path.write_bytes(raw)
        path.chmod(0o600)
        receipt=dict(schema='gomad.rust-qualification-receipt/v1',result_sha256=receiving.digest(raw),operator_process_id=123,
                     linked_after=dict(user_nanos=11,system_nanos=11,peak_rss_bytes=20),operator_peak_rss_bytes=20)
        raw_receipt=receiving.canonical(receipt)+b'\n'
        receipt_path=self.root/(path.name+'.publication-'+receiving.digest(raw_receipt)+'.json')
        receipt_path.write_bytes(raw_receipt)
        receipt_path.chmod(0o600)
        self.config.update(source_sha256='f'*64,identity=dict(host='e'*64,helper='d'*64))
        return path,dict(result=result,receipt=receipt,publication=dict(cpu_nanos=2,peak_rss_bytes=20,status='within'))

    def test_result_pair_recomputes_raw_lineage_provenance_journals_and_publication_tail(self):
        import copy
        path,report=self.result_fixture()
        reader=receiving.Retained(self.config)
        self.assertEqual(reader.result(path,report),report['result'])
        changed=copy.deepcopy(report)
        changed['publication']['cpu_nanos']=1
        with self.assertRaises(driver.Refusal): reader.result(path,changed)
        changed=copy.deepcopy(report)
        changed['result']['manifest']['helper_sha256']='0'*64
        with self.assertRaises(driver.Refusal): reader.result(path,changed)
        self.config['identity']['host']='0'*64
        with self.assertRaises(driver.Refusal): reader.result(path,report)
        self.config['identity']['host']='e'*64
        receipt=next(self.root.glob(path.name+'.publication-*.json'))
        receipt.write_bytes(receipt.read_bytes()+b' ')
        with self.assertRaises(driver.Refusal): reader.result(path,report)

    def test_result_pair_rejects_duplicate_receipt_and_missing_journal(self):
        path,report=self.result_fixture()
        reader=receiving.Retained(self.config)
        receipt=next(self.root.glob(path.name+'.publication-*.json'))
        duplicate=self.root/(path.name+'.publication-'+'0'*64+'.json')
        duplicate.write_bytes(receipt.read_bytes())
        with self.assertRaises(driver.Refusal): reader.result(path,report)
        duplicate.unlink()
        (self.root/'campaign-1/journal.jsonl').unlink()
        with self.assertRaises(driver.Refusal): reader.result(path,report)

    def test_selected_marker_full_metadata_and_absence_preserve_original_variants(self):
        for kind in (1,2):
            data='bWFya2Vy' if kind == 1 else None
            entry=dict(Path='/marker',Kind=kind,Mode=0o600,ModTime=7,Data=data)
            marker=dict(Path='/marker',Kind='file' if kind == 1 else 'directory',Mode=0o600,ModTime=7,
                        Size=6 if kind == 1 else 0,DataSHA256='sha256:'+receiving.digest(b'marker') if kind == 1 else '')
            state=dict(Entries=[entry])
            receiving.selected_marker([dict(ID='sqlite',Persisted=[marker])],state)
            for key,value in dict(Kind='wrong',Mode=1,ModTime=8,Size=10,DataSHA256='changed',Path='/other').items():
                with self.subTest(kind=kind,field=key), self.assertRaises(driver.Refusal):
                    receiving.selected_marker([dict(ID='sqlite',Persisted=[dict(marker,**{key:value})])],state)
            with self.assertRaises(driver.Refusal):
                receiving.selected_marker([dict(ID='sqlite',Persisted=[marker])],dict(Entries=[]))
        receiving.selected_marker([dict(ID='sqlite',Persisted=[])],dict(Entries=[]))

    def test_result_refuses_persistent_hardlink_and_public_file_mode(self):
        import os
        path,report=self.result_fixture()
        alias=self.root/'foreign-hardlink'
        os.link(path,alias)
        with self.assertRaises(driver.Refusal): receiving.Retained(self.config).result(path,report)
        alias.unlink()
        path.chmod(0o644)
        with self.assertRaises(driver.Refusal): receiving.Retained(self.config).result(path,report)
