"""Pure native optional/vector presence and exact raw authentication controls."""
import base64
import copy
import json
from pathlib import Path
import sys
import unittest
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import core_presence as gate
from driver import Refusal


def b64(raw):return base64.b64encode(raw).decode()


def baseline():
    handles=[dict(node=node,incarnation=incarnation) for node,incarnation in [('a',1),('b',1),('a',2)]]
    return dict(schema='gomad.cluster-replay-inputs/v1',helper_sha256='a'*64,host_sha256='b'*64,runtime_implementation_sha256='c'*64,toolchain_sha256=gate.digest(b'gomad.fixture-wat-source/v1'),
                configuration=dict(incarnations=[dict(handle=handle,module=dict(file='target/module.wasm',sha256=gate.MODULE_SHA256),runtime_sha256=gate.digest(b'inspect-runtime-implementation/v1')) for handle in handles],crash_groups=[]),retained=[dict(handle=handle) for handle in handles])


def row(variant,spelling):
    value=gate.shape(baseline(),variant)
    if spelling=='pretty':raw=json.dumps(value,indent=2).encode()
    elif spelling=='order':raw=b'{'+b','.join(gate.canonical(key)+b':'+gate.canonical(value[key]) for key in sorted(value,reverse=True))+b'}'
    else:
        raw=gate.canonical(value)
        if spelling=='escaped':raw=raw.replace(b'"schema":',b'"\\u0073chema":',1).replace(b'gomad.cluster-replay-inputs',b'\\u0067omad.cluster-replay-inputs',1)
    descriptor=dict(file=gate.FILE,bytes=len(raw),sha256=gate.digest(raw),limit=16<<20)
    manifest=dict(schema='gomad.artifact-manifest/v2',files={gate.FILE:dict(bytes=len(raw),sha256=gate.digest(raw),module=False)},record=dict(schema='gomad.execution-record/v2',streams={gate.FILE:dict(bytes=len(raw),sha256=gate.digest(raw),records=0)}))
    mutations=[]
    for variant_auth in ('digest','length','path','raw-append'):
        changed=dict(descriptor);item=dict(variant=variant_auth,descriptor=changed,error='native refused raw mismatch')
        if variant_auth=='digest':changed['sha256']=gate.digest(b'foreign bytes')
        elif variant_auth=='length':changed['bytes']+=1
        elif variant_auth=='path':changed['file']='evidence/foreign.json'
        else:item['raw_base64']=b64(raw+b'\n')
        mutations.append(item)
    refused=variant in gate.DECODER_REFUSALS
    observed=dict(canonical_base64=None if refused else b64(gate.canonical(gate.native_canonical(value))),decoder_error='native invalid vector' if refused else None,
                  admission_error='invalid typed cluster oracle plan' if variant=='optional-some-empty-rejected' else None,authentication=mutations)
    return dict(schema=gate.SCHEMA,original_root=gate.ROOT,variant=variant,spelling=spelling,inputs=dict(baseline=baseline(),raw_base64=b64(raw),manifest=manifest),observed=observed)


class Presence(unittest.TestCase):
    def test_complete_modern_matrix_stays_partial_for_retired_representations(self):
        rows=[row(variant,spelling) for variant in sorted(gate.VARIANTS) for spelling in sorted(gate.SPELLINGS)]
        facts=gate.observations({},'\n'.join(gate.MARKER+json.dumps(item) for item in rows))[gate.ROOT][0]
        self.assertEqual(len(facts['cases']),40)
        self.assertTrue(facts['modern_received'])
        self.assertTrue(facts['remaining_original_predicates'])
        self.assertEqual(len(facts['dispositions']),4)
        for values in (rows[:-1],rows+[rows[0]]):
            with self.assertRaises(Refusal):gate.observations({},'\n'.join(gate.MARKER+json.dumps(item) for item in values))

    def test_some_empty_plan_is_present_and_semantically_rejected(self):
        some=row('optional-some-empty-rejected','canonical')
        gate.validate(some)
        canonical=gate.decode(gate.b64(some['observed']['canonical_base64']))
        self.assertIsNotNone(canonical['configuration']['oracle_plan'])
        changed=copy.deepcopy(some);changed['observed']['admission_error']=None
        with self.assertRaises(Refusal):gate.validate(changed)
        none=row('optional-none-null','canonical')
        canonical=gate.decode(gate.b64(none['observed']['canonical_base64']))
        self.assertNotIn('oracle_plan',canonical['configuration'])
        some['observed']['canonical_base64']=none['observed']['canonical_base64']
        with self.assertRaises(Refusal):gate.validate(some)

    def test_all_four_authentication_mutations_require_actual_raw_preimages(self):
        for mutation in ('hash','length','path','append','missing','duplicate'):
            changed=row('default-vector-empty','pretty')
            if mutation=='hash':changed['observed']['authentication'][0]['descriptor']['sha256']='0'*64
            elif mutation=='length':changed['inputs']['manifest']['files'][gate.FILE]['bytes']+=1
            elif mutation=='path':changed['observed']['authentication'][2]['descriptor']['file']=gate.FILE
            elif mutation=='append':changed['observed']['authentication'][3]['raw_base64']=changed['inputs']['raw_base64']
            elif mutation=='missing':changed['observed']['authentication'].pop()
            else:changed['observed']['authentication'][2]=changed['observed']['authentication'][1]
            with self.subTest(mutation=mutation),self.assertRaises(Refusal):gate.validate(changed)

    def test_required_and_default_vectors_cannot_alias_null_to_empty(self):
        for variant in gate.DECODER_REFUSALS:
            changed=row(variant,'escaped')
            gate.validate(changed)
            changed['observed'].update(decoder_error=None,canonical_base64=b64(gate.canonical(gate.shape(baseline(),'required-vector-empty'))))
            with self.assertRaises(Refusal):gate.validate(changed)

    def test_legacy_field_aliases_and_pass_aliases_do_not_receive_presence(self):
        for field in ('Record','Boots','OutputStops','Oracles'):
            changed=row('optional-none-missing','canonical')
            value=gate.decode(gate.b64(changed['inputs']['raw_base64']));value[field]=[]
            changed['inputs']['raw_base64']=b64(gate.canonical(value))
            with self.assertRaises(Refusal):gate.validate(changed)
        changed=row('optional-none-missing','canonical');changed['observed']=dict(passed=True)
        with self.assertRaises(Refusal):gate.validate(changed)


if __name__=='__main__':unittest.main()
