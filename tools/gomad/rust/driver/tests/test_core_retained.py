import base64
import copy
import unittest
import json
from unittest.mock import patch

import core_retained as reader
from driver import Refusal


def encoded(raw):
    return base64.b64encode(raw).decode()


class RetainedOracleTests(unittest.TestCase):
    def test_literal_braces_and_tagged_messages_are_complete(self):
        for raw, expected in (
            (b'failed assertion: invalid {"left":1}', 'invalid {"left":1}'),
            (b'2000-01-01\terror\tfailed assertion: invalid {state}\t{"failed-assertion":true}', 'invalid {state}'),
            (b'{"FailedAssertion":true,"msg":"whole {state}"}', 'whole {state}'),
        ):
            self.assertEqual(reader.messages(b'', raw, True, False),
                             dict(messages=[expected], report_only=False, inconclusive=False))
        self.assertTrue(reader.messages(b'\xff', b'', True, True)['inconclusive'])
        self.assertTrue(reader.messages(b'', b'failed assertion: bad\t{"failed-assertion":7}', True, False)['inconclusive'])

    def test_offered_output_and_terminal_kind_determine_completeness(self):
        handle = dict(node='n', incarnation=1)
        inputs = dict(handle=handle, stdout_base64=encoded(b'PASS\n'), stderr_base64='',
                      offered_bytes=100, termination='exit', oracle_plan=None)
        result = reader.soft_report(inputs)
        self.assertFalse(result['oracles'][0]['passed'])
        self.assertTrue(result['oracles'][0]['inconclusive'])
        inputs['offered_bytes'] = 5
        self.assertEqual(reader.soft_report(inputs)['oracles'], [])
        inputs['termination'] = 'capacity'
        self.assertTrue(reader.soft_report(inputs)['oracles'][0]['inconclusive'])

    def test_ordinary_raw_authentication_is_independent_of_json_spelling(self):
        rows = []
        expected = dict(configuration={}, schema='native')
        for case in reader.ORDINARY:
            value = copy.deepcopy(expected)
            if case == 'explicit-optional-null': value['configuration']['oracle_plan'] = None
            raw = reader.canonical(value) + (b' \n' if case == 'trailing-space' else b'')
            descriptor = dict(bytes=len(raw), sha256=reader.digest(raw))
            rows.append(dict(case='ordinary/' + case, inputs=dict(payload_base64=encoded(raw),
                appended_base64=encoded(raw + b'\n'), canonical_inputs=expected,
                manifest=dict(files={reader.FILE: descriptor}, record=dict(streams={reader.FILE: descriptor}))),
                observed=dict(canonical_inputs=expected, raw_sha256=reader.digest(raw), appended_error='digest mismatch',
                              public_status=0, public_output_base64=encoded(b'reproduced=true'))))
        self.assertEqual(len(reader.ordinary(rows)), 7)
        forged = copy.deepcopy(rows)
        forged[0]['inputs']['manifest']['files'][reader.FILE] = dict(bytes=0, sha256='0' * 64)
        with self.assertRaises(Refusal): reader.ordinary(forged)
        forged = copy.deepcopy(rows)
        forged[0]['observed']['public_status'] = 2
        with self.assertRaises(Refusal): reader.ordinary(forged)

    def test_ndjson_uses_physical_lf_and_preserves_unicode_line_separators(self):
        row = dict(schema='gomad.retained-reader-observation/v1', case='unicode-control',
                   inputs={'text': 'before\u2028middle\u0085after'}, observed={})
        raw = reader.MARKER + json.dumps(row, ensure_ascii=False) + '\n'
        with patch('native_roles.pinned'), patch.object(reader, 'ordinary', return_value=[]), \
             patch.object(reader, 'soft_cases', return_value=[]) as soft, \
             patch.object(reader, 'oracle_cases', return_value=[]), patch.object(reader, 'producer_cases', return_value=[]):
            reader.observations({'repo': '/sealed'}, raw)
        self.assertEqual(soft.call_args.args[0], [row])

    def test_missing_catalog_does_not_receive_original_root(self):
        with self.assertRaises(Refusal): reader.oracle_cases([])
        with self.assertRaises(Refusal): reader.producer_cases([])
        with self.assertRaises(Refusal): reader.soft_cases([])
