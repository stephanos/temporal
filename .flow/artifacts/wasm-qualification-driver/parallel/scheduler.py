#!/usr/bin/env python3
"""Resource-aware DAG scheduler for the parallel WASM validation ladder.

The scheduler only decides *when* a stage runs. Every stage body (command,
budget, deadline, seed, oracle and post-check) lives in lib.sh and is executed
by the runner command, one process per stage: `RUNNER... <label>`.

Admission rules (see README.md "Scheduling policy"):
  * a stage starts only after all of its dependencies exited 0;
  * declared cores and peak memory of running stages stay within the caps
    (a stage that alone exceeds a cap may start only on an idle host);
  * named locks are mutually exclusive;
  * while a `memguard` stage runs, the others leave it 8 GiB of the cap;
  * `solo` stages run with nothing else; `exclusive` stages run with no other
    stage, except `bg` stages when --allow-background is given and the
    exclusive stage is not `cleantree`;
  * a stage with in-tree scratch never overlaps a `cleantree` stage;
  * when the highest-priority ready stage is exclusive/solo and blocked, no
    lower-priority stage is admitted (drain), so exclusives cannot starve.
Fail-fast: after the first non-zero exit no new stage starts; running stages
finish and are recorded. Past the wall-clock deadline the run continues to
completion (complete evidence) and exits 3; --stop-at-deadline stops admitting.

Exit codes: 0 all stages passed; 1 a stage failed; 3 deadline exceeded
(no failure); 2 usage/DAG error.
"""
import argparse
import json
import os
import signal
import subprocess
import sys
import time

EXIT_OK, EXIT_STAGE_FAILED, EXIT_USAGE, EXIT_WALL = 0, 1, 2, 3
CLASSES = ("normal", "exclusive", "solo")
FLAGS = ("bg", "cleantree", "memguard")
# A `memguard` stage refuses to dispatch unless MemAvailable >= 8 GiB (the
# matched-preparation cold memory guard): other stages may then use at most
# cap - 8 GiB while it runs.
GUARD_GIB = 8.0
COLUMNS = ["label", "deps", "class", "cores", "mem_gib", "est_s", "locks", "scratch", "flags", "tier"]


class DagError(Exception):
    pass


class Stage:
    def __init__(self, row):
        self.label = row["label"]
        self.deps = _list(row["deps"])
        self.cls = row["class"]
        self.cores = float(row["cores"])
        self.mem = float(row["mem_gib"])
        self.est = float(row["est_s"])
        self.locks = _list(row["locks"])
        self.scratch = _list(row["scratch"])
        self.flags = set(_list(row["flags"]))
        self.tier = int(row["tier"])
        self.rank = 0.0
        if self.cls not in CLASSES:
            raise DagError(f"{self.label}: unknown class {self.cls}")
        if self.flags - set(FLAGS):
            raise DagError(f"{self.label}: unknown flags {sorted(self.flags - set(FLAGS))}")
        if self.cores <= 0 or self.mem < 0 or self.est < 0:
            raise DagError(f"{self.label}: invalid resources")


def _list(value):
    value = value.strip()
    return [] if value in ("", "-") else [item for item in value.split(",") if item]


def load_dag(path):
    stages = {}
    with open(path) as handle:
        rows = [line.rstrip("\n") for line in handle if line.strip() and not line.startswith("#")]
    header = rows[0].split("\t")
    if header != COLUMNS:
        raise DagError(f"DAG header must be {COLUMNS}, got {header}")
    for line in rows[1:]:
        fields = line.split("\t")
        if len(fields) != len(COLUMNS):
            raise DagError(f"bad DAG row: {line!r}")
        stage = Stage(dict(zip(COLUMNS, fields)))
        if stage.label in stages:
            raise DagError(f"duplicate stage {stage.label}")
        stages[stage.label] = stage
    return finalize(stages)


