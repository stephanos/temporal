---
satisfies: [R1, R2, R3, R4]
---
# fn-189-gomad-sync-with-upstream-temporal-main.1 Merge upstream temporalio/temporal main into gomad and requalify

## Description
Fetch github.com/temporalio/temporal main, merge it into gomad while preserving retained evidence history and bounded server seams, reconcile packs/pins, rerun default host gates and the WASM qualification ladder, and retain identities and the handover.

The owner committed the resolved source batch as `0077477965` and the seven-test repair as `bad3c401b5`. The [original checkpoint](../artifacts/fn189-upstream-sync-2026-10-17.md) and [repair report](../artifacts/fn189-wasm-gate-fixes-2026-10-17.md) retain historical Darwin evidence; their statements that those changes were uncommitted are historical.

The owner resumed the full WASM milestone objective on 2026-10-17. Continue the remaining R1-R4 acceptance on the current candidate, then fn-187 and fn-185 in milestone order. This session executes on Linux/arm64; retained Darwin results do not qualify this host. Native execution remains deferred. Preserve all workload, exact replay, identity and resource-budget gates. Keep commits with the owner as previously requested; no push, PR or CI action is authorized.

Current qualification work and raw logs live under `.flow/tmp/fn189-linux/`; retain one source-bound batch handover when the commands finish. Fn-189 stays open until the complete current-candidate gates are proved.
## Acceptance
R1-R4 of the spec pass on the merged candidate with one handover.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
