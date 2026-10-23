import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).parents[1]))
import terminal_controls as controls
from driver import Refusal


class OriginalTerminalControls(unittest.TestCase):
    def rows(self):
        result = [dict(Action='start', Package=controls.PACKAGE)]
        for name in (*controls.ROOTS, *controls.CASES):
            result.extend(dict(Action=action, Package=controls.PACKAGE, Test=name) for action in ('run', 'pass'))
        result.append(dict(Action='pass', Package=controls.PACKAGE))
        return result

    def encode(self, rows):
        return b'\n'.join(json.dumps(row).encode() for row in rows)

    def test_complete_original_five_and_twelve_subcases(self):
        self.assertEqual(len(controls.events(self.encode(self.rows()))), 17)

    def test_each_original_case_must_execute_and_pass(self):
        for name in (*controls.ROOTS, *controls.CASES):
            for action in ('run', 'pass'):
                rows = [row for row in self.rows() if (row.get('Test'), row['Action']) != (name, action)]
                with self.subTest(name=name, action=action), self.assertRaises(Refusal):
                    controls.events(self.encode(rows))

    def test_skips_failures_duplicates_foreign_and_package_absence_refused(self):
        good = self.rows()
        for action in ('skip', 'fail', 'pass'):
            rows = copy.deepcopy(good)
            rows.insert(-1, dict(Action=action, Package=controls.PACKAGE, Test=controls.CASES[0]))
            with self.subTest(action=action), self.assertRaises(Refusal):
                controls.events(self.encode(rows))
        for change in ('foreign', 'unknown', 'package'):
            rows = copy.deepcopy(good)
            if change == 'foreign': rows[1]['Package'] = 'foreign'
            elif change == 'unknown': rows[1]['Test'] += '/invented'
            else: rows.pop()
            with self.subTest(change=change), self.assertRaises(Refusal):
                controls.events(self.encode(rows))
