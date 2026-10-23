#!/usr/bin/env python3
"""Execute an original mandatory cell on the current Rust release binary."""
import argparse
import base64
import hashlib
import json
import os
import posixpath
import re
from pathlib import Path
import subprocess
import sys
import time

sys.dont_write_bytecode = True
import workloads


def write(path, value):
    with path.open("xb") as stream:
        stream.write(workloads.canonical(value)+b"\n")
    path.chmod(0o600)


def owned_root(root):
    write(root/"producer-root.json", dict(schema="gomad.rust-producer-root/v1", root=str(root.resolve()),
          argv=sys.argv, cwd=os.getcwd(), pid=os.getpid(), start_ns=time.time_ns()))


def run(root, name, command, expected, json_output=True, environment=None):
    output = root/(name+".stdout")
    error = root/(name+".stderr")
    command = [str(item) for item in command]
    actual_env = dict(os.environ if environment is None else environment)
    observed_env = {key: actual_env.get(key) for key in (
        "PATH", "GOMAD_WASM_INSTALLATION", "GOMAD_STOCK_GO", "GOMAD_WASM_RUNTIME_ROOT",
        "GOMAD_WASM_QUALIFICATION_GUEST_PROFILE", "GOMAD_WASM_TEMPORAL_BUILD_CACHE",
        "GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES")}
    started = time.time_ns()
    with output.open("xb") as stdout, error.open("xb") as stderr:
        process = subprocess.Popen(command, stdout=stdout, stderr=stderr, env=actual_env)
        registered = dict(argv=command, pid=process.pid, stage_pid=os.getpid(), active=False,
                          start_ns=started)
        code = process.wait()
        ended = time.time_ns()
        registered.update(exit=code, observation="completed", end_ns=ended)
    write(root/(name+".process.json"), registered)
    receipt = dict(argv=command, command=command, argv_sha256=hashlib.sha256(workloads.canonical(command)).hexdigest(),
                   cwd=os.getcwd(), pid=process.pid, exit=code, observation="completed", start_ns=started,
                   end_ns=time.time_ns(), environment=observed_env,
                   environment_sha256=hashlib.sha256(workloads.canonical(actual_env)).hexdigest(),
                   stdout=str(output), stderr=str(error),
                   stdout_sha256=hashlib.sha256(output.read_bytes()).hexdigest(),
                   stderr_sha256=hashlib.sha256(error.read_bytes()).hexdigest())
    write(root/(name+".command.json"), receipt)
    if code != expected:
        raise ValueError(f"{name} exit {code}, expected {expected}; raw logs: {output}, {error}")
    if not json_output or not output.stat().st_size:
        return None
    try:
        return json.loads(output.read_bytes())
    except json.JSONDecodeError:
        return [json.loads(line) for line in output.read_bytes().splitlines() if line][-1]


def captured(repo, path):
    """Authenticate original schema bytes against this candidate before reuse."""
    request = json.loads(path.read_bytes())
    value = request.get("Config", request)["CapturedInputs"]
    descriptor_bytes = base64.b64decode(value["Descriptor"], validate=True)
    manifest = value["Manifest"]
    if manifest["SHA256"] != "sha256:"+hashlib.sha256(descriptor_bytes).hexdigest() or manifest["Bytes"] != len(descriptor_bytes):
        raise ValueError("captured schema descriptor changed")
    descriptor = json.loads(descriptor_bytes)
    seen = set()
    for entry in descriptor["entries"]:
        guest = entry["path"]
        prefix = "/workspace/schema/sqlite"
        if guest != prefix and not guest.startswith(prefix+"/"):
            raise ValueError("foreign captured schema mount")
        relative = Path(guest.removeprefix("/workspace/"))
        if ".." in relative.parts:
            raise ValueError("unclean captured schema path")
        source = repo/relative
        if source.is_symlink():
            raise ValueError("schema input became a symlink")
        seen.add(str(relative))
        if entry["kind"] == "file":
            body = source.read_bytes()
            payload = base64.b64decode(value["Payloads"][entry["payload"]], validate=True)
            if body != payload or int(entry["size"]) != len(body) or entry["sha256"] != "sha256:"+hashlib.sha256(body).hexdigest():
                raise ValueError("captured schema differs from current unchanged source")
        elif entry["kind"] == "directory":
            actual = sorted(child.name for child in source.iterdir())
            if actual != sorted(child["name"] for child in entry["children"]):
                raise ValueError("schema directory children changed")
        else:
            raise ValueError("unsupported captured schema kind")
    actual = {str(path.relative_to(repo)) for path in (repo/"schema/sqlite").rglob("*")}
    actual.add("schema/sqlite")
    if seen != actual:
        raise ValueError("captured schema inventory differs from candidate")
    return value


