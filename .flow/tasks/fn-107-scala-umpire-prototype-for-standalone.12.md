---
satisfies: [R6, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.12 Expose a producer-neutral prepared Testpilot assessment seam

Touches: [common/testing/testpilot/prepare.go, common/testing/testpilot/prepared_case.go, common/testing/testpilot/assessment*.go, common/testing/testpilot/internal/execution/contracts.go, proto/internal/temporal/server/api/testpilot/v1/run.proto, api/testpilot/v1/**]

## Description
Create the optional public prepared assessment boundary from the spec's Architecture & Data Models. The Scala IR adapter consumes it in task 7; generic Testpilot never imports that adapter.

**Size:** M
**Files:** assessment facade and tests; prepare.go/prepared_case.go; internal execution/contracts.go only for monitor composition plumbing; generic Run result carrier and generated output where required.

### Approach
- Define a caller-supplied immutable prepared factory with Case/model/query identity and explicit evaluation ceilings. It creates a fresh passive assessor for each execution or replay. Reject typed nil, mismatched identities, invalid limits, and state aliasing before effects.
- Compose it with the existing Contract monitor for live events through public neutral data types. It cannot access scheduling, Slots, Driver effects, or replace the existing Contract Verdict.
- Run records the supplemental conformance/property assessment; Evaluate creates fresh state from the same factory and feeds cloned ordered events, incomplete flags, evaluation-failure sequence, and closure facts. Bind persisted results to the model/query identity and reject foreign replay bindings.
- Retain existing Prepare/Run/Evaluate source compatibility and no-assessor behavior. Ensure any wire evolution regenerates existing fixtures through current protocol gates.

### Investigation targets
**Required:** common/testing/testpilot/prepare.go:15; common/testing/testpilot/prepared_case.go; common/testing/testpilot/internal/execution/contracts.go; proto/internal/temporal/server/api/testpilot/v1/run.proto:9; common/testing/testpilot/facade_external_test.go.
**Optional:** common/testing/testpilot/internal/verification/evaluator_failure_test.go; common/testing/testpilot/internal/execution/evidence.go.

### Quick commands
`mise exec -- go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; run required protocol generation and existing focused conformance gates if the wire carrier changes.

## Acceptance
- [ ] An external-package fixture supplies a bound passive assessment factory without importing internal packages or adding a model dependency to Testpilot.
- [ ] Repeated/concurrent Run and Evaluate create independent state and yield distinct conformance/property assessments while preserving Contract Verdict behavior.
- [ ] Nil/mismatched factories, foreign replay bindings, invalid ceilings, and assessment failure have explicit preparation/evaluation results.
- [ ] Existing no-assessor consumers and generated protocol fixtures retain their supported behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
