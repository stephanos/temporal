#!/usr/bin/env python3
"""Run both inspect CLIs on fresh, independently published lineage artifacts."""
import argparse
import base64
import hashlib
import itertools
import json
import os
from pathlib import Path
import subprocess
import tarfile


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def equal(actual, expected, path):
    if actual != expected:
        raise ValueError(f"{path}: got {actual!r}, expected {expected!r}")


def digest(value):
    return value.removeprefix("sha256:")


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def authenticate_report(actual, root, label):
    equal(set(actual), {"record", "record_sha256", "failure_signature", "stored_bytes", "files",
                        "choices", "world", "simulation", "minimization", "stdout", "stderr"},
          label + ".artifact_fields")
    raw = (root / "record.json").read_bytes()
    record = json.loads(raw)
    equal(raw, canonical(record), label + ".canonical_record")
    equal(actual["record"], record, label + ".record")
    equal(actual["record_sha256"], sha(raw), label + ".record_sha256")
    equal(record["execution_identity"], sha(canonical(record["identity"])), label + ".execution_identity")
    equal(actual["failure_signature"], sha(canonical(dict(record, seed=0))), label + ".failure_signature")
    return record


def compare_outputs(go, rust, label):
    for name in ("stdout", "stderr"):
        equal(set(rust[name]), set(go[name]), label + "." + name + ".fields")
        for key in go[name]:
            expected = digest(go[name][key]) if key == "full_sha256" else go[name][key]
            equal(rust[name][key], expected, label + "." + name + "." + key)


def compare_failure_groups(failures):
    equal(len(failures), 8, "failure_grouping.failures")
    comparisons = 0
    for left, right in itertools.combinations(failures, 2):
        equal(left["rust_signature"] == right["rust_signature"],
              left["go_signature"] == right["go_signature"],
              left["execution"] + ".grouping." + right["execution"])
        comparisons += 1
    equal(comparisons, 28, "failure_grouping.pair_comparisons")
    partitions = {}
    for lineage in ("go", "rust"):
        groups = {}
        for item in failures:
            groups.setdefault(item[lineage + "_signature"], []).append(item["execution"])
        partitions[lineage] = sorted(sorted(group) for group in groups.values())
    return {"failures": len(failures), "groups": len(partitions["go"]),
            "pair_comparisons": comparisons, "partitions": partitions}