def qualification_build_cache(root, override=""):
    cache = override or str(Path(root)/"temporal-cluster-build-cache")
    if not isinstance(cache, str) or not cache.startswith("/") or cache == "/" or cache.startswith("//") or posixpath.normpath(cache) != cache:
        raise ValueError("explicit absolute non-root verified compiler cache required")
    return Path(cache)


def memory_decimal(value):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9]{1,20}", value) or int(value) > (1 << 64)-1:
        raise ValueError("bounded unsigned decimal memory value required")
    return int(value)


def memory_admission(platform, path, minimum, retained_raw=None):
    observation = dict(minimum_bytes=0, available_bytes=0)
    if minimum == "":
        return observation
    value = memory_decimal(minimum)
    if value == 0:
        raise ValueError("positive decimal memory admission threshold required")
    observation["minimum_bytes"] = value
    if platform != "linux":
        raise ValueError("explicit memory admission requires Linux MemAvailable")
    with Path(path).open("rb") as stream:
        raw = stream.read((8 << 10)+1)
    if retained_raw is not None:
        with Path(retained_raw).open("xb") as retained:
            retained.write(raw)
        Path(retained_raw).chmod(0o600)
    if len(raw) > 8 << 10:
        raise ValueError("MemAvailable observation exceeds 8KiB")
    seen = False
    for line in raw.decode("utf-8", errors="replace").split("\n"):
        fields = re.findall(r"[^\t\n\v\f\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+", line)
        if not fields or fields[0].split(":", 1)[0].casefold() != "memavailable":
            continue
        if seen or fields[0] != "MemAvailable:" or len(fields) != 3 or fields[2] != "kB":
            raise ValueError("unique exact MemAvailable field in kB required")
        seen = True
        kilobytes = memory_decimal(fields[1])
        if kilobytes > ((1 << 64)-1)//1024:
            raise ValueError("MemAvailable byte conversion overflow")
        observation["available_bytes"] = kilobytes*1024
    if not seen:
        raise ValueError("MemAvailable observation missing")
    if observation["available_bytes"] < value:
        raise ValueError("insufficient available memory before helper dispatch")
    return observation


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--root", required=True, type=Path)
    parser.add_argument("--compiler", required=True, type=Path)
    parser.add_argument("--runtime-root", required=True, type=Path)
    parser.add_argument("--cache", required=True, type=Path)
    parser.add_argument("--workload", required=True, choices=["sqlite", "replication", "recovery", "retry", "smoke", "frontend"])
    parser.add_argument("--mode", required=True, choices=["success", "canary"])
    parser.add_argument("--source-manifest", type=Path)
    parser.add_argument("--captured-config", type=Path)
    args = parser.parse_args()
    repo, root, binary = args.repo.resolve(), args.root.resolve(), args.binary.resolve()
    root.mkdir(parents=True, mode=0o700, exist_ok=False)
    owned_root(root)
    guest_profile = os.environ.get("GOMAD_WASM_QUALIFICATION_GUEST_PROFILE", "")
    if args.workload != "sqlite":
        workloads.guest_build_tags(guest_profile)
        override = os.environ.get("GOMAD_WASM_TEMPORAL_BUILD_CACHE", "")
        cache = qualification_build_cache(root, override or str(args.cache))
        if override and cache != args.cache:
            raise ValueError("private compiler-cache override differs from frozen --cache")
        minimum = os.environ.get("GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES", "")
        controls = dict(guest_profile=guest_profile, build_cache=str(cache), memory_minimum=minimum,
                        memory_platform=sys.platform, memory_path="/proc/meminfo")
        try:
            controls["memory"] = memory_admission(sys.platform, Path("/proc/meminfo"), minimum,
                                                   retained_raw=root/"host-memory.raw")
        except (ValueError, OSError) as error:
            controls["memory_error"] = str(error)
            write(root/"qualification-controls.json", controls)
            raise
        write(root/"qualification-controls.json", controls)
    else:
        cache = args.cache
    capture = None
    source = None
    if args.workload != "sqlite":
        if not args.captured_config:
            raise ValueError("receiver-owned configuration preimage required")
        capture = captured(repo, args.captured_config)
        if args.workload in ("replication","recovery","retry"):
            names={"replication":["temporal_storage/recovery_test.go","temporal_broker/replication_test.go"],"recovery":["temporal_storage/recovery_test.go"],"retry":["network_streams/temporal_retry_test.go"]}[args.workload]
            body=b"".join((repo/workloads.WASM/"testdata"/name).read_bytes() for name in names)
            stage=repo/".tmp"/("fn232-temporal-"+args.workload+"-"+hashlib.sha256(body).hexdigest()[:16])
            write(root/"unchanged-source-inputs.json",workloads.stage_temporal(repo,stage,args.workload))
            source="./"+str(stage.relative_to(repo))
    policy = root/"policy.json"
    write(policy, workloads.support_policy(args.workload,capture) if args.workload in workloads.SUPPORT_ROWS else workloads.policy(args.workload, capture))
    installation = root/"installation.json"
    run(root, "install", [str(binary), "install", "--compiler", str(args.compiler.resolve()), "--runtime-root", str(args.runtime_root.resolve()),
        "--cache", str(cache.resolve()), "--output", str(installation), "--profile", "gomad.wasi-cooperative/v1", "--config", str(policy),
        "--fuel", str(1 << 50 if args.workload != "sqlite" else 4_000_000_000), "--memory-bytes", str(4 << 30 if args.workload != "sqlite" else 256 << 20)], 0, json_output=False)
    spec = root/"qualification-spec.json"
    write(spec, (workloads.support_spec if args.workload in workloads.SUPPORT_ROWS else workloads.qualification_spec)(repo, root/"cell", args.workload, args.mode == "canary", source, args.source_manifest, guest_profile=guest_profile))
    response = run(root, "qualify", [str(binary), "--installation", str(installation), "qualify", "--spec", str(spec)], 1 if args.mode == "canary" else 0)
    result_path = Path(response["result_path"])
    report = run(root, "report", [str(binary), "report", "--result", str(result_path)], 0)
    retained = json.loads(result_path.read_bytes())
    actual = retained["report"]
    if not actual["deterministic"] or actual["target_success"] != (args.mode == "success") or len(actual["executions"]) != 2:
        raise ValueError("actual mandatory workload result did not satisfy its original verdict")
    if any(not item["artifact_path"] or not item["replay"]["matched"] or not item["replay"]["attempted"] for item in actual["executions"]):
        raise ValueError("actual mandatory workload omitted retained exact replay")
    if not retained["budget"] or not retained["budget"]["within_budget"]:
        raise ValueError("actual mandatory workload budget is unavailable or failed")
    if not retained["cost"] or retained["cost"]["replayable_bugs"] != (1 if args.mode == "canary" else 0):
        raise ValueError("actual mandatory workload bug count differs from original cell")
    write(root/"producer.json", dict(schema="gomad.rust-mandatory-workload-producer/v1", result_path=str(result_path),
        result_sha256=hashlib.sha256(result_path.read_bytes()).hexdigest(), workload=args.workload, mode=args.mode, report=report))
    print(json.dumps(dict(result_path=str(result_path), producer=str(root/"producer.json"))))


if __name__ == "__main__":
    main()
