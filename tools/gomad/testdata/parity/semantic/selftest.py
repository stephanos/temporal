#!/usr/bin/env python3
"""Exercise the parity command at its subprocess boundary."""

import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


SCRIPT = Path(__file__).resolve().parents[3] / "scripts/behavioral-parity"
CORPUS = Path(__file__).resolve().parents[1]


def run(*args, code=0, contains="", env=None):
    result = subprocess.run([sys.executable, str(SCRIPT), *map(str, args)],
                            capture_output=True, text=True, env=env)
    assert result.returncode == code, (args, result.returncode, result.stdout, result.stderr)
    assert contains in result.stdout + result.stderr, (args, result.stdout, result.stderr)
    return result.stdout


def main():
    with tempfile.TemporaryDirectory(prefix="gomad-parity-selftest-") as scratch:
        root = Path(scratch)
        left = json.loads((CORPUS / "semantic/example.json").read_text())
        left_path, right_path = root / "left.json", root / "right.json"
        left_path.write_text(json.dumps(left))

        def compare(right, **kwargs):
            right_path.write_text(json.dumps(right))
            return run("--corpus", CORPUS, "--verb", "fixture", "--left", left_path,
                       "--right", right_path, **kwargs)

        compare(left, contains="match")
        right = copy.deepcopy(left)
        right["diagnostics"].update(campaign_id="campaign-other", run_id="run-other",
                                    timestamp="2099-01-01T00:00:00Z", elapsed_ns=998)
        right["diagnostics"]["output_path"] = "/right/artifacts/case/record.json"
        right["diagnostics"]["scratch_path"] = "/right/scratch/module.wasm"
        right["diagnostics"]["replay_command"] = ["gomad", "replay", "--json", "/right/artifacts/case"]
        right["roots"] = {"output": "/right/artifacts", "scratch": "/right/scratch"}
        compare(right, contains="match")
        mutations = [
            ("outcome", lambda d: d["executions"][0].update(outcome="failure"), "$.executions[0].outcome"),
            ("termination", lambda d: d["executions"][0].update(termination="trap"), "$.executions[0].termination"),
            ("order", lambda d: d["executions"].reverse(), "$.executions[0].execution"),
            ("divergence", lambda d: d["replay"].update(divergence_ordinal=4), "$.replay.divergence_ordinal"),
            ("reason", lambda d: d["replay"].update(reason="different"), "$.replay.reason"),
            ("bounds", lambda d: d["limits"].update(fuel=9), "$.limits.fuel"),
            ("identity", lambda d: d["executions"][0].update(execution_identity="b" * 64), "$.executions[0].execution_identity"),
            ("implementation", lambda d: d["executions"][0].update(implementation_identity="b" * 64), "$.executions[0].implementation_identity"),
            ("digest", lambda d: d["executions"][0].update(trace_sha256="b" * 64), "$.executions[0].trace_sha256"),
            ("record-hash", lambda d: d["executions"][0].update(record_sha256="b" * 64), "$.executions[0].record_sha256"),
            ("payload-hash", lambda d: d["executions"][0].update(payload_sha256="b" * 64), "$.executions[0].payload_sha256"),
            ("signature", lambda d: [e.update(failure_signature="changed") for e in d["executions"]], "$.executions[0].failure_signature"),
            ("seed", lambda d: d["executions"][0].update(seed=8), "$.executions[0].seed"),
            ("virtual-time", lambda d: d["executions"][0].update(virtual_time_ns=1), "$.executions[0].virtual_time_ns"),
            ("replay-match", lambda d: d["replay"].update(matched=False), "$.replay.matched"),
            ("partition", lambda d: d["executions"][1].update(failure_signature="other"), "$.executions[1].failure_signature"),
            ("reduction", lambda d: d["reductions"][0].update(removed_decision="other"), "$.reductions[0].removed_decision"),
            ("verdict", lambda d: d["qualification"][0].update(verdict="fail"), "$.qualification[0].verdict"),
            ("argument", lambda d: d["diagnostics"]["replay_command"].insert(2, "--choices"), "$.diagnostics.replay_command"),
            ("suffix", lambda d: d["diagnostics"].update(output_path="/left/artifacts/other/record.json"), "$.diagnostics.output_path"),
            ("unknown", lambda d: d.update(unknown=1), "$.unknown"),
            ("unknown-nested", lambda d: d["executions"][0].update(unknown=1), "$.executions[0].unknown"),
            ("integer-type", lambda d: d["limits"].update(fuel=True), "$.limits.fuel"),
            ("id-relationships", lambda d: d["diagnostics"].update(run_id="campaign-one"), "$.diagnostics.run_id"),
        ]
        for name, mutate, path in mutations:
            right = copy.deepcopy(left)
            mutate(right)
            compare(right, code=1, contains=path)
            print("PASS", name)
        run("--corpus", root / "missing", "--selftest", code=1, contains="corpus")
        shutil.copytree(CORPUS, root / "missing-entry")
        (root / "missing-entry/targets/foundation.json").unlink()
        run("--corpus", root / "missing-entry", "--selftest", code=1, contains="missing corpus/evidence entry")
        first = run("--corpus", CORPUS, "--selftest", contains="semantic corpus match")
        assert first
        env = dict(os.environ, GOMAD_GOLDEN_REGENERATE="1")
        run("--corpus", CORPUS, "--golden-dir", root / "goldens", "--selftest", env=env)
        before = {p.name: p.read_bytes() for p in (root / "goldens").iterdir()}
        run("--corpus", CORPUS, "--golden-dir", root / "goldens", "--selftest", env=env)
        assert before == {p.name: p.read_bytes() for p in (root / "goldens").iterdir()}
        integrity(root)
        env = dict(os.environ, GOMAD_GOLDEN_REGENERATE="1")
        driver = Path(__file__).resolve()
        for attempt in range(2):
            run("--corpus", CORPUS, "--verb", "fixture", "--go-driver", driver,
                "--rust-driver", driver, "--output-root", root / f"run-{attempt}",
                "--golden-dir", root / "run-goldens", env=env, contains="parity match")
            current = {p.name: p.read_bytes() for p in (root / "run-goldens").iterdir()}
            if attempt == 0:
                first = current
            else:
                assert first == current, "fresh repeated regeneration changed golden bytes"
        bad_env = dict(env, PARITY_SELFTEST_DIFFERENT="1")
        run("--corpus", CORPUS, "--verb", "fixture", "--go-driver", driver,
            "--rust-driver", driver, "--output-root", root / "different-run",
            "--golden-dir", root / "run-goldens", env=bad_env, code=1,
            contains="$.replay.reason")
        for setting, expected in (("PARITY_SELFTEST_CORRUPT", "raw payload size/digest"),
                                  ("PARITY_SELFTEST_NONREPEAT", "$.replay.reason")):
            run("--corpus", CORPUS, "--verb", "fixture", "--go-driver", driver,
                "--rust-driver", driver, "--output-root", root / setting,
                "--golden-dir", root / "run-goldens", env=dict(env, **{setting: "1"}),
                code=1, contains=expected)
        assert first == {p.name: p.read_bytes() for p in (root / "run-goldens").iterdir()}
    print("PASS subprocess parity cases")


