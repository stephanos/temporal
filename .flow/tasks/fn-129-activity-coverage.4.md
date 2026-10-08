---
satisfies: [R4]
---
# fn-129-activity-coverage.4 Exploration on the activity's find Queries

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R4 (comparison P3-13). Add `.explore` to each activity `find` Query where it yields witnesses the pinned Scenarios miss; measure for each Query the witnesses found and the search cost, and leave it pinned where exploration adds nothing. Runs last so it explores the Model with heartbeat, by-ID and reset in place.

## Acceptance
- [ ] Each `find` Query is either exploring, with the new witnesses listed, or pinned, with the measurement that says why.
- [ ] New Cases listed; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
