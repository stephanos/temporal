#!/usr/bin/env python3
"""Exercise explicit compiled replay roles and independently check retained diagnostics."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f"{label}: got {actual!r}, expected {expected!r}")


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("ordinary-binary", "stress-binary", "determinism-binary", "output"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--release-binary", type=Path)
    args = parser.parse_args()
    args.output = args.output.resolve()
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    corpus = Path(__file__).resolve().parent
    commands = []
    binaries = {"ordinary": args.ordinary_binary.resolve(), "stress-jitter": args.stress_binary.resolve(),
                "determinism-fixture": args.determinism_binary.resolve()}
    binary_hashes = {role: sha(binary.read_bytes()) for role, binary in binaries.items()}

    def run(name, binary, arguments, status=0, entered=True, ambient=False):
        marker = args.output / (name + "-guest-start")
        environment = {key: value for key, value in os.environ.items() if not key.startswith("GOMAD_")}
        environment["GOMAD_FIXTURE_GUEST_START_FILE"] = str(marker)
        if ambient:
            environment.update({"GOMAD_WASM_DETERMINISM_FAULT": "output-clock",
                                "GOMAD_WASM_STRESS_JITTER": "1", "GOMAD_WASM_STRESS_LAYOUT": "ambient-padding",
                                "GOMAD_WASM_EVENT_ORDER": "0"})
        result = subprocess.run([str(binary), *map(str, arguments)], capture_output=True,
                                timeout=120, env=environment)
        (args.output / (name + ".stdout")).write_bytes(result.stdout)
        (args.output / (name + ".stderr")).write_bytes(result.stderr)
        commands.append({"name": name, "command": [str(binary), *map(str, arguments)],
                         "status": result.returncode, "expected_status": status,
                         "guest_entered": marker.exists(), "expected_guest_entered": entered,
                         "ambient_controls": ambient, "stdout_sha256": sha(result.stdout),
                         "stderr_sha256": sha(result.stderr)})
        (args.output / "commands.json").write_text(json.dumps(commands, indent=2) + "\n")
        equal(result.returncode, status, name + ".status")
        equal(marker.exists(), entered, name + ".guest_entered")
        if entered:
            equal(marker.read_bytes(), b"linked execution entered\n", name + ".guest_marker")
        else:
            equal(result.stdout, b"", name + ".refusal_stdout")
        return result

    def authenticate(root, raw, role):
        stored = (root / "role-result.json").read_bytes()
        report = json.loads(stored)
        equal(stored, canonical(report), role + ".canonical_report")
        equal(raw, stored + b"\n", role + ".stdout_report")
        equal(set(report), {"schema", "role", "configuration", "helper_sha256", "module_sha256", "executions"}, role + ".fields")
        equal(report["schema"], "gomad.replay-role-fixture/v1", role + ".schema")
        equal(report["role"], role, role + ".selection")
        suffix = "" if role == "ordinary" else "-" + role
        equal(report["configuration"], "cranelift-fuel-nan-canonical-nounwind-epoch-v1" + suffix, role + ".configuration")
        equal(report["module_sha256"], sha((corpus / "targets/inspect-choice.wat").read_bytes()), role + ".module")
        equal([row["name"] for row in report["executions"]],
              ["first", "second", "concurrent-left", "concurrent-right"], role + ".executions")
        first = report["executions"][0]
        for row in report["executions"]:
            label = role + "." + row["name"]
            equal((row["termination"], row["exit_code"]), ("exit", 0), label + ".outcome")
            equal(row["evidence"], first["evidence"], label + ".canonical_evidence")
            equal(row["event_order"], first["event_order"], label + ".event_order")
            if role == "ordinary":
                equal(row["event_order"], None, label + ".no_diagnostic_role")
                continue
            data = (root / (row["name"] + "-event-order.log")).read_bytes()
            equal(sha(data), row["event_order"]["sha256"], label + ".diagnostic_hash")
            equal(len(data), row["event_order"]["bytes"], label + ".diagnostic_bytes")
            lines = data.splitlines(keepends=True)
            frames = [json.loads(line.removeprefix(b"gomad-wasm-event-order-v1 ")) for line in lines]
            equal(frames[0], {"schema": "gomad.wasm-event-order/v1", "type": "header"}, label + ".header")
            equal(frames[-1]["type"], "complete", label + ".complete")
            equal(any(frame["type"] == "incomplete" for frame in frames), False, label + ".not_incomplete")
            events = [frame for frame in frames if frame["type"] == "event"]
            equal([frame["order"] for frame in events], list(range(1, len(events) + 1)), label + ".commit_order")
            equal(len(events), row["event_order"]["events"], label + ".event_count")
            equal(frames[-1]["events"], len(events), label + ".terminal_event_count")
            equal(frames[-1]["bytes"], sum(map(len, lines[:-1])), label + ".terminal_byte_count")
            equal(0 < len(events) <= 131072 and len(data) <= (32 << 20), True, label + ".bounds")
        return report

    reports = {}
    for role, binary in binaries.items():
        root = args.output / (role + "-positive")
        arguments = ["fixture-replay-role", "--root", root, "--corpus", corpus, "--role", role]
        reports[role] = authenticate(root, run(role + "-positive", binary, arguments).stdout, role)
        ambient_root = args.output / (role + "-ambient")
        ambient = authenticate(ambient_root, run(role + "-ambient", binary,
                               ["fixture-replay-role", "--root", ambient_root, "--corpus", corpus, "--role", role], ambient=True).stdout, role)
        equal(ambient, reports[role], role + ".ambient_cannot_change_inputs_or_diagnostics")
        run(role + "-existing-root", binary, arguments, status=2, entered=False)
        default_root = args.output / (role + "-default")
        default = run(role + "-default", binary,
                      ["fixture-replay-role", "--root", default_root, "--corpus", corpus],
                      status=0 if role == "ordinary" else 2, entered=role == "ordinary")
        if role == "ordinary":
            authenticate(default_root, default.stdout, role)
        for other in binaries:
            if other == role:
                continue
            refused = run(role + "-wrong-" + other, binary,
                          ["fixture-replay-role", "--root", args.output / (role + "-wrong-" + other),
                           "--corpus", corpus, "--role", other], status=2, entered=False)
            equal(b"engine/module/import identity mismatch" in refused.stderr, True, role + ".wrong_role_reason")
    equal(len({report["helper_sha256"] for report in reports.values()}), 3, "distinct_compiled_helper_identities")
    if args.release_binary:
        run("release-no-fixture-command", args.release_binary.resolve(),
            ["fixture-replay-role", "--root", args.output / "release-refused", "--corpus", corpus,
             "--role", "determinism-fixture"], status=2, entered=False)
    for role, binary in binaries.items():
        equal(sha(binary.read_bytes()), binary_hashes[role], role + ".binary_changed")
    receipt = {"schema": "gomad.replay-role-check/v1", "binary_sha256": binary_hashes,
               "driver_sha256": sha(Path(__file__).read_bytes()),
               "helper_sha256": {role: report["helper_sha256"] for role, report in reports.items()},
               "scope": "actual fixture CLI role executions; public registered inspection parity is a separate gate",
               "commands": len(commands)}
    (args.output / "role-check-result.json").write_text(json.dumps(receipt, indent=2) + "\n")
    print("explicit compiled replay roles: repeated/concurrent guests, ambient isolation and wrong/default/release refusals")


if __name__ == "__main__":
    main()
