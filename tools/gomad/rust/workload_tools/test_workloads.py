import importlib.util
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
