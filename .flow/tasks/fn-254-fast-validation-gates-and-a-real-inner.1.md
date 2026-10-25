---
satisfies: [R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.1 Baseline: per-stage, per-binary, per-test and TLC-launch timings


## Description
Measure before changing anything, so every later task is judged against real numbers (spec R1).

**Size:** M
**Files:** `model/design/gate-speed.md` (new); probe scripts, README and `run.sh` under `.flow/tmp/fn254-baseline/`.
**Touches:** [model/design/gate-speed.md, .flow/tmp/fn254-baseline/**]

### Approach
- Quiet host: confirm no other cargo/TLC/java processes (`pgrep -fa 'cargo|rustc|java'`), main checkout, shared `CARGO_TARGET_DIR`, warm build.
- Stage times: `UMPIRE_TIME_STEPS=1 make model-check`, then `BASH_ENV=/dev/null UMPIRE_BACKENDS=require make model-validate` (Apalache skipped, matching routine use).
- Compile vs run: time `cargo test --workspace --no-run --message-format=json`, retain binary identities and available build/link timings, then measure binary execution separately. Record per-test timings across the inventory using supported harness output/instrumentation; mark unavailable compile-versus-link subdivisions or test timings unknown with their reason. Do not infer per-binary compile cost from an aggregate build.
- TLC launches: count `tlc2.TLC` processes per fixture via a wrapper on `PATH` or a counter env in a scratch build; record launches, total TLC wall time and the per-launch fixed cost.
- Record the host (cores, memory), tool versions and commands. Unknowns are listed as unknown.
- Retain the exact package/target/name/ignored inventory, doctests and UI-case inventory as the migration baseline. Preserve stage identity mappings and record any interference, warm-state difference or explicitly opted-in backend separately. A prior required gate may supply equivalent measurements, with its actual conditions stated; do not repeat broad gates solely to duplicate evidence.
- Verdict format per `model/design/` convention; probe kept in `.flow/tmp/fn254-baseline/`.

### Investigation targets
**Required:**
- `model/scripts/time-gate.sh` — existing stage timer
- `Makefile:1270-1315` — `model-check`, `model-time`, `model-validate`
- `model/umpire/backends/tlc/src/runner.rs:129-250` — TLC launch path
- `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json` — previous stage timings

## Acceptance
- [ ] `model/design/gate-speed.md` has per-stage, per-binary (compile and run separately) and slowest-test tables, and TLC launch counts per fixture.
- [ ] The five largest costs are named with evidence, and each later fn-254 task is ranked by its expected saving.
- [ ] Probe reproducible from `.flow/tmp/fn254-baseline/run.sh`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
