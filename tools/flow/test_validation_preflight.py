"""Exercise make-pr's real close/commit fence without network or PR writes."""

import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


FLOWCTL = Path(os.environ.get("FLOWCTL_PY", str(Path.home() / ".codex/scripts/flowctl.py"))).resolve()
PREFLIGHT = Path(os.environ.get("MAKE_PR_PREFLIGHT", str(Path.home() / ".codex/scripts/make-pr-preflight.sh"))).resolve()


class ValidationPreflight(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="flow-validation-preflight-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.trace = self.root / "trace.jsonl"
        self.env = dict(os.environ, BASH_ENV="/dev/null", FLOW_RALPH="0")
        for key in ("FLOW_STATE_DIR", "REVIEW_RECEIPT_PATH"):
            self.env.pop(key, None)
        self.env.update(TRACE=str(self.trace), REAL_GIT=shutil.which("git"),
                        REAL_PYTHON=sys.executable, TEST_FLOWCTL=str(FLOWCTL),
                        EXISTING_PR="0")
        self.git("init", "-b", "main")
        self.git("config", "user.name", "Flow Test")
        self.git("config", "user.email", "flow-test@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.flow("init", "--json")
        self.spec = self.flow("spec", "create", "--title", "Validation", "--json")["id"]
        self.task = self.flow("task", "create", "--spec", self.spec,
                              "--title", "Done fixture", "--json")["id"]
        self.spec_path = self.repo / ".flow/specs" / (self.spec + ".json")
        self.task_path = self.repo / ".flow/tasks" / (self.task + ".json")
        task = json.loads(self.task_path.read_text())
        task["status"] = "done"
        self.task_path.write_text(json.dumps(task) + "\n")
        runtime = self.repo / ".git/flow-state/tasks" / (self.task + ".state.json")
        if runtime.exists():
            runtime.write_text(json.dumps({"status": "done"}) + "\n")
        spec = json.loads(self.spec_path.read_text())
        spec["branch_name"] = "work"
        self.spec_path.write_text(json.dumps(spec) + "\n")
        self.set_policy(True)
        self.git("add", "-A")
        self.git("commit", "-m", "Fixture")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")
        self.git("checkout", "-b", "work")
        (self.repo / "work.txt").write_text("Implementation\n")
        self.git("add", "-A")
        self.git("commit", "-m", "Implementation")
        self.install_stubs()
        self.env["PATH"] = str(self.bin) + os.pathsep + self.env["PATH"]
        self.env.update(REPO_ROOT=str(self.repo), FLOWCTL=str(self.bin / "flowctl"),
                        SPEC_ID=self.spec, BASE_REF="main", DRY_RUN="0",
                        UPDATE_MODE="0", AUTONOMOUS="1", TEST_PREFLIGHT=str(PREFLIGHT))

    def run_command(self, command, **kwargs):
        return subprocess.run(command, cwd=self.repo, env=self.env, capture_output=True,
                              text=True, **kwargs)

    def git(self, *args):
        return self.run_command(["git", *args], check=True).stdout

    def flow(self, *args):
        result = self.run_command([sys.executable, str(FLOWCTL), *args], check=True)
        return json.loads(result.stdout)

    def set_policy(self, value):
        path = self.repo / ".flow/config.json"
        config = json.loads(path.read_text())
        config["specClose"] = {"validationReviewRequired": value}
        path.write_text(json.dumps(config) + "\n")

    def report(self):
        path = self.repo / ".flow/artifacts" / self.spec / "validation-cost-review.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps({
            "schema_version": 1, "spec_id": self.spec,
            "gates": [{"command": "python3 focused.py", "mode": "isolated",
                       "cost": {"kind": "estimated", "seconds": 1},
                       "coverage": "R1-R4", "evidence": []}],
            "speedup_decisions": [{"opportunity": "duplicate gates", "decision": "retain",
                                   "rationale": "focused checks cover distinct contracts",
                                   "coverage_preserved": "all refusal and success controls retained"}],
        }, indent=2) + "\n")
        return path

    def install_stubs(self):
        stub = '''#!{python}
import json
import os
from pathlib import Path
import sys
name = Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ["TRACE"], "a") as out:
    out.write(json.dumps([name, *args]) + "\\n")
if name == "git":
    if "fetch" in args:
        sys.exit(0)
    if any(word in args for word in ("push", "ls-remote", "clone")):
        sys.exit(91)
    os.execv(os.environ["REAL_GIT"], [os.environ["REAL_GIT"], *args])
if name == "flowctl":
    os.execv(os.environ["REAL_PYTHON"], [os.environ["REAL_PYTHON"], os.environ["TEST_FLOWCTL"], *args])
if name == "gh":
    if args[:2] == ["auth", "status"]:
        sys.exit(0)
    if args[:2] == ["pr", "view"]:
        if os.environ["EXISTING_PR"] == "1":
            print(json.dumps({{"state": "OPEN", "url": "https://example.invalid/pr/1", "number": 1}}))
        else:
            print("null")
        sys.exit(0)
    if args[:2] == ["pr", "create"]:
        sys.exit(0)
sys.exit(92)
'''.format(python=sys.executable)
        for name in ("git", "flowctl", "gh", "codex", "claude", "cursor-agent",
                     "cargo", "go", "make", "pytest", "curl", "wget"):
            path = self.bin / name
            path.write_text(stub)
            path.chmod(0o755)

    def preflight(self):
        self.trace.write_text("")
        return self.run_command(["bash", "-c", 'source "$TEST_PREFLIGHT"\ngh pr create\n'])

    def calls(self, executable):
        return [row[1:] for row in map(json.loads, self.trace.read_text().splitlines())
                if row[0] == executable]

    def assert_no_extra_work(self, dry_run=False):
        rows = list(map(json.loads, self.trace.read_text().splitlines()))
        self.assertTrue(all(row[0] in ("flowctl", "git", "gh") for row in rows), rows)
        git = self.calls("git")
        fetches = [args for args in git if "fetch" in args]
        self.assertEqual(0 if dry_run else 1, len(fetches), fetches)
        self.assertTrue(all("push" not in args and "ls-remote" not in args for args in git))
        for args in self.calls("flowctl"):
            self.assertTrue(args[:1] == ["show"] or args[:2] in (
                ["spec", "close"], ["spec", "set-branch"]), args)

    def test_enabled_forwards_uncommitted_report_and_commits_exact_artifact(self):
        report = self.report()
        raw = report.read_bytes()
        head = self.git("rev-parse", "HEAD").strip()
        result = self.preflight()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)
        closes = [args for args in self.calls("flowctl") if args[:2] == ["spec", "close"]]
        self.assertEqual(1, len(closes))
        flag = closes[0].index("--validation-review-file")
        self.assertEqual(report.resolve(), (self.repo / closes[0][flag + 1]).resolve())
        spec = json.loads(self.spec_path.read_text())
        self.assertEqual("done", spec["status"])
        reference = spec["validation_review"]
        self.assertEqual(hashlib.sha256(raw).hexdigest(), reference["sha256"])
        committed = self.run_command(["git", "show", "HEAD:" + reference["path"]], check=True)
        self.assertEqual(raw, committed.stdout.encode())
        self.assertEqual("1", self.git("rev-list", "--count", head + "..HEAD").strip())
        self.assertEqual(["pr", "create"], self.calls("gh")[-1])
        self.assert_no_extra_work()

    def test_retry_commits_previously_published_untracked_artifact(self):
        report = self.report()
        raw = report.read_bytes()
        head = self.git("rev-parse", "HEAD").strip()
        lifecycle = [self.spec_path.read_bytes(), self.task_path.read_bytes()]
        name = "flowctl_preflight_retry_test"
        imported = importlib.util.spec_from_file_location(name, FLOWCTL)
        module = importlib.util.module_from_spec(imported)
        sys.modules[name] = module
        self.addCleanup(sys.modules.pop, name, None)
        imported.loader.exec_module(module)
        real_write = module.atomic_write_json

        def fail_before_lifecycle(path, data):
            if path == self.spec_path:
                raise OSError("injected lifecycle failure after artifact publication")
            return real_write(path, data)

        argv = [str(FLOWCTL), "spec", "close", self.spec,
                "--validation-review-file", str(report), "--json"]
        with patch.object(module, "get_repo_root", return_value=self.repo), \
             patch.object(module, "get_flow_dir", return_value=self.repo / ".flow"), \
             patch.object(module, "get_state_dir", return_value=self.repo / ".git/flow-state"), \
             patch.object(module, "atomic_write_json", side_effect=fail_before_lifecycle), \
             patch.object(sys, "argv", argv):
            with self.assertRaisesRegex(OSError, "injected lifecycle failure"):
                module.main()
        self.assertEqual(lifecycle, [self.spec_path.read_bytes(), self.task_path.read_bytes()])
        self.assertEqual(head, self.git("rev-parse", "HEAD").strip())
        artifact = report.with_name("validation-cost-review-" + hashlib.sha256(raw).hexdigest() + ".json")
        self.assertEqual(raw, artifact.read_bytes())
        relative = artifact.relative_to(self.repo).as_posix()
        self.assertEqual("?? " + relative, self.git("status", "--porcelain", "--untracked-files=all", "--", relative).strip())
        result = self.preflight()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)
        reference = json.loads(self.spec_path.read_text())["validation_review"]
        self.assertEqual(relative, reference["path"])
        committed = self.run_command(["git", "show", "HEAD:" + reference["path"]], check=True)
        self.assertEqual(raw, committed.stdout.encode())
        self.assertEqual("1", self.git("rev-list", "--count", head + "..HEAD").strip())
        self.assert_no_extra_work()

    def test_bad_or_missing_report_stops_before_staging_commit_or_pr(self):
        spec = json.loads(self.spec_path.read_text())
        spec["branch_name"] = "previous-branch"
        self.spec_path.write_text(json.dumps(spec) + "\n")
        self.git("add", "-A")
        self.git("commit", "-m", "Different branch fixture")
        for invalid in ("missing", "malformed", "config-type", "config-json"):
            with self.subTest(invalid=invalid):
                self.set_policy(True)
                if invalid == "malformed":
                    self.report().write_text("{invalid")
                elif invalid == "config-type":
                    self.set_policy("true")
                elif invalid == "config-json":
                    (self.repo / ".flow/config.json").write_text("{invalid")
                head = self.git("rev-parse", "HEAD")
                lifecycle = [self.spec_path.read_bytes(), self.task_path.read_bytes()]
                result = self.preflight()
                self.assertNotEqual(0, result.returncode)
                self.assertRegex((result.stdout + result.stderr).lower(), "validation|config")
                self.assertEqual(head, self.git("rev-parse", "HEAD"))
                self.assertEqual(lifecycle, [self.spec_path.read_bytes(), self.task_path.read_bytes()])
                self.assertFalse(any("add" in row or "commit" in row for row in self.calls("git")))
                self.assertNotIn(["pr", "create"], self.calls("gh"))
                self.assertFalse(any(row[:2] == ["spec", "set-branch"] for row in self.calls("flowctl")))
                self.assert_no_extra_work()

    def test_existing_skip_branches_do_not_require_or_forward_report(self):
        original_spec = self.spec_path.read_bytes()
        for mode in ("opt-out", "done", "dry-run", "body-only"):
            with self.subTest(mode=mode):
                self.spec_path.write_bytes(original_spec)
                self.set_policy(mode != "opt-out")
                if mode == "done":
                    spec = json.loads(self.spec_path.read_text())
                    spec["status"] = "done"
                    spec["validation_review"] = {"path": "prior", "sha256": "historical"}
                    self.spec_path.write_text(json.dumps(spec) + "\n")
                self.git("add", "-A")
                self.git("commit", "--allow-empty", "-m", "Set mode " + mode)
                self.env["DRY_RUN"] = "1" if mode == "dry-run" else "0"
                self.env["UPDATE_MODE"] = "1" if mode == "body-only" else "0"
                self.env["EXISTING_PR"] = self.env["UPDATE_MODE"]
                result = self.preflight()
                self.assertEqual(0, result.returncode, result.stdout + result.stderr)
                closes = [args for args in self.calls("flowctl") if args[:2] == ["spec", "close"]]
                self.assertEqual(1 if mode == "opt-out" else 0, len(closes))
                self.assertTrue(all("--validation-review-file" not in args for args in closes))
                if mode == "done":
                    self.assertEqual({"path": "prior", "sha256": "historical"},
                                     json.loads(self.spec_path.read_text())["validation_review"])
                self.assert_no_extra_work(dry_run=mode == "dry-run")


if __name__ == "__main__":
    unittest.main()
