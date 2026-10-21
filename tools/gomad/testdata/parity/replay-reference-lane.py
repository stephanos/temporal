#!/usr/bin/env python3
"""Compare fresh public Go/Rust replay on identical authenticated guest inputs."""
import argparse
import base64
import hashlib
import sys

sys.dont_write_bytecode = True

from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import itertools
import json
import os
from pathlib import Path
import shutil
import subprocess


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f"{label}: got {actual!r}, expected {expected!r}")


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("baseline", "source", "go-compiler", "rust-binary", "output"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--go-reference", type=Path)
    args = parser.parse_args()
    for name in ("baseline", "source", "go_compiler", "rust_binary", "output"):
        setattr(args, name, getattr(args, name).resolve())
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    corpus = Path(__file__).resolve().parent
    loader = SourceFileLoader("behavioral_parity", str(corpus.parent.parent / "scripts/behavioral-parity"))
    parity = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(parity)
    commands = []
    binary_sha256 = sha(args.rust_binary.read_bytes())

    def run(name, command, status=0):
        environment = {key: value for key, value in os.environ.items() if not key.startswith("GOMAD_")}
        try:
            result = subprocess.run(list(map(str, command)), capture_output=True,
                                    timeout=600, env=environment)
        except OSError as error:
            result = subprocess.CompletedProcess(command, 127, b"", str(error).encode())
        except subprocess.TimeoutExpired as error:
            result = subprocess.CompletedProcess(command, 124, error.stdout or b"",
                                                 (error.stderr or b"") + b"\ncommand timeout\n")
        (args.output / (name + ".stdout")).write_bytes(result.stdout)
        (args.output / (name + ".stderr")).write_bytes(result.stderr)
        commands.append({"name": name, "command": list(map(str, command)), "status": result.returncode,
                         "expected_status": status, "stdout_sha256": sha(result.stdout),
                         "stderr_sha256": sha(result.stderr)})
        (args.output / "commands.json").write_text(json.dumps(commands, indent=2) + "\n")
        equal(result.returncode, status, name + ".status")
        return result.stdout

    reference_path = args.go_reference.resolve() if args.go_reference else args.output / "go/replay-reference.json"
    if args.go_reference is None:
        run("go-reference", [sys.executable, corpus / "replay-reference-check.py", "--baseline", args.baseline,
                             "--source", args.source, "--go-compiler", args.go_compiler,
                             "--output", args.output / "go"])
    reference_raw = reference_path.read_bytes()
    reference = json.loads(reference_raw)
    equal(set(reference), {"schema", "producer", "baseline_inputs_sha256", "go_binary_sha256",
                          "source_archive_sha256", "driver_sha256", "go_compiler_sha256",
                          "go_helper_sha256", "template_sha256", "wrapper_sha256", "records"}, "reference.fields")
    equal(reference["schema"], "gomad.replay-go-reference/v1", "reference.schema")
    for field, path in (("baseline_inputs_sha256", args.baseline / "inputs.sha256"),
                        ("go_binary_sha256", args.baseline / "bin/gomad"),
                        ("go_helper_sha256", args.baseline / "bin/gomad-wasmhost"),
                        ("source_archive_sha256", args.baseline / "source-1d7b37f00a.tar.gz"),
                        ("go_compiler_sha256", args.go_compiler),
                        ("driver_sha256", corpus / "replay-reference-check.py"),
                        ("template_sha256", corpus / "reference/replay.go.txt"),
                        ("wrapper_sha256", reference_path.parent / "go-replay-reference")):
        equal(sha(path.read_bytes()), reference[field], "reference." + field)
    equal(len(reference["records"]), 8, "reference.records")
    matches = sorted((row for row in reference["records"] if row["case"] == "match"),
                     key=lambda row: (row["mode"], row["seed"]))
    equal([(row["mode"], row["seed"]) for row in matches],
          [(mode, seed) for mode in ("exact", "observed") for seed in (7, 8)], "reference.selection")
    fixture_root = args.output / "modules"
    fixture_root.mkdir(mode=0o700)
    cases, manifests = [], []
    for row in matches:
        name = row["name"]
        artifact = Path(row["artifact"])
        manifest = json.loads((artifact / "manifest.json").read_bytes())
        inspected = json.loads(run(name + "-go-authenticate", [args.baseline / "bin/gomad", "inspect", "--json", artifact]))
        equal(inspected["artifact"]["record_hash"], row["record_hash"], name + ".go.record_hash")
        equal(manifest["record_hash"], row["record_hash"], name + ".go.manifest_hash")
        for entry in manifest["files"]:
            payload = (artifact / entry["path"]).read_bytes()
            equal(sha(payload), entry["sha256"].removeprefix("sha256:"), name + ".go." + entry["path"])
            equal(len(payload), int(entry["size"]), name + ".go.bytes." + entry["path"])
        source = row["execution_input"]
        config, provenance = source["effective_config"], source["provenance"]
        captured = config["CapturedInputs"]
        equal((captured["Descriptor"], captured["Payloads"], captured["Manifest"]["Entries"],
               captured["Manifest"]["TotalBytes"]), (None, None, 0, 0), name + ".captured_inputs")
        equal((config.get("storage"), config.get("network_v2")), (None, None), name + ".model_scope")
        module = (artifact / manifest["target"]["file"]).read_bytes()
        equal(sha(module), source["module_sha256"], name + ".module_sha256")
        equal(len(module), source["module_bytes"], name + ".module_bytes")
        module_path = fixture_root / (name + ".wasm")
        module_path.write_bytes(module)
        module_path.chmod(0o600)
        environment = {}
        for entry in config["Environment"]:
            key, value = entry.split("=", 1)
            equal(key in environment, False, name + ".environment_duplicate." + key)
            environment[key] = value
        limits = config["Limits"]
        exact = row["mode"] == "exact"
        implementation = manifest["choice_profile"]["implementation_sha256"].removeprefix("sha256:") if exact else None
        choice_limit = int(manifest["choice_profile"]["trace"]["limit"]) if exact else 4096
        configuration = {"engine": "cranelift-fuel-nan-canonical-nounwind-epoch-v1", "profile": config["Profile"],
                         "limits": {"fuel": provenance["Fuel"], "memory_bytes": provenance["MemoryBytes"],
                                    "wall_millis": int(source["limits"]["execution_timeout_nanos"]) // 1000000,
                                    "output_bytes": limits["OutputBytes"],
                                    "evidence_bytes": limits["TranscriptBytes"] + choice_limit + 2 * limits["OutputBytes"] + (16 << 20)},
                         "arguments": config["Args"], "environment": environment,
                         "runtime_sha256": implementation,
                         "model": {"entropy_key": config["EntropyKey"], "mode": 1, "limit": choice_limit,
                                   "diagnostics": False, "epoch_nanos": config["Clock"]["EpochNanos"],
                                   "read_step_nanos": config["Clock"]["ReadStepNanos"], "now": 1 if exact and config["Clock"]["ReadStepNanos"] == 0 else 0,
                                   "working_directory_sha256": sha(config["WorkingDirectory"].encode()),
                                   "arguments": {"bytes": 0, "count": 0, "chunks": 0},
                                   "environment": {"bytes": 0, "count": 0, "chunks": 0},
                                   "limits": {"calls": limits["Calls"], "transcript_bytes": limits["TranscriptBytes"], "pending_events": limits["PendingEvents"]}},
                         "namespace": {"entries": [], "writable_directories": [list(path.encode()) for path in config["WritableDirectories"]],
                                       "preopens": [list(path.encode()) for path in sorted(set(config["WritableDirectories"]))],
                                       "working_directory": list(config["WorkingDirectory"].encode()),
                                       "stdin": list(base64.b64decode(config["Stdin"] or "", validate=True)),
                                       "limits": {"files": limits["Files"], "descriptors": limits["Descriptors"],
                                                  "filesystem_bytes": limits["FilesystemBytes"], "output_bytes": limits["OutputBytes"]},
                                       "epoch_nanos": config["Clock"]["EpochNanos"]}}
        cases.append({"name": name, "module_path": str(module_path), "module_sha256": sha(module),
                      "toolchain_build_key": provenance["BuildKey"], "seed": row["seed"],
                      "choice_identity": {"TargetSHA256": list(bytes.fromhex(sha(module))),
                                          "ToolchainBuildKey": provenance["BuildKey"], "GOOS": "wasip1", "GOARCH": "wasm",
                                          "ImplementationSHA256": list(bytes.fromhex(implementation))} if exact else None,
                      "configuration": configuration})
        manifests.append(manifest)
    input_path = args.output / "replay-input.json"
    input_path.write_bytes(canonical({"schema": "gomad.replay-fixture-input/v1", "cases": cases}))
    root = args.output / "rust"
    bundle = json.loads(run("rust-produce", [args.rust_binary, "fixture-replay-input", "--root", root, "--input", input_path]))
    parity.authenticate(root, bundle)
    equal(len(bundle["records"]), len(matches), "rust.records")
    repeated_root = args.output / "rust-repeat"
    repeated = json.loads(run("rust-repeat", [args.rust_binary, "fixture-replay-input", "--root", repeated_root, "--input", input_path]))
    parity.authenticate(repeated_root, repeated)
    equal(len(repeated["records"]), len(matches), "rust.repeat.records")
    for index, (first, second) in enumerate(zip(bundle["records"], repeated["records"])):
        equal((root / first["artifact"] / "record.json").read_bytes(),
              (repeated_root / second["artifact"] / "record.json").read_bytes(), f"rust.repeat.record[{index}]")
    comparisons = []
    for index, (go, rust) in enumerate(zip(matches, bundle["records"])):
        name = go["name"]
        path = root / rust["artifact"]
        equal(rust["seed"], go["seed"], name + ".seed")
        equal(run(name + "-rust-replay", [args.rust_binary, "replay", path], 0 if go["mode"] == "exact" else 1).decode(),
              go["result"], name + ".match")
        for go_file, rust_file in (("stdout", "output/stdout"), ("stderr", "output/stderr")):
            equal((path / rust_file).read_bytes() if (path / rust_file).exists() else b"",
                  (Path(go["artifact"]) / go_file).read_bytes(), name + "." + go_file)
        divergent_go = next(row for row in reference["records"] if row["name"] == name and row["case"] != "match")
        if go["mode"] == "exact":
            equal((path / "evidence/choice").read_bytes(), (Path(go["artifact"]) / "choices.bin").read_bytes(), name + ".choice_bytes")
            altered = json.loads(run(name + "-rust-publish-divergence", [args.rust_binary, "fixture-replay-divergence", path,
                                     "--root", args.output / (name + "-divergent"), "--trace", Path(divergent_go["artifact"]) / "choices.bin"]))
            altered_path = Path(altered["artifact"])
            expected = divergent_go["result"]
        else:
            altered_path = args.output / (name + "-divergent")
            shutil.copytree(path, altered_path)
            manifest = json.loads((altered_path / "manifest.json").read_bytes())
            data = bytearray((altered_path / "output/stdout").read_bytes())
            equal(bool(data), True, name + ".stdout_nonempty")
            data[0] ^= 1
            (altered_path / "output/stdout").write_bytes(data)
            for entry in (manifest["files"]["output/stdout"], manifest["record"]["streams"]["output/stdout"]):
                entry.update(bytes=len(data), sha256=sha(data))
            raw = canonical(manifest["record"])
            (altered_path / "record.json").write_bytes(raw)
            manifest["record_sha256"] = sha(raw)
            manifest["files"]["record.json"].update(bytes=len(raw), sha256=sha(raw))
            (altered_path / "manifest.json").write_bytes(canonical(manifest))
            equal(divergent_go["result"], "gomad: reproduced=false divergence=stdout.full_sha256 choice-replay=none\n", name + ".go.stdout_divergence")
            expected = divergent_go["result"]
        equal(run(name + "-rust-divergence", [args.rust_binary, "replay", altered_path], 1).decode(), expected, name + ".divergence")
        comparisons.append({"name": name, "match": True, "divergence": "ordinal[0].site" if go["mode"] == "exact" else "stdout.full_sha256"})
    failures = []
    for index, manifest in enumerate(manifests):
        go_failure = manifest["outcome"]["domain"] != "success"
        rust_record = json.loads((root / bundle["records"][index]["artifact"] / "record.json").read_bytes())
        equal(rust_record["outcome"]["class"] != "success", go_failure, f"failure_grouping.class[{index}]")
        if go_failure:
            failures.append(index)
    for left, right in itertools.combinations(failures, 2):
        equal(bundle["records"][left]["failure_signature"] == bundle["records"][right]["failure_signature"],
              manifests[left]["outcome"]["failure_signature"] == manifests[right]["outcome"]["failure_signature"],
              f"failure_grouping.{left}.{right}")
    equal(sha(args.rust_binary.read_bytes()), binary_sha256, "rust.binary_changed")
    report = {"schema": "gomad.replay-reference-parity/v1", "go_reference_sha256": sha(reference_raw),
              "input_sha256": sha(input_path.read_bytes()), "rust_binary_sha256": binary_sha256,
              "driver_sha256": sha(Path(__file__).read_bytes()), "comparisons": comparisons,
              "failure_grouping": {"failures": len(failures), "pair_comparisons": len(failures) * (len(failures) - 1) // 2},
              "evidence_budget": "transcript + choices + 2*output + 16MiB, one linked aggregate bound",
              "stdout_divergence": "stdout.full_sha256"}
    (args.output / "replay-reference-result.json").write_text(json.dumps(report, indent=2) + "\n")
    print("authenticated Go/Rust replay parity: four fresh identical-input executions, four own-artifact divergences")


if __name__ == "__main__":
    main()
