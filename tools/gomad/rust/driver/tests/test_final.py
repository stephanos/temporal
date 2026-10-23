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
        self.assertEqual(len(final.merge_reports(config, rows)), 146)
        for mutation in ('missing', 'duplicate', 'unexecuted', 'foreign', 'synthetic', 'skipped'):
            changed = [report(wanted[:124]), report(wanted[124:])]
            if mutation == 'missing': changed.pop()
            if mutation == 'duplicate': changed.append(report([wanted[0]]))
            if mutation == 'unexecuted': changed[0]['subgates'][0]['executed'] = False
            if mutation == 'foreign': changed[0]['identity'] = dict(host='x', helper='y')
            if mutation == 'synthetic': changed[0]['synthetic'] = True
            if mutation == 'skipped': changed[0]['subgates'][0]['status'] = 'SKIP'
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal): final.merge_reports(config, changed)


class ReceivingPartition(unittest.TestCase):
    def test_exact_124_live_14_receiving_and_eight_unbound(self):
        import final
        self.assertEqual(len(final.live.required()), 124)
        self.assertEqual(len(final.receiving.OWNED), 14)
        self.assertEqual(final.remaining_ids(), {
            'fn185-R4/qualified-workload-choice-wasi-network-stdout-stderr-preservation',
            'fn185-R4/runtime-cost-equivalent-release-behavior',
            'inspection/rotated-three-node-ringpop-120s-fresh-exact-replay',
            'operator/detached-installed-discover-inspect-replay',
            'public-verbs/fresh-differential-and-raw-golden-integrity',
            'retained/20-fresh-observed-replays',
            'stress/41-cases-four-axes-original-repetitions',
            'terminal/current-host-admission-and-linked-attribution',
        })

    def test_receiving_source_presence_cannot_replace_actual_fourteen_results(self):
        import final
        from unittest.mock import patch
        config = dict(source_sha256='a'*64, identity=dict(host='b'*64, helper='c'*64))
        rows = [dict(id=name, status='PASS', executed=True, exit=0) for name in final.receiving.OWNED]
        with patch.object(final.receiving, 'read_receiving', return_value=rows) as reader:
            self.assertEqual(len(final.receiving_report(config)['subgates']), 14)
            reader.assert_called_once_with(config)
        for mutation in ('missing', 'duplicate', 'unexecuted', 'failed'):
            changed = [dict(row) for row in rows]
            if mutation == 'missing': changed.pop()
            if mutation == 'duplicate': changed.append(dict(changed[0]))
            if mutation == 'unexecuted': changed[0].update(status='UNEXECUTED', executed=False, exit=4)
            if mutation == 'failed': changed[0].update(status='FAIL', exit=1)
            with self.subTest(mutation=mutation), patch.object(final.receiving, 'read_receiving', return_value=changed), self.assertRaises(driver.Refusal):
                final.receiving_report(config)

    def test_remaining_eight_missing_refuses_before_owned_reader_dispatch(self):
        import final
        from unittest.mock import patch
        config = dict(root='/actual-attempt', final_sections={})
        with patch.object(final, 'guard'), patch.object(final.receiving, 'read_receiving') as receiving, self.assertRaises(driver.Unexecuted):
            final.collect(config, Path(config['root']))
        receiving.assert_not_called()
