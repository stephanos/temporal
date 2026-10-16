"""Run the existing qualification cells for a matched full/slim comparison."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import time


GATES = {
    "onebox": "TestActualOneboxQualification",
    "multi-instance": "TestActualTemporalClusterQualification",
}
PROFILES = {"base": "full", "cand": "slim"}
BOUND_FIELDS = (
    "IntegratedSourceSHA256", "CompilerSHA256", "HelperSHA256",
    "EngineSHA256", "ModelSHA256", "HostConfigSHA256", "HostOS", "HostArch",
    "Profile", "Seed", "Repeat", "Fuel", "MemoryBytes",
)


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def command(args, cwd):
    return subprocess.check_output(args, cwd=cwd, text=True).strip()


def source_inventory(checkout):
    names = subprocess.check_output([
        "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard",
        "--", ".", ":!.flow",
    ], cwd=checkout).split(b"\0")
    return {
        str(checkout / os.fsdecode(name)): digest(checkout / os.fsdecode(name))
        for name in sorted(set(names)) if name and (checkout / os.fsdecode(name)).is_file()
    }


def host_snapshot():
    return {
        "utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "uname": list(platform.uname()), "load": list(os.getloadavg()),
        "cpuinfo": Path("/proc/cpuinfo").read_text(),
        "boot_id": Path("/proc/sys/kernel/random/boot_id").read_text().strip(),
        "meminfo": Path("/proc/meminfo").read_text(),
        "self_status": Path("/proc/self/status").read_text(),
        "processes": command(["ps", "-eo", "pid,ppid,comm,%cpu,rss", "--sort=-%cpu"], None),
    }


def wait_for_host(maximum_load, minimum_available):
    deadline = time.monotonic() + 15 * 60
    while True:
        meminfo = dict(line.split(":", 1) for line in Path("/proc/meminfo").read_text().splitlines())
        available = int(meminfo["MemAvailable"].split()[0]) * 1024
        if os.getloadavg()[0] < maximum_load and available >= minimum_available:
            return
        if time.monotonic() >= deadline:
            raise RuntimeError("host remained outside reserved load/memory bounds for 15 minutes")
        time.sleep(10)


def schedule(repetitions):
    cells = []
    for workload in GATES:
        for repetition in range(1, repetitions + 1):
            order = ("base", "cand") if repetition % 2 else ("cand", "base")
            for side in order:
                cells.append((f"{side}-{workload}-rep{repetition}", side, workload, "success"))
    for workload in GATES:
        for side in PROFILES:
            cells.append((f"{side}-{workload}-canary", side, workload, "canary"))
    return cells


def read_result(root, side, cell):
    results = list((root / "workloads/wasm-qualifications/v1").glob("result-*.json"))
    results = [path for path in results if not path.name.endswith(".publication.json")]
    if len(results) != 1:
        raise RuntimeError(f"{root}: expected one sealed Result, found {len(results)}")
    result = json.loads(results[0].read_text())
    qualification, manifest = result["Qualification"], result["Manifest"]
    expected_success = cell == "success"
    if (manifest["Provisional"] or manifest["Cell"] != cell
            or not qualification["deterministic"]
            or qualification["target_success"] != expected_success
            or qualification["qualified"] != expected_success
            or not result["Budget"]["WithinBudget"] or result["CostFailure"]):
        raise RuntimeError(f"{root}: final binding, outcome, determinism, cost or budget failed")
    if result["Cost"]["ReplayableBugs"] != int(not expected_success):
        raise RuntimeError(f"{root}: intentional-failure witness count changed")
    if len(qualification["executions"]) != 2:
        raise RuntimeError(f"{root}: two repeated executions were not retained")
    for execution in qualification["executions"]:
        replay = execution.get("replay")
        if not replay or not replay["attempted"] or not replay["match"]:
            raise RuntimeError(f"{root}: fresh retained replay did not match")
    target = qualification["evidence"]["target"]
    expected_tags = ["disable_grpc_modules", "sqlite3_dotlk", "test_dep"]
    if PROFILES[side] == "slim":
        expected_tags.append("gomad_wasm_slim")
    if target["build_tags"] != sorted(expected_tags):
        raise RuntimeError(f"{root}: actual provenance has wrong dependency profile")
    producers = list((root / "workloads").glob("*-producer.json"))
    if len(producers) != 1:
        raise RuntimeError(f"{root}: missing genuine preparation producer")
    producer = json.loads(producers[0].read_text())
    payload = producer["Prepared"]["BackendPayloads"][0]
    data = base64.b64decode(payload["Data"], validate=True)
    if "sha256:" + hashlib.sha256(data).hexdigest() != payload["Reference"]["sha256"]:
        raise RuntimeError(f"{root}: preparation provenance identity changed")
    provenance = json.loads(data)
    runtime = {key: provenance[key] for key in ("RuntimeSHA256", "RuntimeImplementationSHA256")}
    return results[0], result, runtime


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("checkout", type=Path)
    parser.add_argument("evidence", type=Path)
    parser.add_argument("--stock-go", type=Path, required=True)
    parser.add_argument("--helper", type=Path, required=True)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--inputs", type=Path, required=True)
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--maximum-load", type=float, default=1.5)
    parser.add_argument("--minimum-available", type=int, default=8 << 30)
    parser.add_argument("--run", action="store_true", help="dispatch only in the conductor's reserved host window")
    args = parser.parse_args()
    for path in (args.checkout, args.evidence, args.stock_go, args.helper, args.binary, args.inputs):
        if not path.is_absolute() or path != path.resolve():
            parser.error(f"explicit clean absolute non-symlink path required: {path}")
    if args.repetitions < 3:
        parser.error("at least three independent cold roots per side are required")
    for path in (args.stock_go, args.helper, args.binary):
        if not path.is_file() or not os.access(path, os.X_OK):
            parser.error(f"executable missing: {path}")
    if args.helper.stat().st_nlink != 1:
        parser.error("use a private regular single-link helper copy")
    for name in ("active-cluster-commit.request.json", "standby-cluster-commit.request.json", "two-cluster-broker-config.json"):
        if not (args.inputs / name).is_file():
            parser.error(f"configuration input missing: {name}")
    cells = schedule(args.repetitions)
    print(json.dumps({"dispatch": args.run, "order": cells}, indent=2), flush=True)
    if not args.run:
        return
    if platform.system() != "Linux" or platform.machine() != "aarch64":
        parser.error("current acceptance requires the real Linux/arm64 host")
    args.evidence.mkdir(mode=0o700)
    inventory = source_inventory(args.checkout)
    manifest = args.evidence / "source.sha256"
    manifest.write_text("".join(f"{sha}  {path}\n" for path, sha in sorted(inventory.items())))
    source_sha = "sha256:" + digest(manifest)
    executables = {str(path): digest(path) for path in (args.stock_go, args.helper, args.binary)}
    inputs = {path.name: digest(path) for path in args.inputs.glob("*.json")}
    write_json(args.evidence / "binding.json", {
        "head": command(["git", "rev-parse", "HEAD"], args.checkout),
        "status": command(["git", "status", "--porcelain"], args.checkout),
        "integrated_source_sha256": source_sha, "executables": executables,
        "compiler_version": command([str(args.stock_go), "version"], args.checkout),
        "historical_fn178_reference": "d087db1e48723d1164c19f559f78d75fc5731eb9",
        "baseline": "current source full dependency profile; fn178 workloads and architecture",
        "candidate": "identical source/runtime/helper with gomad_wasm_slim",
        "inputs": inputs,
        "schedule": cells,
    })
    write_json(args.evidence / "host.json", host_snapshot())
    fixture_root = args.checkout / ".tmp/fn182-current-cluster-fixture"
    fixture_root.mkdir(parents=True, exist_ok=True)
    wasm = args.checkout / "tools/gomad/internal/backends/wasm"
    for name, owner in (("recovery_test.go", "temporal_storage"), ("replication_test.go", "temporal_broker")):
        data = (wasm / "testdata" / owner / name).read_bytes()
        destination = fixture_root / name
        if destination.exists() and destination.read_bytes() != data:
            raise RuntimeError(f"refusing changed staged fixture {destination}")
        if not destination.exists():
            destination.write_bytes(data)
    observed = {}
    for label, side, workload, cell in cells:
        wait_for_host(args.maximum_load, args.minimum_available)
        if source_inventory(args.checkout) != inventory:
            raise RuntimeError("integrated sources changed during the reserved measurement window")
        if any(digest(Path(path)) != sha for path, sha in executables.items()):
            raise RuntimeError("compiler/helper/qualification binary changed")
        if {path.name: digest(path) for path in args.inputs.glob("*.json")} != inputs:
            raise RuntimeError("configuration-only producer inputs changed")
        root = args.evidence / label
        root.mkdir(mode=0o700)
        (root / "tmp").mkdir(mode=0o700)
        write_json(root / "host-before.json", host_snapshot())
        environment = {
            key: value for key, value in os.environ.items()
            if not key.startswith("GOMAD_WASM_") and key not in (
                "GOROOT", "GOMADSEED", "GOMAD_CHILD_SEED", "GOEXPERIMENT", "GOOS", "GOARCH",
                "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "BASH_ENV",
            )
        }
        environment.update({
            "GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOFLAGS": "",
            "TMPDIR": str(root / "tmp"), "GOMAD_STOCK_GO": str(args.stock_go),
            "GOMAD_WASM_TEST_HELPER": str(args.helper), "GOMAD_WASM_STOCK_GO": str(args.stock_go),
            "GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST": str(manifest),
            "GOMAD_WASM_INTEGRATED_SOURCE_SHA256": source_sha,
            "GOMAD_WASM_QUALIFICATION_ROOT": str(root / "workloads"),
            "GOMAD_WASM_QUALIFICATION_PROFILE": "cooperative",
            "GOMAD_WASM_QUALIFICATION_GUEST_PROFILE": PROFILES[side],
            "GOMAD_WASM_QUALIFICATION_CELL": cell,
            "GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS": str(args.inputs),
            "GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE": "./.tmp/fn182-current-cluster-fixture",
        })
        arguments = [str(args.binary), "-test.v", "-test.count=1", "-test.timeout=90m", "-test.run", "^" + GATES[workload] + "$"]
        write_json(root / "command.json", {"argv": arguments, "environment": environment})
        started = time.monotonic()
        print(f"START {label}", flush=True)
        with (root / "output.log").open("w") as output:
            status = subprocess.run(arguments, cwd=wasm / "qualification", env=environment, stdout=output, stderr=subprocess.STDOUT).returncode
        write_json(root / "exit.json", {"exit": status, "wall_seconds": time.monotonic() - started})
        write_json(root / "host-after.json", host_snapshot())
        print(f"EXIT {label} {status}", flush=True)
        if status:
            raise RuntimeError(f"{label} failed; retain this root and inspect output.log")
        path, result, runtime = read_result(root, side, cell)
        observed[label] = {"path": str(path), "module": result["Qualification"]["evidence"]["target"]["sha256"], "manifest": result["Manifest"], "runtime": runtime}
        reference = observed.setdefault(workload, observed[label])
        if reference["runtime"] != runtime:
            raise RuntimeError(f"{label}: cooperative runtime key or implementation differs")
        for field in BOUND_FIELDS:
            if reference["manifest"][field] != result["Manifest"][field]:
                raise RuntimeError(f"{label}: unmatched {field}")
        write_json(args.evidence / "observed.json", observed)
    if source_inventory(args.checkout) != inventory:
        raise RuntimeError("integrated sources changed before final acceptance")
    summary = Path(__file__).resolve().parents[1] / "fn182-guest-qualification-2026-10-15/summarize.py"
    with (args.evidence / "summary.md").open("w") as output:
        subprocess.run([sys.executable, "-I", str(summary), str(args.evidence)], stdout=output, check=True)
    print(f"PASS retained measured cells and summary: {args.evidence}", flush=True)


if __name__ == "__main__":
    main()
