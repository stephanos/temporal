"""Source-map integrity only: none of these checks execute original predicates."""
import copy
import hashlib
import json
from pathlib import Path
import re
import unittest

REPO = Path(__file__).resolve().parents[5]
MAP = REPO/'tools/gomad/rust/driver/qualification-map.json'


def validate(mapping):
    assert mapping['qualification'] is False
    assert mapping['reference_commit'] == '1d695061b6382009d0659238c652a5a765b1d5f5'
    roots = set()
    for path in (REPO/'tools/gomad/internal/backends/wasm/qualification').glob('*_test.go'):
        for name in re.findall(rb'^func (Test\w+)\(t \*testing.T\)', path.read_bytes(), re.M):
            roots.add(f'{path.relative_to(REPO)}#{name.decode()}')
    assert len(roots) == 72
    assert set(mapping['callables']) == roots
    anchors = mapping['native_anchors']
    for key, anchor in anchors.items():
        assert key == anchor['path']+'#'+anchor['symbol']
        raw = (REPO/anchor['path']).read_bytes().splitlines(keepends=True)
        span = b''.join(raw[anchor['start_line']-1:anchor['end_line']])
        assert hashlib.sha256(span).hexdigest() == anchor['sha256'], key
    for name, row in mapping['callables'].items():
        assert row['status'] == 'UNEXECUTED' and row['evidence'] == []
        assert row['contract'] and row['native_anchors'] and row['assertion_reviews']
        assert all(key in anchors for key in row['native_anchors'])
        original = row['original']
        assert original['callable'] == name
        expected_assertions = []
        spans = [(name, dict(original['source_span'], source=name.split('#')[0])), *original['helper_source_spans'].items()]
        for owner, span in spans:
            raw = (REPO/span['source']).read_bytes()
            assert hashlib.sha256(raw[span['start_byte']:span['end_byte']]).hexdigest() == span['sha256']
            prefix = raw[:span['start_byte']].count(b'\n')
            for offset, line in enumerate(raw[span['start_byte']:span['end_byte']].decode().splitlines(), 1):
                if re.search(r'\bt\.(?:Fatal|Fatalf|Error|Errorf|Fail|FailNow)\(', line):
                    expected_assertions.append(dict(owner=owner, path=span['source'], line=prefix+offset, source=line.strip()))
        assert [review['assertion'] for review in row['assertion_reviews']] == expected_assertions
        for review in row['assertion_reviews']:
            assertion = review['assertion']
            line = (REPO/assertion['path']).read_text().splitlines()[assertion['line']-1]
            assert line.strip() == assertion['source']
            assert review['limitation'] and review['status'] != 'PASS'
    assert mapping['counts']['roots'] == len(roots)
    assert mapping['counts']['assertion_sites'] == sum(len(row['assertion_reviews']) for row in mapping['callables'].values())


class QualificationMap(unittest.TestCase):
    def setUp(self):
        self.mapping = json.loads(MAP.read_bytes())

    def test_exact_original_roots_assertions_and_native_source_bindings(self):
        validate(self.mapping)

    def test_omitted_original_root_cannot_be_hidden_by_summary_count(self):
        self.mapping['callables'].pop(next(iter(self.mapping['callables'])))
        with self.assertRaises(AssertionError):
            validate(self.mapping)

    def test_source_tamper_and_invented_anchor_refuse(self):
        for tamper in ['hash', 'anchor']:
            mapping = copy.deepcopy(self.mapping)
            if tamper == 'hash':
                next(iter(mapping['native_anchors'].values()))['sha256'] = '0'*64
            else:
                next(iter(mapping['callables'].values()))['native_anchors'] = ['invented']
            with self.assertRaises(AssertionError):
                validate(mapping)

    def test_source_map_cannot_claim_execution_or_qualification(self):
        for tamper in ['qualification', 'execution']:
            mapping = copy.deepcopy(self.mapping)
            if tamper == 'qualification':
                mapping['qualification'] = True
            else:
                next(iter(mapping['callables'].values()))['status'] = 'PASS'
            with self.assertRaises(AssertionError):
                validate(mapping)

    def test_omitted_assertion_cannot_be_hidden_by_updated_count(self):
        next(iter(self.mapping['callables'].values()))['assertion_reviews'].pop()
        self.mapping['counts']['assertion_sites'] -= 1
        with self.assertRaises(AssertionError):
            validate(self.mapping)
