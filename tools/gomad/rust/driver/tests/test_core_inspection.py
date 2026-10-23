"""Pure native inspection decoder and independently recomputed search controls."""
import base64
import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_inspection as gate
from driver import Refusal, sha

REPO=Path(__file__).resolve().parents[5]
FIXTURE=REPO/'tools/gomad/rust/wasmhost/tests/testdata/inspection/source-five-decisions.json'


def fixture():
    return json.loads(FIXTURE.read_bytes())


def config():
    return dict(repo=str(REPO),files={str(FIXTURE):sha(FIXTURE)})


def cleanup():
    return dict(status='incomplete',evaluations=[dict(choice=0,outcome='timeout',prefix_verified=True,detail='context deadline exceeded')],
                remaining=dict(first=0,last=4),conditional_on_monotonicity=False,
                raw_input=dict(search=dict(check='healthy',first=0,last=4,mode='scan',attempt_budget=10,wall_time=1000000000),
                               clock_nanos=['1000000','1000000000'],calls=[dict(choice=0,remaining_nanos='999000000',outcome='pass',prefix_verified=True,detail='pass')]))


def row(variant='cleanup-exhausted-wall'):
    observed=cleanup()
    if variant=='already-expired':
        observed.update(evaluations=None)
        observed['raw_input'].update(clock_nanos=['1000000000'],calls=[])
    if variant=='external-cancel-joined-store':
        observed['raw_input'].update(clock_nanos=['1000','2000'],cancelled=[False,True])
        observed['raw_input']['calls'][0]['remaining_nanos']='999999000'
        observed['cleanup']=dict(worker_termination='cancelled',store_joined_nanos='1500',returned_nanos='2500')
    return dict(schema=gate.SCHEMA,original_root='TestInspectionSearchWallBoundIncludesEvaluatorCleanup',variant=variant,observed=observed)


class InspectionReader(unittest.TestCase):
    def test_complete_independent_native_fixture_and_all_runtime_witnesses(self):
        evidence=fixture()
        boundaries,events=gate.decode_evidence(evidence)
        self.assertEqual(gate.mapping(evidence,boundaries),list(range(5)))
        self.assertEqual(len(events),7)
        for index,boundary in enumerate(boundaries):
            self.assertEqual(boundary['decision'],index)
            self.assertEqual(boundary['cuts'][0]['witness']['choices'],index)
            self.assertEqual(boundary['cuts'][0]['witness']['goroutines'],2)
        gate.material(dict(evidence=evidence,boundaries=boundaries))

    def test_raw_boundary_truncation_trailing_bytes_and_declared_count_refuse(self):
        raw=gate.binary(fixture()['boundary_frames'])
        for changed in (raw[:8],raw[:-1],raw+b'\0',raw[:8]+(6).to_bytes(4,'little')+raw[12:]):
            evidence=fixture();evidence['boundary_frames']=base64.b64encode(changed).decode()
            with self.assertRaises(Refusal):
                gate.decode_evidence(evidence)

    def test_raw_choice_and_cut_mutation_cannot_be_a_boolean_refusal_alias(self):
        evidence=fixture();boundaries,_=gate.decode_evidence(evidence)
        with self.assertRaises(Refusal):
            gate.refused_mapping(dict(evidence=evidence,boundaries=boundaries))
        for kind in ('coordinate','bytes','missing'):
            changed=copy.deepcopy(boundaries)
            if kind=='coordinate':changed[2]['trace']=1
            elif kind=='bytes':changed[2]['cuts'][0]['choice_bytes']-=1
            else:changed.pop()
            gate.refused_mapping(dict(evidence=evidence,boundaries=changed))
        evidence['incarnations'][0]['streams']['choice']=base64.b64encode(bytes(5*96)).decode()
        with self.assertRaises(Refusal):
            gate.mapping(evidence,boundaries)

    def test_actual_cleanup_remaining_budget_is_nanoseconds(self):
        report=gate.search_report(cleanup())
        self.assertEqual(report['status'],'incomplete')
        self.assertNotIn('first_failure',report)
        for mutation in ('remaining','coordinate','clock','outcome','duplicate','forged-success'):
            observed=cleanup();inputs=observed['raw_input']
            if mutation=='remaining':inputs['calls'][0]['remaining_nanos']='999'
            elif mutation=='coordinate':inputs['calls'][0]['choice']=1
            elif mutation=='clock':inputs['clock_nanos'][1]='999999999'
            elif mutation=='outcome':observed['evaluations'][0]['outcome']='pass'
            elif mutation=='duplicate':inputs['calls']*=2
            else:observed.update(status='first_failure',first_failure=0)
            with self.subTest(mutation=mutation),self.assertRaises(Refusal):
                gate.search_report(observed)

    def test_missing_duplicate_forged_raw_rows_and_frozen_fixture_refuse(self):
        rows=[row(),row('already-expired'),row('external-cancel-joined-store')]
        stdout='\n'.join(json.dumps(value) for value in rows)
        result=gate.observations(config(),stdout)
        self.assertEqual(len(result[rows[0]['original_root']]),3)
        for values in ([rows[0]],rows+[rows[0]], [dict(rows[0],observed=dict(passed=True)),rows[1]]):
            with self.assertRaises((Refusal,KeyError)):
                gate.observations(config(),'\n'.join(json.dumps(value) for value in values))
        with self.assertRaises(Refusal):
            gate.observations(dict(config(),files={}),stdout)
        duplicate=json.dumps(rows[0]).replace('"variant":', '"variant":"forged","variant":',1)
        with self.assertRaises(Refusal):
            gate.observations(config(),duplicate)

    def test_external_cancellation_cannot_return_before_join_or_omit_cancel_sample(self):
        root=row('external-cancel-joined-store')['original_root']
        observed=row('external-cancel-joined-store')['observed']
        gate.validate_case(root,'external-cancel-joined-store',observed)
        for mutation in ('unjoined','uncancelled','missing-call','duplicate-clock'):
            changed=copy.deepcopy(observed)
            if mutation=='unjoined':changed['cleanup']['store_joined_nanos']='3000'
            elif mutation=='uncancelled':changed['raw_input']['cancelled'][1]=False
            elif mutation=='missing-call':changed['raw_input']['calls']=[]
            else:changed['raw_input']['clock_nanos'].append('2500')
            with self.subTest(mutation=mutation),self.assertRaises(Refusal):
                gate.validate_case(root,'external-cancel-joined-store',changed)

    def test_all_twelve_original_roots_and_original_source_owners_are_pinned(self):
        self.assertEqual(len(gate.ROOT_CASES),12)
        self.assertEqual(len(gate.source_inputs(config())),4)


if __name__=='__main__':
    unittest.main()
