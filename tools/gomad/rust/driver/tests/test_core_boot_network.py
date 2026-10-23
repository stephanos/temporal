"""Synthetic native admission observations exercise fixed bindings and refusal distinctions."""
import copy
import hashlib
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_boot_network as gate
from driver import Refusal


def fixture():
    rows = []
    for variant in gate.VARIANTS:
        caller, declared = gate.expected(variant)
        if 'mismatch' in variant:
            observed = dict(error='cluster boot network limits')
        else:
            raw = json.dumps(declared['shared'], separators=(',', ':')).encode()
            observed = dict(error='', bound=dict(shared=declared['shared'], owners=['a', 'b'], selected=sorted([dict(node=item['node'], incarnation=item['incarnation'], network=item['network']) for item in declared['selected']], key=lambda item:(item['node'],item['incarnation']))), activity=dict(launched_stores=0, remote_callback_attempts=0), shared_json_bytes=list(raw), shared_json_sha256=hashlib.sha256(raw).hexdigest())
        rows.append(dict(schema='gomad.core-predicate-observation/v1', case='boot-network.selected-model-binding', inputs=dict(original_root=gate.ROOT, variant=variant, caller_before=caller, caller_after=copy.deepcopy(caller), declared=declared), observed=observed))
    return rows


def stdout(rows):
    return '\n'.join(gate.MARKER + json.dumps(row) for row in rows)


class BootNetworkReader(unittest.TestCase):
    def test_fixed_original_counterparts_and_explicit_unused_gap(self):
        facts = gate.observations({}, stdout(fixture()))
        self.assertEqual(len(facts['received']), 11)
        self.assertEqual(len(facts['additional_controls']), 2)
        self.assertEqual(facts['unreceived'][0]['original_variant'], 'unused mismatch')
        self.assertEqual(sum(bool(row['error']) for row in facts['received']), 8)

    def test_rejects_overwritten_caller_wrong_owner_missing_restart_and_aliases(self):
        for mutation in ('caller', 'shared', 'owner', 'restart', 'refusal', 'identity', 'capacity', 'missing', 'duplicate', 'bool'):
            rows = fixture()
            with self.subTest(mutation=mutation):
                if mutation == 'caller': rows[2]['inputs']['caller_after']['selected'][0]['network']['limits']['connection_bytes'] = 123
                elif mutation == 'shared': rows[0]['observed']['bound']['shared'] = None
                elif mutation == 'owner': rows[0]['observed']['bound']['owners'] = ['a']
                elif mutation == 'restart': rows[6]['inputs']['declared']['selected'].pop()
                elif mutation == 'refusal': rows[2]['observed'] = dict(error='')
                elif mutation == 'identity': rows[0]['observed']['shared_json_sha256'] = '0' * 64
                elif mutation == 'capacity': rows[10]['observed'] = dict(error='cluster boot network limits')
                elif mutation == 'missing': rows.pop()
                elif mutation == 'duplicate': rows.append(rows[0])
                elif mutation == 'bool': rows[0]['observed']['activity']['launched_stores'] = False
                with self.assertRaises(Refusal):
                    gate.observations({}, stdout(rows))


if __name__ == '__main__':
    unittest.main()
