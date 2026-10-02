---
satisfies: [R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.21 Carry declared fields, off-path kinds and repeated classes through the generic producer; lower the activity Cases

## Description
**Touches:** [model/go/caseproducer/**, model/go/umpire/**, model/scalav2/goir/testpilot/**, model/scalav2/goir/conformance/**, model/scalav2/goir/load.go, model/scalav2/goir/*_test.go, model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/scala/temporal/nexuscaller/Realization.scala, model/scalav2/lifter/**, model/scalav2/ir/**, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/SEMANTICS.md, model/scalav2/README.md, model/scalav2/specimens/**, model/scalav2/run.sh]

Carry what the Scala realization declares through the generic Go producer into the Case, so the standalone activity's completion and retry Queries lower to admitted Cases and `syncCompletion` concludes. Review text with the constraints: `.flow/tmp/fn-107/task19/t19-r1.md` (judgments c, d, e).

**Size:** L

### Approach
- **Declared fields and off-path exhaustive kinds.** `model/go/caseproducer` carries only the kinds on a path and no evidence fields. Carry declared fields (path, type, disposition, role) and the exhaustive kinds a Case's closing read lifts even when the path does not record them, with field paths, types, dispositions and off-path rules inside the projection fingerprint. A realization that declares none of this produces byte-identical Cases; the Nexus caller Cases change because they gain declarations.
- **A class taken more than once on a path.** The producer confirms a class once per path, so `retry` (two `attemptStart` steps) is refused with `evidence.action-repeated`, and `pauseResume` would put one kind in two projection rules. "One rule per kind" is not sound: a confirmed rule's outputs are sequential effects of every occurrence, not alternatives. Give the generic producer a discriminant that makes each occurrence its own confirmation: distinct evidence kinds per occurrence, guarded projection outcomes, or occurrence- and state-aware projection, whichever is smallest and keeps unique-kind admission, transition authorization, Known Gap behaviour and deterministic fingerprints. Existing non-repeated Cases are unchanged.
- **Two things the task 19 review found waiting here.** `pauseResume` and `retry` each put the `statusScheduled` kind in two projection rules, which `testpilot.Prepare` rejects; an occurrence discriminant alone does not create the second stable scheduling observation the retry declaration lacks, so the realization must declare one (for example the retry's own typed attempt fact). Both must be proven with ordinary `testpilot.Prepare`. A gap that belongs to a command off a Query's path (the canceled answer, for one) must not block that Query: make unsupported commands path-specific without losing the declaration inventory. `startToCloseTimeout` needs an attempt that gives no answer and no instruction waits: record it as a limit, do not build it.
- **What task 20 delivered and its two constraints.** Testpilot now has a single-message read source, a Run Event evidence source under a typed guard (`RunEventSource` with `instruction` and `run_keyed`), distinct kinds sharing one dense source, and a canceled-answer instruction arm (handover: `.flow/tmp/fn-107/task20/summary.md`). One Run Event carries one evidence Observation: the activity realization's `delivered("statusStarted")` and `delivered("attemptCount")` cannot both lift from one attempt event; declare one kind with two projection outputs, or disjoint guards. A canceled answer reaches the server only for a delivery the server asked to cancel, so a Query that cancels needs a controller step that requests it. Lift the task-19 `unsupported` entries that named fn-107.20 by lowering to these primitives.
- **Stable activity evidence.** Re-key the standalone activity realization as the review requires: scheduling from the start instruction's outcome, attempt start and identity from the typed `activity_attempt` Run fact through task 20's Run Event source and guard, terminal states from single-message Describe reads. No transient-state poll remains.
- **Result.** The activity Model's completion and retry Queries lower to Cases `testpilot.Prepare` admits under a derived Profile; `syncCompletion` concludes on its witness Run and stays inconclusive on a history that holds a started event; the other six Nexus Properties stay listed as inconclusive with their reasons (action-class evidence, driven-party closed worlds, exhaustive retry attempts and enum discriminants are recorded limits, not work for this task).
- No feature policy in Go; typed and existing key-level behaviour of `model/go/umpire` and `caseproducer` unchanged apart from what the fingerprint must now cover.

### Investigation targets
**Required:** `.flow/tmp/fn-107/handover/task19-summary.md`, task 20's handover; `model/go/caseproducer/{producer,program,correlated,build,localize}.go`; `model/scalav2/goir/testpilot/{lower,realization}.go`; `model/scalav2/scala/temporal/standaloneactivity/Realization.scala`; `model/scalav2/goir/conformance/evidence.go`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; `GOFLAGS=-tags=test_dep make umpire-gen-scala`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `make lint-scala`; scoped `make lint-code` over `./model/scalav2/goir/... ./model/go/caseproducer ./model/go/umpire` with `GOLANGCI_LINT_FIX=false`. Nothing that calls `lake`.

## Acceptance
- [ ] The generic producer carries declared evidence fields and off-path exhaustive kinds, covered by the projection fingerprint; a realization that declares neither produces byte-identical Cases.
- [ ] A class taken more than once on a path is confirmed per occurrence by a generic discriminant; unique-kind admission, transition authorization and Known Gap behaviour hold; the comparative Go Model's existing Cases are unchanged.
- [ ] The standalone activity realization uses no transient-state poll; its completion, retry and pause/resume Queries lower to Cases ordinary `testpilot.Prepare` admits under a derived Profile, the failing attempt as `activity_attempt_failure`; identical inputs give identical bytes.
- [ ] `syncCompletion` concludes on its witness Run and is inconclusive on a history holding a started event, live and replayed alike; the six other Nexus Properties are listed as inconclusive with their reasons and none is narrowed.
- [ ] Inventories close: everything a realization declares is lowered or is a located `unsupported` entry naming its owner.


## Done summary
The generic producer now carries declared fields, off-path exhaustive kinds, the Run's own record and single reads, and confirms a class a path takes twice step by step. Six of the nine find Queries of the standalone activity Model lower to Cases that `testpilot.Prepare` admits under a derived Profile, and all six also run live through Testpilot's executor against a played Driver and replay to the same Verdict and assessment. `syncCompletion` concludes on its witness Run, live and replayed. Nothing is committed, the task stays `in_progress`, and no Testpilot protocol or runtime file changed.

### Read this first

1. **`cancel` and `cancelRequest` do not lower. They are located limits with no owner.** A Run records an attempt once it is answered (`internal/execution/scheduler.go`, `publishCompletion`). On the cancel path the cancel request's answer is recorded before the first attempt's record, so the Contract would meet the evidence out of the path's order and refuse it. The canceled answer itself lowers to `activity_attempt_cancellation` and is proven on the `errand.withdrawn` fixture.
2. **`pauseResume` spends 3,373,731 of the default Profile's 4,000,000 per-event Contract work on its fifth piece of evidence.** The projection cost grows with the cube of the evidence count. Before the Run-recorded kinds shared one evidence source the live Run cost 4,013,955 and became incomplete. A Case with six pieces of evidence of one operation will exceed `DefaultCeilings` (`common/testing/testpilot/temporal/profile.go`, outside my Touches).
3. **The realization keeps the worker stopped across a pause.** A path that pauses carries `stop-worker-until-released` before the start and `resume-worker` after the release. The task text does not ask for this. Without it the pause races the worker's first poll, and the release's answer is evidence of a scheduling that did not happen when the pause meets a held attempt.
4. **No lowered Case has run against a server.** The live Runs here use fake Drivers written against the public facade.
5. **I changed `go.sum` by accident and restored it.** A `GOFLAGS=-mod=mod go doc` call added 174 lines at 16:03. I wrote `git show HEAD:go.sum` back over it, `git status` shows it unmodified, and the Go suite and the scoped lint ran again after that (`final-09`, `final-10`).

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | Producer carries declared fields and off-path exhaustive kinds, inside the projection fingerprint. A realization that declares neither produces byte-identical Cases | Pass | `caseproducer`: `TestACaseDeclaresTheSourceAndTheFieldsOfEachKind`, `TestAnExhaustiveKindIsCarriedOffThePath`, `TestTheProjectionFingerprintCoversFieldsAndOffPathKinds` (8 changes), `TestTheProjectionFingerprintIsOfItsCanonicalForm`. Unchanged bytes: `nexuscaller.TestCasesAreByteIdenticalToTheFixtures` (7 fixtures), untouched |
| 2 | A class taken more than once is confirmed per occurrence. Unique-kind admission, transition authorization and Known Gap behaviour hold. Comparative Cases unchanged | Pass | `TestAClassTakenTwiceIsConfirmedOccurrenceByOccurrence`, `TestTheKindThatConfirmsAStepIsRefusedWhereItIsNotOne` (8 refusals, each also through `Preflight`), `TestAKindThatNamesAStepConfirmsItBeforeAKindThatRecordsItsFact`, `TestAnotherResultOfATakenRowIsConfirmedWithTheStepsBeforeIt`; admission: 8 new cases in `TestARealizationIsAdmittedBeforeItIsLowered`, `TestKindsThatNameTheirStepsMayRecordOneFact`; `TestALoweredCaseIsTheComparativeGoModelsCase` |
| 3 | No transient-state poll. Completion, retry and pause/resume lower to Cases `Prepare` admits under a derived Profile, the failing attempt as `activity_attempt_failure`. Identical inputs give identical bytes | Pass | `TestTheActivityRealizationPollsNoTransientState`, `TestEveryQueryOfTheActivityModelLowersOrNamesItsLimit` (6 Cases, 3 limits), `TestTheRetryCaseFailsItsFirstAttemptAndReadsItsSecondFromTheRunsRecord`, `TestAPathThatPausesKeepsItsWorkerFromPollingUntilTheRelease`, `conformance.TestALoweredActivityCaseRunsLiveAndReplaysAlike` (6 live Runs) |
| 4 | `syncCompletion` concludes on its witness Run and is inconclusive on a history holding a started event, live and replayed alike. The six other Nexus Properties stay inconclusive with their reasons and none is narrowed | Pass | `TestSyncCompletionConcludesOnItsWitnessRunLiveAndReplayed` (2 histories, the lowered Case run by Testpilot's executor), `TestAClosedHistorySettlesSyncCompletionAndLeavesTheOtherSixOpen` (7), `TestAWitnessRunConformsWhileItsPropertyStaysOpen` (7, unchanged) |
| 5 | Inventories close | Pass | `TestEveryFieldOfARealizationHasAPlaceInTheInventory`, the inventory check inside `TestEveryQueryOfTheActivityModelLowersOrNamesItsLimit`, `TestTheInventoryOfAnActivityCaseDoesNotCloseOverAChangedDeclaration` (8), `TestARecordWhoseInstructionTheCaseDoesNotCarryDoesNotClose`, `TestAKindOfEvidenceTheCaseDoesNotCarryAsDeclaredDoesNotClose` (4), `TestARedactedFieldIsNamedAsALimit` |

### The nine activity Queries

| Query | Standing | Evidence, in path order |
|---|---|---|
| `completion` | lowered | start's answer, attempt 1's record, COMPLETED |
| `nonRetryableFailure` | lowered | start's answer, attempt 1's record, FAILED |
| `retry` | lowered | start's answer, attempt 1's record, attempt 2's record (confirms the retried failure, the backoff and the second attempt start), COMPLETED |
| `pauseResume` | lowered | start's answer, PAUSED, release's answer, attempt 1's record, COMPLETED |
| `terminate` | lowered | start's answer, TERMINATED |
| `scheduleToStartTimeout` | lowered | start's answer, TIMED_OUT |
| `cancel`, `cancelRequest` | `unsupported`: "attempt record that follows later evidence", at `evidence.statusStarted`, owner none | |
| `startToCloseTimeout` | `unsupported`: "attempt that gives no answer", at the activity script, owner none | |

On the live Runs `completes` and `nonRetryableFails` are satisfied. `retryCompletes` is inconclusive because the claim fixes the three deadlines and the start's answer is one fact for all eight start classes. `terminated` is inconclusive because a second terminate after the first is not found, records nothing, and fails the claim. `scheduleToStartFires` is inconclusive because the three deadlines record one evidence name.

### What was built

- **Vocabulary and IR.** `Evidence.confirms: Vector[Taking]`, `Taking(step, occurrence)`; `ir.proto` gains `Evidence.confirms = 13` and message `Taking`. The lifter needed no change. `make protoc` changed only `api/modelir/v1`.
- **Producer (`model/go/caseproducer`).** `EvidenceSource` gains `Fields`, `Exhaustive` and `Confirms`; `Recorded` gains `Single` and `RunEvent`. `confirming` is the one decision of which kind confirms a step: the kind that names the step, then the kind that names no step and records the first fact of the step's class. A kind that names several steps confirms them all by one projection rule. `claim` keeps one rule per kind. Off-path exhaustive kinds are declared, lifted and given the meaning `IRRELEVANT`. `alternativeRules` now walks the rules by position.
- **Admission (`goir/load.go`).** A taking names a class the machine binds, counted from one, once per kind, by one kind. Kinds that name steps may share a fact. An exhaustive kind is the one kind of its fact.
- **Lowering (`goir/testpilot`).** Run Event sources, single reads, fields and the canceled answer lower to Testpilot's declarations. Gaps left: durable commit and hold-delivery (fn-107.10), monitors (fn-107.12), and three limits with no owner (redacted field, unanswered attempt, late attempt record). `producerLimit`, `behindTheLimit` and `unconfirmed` are deleted, so the producer's three later checks now run for a path that repeats a class.
- **Conformance.** `reader.read` calls `Admits` for a kind that is the Run's own record. Evidence on an event its source does not take is an `EvidenceError`, and a guard that cannot be evaluated stays a `GuardError`.
- **Activity realization.** The Run-recorded kinds count in one source. `statusStarted` is attempt 1's record and names `(attemptStart, 1)`. `attemptCount` is attempt 2's record and names `(attemptResult-failed-true, 1)` and `(attemptStart, 2)`. A new kind, the release's answer, records `statusScheduled` and names `(control-unpause, 1)`.

### Decisions that differ from the task text

1. The discriminant is a declaration, `confirms`. A step that records an evidence-named fact and is left with no kind is an error (`evidence.taking-unrecorded`), never a silent step.
2. `evidence.action-repeated` stays, as an error, for a class taken again that no kind names. A second new error, `evidence.kind-repeated`, refuses one unnamed kind confirming steps of two classes, which used to produce a Case `Prepare` rejects.
3. Every exhaustive kind is carried by every Case. The Nexus Cases gain 3 or 4 kinds each. `TestALoweredCaseIsTheComparativeGoModelsCase` compares the lowered Case whole with the comparative Case produced under that one declaration, and lists the kinds gained per Query from the checked-in fixtures.
4. `cancel` and `cancelRequest` are limits, items 1 and 3 above.
5. A redacted field is a gap with no owner. No lift carries a field without its value.
6. The fingerprint spells a field's type by its protocol name (`SCALAR_KIND_UINT64`). `String()` renders `Uint64`.
7. Tests removed with what they tested: `TestOnlyARepeatedClassIsALimitOfTheProducer`, `TestThePathsLastStepsAreConfirmedAsTheProducerConfirmsThem`, `TestSyncCompletionConcludesOnceItsCaseCarriesTheStartedEvent` and its helper `carryingStarted`. `TestTheActivityRealizationNamesWhatKeepsItFromACase` became `TestEveryQueryOfTheActivityModelLowersOrNamesItsLimit`.

### Findings and limits

- The conformance adapter reads a kind by the fact it records, and ignores `confirms`. It keeps more executions than the Contract's reading and rules out none that happened.
- The order of the last attempt's record and the terminal status read rests on recording order. The Driver records the attempt when it is answered and the poll reads the status later, but no causal link forces it.
- `late` infers the attempt a record is of from the last step its kind confirms. A record declared for a step that is no attempt start is not checked.
- The Lean mirrors of the realization vocabulary are not written. No Lean was run.
- The tally fixture names no `cleanup`, which every Program needs. The test that lowers it sets one.

### Test-first record

Red before the code, under `.flow/tmp/fn-107/task21-logs/`: `red-01` (compile) and `red-02` (6 producer tests), `red-03` (5 admission cases), `red-04` and `red-05` (9 activity Queries `unsupported`), `red-06` (4 guard-wiring cases), `red-07` (2 admission), `red-08` (5 late-record cases), `red-09` (2 inventory guard cases), `red-10` (2 pause cases). `conformance-01.log` shows `syncCompletion` flip to `satisfied` in the unchanged test once the producer carried the started kind.

Never red: both live tests, which passed on their first run apart from the two items below, and the tests added for surviving mutants.

Three expectations of mine were wrong and I corrected them from the source, each named here. `offPathKinds` listed `failed` as on the path for `syncCompletion` and `retry`; the checked-in fixtures show it is off. I expected `terminated` to be satisfied; the protocol machine answers a control of a finished activity as not found. One admission message spelled an action by its Definition ID where the IR uses the front end's qualified name.

The canonical-form test found the `String()` spelling of the field type, and the live `pauseResume` Run found the work ceiling.

### Mutation

86 mutants by `go test -overlay` on the final tree, 86 killed (`mutants-final-1..4.log`, script `mutants.py`). Sixteen survived a first pass. Fourteen got a test: the canonical form of an off-path rule, the steps before an alternative result, a record of another script, a flag field, a field's role, a payload-keyed record, five late-record cases, and three inventory cases. One showed a condition that could never matter, which I removed. One was killed by an existing test the script had not selected.

### Gates (final tree, exit codes)

| Command | rc |
|---|---|
| Baseline before editing: the Go suite below (57 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/...` (57 ok, after `go.sum` was restored) | 0 |
| `GOFLAGS=-tags=test_dep make umpire-gen-scala` (only `ir/activity.json` differs from the start) | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` (3 `[error]` lines, the refusals `run.sh` requires) | 0 |
| `make lint-scala` (0 `[error]` lines) | 0 |
| `go vet -tags test_dep ./model/...` | 0 |
| `make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/... ./model/go/caseproducer ./model/go/umpire' GOLANGCI_LINT_FIX=false` (0 issues; the first run reported 3, fixed by splitting `resolveEvidence` and `late`) | 0 |
| `make protoc` (the last run changed nothing; `api/testpilot/v1` is byte-identical to task 20's checkpoint) | 0 |
| `go test -race -short` over `goir/...`, `caseproducer`, `nexuscaller` | 0 |
| The two live tests, `-count=40 -race` (340 s) | 0 |

The Scala gates, `protoc`, `go vet` and the two race runs finished before I restored `go.sum`. The Go suite and the scoped lint ran again after it, and no source file changed in between.

Not run: any target that reaches `lake`, any live-cluster suite, `flowctl gate` receipts.

### Files

Before-copies of every directory in the Touches are under `.flow/tmp/fn-107/task21-before/`.

Added: `model/go/caseproducer/occurrence_test.go`, `model/scalav2/goir/testpilot/activity_cases_test.go`, `model/scalav2/goir/conformance/activity_test.go`, `model/scalav2/goir/conformance/played_test.go`.

Changed: `proto/internal/temporal/server/api/modelir/v1/ir.proto`, `api/modelir/v1/ir.pb.go`, `api/modelir/v1/ir.go-helpers.pb.go`; `model/go/caseproducer/{producer,program,correlated}.go`; `model/scalav2/goir/load.go`, `goir/realization_test.go`; `goir/testpilot/{lower.go,realization.go,activity_test.go,lower_test.go,inventory_test.go,random_test.go}`; `goir/conformance/{evidence.go,guard_test.go,nexus_test.go,fixtures_test.go}`; `model/scalav2/scala/umpire/realize/Realize.scala`, `scala/temporal/standaloneactivity/Realization.scala`; `model/scalav2/ir/activity.json`; `lifter/testdata/lifts/Realizations.scala.fixture` (three comment lines, same line count); `model/scalav2/SEMANTICS.md`, `README.md`, `specimens/activity.md`.

Unchanged: `model/go/umpire/**`, `model/go/nexuscaller/**`, `scala/temporal/nexuscaller/Realization.scala`, the lifter, `run.sh`, every Testpilot file.

### Review round 1

Finding (P2): `late()` read every `KIND_DIAGNOSTIC` source as a deferred attempt record, tried it against every activity script, and took its attempt from the last step it confirms. Fixed by declaring the timing and computing from the declaration.

**What changed**

- The realization says which attempt a record is of: `Recorded.RunEvent(..., attempt = Some(AttemptOf(script, number)))` in Scala, `RunEventSource.attempt` (`AttemptOf{position, script, number}`) in the IR. The lifter needed no change.
- `goir/load.go` admits it with located errors: a source that is no diagnostic, no script, a script the realization does not declare, a script no activity activates (the controller), an activity that starts with no delivery, a number below one. A number above the attempts a Query's path starts is a located error at lowering (`unstarted`), since it depends on the path.
- `goir/testpilot/lower.go`: `outOfOrder` is the one place the publication rule lives. A record of attempt n of script S reaches the Run at S's n-th answer on the path; any other evidence, diagnostic or not, is recorded by a controller instruction after the last step it confirms. Nothing is inferred from the last confirmed step.
- The lowered guard of a record is the authored guard and `activity_attempt.sdk_attempt == n`, so the Case selects the record by the declared attempt; `conformance.Admits` checks the same number.
- `caseproducer.Confirmations` tells, per kind, the places of the path steps it confirms; the detector reads those and no longer re-walks the path.

**Tests the review asked for**

| Requirement | Test |
|---|---|
| Unrelated diagnostics are not attempt records | `TestADiagnosticThatIsDeclaredNoRecordOfAnAttemptIsNotDeferred`; table row "a diagnostic that is no record of an attempt" |
| Several activity scripts, each record with its own script | three "two activities" rows of `TestARecordOfAnAttemptReachesARunWithTheAttemptsAnswer` |
| Second-attempt record whose last confirmation precedes `attemptStart#2` | `TestARecordIsLateByTheAttemptItIsDeclaredOf` (refused, located, owner none) |
| Admission errors | `TestTheAttemptARunEventRecordsIsOfAnActivitysScript` (7 cases), `TestARecordOfAnAttemptThePathNeverStartsIsAnError` |

Red first: `r1-red-01-admission.log` (six admission cases got no error), `r1-red-02-confirmations.log` (build fails, no `Confirmations`), `r1-red-03-round0-misses.log` (round-0 code, by overlay, lowers the reviewer's example). The lowering tests in `published_test.go` were written after the lowering; their red is the round-0 overlay and the mutants below, not a pre-implementation run. Three tests were added after the first mutant run for its survivors (the tie row, `TestARecordIsEarlyByTheAttemptItIsDeclaredOf`, `TestTheLoweredGuardOfAnAttemptsRecordStatesTheAttempt`).

**The six Cases**

`terminate` and `scheduleToStartTimeout` are byte-identical to round 0. `completion`, `nonRetryableFailure`, `retry` and `pauseResume` differ only in the guard of their attempt records, which now ends in `sdk_attempt == n`. Projection fingerprints are unchanged. Dumps: `task21-logs/r1-cases-before/`, `r1-cases-after/`.

**`cancel` and `cancelRequest`**

Both still have no Case, for the same reason through the new detector: `statusStarted` is declared attempt 1's record, the path answers attempt 1 after the cancel request's own evidence, so the record follows evidence of later steps. The limit is honest.

**Where publication order is assumed in `goir/testpilot`**

1. Attempt records: declared (`attempt`) and computed from the declared number.
2. Evidence a controller instruction records: checked on the produced Case by `ordered` — each kind has a recording instruction, and it runs after the previous kind's (honoring `After`). A Case that fails this is a located error.
3. Kinds one instruction records together (a history read): they reach the Run in the response's own order, which is the server's history order. Not assumed by the lowering.
4. Still assumed: a carrier call's completion is recorded before the record of the attempt that call starts.
5. Still assumed: an attempt's record is recorded before the status read that follows its answer.

Items 4 and 5 are orders between the controller and the worker inside the Testpilot runtime; no declaration in this task's Touches fixes them. A Run that breaks either is refused by the Contract and is incomplete, never given a wrong Verdict (`TestTheRetryContractRefusesItsEvidenceOutOfOrder`). They are named in the `lower.go` block comment and in SEMANTICS.md.

One imprecision: two kinds declared the record of the same attempt are refused (one Run Event carries one piece of evidence), with the "follows later evidence" reason, which describes it loosely.

**Mutation**: `python3 .flow/tmp/fn-107/task21-logs/r1-mutants.py` — 35 overlay mutants over this round's code, 35 killed (`r1-mutants-02.log`; the first run, `r1-mutants-01.log`, had two survivors and three that did not build).

**Gates, final tree** (logs `task21-logs/r1-final-*`)

| Gate | Exit |
|---|---|
| Go suite over the six trees | 0, 57 ok |
| `make umpire-gen-scala` | 0, IR unchanged by the rerun |
| `make umpire-check-scala` | 0, 3 required `[error]` lines |
| `make lint-scala` | 0, 0 `[error]` lines |
| `go vet -tags test_dep ./model/...` | 0 |
| scoped `make lint-code` | 0, no issues |
| `make protoc` | 0, idempotent; `api/testpilot`, `proto/.../testpilot` and `common/` equal the task-20 checkpoint |
| `-race -short` over `goir/...` and `caseproducer` | 0 |

`go.sum` and `go.mod` are unmodified. No commits; the task is `in_progress`.

**Files this round**: `ir.proto`, `api/modelir/v1/*.pb.go`, `scala/umpire/realize/Realize.scala`, `standaloneactivity/Realization.scala`, `ir/activity.json`, `goir/load.go`, `goir/realization_test.go`, `caseproducer/producer.go`, `caseproducer/occurrence_test.go`, `goir/testpilot/{lower.go,realization.go,published_test.go (new),activity_cases_test.go,activity_test.go}`, `goir/conformance/{guard.go,guard_test.go,activity_test.go}`, SEMANTICS.md, README.md, `specimens/activity.md`. Round-0 state of each is under `.flow/tmp/fn-107/task21-before/r1/`.

### Review round 2

The re-review left two P2 findings and one P3. All three are fixed, and the two remaining runtime orders now have direct regressions.

**Before you read the diff**

- HEAD moved during this round. The owner committed rounds 0 and 1 as `ffac2dd5ab wip`, then `d90379ad34 Update Model.scala`. That second commit replaced a blank line in `scala/temporal/standaloneactivity/Model.scala` (line 296) with the text `/goal`, which stops the Scala build (`task21-logs/r2-gen-scala-01-inherited.log`: "Illegal start of toplevel definition"). I put the blank line back in the working tree so the Scala gates could run. It is the only change in that file; discard it if `/goal` was meant.
- The round-2 diff is against `d90379ad34`. I made no commits.

**Finding 1 (P2): a diagnostic must name its attempt.** Admission (`goir/load.go`, `attemptOf`) now refuses a `KIND_DIAGNOSTIC` source with no `attempt`, located at the evidence: "is what a worker reports of an activation and is declared the record of no attempt". With round 1's rule that an `attempt` belongs only on a diagnostic, a diagnostic and an attempt record are the same thing. `NewRealizer` validates, so no Producer exists for such a Model.

Tests whose expectations were wrong and are now corrected:

- `TestADiagnosticThatIsDeclaredNoRecordOfAnAttemptIsNotDeferred` is replaced by `TestEvidenceTheControllerRecordsIsNotDeferred`. That test now makes the step's evidence the start call's own completion, and adds `TestADiagnosticDeclaredTheRecordOfNoAttemptHasNoProducer`.
- `TestARunEventSourceIsAdmitted` used to admit a diagnostic with no attempt on the Nexus realization. It now admits the same key and guard on a completion source.
- One conformance case ("a guard that cannot be evaluated") now uses a completion source.
- The outOfOrder table row for this case was renamed: "evidence the controller records in the record's place".

**Finding 2 (P2): two activities under one carrier.** A Run Event carries nothing that tells two activity scripts apart under one carrier, so such a Case is refused rather than bound.

- The diagnostic's coordinates are the carrier's (`scheduler.go`, `eventCoordinates(id.Origin)`), and its payload holds only `activity_run_id`, `sdk_attempt`, `delivery_id` and `response`.
- `source_id` (`<node>.r<declaration>.i<ordinal>`) does name the reservation, but a Testpilot guard reads only the payload.
- Testpilot's carrier reserves every activity entrypoint that has at least one instruction (`carrier.go`, `deriveReservations`), and refuses two carriers of one entrypoint.

The lowering now refuses, per Query, every attempt record of a realization whose Case gives instructions to two or more activity scripts. The new `indistinct` helper in `lower.go` gives each record a located gap, "attempt record among several activities" (owner none). `Admits` stays one check on the number, and its comment says why it cannot read the script.

Test: `TestTheAttemptsOfTwoActivitiesUnderOneCarrierAreNotToldApart`. The fixture is one start call carrying two activity scripts, the second performing the retryable failure. `retry` runs both and is refused with both records named. `completion` runs one and still lowers.

What would lift it: a field naming the reserved entrypoint on the record, readable by a guard, for example `ActivityAttempt.entrypoint_id`. A `RunEventSource` selector on the reservation's entrypoint would also work. Either is a Testpilot protocol change and is not made here.

**Finding 3 (P3): two records of one attempt get their own reason.** `outOfOrder` now returns a typed `clash` (`recordedLate`, `recordedEarly`, `recordedTwice`), and equal publication times are `recordedTwice`. `late()` names them "attempt recorded as two kinds of evidence": "a Run records attempt n of script S as one Run Event, which is evidence of one kind, and the path confirms steps by K as well". Tests: the "two records of one attempt" table row, and `TestAnAttemptRecordedAsTwoKindsOfEvidenceHasNoCase`, where `retry` has attemptCount declared attempt 1.

**The two assumed runtime orders.** `TestARunThatReversesAnOrderTheLoweringTakesFromTheRuntimeIsNeverSatisfied` (`goir/conformance/activity_test.go`) evaluates constructed Runs of the `completion` Case with `testpilot` `Evaluate`:

- In order, the Run is satisfied (control).
- The first attempt's record before its carrier's completion: evaluation fails at event 2, no satisfied Verdict.
- The terminal status read before the attempt's record: evaluation fails at event 3, no satisfied Verdict.

This uses constructed Runs, not a live fake-Driver Run: the fake Driver settles reservations through Testpilot's own scheduler, which publishes the carrier's completion first. The two tests passed on first run; they back the claim rather than fix anything.

**Red, then green**

| Log | What it shows |
|---|---|
| `r2-red-01-admission.log` | admission case got no error |
| `r2-red-02-lowering.log` | four lowering tests fail: the tie read as a late record, the Producer built, the fixture refused by admission (fixed in the test), and the old reason text |
| `r2-green-01.log` | all `goir/...` green |

The admission red ran before `load.go` changed. The lowering red ran with only the `clash` type added, behaviour unchanged.

**Mutation**: `r2-mutants.py`, all round-1 mutants (patterns updated) plus seven new ones. 42 of 42 killed (`r2-mutants-02.log`).

**Gates, final tree** (logs `task21-logs/r2-final-*`)

| Gate | Exit |
|---|---|
| Go suite over the six trees | 0, 57 ok |
| `make umpire-gen-scala` | 0; IR unchanged. The first run on the fixed line exited 2 on a stale compile-server diagnostic; the rerun is clean |
| `make umpire-check-scala` | 0, 3 required `[error]` lines |
| `make lint-scala` | 0, 0 `[error]` lines |
| `go vet -tags test_dep ./model/...` | 0 |
| scoped `make lint-code` | 0, 0 issues. The first run found cognitive complexity in `gaps`, and the check moved to `indistinct` |
| `make protoc` | 0, idempotent. `ir.proto` changed only in the comment on `attempt`, and only `api/modelir/v1/ir.pb.go` changed. Testpilot files equal the task-20 checkpoint |
| `-race -short` over `goir/...` and `caseproducer` | 0 |

`go.sum` and `go.mod` are unmodified, and the task is `in_progress`.

**Files this round**: `goir/load.go`, `goir/realization_test.go`, `goir/testpilot/{lower.go,published_test.go}`, `goir/conformance/{guard.go,activity_test.go}`, `scala/umpire/realize/Realize.scala` (doc lines, same count), `ir.proto` (comment) and `api/modelir/v1/ir.pb.go`, SEMANTICS.md, README.md, and the Model.scala line above. The round-1 state of each is under `.flow/tmp/fn-107/task21-before/r2/`.

`AGENTS.md` is also modified in the tree: the model routing now matches gomad's. I did not make that edit; it was already there when the user asked for it mid-round.

Conductor: review round 3 returned SHIP with no findings. Final gates on the tree: `go test -short` over `./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/...` and `make umpire-check-scala` pass.

Recorded limits: `cancel`, `cancelRequest` and `startToCloseTimeout` do not lower (owner none); a Query whose Case runs two activity scripts under one carrier command is refused, since a Run Event names no entrypoint (a protocol field such as `ActivityAttempt.entrypoint_id` would lift it); `pauseResume` uses 84% of the default per-event Contract work ceiling; six Nexus caller Properties stay inconclusive on their lowered Cases; no lowered Case has run against a server (task 9).

Rounds 0-1 are in the owner's commit `ffac2dd5ab`; round 2 and a one-line repair of `standaloneactivity/Model.scala` (a stray `/goal` line committed in `d90379ad34`) are uncommitted. Review outputs are under `.flow/tmp/fn-107/task21/`.

stage: implement - ran (worker subagent, claude-opus-5-5; two fix rounds)
stage: impl-review - ran (codex; rounds 1-2 on gpt-5.6-sol at high: NEEDS_WORK with one P2, then two P2 and one P3, all fixed; round 3 on gpt-6.1-sol at high after the routing change: SHIP, no findings; one session 01a0f9f3-b2ea-7101-8bdf-8518d2d43a01)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ffac2dd5ab
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... rc=0, 57 ok), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... (rc=0, 57 ok), GOFLAGS=-tags=test_dep make umpire-gen-scala (rc=0), GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0, 3 required [error] lines), make lint-scala (rc=0, 0 [error] lines), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/... (rc=0), GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/... ./model/go/caseproducer ./model/go/umpire' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), GOFLAGS=-tags=test_dep mise exec -- make protoc (rc=0, twice; only api/modelir/v1 changed), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -short ./model/scalav2/goir/... ./model/go/caseproducer/ ./model/go/nexuscaller/ (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=40 -race -run 'TestALoweredActivityCaseRunsLiveAndReplaysAlike|TestSyncCompletionConcludesOnItsWitnessRunLiveAndReplayed' ./model/scalav2/goir/conformance/ (rc=0), python3 .flow/tmp/fn-107/task21-logs/mutants.py (86 overlay mutants, 86 killed), review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... (rc=0, 57 ok), review round 1: GOFLAGS=-tags=test_dep make umpire-gen-scala (rc=0), review round 1: GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0, 3 required [error] lines), review round 1: make lint-scala (rc=0, 0 [error] lines), review round 1: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/... (rc=0), review round 1: GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/... ./model/go/caseproducer ./model/go/umpire' GOLANGCI_LINT_FIX=false (rc=0, no issues), review round 1: GOFLAGS=-tags=test_dep mise exec -- make protoc (rc=0, idempotent; only api/modelir/v1 changed by this task), review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -short ./model/scalav2/goir/... ./model/go/caseproducer/ (rc=0), review round 1: python3 .flow/tmp/fn-107/task21-logs/r1-mutants.py (35 overlay mutants, 35 killed), review round 1 red: .flow/tmp/fn-107/task21-logs/r1-red-01-admission.log, r1-red-02-confirmations.log, r1-red-03-round0-misses.log, review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/... ./tools/canary/... (rc=0, 57 ok), review round 2: GOFLAGS=-tags=test_dep make umpire-gen-scala (rc=0; inherited rc=2 from the /goal line of d90379ad34 before the blank line was restored), review round 2: GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0, 3 required [error] lines), review round 2: make lint-scala (rc=0, 0 [error] lines), review round 2: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/... (rc=0), review round 2: GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/... ./model/go/caseproducer ./model/go/umpire' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), review round 2: GOFLAGS=-tags=test_dep mise exec -- make protoc (rc=0, idempotent; only api/modelir/v1/ir.pb.go changed), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -short ./model/scalav2/goir/... ./model/go/caseproducer/ (rc=0), review round 2: python3 .flow/tmp/fn-107/task21-logs/r2-mutants.py (42 overlay mutants, 42 killed), review round 2 red: .flow/tmp/fn-107/task21-logs/r2-red-01-admission.log, r2-red-02-lowering.log, conductor final: go test -tags test_dep -count=1 -short over the six package trees (rc 0), conductor final: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), codex impl-review: .flow/tmp/fn-107/task21/t21-r1.md, t21-r2.md (NEEDS_WORK), t21-r3.md (SHIP)
- PRs: