#!/usr/bin/env python3
"""Scheduler, stage-set and frozen-check tests with stub runners (no gate is executed).

Run: python3 -m unittest discover -s tests -v   (from the parallel/ directory)
"""
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import textwrap
import time
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(HERE))
import scheduler as s  # noqa: E402

HEADER = "\t".join(s.COLUMNS)


def row(label, deps="-", cls="normal", cores=1, mem=1, est=1, locks="-", scratch="-", flags="-", tier=2):
    return "\t".join([label, deps, cls, str(cores), str(mem), str(est), locks, scratch, flags, str(tier)])


def dag(tmp, *rows):
    path = Path(tmp) / "dag.tsv"
    path.write_text("\n".join([HEADER, *rows]) + "\n")
    return str(path)


def overlaps(records, a, b):
    ra, rb = records[a], records[b]
    return ra["start"] < rb["end"] - 1e-9 and rb["start"] < ra["end"] - 1e-9


def simulate(path, cores=12, mem=15, background=False, deadline=None, outcomes=None, stop=False, live_mem=None):
    stages = s.load_dag(path)
    policy = s.Policy(cores, mem, background, deadline, stop, live_mem=live_mem)
    return s.run(stages, policy, s.SimExecutor(stages, outcomes=outcomes)), stages


STUB = textwrap.dedent("""\
    #!/usr/bin/env bash
    # stub runner: records start/end, sleeps STUB_SLEEP_<label>, fails labels in STUB_FAIL
    label="$1"
    printf '%s START %s lane=%s\\n' "$(date +%s.%N)" "$label" "$QUALDRV_LANE" >> "$STUB_LOG"
    var="STUB_SLEEP_${label//-/_}"
    sleep "${!var:-0.3}"
    printf '%s EXIT %s lane=%s\\n' "$(date +%s.%N)" "$label" "$QUALDRV_LANE" >> "$STUB_LOG"
    case " $STUB_FAIL " in *" $label "*) exit 7;; esac
    exit 0
    """)


class SchedulerSimTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def test_dependency_order(self):
        path = dag(self.tmp, row("a", est=5), row("b", "a", est=3), row("c", "a", est=1), row("d", "b,c", est=2))
        result, stages = simulate(path)
        self.assertEqual(result["status"], s.EXIT_OK)
        records = result["stages"]
        for label, stage in stages.items():
            for dep in stage.deps:
                self.assertLessEqual(records[dep]["end"], records[label]["start"], f"{dep} -> {label}")
        self.assertTrue(overlaps(records, "b", "c"))
        self.assertEqual(result["wall_s"], 10)
        self.assertEqual([hop["label"] for hop in result["critical_path"]], ["a", "b", "d"])

    def test_cycle_and_unknown_dependency_rejected(self):
        with self.assertRaises(s.DagError):
            s.load_dag(dag(self.tmp, row("a", "b"), row("b", "a")))
        with self.assertRaises(s.DagError):
            s.load_dag(dag(self.tmp, row("a", "missing")))
        with self.assertRaises(s.DagError):
            s.load_dag(dag(self.tmp, row("a"), row("a")))

    def test_exclusive_runs_alone(self):
        path = dag(self.tmp, row("x", cls="exclusive", est=4), *[row(f"n{i}", est=3) for i in range(5)],
                   row("bg", est=20, flags="bg"))
        result, _ = simulate(path)
        records = result["stages"]
        for other in records:
            if other != "x":
                self.assertFalse(overlaps(records, "x", other), other)

    def test_background_overlaps_only_bg_and_not_cleantree(self):
        path = dag(self.tmp, row("start", est=1), row("bg", "start", est=50, flags="bg", scratch="tmp/*"),
                   row("x", "start", cls="exclusive", est=4), row("y", "x", cls="exclusive", est=4, flags="cleantree"),
                   row("n", "start", est=3))
        result, _ = simulate(path, background=True)
        records = result["stages"]
        self.assertTrue(overlaps(records, "x", "bg"))
        self.assertFalse(overlaps(records, "x", "n"))
        self.assertFalse(overlaps(records, "y", "bg"), "cleantree exclusive must not overlap in-tree scratch")
        strict, _ = simulate(path, background=False)
        self.assertFalse(overlaps(strict["stages"], "x", "bg"))

    def test_solo_never_overlaps_even_with_background(self):
        path = dag(self.tmp, row("bg", est=10, flags="bg"), row("solo", cls="solo", est=2), row("n", est=3))
        result, _ = simulate(path, background=True)
        for other in ("bg", "n"):
            self.assertFalse(overlaps(result["stages"], "solo", other))

    def test_memory_and_core_caps(self):
        path = dag(self.tmp, *[row(f"m{i}", mem=6, cores=2, est=5) for i in range(6)])
        result, stages = simulate(path, mem=15)
        records = result["stages"]
        times = sorted({r["start"] for r in records.values()})
        for t in times:
            active = [l for l, r in records.items() if r["start"] <= t < r["end"]]
            self.assertLessEqual(sum(stages[l].mem for l in active), 15)
        self.assertEqual(result["wall_s"], 15)  # two at a time
        cores, _ = simulate(dag(self.tmp, *[row(f"c{i}", cores=5, est=5) for i in range(3)]), cores=12)
        self.assertEqual(cores["wall_s"], 10)

    def test_memguard_keeps_eight_gib_free(self):
        path = dag(self.tmp, row("g1", mem=2, est=5, flags="memguard"), row("g2", mem=2, est=5, flags="memguard"),
                   row("big", mem=6, est=5), row("big2", mem=6, est=5))
        result, stages = simulate(path, mem=15)
        records = result["stages"]
        for t in sorted({r["start"] for r in records.values()}):
            active = [l for l, r in records.items() if r["start"] <= t < r["end"]]
            for guarded in [l for l in active if "memguard" in stages[l].flags]:
                others = sum(stages[l].mem for l in active if l != guarded)
                self.assertLessEqual(others + s.GUARD_GIB, 15, (t, active))
        self.assertTrue(overlaps(records, "g1", "g2"))

    def test_oversized_stage_runs_alone_instead_of_deadlocking(self):
        result, _ = simulate(dag(self.tmp, row("huge", mem=40, est=2), row("small", est=1)), mem=15)
        self.assertEqual(result["status"], s.EXIT_OK)
        self.assertFalse(overlaps(result["stages"], "huge", "small"))

    def test_locks_are_mutually_exclusive(self):
        result, _ = simulate(dag(self.tmp, row("a", locks="l", est=3), row("b", locks="l,m", est=3), row("c", locks="m", est=3)))
        records = result["stages"]
        self.assertFalse(overlaps(records, "a", "b"))
        self.assertFalse(overlaps(records, "b", "c"))
        self.assertTrue(overlaps(records, "a", "c"))

    def test_cleantree_never_overlaps_scratch(self):
        result, _ = simulate(dag(self.tmp, row("scratchy", scratch="x/*", est=5), row("clean", flags="cleantree", est=1)))
        self.assertFalse(overlaps(result["stages"], "scratchy", "clean"))

    def test_fail_fast_stops_new_stages_and_lets_running_finish(self):
        path = dag(self.tmp, row("a", est=1), row("long", est=10), row("bad", "a", est=1), row("late", "bad", est=1),
                   row("after", "a", est=5, mem=20))
        result, _ = simulate(path, mem=15, outcomes={"bad": 9})
        records = result["stages"]
        self.assertEqual(result["status"], s.EXIT_STAGE_FAILED)
        self.assertEqual(result["failed"], ["bad"])
        self.assertIn("late", result["not_run"])
        self.assertIn("after", result["not_run"])
        self.assertEqual(records["long"]["end"], 10, "running stage finishes and is recorded")
        fail_time = records["bad"]["end"]
        self.assertTrue(all(r["start"] <= fail_time for r in records.values()))

    def test_static_tier_first(self):
        result, _ = simulate(dag(self.tmp, row("heavy", est=50, mem=10), row("static", est=1, mem=10, tier=0)), mem=15)
        self.assertEqual(result["stages"]["static"]["start"], 0)
        self.assertGreater(result["stages"]["heavy"]["start"], 0)

    def test_exclusive_reservation_backfills_only_short_work(self):
        # bg-less exclusive x (top priority) waits for `running`; short n1 may backfill, long n2 may not.
        path = dag(self.tmp, row("running", est=10), row("gate", est=1), row("x", "gate", cls="exclusive", est=30),
                   row("n1", "gate", est=5), row("n2", "gate", est=20))
        result, _ = simulate(path)
        records = result["stages"]
        self.assertEqual(records["x"]["start"], 10)
        self.assertLess(records["n1"]["start"], 10)
        self.assertGreaterEqual(records["n2"]["start"], records["x"]["end"])

    def test_deadline_exit_code(self):
        path = dag(self.tmp, row("a", est=5), row("b", "a", est=5), row("c", "b", est=5))
        result, _ = simulate(path, deadline=7)
        self.assertEqual(result["status"], s.EXIT_WALL, "default: finish for complete evidence, then exit 3")
        self.assertEqual(result["not_run"], [])
        stopped, _ = simulate(path, deadline=7, stop=True)
        self.assertEqual(stopped["status"], s.EXIT_WALL)
        self.assertIn("c", stopped["not_run"])
        self.assertNotIn("c", stopped["stages"])
        within, _ = simulate(path, deadline=20)
        self.assertEqual(within["status"], s.EXIT_OK)

    def test_memguard_requires_live_headroom(self):
        path = dag(self.tmp, row("busy", mem=1, est=5), row("g", mem=1, est=1, flags="memguard"))
        result, _ = simulate(path, live_mem=lambda: 5.0)
        self.assertGreaterEqual(result["stages"]["g"]["start"], result["stages"]["busy"]["end"],
                                "guarded stage waits while live MemAvailable is below 8 GiB + own")

    def test_committed_dag_is_valid_and_estimated(self):
        stages = s.load_dag(str(HERE / "dag.tsv"))
        result = s.run(stages, s.Policy(12, 15), s.SimExecutor(stages))
        self.assertEqual(result["status"], s.EXIT_OK)
        self.assertEqual(len(result["stages"]), len(stages))


