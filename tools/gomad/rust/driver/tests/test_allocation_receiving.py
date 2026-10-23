import copy
import hashlib
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from allocation_receiving import derive


def row(case, inputs, allocated, observed):
    return {"schema": "gomad.native-allocation-observation/v1", "case": case,
            "inputs": inputs, "observed": observed, "allocator": {
                "allocated_bytes": allocated, "allocation_calls": 1,
                "live_requested_bytes_before": 100,
                "live_requested_bytes_after": 100+allocated,
                "peak_live_requested_bytes": 100+allocated}}


def host_rows():
    digest = "sha256:"+hashlib.sha256(b"\xa5"*(8 << 20)).hexdigest()
    return [
        row("allocator-control", {"bytes": 8192}, 8192, {"length": 8192, "last": 165}),
        row("bounded-file-read", {"file_bytes": 1 << 20, "limit_bytes": 512 << 20}, 1 << 20,
            {"length": 1 << 20, "sha256": "sha256:"+hashlib.sha256(b"x"*(1 << 20)).hexdigest()}),
        row("opened-payload", {"payload_bytes": 8 << 20, "limit_bytes": 8 << 20}, 8 << 20,
            {"length": 8 << 20, "sha256": digest}),
        row("retained-decode", {"payload_bytes": 8 << 20}, 56 << 20,
            {"length": 8 << 20, "sha256": digest, "schema": "gomad.cluster-replay-inputs/v1", "descriptor_sha256": digest}),
    ]


class AllocationReceivingTests(unittest.TestCase):
    def test_exact_original_byte_ceiling_accepts_boundary_and_rejects_one_over(self):
        rows = host_rows()
        self.assertEqual(derive(rows, "gomad")["predicates"][-1]["actual"], 56 << 20)
        rows[-1]["allocator"]["allocated_bytes"] += 1
        with self.assertRaisesRegex(ValueError, "original byte ceiling"):
            derive(rows, "gomad")

    def test_allocation_call_count_cannot_substitute_for_byte_ceiling(self):
        rows = host_rows()
        rows[-1]["allocator"]["allocated_bytes"] = 57 << 20
        rows[-1]["allocator"]["allocation_calls"] = 1
        with self.assertRaisesRegex(ValueError, "original byte ceiling"):
            derive(rows, "gomad")

    def test_missing_or_duplicate_rows_cannot_look_complete(self):
        rows = host_rows()
        with self.assertRaisesRegex(ValueError, "missing allocation rows"):
            derive(rows[:-1], "gomad")
        with self.assertRaisesRegex(ValueError, "duplicate allocation row"):
            derive([*rows, copy.deepcopy(rows[-1])], "gomad")

    def test_reported_low_allocation_does_not_replace_retained_bytes(self):
        rows = host_rows()
        rows[-1]["observed"]["sha256"] = "sha256:"+"a"*64
        with self.assertRaisesRegex(ValueError, "read changed retained bytes"):
            derive(rows, "gomad")

    def test_allocator_control_is_required_and_typed(self):
        rows = host_rows()
        rows[0]["allocator"]["allocation_calls"] = True
        with self.assertRaisesRegex(ValueError, "allocator integer"):
            derive(rows, "gomad")


if __name__ == "__main__":
    unittest.main()
