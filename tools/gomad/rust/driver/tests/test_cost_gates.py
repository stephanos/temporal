"""Go-era corpus/catalogue reader cost gates (MILESTONES fn-232.13 ruling 6)."""
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import adapters
import driver

MATCHED = dict(integrated_source_sha256='a' * 64, workload_source_sha256='b' * 64, host_os='linux', host_arch='arm64',
               seed=7, repeat=2, fuel=1 << 50, memory_bytes=4 << 30)


class CorpusReaderCost(unittest.TestCase):
    def corpus(self, change=None, usage=None):
        with tempfile.TemporaryDirectory() as directory:
            base, reports = Path(directory), {}
            for index, (cell, wanted) in enumerate(adapters.CELLS.items()):
                stage = base / 'stages' / cell
                stage.mkdir(parents=True)
                result = stage / 'result-cell.json'
                result.write_text(cell)
                (stage / 'result-cell.json.publication-1.json').write_text('receipt')
                driver.write(stage / 'result.json', dict(path=str(result), sha256=driver.sha(result), pid=100 + index))
                manifest = dict(MATCHED, **wanted)
                if change and change[0] == cell:
                    manifest[change[1]] = change[2]
                reports[str(result)] = dict(manifest=manifest, cost=dict(cpu_nanos=1000 + index),
                                            _driver_publication=dict(cpu_nanos=10))
            root = base / 'stages/workload-corpus'
            root.mkdir()
            readings = iter(usage or [(5, 5, 1 << 20), (7, 9, 1 << 20)])
            with patch.object(adapters, 'result_report', lambda config, root, name, path: reports[path]), \
                    patch.object(adapters, 'reader_usage', lambda: next(readings)):
                observed = adapters.corpus(dict(root=directory), root, 'workload-corpus')
            return observed, driver.read(root / 'corpus.json')

    def test_corpus_cpu_counts_each_producer_and_its_publication_tail_once(self):
        observed, corpus = self.corpus()
        self.assertEqual(observed, adapters.passed('workload-corpus'))
        self.assertEqual((corpus['producer_cpu_nanos'], corpus['producer_terminal_cpu_nanos'], corpus['reader_cpu_nanos']),
                         (sum(1000 + index for index in range(10)), 100, 6))
        self.assertEqual(corpus['total_cpu_nanos'], corpus['producer_cpu_nanos'] + 100 + 6)
        self.assertEqual(corpus['reader_peak_rss_bytes'], 1 << 20)

    def test_corpus_reader_self_measurement_is_bounded_and_monotone(self):
        message = '^mandatory corpus checker self observations unavailable, nonmonotone, or over 2GiB$'
        for after in ((7, 9, 0), (7, 9, (2 << 30) + 1), (4, 9, 1 << 20), (7, 4, 1 << 20)):
            with self.subTest(after=after), self.assertRaisesRegex(driver.Refusal, message):
                self.corpus(usage=[(5, 5, 1 << 20), after])
        self.assertEqual(self.corpus(usage=[(5, 5, 1), (5, 5, 2 << 30)])[1]['reader_cpu_nanos'], 0)

    def test_reader_usage_is_this_process(self):
        user, system, peak = adapters.reader_usage()
        self.assertGreater(peak, 0)
        self.assertGreaterEqual(user + system, 0)


if __name__ == '__main__':
    unittest.main()
