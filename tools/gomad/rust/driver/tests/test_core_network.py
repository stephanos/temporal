"""Pure network reader checks against the retained literal fixtures, without a host build."""
import base64
import copy
import json
from pathlib import Path
import sys
import shutil
import tempfile
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
        with mock.patch.object(gate.native_roles, 'pinned', side_effect=lambda config, path: path):
            with self.assertRaises(Refusal):
                gate.observations(config, '')
            row['observed']['record']['snapshot']['next_connection'] = 100
            with self.assertRaises(Refusal):
                gate.observations(config, gate.MARKER + json.dumps(row))

    def test_original_closure_rejects_joint_candidate_and_expectation_edits(self):
        repo = DATA.parents[5]
        for mutation in ('delete', 'add', 'expected', 'input', 'broker_world', 'wire', 'network_source', 'network_adapter', 'network_pin', 'wasi_pin'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                copied = Path(temporary)
                for directory in gate.ORIGINAL_CORPUS:
                    destination = copied / 'tools/gomad/rust/wasmhost/tests/testdata' / directory
                    shutil.copytree(DATA / directory, destination)
                for name in gate.ORIGINAL_NETWORK_SOURCES:
                    source = repo / 'tools/gomad/internal/model/network' / name
                    destination = copied / source.relative_to(repo)
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(source, destination)
                for name in ('src/network/PINNED_GO_NETWORK_SHA256', 'src/cluster/network/PINNED_GO_WASI_SHA256'):
                    source = DATA.parents[1] / name
                    destination = copied / source.relative_to(repo)
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(source, destination)
                def config():
                    return dict(repo=str(copied), files={str(path):gate.sha(path.read_bytes()) for path in copied.rglob('*') if path.is_file()})
                documents, facts = gate.corpus_inputs(config())
                self.assertEqual(facts['corpora']['broker_orders']['files'], 19)
                self.assertEqual(len(documents['network_cluster_cases']), 51)
                data = copied / 'tools/gomad/rust/wasmhost/tests/testdata'
                path = data / 'network_cluster_cases/cross-node-ready-poll-and-stale-cancel-binary-fixture.json'
                if mutation == 'delete': path.unlink()
                elif mutation == 'add': (path.parent / 'additional.json').write_bytes(path.read_bytes())
                elif mutation in ('expected', 'input'):
                    raw = path.read_bytes()
                    if mutation == 'expected': raw = raw.replace(b'"next_connection":1', b'"next_connection":2')
                    else: raw = raw.replace(b'"nanos":1', b'"nanos":2')
                    path.write_bytes(raw)
                elif mutation == 'broker_world':
                    path = data / 'broker_orders/idle-three-equal-time.json'
                    raw = path.read_bytes(); path.write_bytes(raw.replace(b'"world_base64":', b'"changed_world_base64":'))
                elif mutation == 'wire':
                    path = data / 'cluster_network/b.gmnetev1'; path.write_bytes(path.read_bytes() + b'\0')
                elif mutation in ('network_source', 'network_adapter'):
                    name = 'adapter.go' if mutation == 'network_adapter' else 'simulation.go'
                    path = copied / 'tools/gomad/internal/model/network' / name; path.write_bytes(path.read_bytes() + b'\n')
                elif mutation in ('network_pin', 'wasi_pin'):
                    path = copied / 'tools/gomad/rust/wasmhost' / ('src/network/PINNED_GO_NETWORK_SHA256' if mutation == 'network_pin' else 'src/cluster/network/PINNED_GO_WASI_SHA256')
                    path.write_text('sha256:' + '0' * 64 + '\n')
                # Re-pinning changed candidate bytes must not move the original assertion set.
                with self.assertRaises(Refusal):
                    gate.corpus_inputs(config())

    def test_zero_port_mutation_matches_native_omitted_endpoint_field(self):
        original = gate.decode_network((DATA / 'cluster_network/b.gmnetev1').read_bytes(), world=False)['record']
        expected, limits = gate.semantic_mutation(original, 'listener_port_zero')
        endpoint = expected['snapshot']['listeners'][0]['endpoint']
        self.assertNotIn('port', endpoint)
        self.assertEqual(expected['snapshot']['identity'], 'sha256:3efd21f68454242dd6c427a5fcd567e5ff989f381035ea7c474a14612ce96a02')
        self.assertNotEqual(endpoint, dict(endpoint, port=0))
        self.assertIn('port', original['snapshot']['listeners'][0]['endpoint'])
        with self.assertRaises(Refusal):
            gate.validate_record(expected, limits)

    def test_stdout_uses_physical_lf_preserving_json_unicode_separators(self):
        config = dict(repo=str(DATA.parents[5]))
        row = dict(schema='gomad.core-predicate-observation/v1', case='unrelated', inputs=dict(text='a\u2028b\u2029c'), observed={})
        raw = json.dumps(row, ensure_ascii=False)
        with mock.patch.object(gate.native_roles, 'pinned', side_effect=lambda config, path: path), mock.patch.object(gate.native_roles, 'decode', wraps=gate.native_roles.decode) as decode:
            with self.assertRaises(Refusal):
                gate.observations(config, gate.MARKER + raw + '\n')
        self.assertEqual([call.args for call in decode.call_args_list if call.args == (raw,)], [(raw,)])

    def test_go_json_html_unicode_and_control_escapes(self):
        self.assertEqual(gate.canonical({'text':'<&>\u2028\u2029\n\0é'}), b'{"text":"\\u003c\\u0026\\u003e\\u2028\\u2029\\n\\u0000\xc3\xa9"}')


if __name__ == '__main__':
    unittest.main()
