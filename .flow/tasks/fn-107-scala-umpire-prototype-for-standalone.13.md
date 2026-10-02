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
The Testpilot Temporal Driver now runs an activity entrypoint as a standalone activity. A controller's `StartActivityExecution` carries the reservation, the SDK worker registers one dynamic activity for the queue, and the attempt runs the lowered script (`Finish`) and settles its reservation with a typed outcome. No `.proto` file and nothing under `api/` changed. Nothing is committed and the task stays `in_progress`.

Terminal status: `in_progress`, work uncommitted for the owner (`"commits": []`).

### Files

Changed (before-copies under `.flow/tmp/fn-107/task13-before/`, 19 of the 20 copied files changed; `worker/reservation.go` was copied and left untouched):
- `common/testing/testpilot/internal/execution/dataflow.go`, `prepare.go`, `carrier.go`
- `common/testing/testpilot/temporal/internal/activation/activation.go`
- `common/testing/testpilot/temporal/internal/delivery/codec.go`, `carrier.go`, `ledger.go`
- `common/testing/testpilot/temporal/worker/sdk.go`, `interpreter.go`, `driver.go`, `session.go`, `registry.go`, `routing.go`, `carrier.go`, `README.md`
- `common/testing/testpilot/temporal/driver.go`, `driver_test.go`, `profile.go`, `profile_test.go`

Added:
- `common/testing/testpilot/temporal/internal/delivery/activity.go` (111 lines)
- `common/testing/testpilot/internal/execution/activity_admission_test.go` (7 tests)
- `common/testing/testpilot/temporal/internal/delivery/activity_test.go` (11 tests)
- `common/testing/testpilot/temporal/internal/activation/activity_test.go` (1 test)
- `common/testing/testpilot/temporal/worker/activity_fixture_test.go`, `activity_test.go` (14 tests), `activity_sdk_test.go` (1 test), `registration_pin_test.go` (3 tests)

`git diff --shortstat -- common/testing/testpilot` reports 19 files, 938 insertions, 132 deletions. The eight new files add 1,894 lines. No file outside the Touches changed.

### How it works

1. **Admission (neutral, `internal/execution`).** `Finish` runs in a workflow or an activity entrypoint (`opcodeRow.activity`). Every other instruction in an activity entrypoint still rejects as `unsupported instruction context or Driver capability`. A carrier shape may name `ActivityEntrypoint`. An activity entrypoint that carries a script and that no carrier reserves rejects at `Prepare` with `unavailable at <entrypoint>: no reservation carrier of the Profile activates the activity entrypoint`.
2. **Profile (`temporal/profile.go`).** `DeriveProfile` names `StartActivityExecution` the carrier of the activity kind. `StartWorkflowExecution` keeps exactly the workflow and Nexus-handler shapes.
3. **Delivery (`temporal/internal/delivery`).** A third route kind, `activity`, rides in the start request's header `temporal-testpilot-reserved-activity-v1`. `CreateActivityBundle` holds the one activity reservation. `AdmitActivity` consumes it once. `PinStartResponse` takes the run of either start through `StartResponse` (`GetRunId`).
4. **Worker (`temporal/worker`).** A queue registration gains `activities`. `register` calls `RegisterDynamicActivity` only when the queue names an activity type. `InterceptActivity` reads the SDK info and header and calls `activateActivity`, which checks the allowlist, admits against the reservation, binds the reservation's cancellation to the attempt's context, runs `executeActivity`, and settles the reservation.
5. **Composite (`temporal/driver.go`).** `InvokeRPC` makes an activity carrier when the plan's method is `StartActivityExecution`, and `startResponse` decodes the response of whichever start it carried.

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | The SDK adapter executes the lowered activity script and reports typed completion and failure with correlated activation identity | pass | `TestSDKActivityInterpretsItsScriptUnderTheCarriedRoute` (through the SDK's own activity test environment and interceptor chain), `TestActivityActivationRunsItsScriptUnderItsReservation` (result `done`, coordinate `{run, activity, reservation-1, 1}`, outcome SUCCEEDED), `TestActivityActivationReportsTypedFailures` (5 cases, whole outcomes), `TestRunRecordsAnActivityActivationUnderItsReservation` (the Run records the outcome under `scheduler.g0.n0.a1.r0.i0` with the carrying instruction's coordinates) |
| 2 | Missing registration or capability, invalid script input, cancellation and cleanup follow existing bounded semantics | pass | `TestPrepareRejectsAnActivityItCannotActivate` (9 cases, whole `ir.Error`), `TestPrepareRejectsAnActivityScriptUnderAProfileWithoutItsCarrier`, `TestDriverValidatesActivityCarriersBeforeOpen` (8 rejections, zero registry acquisitions), `TestActivityDeliveryIsRefusedBeforeItsScriptRuns` (8 cases, reservation unconsumed), `TestActivityScriptWithoutAResultFailsItsActivation`, `TestSessionCloseCancelsAnActivityAttemptUnderWay`, `TestActivityRoutesAreBoundedAndReleased`, `TestInvalidActivityDeliveriesRejectBeforeReservationConsumption` (14 cases) |
| 3 | Repeat and concurrent activation state stays isolated through the reservation boundary | pass | `TestConcurrentRunsAdmitTheirOwnActivityDeliveries` (reordered delivery, and two Runs that start the same physical binding), `TestConcurrentRunsRouteActivitiesByIdentity`, `TestActivityRedeliveryDoesNotRunTheScriptTwice`, `TestAdmitActivityConsumesItsReservationOnce`, `TestActivityRegistrationPoolsOnlyWithTheSameActivities` |
| 4 | Existing workflow and Nexus SDK fixtures remain unchanged and worker gates pass | pass | `registration_pin_test.go` (3 tests, written and green before the registration was extended, log `pin-01-worker-before-extension.log`), the unedited `TestRouteCodecWireBytesGolden`, `TestDeriveProfileNamesEachStartTheCarrierOfWhatItActivates/the_workflow_alone,_as_before`, and every pre-existing test in the gate |

### Decisions that differ from the task text

1. **The script vocabulary is `Finish` alone.** Task 6 lowers an activity script to `Finish`, and the protocol has no other instruction an activity could run. `Finish` now ends an activity attempt as it ends a workflow. The comment on `message Finish` in `instruction.proto:101` still says "workflow"; I could not edit it.
2. **One reservation admits one attempt.** A second delivery on an admitted route, the same task or the server's next attempt, returns `ErrRouteConflict`, and every refusal or failure answers the SDK with a non-retryable `umpire_worker` application error. The ledger keeps no SDK attempt number, so the coordinate's attempt is always the carrying instruction's.
3. **Only an activity needs its carrier to prepare.** A scripted workflow or Nexus entrypoint that no carrier reserves still prepares, as before (`TestPrepareStillAdmitsAnUncarriedWorkflowScript`). An activity entrypoint with no script and no carrier also prepares, because the pre-existing `TestPrepareDerivesReservationsFromTheProfileCarriers` pins that.
4. **Cancellation is local.** Canceling the reservation cancels the attempt's Go context. The worker sends no `RequestCancelActivityExecution` to the server, so an activity cleanup cannot fail on target I/O.
5. **`PinStartResponse` takes an interface.** The parameter type changed from `*workflowservice.StartWorkflowExecutionResponse` to `delivery.StartResponse` in the ledger, the worker `Carrier` and the composite's `terminalCarrier`. Existing callers compile unchanged. The fake `recordingTerminalCarrier` in `temporal/driver_test.go` changed its parameter type and one field to match (5 lines).
6. **`workflowError` is renamed `activationError`** in `worker/sdk.go` (4 call sites, same body), because the activity path returns the same error.
7. **The preparation inventory is derived from code.** `.flow/tmp/fn-107/activity-sdk-prep/preparation.md` does not exist; `specimens/README.md:145` and `specimens/activity.md:525` name the gap.
8. **A workflow-scheduled activity is not realized.** Its task names no activity run and the ledger refuses it (`ErrInvalid`), covered by the SDK test.

### Test-first record

- Execution: `red-01-execution.log`. `TestPrepareAdmitsAnActivityScript` failed at `activity_admission_test.go:42` with `unsupported at policy.reservation_carriers: carrier shape has an unsupported activation context`; all 4 tests written at that point failed, 8 of the 9 rejection cases among them.
- Delivery: `red-02-delivery.log`. The package did not build (`undefined: ActivityBinding`, `ledger.CreateActivityBundle undefined`); the API did not exist.
- Activation: `red-03-activation.log`. `TestActivityEntrypointActivates` failed with `activation requires a workflow or Nexus-handler entrypoint`.
- Worker: `red-04-worker.log`. The package did not build (`session.CreateActivityCarrier undefined`, `host.dynamicActivity undefined`).
- Profile: `red-05-profile.log`. `TestDeriveProfileNamesEachStartTheCarrierOfWhatItActivates/a_workflow_and_a_standalone_activity` failed because the derived Profile had no `StartActivityExecution` carrier.
- Written after the code and never red: the composite tests in `driver_test.go`, `TestFinishStillRunsOnlyWhereAScriptEnds`, `TestPrepareStillAdmitsAnUncarriedWorkflowScript`, `TestDefinitionRejectsTwoScriptsForOneActivityType`, `TestActivityAttemptAnswersWithItsFirstEnabledFinish`, and the cases added for surviving mutants. Each is covered by a mutant below.
- Mutation, through `go test -overlay` with the mutated file in the session scratchpad, so no repository file was rewritten (`mutants-*.log`, script `mutants-run.py`): 74 effective mutants plus one that became a literal no-op once its target code was removed (`admit-no-kind`). 14 survived the first run and 5 more hung a test. I added or tightened tests for 8 survivors, removed the code for 2 that were redundant (a route-kind check in `AdmitActivity`, a context check in `executeActivity`), and 4 effective mutants still survive (`mutants-final-survivors.log` has a fifth line, the no-op): `admit-no-stopped`, `driver-activity-queue-ambiguous-ok` and `interp-wrong-kind` are equivalent because another check rejects the same input, and `interp-no-admit` is unobservable because the `Admit` of a terminal `Finish` only charges work, as in the workflow interpreter. The 5 hangs came from waits in my new tests, which now fail within 5 seconds instead.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (baseline, pre-edit) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (baseline, pre-edit) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (13 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race ./common/testing/testpilot/...` (12 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/internal/activation ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false` (0 issues; the first run reported 4 in my new tests, fixed) | 0 |
| `CC=/usr/bin/clang mise exec -- go build ./tools/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./tools/umpire/... ./tools/canary/...` | 0 |

Not run:
- `go test ./tools/umpire/...` and `./tools/canary/...`. Both import Testpilot. The conductor forbade them mid-task because `lake` downloads a Lean toolchain. They build and vet.
- `make umpire-check-testpilot-protocol` and `make umpire-check-testpilot-authoring` (they call `lake`). No protocol file changed.
- `make umpire-check-scala`, `make umpire-gen-scala`, `make lint-scala`. No Scala or IR file changed; the Go tests under `model/scalav2` ran in the model gate.
- No `integration`-tagged or live-cluster test. The activity path has not run against a server.
- No `flowctl gate` receipt: the tree is dirty outside the ignore set.

### Findings

1. **Two pre-existing tests pass for a reason other than their name.** `TestPrepareRejectsReservationCarrierPolicyErrors/unsupported_context` sets an activity shape, which is now admitted, and still errors because the Nexus handler loses its workflow. Every case of `TestInstructionContextMatrix` errors on its unbound worker role before any instruction is read, so it never tested contexts. I left both unedited and added exact tests beside them.
2. **The SDK worker already polls activity tasks.** SDK 1.48 starts the activity worker whether or not an activity is registered (`internal_worker.go:1463`), so a workflow-only worker already fails a foreign activity task as an unknown type. A queue with a registered dynamic activity now fails an unrouted task with a non-retryable error instead.
3. **A scenario whose activity is never dispatched cannot end satisfied.** The scheduler stops the Run as `activation_failed` when the reservation of an entrypoint that performs something ends other than SUCCEEDED (`scheduler.go:779-781`). The specimen's pause race (E8, E10) holds the dispatch, so its activity script would never run. `scheduler.go` is outside the Touches.

### Gaps outside the Touches

- **No instruction makes an activity attempt fail, heartbeat or wait.** `Finish` has only `result` (`instruction.proto:102`). A scripted failure, and therefore a retried attempt (R7's retry translation), needs a protocol addition such as a failure arm, plus a reservation per attempt.
- **`model/scalav2/goir/testpilot/lower.go:233-235`** still reports the activity script as an `unsupported` gap owned by fn-107.13. Lifting it is task 9's. For the lowered Case to validate, its start command must assign `namespace` and `task_queue.name` from the environment bindings of the worker and queue roles, and set `activity_id` and `activity_type.name`; the task-6 fixture's start assigns only `namespace` and `activity_id`.
- **Stale documentation I could not edit:** the `Finish` and `ActivityActivation` comments in `instruction.proto` and `program.proto`, `common/testing/testpilot/temporal/README.md` (carriers are described as `StartWorkflowExecution` only), `common/testing/testpilot/internal/execution/README.md`, and `model/scalav2/specimens/README.md:145`.
- **`tests/testcore/testpilot` has no activity Case**, so the functional and canary consumers (task 9) are the first live run.

### Review round 1

The Codex review returned NEEDS_WORK with two introduced P2 findings. Both are valid and fixed, test-first. Where this section and the sections above disagree, this section is current: decisions 1, 2 and 4 above are replaced, and the acceptance table's test names for the single-attempt behaviour are replaced by the tests named here.

**Blast radius, measured.** All nine Testpilot protocol files are inside the Driver catalog closure, `instruction.proto` and `program.proto` included (`r1-catalog-before.log` lists them), and the catalog identity is the SHA-256 of that whole descriptor set (`internal/ir/catalog.go:236-242`). A new instruction arm or outcome field would therefore stale every pinned Run. Re-recording here would not last either: task 12 changes `run.proto` in a separate copy, so the merged identity would differ again. I made no structural protocol change. The catalog identity is `1f757edd4c18a3f52fcb0205bee2ea3da389517b6048bc79e49a136ab3ebdb57` before and after. The guarded `./tools/umpire/... ./tools/canary/...` run has the same result set before and after: 33 packages ok, and one package failing only in the three tests that need `lake`. No pinned Run is stale and nothing was re-recorded. Existing Cases keep their identities because no Case byte, descriptor or workflow or Nexus admission path changed; the pinned identity tests in `tests/testcore/testpilot`, `tools/umpire`, `tools/canary` and `model/scalav2/goir/testpilot` pass unchanged.

1. **P2, operation, attempt and delivery were one identity (valid).**
   - **The failure arm is the `Finish` result.** A `Finish` of an activity entrypoint whose result is a literal `temporal.api.failure.v1.Failure` fails its attempt with that failure, retryable unless its application failure info says otherwise. Any other result completes the activity. Preparation admits only a failure with application failure info or none, and names the field of any other (`bindAttemptFailure` in `internal/execution/dataflow.go`). A workflow `Finish` is unchanged. This uses a message already in the catalog, so it adds no protocol field.
   - **The script declares the attempts.** An activity entrypoint's instructions are its attempts in order. The carrier reserves one activation per instruction (`internal/execution/carrier.go`), `DeriveProfile` sizes the carrier shape to match, and the worker carries the script's values across attempts as the Nexus handler path does, so a later attempt's default guard reads the earlier one's outcome.
   - **Each attempt is admitted under its own reservation.** `delivery.ActivityDelivery` carries the SDK attempt and a delivery identity, the SHA-256 digest of the task token. `Ledger.AdmitActivity` admits each new attempt against the next reservation in arrival order and records run, attempt and delivery in the activation. Attempt numbers only have to rise.
   - **A delivery of an admitted attempt runs nothing.** It is a replay under whatever delivery identity, and the worker returns the admitted delivery's recorded answer after waiting for it (`activityAnswer`). An attempt past the script's last is answered as the last was, so a server that never saw a completion and retries is told the same completion. An attempt older than one never admitted, or of another run, conflicts.
   - **Declared failures go to the SDK as written.** The activation of an attempt that did what its script said settles SUCCEEDED, as a Nexus handler's instructed error does.
   - **The identities are recorded in the outcome detail**: `activity_run="<run>" sdk_attempt=<n> delivery=<digest>`. The reservation event's source carries the attempt's ordinal (`...r0.i0`, `...r0.i1`) and its coordinates the carrying instruction. A typed field would belong in `InstructionOutcome`, which is `run.proto`: task 12's file and inside the catalog closure.
   - Red first: `r1-red-01-delivery.log` (the package did not build: `unknown field Attempt`, `activation.Attempt undefined`), `r1-red-02-execution.log` (`TestPrepareReservesOneActivationPerDeclaredAttempt`: "An error is expected but got nil"; three failure-admission cases; three Run-record cases), `r1-red-03-worker.log` (10 tests, for example the retry test failed with `invalid Temporal SDK worker Driver input`), `r1-red-04-profile.log`.
   - Proving tests:
     - First attempt fails, retry succeeds: `TestActivityAttemptFailsAsItsScriptSaysAndItsRetryPerformsTheNextInstruction` (retryable and non-retryable), `TestTransportTemporalIsToldWhatEachDeclaredAttemptDoes` (a real SDK worker; Temporal receives the script's failure field for field, then the completion).
     - Duplicate delivery: `TestDuplicateDeliveryOfAnAttemptIsAnsweredOnceRun`, `TestRedeliveryWithoutARecordedAnswerIsRefused`, and the redelivery step of the transport test.
     - Concurrent delivery race: `TestConcurrentDeliveriesOfOneAttemptRunItsScriptOnce` (worker), `TestConcurrentDeliveriesOfOneAttemptAdmitItOnce` (ledger, 8 racers), both under `-race`.
     - Identities distinct and correlated: `TestAdmitActivityAdmitsEachAttemptUnderItsOwnReservation` (7 steps, whole admission values), `TestActivityOperationOutlivesItsSettledAttempts`, `TestRunRecordsEachActivityAttemptUnderItsReservation` (two recorded events, sources `i0` and `i1`), `TestSDKActivityInterpretsItsScriptUnderTheCarriedRoute` (the digest of the SDK's own task token).
     - Admission: `TestPrepareReservesOneActivationPerDeclaredAttempt`, `TestPrepareAdmitsOnlyAnApplicationFailureAsAnAttemptFailure` (6 cases), `TestWorkflowFinishStillCompletesWithAFailureValue`, the retried case of `TestDeriveProfileNamesEachStartTheCarrierOfWhatItActivates`.

2. **P2, a canceled attempt was recorded CANCELED while Temporal was told failed (valid).** I chose to record what the worker reports rather than send `RequestCancelActivityExecution`. Sending it needs a client call, a wait for the server's answer and a heartbeat for the worker to learn of it, none of which the script vocabulary has; recording the report needs no new machinery and is true by construction. The worker answers every attempt it refuses or fails itself, a cancellation by the Run included, with the non-retryable application failure `umpire_worker`, and that is now the recorded outcome: status SDK_FAILURE, code `umpire_worker`, and the cause after the identity in the detail. The Run no longer says CANCELED or TIMED_OUT for an activity attempt Temporal was told had failed.
   - Red first: `TestActivityAttemptTheDriverFailsIsRecordedAsReportedToTemporal` failed in all four cases in `r1-red-03-worker.log`.
   - Transport level: `TestTransportACanceledAttemptIsReportedAndRecordedAsFailed` runs a real SDK client and worker against an in-process `WorkflowService`. The Run cancels the reservation while the attempt is under way; the server receives `RespondActivityTaskFailed` with `umpire_worker`, non-retryable, cause `context canceled`, and receives no `RespondActivityTaskCanceled`; the reservation records the same. That test was written after the fix; the overlay mutant `sdk-records-cause-class`, which restores the old recording, fails `TestActivityAttemptTheDriverFailsIsRecordedAsReportedToTemporal`.

3. **Mutation count (valid).** Corrected above: round 0 had 74 effective mutants and 4 effective survivors.

Round-1 mutation used `go test -overlay` again, so no repository file was rewritten and there is nothing to restore (`r1-mutants-*.log`, script `r1-mutants-run.py`): 49 mutants. 5 survived the first run and 1 did not build until I rewrote it; I added tests for 4 survivors, and 1 survives as equivalent (`driver-workflow-any-count`: preparation only ever reserves one activation of a workflow).

Files this round (round-0 state under `.flow/tmp/fn-107/task13-before/r1/`, first-touch copies under `task13-before/`):
- Changed: `internal/execution/{carrier.go,dataflow.go,README.md,activity_admission_test.go}`, `temporal/internal/delivery/{activity.go,ledger.go,activity_test.go}`, `temporal/worker/{sdk.go,interpreter.go,routing.go,session.go,driver.go,README.md,activity_test.go,activity_fixture_test.go,activity_sdk_test.go}`, `temporal/{profile.go,profile_test.go,README.md}`.
- Protocol comments only: `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` (`Finish`) and `program.proto` (`ActivityActivation`), regenerated with `make protoc`. Of 140 files under `api/`, only `api/testpilot/v1/instruction.pb.go` and `program.pb.go` changed, in comment lines only (`r1-api-files-before.sha`, `r1-api-files-after.sha`).
- Added: `temporal/worker/activity_transport_test.go`.

Gates after the fixes:

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (13 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race ./common/testing/testpilot/...` (12 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./api/testpilot/...` | 0 |
| scoped `make lint-code ... GOLANGCI_LINT_FIX=false` on the five changed packages (0 issues; the first run reported 2, fixed) | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make protoc` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...`, before and after | 1 both times |

The guarded tools run exits 1 both times for the same reason. Unrunnable set, all three failing on the stub's "lake is not available" message, all in `tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog`: `TestRenderedCatalogElaboratesSignedLiteralBoundaries`, `TestRenderedCatalogElaboratesLeanEscapedStrings`, `TestRenderedProductionCatalogIsStableAndLeanElaborates`. Every other package passes, the two pinned-Run probes included.

Not run: the two `lake` protocol targets, the Scala make targets (no Scala or IR file changed), `make umpire-rerecord-pinned-runs` (nothing is stale), and any `integration`-tagged test. The activity path still has not run against a real server.

Findings this round:
- **A pre-existing data race in a test fake.** `-race` over `./tests/testcore/testpilot/...` fails `TestWorkflowStartCaseSequentialAndConcurrentRunIsolation`: `workflowStartDriver.Open` writes `d.runID` from concurrent Runs (`workflow_start_artifact_test.go:356`). I did not touch that file and it is outside my Touches (`r1-race-with-testcore.log`). Without `-race` the package passes.
- **The SDK takes the activity's namespace from the task's `workflow_namespace`.** The server sets it for a standalone activity (`chasm/lib/activity/activity.go:267`); the transport test does the same.

Still open:
- No instruction makes an attempt heartbeat or wait.
- Extra attempts beyond the script consume no reservation, so the Run records only the declared attempts; server evidence shows the rest.
- A script whose guards disable an instruction leaves that attempt's reservation unused, and the scheduler then stops the Run when it is released canceled. The review assigned that scheduler rule to task 10.
- Task 9 lowers the failing `Finish` and lifts the activity gap in `model/scalav2/goir/testpilot/lower.go`. The lowered start must assign `namespace` and `task_queue.name` from the worker and queue role bindings, and set `activity_id` and `activity_type.name`.

### Review round 2

The protocol now carries the activity attempt explicitly, in one additive change. Five findings are fixed test-first, the two pinned Runs and three receipt goldens are re-recorded, and every gate is back at its baseline.

This section supersedes the earlier sections where they differ. The round-1 convention of a `Finish` with a `Failure` result, the identity text in `detail`, and the replay of an attempt past the script no longer exist.

### Protocol diff

- `instruction.proto` gains the oneof arm `ActivityAttemptFailure activity_attempt_failure = 10` and the message `ActivityAttemptFailure { temporal.api.failure.v1.Failure failure = 1; }`. The `Finish` comment now says a result is a value whatever its type.
- `run.proto` gains `ActivityAttempt activity_attempt = 6` on `InstructionOutcome`, the message `ActivityAttempt { activity_run_id = 1; sdk_attempt = 2; delivery_id = 3; response = 4; }` and the enum `ActivityAttemptResponse` with `COMPLETED`, `FAILED_RETRYABLE`, `FAILED_NON_RETRYABLE`, `REFUSED` and `NOT_NEEDED`.
- `program.proto` changes one comment on `ActivityActivation`.
- `make protoc` under the guard exits 0 and regenerates five files, all under `api/testpilot/v1` (`instruction.pb.go`, `instruction.go-helpers.pb.go`, `run.pb.go`, `run.go-helpers.pb.go`, `program.pb.go`). The checksum listing of all 140 files under `api/` shows no other change (`r2-api-files-before.sha`, `r2-api-files-after.sha`).
- The catalog identity moves from `1f757edd…ebdb57` to `cdab0ec9…13a704`. `TestWorkflowServiceCatalogIdentityGolden` pins the new value.
- Existing Cases keep their identity. All 31 checked-in Case files and the 7 Cases `model/scalav2/goir/testpilot` lowers have the same file hash, wire hash and `CaseFingerprint` before and after (`r2-case-identity-diff.txt`, diff rc 0).

### Findings, red then green

Red logs come from the round-2 tests run against the round-1 code through `go test -overlay`. Green is `r2-final-testpilot.log` (rc 0, 13 packages).

| # | Finding | Fix | Red (log, failure line) | Proving tests |
|---|---|---|---|---|
| 1 | Reservations of attempts that never arrive failed the Run | When an attempt answers anything but a retryable failure, `Ledger.ReleaseActivityAttempts` names every later reservation still reserved and the Session settles each as `CANCELED` with `response: NOT_NEEDED`. The scheduler records that outcome and goes on. | `r2-red-04`, `…/an_early_completion_ends_the_activity_before_its_declared_retries (30.01s)` "should have 3 item(s), but has 0"; `r2-red-03`, `TestReleaseActivityAttempts…` expected two identities, got nil | `TestRunRecordsWhatEachDeclaredActivityAttemptDid` (through `PreparedCase.Run`), `TestTransportAnAttemptThatEndsTheActivityReleasesTheLaterOnes`, `TestReleaseActivityAttemptsReleasesTheAttemptsAfterATerminalOne` |
| 2 | Attempts took instructions in arrival order | `AdmitActivity` binds SDK attempt N to reservation N. `executeActivity` runs the instruction at the reservation's ordinal and first waits for every earlier attempt of the same activity to settle. | `r2-red-03`, `TestAdmitActivityBinds…` expected `session-activity-1`, got `session-activity`; `r2-red-04`, `TestActivityAttemptsThatArriveInverted…` | the same two, plus the retry subtest of `TestActivityAttemptFailsAsItsInstructionSays` |
| 3 | An undeclared attempt replayed the last answer | The ledger returns `ErrAttemptUndeclared`, the worker refuses non-retryably and the Session emits the Driver diagnostic `activity_attempt_undeclared`. Only a redelivery of an admitted attempt replays. | `r2-red-04`, `TestDuplicateDeliveryOfAnAttemptIsAnsweredOnceRun` | that test, and the "an attempt past the script" case of `TestAdmitActivityBinds…` |
| 4 | The attempt's real answer never reached the Run | Every attempt settles its reservation with a typed `activity_attempt` and no error of its own. The scheduler publishes that event first and judges second, so a `REFUSED` attempt is recorded and then makes the Run incomplete with `activation_failed`. | `r2-red-03`, `…/an_attempt_the_worker_refused` "should have 2 item(s), but has 1"; `r2-red-04`, `…/the_worker_refuses_an_attempt` | `TestRunRecordsEachActivityAttemptAsATypedFact`, `TestRunRecordsWhatEachDeclaredActivityAttemptDid`, `TestActivityAttemptTheDriverFailsIsRecordedAsRefused` |
| 5 | A `Failure`-typed result could not be a result | Failing is the new arm and the new Opcode `ActivityAttemptFailure` (10). `Finish` completes with any value. The round-1 convention and its admission rule are deleted. | `r2-red-01` and `r2-red-02`, compile errors on the missing arm; `r2-red-04`, `TestActivityFinishCompletesEvenWithAFailureMessageAsItsResult` "a value, not an outcome (type: Unspecified)" | that test, `TestPrepareAdmitsAnAttemptFailureOnlyAsAnApplicationFailureOfAnActivity`, `TestActivityFinishCompletesWithAFailureMessageAsItsResult`, the "failure message as its result" case through `PreparedCase.Run` |

The typed fields replace the text in `detail`. `detail` now holds only the cause of a refusal.

Status and response read together. `COMPLETED`, `FAILED_RETRYABLE` and `FAILED_NON_RETRYABLE` settle `SUCCEEDED`, because the activation did what the Program declared. `REFUSED` settles `SDK_FAILURE` with code `umpire_worker`. `NOT_NEEDED` settles `CANCELED` with the attempt number and no delivery. A `CANCELED` reservation without the attempt fact keeps the scheduler's old rule, so task 10's held and undispatched case is untouched.

### Re-recording

`PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make umpire-rerecord-pinned-runs` exits 0 in 33 seconds (`r2-rerecord.log`). Both live tests ran on the in-process cluster and nothing reached `lake`.

Before it, the guarded tools run failed six packages beyond the three lake-stub tests, all on the stale catalog (`r2-tools-stale.log`). After it, the guarded tools run is back at baseline, 33 packages ok and only the three lake-stub tests failing (`r2-final-tools.log`, rc 1).

Five records changed (`r2-testdata-changed.txt`, from a checksum listing of 108 testdata files):
- `tools/umpire/replay/testdata/nexusCallerControl-forgedCompletion-run.json`
- `tools/canary/assessment/testdata/nexusCallerCanary-syncCompletion-run.json`
- `tools/umpire/evaluation/testdata/receipts/accepted.json`, `incomplete.json`, `rejected.json`

I compared each re-recorded Run with its predecessor field by field. Each has the same event count (26 and 27), the same disposition and verdict, and differs only in the catalog identity, the Run ID, elapsed times and server-issued values such as request IDs and event times.

### Lean API generator

`make umpire-gen-lean-api` runs without `lake` and exits 0 under the guard (`r2-gen-lean-api.log`). It rewrote `model/lean/Temporal/API.lean` and `model/lean/Temporal/API/Types.lean`, and none of that came from this task. The generator skips the Testpilot package (`--skip-package temporal.server.api.testpilot.v1`), and the diff has 0 lines naming Testpilot. All of it is 101 schema nodes of `temporal.server.api.modelir.v1` plus renumbering, so the generated Lean API was already stale against another task's `modelir` proto.

`model/**` is not mine, so I restored both files to their bytes before the run, which equal `HEAD` (checksums in `r2-lean-api-before.sha`). The generator's output is kept at `.flow/tmp/fn-107/task13-logs/r2-lean-api-generated/` for the owner of that staleness. This departs from "run it and say what changed" only in not leaving the unrelated output in the tree.

### Files

Changed, protocol and generated: `proto/internal/temporal/server/api/testpilot/v1/{instruction,run,program}.proto`, `api/testpilot/v1/{instruction.pb.go,instruction.go-helpers.pb.go,run.pb.go,run.go-helpers.pb.go,program.pb.go}`.

Changed, production: `common/testing/testpilot/contract/profile.go`, `common/testing/testpilot/contract.go`, `internal/execution/{dataflow.go,scheduler.go}`, `temporal/internal/delivery/activity.go`, `temporal/worker/{sdk.go,interpreter.go,session.go,routing.go}`.

Changed, prose: `common/testing/testpilot/README.md`, `internal/execution/README.md`, `temporal/worker/README.md`.

Changed, tests and pins: `internal/execution/activity_admission_test.go`, `temporal/internal/delivery/activity_test.go`, `temporal/worker/{activity_fixture_test.go,activity_test.go,activity_sdk_test.go,activity_transport_test.go}`, `temporal/profile_test.go`, `temporal/catalog_test.go`, and the five records above.

Added: `common/testing/testpilot/temporal/activity_run_test.go`.

First-touch copies are under `.flow/tmp/fn-107/task13-before/`, and the round-1 state of every file under `task13-before/r2/`.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (13 packages ok) | 0 |
| `… go test -tags test_dep -race -count=1 ./common/testing/testpilot/...` (12 packages ok) | 0 |
| `… go test -tags test_dep -race -count=5` on the activity tests of `temporal`, `temporal/worker`, `temporal/internal/delivery` | 0 |
| `… go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (8 packages ok) | 0 |
| `… go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... ./api/testpilot/...` | 0 |
| `… go vet -tags 'test_dep integration' ./tests/` and `go build ./...` | 0, 0 |
| scoped `make lint-code … GOLANGCI_LINT_FIX=false` on seven packages, now including `testpilot` and `testpilot/contract` (0 issues; the first run reported 1, a forbidden `require.Eventually`, replaced by `await.Require`) | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … make protoc` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … make umpire-rerecord-pinned-runs` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...` (33 ok, the three lake-stub tests fail) | 1, the baseline |
| `PATH=/tmp/umpire-no-lean:$PATH … make lint-protos lint-api` | 2 |
| `… go test -tags test_dep -race -count=1 ./tests/testcore/testpilot/...` | 1, the pre-existing race |

`lint-protos` passes. `lint-api` reports 1 finding at `case.proto:43` and 52 in `modelir/v1/ir.proto`, none in a file this task changed.

The race at `workflow_start_artifact_test.go:356` is unchanged and still fails `TestWorkflowStartCaseSequentialAndConcurrentRunIsolation` under `-race`. I did not touch it.

Mutation, by `go test -overlay` from the scratchpad (`r2-mutants.py`, `r2-mutation.log`): 58 mutants of the round-2 code. The first pass killed 49, 7 survived and 2 did not build. New assertions kill 5 of the survivors. The other 2 mutated code no test could observe, an `Admit` after a completing `Finish` and a bound check on the reservation ordinal, and I deleted both. One no-build was a harness error and dies once corrected. The last, `MaxOpcode` left at the old value, cannot build because the Opcode table's length enforces it.

### Decisions that differ from the task text

- **Two files outside the listed Touches.** `contract/profile.go` and the facade `contract.go` gain the Opcode `ActivityAttemptFailure` and the moved `MaxOpcode`. An arm's field number is its Opcode and `TestInstructionOpcodesCoverTheInstructionTable` enforces a row per arm, so the proto change cannot build without them. The README's "A new instruction" recipe names both edits as step 6.
- **A refusal also releases later attempts.** The worker's refusal is a non-retryable failure, so it ends the activity like a declared one, and the later reservations settle `NOT_NEEDED`.
- **Only later attempts are released.** An earlier attempt the worker never saw stays reserved, because the activity ending did not make it needless. That case stays under task 10's rule.

### Limits and gaps

- **An undeclared attempt has no Run Event.** No reservation stands for it, so the only trace is the Driver diagnostic through `Session.Diagnose`. A test covers that the Session emits it. No test follows it through `PreparedCase.Run`, because the fake server never sends an attempt past the script.
- **The recorded response is what the worker hands the SDK.** The SDK sends it afterwards. If that send fails, the Run still says the attempt answered.
- **A later attempt waits for its predecessors.** If Temporal times out attempt 1 before this worker sees it and then sends attempt 2, attempt 2 waits until its own context ends and is recorded `REFUSED`.
- **Facts of one activity appear in settle order.** The scheduler observes each reservation on its own, so two `NOT_NEEDED` records can appear in either order. The end-to-end test sorts by `sdk_attempt`.
- **No live-cluster activity Run.** The end-to-end tests use an in-process `WorkflowService` that retries as the server does, with a real SDK client, worker and composite Driver.
- **Lean is not updated.** Recipe step 3 (a `Testpilot.Authoring.Program` constructor for the new arm) and the `static-preparation-rejection` conformance variant of step 7 live in `model/lean`, which is not mine and cannot be built here. `umpire-check-testpilot-protocol` and `umpire-check-testpilot-authoring` did not run. Files that mirror the instruction table or the outcome and may need the new arm or field: `model/lean/Testpilot/Authoring.lean`, `model/lean/Temporal/Testpilot/Conformance.lean`, `model/lean/Umpire/Case/Producer.lean`.
- **Task 9.** A failing attempt lowers to `activity_attempt_failure`, never to a `Finish` with a `Failure` result. `DeriveProfile` authorizes the Opcode from the instruction, so lowering needs no Profile change. The earlier notes on start-request assignments and the gap at `model/scalav2/goir/testpilot/lower.go:233-235` still hold.

### Review round 3

The three findings are fixed test-first, the attempt lifecycle is written down as a state machine in the worker README, and one table-driven test drives it through the real Session and ledger. The enum values now say "offered", which changed the descriptor, so the pinned Runs are re-recorded again and the Case identities re-proved. This section supersedes round 2 where they differ: an offered answer no longer releases anything.

### Findings, red then green

Red logs come from the round-3 tests run against the round-2 code through `go test -overlay`. Green is `r3-final-testpilot.log` (rc 0, 13 packages) and `r3-final-race-activity-x20.log` (rc 0).

| # | Finding | Fix | Red | Proving tests |
|---|---|---|---|---|
| 1 | Later reservations were released on the worker's offer, before the server had the answer | The worker releases nothing when it offers an answer. After an attempt settles while a later one is still reserved, the Session long-polls `PollActivityExecution` for the activity run, once per activity, and settles the reservations after the last admitted attempt as `NOT_NEEDED` only when the server returns an outcome. A lost answer therefore leaves the next attempt its reservation, and a redelivery replays the recorded answer. | `r3-red-02-run.log`: the two lost-answer cases, and `…/a_retryable_failure_the_server_does_not_retry_closes_the_activity (30.02s)` | `TestRunRecordsWhatEachDeclaredActivityAttemptDid` cases "a completion the server loses is offered again on redelivery", "a completion the server rejects is followed by the next attempt" and "a retryable failure the server does not retry closes the activity"; `TestTransportTheServerClosingTheActivityReleasesTheLaterAttempts`; `TestActivityClosureIsOnlyWhatTheServerReports` |
| 2 | The scheduler accepted outcomes no Driver may report | One table, `reservationOutcomes`, lists every allowed (entrypoint kind, status, response) combination. `judgeReservation` reads it, checks the identities each response requires, and rejects everything else before anything is published. A rejected outcome fails the Run unrecorded. | `r3-red-01-scheduler.log`: `TestRunRejectsAnActivityAttemptReportedForAnotherKindOfEntrypoint/one_that_claims_an_activity_position`, `…/an_attempt_that_names_no_response` | `TestReservationOutcomesAreJudgedByOneClosedTable` (6 kinds x 7 statuses x 8 attempt forms x used and unused, 672 verdicts), `TestReservationOutcomeRequiresTheIdentitiesOfItsResponse`, the two above |
| 3 | `NOT_NEEDED` could be published before the attempt it depends on, and named an SDK attempt | The scheduler observes the reservations of one activity with one waiter in position order, so their events keep attempt order. A `NOT_NEEDED` event is caused by the carrying instruction and by the last recorded attempt of the activity, must follow one, and carries no `sdk_attempt` and no delivery. | `r3-red-01-scheduler.log`: events arrive `i1` before `i0`; `r3-red-02-run.log`: `…/an_early_completion…` | `TestRunRecordsEachActivityAttemptAsATypedFact` (first reservation settles last, handles handed back reversed) and the end-to-end test, both asserting source, causes and outcome whole and unsorted |

The end-to-end test no longer sorts. It ran 20 times under `-race` with the other activity tests (rc 0).

### Protocol

- The enum values are now `OFFERED_COMPLETED`, `OFFERED_FAILED_RETRYABLE`, `OFFERED_FAILED_NON_RETRYABLE`, `REFUSED` and `NOT_NEEDED`. The comments on `ActivityAttempt` and the enum say the fact is what the worker offered and never the server's acceptance.
- `sdk_attempt` and `delivery_id` are documented as unset when no attempt was delivered. No field was added.
- `make protoc` under the guard exits 0 and changed `api/testpilot/v1/run.pb.go` and `run.go-helpers.pb.go` only.
- The catalog identity is now `4e41615e…53ba2e`, pinned in `catalog_test.go`.
- All 31 checked-in Case files and the 7 lowered Cases still have the hashes and fingerprints they had before round 2 (`r3-case-identity-diff.txt`, diff rc 0).

### Re-recording

`PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make umpire-rerecord-pinned-runs` exits 0 (`r3-rerecord.log`). The same five records changed as in round 2 (`r3-testdata-changed.txt`), and both Run records carry the new catalog identity. The guarded tools run is at baseline afterwards, 33 packages ok and only the three lake-stub tests failing (`r3-final-tools.log`, rc 1).

I did not rerun `make umpire-gen-lean-api`. Its generator skips the Testpilot package, and this round changed no other proto.

### The lifecycle as one state machine

`temporal/worker/README.md` now holds the eight states of a declared attempt's reservation, what the Run records in each, the five allowed transitions with their causes, and the refused ones. `TestDeclaredActivityAttemptsFollowTheirLifecycle` drives 8 scenarios and 35 steps through the real Session and ledger. After every step it checks the state of all three reservations, and at the end what each settled with, how often each instruction ran and how often the server was asked.

Writing the table found one defect in my round-2 code. I had deleted the `Admit` after a completing `Finish` as unobservable. It is observable once a lost completion is followed by the next attempt, whose default guard reads that outcome, so the scenario failed with "the activity attempt's instruction is disabled". The `Admit` is back and a mutant that removes it dies.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (13 packages ok) | 0 |
| `… go test -tags test_dep -race -count=1 ./common/testing/testpilot/...` (12 packages ok) | 0 |
| `… go test -tags test_dep -race -count=20` on the activity tests of `temporal`, `temporal/worker`, `temporal/internal/delivery`, `internal/execution` | 0 |
| `… go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (9 packages ok) | 0 |
| `… go vet -tags test_dep` on Testpilot, tools and `api/testpilot`; `go vet -tags 'test_dep integration' ./tests/`; `go build ./...` | 0, 0, 0 |
| scoped `make lint-code … GOLANGCI_LINT_FIX=false` on seven packages (0 issues; the first run reported 1, a switch without a default clause, fixed) | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … make protoc` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … make umpire-rerecord-pinned-runs` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...` (33 ok, the three lake-stub tests fail) | 1, the baseline |
| `PATH=/tmp/umpire-no-lean:$PATH … make lint-protos`; `make lint-api` | 0; 2 with no finding in `run.proto` |
| `… go test -tags test_dep -race -count=1 ./tests/testcore/testpilot/...` | 1, the pre-existing race at `workflow_start_artifact_test.go:356` |

Mutation by overlay (`r3-mutants.py`, `r3-mutation.log`): 34 mutants of the round-3 code. Four survived at first and new tests kill them (a poll error releasing, the watch outliving the Session, a second watch per activity, a watch with nothing reserved). One slot mutated two checks that the table's zero rule already makes redundant, so I deleted both checks. The remaining 33 are all killed.

### Files changed this round

Production: `proto/…/testpilot/v1/run.proto`, `api/testpilot/v1/{run.pb.go,run.go-helpers.pb.go}`, `internal/execution/scheduler.go`, `temporal/internal/delivery/activity.go`, `temporal/worker/{sdk.go,session.go,driver.go,interpreter.go,reservation.go}`.

Prose: `internal/execution/README.md`, `temporal/worker/README.md`.

Tests and pins: `internal/execution/activity_admission_test.go`, `temporal/internal/delivery/activity_test.go`, `temporal/worker/{activity_fixture_test.go,activity_test.go,activity_transport_test.go}`, `temporal/activity_run_test.go`, `temporal/catalog_test.go`, the five records.

Added: `internal/execution/activity_outcome_test.go`, `temporal/worker/activity_lifecycle_test.go`.

The round-2 state of every file is under `.flow/tmp/fn-107/task13-before/r3/`.

### Decisions

- **The worker reads the server.** The evidence that an activity closed is `PollActivityExecution`, called through the SDK client the worker already holds. It is a read the Profile does not authorize by method, like the worker's existing `CancelWorkflow` call. If you want it authorized or moved to the controller Session, that is a design change outside this task.
- **The poll runs only when needed.** A script with one declared attempt never asks the server.
- **A refusal releases nothing either.** Round 2 released on any terminal offer. Now only the server's answer does.
- **Order is kept by a single waiter, not by a buffer.** The scheduler waits on an activity's reservations in position order. Its rule for a held or undispatched reservation is unchanged.

### Limits and gaps

- **No live server has answered this poll.** I read the long-poll contract from `chasm/lib/activity/handler.go` and `responses.go` (an expired poll returns no outcome, a closed activity always returns one). The tests use an in-process `WorkflowService`. A live standalone-activity Run would be the first real proof.
- **A poll failure leaves the Run waiting.** The later reservations stay reserved, the Session emits `activity_closure_unobserved`, and the Run ends at its deadline.
- **An earlier attempt the worker never sees hides a later refusal.** The single waiter reaches attempt 2's `REFUSED` only after attempt 1's reservation settles, and that one settles as a bare cancellation that stops the Run first. The refusal is then unrecorded. This is the never-seen case that belongs to task 10.
- **`NOT_NEEDED` is caused by the last recorded attempt.** That is the attempt whose offer closed the activity in the common case. If the server closed the activity for another reason, such as an exhausted retry policy, it is simply the last attempt the Run recorded.
- **An undeclared attempt still has no Run Event,** only the Driver diagnostic.
- **Lean is not updated** (the review's P3). The authoring constructor for `activity_attempt_failure` and the conformance rejection variant need the Lean toolchain. `umpire-check-testpilot-protocol` and `umpire-check-testpilot-authoring` did not run.
- **Task 9** authors the Contract comparisons over `outcome.activity_attempt.*`. It must read `response` as what the worker offered.

### Review round 4

Both crossed-identity findings are fixed test-first, and I swept every place in the diff where an activity identity is read from one source and used with another. The descriptor did not change, so nothing was re-recorded.

### Findings, red then green

Red is `r4-red-01.log` (rc 1), the new tests against the round-3 code. Green is `r4-final-testpilot.log` (rc 0, 13 packages).

| # | Finding | Fix | Red | Proving tests |
|---|---|---|---|---|
| 1 | The poll's response `RunId` was not compared | The poll returns the run the server names. The Session releases only when that run is the run it asked about. Otherwise it releases nothing and reports the Driver diagnostic `activity_closure_crossed`. | `TestTransportAnOutcomeOfAnotherRunReleasesNothing`: expected 1 diagnostic, got 0, and the later reservation was released | that test, through the in-process service and the real poll; the lifecycle scenario "the server answers with the outcome of another run" |
| 2 | The scheduler kept only the prior event's source | `attemptFacts` keeps the activity run the first recorded attempt named. `judgeReservation` rejects, before publishing, any later outcome of the activity that names another run, offered or not needed. | `TestRunRecordsEachActivityAttemptAsATypedFact/a_retry_that_names_another_activity_run` and `/a_position_not_needed_in_another_activity_run`: both crossed outcomes were recorded | those two cases, and `TestReservationOutcomeRequiresTheIdentitiesOfItsResponse` (14 rows beside the 672-combination test, two of them crossed-run) |

### Identity sweep

Each site where `activity_run_id`, `sdk_attempt`, `delivery_id` or an operation or reservation key crosses from one source to another:

| Site | Read from | Used with | Compared, or why it cannot differ |
|---|---|---|---|
| `Ledger.AdmitActivity`, run | the delivery | the start response's run and the run of the first admitted attempt | compared, `ErrRouteConflict` before anything is consumed or replayed; the Session now reports `activity_run_crossed` (new this round) |
| `Ledger.AdmitActivity`, binding | the delivery's namespace, activity ID, type, queue | the binding the start carried | compared, `ErrBindingMismatch` |
| `Ledger.AdmitActivity`, route | the header | the ledger's own route, Session and Run | compared, `ErrRouteCrossed` |
| `Ledger.AdmitActivity`, attempt | the delivery's SDK attempt | the reservation of that position | by construction, attempt N indexes reservation N; past the script is `ErrAttemptUndeclared`, diagnosed |
| replay of an admitted attempt | the redelivery | the recorded activation | the run is compared first (row 1). The delivery identity differs by design, and the record keeps the first delivery's |
| `Driver.activityCandidates` | the delivery's binding | the Sessions indexed under that binding | a crossed binding finds no Session and is refused `ErrRouteCrossed`. No Session is addressed, so there is no diagnostic sink |
| `Ledger.PinStartResponse` | the start response's run | the run of an attempt already admitted | compared, `ErrRouteConflict` |
| `activateActivity`, recorded outcome | the ledger's activation only | the Run Event | one source: run, attempt and delivery all come from the activation, never from the incoming delivery |
| `Session.admitActivity`, cancellation | the delivery identity bound at admission | the identity the cancellation presents | compared, `ErrInvalid` |
| `watchActivityClosure`, request | namespace and activity ID from the admitted delivery, run from the activation | the poll | cannot differ: this path runs only for the delivery `AdmitActivity` just admitted, whose binding and run it compared (rows 1 and 2) |
| `watchActivityClosure`, response | the server's `run_id` | the run asked about | compared (finding 1) |
| `releaseActivityAttempts` | the released identities from the ledger | the Session's reservations, the recorded run | one ledger and one Session; the run recorded is the one the poll confirmed |
| `Ledger.ReleaseActivityAttempts` | the activation given | the ledger, the route kind and the stored activation | compared, `ErrRouteCrossed` |
| `awaitEarlierAttempts`, `scriptOf` | the activation's reservation origin and entrypoint | the Session's reservations and scripts | compared by origin and entrypoint; the ledger checks the consumed coordinate's entrypoint against the reservation's at admission |
| scheduler, `sdk_attempt` | the outcome | the reservation's position | compared, rejected |
| scheduler, `activity_run_id` | the outcome | the run the activity's first recorded attempt named | compared (finding 2) |
| scheduler, causal link | the prior attempt's source | the group of the same carrying instruction and declaration | the group key is the reservation's own source prefix, and `validateReservations` checks each identity against the request |
| scheduler, `delivery_id` | the outcome | nothing | only required to be present; there is no second source to compare it with |
| transport (`ExecuteActivity`) | the SDK's task info | the delivery handed to the ledger | one source |

The scheduler cannot compare the outcome's run with the start response's run: it is generic and does not read Temporal fields. The worker's ledger makes that comparison.

The lifecycle table test gained crossed rows for the sites a delivery or the server can cross: a later attempt naming another run, a redelivery naming another run, this activity's route under another activity's ID, and the server answering with another run's outcome. Each scenario now also asserts the number of diagnostics the Session reported.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (13 packages ok) | 0 |
| `… go test -tags test_dep -race -count=1 ./common/testing/testpilot/...` (12 packages ok) | 0 |
| `… go test -tags test_dep -race -count=20` on the activity tests of four packages | 0 |
| `… go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (9 packages ok) | 0 |
| `… go vet -tags test_dep` on Testpilot, tools and `api/testpilot`; `go vet -tags 'test_dep integration' ./tests/`; `go build ./...` | 0, 0, 0 |
| scoped `make lint-code … GOLANGCI_LINT_FIX=false` on seven packages (0 issues; the first run reported 1, a switch without a default clause, rewritten as if/else) | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH … go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...` (33 ok, the three lake-stub tests fail) | 1, the baseline |

Not run this round: `make protoc` and `make umpire-rerecord-pinned-runs`, because no proto changed and `TestWorkflowServiceCatalogIdentityGolden` still passes on `4e41615e…53ba2e`.

Mutation by overlay (`r4-mutants.py`, `r4-mutation.log`): 14 mutants of the comparisons above, all killed.

### Files changed this round

Production: `internal/execution/scheduler.go`, `temporal/worker/{driver.go,sdk.go,session.go,routing.go}`.

Prose: `internal/execution/README.md`, `temporal/worker/README.md`.

Tests: `internal/execution/{activity_admission_test.go,activity_outcome_test.go}`, `temporal/worker/{activity_fixture_test.go,activity_lifecycle_test.go,activity_test.go,activity_transport_test.go}`.

The round-3 state of every Testpilot file is under `.flow/tmp/fn-107/task13-before/r4.Pezv0M/`.

### Decisions and limits

- **A crossed poll answer ends the watch for that activity.** The later reservations stay reserved and the Run ends at its deadline with the diagnostic. I did not add a retry.
- **An existing expectation changed by intent.** `TestActivityDeliveryIsRefusedBeforeItsScriptRuns/a_run_the_start_did_not_answer_with` now expects 1 diagnostic where it expected 0, because the finding requires the refusal to be diagnosed.
- **A crossed binding has no diagnostic.** No Session is indexed under it, so nothing can report it. The delivery is still refused.
- The round-3 limits stand: no live server has answered the poll, an undeclared or crossed attempt leaves only a Driver diagnostic and no Run Event, and Lean is not updated (the review's P3).

### Review round 5

The review returned SHIP with one P3, now closed. Tests that asserted only how many diagnostics a Session reported now assert what it reported. Only test files changed.

- `captureDiagnostics` in `temporal/worker/activity_fixture_test.go` installs a log as the Session's diagnostic sink and returns each diagnostic's kind, code and detail in order.
- `TestTransportAnOutcomeOfAnotherRunReleasesNothing` compares the one diagnostic whole: invariant, `activity_closure_crossed`, and the detail naming both runs.
- `TestDuplicateDeliveryOfAnAttemptIsAnsweredOnceRun` (`activity_attempt_undeclared`), `TestActivityClosureIsOnlyWhatTheServerReports` (`activity_closure_unobserved`) and `TestActivityDeliveryIsRefusedBeforeItsScriptRuns` (`activity_run_crossed` and the three `activity_delivery_late` cases) compare whole diagnostics too.
- The lifecycle table names the expected codes in order per scenario, for example `activity_attempt_undeclared`, `activity_run_crossed`, `activity_run_crossed`, `activity_delivery_late` on the declared path.

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/...` (12 packages ok) | 0 |
| `… go test -tags test_dep -race -count=5 ./common/testing/testpilot/temporal/worker/` | 0 |
| scoped `make lint-code LINT_CODE_TARGETS='./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

No mutation run and no re-recording this round, as instructed. No production file, proto or pinned record changed.

stage: implement - ran (worker subagent, session model claude-opus-5-5)

Conductor: the review required, and the conductor authorized, a Testpilot protocol addition in this task: an `activity_attempt_failure` instruction arm and a typed `InstructionOutcome.activity_attempt` (activity run id, SDK attempt, delivery id, response `OFFERED_COMPLETED | OFFERED_FAILED_RETRYABLE | OFFERED_FAILED_NON_RETRYABLE | REFUSED | NOT_NEEDED`). All Testpilot protocol files are inside the Driver catalog identity, so the five pinned Runs under `tools/umpire` and `tools/canary` were re-recorded with `make umpire-rerecord-pinned-runs` (twice, once per descriptor change); every checked-in Case file and the seven lowered Cases keep their identities. Final gates on the merged tree (tasks 7, 12 and 13 together): the Testpilot gate and `./model/scalav2/goir/...` pass; the guarded `tools/umpire` and `tools/canary` run fails only in the three tests of `umpire-gen-lean-dynamic-config-catalog` that call `lake`.

Open by necessity or by assignment: no live server has run the activity path or answered the outcome poll; an undeclared or crossed attempt leaves a Driver diagnostic and no Run event; an earlier attempt the worker never saw stays reserved (task 10); the Lean authoring constructor and conformance variant for the new arm are not written (no Lean toolchain); lowering the new arm and authoring the activity realization are task 19.

The work is uncommitted; the owner makes the commits. The task diff and the five review outputs are under `.flow/tmp/fn-107/task13/`.

stage: implement - ran (worker subagent, session model claude-opus-5-5; five fix rounds; one turn lost to an API 529 and resumed)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f766-9220-7603-9ae1-c47d402e7a43 over the uncommitted task diff; rounds 1-4 NEEDS_WORK with 2, 5, 3 and 2 introduced P2 findings, all fixed; round 5 SHIP with two P3, one fixed, one (Lean) open)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... rc=0; ./model/scalav2/... ./model/go/... rc=0, pre-edit), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race ./common/testing/testpilot/... (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/internal/activation ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), CC=/usr/bin/clang mise exec -- go build ./tools/... (rc=0), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./tools/umpire/... ./tools/canary/... (rc=0), round 0 mutation via go test -overlay from the session scratchpad: 74 effective mutants plus 1 no-op, 4 effective survivors (3 equivalent, 1 unobservable), logs .flow/tmp/fn-107/task13-logs/mutants-*.log, NOT RUN (forbidden by the conductor, lake downloads a Lean toolchain): go test ./tools/umpire/... ./tools/canary/..., SKIPPED (need Lean lake): make umpire-check-testpilot-protocol; make umpire-check-testpilot-authoring, NOT RUN (no Scala or IR file changed): make umpire-check-scala; make umpire-gen-scala; make lint-scala, NOT RUN: any integration-tagged or live-cluster test, no flowctl gate receipt attempted: tree dirty outside the ignore set, Review round 1: baseline PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... (rc=1: 33 packages ok; unrunnable on the lake stub: TestRenderedCatalogElaboratesSignedLiteralBoundaries, TestRenderedCatalogElaboratesLeanEscapedStrings, TestRenderedProductionCatalogIsStableAndLeanElaborates in tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog), Review round 1: catalog identity 1f757edd4c18a3f52fcb0205bee2ea3da389517b6048bc79e49a136ab3ebdb57 before and after; all nine testpilot protocol files are in the catalog closure (r1-catalog-before.log, r1-catalog-after.log), Review round 1: PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make protoc (rc=0; of 140 files under api/ only api/testpilot/v1/instruction.pb.go and program.pb.go changed, comments only), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race ./common/testing/testpilot/... (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=1: pre-existing data race in tests/testcore/testpilot/workflow_start_artifact_test.go:356, a file this task did not touch), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./api/testpilot/... (rc=0), Review round 1: GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/internal/activation ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 1: after PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... (rc=1: same result set as the baseline, 33 packages ok, the same three lake-stub tests unrunnable; no pinned Run stale), Review round 1: mutation via go test -overlay (no repository file rewritten): 49 mutants, 1 equivalent survivor, logs .flow/tmp/fn-107/task13-logs/r1-mutants-*.log, Review round 1: NOT RUN: make umpire-rerecord-pinned-runs (nothing stale); lake protocol targets; Scala make targets; integration-tagged tests, review round 2: red logs .flow/tmp/fn-107/task13-logs/r2-red-01-delivery-execution.log and r2-red-02-all-tests-written.log (compile red, vet rc=1), r2-red-03-ledger-scheduler.log and r2-red-04-worker-run.log (behavioural red against round-1 code via go test -overlay, rc=1), review round 2: PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep mise exec -- make protoc (rc=0; only api/testpilot/v1 changed, 5 files), review round 2: Case identity before/after, 31 checked-in Case files and 7 lowered Cases, diff rc=0 (r2-case-identity-diff.txt), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0, 13 packages ok), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=1 ./common/testing/testpilot/... (rc=0, 12 packages ok), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=5 ./common/testing/testpilot/temporal/ ./common/testing/testpilot/temporal/worker/ ./common/testing/testpilot/temporal/internal/delivery/ -run 'Activity|Transport|Delivery|TestRunRecords' (rc=0), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0, 8 packages ok), review round 2: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... ./api/testpilot/... (rc=0); go vet -tags 'test_dep integration' ./tests/ (rc=0); go build ./... (rc=0), review round 2: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make lint-code LINT_CODE_TARGETS='./common/testing/testpilot ./common/testing/testpilot/contract ./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/internal/activation ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), review round 2: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... before re-recording (rc=1; six packages stale on the catalog identity plus the three lake-stub tests, r2-tools-stale.log), review round 2: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make umpire-rerecord-pinned-runs (rc=0; 5 records changed, r2-testdata-changed.txt), review round 2: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... after re-recording (rc=1, baseline: 33 packages ok; unrunnable on the lake stub: TestRenderedCatalogElaboratesSignedLiteralBoundaries, TestRenderedCatalogElaboratesLeanEscapedStrings, TestRenderedProductionCatalogIsStableAndLeanElaborates in tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog), review round 2: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make umpire-gen-lean-api (rc=0; output differs only for temporal.server.api.modelir.v1, 0 Testpilot lines; model/lean/Temporal/API.lean and API/Types.lean restored to their pre-run bytes, generated output kept under task13-logs/r2-lean-api-generated/), review round 2: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make lint-protos lint-api (rc=2; lint-protos passes, lint-api reports case.proto:43 and 52 findings in modelir/v1/ir.proto, none in a file this task changed), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=1 ./tests/testcore/testpilot/... (rc=1, pre-existing race at workflow_start_artifact_test.go:356, not this task's), review round 2 mutation via go test -overlay: 58 mutants; first pass 49 killed, 7 survived, 2 no-build; 5 survivors killed by new assertions, 2 resolved by deleting unobservable code, 1 no-build was a harness error and is killed, 1 no-build is enforced by the compiler (r2-mutation.log, r2-mutants.py), not run in round 2: any make target that calls lake (lint-model, umpire-check-regression, umpire-check-testpilot-protocol, umpire-check-testpilot-authoring, umpire-check-goldens, umpire-check-lean-api), review round 3: red logs .flow/tmp/fn-107/task13-logs/r3-red-01-scheduler.log and r3-red-02-run.log (round-3 tests against round-2 code via go test -overlay, rc=1 each), review round 3: PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep mise exec -- make protoc (rc=0; only api/testpilot/v1/run.pb.go and run.go-helpers.pb.go changed), review round 3: Case identity, 31 checked-in Case files and 7 lowered Cases, diff against the pre-round-2 listing rc=0 (r3-case-identity-diff.txt), review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0, 13 packages ok), review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=1 ./common/testing/testpilot/... (rc=0, 12 packages ok), review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=20 ./common/testing/testpilot/temporal/ ./common/testing/testpilot/temporal/worker/ ./common/testing/testpilot/temporal/internal/delivery/ ./common/testing/testpilot/internal/execution/ -run 'Activity|Transport|Delivery|TestRunRecords|TestRunRejects|TestReservationOutcome|Lifecycle|Closure' (rc=0), review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0, 9 packages ok), review round 3: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... ./api/testpilot/... (rc=0); go vet -tags 'test_dep integration' ./tests/ (rc=0); go build ./... (rc=0), review round 3: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make lint-code LINT_CODE_TARGETS='./common/testing/testpilot ./common/testing/testpilot/contract ./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/internal/activation ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), review round 3: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make umpire-rerecord-pinned-runs (rc=0; 5 records changed, r3-testdata-changed.txt), review round 3: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... (rc=1, baseline: 33 packages ok; unrunnable on the lake stub: TestRenderedCatalogElaboratesSignedLiteralBoundaries, TestRenderedCatalogElaboratesLeanEscapedStrings, TestRenderedProductionCatalogIsStableAndLeanElaborates), review round 3: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make lint-protos (rc=0); make lint-api (rc=2, no finding in testpilot/v1/run.proto; pre-existing findings in case.proto and modelir/v1/ir.proto), review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=1 ./tests/testcore/testpilot/... (rc=1, pre-existing race at workflow_start_artifact_test.go:356, not this task's), review round 3 mutation via go test -overlay: 34 mutants; 4 survivors killed by new tests, 1 slot removed with two redundant checks it mutated, remaining 33 all killed (r3-mutation.log, r3-mutants.py), not run in round 3: make umpire-gen-lean-api (the generator skips the Testpilot package and no other proto changed) and any make target that calls lake, review round 4: red log .flow/tmp/fn-107/task13-logs/r4-red-01.log (new crossed-run tests against round-3 code, rc=1), review round 4: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0, 13 packages ok), review round 4: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=1 ./common/testing/testpilot/... (rc=0, 12 packages ok), review round 4: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=20 ./common/testing/testpilot/temporal/ ./common/testing/testpilot/temporal/worker/ ./common/testing/testpilot/temporal/internal/delivery/ ./common/testing/testpilot/internal/execution/ -run 'Activity|Transport|Delivery|TestRunRecords|TestRunRejects|TestReservationOutcome|Lifecycle|Closure' (rc=0), review round 4: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0, 9 packages ok), review round 4: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... ./api/testpilot/... (rc=0); go vet -tags 'test_dep integration' ./tests/ (rc=0); go build ./... (rc=0), review round 4: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make lint-code LINT_CODE_TARGETS='./common/testing/testpilot ./common/testing/testpilot/contract ./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/internal/activation ./common/testing/testpilot/temporal/internal/delivery ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), review round 4: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... (rc=1, baseline: 33 packages ok; unrunnable on the lake stub: TestRenderedCatalogElaboratesSignedLiteralBoundaries, TestRenderedCatalogElaboratesLeanEscapedStrings, TestRenderedProductionCatalogIsStableAndLeanElaborates), review round 4 mutation via go test -overlay: 14 mutants of the identity comparisons, all killed (r4-mutation.log, r4-mutants.py), not run in round 4: make protoc and make umpire-rerecord-pinned-runs (no proto changed, catalog identity golden passes), and any make target that calls lake, review round 5: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... (rc=0, 12 packages ok), review round 5: CC=/usr/bin/clang mise exec -- go test -tags test_dep -race -count=5 ./common/testing/testpilot/temporal/worker/ (rc=0), review round 5: PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang make lint-code LINT_CODE_TARGETS='./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), review round 5: test-only change (diagnostic capture helper and whole-value diagnostic assertions); no mutation run, no re-recording, conductor final, merged tree: go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/goir/... (rc 0), conductor final: PATH=/tmp/umpire-no-lean:$PATH go test ./tools/umpire/... ./tools/canary/... (only the three lake-stub tests fail), codex impl-review rounds: .flow/tmp/fn-107/task13/t13-r1..r5.md (final SHIP)
- PRs: