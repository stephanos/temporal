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
bash run-validation.sh --config CANDIDATE_ENV HEAD LINT_BASE FRESH_EVIDENCE FRESH_NATIVE_SCRATCH
bash run-perf.sh --config CANDIDATE_ENV VALIDATION_ROOT FRESH_PERF_ROOT # alone, after a PASS
python3 -m unittest discover -s tests -v                        # scheduler/equivalence tests (stub runners)
```

Options: `--config FILE` is required except for `run-validation.sh --plan`. It
binds the frozen worktree, freeze root, expected HEAD and manifest digest,
staged fixture and optional performance baseline. Candidate configurations belong
to their open owner. `--allow-background` (default off) lets `bg` stages overlap exclusive
ones (see "Reaching one hour"). `--deadline-s` (default 3600),
`--mem-margin-gib` (default 2).

Exit codes: `0` pass; `1` a stage or post-check failed (fail-fast); `2` setup
refused (virtiofs/fuse scratch, less than 12 GiB free); `3` the wall-clock
limit was exceeded (the run still completes every stage and the final checks so
the evidence is whole; `--stop-at-deadline` stops admitting stages
instead); `4` the stage-set/command/environment equivalence check failed;
`130` interrupted (SIGTERM/INT/HUP are forwarded and every running stage's
process group is terminated). A stage failure takes precedence over the limit.
`WALL` and `wall-time.json` are written on every exit, `PASS` only when the
run passed within the limit.

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
  the cheap static checks (validate, portable, default build/vet, compile-only,
  lint-router(-vet), workflow-lint, lint-module) gate `wasmmeasure-build` and the
  helper builds; the slow static checks (`architecture`, `lint-root`) run
  alongside the helper builds and stop the run through fail-fast; every
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
- `memguard` (runtime seeds): the test fails if MemAvailable is below 8 GiB
  before any of its dispatches, so other stages leave it 8 GiB of the cap and the
  scheduler also requires live MemAvailable >= 8 GiB + margin + its own peak at
  admission.
- Classes:
  - `exclusive` stages (`busy-boundaries`, `busy-grouped`,
    `terminal-host-admission`, `terminal-exit`: busy12 stress and timing
    evidence) run with no other stage.
  - `budgeted` stages (every workload cell and SQLite selection) run
    concurrently with each other and with light shared stages, within the
    core/memory caps and the runtime 8 GiB headroom. They never run alongside
    `heavy` stages (Cranelift-heavy shared builds and guests: the helper builds,
    the Rust matrix, `ordinary`, `terminal-module`, `storage`, `network`,
    `clusters`, `temporal-inspection`), which are scheduled first, nor alongside
    exclusive or solo stages.
  - Cells of the same workload stay ordered (success then canary; the four
    fixture cells as one chain) because they share fixed per-workload build and
    engine caches and producer files in the workload root.
  - `solo` stages (helper corpus regeneration, which rewrites tracked files)
    run with nothing at all, in every mode.
- Locks: `golangci` (golangci-lint refuses parallel runs), `cargo-check-target`
  (the Rust matrix shares one Cargo target), `fixture-temporal-broker`
  (`ordinary` and `clusters` both create the staged broker fixture files).
- Priority: tier 0 (cheap static checks) first, then the longest remaining
  path. When the top-priority ready stage is exclusive it reserves the host:
  other stages are admitted only if their estimate ends before the running
  stages do (no starvation, no delay of the exclusive stage).
- Fail-fast: after the first non-zero exit (command or post-check) no stage
  starts; running stages finish and are recorded. The driver runs stages from a
  recorded copy under `$QUAL_ROOT/driver/` (hashes in `driver.sha256`).

### Concurrency hazards handled (none weakens a check)

- In-tree scratch: `temporal-inspection` creates
  `service/history/.gomad-inspection-*` and `ordinary` (backend inspection
  tests) `common/gomaddebug/testdata/inspection-*` for the duration of their
  tests; both are untracked and not ignored. The scheduler registers each
  running stage's declared scratch globs (`sched/registry/`) before starting it;
  `frozen.py` exempts only `??` entries under those globs, only for other stages
  whose run interval overlaps the check. Tracked changes, every manifest digest
  and any other untracked path still fail; a stage never exempts its own scratch;
  the preamble and final checks are strict, and exclusive/solo stages see no
  overlapping scratch in the default (strict) mode. Exemption is by glob, not by
  owner: a stage leaving files under another running stage's glob is caught by
  that stage's or the final check, under the other label. `cleantree` stages
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
  `EXIT <label> 0` per required stage, nothing else, no `STEP-FAIL`, a
  `<label>.command` record and a receipt with the label, exit 0 and the run's
  source manifest (`stage-set-executed.json`).
- Limits of the static check: it compares stage-call text, so values resolved
  at run time (helper and module receipts, lint base, workflow prefix) and the
  post-check lines inside each step are verified by review and the unit tests,
  not by the gate. The `QUALDRV_*` variables are extra in every stage's
  environment (no test reads them).
- Kept per stage: `<label>.command`, `<label>.log`, `<label>.receipt.json`
  (now with `lane`), frozen-source checks before and after every stage, the
  serial post-checks. `stages.log` keeps the serial line format plus
  `lane=N`; a `STEP-FAIL <label> <status>` line marks any failing step
  (command or post-check).
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

### Conductor decision 2026-10-20: budgeted cells share the host

Workload cells and SQLite selections may run concurrently with each other.
Their budgets, deadlines, seeds and oracles are unchanged, so this is scheduling
only. Rationale: the cells' wall-clock budgets have wide headroom over observed
values.

| Multi-instance cell | Observed | Budget |
| --- | --- | --- |
| Execution | 304 s | 900 s |
| Preparation | 48 s | 600 s |
| Compiler peak RSS | 2.4 GiB | 6 GiB |
| Helper peak RSS | 3.4 GiB | 4/8 GiB |

Every result is still checked for `Budget.WithinBudget == true`. A budget
violation caused by contention fails the run rather than passing it.

The busy12 and terminal timing stages stay exclusive. `temporal-inspection`, the
longest shared stage on the critical path, now depends only on `helper-build`.
Its test reads the release helper through `GOMAD_WASM_TEST_HELPER`; it does not
read the conformance helper, whose path variable is exported as in the serial
order. It is ranked first among shared stages.

With this policy, `dag.tsv` estimates (12 cores) are:

| Durations | Cap 15 GiB | 15 GiB, shared 20% slower | Cap 13 GiB |
| --- | --- | --- | --- |
| Worst of retained runs | 60.4 min | 66.0 min | 66.0 min |
| q1 run, 2026-10-19 | 46.4 min | 50.8 min | 50.5 min |

The critical path (worst case) is: `helper-build` → `temporal-inspection` →
multi-instance success → canary → corpus → SQLite selections → `busy-*` →
terminal stages. The analysis below predates the decision.

### Wall-time estimate and reaching one hour (strict exclusivity, superseded)

Estimated with `scheduler.py simulate`, which uses the same admission code as a
real run, for 12 cores. Memory cap 15 GiB: 19 GiB host, minus about 2 GiB other
sessions, minus the 2 GiB margin. Durations come from `dag.tsv` (`est_s`, the
worst fresh-run duration in the retained stages.logs). Memory is the peak
process-tree RSS sampled from the 2026-10-19 q1 serial run
(`/home/agent/wt/evidence/qualdriver/q1-mem-samples.tsv`). Details are in
`/home/agent/wt/evidence/qualdriver/estimates.txt`.

| Durations | Cap 15 GiB | 15 GiB, shared stages 20% slower | Cap 13 GiB | 13 GiB, 20% slower |
| --- | --- | --- | --- | --- |
| Worst of retained runs (`dag.tsv`) | 68.4 min | 73.9 min | 71.8 min | 78.0 min |
| q1 2026-10-19 run (91/96 stages observed) | 52.6 min | 57.0 min | 54.8 min | 59.7 min |

The serial sum is 138.5 min (worst case) or 107.8 min (q1 run).

Critical path (worst case), with exclusive time serial wherever it runs:
`validate` → cheap static checks → `wasmmeasure-build` → `helper-build` ‖
`conformance-helper-build` → `ordinary-helper-publication` (≈5.6 min) →
`temporal-inspection` (1296 s, the longest shared stage; its `onebox` subtest
alone is 17 min) → exclusive chain: workload cells (onebox 349 s,
multi-instance 729 + 596 s, …) → SQLite selections → `terminal-host-admission`
→ `terminal-exit` → `busy-grouped` / `busy-boundaries` → corpus and catalogue.

The floor is about 5.6 min (helper ready), plus the longest shared stage
(18–22 min), plus the exclusive and solo serial sum (30.6–40.7 min). That comes
to about 55–68 min. The one-hour limit holds for q1-like durations, but not for
the worst retained durations.

What was done toward the limit (none of it weakens a check):
- Parallel DAG with one process per stage.
- The conformance helper builds alongside the release helper.
- The broker fixture is staged once by the driver instead of serializing
  `ordinary` and `clusters`.
- Shared work is scheduled first, ranked by remaining shared work.
- Live memory headroom is checked only where the 8 GiB guard needs it.
- Native stages are dropped and matched-release is moved to `run-perf.sh`.

Considered and not done:
- Sharing one helper build: the release and conformance helpers differ by
  feature set and both are required.
- Prebuilding guest modules once and reusing them by digest: the cells already
  reuse their preparation within a run (the multi-instance canary prepares in
  1–5 s against 48 s for success), and the inspection onebox module uses
  different build tags.
- Splitting the multi-instance cells: that is a test-contract change (one
  result per cell).

What-if estimates, worst-case durations, 12 cores, 15 GiB:

| Change (owner decision) | Wall |
| --- | --- |
| None (as committed, strict) | 68.4 min |
| Multi-instance cells shared (not exclusive) | 68.4 min |
| `temporal-inspection` split into per-subtest stages | 67.0 min |
| All workload cells and SQLite selections shared, budgets unchanged | 58.0 min |

Only relaxing the exclusivity of the budgeted workload cells brings the
worst-case estimate under one hour. Their budgets have wide headroom: for
multi-instance, execution is 304 s of a 900 s limit and preparation 48 s of
600 s. This is the owner's call. `--allow-background` (a `bg` stage may
overlap a `bgok` exclusive stage) is implemented but gives no modeled gain,
because shared work already ends when `temporal-inspection` does.

The serial q1 run on 2026-10-19 lost about 55 min to host or sandbox
suspensions: sampler gaps of 5, 5, 16.6, 16.6 and 17 min, and the
multi-instance canary showed 58 min of wall time for a 460 s test. The limit is
measured in wall-clock seconds, as the owner stated, so a suspended host fails
the limit (exit 3) even when every stage passes.

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
