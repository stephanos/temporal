#!/usr/bin/env python3
import argparse, fcntl, hashlib, json, os, shutil, subprocess, time
from pathlib import Path
ROOT = Path(__file__).resolve().parent
CONFIG = json.loads((ROOT / "launch-plan.json").read_text())
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
    else:
        if args.step == "current-guest-identity":
            (TARGET / "proof").mkdir(parents=True, exist_ok=True)
        for index, command in enumerate(commands):
            results.append(run_command(command, step["cwd"], env, evidence, index, args.step))
    after = {name: {"head": checked("git", "-C", str(ROOT / name), "rev-parse", "HEAD"), "status": checked("git", "-C", str(ROOT / name), "status", "--porcelain")} for name in CONFIG["expected"]}
    summary = {"before": before, "after": after, "commands": results, "guest_sha256": sha(ROOT / "temporal/model/temporal/hosts/go/umpire_temporal_guest.wasm"), "scope": "Command receipts only; final acceptance/Flow state and proof recovery remain root-controlled."}
    (evidence / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(evidence)
