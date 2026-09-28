---
satisfies: [R4, R2]
---
# fn-99-gomad-f5-one-workflow-executing.2 Tighten the user-timers manifest expectation and record F5

## Description
Set expectation `qualified` where it qualifies; update CI assertion; record darwin numbers in F5 status.

## Acceptance
- manifest and CI updated, F5 status updated

## Done summary
`temporal.json` now expects `qualified` on darwin/arm64 for `user-timers-workflow` and `activity-batch-cancel-boundary` (linux/amd64 stays `intermittent` pending a linux run after the runtime change), and `make gomad3-qualification` on darwin met every expectation with 18/18 supported, 0 failed, 0 infrastructure errors (user-timers 5774/5701 decisions, exact replay on both seeds). The darwin CI assertion now requires 18 supported, 0 failed and three qualified tier 3 workloads; the integration README, F5 status (darwin sysconf and mark-start greying/allp channels with per-seed metrics) and tracking row, an F6 note, and a dated GOMAD3_NEXT deterministic-GC update record the outcome. `TestQualificationManifestsUsePortableV3` was red before this task (core corpus count pinned at 5 after task .1 added two workloads) and now expects 7. Reports copied to the scratchpad `fn99-2-reports/`; the 4.6 GB retained artifacts were deleted.

stage: impl-review - ran (codex fan-out round 1: three draws SHIP, no findings)
## Evidence
- Commits: ba09401c5b585cc55bdee2f9b3fe80c6102dd609
- Tests: make gomad3-qualification (darwin/arm64: expectations-met=true supported=18 unsupported=0 failed=0 infrastructure-errors=0 completed=18/18), jq -e <darwin CI predicate> tools/gomad3/.toolchain/temporal-qualification-set.json (true), make gomad3-integration-test (ok), baseline: none (spec defines no Quick commands); make gomad3-integration-test was red pre-edit (stale core.json count 5 vs 7 from task .1), fixed in this task
- PRs: