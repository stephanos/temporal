---
satisfies: [R2, R3]
---
# fn-229-realization-ergonomics-shared-temporal.2 After .1: support module with Temporal helpers; activity Realization rewritten; Phase TryFrom beside the Model

## Description
Create the realization crate's support module: Observed (method plus typed response or gRPC code), rejection(code), RPC method naming, plain_text(&Payload), Failure::application, reply(&impl Name), callback::<T>. Unit controls per refusal case; no activity names in the module. Rewrite temporal/features/activity/temporal.rs against the helpers: impl TryFrom<ActivityExecutionInfo> for Phase beside the Model, imported actor enums, Projection::step with Cancel's expected outcomes, policy through plain_text, replies through the helpers, payload.clone() for the echo; drop the #[path] prelude if the fixture runner permits, otherwise keep it with a one-line reason. Body at most eighty lines excluding imports and docs. All activity goldens byte-identical; dedicated live slice, Go parity and fn-218 cover tests pass; model-time pasted.

## Acceptance
- [ ] support module with the seven helpers and refusal controls; no activity names
- [ ] Activity Realization rewritten, at most eighty lines of body
- [ ] Activity goldens byte-identical; slice, parity and cover tests pass; model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
