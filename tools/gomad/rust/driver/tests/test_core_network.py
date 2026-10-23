"""Pure network reader checks against the retained literal fixtures, without a host build."""
import base64
import copy
import json
from pathlib import Path
import sys
import unittest
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_network as gate
from driver import Refusal

DATA = Path(__file__).resolve().parents[2] / 'wasmhost/tests/testdata'


class NetworkReader(unittest.TestCase):
    def test_recovers_exact_go_operations_and_native_records(self):
        for path in sorted((DATA / 'broker_orders').glob('*.json')):
            corpus = json.loads(path.read_bytes())
            for node, document in corpus.get('network_documents_base64', {}).items():
                with self.subTest(corpus=path.stem, node=node):
                    decoded = gate.decode_network((path.parent / (path.stem + '.' + node + '.gmnetev1')).read_bytes(), world=False)
                    expected = json.loads(base64.b64decode(document))
                    self.assertEqual(decoded['log'], b''.join(gate.canonical(e) + b'\n' for e in expected['operations']))
                    self.assertEqual(decoded['record'], expected['native'])

    def test_authenticates_every_literal_corpus_record(self):
        for path in sorted((DATA / 'network_cluster_cases').glob('*.json')):
            corpus = json.loads(path.read_bytes())
            values = ([corpus['initial']] if corpus.get('initial') else []) + [step['expected'] for step in corpus['steps'] or []]
            for value in values:
                with self.subTest(corpus=path.stem):
                    gate.validate_record(value['record'], corpus['config']['model']['limits'])

    def test_all_prefixes_trailing_and_nonminimal_are_refused(self):
        raw = (DATA / 'cluster_network/b.gmnetev1').read_bytes()
        for length in range(len(raw)):
            with self.assertRaises((Refusal, UnicodeError)):
                gate.decode_network(raw[:length], world=False)
        for changed in (raw + b'\0', b'X' + raw[1:], raw[:8] + b'\x81\0' + raw[9:]):
            with self.assertRaises(Refusal):
                gate.decode_network(changed, world=False)

    def test_resealed_semantics_refuse_each_actual_mutation(self):
        record = gate.decode_network((DATA / 'cluster_network/b.gmnetev1').read_bytes(), world=False)['record']
        for mutation in ('unknown_transition_kind', 'ordinal_gap', 'listener_port_zero', 'reset_retains_bytes', 'connection_byte_bound', 'total_byte_bound'):
            changed, limits = copy.deepcopy(record), dict(gate.LIMITS)
            if mutation == 'unknown_transition_kind': changed['transitions'][0]['kind'] = 'unknown'
            elif mutation == 'ordinal_gap': changed['transitions'][0]['ordinal'] += 1
            elif mutation == 'listener_port_zero': changed['snapshot']['listeners'][0]['endpoint']['port'] = 0
            elif mutation == 'reset_retains_bytes': changed['snapshot']['connections'][0].update(reset=True, client_bytes=1)
            elif mutation == 'connection_byte_bound': changed['snapshot']['connections'][0]['client_bytes'] = limits['connection_bytes'] + 1
            elif mutation == 'total_byte_bound':
                limits['bytes'] = 1
                changed['snapshot']['connections'][0]['client_bytes'] = 2
            changed['snapshot']['transition_sha256'] = gate.record_hashes(changed)[0]
            changed['snapshot']['identity'] = gate.record_hashes(changed)[1]
            self.assertEqual(gate.record_hashes(changed), (changed['snapshot']['transition_sha256'], changed['snapshot']['identity']))
            with self.subTest(mutation=mutation), self.assertRaises(Refusal):
                gate.validate_record(changed, limits)

    def test_observations_refuse_missing_catalog_and_changed_actual_state(self):
        repo = DATA.parents[5]
        config = dict(repo=str(repo))
        corpus = json.loads((DATA / 'network_cluster_cases/cross-node-ready-poll-and-stale-cancel-binary-fixture.json').read_bytes())
        row = dict(schema='gomad.core-predicate-observation/v1', case='network.corpus.initial',
                   inputs=dict(name=corpus['name']), observed=copy.deepcopy(corpus['initial']))
        with mock.patch.object(gate.native_roles, 'pinned'):
            with self.assertRaises(Refusal):
                gate.observations(config, '')
            row['observed']['record']['snapshot']['next_connection'] = 100
            with self.assertRaises(Refusal):
                gate.observations(config, gate.MARKER + json.dumps(row))

    def test_go_json_html_unicode_and_control_escapes(self):
        self.assertEqual(gate.canonical({'text':'<&>\u2028\u2029\n\0é'}), b'{"text":"\\u003c\\u0026\\u003e\\u2028\\u2029\\n\\u0000\xc3\xa9"}')


if __name__ == '__main__':
    unittest.main()
