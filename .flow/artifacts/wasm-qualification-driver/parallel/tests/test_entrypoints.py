"""Configuration admission checks; no qualification workload is executed."""
import subprocess
import unittest
from pathlib import Path


HERE = Path(__file__).resolve().parent.parent


class EntrypointConfigTest(unittest.TestCase):
    def test_validation_requires_explicit_config(self):
        result = subprocess.run(
            ["bash", str(HERE / "run-validation.sh")],
            capture_output=True, text=True, timeout=10,
        )
        self.assertEqual(result.returncode, 2)
        self.assertIn("--config FILE is required", result.stderr)

    def test_performance_requires_explicit_config(self):
        result = subprocess.run(
            ["bash", str(HERE / "run-perf.sh")],
            capture_output=True, text=True, timeout=10,
        )
        self.assertEqual(result.returncode, 2)
        self.assertIn("--config FILE is required", result.stderr)

    def test_plan_needs_no_candidate_config(self):
        result = subprocess.run(
            ["bash", str(HERE / "run-validation.sh"), "--plan"],
            capture_output=True, text=True, timeout=30,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('"commands_identical": true', result.stdout)
