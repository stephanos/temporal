"""Pure original-corpus framing, source closure and raw response negative controls."""
import base64
import copy
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch
import zlib

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_world as gate
from driver import Refusal, sha

REPO = Path(__file__).resolve().parents[5]


def config():
    paths = list((REPO / 'tools/gomad/rust/wasmhost/tests/testdata').rglob('*'))
    for kind in ('world', 'clock'):
        paths += list((REPO / 'tools/gomad/internal/model' / kind).glob('*.go'))
    return dict(repo=str(REPO), files={str(path): sha(path) for path in paths if path.is_file()})


def worlds():
    return gate.corpus_inputs(config(), 'world')[0]


def envelope(raw):
    compressor = zlib.compressobj(wbits=31)
    gzip = compressor.compress(raw) + compressor.flush()
    return b'GMWORLD2' + gate.u(64, 4) + gate.u(0, 4) + gate.u(len(raw)) + gate.u(len(gzip)) + bytes.fromhex(gate.sha(raw)) + gzip


class WorldReader(unittest.TestCase):
    def test_all_original_world_wires_have_independently_valid_digest_frames(self):
        count = 0
        for name, corpus in worlds().items():
            if not corpus['final']:
                continue
            with self.subTest(corpus=name):
                raw = gate.unbase(corpus['final']['expanded_base64'])
                decoded = gate.decode_world(raw, corpus['config'])
                summary = gate.decode_world(raw, corpus['config'], summary=True)
                self.assertNotIn('transitions', summary)
                self.assertEqual(summary, {key: value for key, value in decoded.items() if key not in ('requests', 'events', 'transitions')})
                self.assertEqual(decoded['expanded_sha256'], corpus['final']['expanded_sha256'])
                final = corpus['steps'][-1]['expected']['state']
                self.assertEqual(decoded['state_digest'], final['state_digest'])
                self.assertEqual(decoded['transcript_digest'], final['transcript_digest'])
                self.assertEqual(len(decoded['requests']), final['requests'])
                self.assertEqual(len(decoded['events']), final['events'])
                self.assertEqual(len(decoded['transitions']), final['transitions'])
                self.assertEqual(gate.expand(envelope(raw)), raw)
                document = gate.world_document(decoded)
                self.assertEqual(document['stateDigest'], final['state_digest'])
                self.assertEqual(document['transitions'][-1]['digest'], final['transcript_digest'])
                count += 1
        self.assertGreaterEqual(count, 50)

    def test_raw_digest_header_transition_and_trailing_mutations_refuse(self):
        raw = gate.unbase(worlds()['broker-incarnation-pattern']['final']['expanded_base64'])
        for label, mutated in [('header', b'X' + raw[1:]), ('schema', raw[:11] + b'\x02' + raw[12:]),
                               ('state', raw[:180] + bytes([raw[180] ^ 1]) + raw[181:]),
                               ('transition', raw[:-1] + bytes([raw[-1] ^ 1])), ('trailing', raw + b'\0')]:
            with self.subTest(mutation=label), self.assertRaises((Refusal, UnicodeDecodeError)):
                gate.decode_world(mutated)
        decoded = gate.decode_world(raw)
        wrong = dict(decoded['config'], seed=decoded['config']['seed'] + 1)
        with self.assertRaises(Refusal):
            gate.decode_world(raw, wrong)
        for length in (0, 8, 12, 64, 120, len(raw) - 1):
            with self.subTest(length=length), self.assertRaises(Refusal):
                gate.decode_world(raw[:length])

    def test_envelope_refuses_profile_bounds_digest_multiple_members_and_trailing(self):
        raw = gate.unbase(worlds()['broker-incarnation-pattern']['final']['expanded_base64'])
        encoded = envelope(raw)
        for mutated in (raw, encoded[:8], encoded + b'\0', encoded[:32] + bytes(32) + encoded[64:], encoded[:-1],
                        encoded[:16] + gate.u(384 << 20 | 1) + encoded[24:]):
            with self.assertRaises((Refusal, zlib.error)):
                gate.expand(mutated)
        doubled = encoded[64:] * 2
        with self.assertRaises(Refusal):
            gate.expand(encoded[:24] + gate.u(len(doubled)) + encoded[32:64] + doubled)

    def test_measured_reader_refuses_foreign_workload_and_cardinality(self):
        raw = gate.unbase(worlds()['broker-incarnation-pattern']['final']['expanded_base64'])
        for count in (True, 3, 20000, 200000):
            with self.subTest(cardinality=count), self.assertRaises(Refusal):
                gate.decode_measured_world(raw, count)
        for index in range(3):
            item = dict(sequence=index + 1, previous_digest='unused', digest='unused')
            if index == 0:
                item.update(kind='register', request=gate.measured_request(), request_id=1)
            elif index == 1:
                item.update(kind='ready', readiness=gate.measured_readiness(1), event_id=1)
            else:
                item.update(kind='quiesce', quiescence=dict(kind='delivered', before=946684800000000000, after=946684800000000001, deliveries=[dict(request_id=1, event_id=1, at=946684800000000001, kind='runtime_net_stream', payload=b'')], blocked=[]))
            gate.measured_transition(item, index)
            item['kind'] = 'cancel'
            with self.assertRaises(Refusal):
                gate.measured_transition(item, index)

    def test_exact_original_source_and_corpus_closures(self):
        for kind, count in (('world', 65), ('clock', 56)):
            docs, facts = gate.corpus_inputs(config(), kind)
            self.assertEqual(len(docs), count)
            self.assertEqual(len(facts['source_inputs']), 13 if kind == 'world' else 3)
        value = config()
        victim = next(path for path in value['files'] if path.endswith('/simulation_time.go'))
        del value['files'][victim]
        with self.assertRaises(Refusal):
            gate.corpus_inputs(value, 'clock')
        with patch.dict(gate.CORPUS_CLOSURE, world=(64, gate.CORPUS_CLOSURE['world'][1])):
            with self.assertRaises(Refusal):
                gate.corpus_inputs(config(), 'world')

    def test_raw_replay_response_cannot_be_a_pass_alias_or_caller_substitution(self):
        corpus = worlds()['broker-incarnation-pattern']
        row = dict(schema=gate.SCHEMA, case='world.corpus-replay', inputs=dict(corpus=corpus['name'], document=corpus),
                   observed=dict(steps=[step['expected'] for step in corpus['steps']], creation_error=None))
        for mode in ('pass', 'response', 'input', 'type', 'omitted'):
            changed = copy.deepcopy(row)
            if mode == 'pass':
                changed['observed'] = dict(passed=True)
            elif mode == 'response':
                changed['observed']['steps'][0]['request_id'] += 1
            elif mode == 'input':
                changed['inputs']['document']['config']['seed'] += 1
            elif mode == 'type':
                changed['observed']['steps'][0]['request_id'] = True
            else:
                changed['observed']['steps'].pop()
            with self.subTest(mode=mode), self.assertRaises(Refusal):
                gate.observations(config(), gate.MARKER + json.dumps(changed))
        with self.assertRaises(Refusal):
            gate.observations(config(), '')


if __name__ == '__main__':
    unittest.main()
