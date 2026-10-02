---
satisfies: [R4, R6, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.10 Add activity race controls and durable observations through Testpilot

Touches: [chasm/lib/activity/**, common/testing/testhooks/**, common/testing/testpilot/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/lifter/**, model/scalav2/ir/**, model/scalav2/goir/**, model/go/caseproducer/**, model/scalav2/SEMANTICS.md, model/scalav2/README.md, model/scalav2/specimens/**, model/scalav2/run.sh, tests/testcore/testpilot/**, tests/testpilot_scala_activity*_test.go, tools/canary/preflight/**, tools/umpire/**/testdata/**, tools/canary/**/testdata/**]

## Description
Realize the local admission-race/fault scenario using Testpilot controls and committed observations. Keep hooks narrow and correlated.

**Size:** M
**Files:** selected CHASM activity commit/control hooks; Testpilot driver binding and capability tests; local activity scenario integration test.

### Approach
- Trace matching enqueue to authoritative activity admission before choosing the hold-delivery point. Reuse an existing control when sufficient.
- Emit attempted-admission outcome only after the actual durable update commits, correlated with activity/attempt and causal pause. Transition-function entry is insufficient.
- Add generic profile/Driver control or observation plumbing only for primitives the existing abstraction lacks. Feature mapping and fault permission remain Scala declarations.
- Run local race/fault cases, record realized decisions, remove commit evidence for the ambiguity control, and exercise canary rejection before I/O. Do not invent a server defect if the actual implementation conforms.

### What earlier tasks handed to this one (added 2026-10-01)
- **Declarations first.** Hold-delivery, its hold and release commands, and durable-commit observation are located `unsupported` entries of the Scala activity realization today, each naming this task (`goir/testpilot`; `TestWhatTestpilotCannotRunIsNamedWithItsOwner`). The control, the commit observation and the permitted fault are declared in Scala, lifted, admitted and lowered; Go supplies only the generic primitive. The admission step of the race needs a delivery key so its two results are told apart (`evidence.kind-ambiguous` in task 6's fixture).
- **A reservation that is deliberately never dispatched.** The scheduler stops a Run when a scripted entrypoint's reservation ends other than SUCCEEDED (`internal/execution/scheduler.go`; task 13 f), so a held or paused activity that is never delivered cannot end satisfied. Add a generic outcome for a declared held or undispatched reservation; arbitrary missing execution must still fail. An earlier attempt the worker never saw stays reserved and hides a later refusal (task 13 limits).
- **Protocol.** Every Testpilot protocol file is inside the Driver catalog identity. Make one additive change, prove every checked-in Case and lowered Case keeps its identity, and re-record the pinned Runs with `make umpire-rerecord-pinned-runs` (tasks 13 and 20 did this; five records change).
- **Server hooks are test-only.** A hook in `chasm/lib/activity` goes through the repository's existing test-hook mechanism (`common/testing/testhooks`: a real implementation under the `test_dep` build tag and `noop_impl.go` otherwise), emits the admission outcome only after the durable update commits, and changes no production behaviour. The canary profile rejects the actuator before any I/O.
- **Evidence the conformance adapter can use.** Declare the commit source exhaustive with a closing read where that is true (task 19 vocabulary), so the stale-design violation and the corrected design are told apart on a live Run; removing the commit evidence must leave the admission Property inconclusive.
- Live tests of lowered Cases exist since task 9 (`tests/testpilot_scala_activity_test.go`, `tests/testcore/testpilot/scala_fixture.go`); extend them.

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