def compare_choices(run, baseline, rust_binary, go_root, rust_root, corpus):
    go_rows = json.loads((go_root / "reference.json").read_bytes())["records"]
    rust_rows = json.loads((rust_root / "foundation-result.json").read_bytes())["records"]
    equal(len(go_rows), 4, "choices.go.count")
    equal(len(rust_rows), 4, "choices.rust.count")
    cases = [(name, seed) for name in ("runtime-choice", "runtime-choice-overflow") for seed in (7, 8)]
    for (name, seed), go_row, rust_row in zip(cases, go_rows, rust_rows):
        overflow = name == "runtime-choice-overflow"
        label = f"{name}.seed{seed}"
        equal((go_row["name"], go_row["seed"]), (name, seed), label + ".go.selection")
        equal(rust_row["seed"], seed, label + ".rust.selection")
        go_path = Path(go_row["artifact"])
        rust_path = rust_root / rust_row["artifact"]
        go_report = json.loads(run(label + ".go-inspect", [baseline / "bin/gomad", "inspect",
                                                          "--json", "--choices", go_path]))
        rust_report = json.loads(run(label + ".rust-inspect", [rust_binary, "inspect",
                                                              "--json", "--choices", rust_path]))
        equal(go_report["schema"], "gomad.inspect/v8", label + ".go.schema")
        equal(rust_report["schema"], "gomad.inspect/v9", label + ".rust.schema")
        go = go_report["artifact"]
        rust = rust_report["artifact"]
        authenticate_report(rust, rust_path, label + ".rust")
        compare_outputs(go, rust, label)
        go_manifest = json.loads((go_path / "manifest.json").read_bytes())
        rust_manifest = json.loads((rust_path / "manifest.json").read_bytes())
        equal(go["record_hash"], go_manifest["record_hash"], label + ".go.record_hash")
        equal(go["record_hash"], go_row["record_hash"], label + ".go.reference_hash")
        record_raw = (rust_path / "record.json").read_bytes()
        record = json.loads(record_raw)
        equal(rust["record"], record, label + ".rust.record")
        equal(rust["record_sha256"], sha(record_raw), label + ".rust.record_sha256")
        equal(rust_manifest["record"], record, label + ".rust.manifest_record")
        equal(rust["files"], rust_manifest["files"], label + ".rust.files")
        for name, entry in rust_manifest["files"].items():
            payload = (rust_path / name).read_bytes()
            equal(sha(payload), entry["sha256"], label + ".rust.payload." + name)
            equal(len(payload), entry["bytes"], label + ".rust.payload_bytes." + name)
        for entry in go_manifest["files"]:
            payload = (go_path / entry["path"]).read_bytes()
            equal(sha(payload), digest(entry["sha256"]), label + ".go.payload." + entry["path"])
            equal(len(payload), int(entry["size"]), label + ".go.payload_bytes." + entry["path"])
        equal(go["seed"], seed, label + ".go.seed")
        equal(record["seed"], seed, label + ".rust.seed")
        module = "targets/inspect-choice-overflow.wat" if overflow else "targets/inspect-choice.wat"
        target = sha((corpus / module).read_bytes())
        equal(digest(go["target"]["sha256"]), target, label + ".go.target")
        equal(record["identity"]["target"], target, label + ".rust.target")
        if overflow:
            equal(go["outcome"]["termination"], "none", label + ".go.termination")
            equal(go["outcome"]["reason"], "choice_trace_overflow", label + ".go.reason")
        else:
            equal(go["outcome"]["termination"], "exit", label + ".go.termination")
        equal(record["outcome"]["termination"], "capacity" if overflow else "exit", label + ".termination")
        equal(go["outcome"]["domain"], "runner" if overflow else "success", label + ".go.outcome_class")
        equal(record["outcome"]["class"], "runner_failure" if overflow else "success", label + ".rust.outcome_class")
        equal(go["outcome"].get("exit_code"), record["outcome"]["exit_code"], label + ".exit_code")
        left = go["choices"]
        right = rust["choices"]
        for key in ("schema", "limit", "payload_bytes", "records", "decisions", "branching_records",
                    "terminal_state", "exact_replay_available", "runnable", "select_poll",
                    "select_result"):
            equal(right[key], left[key], label + ".choices." + key)
        # fn230 moved Rust recording to choice trace v4 (versioned creation
        # sites); the frozen Go reference still records v3. Each lineage must
        # report exactly its own declared profile.
        equal(left["profile"], "gomad-choice-trace/v3", label + ".go.choices.profile")
        equal(right["profile"], "gomad-choice-trace/v4", label + ".rust.choices.profile")
        equal(right['sites'], [dict(site, symbol={'unavailable': 'no_symbols'}) for site in left['sites']],
              label + '.choices.sites')
        equal(right["terminal_state"], "overflow" if overflow else "complete", label + ".choices.terminal_state")
        equal(right["exact_replay_available"], not overflow, label + ".choices.exact_replay_available")
        equal(right["limit"], 160 if overflow else 4096, label + ".choices.limit")
        decisions = [dict(item, site_missing=item.get("site_missing", False), symbol={'unavailable': 'no_symbols'})
                     for item in left.get("replay_decisions", [])]
        equal(right["replay_decisions"], decisions, label + ".choices.replay_decisions")
        equal(left["decisions"], len(decisions), label + ".choices.decisions")
        trace = (go_path / "choices.bin").read_bytes()
        raw = (go_root / go_row["raw_result"]).read_bytes()
        equal(sha(raw), digest(go_row["raw_sha256"]), label + ".go.raw_sha256")
        observed = json.loads(raw)
        equal(observed["Reaped"], True, label + ".go.reaped")
        equal(observed["Cancelled"], False, label + ".go.cancelled")
        equal(observed["Termination"], "capacity" if overflow else "exit", label + ".go.raw_termination")
        raw_trace = observed["PartialHelperChoice"] if overflow else observed["ChoiceTrace"]["Bytes"]
        equal(trace, base64.b64decode(raw_trace or "", validate=True), label + ".go.raw_trace")
        equal(len(trace), 96, label + ".choices.prefix_bytes")
        equal(trace, (rust_path / "evidence/choice").read_bytes(), label + ".choices.raw_trace")
        equal(digest(left["sha256"]), sha(trace), label + ".go.choices.sha256")
        equal(digest(right["sha256"]), sha(trace), label + ".rust.choices.sha256")
        equal(left.get("tape_sha256", ""), go_manifest["choice_profile"]["trace"].get("tape_sha256", ""), label + ".go.tape_sha256")
        details = json.loads((rust_path / "evidence/inspection.json").read_bytes())
        equal(right["tape_sha256"], details["choices"]["tape_sha256"] or "", label + ".rust.tape_sha256")
        if overflow:
            equal(left.get("tape_sha256", ""), "", label + ".go.overflow_tape")
            equal(right["tape_sha256"], "", label + ".rust.overflow_tape")
            equal(decisions, [], label + ".overflow_decisions")


