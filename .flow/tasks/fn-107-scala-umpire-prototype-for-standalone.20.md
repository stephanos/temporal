---
satisfies: [R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.20 Generic Testpilot evidence sources: single message, Run Events under a guard, kinds sharing a source

## Description
**Touches:** [common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/ir/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/contract/**, common/testing/testpilot/*.go, common/testing/testpilot/README.md, common/testing/testpilot/temporal/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, tests/testcore/testpilot/**, tools/umpire/**/testdata/**, tools/canary/**/testdata/**]

Give Testpilot the three generic evidence primitives the Scala activity and Nexus realizations need. Each is named by the task 19 review (`.flow/tmp/fn-107/task19/t19-r1.md`, judgments c and d) and by task 19's handover; none is feature policy.

**Size:** L

### Approach
- **Single-message read source.** Today `ReadEvidence` lifts evidence only from a repeated field (`internal/execution/evidence.go` ~:121), and every standalone activity status is `DescribeActivityExecution`'s single `info` message. Add a distinct single-message source (or an explicit cardinality on the source), preserving method authorization, descriptor checking, limits, polling semantics and dense-ordinal behaviour. Existing repeated-field reads stay byte- and behaviour-compatible.
- **Run Event evidence with a payload guard.** Attempt start and attempt identity are stable only in the typed `InstructionOutcome.activity_attempt` Run fact (task 13); a Describe poll can miss the transient scheduled and started states. Let an evidence declaration take its occurrences from Run Events under a generic payload predicate (for example `sdk_attempt > 0` and a nonempty delivery, which excludes `NOT_NEEDED` records), keyed by the typed identities. `response` is the worker's offer and proves no server acceptance; the primitive must not suggest otherwise.
- **Distinct kinds in one dense source.** Testpilot rejects two evidence declarations with the same source and operation path, so the five Nexus history kinds, which one read lifts from one history, cannot all be declared. Permit distinct kinds lifted by one instruction from one shared dense source, with one ordinal space, keeping unique-kind admission.
- **A canceled answer for an activity attempt.** The Scala vocabulary declares `Instruction.AttemptCanceled` (task 19) and no Testpilot instruction lowers it: beside `Finish` and `activity_attempt_failure`, an activity script needs an arm by which the worker answers the attempt as canceled, recorded as a typed `activity_attempt` response (an offer, like the others) through the closed `reservationOutcomes` table and the attempt lifecycle task 13 documented. An attempt that deliberately gives no answer (for a start-to-close timeout) is NOT in scope; it stays a recorded limit.
- One additive protocol change for all four if the descriptor must change. Every Testpilot protocol file is inside the Driver catalog identity: measure the blast radius as task 13 did (guarded `tools/umpire` and `tools/canary` baseline before and after), prove every checked-in Case and the lowered Cases keep their identity, and re-record the pinned Runs once with `PATH=/tmp/umpire-no-lean:$PATH make umpire-rerecord-pinned-runs`.
- Test-first. No feature-specific code: the tests use neutral fixtures.

### Investigation targets
**Required:** `.flow/tmp/fn-107/handover/task19-summary.md`, `task13-summary.md` (protocol procedure, re-recording, the three lake-stub tests that are the tools baseline); `common/testing/testpilot/internal/execution/{evidence,dataflow,scheduler}.go`; `internal/verification/correlated.go`; `proto/internal/temporal/server/api/testpilot/v1/{case,program,instruction,run}.proto`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/...` (and `-race` over `./common/testing/testpilot/...`); `PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...` (baseline: only three lake-stub tests fail); `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep mise exec -- make protoc`; scoped `make lint-code` with `GOLANGCI_LINT_FIX=false`. Nothing that calls `lake`.

## Acceptance
- [ ] An evidence declaration can lift from a single message of a response, with method authorization, descriptor checks, limits, polling and dense ordinals as for a repeated field; existing repeated-field Cases are byte- and behaviour-identical.
- [ ] An evidence declaration can take its occurrences from Run Events under a generic payload predicate and typed identity key; a `NOT_NEEDED` or undelivered attempt record is not an occurrence; nothing treats the worker's offered response as server acceptance.
- [ ] Distinct evidence kinds can share one dense source lifted by one instruction; two declarations of one kind are still refused.
- [ ] An activity script can answer an attempt as canceled; the answer is a typed attempt response accepted only for an activity entrypoint, and the lifecycle table test covers it.
- [ ] Every checked-in Case file and every lowered Case keeps its identity; pinned Runs are re-recorded if the descriptor changed; the guarded tools run is at its baseline; existing Testpilot consumers pass.
- [ ] No feature policy and no dependency from Testpilot on `model/`.


## Done summary
Testpilot has the four primitives, in one additive protocol change, with the pinned Runs re-recorded and every gate green. One thing task 19 expects is not delivered and cannot be without a design decision: two evidence kinds lifted from one Run Event (finding 1). Nothing is committed and the task stays `in_progress`.

The workspace copy was wiped by the restart at about 12:15 and everything was redone in the recreated copy. All figures below are from the redone run. The files to merge and every log are also under `.flow/tmp/fn-107/task20-checkpoint/` in the main checkout (logs in `task20-checkpoint/logs/`), identical to the workspace at 14:00.

### Read this first

1. **Two kinds cannot be lifted from one event.** A Run Event carries the Program's evidence Observation once: the Contract evaluator rejects a repeated Observation (`internal/verification/evaluator.go`, `checkEvent`), and task 19's adapter rejects it too (`model/scalav2/goir/conformance/evidence.go:238`, "recorded twice"). So `delivered("statusStarted")` and `delivered("attemptCount")`, which both select every delivered record, fail the Run with `outcome_failed` ("a Run Event is evidence of both ... and ..."). The record is kept and no ordinal is taken. Ways out, neither mine to choose: one kind whose projection rule has two outputs (a `CorrelatedProjectionRule` already carries several), or disjoint guards. Emitting a second evidence event per record would be a new Testpilot design; I did not improvise it.
2. **The five Nexus history kinds were never refused.** Kinds with distinct history arms in one source already prepared and lifted before this task, because each arm's key path differs. `TestOneReadLiftsEveryHistoryKindOfItsSource` passes against the pre-change code (`pin-preexisting-against-head.log`), and five checked-in Cases already share a source. What was refused, and is now admitted, is distinct recorded kinds under the same source and the same key path.
3. **The protocol gained two fields the task text does not name**, because task 19's declarations need them: `RunEventSource.instruction` (one controller instruction's events) and `RunEventSource.run_keyed` (the Run's ID as the operation key). Without them `accepted(...)` would match every succeeded controller call and no Run Event evidence could join the Describe reads.
4. **Lean is not updated.** The Testpilot authoring and conformance mirrors under `model/lean` need the new arm, enum value and fields. They are outside my Touches and cannot be built here.

### Protocol change

One additive change, three files, five generated files, all under `api/testpilot/v1` (`api-files-changed.txt`; `api/modelir` is byte-identical).

- `program.proto`: `RunEventSource` gains `Expression guard = 2`, `InstructionReference instruction = 3`, `bool run_keyed = 4`. `ReadSource` gains `bool single = 3`. The file now imports `expression.proto`.
- `instruction.proto`: arm `ActivityAttemptCancellation activity_attempt_cancellation = 11` and the empty message.
- `run.proto`: `ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED = 6`.

Catalog identity moved from `4e41615e5510…53ba2e` to `c30526ab9db2…f4c2b2`, pinned in `temporal/catalog_test.go`. All 31 checked-in Case files and the 7 lowered Nexus Cases keep file hash, wire hash and fingerprint (`identity-before.txt`, `identity-final.txt`; the diff is the catalog line alone).

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | Single-message read | pass | `TestAReadOfOneMessagePollsAndLiftsAsARepeatedFieldDoes` (both cardinalities through one script), `TestAResponseWithoutTheOneMessageSatisfiesNoCondition`, `TestAReadOfOneMessageThatNeverSatisfiesTimesOutWithoutEvidence`, `TestPrepareRejectsAReadOfOneMessageItCannotLift` (10 rows), `TestAReadOfOneMessageEmitsOneEventWhateverTheFanout` |
| 2 | Run Events under a guard, typed key, `NOT_NEEDED` excluded | pass | `TestRunLiftsOnlyTheAttemptRecordsItsGuardSelects` (4 rows, one showing a presence check does not exclude `NOT_NEEDED`), `TestRunLiftsTheEventsOfOneInstructionKeyedByTheRun`, `TestRunLiftsOnlyTheEventsOfTheInstructionItNames`, `TestAGuardSelectsAmongTheEventsOfAnyKindThatCarriesAPayload`, `TestARecordNoDeclarationCanLiftIsKeptAndFailsTheRun`, `TestAGuardTheRunCannotEvaluateFailsTheLift`, `TestPrepareRejectsRunEventDeclarationsItCannotLift` (12 rows), `TestActivityAttemptEvidenceIsTheSameLiveAndReplayed` |
| 3 | Kinds sharing one dense source | pass | `TestDistinctKindsShareOneDenseSourceAndKeyPath`, `TestOneReadLiftsEveryHistoryKindOfItsSource`, `TestPrepareAdmitsDifferentRunEventRecordsInOneSource`, `TestPrepareRefusesASourceTwoEmittersCountOrOneRecordedKindTwice` (5 rows), the unedited `TestPrepareRejectsEvidenceDeclarations` and the `duplicate-evidence` fixture |
| 4 | Canceled answer | pass | `TestPrepareAdmitsACanceledAnswerOnlyOfAnActivityAttempt` (6 rows), `TestReservationOutcomesAreJudgedByOneClosedTable` (now 8 responses, 10 rows), `TestRunRecordsEachActivityAttemptAsATypedFact` (2 new rows), `TestDeclaredActivityAttemptsFollowTheirLifecycle` (4 new scenarios), `TestTransportTemporalIsToldAnAttemptIsCanceledOnceTheServerAsked`, `TestTransportACancellationTheServerNeverAsksForIsRefused`, `TestRunRecordsWhatEachDeclaredActivityAttemptDid` (new end-to-end case) |
| 5 | Identities, re-recording, tools baseline, consumers | pass | see Protocol change, Re-recording and Gates |
| 6 | No feature policy, no dependency on `model/` | pass | `TestPrivateCoreImportBoundary`; fixtures are a neutral `report` service, fault events and the existing activity fixture |

### How each primitive works

- **Single read.** `single` makes the read's value the one message at its path. Admission requires a singular message; the repeated rule and its error texts are unchanged. Polling reads the one value, and an absent message satisfies no condition, not even a negated one. The read needs one emitted event, so the fanout bound applies only to repeated reads.
- **Run Event guard.** The guard binds like every lift guard, so a guard that may have no value is rejected at preparation, located at `run_event.guard`. At run time a guard error fails the lift. The scheduler now also lifts the record of each reservation (the `DIAGNOSTIC` event that carries `activity_attempt`), which it did not before, so the evidence sits on the event the adapter reads the typed attempt from. Evidence is lifted once, at recording, and stored on the event. Live Monitor and `Evaluate` read the same record, and the end-to-end test compares both Verdicts and the violation whole.
- **A record that cannot be lifted** (unreadable field, or two declarations select it) is still published, without evidence and flagged `execution_incomplete`, then the Run fails with `outcome_failed`. A refused attempt keeps `activation_failed` as its diagnostic.
- **Shared source.** Within one source and key, each recorded kind is declared once: a history arm, or a Run Event kind at an instruction under a guard. A source the Run counts cannot also be counted by an instruction, which was admitted before and produced colliding ordinals.
- **Canceled answer.** Go SDK 1.48 sends `RespondActivityTaskCanceled` only when the server asked that delivery to cancel through a heartbeat; any other canceled error is sent as a retryable failure. The server accepts a canceled answer only for a requested cancellation (`chasm/lib/activity/activity.go:539`). So the instruction heartbeats until the SDK cancels the context with a canceled cause, then hands the SDK a canceled error, recorded as `OFFERED_CANCELED`. If the context ends otherwise the attempt is `REFUSED`. A redelivery waits for the server to ask it too. The worker's SDK option `MaxHeartbeatThrottleInterval` is 100 ms, so the request is seen about that soon.

### Re-recording

`make umpire-rerecord-pinned-runs` exited 0 (`rerecord.log`). Five records changed (`testdata-changed.txt`): the two pinned Runs and the three receipt goldens. Each Run has the same event count (26 and 27), disposition, verdict, rule statuses, event kinds and source ids as before, and differs only in catalog identity, Run ID, elapsed times and server-issued values (`rerecord-compare.txt`). Before re-recording, 14 tools tests failed on the stale catalog (`tools-stale.log`).

### Test-first record

- Compile red: `red-01-compile.log` (missing `Guard`, `OFFERED_CANCELED`, the new arm), `red-03-…-compile.log` (`Instruction`, run key).
- Behavioural red after regeneration and before implementation: `red-02-behaviour.log` (for example `TestDistinctKindsShareOneDenseSourceAndKeyPath` with "evidence source and operation key path are declared twice", and the lifecycle, transport and end-to-end tests) and `red-04-…-behaviour.log`.
- Written after the code and never red: `TestOneReadLiftsEveryHistoryKindOfItsSource` and `TestPrepareStillAdmitsOneRecordedKindUnderTwoKeyPaths` (both pin behaviour that existed), the presence-check row, `TestAMonitorStopOnARefusedAttemptIsNotReplacedByItsFailure`, `TestAGuardTheRunCannotEvaluateFailsTheLift`. `TestPrepareAdmitsACanceledAnswerOnlyOfAnActivityAttempt` was red only at compile, because the Opcode table cannot build without its row.
- Mutation by `go test -overlay` (`mutants.py`, `mutation.log`, `mutation-extra.log`): 71 mutants, 71 killed. Survivors of the run before the restart led to added tests and to removing a redundant SDK option, a redundant nil guard and a cleanup branch nothing exercised.

### Gates (final tree)

| Command | rc |
|---|---|
| Baseline before editing: `go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/... ./model/go/...` (23 ok) | 0 |
| Baseline guarded tools run (33 ok, the three lake-stub tests fail) | 1 |
| `go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/...` (17 ok) | 0 |
| `go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (10 ok) | 0 |
| `go test -tags test_dep -count=1 -race ./common/testing/testpilot/...` (12 ok) | 0 |
| `-race -count=10` over the new tests of three packages | 0 |
| `go vet -tags test_dep` over Testpilot, testcore, tools, `api/testpilot`; `go vet -tags 'test_dep integration' ./tests/`; `go build ./...` | 0, 0, 0 |
| scoped `make lint-code … GOLANGCI_LINT_FIX=false` on five packages (0 issues, first run) | 0 |
| `make protoc` under the guard | 0 |
| `make lint-protos`; `make lint-api` | 0; 2, with no finding in the three files I changed |
| `make umpire-rerecord-pinned-runs` | 0 |
| `go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...` after copying the main checkout's `render_test.go` (34 ok) | 0 |

The last two test-only edits (the presence-check row and a fail-fast test helper) were followed by the Testpilot suite, `-race -count=5` on `internal/execution` and its lint, all rc 0 (`final-12` to `final-14`). The model suite was not rerun after them.

Not run: any target that reaches `lake`, the `integration` live suites other than the two re-recording tests, `-race` over `tests/testcore/testpilot` (its known race).

### Files (relative to the repo root)

Protocol and generated: `proto/internal/temporal/server/api/testpilot/v1/{program,instruction,run}.proto`, `api/testpilot/v1/{program.pb.go,instruction.pb.go,instruction.go-helpers.pb.go,run.pb.go,run.go-helpers.pb.go}`.

Production: `common/testing/testpilot/contract.go`, `contract/profile.go`, `internal/execution/{evidence.go,program.go,response_read.go,scheduler.go,dataflow.go}`, `temporal/worker/{driver.go,interpreter.go,sdk.go}`.

Prose: `common/testing/testpilot/README.md`, `internal/execution/README.md`, `temporal/worker/README.md`.

Tests changed: `internal/execution/{activity_admission_test.go,activity_outcome_test.go}`, `temporal/{activity_run_test.go,catalog_test.go}`, `temporal/worker/{activity_fixture_test.go,activity_lifecycle_test.go,activity_transport_test.go}`.

Added: `common/testing/testpilot/internal/execution/evidence_source_test.go`, `internal/execution/activity_cancellation_test.go`, `temporal/activity_evidence_test.go`.

Re-recorded: `tools/umpire/replay/testdata/nexusCallerControl-forgedCompletion-run.json`, `tools/canary/assessment/testdata/nexusCallerCanary-syncCompletion-run.json`, `tools/umpire/evaluation/testdata/receipts/{accepted,incomplete,rejected}.json`.

`changed-files.txt` lists all 36. `tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog/render_test.go` in the workspace is the main checkout's copy and is not to be merged.

### Decisions that differ from the task text

1. **`instruction` and `run_keyed` were added** (see Read this first, 3). `instruction` names a controller instruction only; cleanup instructions are not supported.
2. **`single` is a bool on `ReadSource`**, not a new source arm, so existing reads keep their bytes and `ReadEvidence` binding is unchanged.
3. **Distinct kinds may share a source and key only where their recorded data differs.** The same record under a second key path stays admitted, as before. Two reads in one source and key stay refused, since one `ReadEvidence` names one declaration.
4. **An existing test's structure changed, not its assertions.** `TestRunRecordsWhatEachDeclaredActivityAttemptDid` now uses two extracted helpers (`activityScriptCase`, `runActivityScript`) that the new end-to-end test shares, and its fake server answers heartbeats and cancel requests.
5. **Expectations changed by intent**: `TestReservationOutcomesAreJudgedByOneClosedTable` counts 8 responses and 10 table rows, and the catalog golden holds the new identity.

### For tasks 19 and 21

- `Recorded.Single(method, path)` lowers to `ReadSource{method, path, single: true}`.
- `Recorded.RunEvent(kind, "controller", command, key = Run, guard)` lowers to `RunEventSource{kind, instruction: {controller entrypoint, command}, run_keyed: true, guard}` with an empty `operation`. A payload-path key goes in `operation` as before.
- A guard that compares `status` with an enum literal, or `sdk_attempt` and `delivery_id` as task 19's does, binds and evaluates. `Present(activity_attempt)` beside the typed comparisons is fine.
- `Instruction.AttemptCanceled` lowers to `activity_attempt_cancellation {}`. `DeriveProfile` authorizes the Opcode from the instruction. The Case must have a controller request the cancellation, or the attempt is refused when its context ends.
- `evidence.go:51` is no longer the refusal task 19 names; the two `delivered(...)` kinds are refused at run time for the reason in finding 1.

### Limits

- No live server has run a single read, a guarded Run Event declaration or the canceled answer. The canceled answer ran against in-process `WorkflowService` fakes with a real SDK client and worker.
- An attempt that gives no answer is not realized, as scoped.
- A lift whose spelled-out rules name two sources still counts them in one ordinal stream. That predates this task and I left it.
- A Run Event lift error on an instruction's own completion still drops that completion's events, as before. Only reservation records are kept.

stage: implement - ran (worker subagent, session model claude-opus-5-5)

Conductor: the first Codex review returned SHIP with no findings. The task was implemented in a directory copy (a machine restart wiped the first copy and all its work; the worker redid it and checkpointed into `.flow/tmp/fn-107/task20-checkpoint/`). Its 36 files were merged into the main checkout by path; none of them had changed there since the copy was made. On the merged tree `go build ./...` passes and `go test -short` over `./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/... ./model/go/... ./tools/umpire/... ./tools/canary/...` passes with no package failing, with no Lean stub on PATH (the Lean elaboration tests skip).

For task 21: one Run Event carries one evidence Observation, so two kinds cannot be lifted from one attempt event; use one kind with two projection outputs or disjoint guards. A canceled answer is sent to the server only for a delivery the server asked to cancel, so a Case needs a controller step that requests the cancellation. `RunEventSource` has `instruction` and `run_keyed`.

Open: the Lean-side mirrors of the new arm, enum value and fields are not written (no Lean toolchain, by the owner's decision).

The work is uncommitted; the owner makes the commits. The task diff and the review output are under `.flow/tmp/fn-107/task20/`.

stage: implement - ran (worker subagent in a directory copy, session model claude-opus-5-5; redone once after a machine restart)
stage: impl-review - ran (codex:gpt-5.6-sol:high, session 01a0f947-617b-7e43-a3e1-0231e28d2d75; round 1 SHIP, no findings)
stage: wave-join - ran (36 files copied by path into the main checkout; integrated gates rc 0)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/... ./model/go/... rc 0, 23 packages; guarded tools run rc 1 with only the three lake-stub tests failing), go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/... (rc 0), go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc 0), go test -tags test_dep -count=1 -race ./common/testing/testpilot/... (rc 0), go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... ./api/testpilot/... (rc 0), go vet -tags 'test_dep integration' ./tests/ (rc 0), go build ./... (rc 0), GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS='./common/testing/testpilot ./common/testing/testpilot/contract ./common/testing/testpilot/internal/execution ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/worker' GOLANGCI_LINT_FIX=false (rc 0, 0 issues), GOFLAGS=-tags=test_dep make protoc (rc 0; only api/testpilot/v1 changed), make lint-protos (rc 0); make lint-api (rc 2, no finding in program.proto, instruction.proto or run.proto), make umpire-rerecord-pinned-runs (rc 0; five records changed), go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/... (rc 0, 34 packages, with the main checkout's render_test.go), mutation by go test -overlay: 71 mutants, 71 killed, conductor, merged tree: go build ./... (rc 0), conductor, merged tree: go test -tags test_dep -count=1 -short ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/... ./model/go/... ./tools/umpire/... ./tools/canary/... (rc 0, no package failing, no Lean stub), conductor, merged tree: scoped make lint-code over testpilot, internal/execution, temporal/worker, GOLANGCI_LINT_FIX=false, codex impl-review: .flow/tmp/fn-107/task20/t20-r1.md (SHIP)
- PRs: