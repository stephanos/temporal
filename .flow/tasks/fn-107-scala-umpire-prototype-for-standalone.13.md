---
satisfies: [R6, R7]
---
# fn-107-scala-umpire-prototype-for-standalone.13 Implement activity activation in the existing Testpilot SDK adapter

Touches: [common/testing/testpilot/temporal/worker/sdk.go, common/testing/testpilot/temporal/worker/interpreter.go, common/testing/testpilot/temporal/worker/driver.go, common/testing/testpilot/temporal/worker/session.go, common/testing/testpilot/temporal/worker/registry.go, common/testing/testpilot/temporal/worker/routing.go, common/testing/testpilot/temporal/worker/reservation.go, common/testing/testpilot/temporal/worker/carrier.go, common/testing/testpilot/temporal/worker/*_test.go, common/testing/testpilot/temporal/worker/README.md, common/testing/testpilot/temporal/internal/activation/**, common/testing/testpilot/temporal/internal/delivery/**, common/testing/testpilot/temporal/driver.go, common/testing/testpilot/temporal/driver_test.go, common/testing/testpilot/temporal/profile.go, common/testing/testpilot/temporal/profile_test.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/internal/execution/dataflow.go, common/testing/testpilot/internal/execution/carrier.go, common/testing/testpilot/internal/execution/*_test.go]

## Description
Fill the existing ActivityActivation protocol's SDK execution gap. Consumer and canary binding integration stays in task 9.

**Size:** L
**Files:** existing worker/session/registration/reservation/carrier modules and fixtures; shared activation and delivery modules; neutral activity capability, profile and execution admission; worker README.

### Approach
- Register and interpret activity entrypoints through the existing SDK worker/session/reservation/routing boundary. Reuse the Scala-lowered script instruction vocabulary from task 6.
- Keep operation/attempt/activation identities correlated, and propagate SDK failure/completion through existing typed outcomes and Run observation recording.
- Reject unavailable worker capability during preparation/validation; maintain bounded activation and cleanup behavior.
- Pin current workflow/Nexus worker output behavior in focused fixtures before extending registration.
- Apply the grounded activity-sdk-prep inventory in .flow/tmp/fn-107/activity-sdk-prep/preparation.md. Task 6 owns the instruction/observation/carrier vocabulary and schema; this task supplies neutral Go execution and admission against those declarations, preserving existing workflow/Nexus wire and pooling behavior. Keep logical operation, SDK attempt and delivery identities distinct. Conditional scheduler or Contract changes remain outside this scope unless the task-6 representation proves they are required and the conductor explicitly updates scope.

### Investigation targets
**Required:** common/testing/testpilot/temporal/worker/sdk.go:23; common/testing/testpilot/temporal/worker/interpreter.go:60; common/testing/testpilot/temporal/worker/routing.go; common/testing/testpilot/temporal/worker/reservation.go; proto/internal/temporal/server/api/testpilot/v1/program.proto:131; activity-sdk-prep/preparation.md; common/testing/testpilot/temporal/internal/activation/activation.go; common/testing/testpilot/temporal/internal/delivery/{carrier,codec,ledger}.go; common/testing/testpilot/temporal/{driver,profile}.go; common/testing/testpilot/internal/execution/{prepare,dataflow,carrier}.go.
**Optional:** common/testing/testpilot/temporal/worker/sdk_test.go; common/testing/testpilot/temporal/worker/runtime_fixture_test.go.

### Quick commands
`mise exec -- go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; scoped Go lint for changed runtime/admission packages.

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
