---
satisfies: [R4]
---
# fn-128-close-the-activitys-precision-gaps.4 Stutter facts checked: visible on ActivitySystem's refinement

## Description
**Batch:** ACTIVE approved Activity Model batch (MILESTONES.md, fn-128 → fn-138 → fn-129). Do not run `make umpire-gen-model`, regenerate production fixtures or Cases, or run the full gates in this task; production IR/answer/Case comparison is checked at the shared batch's single regeneration against ee32b5fa6023c4ab8dfcc928f07d79afe5587186, not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own. Fresh independent review and the single live run remain mandatory at the shared fn-128.6/fn-129.5 boundary.

R4 (comparison P2-8). `ActivitySystem`'s `refinement` declares exhaustive public-status-fact `visible`, including each typed `statusTimedOut`; exclude only `attemptCount`, which the Product omits. Decide each refinement rejection this surfaces (Product row, System fact, or a recorded exception) and record the decision in the done summary. Preserve the System mapping/facts and strict checker. The Product carries creation, held pause and pause withdrawal observations while retaining its existing abstract acceptance/rejection alternatives. Runs after tasks 1-3 because they change the facts.

### Touches
- `model/temporal/features/activity/standalone/{product/Product.scala,system/System.scala}`
- Focused Activity/refinement tests in `model/temporal/features/activity/standalone/` and only directly required tests in `tools/umpire/check/`.
- `MILESTONES.md`: own fn-128.4 row and As-of date only.
- Ignored current-source scratch lift/admission/refinement proof and task evidence under `.flow/tmp/activity-batch/`; no generic engine/DSL edits or production generated artifacts.
## Acceptance
- [ ] `visible` is declared and the refinement check passes.
- [ ] Every rejection it surfaced is listed with its decision.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
