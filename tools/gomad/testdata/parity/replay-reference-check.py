#!/usr/bin/env python3
"""Retain authenticated fresh replay receipts from the frozen Go public CLI."""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile

sys.dont_write_bytecode = True


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f"{label}: got {actual!r}, expected {expected!r}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("baseline", "source", "go-compiler", "output"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--preparation-root", type=Path)
    args = parser.parse_args()
    for name in ("baseline", "source", "go_compiler", "output"):
        setattr(args, name, getattr(args, name).resolve())
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    preparation = args.preparation_root.resolve() if args.preparation_root else args.output
    corpus = Path(__file__).resolve().parent
    inventory = (args.baseline / "inputs.sha256").read_bytes()
    for line in inventory.decode().splitlines():
        expected, relative = line.split()
        equal(sha((args.baseline / relative).read_bytes()), expected, relative)
    equal(sha(args.go_compiler.read_bytes()),
          "1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64",
          "Go compiler identity")
    with tarfile.open(args.baseline / "source-1d7b37f00a.tar.gz") as archive:
        for member in archive.getmembers():
            parts = Path(member.name).parts
            if member.isfile() and parts[1:3] == ("tools", "gomad"):
                relative = Path(*parts[1:])
                equal(sha((args.source / relative).read_bytes()),
                      sha(archive.extractfile(member).read()), str(relative))
    env = {key: value for key, value in os.environ.items()
           if key not in ("GOROOT", "GOFLAGS", "GOMOD", "GOPATH", "GOEXPERIMENT")
           and not key.startswith("GOMAD_")}
    env.update({"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "CGO_ENABLED": "0",
                "GOPROXY": "off", "GOSUMDB": "off",
                "GOCACHE": str(args.output / "go-cache"), "TMPDIR": str(args.output / "tmp")})
    (args.output / "tmp").mkdir(mode=0o700)
    commands = []

    def run(name, command, status=0, cwd=None, input_bytes=None):
        try:
            result = subprocess.run([str(part) for part in command], cwd=cwd, env=env,
                                    capture_output=True, timeout=600, input=input_bytes)
        except OSError as error:
            result = subprocess.CompletedProcess(command, 127, b"", str(error).encode())
        except subprocess.TimeoutExpired as error:
            result = subprocess.CompletedProcess(command, 124, error.stdout or b"",
                                                 (error.stderr or b"") + b"\ncommand timeout\n")
        (args.output / (name + ".stdout")).write_bytes(result.stdout)
        (args.output / (name + ".stderr")).write_bytes(result.stderr)
        commands.append({"name": name, "command": [str(part) for part in command],
                         "status": result.returncode, "expected_status": status,
                         "stdout_sha256": sha(result.stdout), "stderr_sha256": sha(result.stderr)})
        (args.output / "commands.json").write_text(json.dumps(commands, indent=2) + "\n")
        equal(result.returncode, status, name + ".status")
        return result.stdout

    fixture_path = args.source / "tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/helper_transcript/storage_crash_before_helper_originated.json"
    fixture_raw = fixture_path.read_bytes()
    fixture = json.loads(fixture_raw)
    guest = fixture["guest"]["wat"].encode()
    guest_path = args.output / "storage-origin.wat"
    guest_path.write_bytes(guest)
    request = dict(fixture["request"], module_path=str(guest_path), module_sha256=sha(guest))
    authorize = {"type": "authorize", "module_sha256": sha(guest),
                 "configuration": fixture["expected"]["frames"][0]["engine"]["configuration"]}
    namespace_frames = fixture["replies"][0]["send"][1:]
    wire = [request, *fixture["inputs"], authorize, *namespace_frames]
    storage_raw = run("storage-origin", [args.baseline / "bin/gomad-wasmhost"],
                      input_bytes=b"".join(json.dumps(frame, separators=(",", ":")).encode() + b"\n" for frame in wire))
    storage_frames = [json.loads(line) for line in storage_raw.splitlines()]
    storage_terminal = [frame for frame in storage_frames if frame["type"] == "result"]
    equal(len(storage_terminal), 1, "storage_origin.terminals")
    equal(storage_terminal[0]["termination"], "storage_crash", "storage_origin.termination")
    equal(storage_terminal[0]["message"], "modeled storage crash: stop-sync", "storage_origin.message")
    storage_evidence = {}
    for frame in storage_frames:
        if frame["type"] == "evidence":
            storage_evidence.setdefault(frame["stream"], bytearray()).extend(base64.b64decode(frame["data_base64"], validate=True))
    equal(set(storage_evidence), {"transcript", "storage", "storage_state", "storage_image"}, "storage_origin.streams")
    storage_receipt = {"schema": "gomad.replay-storage-origin-reference/v1",
                       "scope": "unchanged frozen helper transcript fixture; zero runtime decisions",
                       "fixture_sha256": sha(fixture_raw), "module_sha256": sha(guest),
                       "module_path": str(guest_path), "request": request,
                       "storage_input": {"descriptor": request["storage"], "image": None, "context": None},
                       "namespace_image_base64": "".join(frame["data_base64"] for frame in namespace_frames if frame["type"] == "namespace_input"),
                       "terminal": storage_terminal[0],
                       "evidence": {name: {"bytes": len(data), "sha256": sha(data)} for name, data in storage_evidence.items()}}
    for name, data in storage_evidence.items():
        (args.output / ("storage-origin-" + name + ".bin")).write_bytes(data)
    (args.output / "storage-origin-input.json").write_text(json.dumps(storage_receipt, indent=2) + "\n")

    module = args.source / "tools/gomad"
    wrapper = module / ("cmd/fn232-replay-reference-" + sha(str(args.output).encode())[:12])
    wrapper.mkdir(mode=0o700)
    template = corpus / "reference/replay.go.txt"
    (wrapper / "main.go").write_bytes(template.read_bytes())
    executable = args.output / "go-replay-reference"
    run("go-build", [args.go_compiler, "build", "-trimpath", "-o", executable,
                     "./" + str(wrapper.relative_to(module))], cwd=module)
    baseline = args.baseline / "bin/gomad"
    rows = []
    for mode, profile in (("exact", "gomad.wasi-cooperative/v1"), ("observed", "gomad.wasi-stock-exploratory/v1")):
        installation = args.output / (mode + "-installation.json")
        install = [baseline, "install", "--compiler", args.go_compiler,
                   "--helper", args.baseline / "bin/gomad-wasmhost", "--profile", profile,
                   "--cache", preparation / (mode + "-preparation"), "--output", installation]
        if mode == "exact":
            install.extend(["--runtime-root", module / "internal/model/runtimesource"])
        run(mode + "-install", install)
        artifact_root = args.output / (mode + "-artifacts")
        working = module if mode == "exact" else module / "internal/backends/wasm/testdata/io_failure_module"
        target = (["go-run", "./internal/backends/wasm/testdata/runtime_probes"] if mode == "exact"
                  else ["go-test", "./io_failure", "--", "-test.run=^TestDeterministicIOFailure$", "-test.v"])
        expected_status = 0 if mode == "exact" else 1
        run(mode + "-explore", [baseline, "--installation", installation, "explore", "--json",
                               *(["--choices", "--keep-successes=all", "--success-limit=2", "--success-bytes=64MiB"] if mode == "exact" else []),
                               "--seeds=7,8", "--parallel=1", "--on-failure=all", "--working-dir", working,
                               "--build-tag=gomad_fixture", "--build-tag=test_dep",
                               "--execution-timeout=30s", "--overall-timeout=5m", "--artifacts",
                               artifact_root, *target], status=expected_status)
        artifacts = sorted(path.parent for path in artifact_root.rglob("manifest.json")
                           if json.loads(path.read_bytes()).get("artifact_kind") ==
                           ("gomad.success/v1" if mode == "exact" else "gomad.target-failure/v1"))
        equal(len(artifacts), 2, mode + ".artifacts")
        for artifact in artifacts:
            manifest = json.loads((artifact / "manifest.json").read_bytes())
            seed = int(manifest["seed"])
            equal(seed in (7, 8), True, mode + ".seed")
            for entry in manifest["files"]:
                data = (artifact / entry["path"]).read_bytes()
                equal(sha(data), entry["sha256"].removeprefix("sha256:"), entry["path"] + ".sha256")
                equal(len(data), int(entry["size"]), entry["path"] + ".bytes")
            name = f"{mode}-seed{seed}"
            result = run(name + "-replay", [baseline, "--installation", installation,
                                            "replay", artifact], status=expected_status).decode()
            equal(result, "gomad: reproduced=true diagnostic=false result=" +
                  ("success" if mode == "exact" else "target_failure") +
                  " choice-replay=" + ("exact" if mode == "exact" else "none") + "\n", name + ".match")
            provenance = json.loads((artifact / manifest["target"]["backend"]["provenance"]["file"]).read_bytes())
            retained_module = (artifact / manifest["target"]["file"]).read_bytes()
            equal(sha(retained_module), manifest["target"]["sha256"].removeprefix("sha256:"), name + ".retained_module")
            effective = json.loads(json.dumps(provenance["Config"]))
            effective["Args"] = manifest["target"]["argv"]
            effective["Environment"] = [entry["name"] + "=" + entry["value"]
                                         for entry in manifest["environment"]]
            effective["Environment"].append("PWD=" + effective["WorkingDirectory"])
            if mode == "observed":
                effective["EntropyKey"] = list(hashlib.sha256((provenance["BuildKey"] + ":" + str(seed)).encode()).digest())
            effective["Limits"]["OutputBytes"] = int(manifest["limits"]["output_bytes"])
            effective["Limits"]["TranscriptBytes"] = int(manifest["limits"]["io_transcript_bytes"])
            rows.append({"name": name, "mode": mode, "seed": seed, "case": "match",
                         "artifact": str(artifact), "record_hash": manifest["record_hash"],
                         "result": result, "execution_input": {"provenance": provenance, "effective_config": effective,
                         "module_path": str(artifact / manifest["target"]["file"]),
                         "module_sha256": sha(retained_module), "module_bytes": len(retained_module),
                         "environment": manifest["environment"], "limits": manifest["limits"],
                         "target": manifest["target"], "toolchain": manifest["toolchain"]}})
            mutation = "site" if mode == "exact" else "stdout"
            altered = json.loads(run(name + "-publish-divergence", [executable, "--artifact", artifact,
                                     "--output", args.output / (name + "-divergent"), "--mutation", mutation]))
            divergent = run(name + "-divergence", [baseline, "--installation", installation,
                             "replay", altered["artifact"]], status=1).decode()
            expected_divergence = "choice_profile.divergence.ordinal[0].site" if mode == "exact" else "stdout.full_sha256"
            equal(divergent, "gomad: reproduced=false divergence=" + expected_divergence +
                  " choice-replay=" + ("diverged" if mode == "exact" else "none") + "\n", name + ".divergence")
            rows.append({"name": name, "mode": mode, "seed": seed, "case": mutation,
                         "artifact": altered["artifact"], "record_hash": altered["record_hash"],
                         "result": divergent})
    receipt = {"schema": "gomad.replay-go-reference/v1", "producer": "frozen public Go CLI",
               "baseline_inputs_sha256": sha(inventory), "go_binary_sha256": sha(baseline.read_bytes()),
               "source_archive_sha256": sha((args.baseline / "source-1d7b37f00a.tar.gz").read_bytes()),
               "driver_sha256": sha(Path(__file__).read_bytes()),
               "go_compiler_sha256": sha(args.go_compiler.read_bytes()),
               "go_helper_sha256": sha((args.baseline / "bin/gomad-wasmhost").read_bytes()),
               "template_sha256": sha(template.read_bytes()), "wrapper_sha256": sha(executable.read_bytes()),
               "records": rows}
    (args.output / "replay-reference.json").write_text(json.dumps(receipt, indent=2) + "\n")
    print("fresh authenticated Go replay reference: " + str(args.output / "replay-reference.json"))


if __name__ == "__main__":
    main()
