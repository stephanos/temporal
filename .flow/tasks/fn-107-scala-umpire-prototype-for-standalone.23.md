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
Implemented the generated `admissionResponseLoss.committed` Case and generic `ADMISSION_RESPONSE_LOSS` primitive. Scala declares a one-loss budget, the committed/failed-update alternatives behind a lost answer, the durable admission evidence, and the expected satisfied assessment; the existing generic runner needs no Query-specific branch. The manifest now accounts for 261 Queries, including 15 lowered Cases.

The in-process cut replaces one successful post-handler `RecordActivityTaskStarted` response with `Unavailable`. Success is withheld until a retry with the same execution, delivery stamp and nonempty request ID proves the replacement reached the caller. The original durable admission supplies activity/run/delivery/attempt correlation; a retry's obsolete answer never becomes rejection evidence. Closing or cancelling disarms the control and stops polling. A successful loss records exactly one loss FAULT_INJECTED event, separate from its one declared hold event; refusal, cancellation and incomplete injection record no successful loss.

All required gates pass: full task unit command, Scala generation and deterministic check, final generic live suite (46.680s), default `make lint-code-fast` (0 issues), and diff check. Eight final captured loss Runs cover both implementations, two resource bindings and two concurrent rounds; the unchanged runner verifies live/offline assessment agreement, cleanup and inconclusive assessment after removing durable evidence. Unit regressions cover the response/retry/cancellation cuts, exact correlation, capability refusal, malformed outcomes and fault-event counts. Existing generic violation-precedence tests pass in the full unit suite. The protocol fingerprint changed, so both pinned live Runs and three receipt goldens were re-recorded through the established Go-only target; replay/evaluation/canary compatibility probes pass.

The second realization exposed a fixture metadata bug: durable evidence is now scoped to the selected machine, with a synthetic regression for reused evidence IDs with different commitments. Existing declaration controls were extracted unchanged into a private helper to clear the inherited complexity finding; fault-kind translation uses the existing enum-map pattern. One pre-existing held-delivery test race was fixed by checking the initial state before launching arrival.

Support remains one in-process lost admission response, confirmed through history's same-request retry. No physical task redelivery, failed-commit/persistence fault, or remote/canary actuator is claimed. Baseline: runtime green; unchanged baseline gates reused from .22; its documented inherited lint complexity is now resolved. No staging, commits, branch changes, worktrees, task/spec-state changes or MILESTONES edits were made.

Exact 35 paths and saved originals: `.flow/tmp/fn107-23/changed-paths.json`, `baseline.tar.gz`, `extra-original-paths.json`. Commands, logs and support details: `.flow/tmp/fn107-23-evidence.json`. Final recordings: `.flow/tmp/fn107-23/runs-final`; independent conductor audit: `.flow/tmp/fn107-23/live-final-artifact-audit.json`.

Independent review: gpt-6.1-sol at high reviewed the exact incremental .23 delta, refreshed recordings/receipts, fixture metadata correction and final lint helpers. Every round returned SHIP with no findings; all 35 paths reviewed. Digest: .flow/tmp/fn107-23-review.md. Native worktree review substitutes for the committed-only wrapper because the user reserves commits for the owner. Root independently inspected final test outputs and all eight final loss Run artifacts before completion.

stage: impl-review - ran (native independent-context reviewer; SHIP across initial and incremental reviews)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang mise exec -- go test -tags test_dep ./common/testing/testpilot/temporal/control ./common/testing/testpilot/temporal, CC=/usr/bin/clang mise exec -- make protoc PROTO_DIRS='testpilot/v1 modelir/v1', GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-gen-scala, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot/internal/execution, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tests/testcore/testpilot/..., CC=/usr/bin/clang mise exec -- go test -tags test_dep ./chasm/lib/activity/... ./common/testing/testhooks/... ./common/testing/testpilot/... ./model/scalav2/... ./tests/testcore/testpilot/..., CC=/usr/bin/clang make umpire-rerecord-pinned-runs, CC=/usr/bin/clang mise exec -- make lint-code-fast, GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala, cd tests && UMPIRE_REPEAT_RUN_DIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/fn107-23/runs-final CC=/usr/bin/clang mise exec -- go test -tags 'test_dep integration canary_harness' -run '^TestTestpilotScala' -count=1 ., git diff --check
- PRs: