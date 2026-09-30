---
satisfies: [R4, R6, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.10 Add activity race controls and durable observations through Testpilot

Touches: [chasm/lib/activity/**, common/testing/testpilot/temporal/**, tests/testpilot_scala_activity*_test.go]

## Description
Realize the local admission-race/fault scenario using Testpilot controls and committed observations. Keep hooks narrow and correlated.

**Size:** M
**Files:** selected CHASM activity commit/control hooks; Testpilot driver binding and capability tests; local activity scenario integration test.

### Approach
- Trace matching enqueue to authoritative activity admission before choosing the hold-delivery point. Reuse an existing control when sufficient.
- Emit attempted-admission outcome only after the actual durable update commits, correlated with activity/attempt and causal pause. Transition-function entry is insufficient.
- Add generic profile/Driver control or observation plumbing only for primitives the existing abstraction lacks. Feature mapping and fault permission remain Scala declarations.
- Run local race/fault cases, record realized decisions, remove commit evidence for the ambiguity control, and exercise canary rejection before I/O. Do not invent a server defect if the actual implementation conforms.

### Investigation targets
**Required:** chasm/lib/activity/tasks.go:75; chasm/lib/activity/activity.go:225; chasm/lib/activity/statemachine.go:423; chasm/lib/activity/handler.go:406; common/testing/testpilot/driver.go:98.
**Optional:** tests/nexus_matching_test.go:37; common/testing/testpilot/internal/execution/fault_test.go.

### Quick commands
`mise exec -- go test -tags test_dep ./chasm/lib/activity/... ./common/testing/testpilot/...`; run the focused local scenario with the configured integration harness.

## Acceptance
- [ ] The control holds the declared delivery/admission cut and the observation establishes durable outcome with correct causal identity.
- [ ] The local scenario evaluates actual conformance without asserting an unobserved server bug.
- [ ] Removing sufficient internal evidence yields the predicted inconclusive property result; the canary rejects the actuator.
- [ ] One permitted fault is realized/recorded and its live/offline assessments agree.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