def integrity(root):
    artifact = root / "raw/artifact"
    artifact.mkdir(parents=True)
    encode = lambda value: json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
    record = {"schema": "gomad.execution-record/v2", "seed": 7,
              "execution_identity": "a" * 64,
              "identity": dict.fromkeys(("target", "toolchain", "platform", "implementation", "configuration"), "a" * 64),
              "outcome": {"class": "success", "termination": "exit", "exit_code": 0, "reason": ""},
              "limits": dict.fromkeys(("fuel", "memory_bytes", "wall_millis", "output_bytes", "evidence_bytes"), 100),
              "profile": "fixture", "arguments": [], "environment": {}, "virtual_time_nanos": 0,
              "streams": {"payload": {"bytes": 3, "records": 1,
                          "sha256": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}}}
    record["execution_identity"] = hashlib.sha256(encode(record["identity"])).hexdigest()
    raw = encode(record)
    record_hash = hashlib.sha256(raw).hexdigest()
    (artifact / "record.json").write_bytes(raw)
    (artifact / "payload").write_bytes(b"abc")
    (artifact / "storage.json").write_text('{"format":"gomad.artifact-storage/v2"}')
    manifest = {"schema": "gomad.artifact-manifest/v2", "record": record,
                "record_sha256": record_hash,
                "files": {"payload": {"bytes": 3, "module": False,
                           "sha256": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}}}
    (artifact / "manifest.json").write_text(json.dumps(manifest))
    body = {"ordinal": 0, "previous_sha256": "0" * 64, "record_sha256": record_hash}
    row = dict(body, sha256=hashlib.sha256(encode(body)).hexdigest())
    ledger = root / "raw/ledger.jsonl"
    ledger.write_bytes(encode(row) + b"\n")
    result = {"schema": "gomad.foundation-result/v1", "case": "raw-test", "records": [{
        "record_file": "artifact/record.json", "record_sha256": record_hash,
        "artifact": "artifact", "outcome": "success", "termination": "exit", "seed": 7,
        "failure_signature": hashlib.sha256(encode(dict(record, seed=0))).hexdigest(),
        "execution_identity": record["execution_identity"]}], "ledger_file": "ledger.jsonl"}
    report = root / "raw/result.json"
    report.write_text(json.dumps(result))
    args = ("--corpus", CORPUS, "--foundation-root", root / "raw", "--foundation-result", report)
    run(*args)
    (artifact / "unlisted").write_bytes(b"not listed")
    run(*args, code=1, contains="unlisted")
    (artifact / "unlisted").unlink()
    (artifact / "record.json").write_bytes(encode(dict(record, seed=8)))
    run(*args, code=1, contains="$raw.records[0].record_sha256")
    (artifact / "record.json").write_bytes(b" " + raw)
    run(*args, code=1, contains="not canonical")
    (artifact / "record.json").write_bytes(raw)
    (artifact / "payload").write_bytes(b"abd")
    run(*args, code=1, contains="raw payload")
    (artifact / "payload").write_bytes(b"abc")
    (artifact / "target").mkdir()
    (artifact / "target/module.wasm").write_bytes(b"abc")
    manifest["files"]["target/module.wasm"] = {"bytes": 3, "sha256": hashlib.sha256(b"abc").hexdigest(), "module": False}
    (artifact / "manifest.json").write_bytes(encode(manifest))
    run(*args, code=1, contains="target/module.wasm.module")
    del manifest["files"]["target/module.wasm"]
    (artifact / "target/module.wasm").unlink()
    (artifact / "target").rmdir()
    (artifact / "manifest.json").write_bytes(encode(manifest))
    ledger.write_bytes(encode(dict(row, previous_sha256="1" * 64)) + b"\n")
    run(*args, code=1, contains="$raw.ledger[0].sha256")
    body["previous_sha256"] = "1" * 64
    ledger.write_bytes(encode(dict(body, sha256=hashlib.sha256(encode(body)).hexdigest())) + b"\n")
    run(*args, code=1, contains="$raw.ledger[0].previous_sha256")
    ledger.write_bytes(b"")
    run(*args, code=1, contains="count/bounds")
    for changed, path in ((dict(body, ordinal=1), ".ordinal"),
                          (dict(body, record_sha256="b" * 64), ".ordinal")):
        changed["previous_sha256"] = "0" * 64
        ledger.write_bytes(encode(dict(changed, sha256=hashlib.sha256(encode(changed)).hexdigest())) + b"\n")
        run(*args, code=1, contains=path)
    print("PASS raw integrity before projection")


