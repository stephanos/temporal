"""Keep retired crate spellings out of active files while preserving evidence."""

from pathlib import Path
import os
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
            helper = SCRIPT.with_name("retired-name-matches.py")
            shutil.copyfile(helper, script.with_name(helper.name))
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            legacy = ["umpire" + separator + name
                      for separator in ["-", "_", "/"] for name in ["wasm", "tests"]]
            legacy += [prefix + name + "/x.rs"
                       for prefix in ["/repo/model/umpire/", "/repo/umpire/"]
                       for name in ["wasm", "tests"]]
            legacy += ["author/umpire/tests/consumer /repo/umpire/" + "tests/x.rs",
                       "not_author/umpire/" + "tests/x.rs"]
            legacy += ["embedded_umpire" + separator + name + "_suffix"
                       for separator in ["-", "_"] for name in ["wasm", "tests"]]
            layout = ["umpire" + separator + name
                      for separator in ["-", "_"] for name in ["lift", "annotations"]]
            layout += [prefix + "umpire/" + name
                       for prefix in ["", "model/", "../../", "../model/", "/repo/model/"] for name in [
                "dsl", "suite", "checker", "lift", "runtime", "proto",
                "testing", "fixtures", "ir-derive", "catalog-build", "annotations"]]
            layout += ["model/umpire/" + "annotations/" + suffix
                       for suffix in ["Cargo.toml", "README.md", "build.rs", "src/lib.rs", "tests/control.rs"]]

            def run(env=None):
                return subprocess.run(["bash", str(script)], capture_output=True, text=True, env=env)

            def refuses(relative, spelling):
                active = root / relative
                active.parent.mkdir(parents=True, exist_ok=True)
                active.write_text(spelling + "\n")
                result = run()
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertIn(relative + ":1:", result.stdout)

            for spelling in legacy + layout:
                with self.subTest(spelling=spelling):
                    refuses("model/source.rs", spelling)
            (root / "model/source.rs").write_text(
                "umpire-runtime umpire_fixtures umpire/author/dsl "
                "umpire/protocol/proto umpire/annotations/v1 "
                "file_umpire_annotations_v1_annotations_proto __umpire_lifted "
                "umpire/author/umpire/tests/consumer "
                "/repo/model/umpire/author/umpire/tests/consumer "
                "/repo/umpire/author/umpire/tests/consumer author/umpire/tests/consumer "
                "umpire/protocolv2 "
                "umpire/protobuf-lean/v1\n")
            for relative in ["model/archived/source.rs", ".flow/tasks/history.md",
                             ".plans/umpire-migration-audit.json",
                             ".plans/umpire-migration-manifest.json",
                             "model/design/rust-verdict-measurements.md"]:
                evidence = root / relative
                evidence.parent.mkdir(parents=True, exist_ok=True)
                evidence.write_text("\n".join(legacy + layout))
            layout_history = [".plans/archive/history.md", ".turbo/plans/history.md",
                              "model/temporal/feature/activity/compatibility/original/rows.snap"]
            for relative in layout_history:
                evidence = root / relative
                evidence.parent.mkdir(parents=True, exist_ok=True)
                evidence.write_text("\n".join(layout))
            subprocess.run(["git", "-C", str(root), "add", "-A"], check=True)
            result = run()
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            for relative in layout_history:
                for spelling in legacy:
                    with self.subTest(path=relative, spelling=spelling):
                        refuses(relative, spelling)
                (root / relative).write_text("\n".join(layout))
            result = run()
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            for index, (tool, body) in enumerate([
                ("git", "exit 2"), ("git", "printf malformed"), ("grep", "exit 2"),
            ]):
                with self.subTest(tool=tool, body=body):
                    binaries = root / f"tool-error-{index}"
                    binaries.mkdir()
                    binary = binaries / tool
                    binary.write_text("#!/bin/sh\n" + body + "\n")
                    binary.chmod(0o755)
                    env = dict(os.environ, PATH=str(binaries) + os.pathsep + os.environ["PATH"])
                    result = run(env)
                    self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
                    self.assertNotIn("clean", result.stdout)
