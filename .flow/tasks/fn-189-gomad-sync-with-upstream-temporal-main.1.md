---
satisfies: [R1, R2, R3, R4]
---
# fn-189-gomad-sync-with-upstream-temporal-main.1 Merge upstream temporalio/temporal main into gomad and requalify

## Description
Fetch github.com/temporalio/temporal main, merge it into gomad while preserving retained evidence history and bounded server seams, reconcile packs/pins, rerun default host gates and the WASM qualification ladder, and retain identities and the handover.

The owner committed the resolved source batch as `0077477965` and the seven-test repair as `bad3c401b5`. The [original checkpoint](../artifacts/fn189-upstream-sync-2026-10-17.md) and [repair report](../artifacts/fn189-wasm-gate-fixes-2026-10-17.md) retain historical Darwin evidence; their statements that those changes were uncommitted are historical.

The latest owner objective is fn-189 and fn-187, prioritizing tasks whose acceptance is independent of performance/resource measurements. This session executes on Linux/arm64. Native execution stays deferred under fn-128/fn-149. Keep commits with the owner; no push, PR or CI action is authorized. Do not resume fn-185 profiling in this batch.

The [source reconciliation checkpoint](../artifacts/fn189-pack-source-reconciliation-2026-10-17.md) records five mapped compatibility-pack refreshes, exact source review, unchanged capability dispositions and historical fixtures, affected lint, integrated `make -C tools/gomad validate`, and integrated focused source regressions (all passing). These are source-only results, not native execution or full fn-189 acceptance. The raw all-pack pin-impact report retains its historical-fixture failures; the mapped refresh gate is green.

The [Linux source-gate checkpoint](../artifacts/fn189-linux-source-gates-2026-10-17.md) now records a 2026-10-18 current-candidate canonical default lint pass (exit 0): original 112 host packages, unchanged baseline/configuration/ten-minute timeout, fixes disabled, Linux pinned tools, and zero reported new issues. Both golangci-lint and errortype vet completed. This supersedes only the earlier interrupted default lint result, not other incomplete gates or historical results.

Qualification work and raw logs remain under `.flow/tmp/fn189-linux/`. Current-candidate runtime/workload, exact replay, identity, resource-budget and remaining default host gates remain required and unproved. Fn-189 stays in progress until every retained R1-R4 obligation is evidenced.

2026-10-18 documentation reconciliation: MILESTONES now names exact upstream base `48a1e5ff025e55b3d4d0f44247423d4c279ff636` and merge `0077477965a5b766b95a667247a9f562c100941b`, linking the integration report. That report now opens with a current-state note and explicitly labels its preserved Darwin checkpoint as historical. Root verified the upstream ancestor at HEAD `95eed5c83a26eb1e93f2b1ba2f971e33bc1d51bc`, exact merge parents, repair commit and missing MERGE_HEAD. Targeted diff whitespace checks pass. This advances R4 documentation only; it is not a fresh source review, qualification result, completion or publishing authorization.
## Acceptance
R1-R4 of the spec pass on the merged candidate with one handover.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
