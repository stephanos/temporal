---
satisfies: [R2, R5]
---
# fn-129-activity-coverage.2 Respond by ID: a service actor

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R2, R5 (comparison P2-7). A `service` actor (fn-126 decision 26's `Actor`) whose completion, failure and cancel answers go by activity ID rather than task token, enabled in the phases the server allows (`chasm/lib/activity/model/model.go`'s `ByID` forms, e.g. `scheduled` and `paused` too). Their outcomes in other phases follow fn-128's rejection rows. Realize through RespondActivityTaskCompletedById / FailedById / CanceledById, with at least one live Query (e.g. a by-ID completion of a scheduled activity no worker polled).

## Acceptance
- [ ] The by-ID answers are rows in exactly the phases `model.go` allows, citing it.
- [ ] At least one live Query is realized and its Case ran once.
- [ ] New Cases listed; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
