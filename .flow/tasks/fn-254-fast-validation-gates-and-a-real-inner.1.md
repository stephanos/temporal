---
satisfies: [R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.1 Consolidate historical validation timings and unknowns


## Description
Consolidate existing evidence for R1. The owner explicitly waived fresh baseline measurements on 2026-10-25: "can we ignore baseline measurement? we already have historic data". Proceed to optimization after this lightweight report; do not wait for a quiet-host or build-cache lease.

**Size:** S
**Files:** `model/design/gate-speed.md` (new); retained optional probe scripts and historical source copies under `.flow/tmp/fn254-baseline/`.
**Touches:** [model/design/gate-speed.md, .flow/tmp/fn254-baseline/**]

### Approach
- Read fn-247's committed validation record for its 2,093s outer required/Apalache-opt-in run and stage breakdown; retain its exact source/mode rather than treating it as routine default cost.
- Read fn-237's retained default/required runs and existing per-binary/UI/build logs, including fn2511's annotate UI evidence. State workload/profile/host differences and any absent attribution.
- Report unknown per-test/linker/TLC counts explicitly. Do not turn a code-derived launch prediction into a measurement or infer per-binary compile time from an aggregate build.
- Name the five largest supported costs and rank expected optimization savings as estimates. Preserve source receipts/logs needed for review, without new production builds/tests/timing runs.
- Keep the prepared synthetic-tested observer/probe optional for R6, not a completion gate here. Correctness inventories remain owned by .3/.4/.5/.7 before their migrations.

### Investigation targets
**Required:**
- `model/scripts/time-gate.sh` — existing stage timer
- `Makefile:1270-1315` — `model-check`, `model-time`, `model-validate`
- `model/umpire/backends/tlc/src/runner.rs:129-250` — TLC launch path
- `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json` — previous stage timings

## Acceptance
- [ ] Historical timings and available counts are cited with their actual source/mode/coverage; unavailable per-test, compile/link or launch data is unknown with a reason.
- [ ] Supported largest costs and estimated optimization priorities are clear; incomparable runs and failed/partial gates are labelled honestly.
- [ ] No fresh baseline production measurement is required or claimed. Optional prepared probes and their synthetic evidence are retained separately for possible final measurements.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
