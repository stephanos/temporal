#!/usr/bin/env python3
import argparse, fcntl, hashlib, json, os, shutil, subprocess, time
from pathlib import Path
CONFIG = json.loads(Path(__file__).resolve().with_name("launch-plan.json").read_text())
ROOT = Path(CONFIG["pair"]).resolve()
TARGET = Path(CONFIG["target"])
STATE = ROOT / "execution-state.json"
def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def checked(*args):
    return subprocess.check_output(args, text=True).strip()
def environment(step):
    env = os.environ.copy()
    for key in CONFIG["clear_environment"]:
        env.pop(key, None)
    env.update(CONFIG["base_environment"])
    env.update(CARGO_TARGET_DIR=str(TARGET), GOCACHE=str(TARGET / "go-build"), GOWORK=str(ROOT / "go.work"), GOTOOLCHAIN="local")
    trusted = env.get("MISE_TRUSTED_CONFIG_PATHS", "")
    env["MISE_TRUSTED_CONFIG_PATHS"] = os.pathsep.join(x for x in [trusted, str(ROOT / "temporal"), str(ROOT / "umpire")] if x)
    for key, value in step.get("environment", {}).items():
        if value is None:
            env.pop(key, None)
        else:
            env[key] = value
    return env
def preflight(step_id, reserve):
    sources = {}
    for name, expected in CONFIG["expected"].items():
        repo = ROOT / name
        head = checked("git", "-C", str(repo), "rev-parse", "HEAD")
        status = checked("git", "-C", str(repo), "status", "--porcelain")
        if head != expected or status or (repo / ".git/objects/info/alternates").exists():
            raise SystemExit(f"refuse changed or non-independent source: {name} {head} {status}")
        sources[name] = head
    guest = ROOT / "temporal/model/temporal/hosts/go/umpire_temporal_guest.wasm"
    if sha(guest) != CONFIG["guest_sha256"]:
        raise SystemExit("canonical guest changed; investigate before running")
    if not STATE.exists() and TARGET.exists() and any(TARGET.iterdir()):
        raise SystemExit("fresh paired target already contains files; no artifact reuse admitted")
    free = shutil.disk_usage(TARGET.parent).free
    if free - reserve * 1024**3 < CONFIG["disk_floor_bytes"]:
        raise SystemExit(f"insufficient target-filesystem headroom: free={free}, reserved GiB={reserve}, floor30GiB")
    return {"sources": sources, "target": str(TARGET), "target_exists": TARGET.exists(), "free_bytes": free, "additional_reservation_gib": reserve, "guest_sha256": sha(guest), "workspace_sha256": sha(ROOT / "go.work"), "step": step_id}
def run_command(cmd, cwd, env, evidence, ordinal, phase):
    log = evidence / f"{ordinal:02}-{phase}.log"
    before = shutil.disk_usage(TARGET.parent).free
    start = time.time()
    with log.open("w") as stream:
        stream.write("argv: " + json.dumps(cmd) + "\ncwd: " + cwd + "\n")
        stream.flush()
        result = subprocess.run(cmd, cwd=cwd, env=env, stdout=stream, stderr=subprocess.STDOUT)
    record = {"argv": cmd, "cwd": cwd, "started_unix": start, "ended_unix": time.time(), "exit": result.returncode, "log": str(log), "log_sha256": sha(log), "free_before": before, "free_after": shutil.disk_usage(TARGET.parent).free}
    (evidence / f"{ordinal:02}-{phase}.json").write_text(json.dumps(record, indent=2) + "\n")
    if result.returncode:
        raise SystemExit(result.returncode)
    return record
parser = argparse.ArgumentParser()
parser.add_argument("step", choices=CONFIG["steps"])
parser.add_argument("--execute", action="store_true")
parser.add_argument("--reserve-gib", type=int)
args = parser.parse_args()
step = CONFIG["steps"][args.step]
reserve = args.reserve_gib if args.reserve_gib is not None else CONFIG["phase_budget_default_gib"].get(args.step, 0)
if reserve < 0:
    raise SystemExit("negative reservation refused")
if not args.execute:
    print(json.dumps({"preflight": preflight(args.step, reserve), "planned_commands": step.get("commands", []), "procedure": step.get("procedure", []), "execution": "NOT RUN; explicit --execute plus matching FN258_TASK4_ADMIT_STEP required"}, indent=2))
    raise SystemExit(0)
if os.environ.get("FN258_TASK4_ADMIT_STEP") != args.step:
    raise SystemExit("root phase admission missing: FN258_TASK4_ADMIT_STEP must match requested step")
