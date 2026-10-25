---
satisfies: [R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.1 Baseline: per-stage, per-binary, per-test and TLC-launch timings

Touches: [model/design/gate-speed.md, .flow/tmp/fn254-baseline/**]

## Description
Measure before changing anything, so every later task is judged against real numbers (spec R1).

**Size:** M
**Files:** `model/design/gate-speed.md` (new); probe scripts, README and `run.sh` under `.flow/tmp/fn254-baseline/`.
**Touches:** [model/design/gate-speed.md, .flow/tmp/fn254-baseline/**]

### Approach
- Quiet host: confirm no other cargo/TLC/java processes (`pgrep -fa 'cargo|rustc|java'`), main checkout, shared `CARGO_TARGET_DIR`, warm build.
- Stage times: `UMPIRE_TIME_STEPS=1 make model-check`, then `BASH_ENV=/dev/null UMPIRE_BACKENDS=require make model-validate` (Apalache skipped, matching routine use).
- Compile vs run: `cargo test --workspace --no-run --message-format=json` (timed), then run each listed test binary on its own with wall time, so the serial cost per binary is visible. Per-test times from each binary's output with `--test-threads=1` only for the slowest five binaries.
- TLC launches: count `tlc2.TLC` processes per fixture via a wrapper on `PATH` or a counter env in a scratch build; record launches, total TLC wall time and the per-launch fixed cost.
- Record the host (cores, memory), tool versions and commands. Unknowns are listed as unknown.
- Verdict format per `model/design/` convention; probe kept in `.flow/tmp/fn254-baseline/`.

### Investigation targets
**Required:**
- `model/scripts/time-gate.sh` — existing stage timer
- `Makefile:1270-1315` — `model-check`, `model-time`, `model-validate`
- `model/umpire/backends/tlc/src/runner.rs:129-250` — TLC launch path
- `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json` — previous stage timings

### Acceptance
- [ ] `model/design/gate-speed.md` has per-stage, per-binary (compile and run separately) and slowest-test tables, and TLC launch counts per fixture.
- [ ] The five largest costs are named with evidence, and each later fn-254 task is ranked by its expected saving.
- [ ] Probe reproducible from `.flow/tmp/fn254-baseline/run.sh`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
