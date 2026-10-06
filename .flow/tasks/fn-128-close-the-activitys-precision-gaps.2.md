---
satisfies: [R2]
---
# fn-128-close-the-activitys-precision-gaps.2 Rejections are rows: failedPrecondition and invalidArgument

## Description
R2 (comparison P1-3, P1-4). `Outcome` gains `failedPrecondition` and `invalidArgument`. Every (state, action class) the server answers with one of them becomes a rule with that outcome, citing the server code, in both `ActivityProduct` and `ActivitySystem`; the refinement maps outcomes by name. The seven `silent-rejection` acceptances in `model/ir/activity-standalone*.lint.json` are removed. `closedIsRejectedUniformly` keeps `notFound` for closed activities. A repeated `RequestCancel` in `cancelRequested` answers `failedPrecondition` (`model.go:201-202`).

Each realized control declares the gRPC code of each outcome through fn-133.1's `answers(…)` (read from the Run's `InstructionOutcome.protocol_code`), so the evidence confirms the rejection, not only that a call returned. Runs after task 1 so the rows are written against the dispatch field.
## Acceptance
- [ ] Each formerly silent pair is a row with its outcome and a citation in both levels; `make lint-model` reports no `silent-rejection` for the activity.
- [ ] The repeated-`RequestCancel` divergence is a Query or fixture answering `failedPrecondition`.
- [ ] `closedIsRejectedUniformly` still holds with `notFound`.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
