"""Exercise retained-cell discovery without running qualification workloads."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class SummarizeTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        for side in ("base", "cand"):
            cell = self.root / f"{side}-onebox-rep1" / "workloads"
            results = cell / "wasm-qualifications/v1"
            results.mkdir(parents=True)
            cache = cell / "onebox-engine-cache"
            cache.mkdir()
            (cache / "module.gomad-cache-v3").write_bytes(b"cache")
            samples = [{
                "Phase": "execute", "Iteration": iteration,
                "CompileCacheHit": warm, "HostPeakRSSBytes": 1000,
                "HostPeakRSSUpperBound": False,
                "EngineCompileNanos": 0 if warm else 100,
                "CacheLoadNanos": 10 if warm else 0,
                "EnginePrepareNanos": 20 if warm else 100,
                "InstantiationNanos": 30, "ExecutionWallNanos": 40,
            } for iteration, warm in enumerate((False, True))]
            result = {
                "Qualification": {"qualified": True, "evidence": {
                    "target": {"sha256": "sha256:fixture", "size": 100}}},
                "Manifest": {"Provisional": False},
                "Budget": {"WithinBudget": True},
                "PreparationProvenanceSHA256": "sha256:provenance",
                "InitialPreparationWallNanos": 50, "Samples": samples,
            }
            (results / "result-fixture.json").write_text(json.dumps(result))
            (results / "result-fixture.publication.json").write_text("{}")

    def summarize(self):
        result = subprocess.run(
            [sys.executable, "-I", str(Path(__file__).with_name("summarize.py")),
             str(self.root)], capture_output=True, text=True, check=False)
        summary = json.loads((self.root / "summary.json").read_text())
        return result, summary

    def test_retained_cells_have_matched_measurements(self):
        result, summary = self.summarize()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(summary["failures"], [])
        self.assertEqual(summary["measurements"]["onebox"]["module_bytes"]["base"]["raw"], [100])

    def test_host_sidecar_is_not_a_measurement_cell(self):
        (self.root / "base-onebox-rep1-host.txt").write_text("synthetic host observation\n")
        result, summary = self.summarize()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(summary["failures"], [])
        self.assertEqual(set(summary["identities"]), {"onebox/base", "onebox/cand"})

    def test_unsealed_cell_directory_remains_a_failure(self):
        cell = self.root / "base-onebox-rep2"
        cell.mkdir()
        result, summary = self.summarize()
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertEqual(summary["failures"], [
            f"{cell}: expected one sealed Result, found 0"])


if __name__ == "__main__":
    unittest.main()
