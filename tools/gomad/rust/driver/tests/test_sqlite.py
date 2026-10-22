import base64
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


class SQLiteReceiverContract(unittest.TestCase):
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

    def test_selected_spec_carries_full_original_state_and_actual_primary_hash(self):
        import sqlite
        source = {'Manifest': {'ID': 'sqlite-durable-recovery', 'Seed': 7, 'Repeat': 2},
                  'Campaign': {'Target': {'Scenario': base64.b64encode(json.dumps({'workload': {
                      'kind': 'sqlite-durable-recovery', 'selection': None}}).encode()).decode()}}}
        state = dict(Identity='sha256:'+'1'*64, Volume='sqlite', PendingSHA256='sha256:'+'a'*64,
                     SelectedOperations=[3], Entries=[{'Path': '/db', 'Data': 'AA=='}])
        context = {'PendingSHA256': state['PendingSHA256'], 'Operations': [{'ID': 3}]}
        selected = sqlite.selection_spec(source, 'b'*64, state, context, '/fresh')
        scenario = json.loads(base64.b64decode(selected['Campaign']['Target']['Scenario']))
        self.assertEqual(scenario['workload']['selection'],
                         dict(primary_result_sha256='b'*64, state=state, context=context))
        self.assertEqual(selected['Manifest']['ID'], 'sqlite-durable-selection-'+'1'*64)
        self.assertEqual(source['Manifest']['ID'], 'sqlite-durable-recovery')


if __name__ == '__main__':
    unittest.main()
