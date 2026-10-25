# fn-242 amendment: gate speed

Status: **applied to fn-242 on 2026-10-25 following the owner's direction to the conductor.** The authoritative text is the Owner gate-speed amendment in [the fn-242 spec](../specs/fn-242-gomad-shrink-the-qualification.md). This artifact preserves the proposal and its measurement rationale. The follow-on work outside fn-242 (cheap-first ordering, stage reuse, inner loop) lives in fn-253, filed the same day.

## Why

A full cycle has never finished inside the hour. Cycle 8 (2026-10-24, parallel, 12 cores; `/home/agent/wt/evidence/fn232-split-closure/resume-closure-cycle8/cycle/root/`) used 3,554 s, of which the scheduler took 2,893 s. It ran 25 of 96 stages before failing, and 71 never started. The sum of stage estimates in `parallel/dag.tsv` is about 8,310 s. Three choices in the current driver account for most of that time, and R2 as written ("the same … resource policy") would carry all three into the Rust cycle verb. The port already carries two of them: `cycle/custody` `check_files`, and `CARGO_BUILD_JOBS=1` in `cycle/plan/leaf/primary.rs:394`.

1. **Each stage re-checks the whole frozen inventory, twice.** `guard()` (`driver.py:449`) runs before and after each stage's work. It SHA-hashes all 266,363 frozen files (7.4 GB, `cycle/freeze/driver.json`) and runs `git ls-files` and `git status`. In `portable` that was about 43 s before and 50 s after, around 58 s of actual work plus a 69 s reader. That is why stages estimated at 1–7 s took 160–220 s.
2. **Stages are pinned to one job.** `static_stages.py:426` sets `CARGO_BUILD_JOBS=1`, `RUST_TEST_THREADS=1` and `GOFLAGS=-p=1`, and primary stages do the same, even though the scheduler has already admitted the stage against a core budget.
3. **The same code is compiled repeatedly.** `rust-lint` ran 13 release clippy builds, one per feature set: 650 s against a 94 s estimate. The Rust test chain rebuilds `gomad-wasmhost` once per feature set, one after another, under the `cargo-check-target` lock.

## Proposed amendment text

**Owner gate-speed amendment, 2026-10-25.** R2's "same resource policy" does not cover the three mechanisms below. The Rust cycle verb replaces each one with the version described here. Integrity, bounds, the 3,600-second limit and every stage's leaf assertions are unchanged.

- **G1 (custody once per cycle):**
  - The freeze hashes the full inventory once, as it does today.
  - Each stage boundary checks only the cheap invariants: the config digest, the frozen HEAD, the symlink inventory, and `git status --porcelain --untracked-files=all` limited to the allowed scratch paths.
  - The cycle's final step (before the verdict) hashes the full inventory again.
  - A digest mismatch at any point fails the cycle, exactly as today.
  - Catching proof (R1 style):
    - Mutating a tracked file mid-cycle fails the cycle at the final check.
    - Adding an untracked file or moving HEAD fails it at the next stage boundary.
    - The mutation and its control are recorded against the original `guard()` obligation IDs.
  - A stage that reads a frozen file whose content changed mid-cycle still fails, because the leaf digests it consumes are unchanged.
- **G2 (job count matches the admitted cores):**
  - Each stage's `CARGO_BUILD_JOBS`, `GOFLAGS=-p` and test-thread count equal the cores the scheduler admitted it with, not 1.
  - Exceptions are only stages whose assertions depend on single-threaded execution (timing, exclusive and solo stages, and any stage whose leaf contract names a thread count). Each exception is listed in the stage table with its reason.
  - Memory admission is unchanged.
- **G3 (compile once):**
  - Stages that need the same artifacts get them from one producer stage, sharing one cycle-scoped target directory.
  - The clippy feature matrix runs as one stage over a shared target.
  - The Rust test chain builds each `gomad-wasmhost` feature set once.
  - The `cargo-check-target` lock serialises only the stages that actually write that target.
  - Leaf argv for lint and test assertions is unchanged apart from `--target-dir` and the job count.

Acceptance addition to R6: a parallel 12-core run shows the per-stage boundary overhead (stage wall time minus leaf wall time) under 5 s at the median. The stage table records the old and new wall time for `portable`, `rust-lint` and the Rust test chain.

## What does not change

- Fail-fast.
- Exclusive and solo admission.
- The memory guard.
- Every leaf assertion and bound.
- Final qualification in fn-246 still runs every stage fresh.
