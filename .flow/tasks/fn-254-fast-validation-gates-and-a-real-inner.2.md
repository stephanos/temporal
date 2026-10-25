---
satisfies: [R3]
---
# fn-254-fast-validation-gates-and-a-real-inner.2 TLC runner: one batched run per check, split on failure, parallel properties

Touches: [model/umpire/backends/tlc/src/**, model/umpire/backends/tlc/tests/**, model/umpire/backends/process/src/**]

## Description
Cut TLC launches from about one per property to one or two per check for healthy modules (spec R3). Verdicts and traces stay exactly as today.

**Size:** M
**Files:** `umpire/backends/tlc/src/runner.rs`, `module.rs`; `umpire/backends/process/src/` if the limit lives there; TLC tests.
**Touches:** [model/umpire/backends/tlc/src/**, model/umpire/backends/tlc/tests/**, model/umpire/backends/process/src/**]

### Approach
- Keep the claim-free exploration run (it produces the state dump). Then run one cfg with every INVARIANT/PROPERTY from `split_cfg`.
- No violation: every property Verified from that run. Violation: record the reported property's verdict and trace exactly as `execute` does now, then rerun the remaining properties as a batch, until each has a verdict. Properties TLC cannot report exactly in a batch (check liveness under fairness first) keep individual runs; document which.
- Optional parallelism: independent reruns (and the exploration with the first batch) may run concurrently up to a `Limits` setting (default from available cores, capped). Each run keeps its own scratch directory.
- Measure the JVM flags from the baseline probe (`-XX:TieredStopAtLevel=1`, in-memory vs `MSBDiskFPSet`) on real fixtures. Adopt only flags that are faster on fixtures and keep large-model behaviour.
- **Equivalence pin:** a test-only switch runs the old per-property path. A control runs both over every TLC fixture and mutant in the workspace and asserts identical `TlcResult` verdicts and traces, including a module with two violated properties and a liveness violation under fairness.

### Investigation targets
**Required:**
- `model/umpire/backends/tlc/src/runner.rs:76-250` — `check_module`, `check`, `execute`, `selected_violation`
- `model/umpire/backends/tlc/tests/controls.rs` — real TLC controls
- `model/umpire/dev/testing/src/backend.rs:78-120` — fixture callers

### Acceptance
- [ ] Healthy modules take at most two TLC launches per check (exploration plus one batch); launch counts recorded against the baseline.
- [ ] Equivalence control passes over every fixture and mutant, including multi-violation and liveness cases.
- [ ] `UMPIRE_BACKENDS=require` workspace tests pass; TLC wall time recorded against the baseline.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
