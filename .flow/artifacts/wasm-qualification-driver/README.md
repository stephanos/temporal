# WASM qualification drivers

| Path | What it is |
| --- | --- |
| `parallel/` | **Current validation driver** (2026-10-19): the q1 stage ladder as a dependency DAG under a resource-aware scheduler, with a one-hour wall-clock limit. |
| `q1-serial/` | The serial q1 reference driver, verbatim (`run-qualification.sh`, `stage-map.tsv`, `frozen.py`, `required-roots.py`, `helper-files.py`, `check-stage-set.py`, `run-matched-pair.sh`). Still runnable as before; the parallel driver checks its own stage set and commands against it. |
| `run-qualification.sh`, `run-qualification-fast.sh`, `quiet-window-fast.fn178.sh`, `stage-list.py` | fn-176/fn-178 drivers (2026-10-15), kept as history; see "fn-176/fn-178 drivers" below. |

## Parallel validation driver (`parallel/`)

Owner decision 2026-10-19: full validation of a candidate takes at most one
hour wall clock. `run-validation.sh` runs the same stages, commands, budgets,
deadlines, seeds, oracles and post-checks as `q1-serial/run-qualification.sh`;
only the order and overlap of stages change.

### Run it

```bash
cd .flow/artifacts/wasm-qualification-driver/parallel
bash run-validation.sh --plan                                   # equivalence check, DAG, estimate; runs nothing
bash run-validation.sh HEAD LINT_BASE FRESH_EVIDENCE FRESH_NATIVE_SCRATCH   # same arguments as the serial driver
bash run-perf.sh VALIDATION_ROOT FRESH_PERF_ROOT                # matched-release pair, alone, after a PASS
python3 -m unittest discover -s tests -v                        # scheduler/equivalence tests (stub runners)
```

Options: `--config FILE` binds the candidate (default `config/q1.env`: frozen
worktree, freeze root, expected HEAD and manifest digest, staged fixture, perf
baseline). `--allow-background` (default off) lets `bg` stages overlap exclusive
ones (see "Reaching one hour"). `--deadline-s` (default 3600),
`--mem-margin-gib` (default 2).

Exit codes: `0` pass; `1` a stage or post-check failed (fail-fast); `2` setup
refused (e.g. virtiofs scratch); `3` the wall-clock limit was exceeded; `4` the
stage-set/command equivalence check failed; `130` interrupted (running stage
process groups are terminated).

### Files

- `dag.tsv`: one row per stage: dependencies, class, declared cores and peak
  memory, observed duration (`est_s`), named locks, in-tree scratch globs,
  flags and tier.
- `scheduler.py`: the scheduler (`run`) and a discrete-event estimator
  (`simulate`) sharing the same admission code.
- `lib.sh`: the serial driver's helpers and one `step_<label>` per stage
  (the serial lines for that stage, its cwd and its `GOMAD_*` environment).
- `stage.sh`: runs one step as its own process; `run-validation.sh`: preamble,
  scheduler, final checks; `run-perf.sh`: performance entry point.
- `stage_set.py`: equivalence before and after the run; `frozen.py`: the q1
  frozen-source guard with its constants as arguments.
- `tests/`: unit tests; `tests/envtrace.py` traces cwd and `GOMAD_*` exports at
  every stage call in both drivers.

### Scheduling policy

- A stage starts when all dependencies exited 0. Dependencies are explicit:
  cheap static checks gate `wasmmeasure-build` and the helper builds; every
  helper-consuming stage depends on `ordinary-helper-publication` (release and
  conformance helpers admitted, installed and published); data dependencies
  follow the serial driver (runtime seeds before the aggregate, `clusters`
  before the workload matrix, the workload cells and SQLite selections as an
  ordered chain, `terminal-module` before the host admission and terminal exit).
- Resources: the sum of declared cores stays within `nproc` and the sum of
  declared peak memory within MemAvailable at start minus the margin; a stage
  larger than a cap runs only on an otherwise idle host. Seeds: Cranelift-heavy
  preparation 5 GiB, workload cells 9 GiB (about 4 GiB guest plus 5 GiB helper),
  measured tree RSS for the light stages (q1 run, `stage-mem-sampler.py`).
- `memguard` (runtime seeds): the test refuses to dispatch below 8 GiB
  MemAvailable, so other stages leave it 8 GiB of the cap.
- Classes: `exclusive` stages (busy12 stress, terminal host admission and
  terminal exit, every workload cell and SQLite selection — the stages with
  wall-clock budgets or timing evidence) run with no other stage; `solo`
  stages (helper corpus regeneration, which rewrites tracked files) run with
  nothing at all, in every mode.
- Locks: `golangci` (golangci-lint refuses parallel runs), `cargo-check-target`
  (the Rust matrix shares one Cargo target), `fixture-temporal-broker`
  (`ordinary` and `clusters` both create the staged broker fixture files).
- Priority: tier 0 (cheap static checks) first, then the longest remaining
  path. When the top-priority ready stage is exclusive it reserves the host:
  other stages are admitted only if their estimate ends before the running
  stages do (no starvation, no delay of the exclusive stage).
- Fail-fast: after the first non-zero exit no stage starts; running stages
  finish and are recorded. Once the deadline passes no stage starts and the
  driver exits 3.

### Concurrency hazards handled (none weakens a check)