class SchedulerRealTest(unittest.TestCase):
    """RealExecutor with a stub bash runner: real processes, lanes, registry, exit codes."""

    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.stub = Path(self.tmp) / "stub.sh"
        self.stub.write_text(STUB)
        self.log = Path(self.tmp) / "stub.log"

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def run_cli(self, path, *extra, fail="", sleeps=None):
        env = dict(os.environ, STUB_LOG=str(self.log), STUB_FAIL=fail)
        for label, seconds in (sleeps or {}).items():
            env["STUB_SLEEP_" + label.replace("-", "_")] = str(seconds)
        state = Path(self.tmp) / "state"
        proc = subprocess.run([sys.executable, str(HERE / "scheduler.py"), "run", "--dag", path, "--state-dir", str(state),
                               "--cores", "12", "--mem-gib", "15", *extra, "--", "bash", str(self.stub)],
                              env=env, capture_output=True, text=True, timeout=120)
        events = {}
        if self.log.exists():
            for line in self.log.read_text().splitlines():
                stamp, kind, label, lane = line.split()
                events.setdefault(label, {})[kind] = float(stamp)
                events[label]["lane"] = lane
        return proc, events, state

    def test_real_order_exclusivity_lanes_and_registry(self):
        path = dag(self.tmp, row("a"), row("b", "a", scratch="tmp/x-*"), row("c", "a"), row("x", "b,c", cls="exclusive"),
                   row("d", "x"))
        proc, events, state = self.run_cli(path)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertLessEqual(events["a"]["EXIT"], events["b"]["START"])
        self.assertLessEqual(events["x"]["EXIT"], events["d"]["START"])
        self.assertLess(events["b"]["START"], events["c"]["EXIT"], "independent stages run concurrently")
        self.assertNotEqual(events["b"]["lane"], events["c"]["lane"])
        for other in ("a", "b", "c", "d"):
            self.assertTrue(events[other]["EXIT"] <= events["x"]["START"] or events[other]["START"] >= events["x"]["EXIT"])
        record = json.loads((state / "registry" / "b.json").read_text())
        self.assertEqual(record["scratch"], ["tmp/x-*"])
        self.assertLess(record["start_ns"], record["end_ns"])
        schedule = json.loads((state / "schedule.json").read_text())
        self.assertEqual(schedule["status"], 0)
        self.assertEqual(schedule["critical_path"][-1]["label"], "d")

    def test_real_fail_fast_exit_code(self):
        path = dag(self.tmp, row("a"), row("slow"), row("bad", "a"), row("never", "bad"))
        proc, events, state = self.run_cli(path, fail="bad", sleeps={"slow": 1.5, "bad": 0.1})
        self.assertEqual(proc.returncode, s.EXIT_STAGE_FAILED)
        self.assertNotIn("never", events)
        self.assertIn("EXIT", events["slow"], "running stage allowed to finish")
        self.assertIn("FAIL-FAST bad", (state / "scheduler.log").read_text())

    def test_real_deadline_exit_code(self):
        path = dag(self.tmp, row("a"), row("b", "a"), row("c", "b"))
        proc, events, _ = self.run_cli(path, "--deadline-s", "0.5", "--stop-at-deadline", sleeps={"a": 1.0})
        self.assertEqual(proc.returncode, s.EXIT_WALL)
        self.assertNotIn("b", events)
        self.log.unlink()
        shutil.rmtree(Path(self.tmp) / "state")
        proc, events, _ = self.run_cli(path, "--deadline-s", "0.5", sleeps={"a": 1.0})
        self.assertEqual(proc.returncode, s.EXIT_WALL)
        self.assertIn("EXIT", events["c"], "runs to completion, then exit 3")


