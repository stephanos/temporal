---
satisfies: [R6, R7]
---
# fn-107-scala-umpire-prototype-for-standalone.13 Implement activity activation in the existing Testpilot SDK adapter

Touches: [common/testing/testpilot/temporal/worker/sdk.go, common/testing/testpilot/temporal/worker/interpreter.go, common/testing/testpilot/temporal/worker/sdk_test.go, common/testing/testpilot/temporal/worker/runtime_fixture_test.go, common/testing/testpilot/temporal/worker/README.md]

## Description
Fill the existing ActivityActivation protocol's SDK execution gap. Consumer and canary binding integration stays in task 9.

**Size:** M
**Files:** worker sdk.go/interpreter.go, focused SDK/runtime fixtures, worker README.

### Approach
- Register and interpret activity entrypoints through the existing SDK worker/session/reservation/routing boundary. Reuse the Scala-lowered script instruction vocabulary from task 6.
- Keep operation/attempt/activation identities correlated, and propagate SDK failure/completion through existing typed outcomes and Run observation recording.
- Reject unavailable worker capability during preparation/validation; maintain bounded activation and cleanup behavior.
- Pin current workflow/Nexus worker output behavior in focused fixtures before extending registration.

### Investigation targets
**Required:** common/testing/testpilot/temporal/worker/sdk.go:23; common/testing/testpilot/temporal/worker/interpreter.go:60; common/testing/testpilot/temporal/worker/routing.go; common/testing/testpilot/temporal/worker/reservation.go; proto/internal/temporal/server/api/testpilot/v1/program.proto:131.
**Optional:** common/testing/testpilot/temporal/worker/sdk_test.go; common/testing/testpilot/temporal/worker/runtime_fixture_test.go.

### Quick commands
`mise exec -- go test -tags test_dep ./common/testing/testpilot/temporal/worker/...`.

## Acceptance
- [ ] The existing SDK adapter executes the Scala-lowered activity script and reports typed completion/failure with correlated activation identity.
- [ ] Missing registration/capability, invalid script input, cancellation, and cleanup failures follow existing bounded adapter semantics.
- [ ] Repeat/concurrent activation state remains isolated through the existing reservation boundary.
- [ ] Existing workflow/Nexus SDK fixtures remain unchanged and focused worker gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