- In-tree scratch: `temporal-inspection` creates
  `service/history/.gomad-inspection-*` and `ordinary` (backend inspection
  tests) `common/gomaddebug/testdata/inspection-*` for the duration of their
  tests; both are untracked and not ignored. The scheduler registers each
  running stage's declared scratch globs (`sched/registry/`) before starting it;
  `frozen.py` exempts only `??` entries under those globs, only for other stages
  whose run interval overlaps the check. Tracked changes, every manifest digest
  and any other untracked path still fail; a stage never exempts its own scratch;
  the preamble, final and exclusive/solo checks are strict. `cleantree` stages
  (terminal host admission and exit; the measurement producers check
  `git status` themselves) never overlap a scratch stage.
- Fixture staging: `.tmp/wasm-backend-temporal` (ignored) is created once by
  the driver because the tests create missing parents with a non-idempotent
  `Mkdir`; the shared broker fixture is serialized by the lock above.
- Files other stages read while they run (`helper.sha256`, `canonical-helper.sha256`,
  `conformance-helper.sha256`, `actionlint.sha256`, `wasmmeasure.sha256`, run
  state) are written atomically. This is the one textual change inside a stage
  command (`publish_ordinary_helper`); `stage_set.py` allows exactly it.
- `conformance-helper-build` builds next to `helper-build` (its Cargo command
  does not read the release helper; `wasmhost` has no build script); its
  installation and the distinct-file check run before `ordinary-helper-publication`,
  as in the serial order.

### Equivalence and evidence

- Before any stage: `stage_set.py static` requires the DAG labels to equal the
  serial reference's labels minus `native-*`, each label to resolve to exactly
  one stage call in `lib.sh` with byte-identical command text, the shared
  helper bodies (`named_test`, `rust_test`, `check_pair`, `clean_measure`,
  `busy12`, …) to be identical, and the workload entry list to match
  (`stage-set.json`). `tests/envtrace.py` checks cwd and every `GOMAD_*` export
  at each stage call against the serial order.
- After the run: `stage_set.py verify` requires exactly one `START` and one
  `EXIT <label> 0` per required stage, nothing else, and a zero-exit receipt
  per stage (`stage-set-executed.json`).
- Kept per stage: `<label>.command`, `<label>.log`, `<label>.receipt.json`
  (now with `lane`), frozen-source checks before and after every stage, the
  serial post-checks. `stages.log` keeps the serial line format plus
  `lane=N`; `STEP-FAIL` lines name a failing post-check.
- Run records: `sched/scheduler.log` (admissions, exits, critical path),
  `sched/schedule.json` (per-stage start/end/lane/exit, critical path),
  `wall-time.json` (total wall seconds, limit, critical path), `driver/` and
  `driver.sha256` (the exact driver used).

### Dropped and moved stages

- Dropped: `native-profile-inventory`, `native-bootstrap-pin`,
  `native-semantic-pin`, `native-minimizer-preservation`, `native-source`. The
  native backend was retired on 2026-10-19 and its specs closed won't-do, so
  these checks no longer qualify anything. 96 of the reference's 101 stage
  labels remain.
- Moved: `matched-release-candidate` (in the serial `stage-map.tsv` as
  `SEPARATE_OPEN`) measures performance and is not a validation gate. It is
  `run-perf.sh`, a path-parameterized copy of `q1-serial/run-matched-pair.sh`
  (same producers, admission boundaries, seeds and repetitions). Run it alone
  in fn-185's timing window after a passing validation run.

### Wall-time estimate and reaching one hour

ESTIMATE_PLACEHOLDER

## fn-176/fn-178 drivers

Snapshot of the drivers used for fn-176/fn-178 frozen qualification (2026-10-15).

- `run-qualification.sh`: the serial reference driver (fn-176 84 stages plus the fn-178 cluster stages; 89 total).
- `run-qualification-fast.sh`: owner-approved fast variant (2026-10-15). Same stage set, commands and post-checks, reordered into a fail-fast static phase with a tool preflight, the helper-build gate, three parallel functional lanes (heavy Temporal guests share one token), a serial selection/catalogue phase and an exclusive phase for the busy12 stress and timing-sensitive stages. At startup it asserts its stage set equals the reference driver's (`stage-list.py`) and aborts on any difference. `--plan` prints lanes and reuse keys without running.
- `--reuse <prior QUAL_ROOT>` (repeatable, off by default) copies a prior stage pass only when its key matches exactly: stage label, command, helper and compiler identities, and the stage's declared input digest. Reused stages are logged as `REUSED <label> <root>` and never count as fresh passes.
- `quiet-window-fast.fn178.sh`: the fn-178 measurement window (fast driver, concurrent measurement prebuilds, then baseline/candidate/baton timing alone). Its workspace paths are specific to the fn-178 run.

Scratch must live on native disk, not virtiofs (the driver refuses virtiofs/fuse scratch). Default paths name the sandbox layout; pass explicit arguments elsewhere.

### fn-187 producer move (2026-10-16)

These drivers target the frozen fn-178 checkouts named in each script, which
still contain the `go test` producers they call, so the scripts are unchanged.
On a checkout with `tools/gomad/internal/backends/wasm/cmd/wasmmeasure`,
`TestBuildTerminalExitReleaseHelper` is `wasmmeasure terminal-exit-helper-build
--terminal-exit-helper-build-root ...`, `TestTemporalTerminalToProcessExit` is
`wasmmeasure terminal-exit --terminal-exit-root ...` (with its helper/module
receipt flags), and `TestMatchedReleaseClusterCandidate` is `wasmmeasure
matched-release-candidate`. The command refuses inherited `GOMAD_*` variables,
so a driver must unset them and pass flags; see
`tools/gomad/internal/backends/wasm/QUALIFICATION.md#measurement-producers`.
