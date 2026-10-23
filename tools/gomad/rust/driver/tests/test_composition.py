import sys
from pathlib import Path
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import compose
from driver import Refusal, mapping


class CompositionTests(unittest.TestCase):
    def base(self):
        return dict(python=sys.executable, identity=dict(host='a' * 64, helper='b' * 64), stages={name: dict(adapter='pending') for name in mapping()})

    def test_missing_label(self):
        config = self.base()
        del config['stages']['storage']
        with self.assertRaisesRegex(Refusal, 'missing/foreign'):
            compose.validate_stages(config)

    def test_foreign_label(self):
        config = self.base()
        config['stages']['new-stage'] = dict(adapter='pending')
        with self.assertRaisesRegex(Refusal, 'missing/foreign'):
            compose.validate_stages(config)

    def test_pending_refuses_before_execution(self):
        with self.assertRaisesRegex(Refusal, 'unbound original stage endpoints'):
            compose.validate_stages(self.base())

    def test_unknown_candidate(self):
        config = self.base()
        config['identity']['host'] = '0' * 64
        with self.assertRaisesRegex(Refusal, 'unknown candidate'):
            compose.validate_stages(config)

    def test_wrong_semantic_adapter(self):
        config = self.base()
        config['stages']['runtime-0'] = dict(adapter='corpus')
        with self.assertRaisesRegex(Refusal, 'wrong semantic adapter'):
            compose.validate_stages(config)

    def test_live_exit_only_reader_refused(self):
        config = self.base()
        config['stages']['ordinary'] = dict(adapter='checked-command', reader='static-exit')
        with self.assertRaisesRegex(Refusal, 'actual semantic reader'):
            compose.validate_stages(config)

    def test_fixed_final_partition_and_order(self):
        sections = compose.final_sections('/sealed/python', Path('/sealed/driver.json'))
        self.assertEqual(len([item for row in sections.values() for item in row['required']]), 8)
        self.assertLess(sorted(sections).index('10-stress'), sorted(sections).index('90-public-verbs'))
        self.assertIn('--read-only', sections['40-retained']['argv'])
        self.assertNotIn('--read-only', sections['60-inspection']['argv'])


if __name__ == '__main__':
    unittest.main()
