"""Raw writer counterexamples; no native test process or original-root credit."""
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_predicates as gate
from driver import Refusal


def fixture():
    payloads = {'evidence/binary-writer-tail': bytes(range(256))+b'\0', 'output/stdout': b'foundation-output', 'target/module.wasm': b'(module)'}
    record = dict(arguments=['quoted="tail"', 'Unicode=☃'], environment={'EMPTY':''}, virtual_time_nanos=(1<<64)-8,
                  streams={name:dict(bytes=len(payloads[name]),records=count,sha256=gate.digest(payloads[name])) for name,count in
                           (('evidence/binary-writer-tail',1),('output/stdout',0))})
    payloads['record.json']=gate.canonical(record)
    record_sha=gate.digest(payloads['record.json'])
    manifest=gate.canonical(dict(schema='gomad.artifact-manifest/v2',record=record,record_sha256=record_sha,
        files={name:dict(bytes=len(raw),sha256=gate.digest(raw),module=name=='target/module.wasm') for name,raw in payloads.items()}))
    total=sum(map(len,payloads.values()))+len(manifest)+len(gate.LAYOUT)+len(gate.ENVELOPE)
    rows=[]
    for mode in ('zero','error'):
        for budget in (0,1,19,100,1024,total-2,total-1):
            rows.append(dict(mode=mode,budget=budget,written=budget,write_calls=budget//3+10,failures=1,
                error='failed to write whole buffer' if mode=='zero' else 'injected retained writer error',committed_before_retry=0,
                retry_record_sha256=record_sha,retry_record_bytes=list(payloads['record.json']),retry_manifest_bytes=list(manifest)))
    return dict(schema=gate.WRITER_SCHEMA,encoding=gate.WRITER_ENCODING,record_sha256=record_sha,payloads={name:list(raw) for name,raw in payloads.items()},
                manifest_bytes=list(manifest),layout_bytes=list(gate.LAYOUT),envelope_bytes=list(gate.ENVELOPE),total_written=total,write_calls=total//3+10,cases=rows)


class WriterPreimages(unittest.TestCase):
    def test_all_raw_preimages_and_original_failure_offsets(self):
        value=fixture();result=gate.retained_writer(value)
        self.assertEqual(result['total_written'],value['total_written'])
        self.assertEqual(len(result['cases']),14)
        self.assertEqual(result['cases'][-1]['budget'],value['total_written']-1)

    def test_hash_only_payload_claims_and_resealed_mutations_refuse(self):
        for mutation in ('missing-raw','payload','manifest','retry-record','retry-manifest','total','bool-byte','baseline-count'):
            with self.subTest(mutation=mutation):
                value=fixture()
                if mutation=='missing-raw':value['payloads']['record.json']={'sha256':value['record_sha256']}
                if mutation=='payload':value['payloads']['output/stdout'][0]^=1
                if mutation=='manifest':value['manifest_bytes'][-1]^=1
                if mutation=='retry-record':value['cases'][-1]['retry_record_bytes'][-1]^=1
                if mutation=='retry-manifest':value['cases'][-1]['retry_manifest_bytes'][-1]^=1
                if mutation=='total':value['total_written']+=1
                if mutation=='bool-byte':value['layout_bytes'][0]=True
                if mutation=='baseline-count':value['write_calls']=33
                with self.assertRaises(Refusal):gate.retained_writer(value)

    def test_original_final_byte_offsets_modes_and_atomic_retry_are_required(self):
        for field,bad in (('budget',1023),('written',0),('failures',2),('committed_before_retry',1),('error','success'),('mode','zero')):
            with self.subTest(field=field):
                value=fixture();value['cases'][-1][field]=bad
                with self.assertRaises(Refusal):gate.retained_writer(value)
        value=fixture();value['cases'].pop()
        with self.assertRaises(Refusal):gate.retained_writer(value)
        value=fixture();value['cases'][2]['budget']=2;value['cases'][2]['written']=2
        with self.assertRaises(Refusal):gate.retained_writer(value)


if __name__=='__main__':unittest.main()
