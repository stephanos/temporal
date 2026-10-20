"""Run the existing qualification cells for a matched full/slim comparison."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import stat
import subprocess
import sys
import time
from urllib.parse import urlsplit


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
MAX_SOURCE_ENTRIES = 20000
MAX_SOURCE_FILE_BYTES = 128 << 20
MAX_SOURCE_BYTES = 512 << 20
MAX_SOURCE_MANIFEST_BYTES = 16 << 20


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def command(args, cwd):
    return subprocess.check_output(args, cwd=cwd, env=gate_environment(os.environ), text=True).strip()


def gate_environment(inherited):
    allowed = ("PATH", "HOME", "USER", "LOGNAME", "LANG", "LC_ALL", "TZ",
               "TMPDIR", "TMP", "TEMP", "XDG_CACHE_HOME", "GOCACHE", "GOMODCACHE", "GOPATH",
               "GOPROXY", "GOSUMDB", "GOPRIVATE", "GONOPROXY", "GONOSUMDB",
               "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
               "http_proxy", "https_proxy", "all_proxy", "no_proxy",
               "SSL_CERT_FILE", "SSL_CERT_DIR")
    environment = {key: inherited[key] for key in allowed if key in inherited}
    for key, value in environment.items():
        if not isinstance(value, str) or any(character in value for character in "\0\r\n"):
            raise ValueError("refusing malformed environment setting " + key)
    if not environment.get("PATH") or any(not Path(part).is_absolute() for part in environment["PATH"].split(os.pathsep)):
        raise ValueError("PATH must contain explicit absolute directories")
    for key in ("GOPROXY", "GOSUMDB", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
                "http_proxy", "https_proxy", "all_proxy", "no_proxy"):
        value = environment.get(key, "")
        if not value:
            continue
        if any(character in value for character in "@?#"):
            raise ValueError("refusing auth-bearing proxy setting " + key)
        if key.lower() == "no_proxy":
            continue
        if key == "GOSUMDB":
            parts = value.split()
            if len(parts) not in (1, 2) or ":" in parts[0] or parts[0] == "off" and len(parts) != 1:
                raise ValueError("refusing malformed proxy setting " + key)
            values = parts[1:]
        else:
            values = re.split(r"[,|]", value) if key == "GOPROXY" else [value]
        for entry in values:
            if key == "GOPROXY" and entry in ("direct", "off"):
                continue
            try:
                url = urlsplit(entry if "://" in entry else "//" + entry)
                schemes = ("http", "https") if key in ("GOPROXY", "GOSUMDB") else ("", "http", "https", "socks5", "socks5h")
                valid_file = key == "GOPROXY" and url.scheme == "file" and not url.netloc and url.path.startswith("/")
                valid_host = (url.scheme in schemes and url.hostname
                              and not any(character in url.hostname for character in "%\\") and url.port != 0)
                valid = bool(entry) and not any(character.isspace() for character in entry) and (valid_file or valid_host)
            except ValueError:
                valid = False
            if not valid:
                raise ValueError("refusing malformed proxy setting " + key)
    environment.update({"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOFLAGS": "", "GOMAXPROCS": "2"})
    return environment


def source_inventory(checkout, links_path=None):
    names = subprocess.check_output([
        "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard",
        "--", ".", ":!.flow",
    ], cwd=checkout, env=gate_environment(os.environ)).split(b"\0")
    names = sorted(set(name for name in names if name))
    if not names or len(names) > MAX_SOURCE_ENTRIES:
        raise RuntimeError("integrated source inventory entry limit")
    staged = subprocess.check_output([
        "git", "ls-files", "--stage", "-z", "--cached", "--", ".", ":!.flow",
    ], cwd=checkout, env=gate_environment(os.environ)).split(b"\0")
    index = {}
    for entry in (entry for entry in staged if entry):
        try:
            metadata, name = entry.split(b"\t", 1)
            mode, oid, stage = metadata.split(b" ")
        except ValueError as error:
            raise RuntimeError("malformed integrated source index") from error
        if (mode not in (b"100644", b"100755", b"120000", b"160000") or stage != b"0"
                or not re.fullmatch(rb"(?:[0-9a-f]{40}|[0-9a-f]{64})", oid) or name in index):
            raise RuntimeError("unsupported integrated source index entry")
        index[name] = (mode.decode(), oid.decode())
    if len(index) > MAX_SOURCE_ENTRIES:
        raise RuntimeError("integrated source inventory entry limit")
    files, links, gitlinks, total = {}, {}, {}, 0
    paths = [(checkout / os.fsdecode(name), True) for name in names]
    if links_path is not None:
        paths.append((links_path, False))
    for path, source in paths:
        try:
            resolved = path.resolve(strict=True)
            if (source and not resolved.is_relative_to(checkout)
                    or path.parent.resolve(strict=True) != path.parent
                    or not source and resolved != path
                    or any(character in str(path) + str(resolved) for character in "\0\r\n")):
                raise RuntimeError("integrated source path escapes or aliases its owner")
            info = resolved.lstat()
            mode, oid = index.get(os.fsencode(str(path.relative_to(checkout))), (None, None)) if source else (None, None)
            if mode == "160000":
                if resolved != path or not stat.S_ISDIR(path.lstat().st_mode) or any(path.iterdir()):
                    raise RuntimeError("tracked gitlink must be an empty uninitialized directory")
                name = str(path.relative_to(checkout))
                gitlinks[name] = {"path": name, "mode": mode, "oid": oid, "state": "empty-uninitialized"}
                continue
            if not stat.S_ISREG(info.st_mode):
                raise RuntimeError("integrated source must resolve to a regular file")
            if str(resolved) not in files:
                total += info.st_size
                if info.st_size > MAX_SOURCE_FILE_BYTES or total > MAX_SOURCE_BYTES:
                    raise RuntimeError("integrated source inventory byte limit")
                files[str(resolved)] = digest(resolved)
            current = path
            while current.is_symlink():
                target = os.readlink(current)
                links[str(current.relative_to(checkout))] = {
                    "path": str(current.relative_to(checkout)), "target": target,
                    "resolved": str(resolved.relative_to(checkout)), "sha256": files[str(resolved)],
                }
                current = Path(os.path.abspath(current.parent / target))
                if not current.is_relative_to(checkout) or current.parent.resolve(strict=True) != current.parent:
                    raise RuntimeError("integrated source link escapes or aliases its owner")
        except (OSError, ValueError) as error:
            raise RuntimeError("invalid integrated source path " + str(path)) from error
    if len(files) > MAX_SOURCE_ENTRIES or len(links) + len(gitlinks) > MAX_SOURCE_ENTRIES:
        raise RuntimeError("integrated source inventory entry limit")
    return {"files": files, "links": [links[name] for name in sorted(links)],
            "gitlinks": [gitlinks[name] for name in sorted(gitlinks)]}


def write_source_manifest(checkout, root):
    inventory = source_inventory(checkout)
    links_path = root / "source-links.json"
    write_json(links_path, {"schema": "gomad.measurement-source-links/v1", "links": inventory["links"], "gitlinks": inventory["gitlinks"]})
    sealed = source_inventory(checkout, links_path)
    if (sealed["links"] != inventory["links"] or sealed["gitlinks"] != inventory["gitlinks"]
            or {name: sha for name, sha in sealed["files"].items() if name != str(links_path)} != inventory["files"]):
        raise RuntimeError("integrated sources changed while binding links")
    manifest = root / "source.sha256"
    encoded = "".join(f"{sha}  {path}\n" for path, sha in sorted(sealed["files"].items()))
    if len(encoded.encode()) > MAX_SOURCE_MANIFEST_BYTES:
        raise RuntimeError("integrated source manifest byte limit")
    manifest.write_text(encoded)
    return sealed, manifest


def verify_source_inventory(checkout, inventory, manifest, source_sha):
    if (source_inventory(checkout, manifest.with_name("source-links.json")) != inventory
            or manifest.resolve(strict=True) != manifest or not stat.S_ISREG(manifest.lstat().st_mode)
            or "sha256:" + digest(manifest) != source_sha):
        raise RuntimeError("integrated sources changed during the reserved measurement window")


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
    inventory, manifest = write_source_manifest(args.checkout, args.evidence)
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
        verify_source_inventory(args.checkout, inventory, manifest, source_sha)
        if any(digest(Path(path)) != sha for path, sha in executables.items()):
            raise RuntimeError("compiler/helper/qualification binary changed")
        if {path.name: digest(path) for path in args.inputs.glob("*.json")} != inputs:
            raise RuntimeError("configuration-only producer inputs changed")
        root = args.evidence / label
        root.mkdir(mode=0o700)
        (root / "tmp").mkdir(mode=0o700)
        write_json(root / "host-before.json", host_snapshot())
        environment = gate_environment(os.environ)
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
    verify_source_inventory(args.checkout, inventory, manifest, source_sha)
    summary = Path(__file__).resolve().with_name("summarize.py")
    with (args.evidence / "summary.md").open("w") as output:
        subprocess.run([sys.executable, "-I", str(summary), str(args.evidence)], env=gate_environment(os.environ), stdout=output, check=True)
    print(f"PASS retained measured cells and summary: {args.evidence}", flush=True)


if __name__ == "__main__":
    main()
