---
satisfies: [R3, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.3 Generated request_findings replaces Temporal's Wire enum

## Description
Implement **Generated findings (B2)**: generate `request_findings(method, bytes)` per service with supplemental-catalog hooks; delete `Wire` in `temporal/realization/src/probes.rs:311-551`. Answer whether `mutate`/`mutate_start` can use the generated `mutate_workflow_request` and do it if so.

## Acceptance
- [ ] No hand-written `Wire` enum; Temporal probe findings unchanged.
- [ ] The `mutate` question is answered in the done summary.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