def go_minimization(run, args):
    original = args.source / "tools/gomad/internal/backends/wasm/testdata/io_failure_module"
    guest = args.output / "minimization-guest"
    (guest / "io_failure").mkdir(parents=True, mode=0o700)
    (guest / "go.mod").write_bytes((original / "go.mod").read_bytes())
    original_test = (original / "io_failure/io_failure_test.go").read_text()
    trigger = "func TestDeterministicIOFailure(t *testing.T) {\n"
    equal(original_test.count(trigger), 1, "minimization.fixture_trigger")
    concurrency = ("\tdone := make(chan struct{}, 2)\n"
                   "\tgo func() { done <- struct{}{} }()\n"
                   "\tgo func() { done <- struct{}{} }()\n"
                   "\t<-done\n\t<-done\n")
    guest_test = original_test.replace(trigger, trigger + concurrency)
    (guest / "io_failure/io_failure_test.go").write_text(guest_test)
    installation = args.output / "go-installation.json"
    baseline = args.baseline / "bin/gomad"
    run("go-install", [baseline, "install", "--compiler", args.go_compiler,
                       "--helper", args.baseline / "bin/gomad-wasmhost",
                       "--runtime-root", args.source / "tools/gomad/internal/model/runtimesource",
                       "--profile", "gomad.wasi-cooperative/v1", "--cache", args.output / "preparation",
                       "--output", installation])
    artifact_root = args.output / "go-minimization-artifacts"
    run("go-minimization-explore", [baseline, "--installation", installation, "explore", "--json",
                                    "--choices", "--seeds=7", "--parallel=1", "--working-dir",
                                    guest,
                                    "--build-tag=gomad_fixture", "--build-tag=test_dep",
                                    "--execution-timeout=30s", "--overall-timeout=2m",
                                    "--artifacts", artifact_root, "go-test", "./io_failure", "--",
                                    "-test.run=^TestDeterministicIOFailure$", "-test.v"], expected_status=1)
    parents = [path.parent for path in artifact_root.rglob("manifest.json")
               if json.loads(path.read_bytes()).get("artifact_kind") == "gomad.target-failure/v1"]
    equal(len(parents), 1, "minimization.parent_count")
    minimized = json.loads(run("go-minimize", [baseline, "--installation", installation, "minimize",
                                              "--json", "--attempt-budget=4", "--artifacts",
                                              args.output / "go-minimized", parents[0]]))
    equal(minimized["changed"], True, "minimization.changed")
    report = json.loads(run("go-minimized-inspect", [baseline, "inspect", "--json",
                                                    minimized["artifact_path"]]))
    manifest = json.loads((Path(minimized["artifact_path"]) / "manifest.json").read_bytes())
    equal(report["artifact"]["record_hash"], manifest["record_hash"], "minimization.record_hash")
    accepted = [{**item, "removed": [{**decision, "ordinal": int(decision["ordinal"])}
                                     for decision in item["removed"]]}
                for item in minimized["accepted"]]
    equal(report["artifact"]["minimization"]["accepted"], accepted, "minimization.accepted")
    specimen = {"schema": "gomad.inspect-go-minimization-specimen/v1",
                "producer": "actual baseline gomad explore and minimize",
                "guest_source_sha256": sha(guest_test.encode()),
                "guest_source": str(guest / "io_failure/io_failure_test.go"),
                "artifact": minimized["artifact_path"], "record_hash": manifest["record_hash"],
                "minimization": report["artifact"]["minimization"]}
    path = args.output / "go-minimization-specimen.json"
    path.write_text(json.dumps(specimen, indent=2) + "\n")
    return path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--go-compiler", type=Path, required=True)
    parser.add_argument("--rust-binary", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--go-minimization-only", action="store_true")
    args = parser.parse_args()
    corpus = Path(__file__).resolve().parent
    for name in ("baseline", "source", "go_compiler", "rust_binary", "output"):
        setattr(args, name, getattr(args, name).resolve())
    args.output.mkdir(mode=0o700)
    commands = []

    def run(name, command, cwd=None, env=None, expected_status=0):
        try:
            result = subprocess.run([str(part) for part in command], cwd=cwd,
                                    env=env, capture_output=True, timeout=120)
        except OSError as error:
            result = subprocess.CompletedProcess(command, 127, b"", str(error).encode())
        except subprocess.TimeoutExpired as error:
            result = subprocess.CompletedProcess(command, 124, error.stdout or b"",
                                                 (error.stderr or b"") + b"\ncommand timeout\n")
        (args.output / f"{name}.stdout").write_bytes(result.stdout)
        (args.output / f"{name}.stderr").write_bytes(result.stderr)
        commands.append({"command": [str(part) for part in command],
                         "cwd": str(cwd) if cwd else None,
                         "environment": {key: env[key] for key in ("GOENV", "GOTOOLCHAIN", "GOWORK", "CGO_ENABLED")} if env else None,
                         "status": result.returncode, "expected_status": expected_status,
                         "stdout": f"{name}.stdout", "stderr": f"{name}.stderr"})
        (args.output / "commands.json").write_text(json.dumps(commands, indent=2) + "\n")
        equal(result.returncode, expected_status, f"{name}.status")
        return result.stdout

    for line in (args.baseline / "inputs.sha256").read_text().splitlines():
        expected, relative = line.split()
        equal(sha((args.baseline / relative).read_bytes()), expected, relative)
    equal(sha(args.go_compiler.read_bytes()),
          "1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64",
          "Go compiler identity")
    module = args.source / "tools/gomad"
    with tarfile.open(args.baseline / "source-1d7b37f00a.tar.gz") as archive:
        for member in archive.getmembers():
            parts = Path(member.name).parts
            if member.isfile() and parts[1:3] == ("tools", "gomad"):
                relative = Path(*parts[1:])
                equal(sha((args.source / relative).read_bytes()),
                      sha(archive.extractfile(member).read()), str(relative))
    if args.go_minimization_only:
        specimen = go_minimization(run, args)
        print(f"fresh baseline Go minimization inspect specimen: {specimen}")
        return
    wrapper = module / ("cmd/fn232-inspect-reference-" + sha(str(args.output).encode())[:12])
    wrapper.mkdir()
    template = corpus / "reference/inspect.go.txt"
    (wrapper / "main.go").write_bytes(template.read_bytes())
    build_env = {key: value for key, value in os.environ.items()
                 if key not in ("GOROOT", "GOFLAGS", "GOMOD", "GOPATH", "GOEXPERIMENT")
                 and not key.startswith("GOMAD_")}
    build_env.update({"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "CGO_ENABLED": "0"})
    executable = args.output / "go-inspect-reference"
    run("go-build", [args.go_compiler, "build", "-trimpath", "-o", executable,
                     "./" + str(wrapper.relative_to(module))], module, build_env)
    go_root = args.output / "go"
    rust_root = args.output / "rust"
    run("go-execute", [executable, "--helper", args.baseline / "bin/gomad-wasmhost",
                       "--corpus", corpus, "--output", go_root])
    go_choices = args.output / "go-choices"
    rust_choices = args.output / "rust-choices"
    run("go-choices-execute", [executable, "--helper", args.baseline / "bin/gomad-wasmhost",
                               "--corpus", corpus, "--output", go_choices, "--choices"])
    rust_binary_sha = sha(args.rust_binary.read_bytes())
    run("rust-execute", [args.rust_binary, "fixture-execution-bundle", "--root",
                         rust_root, "--corpus", corpus])
    run("rust-choices-execute", [args.rust_binary, "fixture-inspect-execution", "--root",
                                 rust_choices, "--corpus", corpus])
    run("rust-integrity", ["python3", corpus.parents[1] / "scripts/behavioral-parity",
                           "--corpus", corpus, "--foundation-root", rust_root,
                           "--foundation-result", rust_root / "foundation-result.json"])
    index_raw = (corpus / "foundation-execution.json").read_bytes()
    index = json.loads(index_raw)
    reference = json.loads((go_root / "reference.json").read_bytes())
    equal(reference["schema"], "gomad.inspect-go-reference/v1", "go.schema")
    equal(digest(reference["corpus_sha256"]), sha(index_raw), "go.corpus_sha256")
    rust = json.loads((rust_root / "foundation-result.json").read_bytes())
    cases = [(case, seed) for case in index["cases"] for seed in index["seeds"]]
    equal(len(cases), 12, "corpus.count")
    equal(len(reference["records"]), len(cases), "go.count")
    equal(len(rust["records"]), len(cases), "rust.count")
    failures = []
    for (case, seed), go_row, rust_row in zip(cases, reference["records"], rust["records"]):
        label = f"{case['name']}.seed{seed}"
        equal((go_row["name"], go_row["seed"]), (case["name"], seed), label + ".selection")
        equal(rust_row["seed"], seed, label + ".rust.selection")
        go_path = Path(go_row["artifact"])
        rust_path = rust_root / rust_row["artifact"]
        go_report = json.loads(run(label + ".go-inspect",
                                   [args.baseline / "bin/gomad", "inspect", "--json", go_path]))
        rust_report = json.loads(run(label + ".rust-inspect",
                                     [args.rust_binary, "inspect", "--json", rust_path]))
        equal(go_report["schema"], "gomad.inspect/v8", label + ".go.schema")
        equal(rust_report["schema"], "gomad.inspect/v9", label + ".rust.schema")
        for name, report, path in (("go", go_report, go_path), ("rust", rust_report, rust_path)):
            equal(report["kind"], "artifact", label + f".{name}.kind")
            equal(Path(report["path"]), path, label + f".{name}.path")
        go = go_report["artifact"]
        actual = rust_report["artifact"]
        authenticate_report(actual, rust_path, label + ".rust")
        compare_outputs(go, actual, label)
        record_raw = (rust_path / "record.json").read_bytes()
        record = json.loads(record_raw)
        equal(actual["record"], record, label + ".rust.record")
        equal(actual["record_sha256"], sha(record_raw), label + ".rust.record_sha256")
        equal(actual["failure_signature"], rust_row["failure_signature"], label + ".rust.failure_signature")
        rust_manifest_raw = (rust_path / "manifest.json").read_bytes()
        rust_manifest = json.loads(rust_manifest_raw)
        equal(actual["files"], rust_manifest["files"], label + ".rust.files")
        stored_bytes = len(rust_manifest_raw) + len((rust_path / "storage.json").read_bytes())
        for name, file in rust_manifest["files"].items():
            payload = (rust_path / name).read_bytes()
            equal(sha(payload), file["sha256"], label + ".rust.files." + name)
            equal(len(payload), file["bytes"], label + ".rust.file_size." + name)
            stored_bytes += len(payload)
        equal(actual["stored_bytes"], stored_bytes, label + ".rust.stored_bytes")
        manifest = json.loads((go_path / "manifest.json").read_bytes())
        equal(go["record_hash"], manifest["record_hash"], label + ".go.record_hash")
        equal(go["record_hash"], go_row["record_hash"], label + ".go.reference_hash")
        raw = (go_root / go_row["raw_result"]).read_bytes()
        equal(sha(raw), digest(go_row["raw_sha256"]), label + ".go.raw_sha256")
        observed = json.loads(raw)
        equal(observed["Reaped"], True, label + ".go.reaped")
        equal(observed["Cancelled"], False, label + ".go.cancelled")
        termination = "watchdog_timeout" if observed["WatchdogTimeout"] else observed["Termination"]
        # v1's generic record uses 'none' plus the backend reason for non-exit stops.
        inspected_termination = go["outcome"]["termination"]
        if inspected_termination == "none":
            inspected_termination = go["outcome"]["reason"]
        equal(inspected_termination, termination, label + ".go.termination")
        equal(record["outcome"]["termination"], termination, label + ".termination")
        equal(go["outcome"].get("exit_code"), observed["ExitCode"], label + ".go.exit_code")
        equal(record["outcome"]["exit_code"], observed["ExitCode"], label + ".exit_code")
        go_failure = go["outcome"]["domain"] != "success"
        rust_failure = record["outcome"]["class"] != "success"
        equal(rust_failure, go_failure, label + ".failure_class")
        equal(go["outcome"]["failure_signature"], manifest["outcome"]["failure_signature"],
              label + ".go.failure_signature")
        if go_failure:
            failures.append({"execution": label, "go_signature": go["outcome"]["failure_signature"],
                             "rust_signature": actual["failure_signature"]})
        equal(go["seed"], seed, label + ".go.seed")
        equal(record["seed"], seed, label + ".seed")
        arguments = ["fixture"] if case["modeled"] else []
        environment = {"PWD": "/"} if case["modeled"] else {}
        equal(go["target"]["argv"], ["gomad-target"] + arguments, label + ".go.arguments")
        equal(record["arguments"], arguments, label + ".arguments")
        equal(go_row["arguments"], arguments, label + ".go.request.arguments")
        equal(go_row["environment"], environment, label + ".go.request.environment")
        # Go v8 exposes no environment; authenticate it through its manifest envelope.
        go_environment = {entry["name"]: entry["value"] for entry in manifest["environment"]}
        equal(go_environment, {"GOMADSEED": str(seed), "GOMAD_IO_PROFILE": index["profile"],
                               "TZ": "UTC", **environment}, label + ".go.environment")
        equal(record["environment"], environment, label + ".environment")
        target_sha = sha((corpus / case["module"]).read_bytes())
        equal(digest(go["target"]["sha256"]), target_sha, label + ".go.module")
        equal(record["identity"]["target"], target_sha, label + ".module")
        equal(go_row["helper_sha256"], "sha256:" + sha((args.baseline / "bin/gomad-wasmhost").read_bytes()),
              label + ".go.helper_sha256")
        if observed["Started"] is not None:
            equal(observed["Started"]["module_sha256"], target_sha, label + ".go.started.module_sha256")
        for file in manifest["files"]:
            payload = (go_path / file["path"]).read_bytes()
            equal(sha(payload), digest(file["sha256"]), label + ".go.files." + file["path"])
            equal(len(payload), int(file["size"]), label + ".go.file_size." + file["path"])
        for go_name, rust_name, raw_name in (("stdout", "output/stdout", "Stdout"),
                                              ("stderr", "output/stderr", "Stderr"),
                                              ("transcript", "evidence/transcript", "Transcript")):
            payload = base64.b64decode(observed[raw_name] or "", validate=True)
            stream = record["streams"].get(rust_name)
            rust_payload = (rust_path / rust_name).read_bytes() if stream else b""
            equal(rust_payload, payload, label + "." + go_name + ".bytes")
            if stream:
                equal(stream["sha256"], sha(payload), label + "." + go_name + ".sha256")
                equal(stream["bytes"], len(payload), label + "." + go_name + ".size")
            if go_name == "transcript":
                if payload:
                    equal(digest(go[go_name]["sha256"]), sha(payload), label + ".go.transcript.sha256")
                    equal(go[go_name]["bytes"], len(payload), label + ".go.transcript.size")
                    equal(go[go_name]["records"], observed["Calls"], label + ".go.transcript.records")
                    equal(stream["records"], observed["Calls"], label + ".transcript.records")
            else:
                equal(digest(go[go_name]["full_sha256"]), sha(payload), label + ".go." + go_name + ".sha256")
                equal(go[go_name]["total_bytes"], len(payload), label + ".go." + go_name + ".size")
                equal(go[go_name]["retained_bytes"], len(payload), label + ".go." + go_name + ".retained")
                equal(go[go_name]["discarded_bytes"], 0, label + ".go." + go_name + ".discarded")
                equal(go[go_name]["truncated"], False, label + ".go." + go_name + ".truncated")
        for name in ("choices", "world", "simulation", "minimization"):
            equal(go.get(name), None, label + ".go." + name)
            equal(actual[name], None, label + ".rust." + name)
    compare_choices(run, args.baseline, args.rust_binary, go_choices, rust_choices, corpus)
    grouping = compare_failure_groups(failures)
    refused = run("rust-refuse-fresh-go", [args.rust_binary, "inspect", "--json",
                                           reference["records"][0]["artifact"]], expected_status=2)
    equal(refused, b"", "rust.refuse_fresh_go.stdout")
    refusal_stderr = (args.output / "rust-refuse-fresh-go.stderr").read_bytes()
    equal(b"unsupported artifact storage format" in refusal_stderr, True, "rust.refuse_fresh_go.diagnostic")
    equal(sha(args.rust_binary.read_bytes()), rust_binary_sha, "rust.binary_changed_during_run")
    receipt = {"schema": "gomad.inspect-parity/v1", "executions": len(cases), "choice_executions": 4,
               "choice_overflow_executions": 2,
               "failure_grouping": grouping,
               "expected_refusals": [{"command": "rust-refuse-fresh-go", "status": 2, "stdout_bytes": 0}],
               "command_status_counts": {str(status): sum(row["status"] == status for row in commands)
                                         for status in sorted({row["status"] for row in commands})},
               "go_binary_sha256": sha((args.baseline / "bin/gomad").read_bytes()),
               "rust_binary_sha256": rust_binary_sha,
               "go_reference_binary_sha256": sha(executable.read_bytes()),
               "template_sha256": sha(template.read_bytes()), "corpus_sha256": sha(index_raw)}
    (args.output / "inspect-result.json").write_text(json.dumps(receipt, indent=2) + "\n")
    print("authenticated Go/Rust inspect field parity: 12/12 fresh executions, 4/4 runtime choices (2 overflow), Go-lineage refusal status 2")


if __name__ == "__main__":
    main()
