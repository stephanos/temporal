"""Synthetic checker tests; these fixtures are never guest observations."""
import copy
from collections import Counter, defaultdict
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent
EVIDENCE = Path('/home/agent/wt/evidence/fn221')
CHECKER = Path(os.environ.get('FN221_CHECKER', ROOT / 'merge-check.py'))


class MergeCheckTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='synthetic-', dir=ROOT)
        self.addCleanup(self.tmp.cleanup)
        self.runroot = Path(self.tmp.name)
        (self.runroot / 'matrix').mkdir()
        self.original = EVIDENCE / 'full-1d7b37f00a'
        self.sidecar = json.loads((self.original / 'matrix/sweep.json').read_text())
        self.prefix = (self.original / 'matrix/sweep.jsonl').read_bytes()
        self.invalidation = (EVIDENCE / 'rerun-20261022T1705Z/invalidation-rows.jsonl').read_bytes()
        self.selected = {(r['test']['package'], r['test']['name']) for r in map(json.loads, self.invalidation.splitlines())}
        self.header = next(json.loads(line) for line in reversed((EVIDENCE / 'rerun-20261022T1705Z/matrix/sweep.jsonl').read_bytes().splitlines()) if json.loads(line)['kind'] == 'header')
        self.tests = [dict(schema=self.header['schema'], kind='test', candidate=self.header['candidate'], time=self.header['time'], test=copy.deepcopy(r)) for r in self.sidecar['rows'] if (r['package'], r['name']) in self.selected]
        self.records = [copy.deepcopy(self.header), *self.tests[:4], copy.deepcopy(self.header), *self.tests[4:]]

    def run_checker(self):
        (self.runroot / 'invalidation-rows.jsonl').write_bytes(self.invalidation)
        (self.runroot / 'matrix/sweep.jsonl').write_bytes(self.prefix + self.invalidation + b''.join(json.dumps(r).encode() + b'\n' for r in self.records))
        (self.runroot / 'matrix/sweep.json').write_text(json.dumps(self.sidecar))
        out = self.runroot / 'checker.json'
        result = subprocess.run([sys.executable, '-I', str(CHECKER), str(self.original), str(self.runroot), str(EVIDENCE / 'rerun-prep/rerun-units.tsv'), str(out)], capture_output=True, text=True)
        return result, json.loads(out.read_text()) if out.exists() else {}

    def test_repeated_same_candidate_headers(self):
        result, report = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(report['findings'], [])
        self.assertEqual((report['units'], report['kept_original_rows'], report['replaced_by_rerun']), (1488, 1144, 344))

    def test_many_headers_including_empty_invocations(self):
        self.records = [copy.deepcopy(self.header), copy.deepcopy(self.header), *self.records,
                        copy.deepcopy(self.header)]
        result, report = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(report['rerun_invocation_headers'], 5)

    def test_timeout_to_network_capacity_is_explicitly_flagged(self):
        row = next(r['test'] for r in self.records if r['kind'] == 'test' and r['test']['class'] == 'timeout')
        row.update({'class': 'capacity-exceeded', 'reason': 'WASI capacity: network-v2-evidence-bytes'})
        index = next(i for i, r in enumerate(self.sidecar['rows']) if (r['package'], r['name']) == (row['package'], row['name']))
        self.sidecar['rows'][index] = copy.deepcopy(row)
        self.sidecar['by_class'] = dict(Counter(r['class'] for r in self.sidecar['rows']))
        counts = defaultdict(Counter)
        for r in self.sidecar['rows']:
            counts[r['package']][r['class']] += 1
        self.sidecar['by_package'] = {k: dict(v) for k, v in counts.items()}
        self.sidecar['reasons'] = [dict(zip(('class', 'reason', 'count'), (cls, reason, count)))
                                   for (cls, reason), count in Counter((r['class'], r.get('reason', '')) for r in self.sidecar['rows'] if r['class'] != 'pass').items()]
        result, report = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(report['flags']), 1)
        self.assertIn('network-v2-evidence-bytes', report['flags'][0]['flag'])

    def test_stopped_actual_rerun_is_incomplete(self):
        out = self.runroot / 'incomplete.json'
        result = subprocess.run([sys.executable, '-I', str(CHECKER), str(self.original),
                                 str(EVIDENCE / 'rerun-20261022T1705Z'),
                                 str(EVIDENCE / 'rerun-prep/rerun-units.tsv'), str(out)],
                                capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(json.loads(out.read_text())['status'], 'incomplete')

    def test_foreign_resume_header(self):
        self.records = [copy.deepcopy(self.header), *self.tests]
        cases = [
            ('candidate key', lambda r: r.update(candidate='sha256:foreign')),
            ('candidate payload', lambda r: r['header']['candidate'].update(guest_config_sha256='sha256:foreign')),
            ('execution budget', lambda r: r['header']['candidate'].update(per_test_timeout='4m0s')),
            ('overall budget', lambda r: r['header'].update(overall_budget='25h0m0s')),
            ('retention budget', lambda r: r['header'].update(retain_bytes=1)),
            ('enumeration', lambda r: r['header']['packages'][0]['tests'].pop()),
            ('statement', lambda r: r['header'].update(statement='qualified')),
            ('schema', lambda r: r.update(schema='foreign')),
            ('argv budget', lambda r: r['header']['command'].__setitem__(r['header']['command'].index('--per-test-timeout') + 1, '4m')),
            ('max tests', lambda r: r['header'].update(max_tests=10)),
        ]
        for name, mutate in cases:
            with self.subTest(name=name):
                saved = copy.deepcopy(self.records)
                mutate(self.records[0])
                result, report = self.run_checker()
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(report['findings'])
                self.records = saved

    def test_foreign_later_resume_header(self):
        self.records[5]['header']['candidate']['helper_sha256'] = 'sha256:foreign'
        result, report = self.run_checker()
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any('header 2:' in f and 'candidate' in f for f in report['findings']))

    def test_missing_duplicate_or_nonselected_unit(self):
        for name in ['missing', 'duplicate', 'nonselected', 'not-run']:
            with self.subTest(name=name):
                saved = copy.deepcopy(self.records)
                if name == 'missing': self.records.pop()
                if name == 'duplicate': self.records.append(copy.deepcopy(self.tests[0]))
                if name == 'nonselected': self.records[-1]['test']['name'] = 'TestForeign'
                if name == 'not-run': self.records[-1]['test']['class'] = 'not-run'
                result, report = self.run_checker()
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(report['findings'])
                self.records = saved

    def test_original_prefix_and_invalidation_preserved(self):
        for field in ['prefix', 'invalidation']:
            with self.subTest(field=field):
                saved = getattr(self, field)
                setattr(self, field, saved + saved.splitlines(keepends=True)[0] if field == 'invalidation' else saved.replace(b'header', b'headeX', 1))
                result, report = self.run_checker()
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(report['findings'])
                setattr(self, field, saved)

    def test_changed_row_module_or_counts(self):
        for field in ['kept', 'module', 'counts']:
            with self.subTest(field=field):
                saved = copy.deepcopy(self.sidecar)
                saved_records = copy.deepcopy(self.records)
                if field == 'kept':
                    next(r for r in self.sidecar['rows'] if (r['package'], r['name']) not in self.selected)['reason'] = 'changed'
                if field == 'module':
                    row = next(r['test'] for r in self.records if r['kind'] == 'test' and r['test'].get('module_sha256'))
                    row['module_sha256'] = 'sha256:foreign'
                    next(r for r in self.sidecar['rows'] if (r['package'], r['name']) == (row['package'], row['name']))['module_sha256'] = 'sha256:foreign'
                if field == 'counts':
                    self.sidecar['by_class']['pass'] = 9999
                result, report = self.run_checker()
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(report['findings'])
                if field == 'module':
                    self.assertTrue(any('rerun module' in f for f in report['findings']))
                if field == 'counts':
                    self.assertTrue(any('by_class' in f for f in report['findings']))
                self.sidecar = saved
                self.records = saved_records


if __name__ == '__main__':
    unittest.main()
