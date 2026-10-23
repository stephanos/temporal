"""Bounded synthetic native-writer processes, never qualification producers."""
import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(HERE))
import driver
import native_observations as observations


def put(path, value):
    path.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    path.write_bytes(observations.canonical(value))
    path.chmod(0o600)
    return observations.digest(path.read_bytes())


def write_campaign(campaign, config, owner):
    campaign.mkdir(parents=True, mode=0o700)
    module = b'\0asm\x01\0\0\0'
    inputs = dict(host=config['identity']['host'], helper=config['identity']['helper'], compiler={'pinned-go':'c'*64}, sources={'selected.go':'d'*64})
    provenance = dict(schema='gomad.rust-wasm-provenance/v1', inputs=inputs, build_key=observations.digest(observations.canonical(inputs)),
                      module_sha256=observations.digest(module))
    record = dict(identity=dict(implementation=observations.digest(observations.canonical(config['identity'])),target=observations.digest(module)))
    artifact = campaign/'prepared'
    put(artifact/'record.json',record)
    (artifact/'target').mkdir(mode=0o700)
    (artifact/'target/module.wasm').write_bytes(module)
    put(artifact/'target/provenance.json',provenance)
    payloads = {}
    for name in ('target/module.wasm','target/provenance.json'):
        raw = (artifact/name).read_bytes()
        payloads[name] = dict(bytes=len(raw),sha256=observations.digest(raw),module=name.endswith('.wasm'))
    put(artifact/'manifest.json',dict(schema='gomad.artifact-manifest/v2', record=record,
                                    record_sha256=observations.digest(observations.canonical(record)),files=payloads))
    put(campaign/'plan.json',dict(schema='gomad.seed-campaign-plan/v2',prepared_artifact='prepared'))
    sidecar = campaign/'preparation-observations'
    sidecar.mkdir(mode=0o700)
    files = {}
    for phase in ('compile','admission'):
        prefix = observations.PHASE_PREFIX+phase+'/'
        blob = b'bounded raw output\n'
        name = observations.digest(blob)+'.bin'
        path = sidecar/(prefix+name)
        path.parent.mkdir(parents=True,mode=0o700)
        path.write_bytes(blob); path.chmod(0o600)
        metadata = dict(schema='gomad.native-preparation-observations/v1',phase=phase,
                        commands=[dict(owner_pid=owner,pid=owner+1,name='bounded fixture')],snapshots={},blobs={})
        put(sidecar/(prefix+'metadata.json'),metadata)
        for filename in (name,'metadata.json'):
            raw=(sidecar/(prefix+filename)).read_bytes()
            files[prefix+filename]=dict(bytes=len(raw),sha256=observations.digest(raw))
    expected = observations.commitments(config,campaign)
    put(sidecar/'receipt.json',dict(schema='gomad.native-preparation-observation-receipt/v1',owner_pid=owner,
                                    host_sha256=config['identity']['host'],helper_sha256=config['identity']['helper'],**expected,files=files))
    return expected


PROGRAM = '''#!{python}
import json,os,sys
from pathlib import Path
sys.path.insert(0,{tests!r})
from test_native_observations import write_campaign,put
import native_observations as observations
config=json.loads(Path({config!r}).read_bytes())
args=sys.argv[2:]
def arg(name):return args[args.index(name)+1]
workspace=Path(arg('--artifacts'))
campaign=workspace/'campaigns/campaign-fixture'
expected=write_campaign(campaign,config,os.getpid())
plan=Path(arg('--output'))
plan_sha=put(plan,json.loads((campaign/'plan.json').read_bytes()))
receipt=plan.with_name(plan.name+'.preparation.json')
body=dict(schema='gomad.rust-preparation-receipt/v1',process_id=os.getpid(),campaign=str(campaign),plan=str(plan),plan_sha256=plan_sha,
          module_sha256=expected['module_sha256'],record_sha256=expected['prepared_record_sha256'],provenance_sha256=expected['provenance_sha256'])
receipt_sha=put(receipt,body)
print(json.dumps(dict(schema='gomad.plan-result/v1',campaign=str(campaign),plan=str(plan),plan_sha256=plan_sha,
                     module_sha256=expected['module_sha256'],preparation_receipt=str(receipt),preparation_receipt_sha256=receipt_sha)))
'''