def finalize(stages):
    for stage in stages.values():
        for dep in stage.deps:
            if dep not in stages:
                raise DagError(f"{stage.label}: unknown dependency {dep}")
    order = topological(stages)
    for label in reversed(order):
        stage = stages[label]
        successors = [s.rank for s in stages.values() if label in s.deps]
        stage.rank = stage.est + max(successors, default=0.0)
    return stages


def topological(stages):
    state, order = {}, []

    def visit(label, path):
        if state.get(label) == "done":
            return
        if state.get(label) == "active":
            raise DagError("dependency cycle: " + " -> ".join(path + [label]))
        state[label] = "active"
        for dep in stages[label].deps:
            visit(dep, path + [label])
        state[label] = "done"
        order.append(label)

    for label in sorted(stages):
        visit(label, [])
    return order


def static_critical_path(stages):
    """Longest dependency chain by estimated duration (resource-free lower bound)."""
    finish, via = {}, {}
    for label in topological(stages):
        stage = stages[label]
        best = max(stage.deps, key=lambda d: finish[d], default=None)
        finish[label] = stage.est + (finish[best] if best else 0.0)
        via[label] = best
    last = max(finish, key=finish.get)
    path = []
    while last:
        path.append(last)
        last = via[last]
    return finish[max(finish, key=finish.get)], list(reversed(path))


class Policy:
    def __init__(self, cores, mem_gib, allow_background=False, deadline=None, stop_at_deadline=False,
                 live_mem=None, margin=0.0):
        self.cores = cores
        self.mem = mem_gib
        self.allow_background = allow_background
        self.deadline = deadline
        self.stop_at_deadline = stop_at_deadline
        self.live_mem = live_mem  # callable -> current MemAvailable GiB (real runs only)
        self.margin = margin


def can_start(stage, running, policy):
    """running: list of Stage currently executing."""
    if not running:
        return True
    if stage.cls == "solo" or any(r.cls == "solo" for r in running):
        return False
    exclusive_running = [r for r in running if r.cls == "exclusive"]
    if stage.cls == "exclusive":
        if exclusive_running:
            return False
        if not policy.allow_background or "cleantree" in stage.flags:
            return False
        if any("bg" not in r.flags for r in running):
            return False
    elif exclusive_running:
        if not policy.allow_background or "bg" not in stage.flags:
            return False
        if any("cleantree" in r.flags for r in exclusive_running):
            return False
    if "cleantree" in stage.flags and any(r.scratch for r in running):
        return False
    if stage.scratch and any("cleantree" in r.flags for r in running):
        return False
    held = {lock for r in running for lock in r.locks}
    if held.intersection(stage.locks):
        return False
    if sum(r.cores for r in running) + stage.cores > policy.cores:
        return False
    if sum(r.mem for r in running) + stage.mem > policy.mem:
        return False
    if "memguard" in stage.flags and sum(r.mem for r in running) + GUARD_GIB > policy.mem:
        return False
    for guarded in (r for r in running if "memguard" in r.flags):
        if sum(r.mem for r in running if r is not guarded) + stage.mem + GUARD_GIB > policy.mem:
            return False
    if "memguard" in stage.flags and policy.live_mem is not None:
        # Declared peaks are estimates: also require the guard's headroom now.
        if policy.live_mem() < GUARD_GIB + policy.margin + stage.mem:
            return False
    return True


def priority(stage):
    # Tier 0 (cheap static checks) first for fail-fast; then longest remaining path.
    return (0 if stage.tier == 0 else 1, -stage.rank, stage.label)


def select(stages, done, running, started, policy, now=0.0, started_at=None):
    """Return the stages to start now, in priority order.

    When the highest-priority ready stage is exclusive/solo and blocked, it
    reserves the host: other stages are admitted only if their estimated end
    does not pass the moment the running stages are expected to finish
    (backfilling that cannot delay the exclusive stage)."""
    started_at = started_at or {}
    ready = sorted((s for s in stages.values() if s.label not in started and all(d in done for d in s.deps)),
                   key=priority)
    chosen = []
    active = list(running)
    shadow = None
    for index, stage in enumerate(ready):
        if shadow is not None and (stage.cls != "normal" or now + stage.est > shadow):
            continue
        if can_start(stage, active, policy):
            chosen.append(stage)
            active.append(stage)
            continue
        if stage.cls in ("exclusive", "solo") and index == 0 and not chosen:
            shadow = max((started_at.get(r.label, now) + r.est for r in running), default=now)
            shadow = max(shadow, now)
    return chosen