def driver():
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--verb")
    parser.add_argument("--corpus", type=Path)
    parser.add_argument("--case")
    parser.add_argument("--artifact-root", type=Path)
    args = parser.parse_args()
    value = json.loads((args.corpus / "semantic/example.json").read_text())
    encode = lambda data: json.dumps(data, sort_keys=True, separators=(",", ":")).encode()
    sha = lambda data: hashlib.sha256(data).hexdigest()
    result = {"schema": "gomad.foundation-result/v1", "case": args.case,
              "records": [], "ledger_file": "ledger.jsonl"}
    rows = []
    previous = "0" * 64
    for ordinal, execution in enumerate(value["executions"]):
        name = f"artifact-{ordinal}"
        artifact = args.artifact_root / name
        artifact.mkdir()
        implementation = "b" * 64 if "rust" in args.artifact_root.parts else "a" * 64
        identity = dict.fromkeys(("target", "toolchain", "platform", "implementation", "configuration"), "a" * 64)
        identity["implementation"] = implementation
        identity["target"] = sha(b"abc")
        record = {"schema": "gomad.execution-record/v2", "seed": execution["seed"],
                  "execution_identity": sha(encode(identity)), "identity": identity,
                  "outcome": {"class": "success", "termination": "exit", "exit_code": 0, "reason": ""},
                  "limits": {"fuel": 100, "memory_bytes": 65536, "wall_millis": 100,
                             "output_bytes": 100, "evidence_bytes": 100},
                  "profile": "fixture", "arguments": [], "environment": {},
                  "virtual_time_nanos": 0, "streams": {"trace": {"bytes": 5,
                       "records": 1, "sha256": sha(b"trace")}}}
        raw = encode(record)
        record_hash = sha(raw)
        signature = sha(encode(dict(record, seed=0)))
        (artifact / "record.json").write_bytes(raw)
        (artifact / "target").mkdir()
        (artifact / "target/module.wasm").write_bytes(b"abc")
        (artifact / "trace").write_bytes(b"trace")
        (artifact / "storage.json").write_bytes(encode({"format": "gomad.artifact-storage/v2"}))
        (artifact / "manifest.json").write_bytes(encode({"schema": "gomad.artifact-manifest/v2",
            "record": record, "record_sha256": record_hash,
            "files": {"record.json": {"bytes": len(raw), "sha256": record_hash, "module": False},
                      "target/module.wasm": {"bytes": 3, "sha256": sha(b"abc"), "module": True},
                      "trace": {"bytes": 5, "sha256": sha(b"trace"), "module": False}}}))
        result["records"].append({"record_file": name + "/record.json", "record_sha256": record_hash,
            "artifact": name, "outcome": "success", "termination": "exit", "seed": execution["seed"],
            "failure_signature": signature, "execution_identity": record["execution_identity"]})
        execution.update(record_sha256=record_hash, execution_identity=record["execution_identity"],
                         implementation_identity=implementation, failure_signature=signature,
                         payload_file="target/module.wasm", payload_sha256=sha(b"abc"), trace_sha256=sha(b"trace"))
        body = {"ordinal": ordinal, "record_sha256": record_hash, "previous_sha256": previous}
        previous = sha(encode(body))
        rows.append(encode(dict(body, sha256=previous)))
    (args.artifact_root / "foundation-result.json").write_bytes(encode(result))
    (args.artifact_root / "ledger.jsonl").write_bytes(b"\n".join(rows) + b"\n")
    if os.environ.get("PARITY_SELFTEST_DIFFERENT") and "rust" in args.artifact_root.parts:
        value["replay"]["reason"] = "intentionally different"
    if os.environ.get("PARITY_SELFTEST_NONREPEAT") and args.artifact_root.name == "1":
        value["replay"]["reason"] = "unstable observation"
    if os.environ.get("PARITY_SELFTEST_CORRUPT") and "rust" in args.artifact_root.parts:
        (args.artifact_root / "artifact-0/target/module.wasm").write_bytes(b"corrupt")
    print(encode(value).decode())


if __name__ == "__main__":
    if "--artifact-root" in sys.argv:
        driver()
    else:
        main()
