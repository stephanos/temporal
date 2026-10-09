"""Exercise the installed Flow CLI against unpublished local Git histories."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


FLOWCTL = Path(os.environ.get("FLOWCTL_PY", str(Path.home() / ".codex/scripts/flowctl.py")))


class CommittedDependencies(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="flow-committed-dependency-")
        self.addCleanup(self.scratch.cleanup)
        self.repo = Path(self.scratch.name)
        self.env = dict(os.environ, BASH_ENV="/dev/null")
        self.env.pop("FLOW_STATE_DIR", None)
        self.git("init", "-b", "main")
        self.git("config", "user.name", "Flow Test")
        self.git("config", "user.email", "flow-test@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.flow("init", "--json")
        self.parent = self.flow("spec", "create", "--title", "Parent", "--json")["id"]
        self.child = self.flow("spec", "create", "--title", "Child", "--json")["id"]
        self.flow("task", "create", "--spec", self.parent, "--title", "Parent task", "--json")
        self.task = self.flow("task", "create", "--spec", self.child, "--title", "Child task", "--json")["id"]
        self.parent_path = self.repo / ".flow/specs" / (self.parent + ".json")
        child_path = self.repo / ".flow/specs" / (self.child + ".json")
        child = json.loads(child_path.read_text())
        child["depends_on_epics"] = [self.parent]
        child_path.write_text(json.dumps(child) + "\n")
        self.git("add", "-A")
        self.git("commit", "-m", "Initial open specs")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")
        self.git("checkout", "-b", "work")

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.repo, env=self.env,
                              check=True, capture_output=True, text=True).stdout

    def flow(self, *args):
        result = subprocess.run([sys.executable, str(FLOWCTL), *args], cwd=self.repo,
                                env=self.env, check=True, capture_output=True, text=True)
        return json.loads(result.stdout)

    def close_parent(self):
        parent = json.loads(self.parent_path.read_text())
        parent["status"] = "done"
        self.parent_path.write_text(json.dumps(parent) + "\n")

    def commit_close(self):
        self.close_parent()
        self.git("add", "-A")
        self.git("commit", "-m", "Close parent locally")

    def test_committed_unpublished_close_admits_child(self):
        self.commit_close()
        self.git("branch", self.parent)
        self.assertTrue(self.flow("spec", "chain", self.child, "--json")["eligible"])
        ready = self.flow("ready", "--spec", self.child, "--json")
        self.assertEqual([self.task], [task["id"] for task in ready["ready"]])

    def test_deleted_parent_branch_does_not_require_publication(self):
        self.commit_close()
        self.assertTrue(self.flow("spec", "chain", self.child, "--json")["eligible"])

    def test_uncommitted_close_does_not_admit_child(self):
        self.close_parent()
        self.assertFalse(self.flow("spec", "chain", self.child, "--json")["eligible"])
        self.assertEqual([], self.flow("ready", "--spec", self.child, "--json")["ready"])

    def test_close_on_unmerged_branch_does_not_admit_child(self):
        self.git("checkout", "-b", self.parent)
        self.commit_close()
        self.git("checkout", "work")
        self.close_parent()
        self.assertFalse(self.flow("spec", "chain", self.child, "--json")["eligible"])


if __name__ == "__main__":
    unittest.main()