class NativeObservationSeal(unittest.TestCase):
    def setUp(self):
        temporary=tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root=Path(temporary.name).resolve()
        self.binary=self.root/'native-fixture'
        self.config_path=self.root/'config.json'
        self.binary.write_text(PROGRAM.format(python=sys.executable,tests=str(HERE/'tests'),config=str(self.config_path)))
        self.binary.chmod(0o700)
        source=self.root/'source.sha256';source.write_text('bounded source fixture, not candidate evidence')
        self.config=dict(root=str(self.root),repo=str(self.root),binary=str(self.binary),binary_sha256=driver.sha(self.binary),
                         source_manifest=str(source),source_sha256=driver.sha(source),identity=dict(host='a'*64,helper='b'*64),
                         files={str(self.binary):driver.sha(self.binary),str(source):driver.sha(source)})
        self.config_path.write_text(json.dumps(self.config))

    def run_native(self,name,callback=True,producer=False):
        workspace=self.root/name
        workspace.mkdir()
        argv=[str(self.binary),'plan','--json','--artifacts',str(workspace),'--output',str(self.root/(name+'.plan.json'))]
        if producer:
            sys.path.insert(0,str(HERE.parent/'workload_tools'))
            import producer as owner
            result=owner.run(self.root,name,argv,0,private_observations=self.config if callback else None)
        else:
            result=driver.command(self.config,self.root,name,argv,timeout=5,private_observations=self.config if callback else None)
        receipt=driver.read(self.root/(name+'.command.json'))
        return result,receipt,workspace/'campaigns/campaign-fixture'

    def test_actual_child_snapshot_is_bound_before_both_witness_types_return(self):
        for name,producer in (('generic',False),('producer',True)):
            _,receipt,campaign=self.run_native(name,producer=producer)
            seal=receipt['private_observations']
            self.assertEqual(seal['campaigns'][0]['path'],str(campaign))
            native=json.loads((campaign/'preparation-observations/receipt.json').read_bytes())
            self.assertEqual(native['owner_pid'],receipt['pid'])
            observations.verify_snapshot(self.config,receipt['pid'],seal)

    def test_default_none_does_not_change_unobserved_receipt(self):
        _,receipt,_=self.run_native('default',callback=False)
        self.assertNotIn('private_observations',receipt)

    def test_same_record_has_different_private_observation_receipts(self):
        _,first,a=self.run_native('first')
        _,second,b=self.run_native('second')
        self.assertEqual((a/'prepared/record.json').read_bytes(),(b/'prepared/record.json').read_bytes())
        self.assertEqual((a/'prepared/manifest.json').read_bytes(),(b/'prepared/manifest.json').read_bytes())
        self.assertNotEqual(first['private_observations']['campaigns'][0]['receipt_sha256'],second['private_observations']['campaigns'][0]['receipt_sha256'])

    def test_raw_mutation_and_self_resealed_native_receipt_refuse_old_outer_seal(self):
        _,receipt,campaign=self.run_native('tamper')
        seal=receipt['private_observations']; context=seal['campaigns'][0]
        path=Path(next(name for name in context['files'] if name.endswith('.bin')))
        path.write_bytes(b'changed output')
        with self.assertRaises(ValueError):observations.verify_snapshot(self.config,receipt['pid'],seal)
        native_path=campaign/'preparation-observations/receipt.json'
        native=json.loads(native_path.read_bytes())
        relative=str(path.relative_to(native_path.parent))
        native['files'][relative]=dict(bytes=path.stat().st_size,sha256=driver.sha(path))
        put(native_path,native)
        with self.assertRaises(ValueError):observations.verify_snapshot(self.config,receipt['pid'],seal)

    def test_foreign_owner_missing_extra_and_changed_admission_owner_refuse(self):
        _,receipt,campaign=self.run_native('scope')
        native_path=campaign/'preparation-observations/receipt.json'
        original=json.loads(native_path.read_bytes())
        native=dict(original,owner_pid=receipt['pid']+100)
        put(native_path,native)
        with self.assertRaises(ValueError):observations.capture(self.config,receipt['pid'],[str(campaign)])
        put(native_path,original)
        extra=native_path.parent/'unlisted';extra.write_bytes(b'foreign')
        with self.assertRaises(ValueError):observations.capture(self.config,receipt['pid'],[str(campaign)])
        extra.unlink()
        extra.mkdir()
        with self.assertRaises(ValueError):observations.capture(self.config,receipt['pid'],[str(campaign)])
        extra.rmdir()
        metadata_path=native_path.parent/(observations.PHASE_PREFIX+'admission/metadata.json')
        metadata=json.loads(metadata_path.read_bytes());metadata['commands'][0]['owner_pid']+=100
        put(metadata_path,metadata)
        relative=str(metadata_path.relative_to(native_path.parent))
        original['files'][relative]=dict(bytes=metadata_path.stat().st_size,sha256=driver.sha(metadata_path))
        put(native_path,original)
        with self.assertRaises(ValueError):observations.capture(self.config,receipt['pid'],[str(campaign)])

    def test_explicit_selector_ignores_unrelated_campaign_and_rejects_pointer_escape(self):
        raw,receipt,campaign=self.run_native('selector')
        unrelated=self.root/'unrelated';unrelated.mkdir();(unrelated/'receipt.json').write_text('not a native receipt')
        response=json.loads(raw)
        argv=receipt['argv']
        self.assertEqual(len(observations.from_command(self.config,receipt['pid'],argv,raw)['campaigns']),1)
        response['campaign']=str(unrelated)
        with self.assertRaises(ValueError):observations.from_command(self.config,receipt['pid'],argv,json.dumps(response).encode())

    def test_capture_failure_retains_actual_exit_and_wait4_resources(self):
        Path(self.config['source_manifest']).write_text('changed after context freeze')
        for name, producer in (('refused-generic', False), ('refused-producer', True)):
            with self.assertRaisesRegex(ValueError, 'source changed/unsealed'):
                self.run_native(name, producer=producer)
            receipt = driver.read(self.root/(name+'.command.json'))
            self.assertEqual(receipt['exit'], 0)
            self.assertNotIn('private_observations', receipt)
            self.assertTrue((self.root/(name+'.stdout')).read_bytes())
            if not producer:
                self.assertEqual(receipt['observation'], 'inconclusive')
                self.assertEqual(receipt['resources']['waited_pid'], receipt['pid'])
            else:
                self.assertIn('private_observations_error', receipt)

    def test_qualify_selector_requires_complete_distinct_ordered_campaign_set(self):
        _, receipt, first = self.run_native('qualify-context')
        workspace = first.parent.parent
        second = workspace/'campaigns/campaign-second'
        write_campaign(second, self.config, receipt['pid'])
        specification = self.root/'spec.json'
        put(specification, {'ArtifactRoot': str(workspace)})
        result = dict(schema='gomad.rust-qualification-result/v1',
                      manifest=dict(integrated_source_sha256=self.config['source_sha256'], repeat=2),
                      observations=dict(operator_process_id=receipt['pid']),
                      report=dict(repeat=2, failure=None, executions=[
                          dict(iteration=1, campaign_path=str(first)), dict(iteration=2, campaign_path=str(second))]))
        result_path = workspace/'result.json'
        argv = [self.config['binary'], 'qualify', '--spec', str(specification)]
        output = json.dumps(dict(result_path=str(result_path))).encode()
        put(result_path, result)
        seal = observations.from_command(self.config, receipt['pid'], argv, output)
        self.assertEqual([row['path'] for row in seal['campaigns']], [str(first), str(second)])
        for malformed in (
            [dict(iteration=1, campaign_path=str(first))],
            [dict(iteration=1, campaign_path=str(first)), dict(iteration=2, campaign_path=str(first))],
            [dict(iteration=2, campaign_path=str(first)), dict(iteration=1, campaign_path=str(second))],
        ):
            result['report']['executions'] = malformed
            put(result_path, result)
            with self.assertRaises(ValueError):
                observations.from_command(self.config, receipt['pid'], argv, output)

    def test_explore_selector_binds_actual_result_and_explicit_workspace(self):
        _, receipt, campaign = self.run_native('explore-context')
        argv = [self.config['binary'], 'explore', '--artifacts', str(campaign.parent.parent)]
        output = json.dumps(dict(type='result', campaign_path=str(campaign))).encode()
        self.assertEqual(observations.from_command(self.config, receipt['pid'], argv, output)['campaigns'][0]['path'], str(campaign))
        for response in (dict(type='progress', campaign_path=str(campaign)),
                         dict(type='result', campaign_path=str(self.root))):
            with self.assertRaises(ValueError):
                observations.from_command(self.config, receipt['pid'], argv, json.dumps(response).encode())

    def test_changed_candidate_symlink_and_missing_payload_refuse(self):
        _, receipt, campaign = self.run_native('files')
        context = receipt['private_observations']['campaigns'][0]
        payload = Path(next(name for name in context['files'] if name.endswith('.bin')))
        raw = payload.read_bytes()
        payload.unlink()
        with self.assertRaises((ValueError, OSError)):
            observations.capture(self.config, receipt['pid'], [str(campaign)])
        elsewhere = self.root/'external-blob'
        elsewhere.write_bytes(raw)
        payload.symlink_to(elsewhere)
        with self.assertRaises(ValueError):
            observations.capture(self.config, receipt['pid'], [str(campaign)])
        payload.unlink(); payload.write_bytes(raw); payload.chmod(0o600)
        self.binary.write_text(self.binary.read_text()+'\n# changed executable\n')
        with self.assertRaisesRegex(ValueError, 'binary changed/unsealed'):
            observations.capture(self.config, receipt['pid'], [str(campaign)])

    def test_driver_opt_in_is_explicit_boolean_and_context_cannot_be_replaced(self):
        import adapters
        self.assertEqual(adapters.private_capture(self.config, {}), {})
        self.assertIs(adapters.private_capture(self.config, {'private_observations': True})['private_observations'], self.config)
        for value in ('true', 1, None):
            with self.assertRaises(ValueError):
                adapters.private_capture(self.config, {'private_observations': value})
        with self.assertRaisesRegex(ValueError, 'foreign private observation context'):
            driver.command(self.config, self.root, 'foreign-context', [sys.executable, '-c', 'pass'], private_observations={})
        self.assertFalse((self.root/'foreign-context.process.json').exists())
