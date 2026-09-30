---
satisfies: [R3, R6, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.7 Connect partial-evidence conformance to Testpilot evaluation

Touches: [model/scalav2/goir/conformance/**]

## Description
Implement the generic IR conformance adapter and connect it to Testpilot's existing evidence/evaluation boundary, without replacing execution.

**Size:** M
**Files:** proposed goir/conformance adapter, candidates, and focused evidence/replay fixtures. The neutral Testpilot assessment facade lands in its prerequisite task.

### Approach
- Consume immutable Testpilot Run events/declared Observations and retain compatible model/monitor states under causal order and hidden steps.
- Bound candidate exploration explicitly and keep conformance separate from each property verdict.
- Implement the spec's public producer-neutral prepared assessment interface using only exported Testpilot types. Snapshot admitted IR/query data and bind Case/model/query identities and candidate ceilings. New creates independent state per run/replay. Use the neutral facade's Run/Evaluate composition rather than a second offline evaluator or an import of internal packages.
- Test the same bound adapter on live events and recorded events, including incomplete flags, evaluation-failure sequence, and closure facts. Assert separate conformance/property assessments without overwriting the existing Contract Verdict or obtaining Driver/scheduler/slot access. Reject foreign model/query replay identity before evaluation.
- Add crossed correlation, missing admission commit, visible mismatch, hole, operational failure, and skew fixtures. Explicitly pair live/offline timeout-after-commit and lost-acknowledgment fixtures with sufficient commit evidence. Pin conformance and property verdicts independently; pair each with removed commit evidence to test ambiguity rather than infer absent effects from a transport timeout.

### Investigation targets
**Required:** common/testing/testpilot/prepare.go:15; common/testing/testpilot/prepared_case.go; common/testing/testpilot/internal/execution/contracts.go; proto/internal/temporal/server/api/testpilot/v1/run.proto:9; model/scalav2/goir/machine.go.
**Optional:** common/testing/testpilot/conformance_test.go; common/testing/testpilot/internal/verification/evaluator_failure_test.go.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./common/testing/testpilot/...`.

## Acceptance
- [ ] A trace can conform while a history-sensitive property remains inconclusive.
- [ ] Missing/crossed evidence and reachable holes preserve the declared conclusion limits.
- [ ] Live/offline assessments agree, including violation-before-failure, candidate limits, timeout-after-commit, and lost acknowledgment; conformance/property conclusions are pinned separately with and without commit evidence.
- [ ] Timestamp skew with unchanged causal/model timing facts preserves verdicts.
- [ ] The optional public assessment factory creates fresh bound state in Run/Evaluate without internal imports or a Testpilot-to-Scala dependency; existing callers retain Contract behavior, and mismatched model/query replay identities reject.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