def require_prior_default():
    summaries = sorted((ROOT / "execution-evidence").glob("paired-default-check-*/summary.json"))
    valid = [json.loads(path.read_text()) for path in summaries]
    if not any(item.get("before", {}).get("sources") == CONFIG["expected"] and item.get("commands") and all(cmd.get("exit") == 0 for cmd in item["commands"]) for item in valid):
        raise SystemExit("paired TLC setup requires successful default gate at this exact pair")
def require_tlc_setup():
    receipt = ROOT / "tlc-setup-receipt.json"
    if not receipt.is_file():
        raise SystemExit("run documented paired-tlc-setup before paired-required-validate")
    value = json.loads(receipt.read_text())
    jar = Path(CONFIG["tlc_setup"]["artifact"])
    if value.get("status") != "PASS" or value.get("sources") != CONFIG["expected"] or jar.is_symlink() or not jar.is_file() or sha(jar) != CONFIG["tlc_setup"]["sha256"]:
        raise SystemExit("paired TLC setup receipt or current jar failed exact-pair checksum admission")
if args.step == "paired-tlc-setup":
    require_prior_default()
if args.step == "paired-required-validate":
    require_tlc_setup()
with (ROOT / "execution.lock").open("a") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    before = preflight(args.step, reserve)
    if not STATE.exists():
        STATE.write_text(json.dumps({"initial_target_absent_or_empty": True, "first_step": args.step, "sources": before["sources"], "started_unix": time.time()}, indent=2) + "\n")
    evidence = ROOT / "execution-evidence" / (args.step + "-" + str(time.time_ns()))
    evidence.mkdir(parents=True)
    env = environment(step)
    selected = set(CONFIG["base_environment"]) | set(CONFIG["clear_environment"]) | {"GOWORK", "GOCACHE", "GOMODCACHE", "CARGO_HOME", "RUSTUP_HOME", "PATH", "TMPDIR", "MISE_TRUSTED_CONFIG_PATHS"}
    (evidence / "environment.json").write_text(json.dumps({key: env.get(key) for key in sorted(selected)}, indent=2) + "\n")
    results = []
    commands = step.get("commands", [])
    if args.step == "warm-activity-quick":
        path = ROOT / "temporal/model/temporal/feature/activity/model.rs"
        original = path.read_bytes()
        results.append(run_command(commands[0], step["cwd"], env, evidence, 0, "warmup-setup-not-edited-timing"))
        try:
            path.write_bytes(original + b"\n// fn258 semantic-neutral Activity timing edit\n")
            results.append(run_command(commands[0], step["cwd"], env, evidence, 1, "edited-activity-timing"))
        finally:
            path.write_bytes(original)
            if path.read_bytes() != original:
                raise SystemExit("failed exact Activity source restoration")
    elif args.step == "paired-tlc-setup":
        try:
            for index, command in enumerate(commands):
                results.append(run_command(command, step["cwd"], env, evidence, index, args.step))
        finally:
            jar = Path(CONFIG["tlc_setup"]["artifact"])
            regular = jar.is_file() and not jar.is_symlink() and jar.resolve().is_relative_to(ROOT / "umpire")
            actual = sha(jar) if regular else None
            records = [json.loads(path.read_text()) for path in sorted(evidence.glob("[0-9][0-9]-paired-tlc-setup.json"))]
            passed = len(records) == len(commands) and all(record["exit"] == 0 for record in records) and actual == CONFIG["tlc_setup"]["sha256"]
            setup = {"status": "PASS" if passed else "FAIL", "sources": CONFIG["expected"], "artifact": str(jar), "expected_sha256": CONFIG["tlc_setup"]["sha256"], "actual_sha256": actual, "regular_contained_file": regular, "command_records": records, "seconds": sum(record["ended_unix"] - record["started_unix"] for record in records), "method": "candidate model/scripts/fetch-tlc.sh; documented manual setup, no production gate edge"}
            (evidence / "tlc-setup.json").write_text(json.dumps(setup, indent=2) + "\n")
            if passed:
                (ROOT / "tlc-setup-receipt.json").write_text(json.dumps(setup, indent=2) + "\n")
        if not passed:
            raise SystemExit("paired TLC setup failed checksum/status admission")
    else:
        if args.step == "current-guest-identity":
            (TARGET / "proof").mkdir(parents=True, exist_ok=True)
        for index, command in enumerate(commands):
            results.append(run_command(command, step["cwd"], env, evidence, index, args.step))
    after = {name: {"head": checked("git", "-C", str(ROOT / name), "rev-parse", "HEAD"), "status": checked("git", "-C", str(ROOT / name), "status", "--porcelain")} for name in CONFIG["expected"]}
    summary = {"before": before, "after": after, "commands": results, "guest_sha256": sha(ROOT / "temporal/model/temporal/hosts/go/umpire_temporal_guest.wasm"), "scope": "Command receipts only; final acceptance/Flow state and proof recovery remain root-controlled."}
    (evidence / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(evidence)