class StageSetTest(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.reference = HERE.parent / "q1-serial" / "run-qualification.sh"

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def static(self, lib=None, dag_path=None):
        return subprocess.run([sys.executable, str(HERE / "stage_set.py"), "static", "--reference", str(self.reference),
                               "--lib", str(lib or HERE / "lib.sh"), "--dag", str(dag_path or HERE / "dag.tsv")],
                              capture_output=True, text=True)

    def mutated(self, name, old, new):
        path = self.tmp / name
        text = (HERE / name).read_text()
        self.assertIn(old, text)
        path.write_text(text.replace(old, new, 1))
        return path

    def test_committed_driver_is_equivalent(self):
        proc = self.static()
        self.assertEqual(proc.returncode, 0, proc.stdout)
        report = json.loads(proc.stdout)
        self.assertEqual(report["required"], 96)
        self.assertEqual(report["dropped"], ["native-bootstrap-pin", "native-minimizer-preservation",
                                             "native-profile-inventory", "native-semantic-pin", "native-source"])

    def test_changed_command_detected(self):
        lib = self.mutated("lib.sh", "TestTemporalCommittedWorkflowFreshGuestCrashReopen 90m", "TestTemporalCommittedWorkflowFreshGuestCrashReopen 120m")
        self.assertEqual(self.static(lib=lib).returncode, 4)
        lib = self.mutated("lib.sh", "-count=10 -timeout=30m", "-count=1 -timeout=30m")
        self.assertEqual(self.static(lib=lib).returncode, 4)

    def test_dynamic_label_and_environment_changes_detected(self):
        lib = self.mutated("lib.sh", "  stage terminal-controls ", '  local label=terminal-controls\n  stage "$label" ')
        self.assertEqual(self.static(lib=lib).returncode, 4)
        lib = self.mutated("lib.sh", "export GOMAD_WASM_CONFORMANCE_REQUIRED=1", "export GOMAD_WASM_CONFORMANCE_REQUIRED=0")
        self.assertEqual(self.static(lib=lib).returncode, 4)

    def test_changed_helper_body_detected(self):
        lib = self.mutated("lib.sh", ".Budget.WithinBudget == true and", "")
        self.assertEqual(self.static(lib=lib).returncode, 4)

    def test_missing_or_extra_dag_label_detected(self):
        lines = (HERE / "dag.tsv").read_text().splitlines()
        missing = self.tmp / "missing.tsv"
        missing.write_text("\n".join(line for line in lines if not line.startswith("busy-grouped\t")) + "\n")
        self.assertEqual(self.static(dag_path=missing).returncode, 4)
        extra = self.tmp / "extra.tsv"
        extra.write_text("\n".join(lines + [row("native-source")]) + "\n")
        self.assertEqual(self.static(dag_path=extra).returncode, 4)

    def test_verify_requires_exactly_one_pass_per_stage(self):
        dag_path = dag(self.tmp, row("a"), row("b"))
        for label in ("a", "b"):
            (self.tmp / f"{label}.receipt.json").write_text(json.dumps({"exit": 0, "label": label}))
            (self.tmp / f"{label}.command").write_text("cwd=/\n")
        log = self.tmp / "stages.log"

        def verify(lines):
            log.write_text("\n".join(lines) + "\n")
            return subprocess.run([sys.executable, str(HERE / "stage_set.py"), "verify", "--dag", dag_path,
                                   "--stages-log", str(log), "--root", str(self.tmp)], capture_output=True, text=True).returncode

        ok = ["t START a lane=1", "t EXIT a 0 NEW_RUN lane=1", "t START b lane=2", "t EXIT b 0 NEW_RUN lane=2"]
        self.assertEqual(verify(ok), 0)
        self.assertEqual(verify(ok + ["t START a lane=1", "t EXIT a 0 NEW_RUN lane=1"]), 4)
        self.assertEqual(verify(ok[:2]), 4)
        self.assertEqual(verify(ok + ["t START native-source lane=1"]), 4)
        self.assertEqual(verify(ok[:3] + ["t EXIT b 1 NEW_RUN lane=2"]), 4)
        self.assertEqual(verify(ok + ["t STEP-FAIL b 1 lane=2"]), 4)
        (self.tmp / "b.command").unlink()
        self.assertEqual(verify(ok), 4)


class EnvironmentTraceTest(unittest.TestCase):
    """Each stage call sees the serial driver's cwd and GOMAD_* exports (envtrace.py)."""

    def setUp(self):
        import envtrace
        self.envtrace = envtrace
        self.reference = HERE.parent / "q1-serial" / "run-qualification.sh"
        self.tmp = Path(tempfile.mkdtemp())

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def test_committed_lib_matches_serial_environment(self):
        serial, parallel, problems = self.envtrace.compare(self.reference, HERE / "lib.sh")
        self.assertEqual(problems, [])
        self.assertEqual(len([label for label in serial if not label.startswith("native-")]), 53)  # stage call sites (loops count once)

    def test_mutations_detected(self):
        text = (HERE / "lib.sh").read_text()
        for old, new in [('export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/storage"\n  export GOMAD_WASM_NETWORK', 'export GOMAD_WASM_NETWORK'),
                         ('step_terminal_controls() {\n  cd "$wasm"', 'step_terminal_controls() {\n  cd "$module"'),
                         ("export GOMAD_WASM_CONFORMANCE_REQUIRED=1", "export GOMAD_WASM_CONFORMANCE_REQUIRED=0")]:
            self.assertIn(old, text)
            mutated = self.tmp / "lib.sh"
            mutated.write_text(text.replace(old, new, 1))
            self.assertNotEqual(self.envtrace.compare(self.reference, mutated)[2], [], new)


LIB_HARNESS = textwrap.dedent("""\
    set -euo pipefail
    set -o noclobber
    shopt -s nullglob dotglob
    source "$LIB"
    stage() { printf '%s\\n' "${STUB_OUT:-}" > "$QUAL_ROOT/$1.log"; printf '%s\\n' "$1" >> "$QUAL_ROOT/stubbed-stages"; }
    check_source() { :; }
    QUAL_REPO="$ROOT/repo" QUAL_ROOT="$ROOT/qual" wasm="$ROOT/repo" driver_dir="$DRIVER" qualification_go=go
    installed_helper="$QUAL_ROOT/helpers/ordinary/gomad-wasmhost"
    mkdir -p "$QUAL_ROOT/state" "$QUAL_ROOT/bin" "$QUAL_ROOT/helper-build/x"
    printf 'ACTIONLINT_VER := v1.7.7\\n' > "$QUAL_REPO/Makefile"
    printf 'tool' > "$QUAL_ROOT/bin/actionlint"; printf 'tool' > "$QUAL_ROOT/bin/wasmmeasure"
    printf 'helper-bytes' > "$ROOT/admitted"
    jq -n --arg h "$ROOT/admitted" --arg d "$(sha256sum "$ROOT/admitted" | cut -d' ' -f1)" '{Helper:$h,HelperSHA256:$d}' \\
      > "$QUAL_ROOT/helper-build/x/helper-admission.json"
    STUB_OUT=v1.7.7 step_actionlint_version
    step_wasmmeasure_build
    step_helper_build
    load_helper_state
    test "$helper_ready" = true && test "$expected_helper" = "$(sha256sum "$ROOT/admitted" | cut -d' ' -f1)"
    sha256sum -c "$QUAL_ROOT/helper.sha256" "$QUAL_ROOT/actionlint.sha256" "$QUAL_ROOT/wasmmeasure.sha256" > /dev/null
    if atomic_output "$QUAL_ROOT/helper.sha256" printf x; then exit 9; fi
    printf 'harness-ok\\n'
    """)


class LibHarnessTest(unittest.TestCase):
    """lib.sh steps under the stage process's shell options (errexit, pipefail, noclobber)."""

    def test_state_writing_steps_run_under_noclobber(self):
        root = Path(tempfile.mkdtemp())
        try:
            (root / "repo").mkdir()
            proc = subprocess.run(["bash", "-c", LIB_HARNESS], capture_output=True, text=True, timeout=60,
                                  env=dict(os.environ, LIB=str(HERE / "lib.sh"), ROOT=str(root), DRIVER=str(HERE)))
            self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
            self.assertIn("harness-ok", proc.stdout)
        finally:
            shutil.rmtree(root)


class FrozenScratchTest(unittest.TestCase):
    """frozen.py: strict by default; only overlapping registered scratch globs exempt untracked paths."""

    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.repo = self.tmp / "repo"
        env = dict(os.environ, GIT_DISCOVERY_ACROSS_FILESYSTEM="1", GIT_AUTHOR_NAME="t", GIT_AUTHOR_EMAIL="t@t",
                   GIT_COMMITTER_NAME="t", GIT_COMMITTER_EMAIL="t@t")
        self.env = env
        (self.repo / "service/history").mkdir(parents=True)
        (self.repo / "stage").mkdir()
        (self.repo / "service/history/a.go").write_text("package history\n")
        (self.repo / "stage/fixture.go").write_text("fixture\n")
        subprocess.run(["git", "init", "-q", str(self.repo)], check=True, env=env)
        subprocess.run(["git", "-C", str(self.repo), "add", "."], check=True, env=env)
        subprocess.run(["git", "-C", str(self.repo), "commit", "-qm", "x"], check=True, env=env)
        self.head = subprocess.check_output(["git", "-C", str(self.repo), "rev-parse", "HEAD"], text=True, env=env).strip()
        self.freeze = self.tmp / "freeze"
        self.freeze.mkdir()
        rows = []
        for name in ("service/history/a.go", "stage/fixture.go"):
            path = self.repo / name
            rows.append(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path}")
        (self.freeze / "source.sha256").write_text("\n".join(rows) + "\n")
        self.manifest = hashlib.sha256((self.freeze / "source.sha256").read_bytes()).hexdigest()
        (self.freeze / "source-symlinks.json").write_text("{}")
        (self.freeze / "preflight.json").write_text(json.dumps({
            "head": self.head, "source_manifest_sha256": self.manifest, "source_files": 2,
            "staged_temporal_source": "stage", "staged_sha256": {"fixture.go": hashlib.sha256(b"fixture\n").hexdigest()}}))
        self.registry = self.tmp / "registry"
        self.registry.mkdir()

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def frozen(self, *extra):
        return subprocess.run([sys.executable, str(HERE / "frozen.py"), str(self.repo), str(self.freeze), self.head,
                               self.manifest, *extra], capture_output=True, text=True, env=self.env).returncode

    def register(self, label, scratch, end=None):
        record = {"label": label, "scratch": scratch, "start_ns": time.time_ns() - 10**9}
        if end is not None:
            record["end_ns"] = end
        (self.registry / f"{label}.json").write_text(json.dumps(record))

    def test_clean_and_strict(self):
        self.assertEqual(self.frozen(), 0)
        scratch = self.repo / "service/history/.gomad-inspection-onebox-1"
        scratch.mkdir()
        (scratch / "x.go").write_text("x")
        self.assertNotEqual(self.frozen(), 0, "strict check rejects untracked scratch")
        self.register("temporal-inspection", ["service/history/.gomad-inspection-*"])
        self.assertEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "storage"), 0)
        self.assertNotEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "temporal-inspection"), 0,
                            "a stage never exempts its own scratch")
        self.assertNotEqual(self.frozen(), 0, "final/strict check stays strict")

    def test_ended_or_unmatched_scratch_not_exempt(self):
        scratch = self.repo / "service/history/.gomad-inspection-shard-2"
        scratch.mkdir()
        (scratch / "y.go").write_text("y")
        self.register("temporal-inspection", ["service/history/.gomad-inspection-*"], end=time.time_ns() - 10**8)
        self.assertNotEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "x"), 0)
        self.register("temporal-inspection", ["common/gomaddebug/testdata/inspection-*"])
        self.assertNotEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "x"), 0)

    def test_tracked_change_never_exempt(self):
        self.register("temporal-inspection", ["service/history/*"])
        (self.repo / "service/history/a.go").write_text("package changed\n")
        self.assertNotEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "x"), 0)
        (self.repo / "service/history/a.go").write_text("package history\n")
        self.assertEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "x"), 0)
        (self.repo / "stray.txt").write_text("stray")
        self.assertNotEqual(self.frozen("--scratch-registry", str(self.registry), "--self", "x"), 0)


if __name__ == "__main__":
    unittest.main()
