import copy
import hashlib
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import driver
import receiving_cost

PARITY = Path(__file__).resolve().parents[3]/'testdata/parity'


def report(case):
    executions=[]
    for index,run in enumerate(case['runs'],1):
        record=dict(outcome=dict({'class':run['class']}),seed=7,virtual_time_nanos=run['virtual_time_nanos'],
                    limits=dict(memory_bytes=256<<20))
        encoded=json.dumps(record,sort_keys=True,separators=(',',':')).encode()
        executions.append(dict(iteration=index,record=record,record_sha256=hashlib.sha256(encoded).hexdigest(),
                               replay=dict(matched=run['replayed'])))
    return dict(executions=executions)


class RawAccountingContract(unittest.TestCase):
    def test_all_53_native_accounting_corpus_cases_match_every_golden_cost_and_budget_field(self):
        cases=json.loads((PARITY/'qualify-accounting-corpus.json').read_bytes())['cases']
        goldens={row['name']:row for row in json.loads((PARITY/'goldens-qualify/accounting.json').read_bytes())['cases']}
        self.assertEqual(len(cases),53)
        for case in cases:
            expected=goldens[case['name']]
            for field,derive in [('cost',lambda:receiving_cost.cost(case['observations'],report(case))),
                                 ('budget',lambda:receiving_cost.budget(case['observations']))]:
                with self.subTest(case=case['name'],field=field):
                    if expected[field] is None:
                        with self.assertRaises((driver.Refusal,KeyError,TypeError)): derive()
                    else:
                        self.assertEqual(derive(),expected[field])

    def test_final_result_rejects_omission_cpu_double_count_rss_forgery_and_counter_overflow(self):
        cases=json.loads((PARITY/'qualify-accounting-corpus.json').read_bytes())['cases']
        case=next(row for row in cases if row['name']=='phases-all-within')
        result=dict(observations=copy.deepcopy(case['observations']),report=report(case),cost_failure=None,budget_failure=None)
        result['cost']=receiving_cost.cost(result['observations'],result['report'])
        result['budget']=receiving_cost.budget(result['observations'])
        receiving_cost.validate(result)
        mutations=[lambda row:row['observations'].__setitem__('start',None),
                   lambda row:row['observations']['samples'].pop(),
                   lambda row:row['observations']['samples'].append(copy.deepcopy(row['observations']['samples'][-1])),
                   lambda row:row['cost'].__setitem__('cpu_nanos',row['cost']['cpu_nanos']+1),
                   lambda row:row['budget']['checks'][0].__setitem__('limit',1<<63),
                   lambda row:row['observations']['samples'][0].__setitem__('store_peak_rss_bytes',1<<64),
                   lambda row:row['observations']['executed']['children'].__setitem__('user_nanos',(1<<64)-1)]
        for index,mutate in enumerate(mutations):
            changed=copy.deepcopy(result);mutate(changed)
            with self.subTest(mutation=index),self.assertRaises(driver.Refusal): receiving_cost.validate(changed)
