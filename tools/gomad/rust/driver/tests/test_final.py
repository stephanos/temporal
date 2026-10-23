"""Final-reader partition fixtures, not qualification evidence."""
import sys
import unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


class FinalReader(unittest.TestCase):
    def test_complete_actual_partition_required(self):
        import final
        wanted = driver.read(driver.HERE / 'required-final.json')['subgates']
        config = dict(source_sha256='a' * 64, identity=dict(host='b' * 64, helper='c' * 64))
        def report(ids):
            return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                        identity=config['identity'], subgates=[dict(id=name, status='PASS', executed=True, exit=0) for name in ids])
        rows = [report(wanted[:124]), report(wanted[124:])]
        self.assertEqual(len(final.merge_reports(config, rows)), 145)
        for mutation in ('missing', 'duplicate', 'unexecuted', 'foreign', 'synthetic', 'skipped'):
            changed = [report(wanted[:124]), report(wanted[124:])]
            if mutation == 'missing': changed.pop()
            if mutation == 'duplicate': changed.append(report([wanted[0]]))
            if mutation == 'unexecuted': changed[0]['subgates'][0]['executed'] = False
            if mutation == 'foreign': changed[0]['identity'] = dict(host='x', helper='y')
            if mutation == 'synthetic': changed[0]['synthetic'] = True
            if mutation == 'skipped': changed[0]['subgates'][0]['status'] = 'SKIP'
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal): final.merge_reports(config, changed)
