---
satisfies: [R1, R2, R3, R4]
---
# fn-189-gomad-sync-with-upstream-temporal-main.1 Merge upstream temporalio/temporal main into gomad and requalify

## Description
Fetch github.com/temporalio/temporal main, merge it into gomad while preserving retained evidence history and bounded server seams, reconcile packs/pins, rerun default host gates and the WASM qualification ladder, and retain identities and the handover.

The owner committed the resolved source batch as `0077477965`. The [original checkpoint](../artifacts/fn189-upstream-sync-2026-10-17.md) preserves its passing gates and seven unresolved WASM tests. The owner then explicitly requested fixing those seven failures. Current work is limited to preparation/replay and warm-preparation admission repairs plus their verification; later fn-187/fn-185 milestones and native execution remain paused/deferred. Full fn-189 acceptance still requires all retained R1-R4 gates, including the current-candidate workload ladder and budgets; fixing these seven tests alone does not establish that acceptance.

The [seven-test repair report](../artifacts/fn189-wasm-gate-fixes-2026-10-17.md) records the targeted fixes and verification. These changes remain uncommitted.
## Acceptance
R1-R4 of the spec pass on the merged candidate with one handover.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
