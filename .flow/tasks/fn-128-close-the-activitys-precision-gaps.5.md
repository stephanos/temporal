---
satisfies: [R5]
---
# fn-128-close-the-activitys-precision-gaps.5 Adopted from the Go model: precedence Property, attempt counts, time-window note

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R5 (comparison section 5 items 5-7). (1) A checked Property `cancelIsNotUndone`: from `cancelRequested` no pause leaves it (reset joins in fn-129). (2) An `attemptCount` observation in every activity Case, not only the retry Query; the realization reads it in every Case. (3) The Go model's nominal vs real time window note (`model.go:300-310`) as `because` text on the deadline rules.

## Acceptance
- [ ] `cancelIsNotUndone` is checked and holds.
- [ ] Every activity Case reads the attempt count.
- [ ] The deadline rules carry the time-window `because`.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
