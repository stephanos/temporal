---
satisfies: [R3, R5]
---
# fn-129-activity-coverage.3 Reset: deferred reset with keepPaused

## Description

Implement the independent reset/control rows and truthful delivery identity after .2. Preserve ordinary fatal/retry obligations. This task commits focused source/native bridge proof; shared production generation and live evidence remain .5.

**Size:** L by file count, bounded to the reset/control vertical slice under the owner's fixed five-task decomposition.
**Files:** standalone reset state/control/Properties/realization and co-located pins; scoped capability override and refinement pins; delivery ledger, reservation admission, typed identity/publication bridge and lowering; necessary internal-proto native mirrors; owning protocol docs.
**Touches:** [model/temporal/features/activity/standalone/**, model/temporal/realize/Realize.scala, model/temporal/realize/Kit.scala, model/temporal/realize/Modules.scala, model/temporal/realize/Behavior.scala, model/umpire/realize/Scripts.scala, model/irgen/Realizations.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/realizationRefusals/**, proto/internal/temporal/server/api/umpire/v1/ir.proto, proto/internal/temporal/server/api/testpilot/v1/program.proto, proto/internal/temporal/server/api/testpilot/v1/instruction.proto, proto/internal/temporal/server/api/testpilot/v1/run.proto, api/umpire/v1/ir*.pb.go, api/testpilot/v1/program*.pb.go, api/testpilot/v1/instruction*.pb.go, api/testpilot/v1/run*.pb.go, common/testing/testpilot/contract.go, common/testing/testpilot/contract/profile.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/temporal/profile.go, common/testing/testpilot/temporal/profile_test.go, common/testing/testpilot/temporal/internal/delivery/activity.go, common/testing/testpilot/temporal/internal/delivery/activity_test.go, common/testing/testpilot/temporal/worker/interpreter.go, common/testing/testpilot/temporal/worker/sdk.go, common/testing/testpilot/temporal/worker/*activity*test.go, tools/umpire/realization/**, tools/umpire/lower/**, model/README.md, model/SEMANTICS.md, common/testing/testpilot/README.md, common/testing/testpilot/temporal/README.md, common/testing/testpilot/temporal/worker/README.md]

## Approach

- Author per-RPC reset with `keepPaused`, pending-reset state and retained option. Direct waiting/paused reset clears retry state and chooses waiting/paused. Held reset is deferred; completion wins, while eligible failure or per-attempt timeout applies reset before fatal/exhausted settlement. Schedule-to-close remains terminal. Preserve Cancel > Reset > Pause and cancellation stickiness. Distinct repeated reset/conflicting control/terminal requests reject unchanged; distinguish request-id retry deduplication from a new reset. Cite comparator/server behavior as comparison evidence, not the source of the independent Model.
- Keep `nonRetryableFails`, its ordinary Scenario, pinned FIND Query and fatal SATISFIED expectation exactly unchanged. That FIND checks its authored no-reset path, not universal reset coverage. Add a separate independent reset-settlement `holdsAcross` Property with complete reset/no-reset branches, plus reset-fatal, reset-exhausted, timer, keepPaused and precedence witnesses. The independent predicate must not invoke the Effect it checks.
- Use Activity-local `capabilities.overriding` for the generated universal Retry failure-return/failure-terminal/failure-pause and Deadline per-attempt timeout laws that reset changes. Preserve exact companion signatures, original Query identities/bounds and unaffected cancellation/count/window laws. Each replacement states full reset/no-reset transitions, including fatal/exhausted policy and keepPaused; no `except`, vacuous exclusion, shared companion edit or blanket waiver. Inspect emitted overriding metadata and test both branches. Re-anchor Product, Fact visibility and all four existing Realizes declarations; recompute finite state totals/Limits without inventing a reset epoch.
- Reuse the existing typed `ResetActivityExecution` unary with true learned execution run ID. Retain raw Describe scheduling count, actual SDK Attempt (including rewind to 1), independent per-execution Model count (waiting reset is zero, poll increments) and monotonically assigned reservation/delivery ordinal as distinct typed values. Replace `SDKAttempt - 1` ledger selection and scalar equality admission with explicit declared reservation identity bound to the new token/delivery. Duplicate identical delivery replays its settled answer; fresh delivery with rewound SDK Attempt consumes the next group. MaximumAttempts.one must permit reset's fresh attempt without relabeling the SDK value.
- Before dispatching a deferred controller reset, prove .1's local pending outcome has been chronologically published through .2's proven bounded typed publication handoff. Existing `AwaitCommand` is workflow-only and is not that barrier. Reuse the existing Program/Profile/VM outcome/slot contract and bounds; extend only reset-specific typed identity if required. Server Describe alone proves holding, not that local record publication already happened. This plan's deferred live witness retains a positive request-field per-attempt timeout basis and one selected timer occurrence after reset; reset requests defer control, not external settlement. On that timeout basis, no selected timer, unknown basis and early/late publication refuse. Separately .2's valid typed external-control basis admits its timer-free ByID paths; do not globally reject no-timer or implicitly authorize external control under R1's timeout declaration. A reset RPC is not an activity answer; lower the pending record before the reset without moving late evidence backward. First pin reset handoff and fresh-SDK-1 delivery in native recording/replay fixtures; if this narrow contract cannot be proved, return the located blocker before adding unsupported live realization.
- Author direct keepPaused and deferred reset settlement Queries/Cases, including pending reset followed by HEARTBEAT/start-to-close timeout and fresh SDK Attempt 1 completion. Keep task .1's truth about receipt/type and no first-attempt answer RPC. Separate Model-only reset-fatal/exhausted and Cancel > Reset > Pause checks from live claims until the exact Cases execute at .5. Scratch lift/lower unfiltered current source with ordered receipts, Query/Case/Program/Contract inventory and intentional delta explanation; do not rewrite sealed fn-138 evidence.
- Seed failures for reset fatality/exhaustion precedence, keepPaused landing, uncleared retry/backoff state, wrong terminal timeout Fact, schedule-to-close incorrectly resetting, stale keepPaused after ordinary settlement, raw-SDK/ordinal equality, duplicate versus fresh token and late publication. Update protocol docs. Native schema generation is only the changed closure (Go paths above and ignored `model/build/ir-scalapb.jar`/`api-scalapb.jar`); preserve unrelated generated bytes and all production Model IR/Cases/managed fixtures.

## Investigation targets

**Required** (read before coding):
- `model/temporal/features/activity/standalone/system/System.scala` (Properties and Scenarios are authored here).
- `model/temporal/features/activity/standalone/system/Realization.scala` and `Standalone.scala`.
- `model/temporal/capabilities/Retries.scala` and `model/temporal/capabilities/Deadline.scala`; `model/umpire/Capabilities.scala` override admission and law metadata.
- `chasm/lib/activity/model/model.go` and `chasm/lib/activity/activity.go` reset/settlement transitions.
- `common/testing/testpilot/temporal/internal/delivery/activity.go:132` and `common/testing/testpilot/internal/execution/scheduler.go:203`.
- `common/testing/testpilot/temporal/worker/sdk.go:281`, `tools/umpire/lower/realization.go:254` and `tools/umpire/lower/lower.go:572`.
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` AwaitSlot/AwaitInstruction and `model/temporal/realize/Kit.scala:372`.

## Quick commands

```bash
mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*Activity*' --require-tests
go test -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal/worker -run 'Activity|ReservationOutcome|Reset|Replay'
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/realization ./tools/umpire/lower -run 'Activity|Reset|Carriers|ReadsWait'
```

Serialize heavy scratch/Go commands with the shared flock. Include focused capability/override and lifter fixtures, scoped format/lint, actual selectors and nonzero counts. No production generation/full suites/live acceptance here.

## Acceptance

- [ ] Independent reset/control transition matrix, complete settlement Property and strict seeded negatives pass. Ordinary `nonRetryableFails` Property/Scenario/Query/SATISFIED expectation and all old retry reason values remain byte-equivalent in meaning; scoped universal overrides cover reset and no-reset without vacuity or shared capability edits.
- [ ] Typed reset unary, true execution run ID, actual SDK Attempt rewind, separate reservation ordinal/Model count and raw Describe count are pinned. Native fixture proves fresh token plus SDK Attempt 1 uses the next group, while duplicate same delivery replays exactly once; invalid declarations refuse before effects.
- [ ] Focused recording/replay proves bounded pending publication before controller reset and eligible deferred settlement before terminal classification, with no invented epoch, backward evidence, synthetic answer or server receipt. Direct keepPaused and deferred reset scratch Cases preserve authored expected status/reason, identities and bounded cleanup.
- [ ] Current-source unfiltered scratch lift/lower, override metadata, seeded mutations, new Query/Case inventory and original source/native commit are retained for .4 and the fn-128.7/.8 rescan. Production artifacts/full/live R5 evidence remains pending the shared .5 boundary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
