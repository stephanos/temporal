---
satisfies: [R6]
---
# fn-128-close-the-activitys-precision-gaps.6 Close: evidence map, live Cases, MILESTONES

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Runs after that batch's single regeneration and full gates; its live run is shared by fn-128.6 and fn-129.5.
R6. Write the evidence map: each divergence the comparison named (unpause after backoff, schedule-to-start in backoff, repeated RequestCancel, silent rejections, unlimited retries, unchecked stutter facts) to the rule, row or Query that fixes it; and every changed table, Query answer and Case with why, collected from tasks 1-5. Run the live generated Cases once (only the known ShutdownWorker-race INCONCLUSIVEs allowed). Remove fn-128 from `MILESTONES.md` and drop it from fn-129's gate. Close the spec.

## Acceptance
- [ ] The evidence map covers every divergence and change.
- [ ] Live Cases ran once with only the known INCONCLUSIVEs; log path recorded.
- [ ] `MILESTONES.md` updated and the spec closed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
