---
satisfies: [R3, R5, R6]
---
# fn-33-run-resumable-semantic-exploration.5 Prove one bounded campaign and document the ownership split

## Description
Run one bounded local campaign, close adversarial matrices, and document R3/R5/R6.

### Review reconciliation (normative)

The live proof selects the existing fn-5 catalog subject `workflow-nexus.query.exact-action-caller-closure`, joined to Task `.6`'s runnable binding. Its internal fn-17 exact source is `temporal.nexus.caller-closure.runtime-smoke`: one byte-/identity-preserved caller-closure ExperimentSpec, ephemeral-local RuntimeConfiguration, exhaustive budget one, seed zero, no faults/pins, parallelism one, and a campaign deadline within the checked 1-second-to-5-minute range. Fake campaigns own the parallelism and pinned-regression proof.

**Size:** M
**Files:** `tools/umpire/campaign/**`, `tools/umpire/cmd/umpire-fuzz/**`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `docs/development/**`
**Touches:** [tools/umpire/campaign/**, tools/umpire/cmd/umpire-fuzz/**, model/README.md, model/Umpire/ARCHITECTURE.md, docs/development/**]

### Approach
- Prove pinned regressions, parallel leases, time exhaustion, resume, lineage forks, state locks, progress, and semantic coverage lineage with fakes; prove the one-point runner/conformance vertical path live.
- Document Lean versus Go ownership and honest non-completeness.
- Preserve existing comments and generated projections.

### Investigation targets
**Required** (read before coding):
- `model/README.md` — model workflow documentation
- `model/Umpire/ARCHITECTURE.md` — semantic module contracts
- `docs/development/testing.md` — developer testing entry point
## Acceptance
- [ ] R3/R5/R6 end-to-end and mutation checks pass.
- [ ] Focused Go/Lean tests and aggregate regression gates pass.
- [ ] Documentation distinguishes exhaustion from completeness and execution from conformance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
