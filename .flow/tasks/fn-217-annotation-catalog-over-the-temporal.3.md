---
satisfies: [R3, R5]
---
# fn-217-annotation-catalog-over-the-temporal.3 Mini activity catalog with expected errors; fill and validate in the slice test

## Description
`model/temporal/features/activity_mini/api.rs`: the complete catalog for every message fn-216 touches (`StartActivityExecutionRequest`, `DescribeActivityExecutionResponse`, `PollActivityTaskQueueResponse`, `RespondActivityTaskCompletedRequest`, `RespondActivityTaskFailedRequest`) and the `WorkflowService` method classification; every request-side rule and constraint with its `Expectation` restated from the server's validator (`covers:` names the Go function); Temporal tags (`Visibility { lag_polls }`) defined here if used. Wire `fill` (request_id, inputs) and `validate` (every request sent, every response observed) into fn-216.3's slice test; `judge` honours `Visibility` if declared. After .2. **Touches:** `model/temporal/features/activity_mini/**`, `model/temporal/wasm/**`, `common/testing/testpilot/temporal/**` (test only).

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
