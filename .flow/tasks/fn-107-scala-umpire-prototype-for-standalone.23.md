---
satisfies: [R8]
---
# fn-107-scala-umpire-prototype-for-standalone.23 Realize one model-permitted activity fault against the live server

## Description
**Touches:** [chasm/lib/activity/**, common/testing/testhooks/**, common/testing/testpilot/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/lifter/**, model/scalav2/ir/**, model/scalav2/cases/**, model/scalav2/goir/**, model/scalav2/SEMANTICS.md, model/scalav2/README.md, tests/testcore/testpilot/**, tests/testpilot_scala*_test.go, tools/canary/preflight/**]

Close the remaining live-fault clause of fn-107 R8. Task .10's reviewed scope records delivery hold/release, but explicitly does not realize a fault permitted by the Model's budget (redelivery, failed commit, or lost acknowledgment). Its offline played-Driver checks are not evidence of a server fault. This task owns that gap; it does not reopen .10's completed held-race work.

**Size:** M

### Approach
- Choose one existing model-permitted fault with a narrow, correlated server injection point after inspecting the implementation. Prefer loss of an acknowledgment after an independently observed admission commit if the existing response hook can establish both facts; otherwise use an equally precise modeled redelivery or failed commit.
- Declare the fault decision, its budget, evidence and expected assessment in Scala. Extend the existing IR, Testpilot instruction and Profile vocabulary only for the missing generic primitive. Server injection remains test_dep-only and scoped to resources owned by one Run.
- Generate the Case and run it through .22's generic runner. No per-Query Go scenario or feature-policy branch is introduced.
- A successful fault instruction records exactly one realized FAULT_INJECTED event. Refused, cancelled or uncompleted injection records none. Distinguish the actual durable outcome from a dropped response; never infer a rejected admission from an obsolete retry after an unknown outcome.
- Check the property with and without the sufficient durable evidence, pre-I/O refusal on an incapable consumer, live/offline agreement, cleanup and concurrent isolation. Preserve proved violations across later operational failure.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep ./chasm/lib/activity/... ./common/testing/testhooks/... ./common/testing/testpilot/... ./model/scalav2/... ./tests/testcore/testpilot/...`; `GOFLAGS=-tags=test_dep make umpire-gen-scala`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `cd tests && CC=/usr/bin/clang mise exec -- go test -tags 'test_dep integration canary_harness' -run '^TestTestpilotScala' -count=1 .`; `make lint-code-fast`.

## Acceptance
- [ ] At least one fault permitted by a Scala Model's declared budget is actually realized against the in-process Temporal server; the generated Case and recorded Run identify the same fault and its causal operation/attempt/delivery.
- [ ] The generated runner verifies the declared conformance and property expectations live and offline, without a Query-specific Go branch.
- [ ] Removing the evidence needed to distinguish commit from lost response leaves the affected property inconclusive, and an incapable consumer rejects the required control before target I/O.
- [ ] A refused, cancelled or unrealized injection produces no success fault event; concurrent Runs use independent resources and cannot inject into each other; cleanup releases all injected controls.
- [ ] The relevant unit, generated fixture, live and lint gates pass. The receipt identifies the realized fault, injection cut, observed durable outcome, and remaining support limits.


## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: