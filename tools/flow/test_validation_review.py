"""Isolated controls for the installed spec-close validation-cost review."""

import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


FLOWCTL = Path(os.environ.get("FLOWCTL_PY", str(Path.home() / ".codex/scripts/flowctl.py"))).resolve()


class ValidationReview(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="flow-validation-review-")
        self.addCleanup(self.scratch.cleanup)
        self.repo = Path(self.scratch.name).resolve()
        self.env = dict(os.environ, BASH_ENV="/dev/null")
        self.env.pop("FLOW_STATE_DIR", None)
        self.git("init", "-b", "main")
        self.git("config", "user.name", "Flow Test")
        self.git("config", "user.email", "flow-test@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.flow("init", "--json")
        self.id = self.flow("spec", "create", "--title", "Cost controls", "--json")["id"]
        self.spec_path = self.repo / ".flow/specs" / (self.id + ".json")
        self.config_path = self.repo / ".flow/config.json"
        self.config(True)
        self.git("add", "-A")
        self.git("commit", "-m", "Open spec")
        self.input = self.repo / ".flow/tmp/cost.json"
        self.input.parent.mkdir(exist_ok=True)
        self.report = {
            "schema_version": 1, "spec_id": self.id,
            "gates": [{"command": "python3 focused.py", "mode": "isolated",
                       "cost": {"kind": "measured", "seconds": 1.5},
                       "coverage": "R1-R4", "evidence": ["existing test log"]}],
            "speedup_decisions": [{"opportunity": "duplicate full suites", "decision": "retain",
                                   "rationale": "Focused controls cover distinct behavior",
                                   "coverage_preserved": "Keep every success and refusal control"}],
        }
        self.write_report()

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.repo, env=self.env,
                              check=True, capture_output=True, text=True).stdout

    def flow(self, *args, ok=True):
        result = subprocess.run([sys.executable, str(FLOWCTL), *args], cwd=self.repo,
                                env=self.env, capture_output=True, text=True)
        if ok:
            self.assertEqual(0, result.returncode, result.stdout + result.stderr)
            return json.loads(result.stdout)
        self.assertNotEqual(0, result.returncode, result.stdout + result.stderr)
        return result.stdout + result.stderr

    def config(self, required):
        self.config_path.write_text(json.dumps({"specClose": {"validationReviewRequired": required}}))

    def write_report(self):
        self.input.write_text(json.dumps(self.report, indent=2) + "\n")

    def close(self, *args, ok=True):
        return self.flow("spec", "close", self.id, *args, "--json", ok=ok)

    def snapshot(self):
        return {p: p.read_bytes() for directory in ("specs", "tasks")
                for p in (self.repo / ".flow" / directory).glob("*") if p.is_file()}

    def assert_refusal(self, *args, text="validation"):
        before = self.snapshot()
        result = self.close(*args, ok=False)
        self.assertIn(text.lower(), result.lower())
        self.assertEqual(before, self.snapshot())

    def load_module(self):
        name = "flowctl_validation_test"
        spec = importlib.util.spec_from_file_location(name, FLOWCTL)
        module = importlib.util.module_from_spec(spec)
        sys.modules[name] = module
        spec.loader.exec_module(module)
        return module

    def invoke(self, module):
        with patch.object(module, "get_repo_root", return_value=self.repo), \
             patch.object(module, "get_flow_dir", return_value=self.repo / ".flow"), \
             patch.object(module, "get_state_dir", return_value=self.repo / ".git/flow-state"):
            with patch.object(sys, "argv", [str(FLOWCTL), "spec", "close", self.id, "--validation-review-file", str(self.input), "--json"]):
                module.main()

    def test_exact_uncommitted_staged_and_cost_kinds(self):
        for kind in ("measured", "estimated", "unknown"):
            with self.subTest(kind=kind):
                self.report["gates"][0]["cost"] = ({"kind": kind, "reason": "Historical timing unavailable"}
                                                      if kind == "unknown" else {"kind": kind, "seconds": 1.5})
                self.write_report()
                if kind == "estimated":
                    self.input = self.repo / "staged-cost.json"
                    self.write_report()
                    self.git("add", str(self.input))
                head = self.git("rev-parse", "HEAD")
                result = self.close("--validation-review-file", str(self.input))
                data = json.loads(self.spec_path.read_text())
                ref = data["validation_review"]
                raw = self.input.read_bytes()
                self.assertEqual(hashlib.sha256(raw).hexdigest(), ref["sha256"])
                self.assertEqual(1, ref["schema_version"])
                artifact = self.repo / ref["path"]
                self.assertEqual(raw, artifact.read_bytes())
                self.assertIn(str(artifact), result["modified_paths"])
                self.assertEqual(head, self.git("rev-parse", "HEAD"))
                self.git("add", "--", *result["modified_paths"])
                self.git("commit", "-m", "Normal closure")
                committed = subprocess.check_output(["git", "show", "HEAD:" + ref["path"]], cwd=self.repo)
                self.assertEqual(raw, committed)
                data["status"] = "open"
                self.spec_path.write_text(json.dumps(data))

    def test_no_validation_no_safe_savings_and_done_reopen(self):
        self.report["gates"] = []
        self.report["no_validation_reason"] = "Documentation-only spec; existing review covers prose"
        self.write_report()
        self.close("--validation-review-file", str(self.input))
        ref = json.loads(self.spec_path.read_text())["validation_review"]
        self.config_path.write_text("{broken")
        self.close("--validation-review-file", "missing.json")
        self.assertEqual(ref, json.loads(self.spec_path.read_text())["validation_review"])
        data = json.loads(self.spec_path.read_text()); data["status"] = "open"
        self.spec_path.write_text(json.dumps(data)); self.config(True)
        self.assert_refusal(text="Can these validation gates be sped up")
        result = self.close("--validation-review-file", str(self.input))
        self.assertIn(str(self.repo / ref["path"]), result["modified_paths"])

    def test_bad_reports_leave_lifecycle_unchanged(self):
        cases = [
            ("version", lambda r: r.update(schema_version=True)),
            ("wrong spec", lambda r: r.update(spec_id="fn-999-other")),
            ("gate type", lambda r: r.update(gates={})),
            ("empty gates", lambda r: r.update(gates=[])),
            ("negative", lambda r: r["gates"][0]["cost"].update(seconds=-1)),
            ("infinite", lambda r: r["gates"][0]["cost"].update(seconds=float("inf"))),
            ("boolean seconds", lambda r: r["gates"][0]["cost"].update(seconds=True)),
            ("unknown reason", lambda r: r["gates"][0].update(cost={"kind": "unknown"})),
            ("coverage", lambda r: r["gates"][0].update(coverage="")),
            ("evidence", lambda r: r["gates"][0].update(evidence=[9])),
            ("decisions", lambda r: r.update(speedup_decisions=[])),
            ("rationale", lambda r: r["speedup_decisions"][0].update(rationale="")),
            ("collection bound", lambda r: r.update(gates=r["gates"] * 129)),
            ("size bound", lambda r: r.update(extra="x" * (256 * 1024))),
        ]
        original = copy.deepcopy(self.report)
        for name, mutate in cases:
            with self.subTest(name=name):
                self.report = copy.deepcopy(original); mutate(self.report); self.write_report()
                self.assert_refusal("--validation-review-file", str(self.input))
        self.input.write_text("{bad")
        self.assert_refusal("--validation-review-file", str(self.input))
        self.assert_refusal("--validation-review-file", "absent.json")
        self.assert_refusal(text="Can these validation gates be sped up")
        result = self.flow("spec", "close", self.id, ok=False)
        self.assertIn("--validation-review-file", result)

    def test_config_errors_and_optout(self):
        for raw in ('{bad', '[]', '{"specClose":null}', '{"specClose":{"validationReviewRequired":"true"}}'):
            with self.subTest(raw=raw):
                self.config_path.write_text(raw); self.assert_refusal(text="config")
        self.config(False)
        self.input.write_text("invalid")
        self.assert_refusal("--validation-review-file", str(self.input))
        self.config_path.write_text("{}")
        self.assertFalse(self.flow("config", "get", "specClose.validationReviewRequired", "--json")["value"])
        self.close()

    def test_unsafe_paths_and_corrupt_artifact(self):
        self.assert_refusal("--validation-review-file", str(self.repo.parent / "outside.json"))
        link = self.repo / "link.json"; link.symlink_to(self.input)
        self.assert_refusal("--validation-review-file", str(link))
        directory = self.repo / "linkdir"; directory.symlink_to(self.input.parent, target_is_directory=True)
        self.assert_refusal("--validation-review-file", str(directory / self.input.name))
        self.assert_refusal("--validation-review-file", str(self.input.parent))
        raw = self.input.read_bytes(); digest = hashlib.sha256(raw).hexdigest()
        artifact = self.repo / ".flow/artifacts" / self.id / ("validation-cost-review-" + digest + ".json")
        artifact.parent.mkdir(parents=True, exist_ok=True); artifact.write_text("corrupt")
        self.assert_refusal("--validation-review-file", str(self.input))
        self.assertEqual(b"corrupt", artifact.read_bytes())
        artifact.unlink(); artifact.symlink_to(self.input)
        self.assert_refusal("--validation-review-file", str(self.input))
        artifact.unlink(); artifact.parent.rmdir()
        artifact.parent.symlink_to(self.input.parent, target_is_directory=True)
        self.assert_refusal("--validation-review-file", str(self.input))

    def test_publication_failure_and_retry_commit_fence(self):
        module = self.load_module()
        before = self.snapshot()
        with patch.object(module.os, "link", side_effect=OSError("injected publication failure")):
            with self.assertRaises(SystemExit):
                self.invoke(module)
        self.assertEqual(before, self.snapshot())
        real_write = module.atomic_write_json
        def fail_before_lifecycle(path, data):
            if path == self.spec_path:
                raise OSError("injected lifecycle failure after publication")
            return real_write(path, data)
        with patch.object(module, "atomic_write_json", side_effect=fail_before_lifecycle):
            with self.assertRaises(OSError):
                self.invoke(module)
        self.assertEqual(before, self.snapshot())
        published = list((self.repo / ".flow/artifacts" / self.id).glob("validation-cost-review-*.json"))
        self.assertEqual(1, len(published))
        result = self.close("--validation-review-file", str(self.input))
        self.assertIn(str(published[0]), result["modified_paths"])
        self.git("add", "--", *result["modified_paths"])
        self.git("commit", "-m", "Retry normal close fence")
        ref = json.loads(self.spec_path.read_text())["validation_review"]
        self.assertEqual(self.input.read_bytes(), subprocess.check_output(["git", "show", "HEAD:" + ref["path"]], cwd=self.repo))

    def test_partial_task_publication_retains_existing_failure_semantics(self):
        task = self.flow("task", "create", "--spec", self.id, "--title", "Done runtime", "--json")["id"]
        task_path = self.repo / ".flow/tasks" / (task + ".json")
        runtime = self.repo / ".git/flow-state/tasks" / (task + ".state.json")
        runtime.parent.mkdir(parents=True, exist_ok=True)
        runtime.write_text(json.dumps({"status": "done"}))
        before_spec = self.spec_path.read_bytes()
        self.assertNotEqual("done", json.loads(task_path.read_text())["status"])
        module = self.load_module()
        real_write = module.atomic_write_json
        def fail_final_spec(path, data):
            if path == self.spec_path:
                raise OSError("injected final spec publication failure")
            return real_write(path, data)
        with patch.object(module, "atomic_write_json", side_effect=fail_final_spec):
            with self.assertRaises(OSError):
                self.invoke(module)
        self.assertEqual(before_spec, self.spec_path.read_bytes())
        self.assertEqual("done", json.loads(task_path.read_text())["status"])
        self.close("--validation-review-file", str(self.input))

    def test_optout_performs_no_report_work(self):
        self.config(False)
        module = self.load_module()
        with patch.object(module, "get_repo_root", return_value=self.repo), \
             patch.object(module, "get_flow_dir", return_value=self.repo / ".flow"), \
             patch.object(module, "_prepare_cost_review", side_effect=AssertionError("unexpected report work")), \
             patch.object(sys, "argv", [str(FLOWCTL), "spec", "close", self.id, "--json"]):
            module.main()

    def test_unreadable_input_and_no_external_work(self):
        module = self.load_module()
        before = self.snapshot()
        with patch.object(module, "_read_cost_review_at", side_effect=PermissionError("input unreadable")):
            with self.assertRaises(SystemExit):
                self.invoke(module)
        self.assertEqual(before, self.snapshot())
        with patch.object(module, "get_repo_root", return_value=self.repo), \
             patch.object(module.subprocess, "run", side_effect=AssertionError("unexpected subprocess")):
            reference, artifact = module._prepare_cost_review(self.repo / ".flow", self.id, str(self.input))
        self.assertEqual(self.input.read_bytes(), artifact.read_bytes())
        self.assertEqual(hashlib.sha256(self.input.read_bytes()).hexdigest(), reference["sha256"])

    def test_incomplete_task_precedes_report(self):
        self.flow("task", "create", "--spec", self.id, "--title", "Open", "--json")
        self.assert_refusal(text="incomplete tasks")


class ValidationReviewInstall(unittest.TestCase):
    def test_install_mismatch_recovery_and_atomic_per_file_replacement(self):
        spec = importlib.util.spec_from_file_location("validation_install", Path(__file__).with_name("install_validation_review.py"))
        module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory(prefix="flow-validation-install-") as scratch:
            root = Path(scratch)
            stage, target, backup = root / "stage", root / "installed", root / "backups"
            (stage / "scripts").mkdir(parents=True); target.mkdir()
            for name in module.TARGETS:
                (stage / (name + ".baseline")).write_bytes(b"baseline")
                (stage / "scripts" / name).write_bytes(b"tested replacement")
                (target / name).write_bytes(b"baseline")
            (target / module.TARGETS[1]).write_bytes(b"another installer")
            with self.assertRaisesRegex(ValueError, "baseline changed"):
                module.install(stage, target, backup)
            self.assertEqual(b"baseline", (target / module.TARGETS[0]).read_bytes())
            self.assertEqual(b"another installer", (target / module.TARGETS[1]).read_bytes())
            (target / module.TARGETS[1]).write_bytes(b"baseline")
            original_mode = (target / module.TARGETS[0]).stat().st_mode
            module.install(stage, target, backup)
            for name in module.TARGETS:
                self.assertEqual(b"tested replacement", (target / name).read_bytes())
                self.assertEqual(b"baseline", (backup / name).read_bytes())
            self.assertEqual(original_mode, (target / module.TARGETS[0]).stat().st_mode)
            # A changed target between staging and the last pre-replace check survives.
            staged = stage / "scripts" / module.TARGETS[0]
            baseline = stage / (module.TARGETS[0] + ".baseline")
            destination = target / module.TARGETS[0]
            destination.write_bytes(b"baseline")
            real_mode = module.shutil.copymode
            def competing_installer(source, dest):
                real_mode(source, dest)
                if Path(dest).name.startswith(".flow-validation-install-"):
                    destination.write_bytes(b"concurrent change")
            with patch.object(module.shutil, "copymode", side_effect=competing_installer):
                with self.assertRaisesRegex(ValueError, "baseline changed"):
                    module.replace_checked(destination, staged, baseline, root / "race-backup")
            self.assertEqual(b"concurrent change", destination.read_bytes())
            self.assertEqual(b"baseline", (root / "race-backup").read_bytes())


if __name__ == "__main__":
    unittest.main()
