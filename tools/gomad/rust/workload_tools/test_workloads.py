import importlib.util
import sys
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("mandatory_workloads", Path(__file__).with_name("workloads.py"))
workloads = importlib.util.module_from_spec(spec)
spec.loader.exec_module(workloads)


class WorkloadTests(unittest.TestCase):
    def test_sqlite_original_three_phases_and_storage_bounds(self):
        value = workloads.sqlite_scenario(False)
        self.assertEqual([boot["args"] for boot in value["boots"]], [["setup-live", "WAL"], ["append-crash", "WAL"], ["recover", "WAL"]])
        self.assertEqual(value["workload"], {"kind": "sqlite-durable-recovery", "selection": None})
        self.assertEqual([row["action"] for row in value["actions"]], ["start", "wait", "restart", "crash", "restart", "wait"])
        canary = workloads.sqlite_scenario(True)
        self.assertEqual(canary["boots"][1]["args"], ["unsynced", "WAL"])
        self.assertEqual(canary["actions"][3]["action"], "wait")

    def test_replication_preserves_topology_faults_and_original_selection(self):
        value = workloads.temporal_replication_scenario(False)
        self.assertEqual([(link["from"], link["to"], link["enabled"]) for link in value["spec"]["links"]], [("active", "standby", True), ("standby", "active", True)])
        self.assertEqual([fault["AtNanos"] for fault in value["network_faults"]], [1, 30_000_000_000])
        self.assertEqual(value["boots"][0]["args"], ["-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"])

    def test_recovery_and_retry_preserve_original_storage_and_network_domains(self):
        recovery=workloads.temporal_recovery_scenario()
        self.assertEqual(recovery["workload"],dict(kind="temporal-committed-workflow-crash-reopen",forcing=None))
        self.assertEqual([row["action"] for row in recovery["actions"]],["start","crash","restart","wait","restart","wait"])
        self.assertEqual(recovery["boots"][0]["args"][3],"-test.run=^TestCommittedWorkflowCrashRecovery$")
        self.assertNotIn("network_v2",workloads.policy("recovery",{"schema":"authenticated-test-input"}))
        policy=workloads.support_policy("retry",{"schema":"authenticated-test-input"})
        self.assertEqual(policy["network_v2"]["faults"],[dict(name="rpc-reset",action="reset",at_nanos=40_000_000_000)])
        self.assertEqual(policy["Limits"]["TranscriptBytes"],1<<30)

    def test_support_rows_keep_exact_selection_argument_order_and_nil_network(self):
        for row,selection,package in [("smoke","^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$","./tests"),("frontend","^TestFrontendSystemInfo$","./tests/gomadfunctional")]:
            with tempfile.TemporaryDirectory() as root:
                spec=workloads.support_spec("/candidate",root,row)
            target=spec["Campaign"]["Target"]
            self.assertEqual(target["Source"],package)
            self.assertEqual(target["Args"],["-test.v","-test.count=1","-test.timeout=72h","-test.run="+selection,"-persistenceType=sql","-persistenceDriver=sqlite"])
            self.assertNotIn("network_v2",workloads.support_policy(row,{"schema":"authenticated-test-input"}))
            self.assertEqual(spec["Campaign"]["ChoiceTraceLimit"],64<<20)

    def test_staged_source_working_directory_is_explicit_and_other_defaults_stay_original(self):
        for factory, row in [(workloads.qualification_spec, "recovery"), (workloads.support_spec, "retry")]:
            value = factory("/candidate", "/producer/cell", row, source="./.tmp/wrapper",
                            source_working_dir="/producer/source-module")
            self.assertEqual(value["Campaign"]["Target"]["WorkingDir"], "/producer/source-module")
            self.assertEqual(value["Campaign"]["Target"]["Source"], "./.tmp/wrapper")
            default = factory("/candidate", "/producer/cell", row, source="./.tmp/wrapper")
            self.assertEqual(default["Campaign"]["Target"]["WorkingDir"], "/candidate")

    def test_support_requires_actual_selected_leaf_pass(self):
        sys.path.insert(0, str(Path(__file__).parent))
        module_spec = importlib.util.spec_from_file_location("temporal_support", Path(__file__).with_name("temporal-support.py"))
        support = importlib.util.module_from_spec(module_spec)
        module_spec.loader.exec_module(support)
        support.selected_pass(b"    --- PASS: TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds (0.01s)\n", "smoke")
        for raw in [b"--- PASS: TestWorkflowTypeEncodingSuite\n--- SKIP: TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds\n", b"--- PASS: TestWorkflowTypeEncodingSuite\nTestPlainASCII/Succeeds", b"--- PASS: TestWorkflowTypeEncodingSuite/TestPlainASCII/SucceedsOther (0.01s)"]:
            with self.subTest(raw=raw), self.assertRaises(ValueError):
                support.selected_pass(raw, "smoke")

    def test_staging_rejects_changed_original_and_unexpected_files(self):
        repo = Path(__file__).resolve().parents[4]
        with tempfile.TemporaryDirectory() as temporary:
            target = Path(temporary) / "broker"
            workloads.stage_temporal(repo, target, "replication")
            self.assertEqual(sorted(path.name for path in target.iterdir()), ["recovery_test.go", "replication_test.go"])
            (target / "recovery_test.go").write_text("changed")
            with self.assertRaises(ValueError):
                workloads.stage_temporal(repo, target, "replication")


if __name__ == "__main__":
    unittest.main()
