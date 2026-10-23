"""Raw oracle mutation tests only; synthetic fixtures earn no qualification gate."""
import base64
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import cell_reader
from driver import Refusal


class CellOracles(unittest.TestCase):
    def test_actual_target_and_scenario_are_bound_to_original_request(self):
        import configure
        import hashlib
        with tempfile.TemporaryDirectory() as temporary:
            manifest = Path(temporary) / 'source.sha256'
            manifest.write_text('synthetic sealed source')
            label = 'workload-multi-instance-cooperative-success'
            specification = configure.cell_spec(label, '/attempt/cells/' + label, str(manifest),
                hashlib.sha256(manifest.read_bytes()).hexdigest(), bindings=dict(root='/attempt'))
            target = specification['Campaign']['Target']
            scenario = cell_reader.normalized_scenario(target)
            raw = json.dumps(scenario).encode()
            installation = dict(compiler_path='/compiler', runtime_implementation_sha256='runtime', build_tags=['installed'])
            inputs = dict(kind=target['Kind'], source=target['Source'], directory=target['WorkingDir'], arguments=[],
                          tags=sorted(set(target['BuildTags'] + ['installed'])), compiler_path='/compiler',
                          runtime_implementation='runtime', scenario=list(raw))
            result = dict(provenance=dict(inputs=inputs, scenario=list(raw)),
                          manifest=dict(scenario_sha256=hashlib.sha256(raw).hexdigest()))
            cell_reader.requested_target(result, specification, installation)
            for field, value in (('kind', 'go-run'), ('source', './foreign'), ('directory', '/foreign'),
                                 ('arguments', ['foreign']), ('tags', []), ('compiler_path', '/foreign')):
                changed = copy.deepcopy(result)
                changed['provenance']['inputs'][field] = value
                with self.subTest(field=field), self.assertRaises(Refusal):
                    cell_reader.requested_target(changed, specification, installation)
            for field in ('network_faults', 'actions'):
                changed = copy.deepcopy(result)
                altered = copy.deepcopy(scenario)
                altered[field] = []
                raw = json.dumps(altered).encode()
                changed['provenance']['scenario'] = list(raw)
                changed['provenance']['inputs']['scenario'] = list(raw)
                changed['manifest']['scenario_sha256'] = hashlib.sha256(raw).hexdigest()
                with self.subTest(field=field), self.assertRaises(Refusal):
                    cell_reader.requested_target(changed, specification, installation)

    def test_replay_binds_partition_heal_grouped_crash_and_volume_model(self):
        from driver import HERE, load_module
        workload = load_module('cell_test_workloads', HERE.parent / 'workload_tools/workloads.py')
        scenario = workload.temporal_replication_scenario(False)
        target = dict(Scenario=base64.b64encode(json.dumps(scenario).encode()).decode())
        scenario = cell_reader.normalized_scenario(target)
        base = dict(arguments=['program'], namespace={}, limits=dict(evidence_bytes=100), engine='engine', profile='profile', model={}, environment={}, derive_seed_environment=False, runtime_sha256='runtime')
        network = dict(profile='synthetic-network')
        actual = dict(steps=scenario['actions'], workload=scenario['workload'], nodes=['active', 'standby'], boundaries=[],
                      crash_groups=[dict(nodes=['active', 'standby'], marker=list(b'GOMAD_TEMPORAL_OBSERVATION '))],
                      engine='engine', profile='profile',
                      limits=dict(evidence_bytes=100*sum(step['action'] in ('start', 'restart') for step in scenario['actions'])),
                      broker=dict(seed=7, pause_on_settle=False, limits=dict(max_requests=2<<20, max_events=2<<20,
                          max_queued_events=250000, max_transitions=8<<20, max_payload_bytes=64<<20, max_string_bytes=(1<<32)-1)),
                      network=dict(model=network, strict_topology=True, addresses={n['id']:n['address'] for n in scenario['spec']['nodes']},
                                   links=scenario['spec']['links'], faults=[dict(name=f['Name'], action=f['Action'],
                                   **{'from':f['From'], 'to':f['To']}, at_nanos=f['AtNanos'], delay_nanos=0, symmetric=False)
                                   for f in scenario['network_faults']]), incarnations=[])
        for step in scenario['actions']:
            if step['action'] not in ('start', 'restart'):
                continue
            handle = step['handle']
            boot = next(b for b in scenario['boots'] if b['id'] == handle['node'] + ('-commit' if handle['incarnation']==1 else '-reopen'))
            import hashlib
            role = handle['node']
            peer = '10.0.0.2' if role == 'active' else '10.0.0.1'
            environment = dict(GOMAD_TEMPORAL_CLUSTER=role, GOMAD_TEMPORAL_PHASE='commit' if handle['incarnation']==1 else 'reopen',
                GOMAD_TEMPORAL_REMOTE_FRONTEND=peer+':7233', GOMAD_TEMPORAL_REMOTE_HTTP=peer+':7243', GOMAD_TEMPORAL_REMOTE_HISTORY=peer+':7234')
            model = dict(limit=64<<20, entropy_key=list(hashlib.sha256(('gomad-wasm-temporal-replication/v1/'+role).encode()).digest()))
            actual['incarnations'].append(dict(handle=handle, arguments=['program']+boot['args'], namespace={}, limits=dict(evidence_bytes=100),
                                               model=model, environment=environment, network=network,
                                               module=dict(file='target/module.wasm', bytes=6, sha256='module', limit=512<<20),
                                               imports=[], runtime_sha256='runtime', choice_identity={},
                                               storage=dict(image=None, context=None, descriptor=dict(model=boot['storage'],
                                                   model_identity=cell_reader.storage_identity(boot['storage']), return_image=True,
                                                   crash_limits=None, crash_selection=None, crash_context=None, image=None))))
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            path = root / 'evidence/cluster-replay.json'
            path.parent.mkdir()
            (root/'target').mkdir()
            (root/'target/module.wasm').write_bytes(b'module')
            result = dict(provenance=dict(configuration=base, network=network, module_sha256='module',
                                         module_metadata=dict(imports=[]), choice_identity={}), report=dict(executions=[dict(
                          artifact_path=str(root), record=dict(identity={}))]))
            def check(value):
                path.write_text(json.dumps(dict(configuration=value)))
                result['report']['executions'][0]['record']['identity']['configuration'] = cell_reader.receiving.digest(cell_reader.receiving.canonical(value))
                cell_reader.replay_scenario(result, scenario)
            check(actual)
            for field in ('network', 'crash_groups', 'steps', 'incarnations', 'broker'):
                changed = copy.deepcopy(actual)
                if field == 'network':
                    changed[field]['faults'] = []
                else:
                    changed[field] = []
                with self.subTest(field=field), self.assertRaises(Refusal):
                    check(changed)
            for field in ('module', 'runtime_sha256', 'choice_identity', 'imports', 'wall_nanos'):
                changed = copy.deepcopy(actual)
                changed['incarnations'][0][field] = 'foreign'
                with self.subTest(field=field), self.assertRaises(Refusal):
                    check(changed)
            for field in ('image', 'context', 'return_image', 'crash_limits', 'model_identity'):
                changed = copy.deepcopy(actual)
                destination = changed['incarnations'][0]['storage'] if field in ('image', 'context') else changed['incarnations'][0]['storage']['descriptor']
                destination[field] = 'foreign'
                with self.subTest(field=field), self.assertRaises(Refusal):
                    check(changed)
            changed = copy.deepcopy(actual)
            changed['incarnations'][0]['storage']['descriptor']['model']['volumes'][0]['CapacityBytes'] = 1
            with self.assertRaises(Refusal):
                check(changed)

    def test_sqlite_canary_recomputes_actual_ack_history_and_oracle(self):
        with tempfile.TemporaryDirectory() as temporary:
            artifact = Path(temporary)
            for number, output in ((2, b'ACK 2\n'), (3, b'INTEGRITY ok\nHISTORY 1:committed\n')):
                path = artifact / f'evidence/cluster/sqlite/{number}/stdout'
                path.parent.mkdir(parents=True)
                path.write_bytes(output)
            oracle = artifact / 'evidence/workload-oracles.json'
            checks = {'acknowledged-durable-recovery.events': False,
                      'acknowledged-durable-recovery.integrity': True,
                      'acknowledged-durable-recovery.acknowledged': True}
            oracle.write_text(json.dumps(dict(schema='gomad.rust-workload-oracles/v1', checks=checks)))
            result = dict(report=dict(executions=[dict(artifact_path=str(artifact))]))
            cell_reader.sqlite_oracles(None, result, False)
            with self.assertRaises(Refusal):
                cell_reader.sqlite_oracles(None, result, True)
            checks['acknowledged-durable-recovery.events'] = True
            oracle.write_text(json.dumps(dict(schema='gomad.rust-workload-oracles/v1', checks=checks)))
            with self.assertRaises(Refusal):
                cell_reader.sqlite_oracles(None, result, False)

    def test_replication_reopens_both_histories_and_rejects_oracle_claims(self):
        with tempfile.TemporaryDirectory() as temporary:
            artifact = Path(temporary)
            observations = {}
            for role in ('active', 'standby'):
                value = dict(cluster_id=role, history=base64.b64encode(b'actual history').decode(),
                             event_ids=[1, 2, 3, 4, 5], types=['event'] * 5)
                observations[role] = value
                for number in (1, 2):
                    path = artifact / f'evidence/cluster/{role}/{number}/stdout'
                    path.parent.mkdir(parents=True)
                    path.write_bytes(b'GOMAD_TEMPORAL_OBSERVATION ' + json.dumps(value).encode() + b'\n')
            checks = {role + suffix: True for role in ('active', 'standby')
                      for suffix in ('-durable-recovery.history', '-durable-recovery.events', '-durable-recovery.identity')}
            checks['independent-replication-convergence'] = True
            oracle = artifact / 'evidence/workload-oracles.json'
            oracle.write_text(json.dumps(dict(schema='gomad.rust-workload-oracles/v1', checks=checks)))
            result = dict(report=dict(executions=[dict(artifact_path=str(artifact))]))
            cell_reader.replication_oracles(result, True)
            path = artifact / 'evidence/cluster/standby/2/stdout'
            for mutation in ('identity', 'events', 'history'):
                value = copy.deepcopy(observations['standby'])
                if mutation == 'identity':
                    value['cluster_id'] = 'foreign'
                if mutation == 'events':
                    value['event_ids'] = [1, 2, 2, 4, 5]
                if mutation == 'history':
                    value['history'] = base64.b64encode(b'changed history').decode()
                path.write_bytes(b'GOMAD_TEMPORAL_OBSERVATION ' + json.dumps(value).encode() + b'\n')
                with self.subTest(mutation=mutation), self.assertRaises(Refusal):
                    cell_reader.replication_oracles(result, True)


if __name__ == '__main__':
    unittest.main()
