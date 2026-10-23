#!/usr/bin/env python3
"""Offline regression checks against copied, authenticated public scope output."""
import copy
import json
from pathlib import Path
import runpy
import unittest

HERE = Path(__file__).resolve().parent
check = runpy.run_path(str(HERE / 'site-minimizer-display.py'))


class DisplayGoldenTests(unittest.TestCase):
    def setUp(self):
        self.fixture = json.loads((HERE / 'goldens-minimize/fixtures/scoped-display-input.json').read_bytes())

    def compare(self):
        view = check['projection'](self.fixture['minimize'], self.fixture['inspect'], self.fixture['retained'])
        check['compare'](view)
        return view

    def test_actual_positive_symbols_counts_and_groups_match_reviewed_golden(self):
        before = copy.deepcopy(self.fixture)
        view = self.compare()
        self.assertEqual(view['minimize']['site_scope']['patterns'], ['main'])
        self.assertEqual(view['minimize']['accepted'][0]['removed_in_scope'], 2)
        self.assertEqual(view['inspect']['accepted'][0]['removed'][0]['symbol'], dict(
            package='main', function='choose', file='gomad.wasm.exploration.fixture/main.go', line=6,
            inlined_from=[dict(package='main', function='main')]))
        self.assertEqual(view['inspect']['final_forced_by_symbol'], [dict(package='main', function='choose', count=1)])
        self.assertEqual(self.fixture, before)

    def test_single_projection_cannot_regenerate_checked_golden(self):
        view = self.compare()
        with self.assertRaisesRegex(ValueError, 'two authenticated projections'):
            check['repeated']([view])

    def test_disagreeing_projections_cannot_regenerate_checked_golden(self):
        view = self.compare()
        changed = copy.deepcopy(view)
        changed['inspect']['final_forced_by_symbol'][0]['function'] = 'incorrect'
        with self.assertRaisesRegex(ValueError, 'repeated scoped display projection'):
            check['repeated']([view, changed])

    def test_missing_removed_symbol_refuses(self):
        del self.fixture['inspect']['accepted'][0]['removed'][0]['symbol']
        with self.assertRaises((KeyError, ValueError)):
            self.compare()

    def test_mutated_removed_symbol_refuses(self):
        self.fixture['inspect']['accepted'][0]['removed'][0]['symbol']['function'] = 'incorrect'
        with self.assertRaisesRegex(ValueError, 'golden'):
            self.compare()

    def test_missing_final_group_refuses(self):
        del self.fixture['inspect']['final_forced_by_symbol']
        with self.assertRaises((KeyError, ValueError)):
            self.compare()

    def test_mutated_final_group_refuses(self):
        self.fixture['inspect']['final_forced_by_symbol'][0]['package'] = 'incorrect'
        with self.assertRaisesRegex(ValueError, 'golden'):
            self.compare()

    def test_wrong_final_group_count_refuses(self):
        self.fixture['inspect']['final_forced_by_symbol'][0]['count'] = 2
        with self.assertRaisesRegex(ValueError, 'final group count'):
            self.compare()

    def test_wrong_removal_count_refuses(self):
        self.fixture['minimize']['accepted'][0]['removed_in_scope'] = 3
        with self.assertRaisesRegex(ValueError, 'scope removal total'):
            self.compare()

    def test_balanced_but_incorrect_scope_counts_refuse(self):
        self.fixture['minimize']['accepted'][0].update(removed_in_scope=1, removed_out_of_scope=1)
        with self.assertRaisesRegex(ValueError, 'golden'):
            self.compare()

    def test_changed_inspect_ordinal_refuses_raw_join(self):
        self.fixture['inspect']['accepted'][0]['removed'][0]['ordinal'] = 99
        with self.assertRaisesRegex(ValueError, 'complete retained minimization block'):
            self.compare()

    def test_changed_cli_identity_refuses_without_normalization(self):
        self.fixture['minimize']['accepted'][0]['removed'][0]['identity'] = 'sha256:' + '0' * 64
        with self.assertRaisesRegex(ValueError, 'exact removed identities'):
            self.compare()

    def test_changed_raw_predicate_metadata_refuses(self):
        self.fixture['retained']['predicate']['failure_signature'] = '0' * 64
        with self.assertRaisesRegex(ValueError, 'complete retained minimization block'):
            self.compare()

    def test_changed_scope_refuses(self):
        self.fixture['minimize']['site_scope']['patterns'] = ['runtime']
        with self.assertRaisesRegex(ValueError, 'golden'):
            self.compare()


if __name__ == '__main__':
    unittest.main()
