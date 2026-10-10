import copy
import gzip
import hashlib
import json
import pathlib
import subprocess
import sys
import tempfile
import unittest

import replay


class ReplayTests(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory()
        self.addCleanup(self.scratch.cleanup)
        self.root = pathlib.Path(self.scratch.name)
        self.records = [
            {"kind": "header", "owner": "choice", "subject_kind": "machine",
             "artifact": "fixture.json", "states": ["a", "b"], "actions": ["go"],
             "outcomes": ["ok", "no"], "facts": ["first", "second"],
             "starts": ["a"], "ends": ["b"], "declaration": {"steps": ["go", "wait"]}},
            {"kind": "binding", "index": 0, "declaration": "go"},
            {"kind": "binding", "index": 1, "declaration": "wait"},
            {"kind": "row", "source": "a", "action": "go", "results": [
                {"outcome": "ok", "state": "b", "facts": ["first", "second"], "because": "why"},
                {"outcome": "no", "state": "a", "facts": [], "because": "held"}]},
            {"kind": "disabled", "pair": [1, 0]},
            {"kind": "footer", "status": "complete", "states": 2, "actions": 1,
             "pairs": 2, "enabled": 1, "disabled": 1, "unknown": 0, "results": 2},
        ]
        self.pin = {"artifact": "fixture.json", "kind": "machine", "owner": "choice",
                    "states": 2, "actions": 1, "state_action_pairs": 2,
                    "enabled_pairs": 1, "disabled_pairs": 1, "unknown_pairs": 0,
                    "results": 2, "lines": len(self.records), "status": "complete",
                    "uncompressed_jsonl_sha256": hashlib.sha256(self.encode(self.records)).hexdigest()}

    @staticmethod
    def encode(records):
        return b"".join((json.dumps(item, separators=(",", ":")) + "\n").encode() for item in records)

    def write(self, records):
        path = self.root / "machine-choice.jsonl.gz"
        with gzip.open(path, "wb") as output:
            output.write(self.encode(records))
        return path

    def test_full_stream_accepts_exact_pin(self):
        self.assertEqual(replay.verify_stream(self.write(self.records), self.pin)["records"], 6)

    def test_missing_and_extra_machine_rejected(self):
        expected = {("machine", "choice")}
        for actual in (set(), expected | {("machine", "extra")}):
            with self.subTest(actual=actual), self.assertRaisesRegex(ValueError, "subject universe"):
                replay.require_universe(expected, actual, "fixture")

    def test_named_mutants_rejected(self):
        def row(records):
            records[3]["source"] = "b"
        def result(records):
            records[3]["results"][0]["outcome"] = "no"
        def result_order(records):
            records[3]["results"].reverse()
        def fact_order(records):
            records[3]["results"][0]["facts"].reverse()
        def binding_order(records):
            records[1], records[2] = records[2], records[1]
        def because(records):
            records[3]["results"][0]["because"] = "other"
        def omitted_pair(records):
            del records[4]
        for name, mutate in (("row", row), ("result", result), ("result-order", result_order),
                             ("fact-order", fact_order), ("binding-order", binding_order),
                             ("explanation", because), ("omitted-pair", omitted_pair)):
            with self.subTest(mutant=name):
                records = copy.deepcopy(self.records)
                mutate(records)
                with self.assertRaises(ValueError):
                    replay.verify_stream(self.write(records), self.pin)

    def test_incomplete_receipt_never_passes(self):
        pin = dict(self.pin, status="unavailable; NOT PASS")
        with self.assertRaisesRegex(ValueError, "unavailable"):
            replay.verify_stream(self.write(self.records), pin)

    def test_missing_or_changed_seam_is_inconclusive_not_skipped(self):
        for dumper, error in ((self.root / "missing-dumper", "inherited dumper missing"),
                              (pathlib.Path(__file__), "inherited dumper identity changed")):
            with self.subTest(dumper=dumper):
                receipt = self.root / "receipt.json"
                candidate = self.root / "candidate"
                process = subprocess.run([
                    sys.executable, replay.__file__, "--ir", str(self.root),
                    "--dumper", str(dumper), "--out", str(candidate),
                    "--receipt", str(receipt),
                ], capture_output=True, text=True)
                self.assertEqual(process.returncode, 1)
                self.assertEqual(json.loads(receipt.read_text())["status"], "inconclusive")
                self.assertIn(error, process.stderr)
                self.assertFalse(candidate.exists())

    def dumps(self, name, records_by_artifact):
        folder = self.root / name
        for artifact, records in records_by_artifact.items():
            (folder / artifact).mkdir(parents=True)
            with gzip.open(folder / artifact / "machine-choice.jsonl.gz", "wb") as output:
                output.write(self.encode(records))
        return folder

    def test_equivalence_under_a_rename_map(self):
        base = self.dumps("base", {"fixture-race": self.records, "fixture": self.records})
        renamed = json.loads(json.dumps(self.records).replace('"go"', '"start"'))
        renamed[0]["declaration"]["position"] = {"file": "moved.scala", "line": 9}
        now = self.dumps("now", {"fixture": renamed})
        result = replay.compare_streams(base, now, replay.Renames("go start\n"))
        self.assertEqual(result["differences"], [])
        self.assertEqual(result["used"], ["go"])
        unmapped = replay.compare_streams(base, now, replay.Renames(""))
        self.assertTrue(any("line 4 differs" in d for d in unmapped["differences"]), unmapped)

    def test_equivalence_rejects_mutants_and_missing_or_extra_subjects(self):
        base = self.dumps("base", {"fixture": self.records})
        changed = copy.deepcopy(self.records)
        changed[3]["results"][0]["facts"].reverse()
        result = replay.compare_streams(base, self.dumps("fact", {"fixture": changed}), replay.Renames(""))
        self.assertTrue(result["differences"])
        missing = replay.compare_streams(base, self.dumps("other", {"fixture": []}), replay.Renames(""))
        self.assertTrue(any("is gone" in d for d in missing["differences"]) or missing["differences"])


if __name__ == "__main__":
    unittest.main()
