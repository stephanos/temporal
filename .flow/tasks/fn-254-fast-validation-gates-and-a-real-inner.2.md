---
satisfies: [R3]
---
# fn-254-fast-validation-gates-and-a-real-inner.2 TLC runner: one batched run per check, split on failure, parallel properties


## Description
Cut TLC launches from about one per property to one or two per check for healthy modules (spec R3). Verdicts and traces stay exactly as today.

**Size:** M
**Files:** `umpire/backends/tlc/src/runner.rs`, `module.rs`; `umpire/backends/process/src/` if the limit lives there; TLC tests.
**Touches:** [model/umpire/backends/tlc/Cargo.toml, model/umpire/backends/tlc/src/**, model/umpire/backends/tlc/tests/**, model/umpire/backends/process/src/**]

### Approach

Admission refinement: the TLC crate manifest may declare the task's test-only independent-strategy equivalence switch; no new dependency is assumed. Scope overlaps with consolidation/workspace moves remain serialized.
- Keep the claim-free exploration run (it produces the state dump). Then run one cfg with every INVARIANT/PROPERTY from `split_cfg`.
- No violation: every property Verified from that run. Violation: record the reported property's verdict and trace exactly as `execute` does now, then rerun the remaining properties as a batch, until each has a verdict. Properties TLC cannot report exactly in a batch (check liveness under fairness first) keep individual runs; document which.
- Optional parallelism: independent reruns (and the exploration with the first batch) may run concurrently up to a `Limits` setting (default from available cores, capped). Each run keeps its own scratch directory.
- Retain existing JVM flags initially. Optional scoped experiments may justify a flag change only with measured gains and unchanged large-model behavior; do not recreate a fresh baseline campaign before implementing batching. Historical startup observations are references, not a new measured claim.
- **Equivalence pin:** a test-only switch runs the independent per-property path over every TLC fixture and mutant, including multiple and mixed safety/liveness violations under fairness. Compare semantic fields: property identity/verdict, trace states/private observers and lasso destination. Raw subprocess `output` necessarily differs; retain and order it deterministically as evidence. If batch attribution or a trace differs, rerun that property independently. The two-launch healthy target applies per check configuration, including separately inventoried progress/negative configurations.
- Malformed output, timeouts/resource refusal, evaluation errors, unattributed violations and incomplete batches never verify unresolved properties. Preserve existing subprocess capture, output/dump bounds and cleanup; each concurrent run owns its scratch.
- Coordinate the runner limit with nextest groups and gate overlap: bound the product of concurrent tests, JVMs per test and heap allocation. Record the supported concurrent-invocation policy and instrument peak JVM count. Serialize cache/process ownership with .13 without inventing a semantic prerequisite.

### Investigation targets
**Required:**
- `model/umpire/backends/tlc/src/runner.rs:76-250` — `check_module`, `check`, `execute`, `selected_violation`
- `model/umpire/backends/tlc/tests/controls.rs` — real TLC controls
- `model/umpire/dev/testing/src/backend.rs:78-120` — fixture callers

## Acceptance
- [ ] Healthy modules take at most two TLC launches per check (exploration plus one batch); launch counts recorded against the baseline.
- [ ] Equivalence control passes over every fixture and mutant, including multi-violation and liveness cases.
- [ ] `UMPIRE_BACKENDS=require` workspace tests pass; TLC wall time recorded against the baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
