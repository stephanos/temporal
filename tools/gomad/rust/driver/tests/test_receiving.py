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
        self.assertEqual(reader.nested(self.root,'fixture',row['argv']),b'123\n')
        (self.root/'fixture.stdout').write_bytes(b'124\n')
        with self.assertRaises(driver.Refusal): reader.nested(self.root,'fixture',row['argv'])

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
        compiler=self.root/'compiler'
        compiler.write_bytes(b'compiler')
        compiler_hash=receiving.digest(b'compiler')
        self.config['files'][str(compiler)]=compiler_hash
        inputs.update(compiler_path=str(compiler),module_metadata=dict(imports=[],initial_memory_pages=1),schema='gomad.rust-wasm-build-inputs/v1',configuration=configuration,
                      compiler=dict(runtime='compiler'),compiler_sha256=compiler_hash,helper='d'*64,host='e'*64,sources=[])
        module=b'\x00asm\x01\x00\x00\x00\x05\x03\x01\x00\x01'
        module_hash=receiving.digest(module)
        key=receiving.digest(receiving.canonical(inputs))
        provenance=dict(module_metadata=inputs['module_metadata'],schema='gomad.rust-wasm-provenance/v1',inputs=inputs,configuration=configuration,
                        build_key=key,module_sha256=module_hash,choice_identity=None,network=None,scenario=None)
        source_manifest=self.root/'source.sha256'
        source_manifest.write_text(compiler_hash+'  '+str(compiler)+'\n')
        source_hash=receiving.digest(source_manifest.read_bytes())
        self.config['source_manifest']=str(source_manifest)
        self.config['files'][str(source_manifest)]=source_hash
        manifest=dict(integrated_source_sha256=source_hash,host_os='linux',host_arch='fixture',provisional=False,repeat=2,workload_source_sha256=receiving.digest(receiving.canonical([])),
                      runtime_sha256=receiving.digest(receiving.canonical(inputs['compiler'])),compiler_sha256=compiler_hash,
                      helper_sha256='d'*64,host_sha256='e'*64,model_sha256=receiving.digest(b'null'),
                      engine_sha256=receiving.digest(b'engine'),host_config_sha256=receiving.digest(receiving.canonical(configuration)),
                      module_sha256=module_hash,toolchain_sha256=key,scenario_sha256=None,profile='controlled',
                      fuel=4_000_000_000,memory_bytes=256<<20,seed=7)
        executions=[]
        for iteration in range(1,3):
            artifact=self.root/f'artifact-{iteration}'
            (artifact/'target').mkdir(parents=True)
            (artifact/'target/module.wasm').write_bytes(module)
            (artifact/'evidence').mkdir()
            (artifact/'evidence/transcript').write_bytes(b'x'*4096)
            stream=dict(bytes=4096,sha256=receiving.digest(b'x'*4096))
            record=dict(identity=dict(target=module_hash,toolchain=key,configuration=manifest['host_config_sha256'],
                        implementation=receiving.digest(receiving.canonical(dict(host='e'*64,helper='d'*64))),
                        platform=receiving.digest(b'linux/fixture')),seed=7,streams={'evidence/transcript':stream},
                        virtual_time_nanos=5000000,limits=configuration['limits'],outcome={'class':'success','termination':'exit','exit_code':0},
                        profile=configuration['profile'])
            record_hash=receiving.digest(receiving.canonical(record))
            native=dict(schema='gomad.artifact-manifest/v2',record=record,record_sha256=record_hash,
                        files={'target/module.wasm':dict(bytes=len(module),sha256=module_hash,module=True),'evidence/transcript':dict(stream,module=False)})
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
                                   record_sha256=record_hash,replay=dict(attempted=True,matched=True,divergence=None,choice_replay='none')))
        parity=Path(receiving.__file__).resolve().parents[2]/'testdata/parity'
        cases=json.loads((parity/'qualify-accounting-corpus.json').read_bytes())['cases']
        import copy
        observations=copy.deepcopy(next(row['observations'] for row in cases if row['name']=='phases-all-within'))
        result=dict(schema='gomad.rust-qualification-result/v1',manifest=manifest,observations=observations,
                    provenance=provenance,integrated_source_manifest=source_manifest.read_text(),
                    report=dict(schema='gomad.rust-qualification/v1',executions=executions,seed=7,repeat=2,
                                qualified=True,deterministic=True,target_success=True,first_divergence=None,failure=None,
                                record_sha256=executions[0]['record_sha256'],command=['qualification-fixture']))
        result.update(cost=receiving.receiving_cost.cost(observations,result['report']),cost_failure=None,
                      budget=receiving.receiving_cost.budget(observations),budget_failure=None)
        raw=receiving.canonical(result)+b'\n'
        path=self.root/('result-'+receiving.digest(raw)+'.json')
        path.write_bytes(raw)
        path.chmod(0o600)
        receipt=dict(schema='gomad.rust-qualification-receipt/v1',result_sha256=receiving.digest(raw),operator_process_id=observations['operator_process_id'],
                     linked_after=dict(observations['executed']['linked'],user_nanos=observations['executed']['linked']['user_nanos']+2),
                     operator_peak_rss_bytes=observations['operator_peak_rss_bytes'])
        raw_receipt=receiving.canonical(receipt)+b'\n'
        receipt_path=self.root/(path.name+'.publication-'+receiving.digest(raw_receipt)+'.json')
        receipt_path.write_bytes(raw_receipt)
        receipt_path.chmod(0o600)
        self.config.update(source_sha256=source_hash,identity=dict(host='e'*64,helper='d'*64))
        return path,dict(result=result,receipt=receipt,publication=dict(cpu_nanos=2,peak_rss_bytes=observations['operator_peak_rss_bytes'],status='within'))

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
        (self.root/'campaign-2/journal.jsonl').unlink()
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

    def test_genuine_preparation_module_metadata_is_recomputed_from_raw_wasm(self):
        module=b'\x00asm\x01\x00\x00\x00\x01\x06\x01\x60\x01\x7f\x01\x7e\x02\x07\x01\x01m\x01f\x00\x00\x05\x03\x01\x00\x01'
        self.assertEqual(receiving.module_metadata(module),dict(imports=[dict(module='m',name='f',params=['i32'],results=['i64'])],initial_memory_pages=1))
        for wrong in (module[:-5],module[:-1]+b'\x00',module[:-2]+b'\x02\x01',module+b'\x05\x03\x01\x00\x01'):
            with self.subTest(raw=wrong),self.assertRaises(driver.Refusal): receiving.module_metadata(wrong)

    def test_runtime_selection_requires_exact_owner_context_and_no_leak_to_other_boots(self):
        import copy
        context=dict(PendingSHA256='sha256:'+'a'*64,Operations=[])
        state=dict(Volume='temporal',PendingSHA256=context['PendingSHA256'],SelectedOperations=[],Entries=[])
        state['Identity']=receiving.state_identity(state)
        raw=receiving.canonical(dict(temporal=context))
        storage=dict(descriptor=dict(crash_selection=dict(temporal=[]),crash_context=dict(bytes=len(raw),chunks=1,sha256=receiving.digest(raw))),context=list(raw))
        value=dict(incarnations=[dict(handle=dict(node='temporal',incarnation=1),storage=storage),
                                 dict(handle=dict(node='temporal',incarnation=2),storage=dict(descriptor={},context=None))])
        receiving.runtime_selection(value,'temporal',1,state,context)
        mutations=[lambda row:row['incarnations'][0]['storage']['descriptor'].__setitem__('crash_selection',None),
                   lambda row:row['incarnations'][0]['storage'].__setitem__('context',[]),
                   lambda row:row['incarnations'][1]['storage']['descriptor'].__setitem__('crash_selection',dict(temporal=[]))]
        for mutate in mutations:
            changed=copy.deepcopy(value);mutate(changed)
            with self.assertRaises(driver.Refusal): receiving.runtime_selection(changed,'temporal',1,state,context)

    def test_original_typed_zero_capture_has_the_empty_namespace_projection(self):
        import copy
        policy=receiving.workloads.inspection_policy('shard')
        policy.pop('CapturedInputs',None)
        expected=receiving.inspection_policy_projection(policy)
        policy['CapturedInputs']=dict(Manifest=dict(Bytes=0,Entries=0,File='',Limits=dict(
            DirectoryEntries=0,Files=0,PathBytes=0,Requests=0,SingleFileBytes=0,TotalBytes=0),
            Mappings=None,NotExist=0,Schema='',SHA256='',TotalBytes=0),Descriptor=None,Payloads=None)
        self.assertEqual(receiving.inspection_policy_projection(policy),expected)
        empty=copy.deepcopy(policy)
        empty['CapturedInputs'].update(Descriptor='',Payloads={})
        self.assertEqual(receiving.inspection_policy_projection(empty),expected)
        partial=copy.deepcopy(policy);partial['CapturedInputs']=dict(Manifest={})
        self.assertEqual(receiving.inspection_policy_projection(partial),expected)
        for descriptor,payloads in [('eA==',None),('!',None),(None,{'blob':'eA=='}),(False,None),(None,[])]:
            changed=copy.deepcopy(policy)
            changed['CapturedInputs'].update(Descriptor=descriptor,Payloads=payloads)
            with self.subTest(descriptor=descriptor,payloads=payloads),self.assertRaises(driver.Refusal):
                receiving.inspection_policy_projection(changed)

    def test_inspection_admission_rejects_foreign_release_source_toolchain_and_policy(self):
        import copy
        policy=receiving.workloads.inspection_policy('shard')
        model,namespace,environment,network=receiving.inspection_policy_projection(policy)
        compiler=str(self.root/'compiler')
        inputs={compiler:'c'*64}
        installed=dict(schema='gomad.rust-wasm-installation/v1',profile='gomad.wasi-cooperative/v1',fuel=50_000_000_000,
                       memory_bytes=1<<30,host_sha256='a'*64,helper_sha256='b'*64,compiler_path=compiler,compiler_sha256='c'*64,
                       runtime_implementation_sha256='d'*64,model=model,namespace=namespace,storage=None,network=network,build_tags=[],
                       admission=dict(host_executable_sha256='e'*64,guest_configuration=policy,configuration_sha256=receiving.digest(receiving.canonical(policy)),
                                      compiler_inputs=inputs,inputs_sha256=receiving.digest(receiving.canonical(inputs)),environment=environment,arguments=[]))
        self.put('frozen-installation.json',installed)
        self.config.update(identity=dict(host='a'*64,helper='b'*64),binary_sha256='e'*64,
                           installations=dict(runtime=str(self.root/'frozen-installation.json')))
        self.config['files'][compiler]='c'*64
        reader=receiving.Retained(self.config)
        reader.installed(installed,policy,50_000_000_000,1<<30)
        mutations=[lambda row:row.__setitem__('host_sha256','f'*64),
                   lambda row:row.__setitem__('helper_sha256','f'*64),
                   lambda row:row.__setitem__('runtime_implementation_sha256','f'*64),
                   lambda row:row['admission'].__setitem__('host_executable_sha256','f'*64),
                   lambda row:row['model'].__setitem__('entropy_key',[0]*31+[1]),
                   lambda row:row['namespace']['limits'].__setitem__('files',1),
                   lambda row:row['admission']['guest_configuration']['Clock'].__setitem__('EpochNanos',1),
                   lambda row:row['admission']['compiler_inputs'].__setitem__(compiler,'f'*64)]
        for mutate in mutations:
            changed=copy.deepcopy(installed);mutate(changed)
            with self.assertRaises(driver.Refusal): reader.installed(changed,policy,50_000_000_000,1<<30)

    def test_nested_cannot_choose_self_reported_foreign_argv_or_duplicate_owner_selector(self):
        self.observation()
        row=driver.read(self.root/'discover.command.json')
        row.update(command=row['argv'],stdout=str(self.root/'discover.stdout'),stderr=str(self.root/'discover.stderr'),
                   stdout_sha256=receiving.digest(b'raw'),stderr_sha256=receiving.digest(b''),
                   argv_sha256=receiving.digest(receiving.canonical(row['argv'])),environment={},environment_sha256='0'*64)
        self.put('discover.command.json',row)
        reader=receiving.Retained(self.config)
        with self.assertRaises(driver.Refusal): reader.nested(self.root,'discover',['/sealed/foreign','discover'])
        with self.assertRaises(driver.Refusal): receiving.argument(['--owner','all','--owner','shard'],'--owner')

    def test_selected_recovery_configuration_cannot_substitute_unselected_primary_or_other_context(self):
        import copy
        primary=self.root/'primary'
        selected=self.root/'selected'
        prefix=primary/'evidence/cluster/temporal/1'
        capsule=selected/'evidence/primary-crash'
        prefix.mkdir(parents=True);capsule.mkdir(parents=True)
        context=dict(PendingSHA256='sha256:'+'a'*64,Operations=[])
        state=dict(Volume='temporal',PendingSHA256=context['PendingSHA256'],SelectedOperations=[],Entries=[])
        state['Identity']=receiving.state_identity(state)
        expected_history=dict(cluster_id='primary',history='eA==',event_ids=[1],types=['start'])
        for name in ('choice','storage_state','storage','stdout','recovery_image_primary'):
            (prefix/name).write_bytes(name.encode())
            (capsule/name).write_bytes(name.encode())
        primary_record=dict(identity=dict(configuration='unused'))
        (primary/'manifest.json').write_bytes(receiving.canonical(dict(record=primary_record)))
        (capsule/'record.json').write_bytes(receiving.canonical(primary_record))
        configurations=[]
        for incarnation in (1,2,3):
            configurations.append(dict(handle=dict(node='temporal',incarnation=incarnation),storage=dict(descriptor={},context=None)))
        original=dict(incarnations=configurations,steps=[dict(handle=dict(node='temporal',incarnation=i)) for i in (1,2,3)],
                      workload=dict(kind='temporal-committed-workflow-crash-reopen',forcing=None))
        (primary/'evidence/cluster-replay.json').write_bytes(receiving.canonical(dict(configuration=original)))
        (capsule/'cluster-replay.json').write_bytes(receiving.canonical(dict(configuration=original)))
        request=dict(kind='temporal-committed-workflow-crash-reopen',primary=dict(artifact_path=str(primary),
                     record_sha256=receiving.digest(receiving.canonical(primary_record)),prefix_sha256=receiving.digest(b'choice'),
                     storage_state_sha256=receiving.digest(b'storage_state'),state=state,context=context))
        forcing=dict(primary_record_sha256=request['primary']['record_sha256'],prefix_sha256=request['primary']['prefix_sha256'],
                     storage_state_sha256=request['primary']['storage_state_sha256'],state=state,context=context,expected=expected_history,
                     storage_sha256=receiving.digest(b'storage'),stdout_sha256=receiving.digest(b'stdout'),primary_image_sha256=receiving.digest(b'recovery_image_primary'))
        actual=copy.deepcopy(original)
        actual['incarnations']=actual['incarnations'][:2]
        actual['steps']=actual['steps'][:2]
        actual['workload']['forcing']=forcing
        raw=receiving.canonical(dict(temporal=context))
        actual['incarnations'][0]['storage']=dict(descriptor=dict(crash_selection=dict(temporal=[]),
            crash_context=dict(bytes=len(raw),chunks=1,sha256=receiving.digest(raw))),context=list(raw))
        def retain(value):
            (selected/'evidence/cluster-replay.json').write_bytes(receiving.canonical(dict(configuration=value)))
            (selected/'manifest.json').write_bytes(receiving.canonical(dict(record=dict(identity=dict(configuration=receiving.digest(receiving.canonical(value)))))))
        retain(actual)
        reader=receiving.Retained(self.config)
        reader.recovery_configuration(primary,selected,request,expected_history)
        retain(original)
        with self.assertRaises(driver.Refusal): reader.recovery_configuration(primary,selected,request,expected_history)
        wrong=copy.deepcopy(actual);wrong['workload']['forcing']['context']['PendingSHA256']='changed';retain(wrong)
        with self.assertRaises(driver.Refusal): reader.recovery_configuration(primary,selected,request,expected_history)
        retain(actual)
        (capsule/'choice').write_bytes(b'changed')
        with self.assertRaises(driver.Refusal): reader.recovery_configuration(primary,selected,request,expected_history)

    def test_inspection_prepared_policy_keeps_exact_original_clock_choice_and_execution_bounds(self):
        policy=receiving.workloads.inspection_policy('shard')
        model,namespace,environment,network=receiving.inspection_policy_projection(policy)
        installed=dict(model=model,namespace=namespace,admission=dict(environment=environment),fuel=50_000_000_000,memory_bytes=1<<30,network=network)
        configured=receiving.inspection_configuration(installed,['-test.run=^original$','-test.v'],'shard','a'*64)
        self.assertEqual(configured['model']['limit'],8<<20)
        self.assertEqual(configured['limits'],dict(fuel=50_000_000_000,memory_bytes=1<<30,wall_millis=300_000,
                         output_bytes=1<<20,evidence_bytes=(8<<20)+(2<<20)+(128<<20)))
        self.assertEqual(configured['model']['epoch_nanos'],946684800000000000)
        self.assertNotIn('seed',configured['model'])
        self.assertNotIn('network',configured)
        self.assertEqual(configured['environment']['GOMAD_CHOICE_PROFILE'],'gomad-choice-trace/v4')
