"""Keep retired crate spellings out of active files while preserving evidence."""

from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "retired-names.sh"


class RetiredNames(unittest.TestCase):
    def test_active_spellings_and_historical_exclusions(self):
        with tempfile.TemporaryDirectory(prefix="retired-names-") as scratch:
            root = Path(scratch)
            script = root / "model/scripts/retired-names.sh"
            script.parent.mkdir(parents=True)
            shutil.copyfile(SCRIPT, script)
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            spellings = ["umpire" + separator + name
                         for separator in ["-", "_", "/"] for name in ["wasm", "tests"]]
            active = root / "model/source.rs"
            for spelling in spellings:
                with self.subTest(spelling=spelling):
                    active.write_text(spelling + "\n")
                    result = subprocess.run(["bash", str(script)], capture_output=True, text=True)
                    self.assertEqual(result.returncode, 1, result.stderr)
                    self.assertIn("model/source.rs:1:", result.stdout)
            active.write_text("umpire-runtime umpire_fixtures\n")
            for relative in ["model/archived/source.rs", ".flow/tasks/history.md",
                             ".plans/umpire-migration-audit.json",
                             ".plans/umpire-migration-manifest.json",
                             "model/design/rust-verdict-measurements.md"]:
                evidence = root / relative
                evidence.parent.mkdir(parents=True, exist_ok=True)
                evidence.write_text("\n".join(spellings))
            subprocess.run(["git", "-C", str(root), "add", "-A"], check=True)
            result = subprocess.run(["bash", str(script)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
