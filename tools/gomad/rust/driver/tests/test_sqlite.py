import base64
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


class SQLiteReceiverContract(unittest.TestCase):
    def test_native_oracle_uses_authenticated_raw_precrash_image_and_actual_model(self):
        import sqlite
        import hashlib
        import tempfile
        import copy
        configuration = dict(incarnations=[dict(handle=dict(node='sqlite', incarnation=2),
            storage=dict(descriptor=dict(model_identity='sha256:'+'c'*64)))])
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            bodies = {'evidence/cluster/sqlite/2/storage_state': b'actual-state',
                      'evidence/cluster/sqlite/2/storage': b'actual-log',
                      'evidence/cluster/sqlite/2/storage_precrash_image': b'actual-precrash-image',
                      'evidence/cluster-replay.json': json.dumps(dict(configuration=configuration)).encode()}
            files = {}
            for name, raw in bodies.items():
                path = root/name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(raw)
                files[name] = dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
            digest = hashlib.sha256(json.dumps(configuration,sort_keys=True,separators=(',', ':')).encode()).hexdigest()
            inspected = dict(artifact=dict(files=files,record=dict(identity=dict(configuration=digest))))
            native = dict(storage_state_sha256=files['evidence/cluster/sqlite/2/storage_state']['sha256'])
            request = sqlite.native_input(root, inspected, native)
            self.assertEqual(request['precrash_image_path'], str(root/'evidence/cluster/sqlite/2/storage_precrash_image'))
            self.assertEqual(request['precrash_image_sha256'], files['evidence/cluster/sqlite/2/storage_precrash_image']['sha256'])
            self.assertEqual(request['storage_log_sha256'], files['evidence/cluster/sqlite/2/storage']['sha256'])
            self.assertEqual(request['model_identity'], 'sha256:'+'c'*64)
            for mutation in ('missing-image','changed-image','missing-model','configuration'):
                wrong = copy.deepcopy(inspected)
                if mutation == 'missing-image': del wrong['artifact']['files']['evidence/cluster/sqlite/2/storage_precrash_image']
                if mutation == 'changed-image': wrong['artifact']['files']['evidence/cluster/sqlite/2/storage_precrash_image']['sha256'] = '0'*64
                if mutation == 'missing-model':
                    changed = copy.deepcopy(configuration)
                    del changed['incarnations'][0]['storage']['descriptor']['model_identity']
                    raw = json.dumps(dict(configuration=changed)).encode()
                    (root/'evidence/cluster-replay.json').write_bytes(raw)
                    wrong['artifact']['files']['evidence/cluster-replay.json'] = dict(bytes=len(raw),sha256=hashlib.sha256(raw).hexdigest())
                    wrong['artifact']['record']['identity']['configuration'] = hashlib.sha256(json.dumps(changed,sort_keys=True,separators=(',', ':')).encode()).hexdigest()
                if mutation == 'configuration': wrong['artifact']['record']['identity']['configuration'] = '0'*64
                with self.subTest(mutation=mutation), self.assertRaises((driver.Refusal, driver.Unexecuted)):
                    sqlite.native_input(root, wrong, native)
                (root/'evidence/cluster-replay.json').write_bytes(bodies['evidence/cluster-replay.json'])

    def test_selected_qualify_captures_actual_private_preparation(self):
        import adapters
        import sqlite
        from unittest.mock import patch
        class QualifyReached(Exception):
            pass
        config = dict(root='/attempt', binary='/sealed/gomad', installations={'sqlite': '/sealed/installation'},
                      stages={'workload-sqlite-cooperative-success': dict(installation='sqlite', spec='/sealed/spec')})
        primary = dict(path='/attempt/primary-result.json', sha256='a' * 64)
        native = dict(enumeration=dict(States=[{}]), context={})
        with patch.object(sqlite, 'read', side_effect=[primary, {}]), patch.object(sqlite, 'sha', return_value='a' * 64), \
             patch.object(adapters, 'result_report', return_value={}), patch.object(sqlite, 'catalogue', return_value=native), \
             patch.object(sqlite, 'selection_spec', return_value={}), patch.object(sqlite, 'write'), \
             patch.object(sqlite, 'command', side_effect=QualifyReached) as invoke:
            with self.assertRaises(QualifyReached):
                sqlite.select(config, Path('/attempt/selected'), 'sqlite-selection-0', dict(installation='sqlite'))
            self.assertIs(invoke.call_args.kwargs['private_observations'], config)

    def test_actual_catalogue_is_complete_exact_four_states_and_context_bound(self):
        import sqlite
        context = {'PendingSHA256': 'sha256:'+'a'*64, 'Operations': []}
        states = [dict(Identity='sha256:'+str(i)*64, Volume='sqlite', PendingSHA256=context['PendingSHA256'],
                       SelectedOperations=[], Entries=[]) for i in range(4)]
        catalogue = dict(handle=dict(node='sqlite', incarnation=2), volume='sqlite', storage_state_sha256='b'*64,
                         enumeration=dict(Complete=True, Capacity='', States=states), context=context)
        result = {'native_crash_catalogues': [catalogue]}
        self.assertEqual(sqlite.catalogue(result), catalogue)
        for mutation in ('missing', 'duplicate', 'incomplete', 'three', 'context'):
            wrong = json.loads(json.dumps(result))
            if mutation == 'missing': wrong = {}
            if mutation == 'duplicate': wrong['native_crash_catalogues'] *= 2
            if mutation == 'incomplete': wrong['native_crash_catalogues'][0]['enumeration']['Complete'] = False
            if mutation == 'three': wrong['native_crash_catalogues'][0]['enumeration']['States'].pop()
            if mutation == 'context': wrong['native_crash_catalogues'][0]['context']['PendingSHA256'] = 'changed'
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal): sqlite.catalogue(wrong)

    def test_selected_result_cannot_change_primary_execution_inputs(self):
        import sqlite
        import copy
        fields = ('workload_source_sha256', 'integrated_source_sha256', 'host_os', 'host_arch', 'fuel',
                  'memory_bytes', 'seed', 'repeat', 'compiler_sha256', 'helper_sha256', 'engine_sha256',
                  'runtime_sha256', 'model_sha256', 'profile', 'host_sha256', 'host_config_sha256', 'module_sha256')
        primary = dict(manifest={field: field for field in fields},
                       provenance=dict(inputs=dict(configuration={'model':'original'}, compiler='original',
                                                   scenario={'selection':None})))
        selected = copy.deepcopy(primary)
        intent = dict(kind='sqlite-durable-selection', primary_result='/primary/result.json',
                      selection=dict(primary_result_sha256='b'*64, state={'Identity':'actual'}, context={}))
        selected['provenance']['runtime_workload'] = intent
        sqlite.matched_inputs(primary, selected, intent)
        for field in ('compiler_sha256','runtime_sha256','engine_sha256','model_sha256','profile','host_config_sha256'):
            wrong = copy.deepcopy(selected)
            wrong['manifest'][field] = 'different'
            with self.subTest(field=field), self.assertRaises(driver.Refusal):
                sqlite.matched_inputs(primary, wrong, intent)
        for mutation in ('missing', 'intent', 'scenario', 'extra'):
            wrong = copy.deepcopy(selected)
            if mutation == 'missing': del wrong['provenance']['runtime_workload']
            if mutation == 'intent': wrong['provenance']['runtime_workload'] = {'kind':'unregistered'}
            if mutation == 'scenario': wrong['provenance']['inputs']['scenario'] = {'selection':'changed'}
            if mutation == 'extra': wrong['provenance']['unreviewed'] = True
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal):
                sqlite.matched_inputs(primary, wrong, intent)
        selected['provenance']['inputs']['configuration']['model'] = 'different'
        with self.assertRaises(driver.Refusal): sqlite.matched_inputs(primary, selected, intent)

    def test_selected_spec_carries_full_original_state_and_actual_primary_hash(self):
        import sqlite
        source = {'Manifest': {'ID': 'sqlite-durable-recovery', 'Seed': 7, 'Repeat': 2},
                  'Campaign': {'Target': {'Scenario': base64.b64encode(json.dumps({'workload': {
                      'kind': 'sqlite-durable-recovery', 'selection': None}}).encode()).decode()}}}
        state = dict(Identity='sha256:'+'1'*64, Volume='sqlite', PendingSHA256='sha256:'+'a'*64,
                     SelectedOperations=[3], Entries=[{'Path': '/db', 'Data': 'AA=='}])
        context = {'PendingSHA256': state['PendingSHA256'], 'Operations': [{'ID': 3}]}
        selected = sqlite.selection_spec(source, '/primary/result.json', 'b'*64, state, context, '/fresh')
        self.assertEqual(selected['Campaign']['Target']['Scenario'], source['Campaign']['Target']['Scenario'])
        self.assertEqual(selected['RuntimeWorkload'], dict(kind='sqlite-durable-selection',
                         primary_result='/primary/result.json',
                         selection=dict(primary_result_sha256='b'*64, state=state, context=context)))
        self.assertEqual(selected['Manifest']['ID'], 'sqlite-durable-recovery')
        self.assertEqual(source['Manifest']['ID'], 'sqlite-durable-recovery')
        another = dict(state, Identity='sha256:'+'2'*64, SelectedOperations=[])
        other = sqlite.selection_spec(source, '/primary/result.json', 'b'*64, another, context, '/another-fresh')
        self.assertEqual(other['Manifest'], selected['Manifest'])
        self.assertNotEqual(other['RuntimeWorkload'], selected['RuntimeWorkload'])
        self.assertNotEqual(other['ArtifactRoot'], selected['ArtifactRoot'])
        foreign = dict(source, Manifest=dict(source['Manifest'], ID='sqlite-durable-selection-'+'1'*64))
        with self.assertRaises(driver.Refusal):
            sqlite.selection_spec(foreign, '/primary/result.json', 'b'*64, state, context, '/foreign')



if __name__ == '__main__':
    unittest.main()