class RealExecutor:
    def __init__(self, runner, state_dir, env):
        self.runner = runner
        self.state_dir = state_dir
        self.env = env
        self.procs = {}
        self.registry = os.path.join(state_dir, "registry")
        os.makedirs(self.registry, exist_ok=True)

    def now(self):
        return time.time()

    def _register(self, stage, **fields):
        path = os.path.join(self.registry, stage.label + ".json")
        record = {}
        if os.path.exists(path):
            with open(path) as handle:
                record = json.load(handle)
        record.update(fields)
        temporary = path + ".tmp"
        with open(temporary, "w") as handle:
            json.dump(record, handle)
        os.replace(temporary, path)

    def start(self, stage, lane):
        # Registration precedes the process, so any in-tree scratch it creates is
        # always covered by a registry interval when another stage observes it.
        self._register(stage, label=stage.label, scratch=stage.scratch, start_ns=time.time_ns(), lane=lane)
        env = dict(self.env, QUALDRV_LANE=str(lane), QUALDRV_STAGE=stage.label)
        log = open(os.path.join(self.state_dir, stage.label + ".runner.log"), "w")
        blocked = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGTERM, signal.SIGHUP, signal.SIGINT})
        try:
            self.procs[stage.label] = (subprocess.Popen(self.runner + [stage.label], env=env, stdout=log,
                                                        stderr=subprocess.STDOUT, start_new_session=True), log, stage)
        finally:
            signal.pthread_sigmask(signal.SIG_SETMASK, blocked)

    def terminate(self):
        """Interrupted driver: stop every running stage's whole process group."""
        for proc, _, _ in self.procs.values():
            try:
                os.killpg(proc.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
        deadline = time.time() + 30
        for proc, _, _ in self.procs.values():
            try:
                proc.wait(max(0.1, deadline - time.time()))
            except subprocess.TimeoutExpired:
                os.killpg(proc.pid, signal.SIGKILL)

    def wait_any(self):
        while True:
            for label, (proc, log, stage) in list(self.procs.items()):
                code = proc.poll()
                if code is not None:
                    log.close()
                    del self.procs[label]
                    self._register(stage, end_ns=time.time_ns(), exit=code)
                    return label, code, time.time()
            time.sleep(0.2)


class SimExecutor:
    """Discrete-event executor: each stage takes est_s * slowdown seconds."""

    def __init__(self, stages, slowdown=1.0, outcomes=None):
        self.stages = stages
        self.slowdown = slowdown
        self.outcomes = outcomes or {}
        self.clock = 0.0
        self.events = []

    def now(self):
        return self.clock

    def start(self, stage, lane):
        factor = self.slowdown if stage.cls == "normal" else 1.0
        self.events.append((self.clock + stage.est * factor, stage.label))

    def wait_any(self):
        self.events.sort()
        end, label = self.events.pop(0)
        self.clock = max(self.clock, end)
        return label, self.outcomes.get(label, 0), self.clock


def run(stages, policy, executor, log=lambda line: None):
    done, started, running, lanes = set(), set(), {}, {}
    records = {}
    failed = []
    wall_exceeded = False
    origin = executor.now()
    while True:
        if not wall_exceeded and policy.deadline is not None and executor.now() > policy.deadline:
            wall_exceeded = True
            log("DEADLINE passed; " + ("no new stages" if policy.stop_at_deadline else
                                       "finishing the run for complete evidence; exit will be 3"))
        if not failed and not (wall_exceeded and policy.stop_at_deadline):
            starts = {l: origin + r["start"] for l, r in records.items()}
            for stage in select(stages, done, [stages[l] for l in running], started, policy,
                                executor.now(), starts):
                lane = min(set(range(1, len(running) + 2)) - set(lanes.values()))
                lanes[stage.label] = lane
                running[stage.label] = stage
                started.add(stage.label)
                records[stage.label] = {"label": stage.label, "lane": lane, "start": executor.now() - origin}
                log(f"START {stage.label} lane={lane} class={stage.cls} cores={stage.cores:g} mem_gib={stage.mem:g}")
                executor.start(stage, lane)
        if not running:
            break
        label, code, end = executor.wait_any()
        del running[label]
        lane = lanes.pop(label)
        records[label].update(end=end - origin, exit=code)
        log(f"EXIT {label} {code} lane={lane}")
        if code == 0:
            done.add(label)
        else:
            failed.append(label)
            log(f"FAIL-FAST {label} exit={code}; no new stages")
    wall = executor.now() - origin
    not_run = sorted(set(stages) - started)
    if failed:
        status = EXIT_STAGE_FAILED
    elif wall_exceeded or (policy.deadline is not None and executor.now() > policy.deadline):
        status = EXIT_WALL
    elif not_run:
        status = EXIT_STAGE_FAILED  # unreachable stages (should not happen in a valid DAG)
    else:
        status = EXIT_OK
    return {"status": status, "wall_s": wall, "failed": failed, "not_run": not_run,
            "stages": records, "critical_path": actual_critical_path(stages, records)}


def actual_critical_path(stages, records):
    """Walk back from the last-finishing stage through whichever predecessor
    (dependency, or the stage whose exit freed resources) ended last before it started."""
    finished = {l: r for l, r in records.items() if "end" in r}
    if not finished:
        return []
    current = max(finished, key=lambda l: finished[l]["end"])
    path = []
    while current:
        record = finished[current]
        deps = [d for d in stages[current].deps if d in finished]
        dep = max(deps, key=lambda d: finished[d]["end"], default=None)
        dep_end = finished[dep]["end"] if dep else 0.0
        reason = "dep" if dep else "start"
        previous = dep
        if record["start"] - dep_end > 1.0:
            blockers = [l for l, r in finished.items() if l != current and r["end"] <= record["start"] + 1e-6]
            if blockers:
                previous = max(blockers, key=lambda l: finished[l]["end"])
                reason = "resource"
        path.append({"label": current, "start": round(record["start"], 1), "end": round(record["end"], 1),
                     "duration": round(record["end"] - record["start"], 1), "via": reason})
        current = previous
    return list(reversed(path))


def format_report(result, title):
    lines = [f"{title}: wall {result['wall_s'] / 60:.1f} min ({result['wall_s']:.0f} s), status {result['status']}"]
    lines.append("critical path:")
    for hop in result["critical_path"]:
        lines.append(f"  {hop['start'] / 60:6.1f} -> {hop['end'] / 60:6.1f} min  {hop['duration']:7.0f} s  "
                     f"{hop['label']} [{hop['via']}]")
    return "\n".join(lines)


def mem_available_gib():
    with open("/proc/meminfo") as handle:
        for line in handle:
            if line.startswith("MemAvailable:"):
                return int(line.split()[1]) / (1 << 20)
    raise RuntimeError("MemAvailable missing")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="mode", required=True)
    for name in ("run", "simulate"):
        p = sub.add_parser(name)
        p.add_argument("--dag", required=True)
        p.add_argument("--cores", type=float, default=os.cpu_count())
        p.add_argument("--mem-gib", type=float, help="memory cap; default MemAvailable minus --mem-margin-gib")
        p.add_argument("--mem-margin-gib", type=float, default=2.0)
        p.add_argument("--allow-background", action="store_true")
        p.add_argument("--deadline-s", type=float, help="seconds from scheduler start (run) or t=0 (simulate)")
        p.add_argument("--json", help="write the result record here")
        p.add_argument("--stop-at-deadline", action="store_true",
                       help="start no stage after the deadline (default: finish, then exit 3)")
    sub.choices["run"].add_argument("--state-dir", required=True)
    sub.choices["run"].add_argument("--deadline-epoch", type=float, help="absolute wall-clock deadline")
    sub.choices["run"].add_argument("--log", help="append scheduler events here")
    sub.choices["run"].add_argument("runner", nargs=argparse.REMAINDER)
    sub.choices["simulate"].add_argument("--slowdown", type=float, default=1.0,
                                         help="duration factor for normal (shared) stages")
    sub.choices["simulate"].add_argument("--host-mem-gib", type=float, default=19.0,
                                         help="simulated cap = host minus --mem-margin-gib")
    args = parser.parse_args(argv)
    try:
        stages = load_dag(args.dag)
    except DagError as error:
        print(f"DAG error: {error}", file=sys.stderr)
        return EXIT_USAGE

    if args.mode == "simulate":
        mem = args.mem_gib if args.mem_gib is not None else args.host_mem_gib - args.mem_margin_gib
        policy = Policy(args.cores, mem, args.allow_background, args.deadline_s, args.stop_at_deadline)
        result = run(stages, policy, SimExecutor(stages, args.slowdown))
        bound, path = static_critical_path(stages)
        print(format_report(result, f"simulated ({args.cores:g} cores, {mem:g} GiB cap, slowdown {args.slowdown:g}, "
                                    f"background={'on' if args.allow_background else 'off'})"))
        exclusive = sum(s.est for s in stages.values() if s.cls != "normal")
        print(f"dependency-only lower bound: {bound / 60:.1f} min via {' > '.join(path)}")
        print(f"exclusive+solo serial sum: {exclusive / 60:.1f} min; all-stage serial sum: "
              f"{sum(s.est for s in stages.values()) / 60:.1f} min")
        if result["not_run"]:
            print(f"NOT RUN ({len(result['not_run'])}): {' '.join(result['not_run'])}")
        if args.json:
            with open(args.json, "w") as handle:
                json.dump(result, handle, indent=1)
        return result["status"]

    runner = [a for a in args.runner if a != "--"]
    if not runner:
        parser.error("runner command required after --")
    os.makedirs(args.state_dir, exist_ok=True)
    mem = args.mem_gib if args.mem_gib is not None else mem_available_gib() - args.mem_margin_gib
    deadline = args.deadline_epoch
    if deadline is None and args.deadline_s is not None:
        deadline = time.time() + args.deadline_s
    policy = Policy(args.cores, mem, args.allow_background, deadline, args.stop_at_deadline,
                    live_mem=mem_available_gib, margin=args.mem_margin_gib)
    log_path = args.log or os.path.join(args.state_dir, "scheduler.log")

    def log(line):
        stamped = f"{time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())} {line}"
        print(stamped, flush=True)
        with open(log_path, "a") as handle:
            handle.write(stamped + "\n")

    log(f"POLICY cores={policy.cores:g} mem_gib={policy.mem:.1f} background={policy.allow_background} "
        f"deadline={'-' if deadline is None else time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(deadline))} "
        f"stages={len(stages)}")
    executor = RealExecutor(runner, args.state_dir, dict(os.environ))

    def interrupted(signum, _frame):
        raise KeyboardInterrupt(signal.Signals(signum).name)

    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGHUP, interrupted)
    completed = False
    try:
        result = run(stages, policy, executor, log)
        completed = True
    except KeyboardInterrupt as error:
        log(f"INTERRUPTED {error}; terminating running stages {' '.join(sorted(executor.procs)) or '-'}")
        return 130
    finally:
        if not completed:
            executor.terminate()
    report = format_report(result, "scheduler")
    log(report.replace("\n", "\n    "))
    with open(args.json or os.path.join(args.state_dir, "schedule.json"), "w") as handle:
        json.dump(result, handle, indent=1)
    return result["status"]


if __name__ == "__main__":
    sys.exit(main())
