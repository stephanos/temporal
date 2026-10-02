---
satisfies: [R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.19 Author the activity realization and the evidence declarations conformance needs

## Description
**Touches:** [model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/scala/temporal/nexuscaller/Realization.scala, model/scalav2/lifter/**, model/scalav2/ir/**, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/goir/load.go, model/scalav2/goir/testpilot/**, model/scalav2/goir/conformance/**, model/scalav2/goir/*_test.go, model/scalav2/SEMANTICS.md, model/scalav2/README.md, model/scalav2/specimens/**, model/scalav2/run.sh]

Give the Scala realization vocabulary what tasks 6, 7 and 13 found missing, author the activity realization, and lower it, so task 9 has an activity Case to run and the conformance adapter can reach conclusions. No Testpilot runtime or protocol change here: the protocol additions are task 13's and are consumed as they are.

**Size:** L

### Approach
- **Activity realization.** Author a realization for the standalone activity Model in Scala beside the Model (start, completion, retry through a failing first attempt, pause/unpause as far as the public API allows), with its worker script. A failing attempt lowers to the `activity_attempt_failure` instruction arm, never to a `Finish` whose result is a `Failure`. The lowered start assigns `namespace` and `task_queue.name` from the worker and queue role bindings and sets `activity_id` and `activity_type.name` (task 13 handover). Lift the "activity activation" gap in `goir/testpilot/lower.go`; what Testpilot still cannot run (hold-delivery, durable-commit observation: task 10; authored monitors as Contract rules) stays a located `unsupported` entry naming its owner.
- **Evidence roles.** Let a retained evidence field declare its role (operation, attempt, delivery), lift it, admit it, and carry it into the lowered Case; `goir/conformance/evidence.go` then reads task 13's typed `InstructionOutcome.activity_attempt { activity_run_id, sdk_attempt, delivery_id, response }` and the declared roles, at the places task 7's handover lists, and stops refusing a Case that retains a field.
- **Exhaustive sources.** Add the declaration task 7 specified: an evidence kind may be declared exhaustive and a read command may close it (`Realize.scala`, `ir.proto`, admission refusing `exhaustive` on a kind no command closes, the lowered Case carrying which kinds are exhaustive and which instruction is each one's closing read). The conformance adapter then infers absence only for an exhaustive kind whose closing read succeeded with dense source ordinals on a Run that closed complete; without the declaration it still infers nothing.
- **Discriminating Nexus evidence.** Apply the table in task 7's handover to the Nexus caller realization so the seven Properties can conclude on their witness Runs: the started event carried with its history source exhaustive, a failure-kind role or separate kinds, completion-result evidence, distinct timeout kinds, a typed attempt count. Where a Property still cannot conclude, say why and leave it inconclusive; do not narrow a claim to make it pass.
- Derive every expectation from the specimens and the spec. Byte parity with the old Case fixtures is not required; identical inputs still give identical bytes.

### Investigation targets
**Required:** `.flow/tmp/fn-107/handover/task6-summary.md`, `task7-summary.md` (exhaustive declaration spec; Nexus evidence table; where typed fields plug in), `task13-summary.md` (protocol diff; what task 9 must lower); `model/scalav2/scala/umpire/realize/Realize.scala`; `model/scalav2/goir/testpilot/{lower,realization,descriptor}.go`; `model/scalav2/goir/conformance/{evidence,candidates,conclude}.go`; `model/scalav2/specimens/{activity,nexus}.md`; `proto/internal/temporal/server/api/testpilot/v1/{instruction,run}.proto`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; `GOFLAGS=-tags=test_dep make umpire-gen-scala`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `make lint-scala`; `GOFLAGS=-tags=test_dep mise exec -- make protoc` when `ir.proto` changes; scoped `make lint-code` over `./model/scalav2/goir/...` with `GOLANGCI_LINT_FIX=false`. Nothing that calls `lake`.

## Acceptance
- [ ] The standalone activity Model has a Scala realization that uses no transient-state poll: scheduling is keyed to the start instruction's outcome, attempt start and identity to the typed `activity_attempt` Run fact, terminal states to Describe reads. Activity-script lowering is proven on a fixture whose Case ordinary `testpilot.Prepare` admits under a derived Profile, with the failing attempt as an `activity_attempt_failure` instruction and the start carrying namespace, task queue, activity id and type from the declared bindings; identical inputs give identical bytes. Every Query of the real activity Model is either lowered or a located `unsupported` entry naming the task that owns the missing primitive (fn-107.20 for Testpilot evidence sources, fn-107.21 for the generic producer, fn-107.10 for race controls). Lowering the real completion and retry Cases is fn-107.21's acceptance.
- [ ] A retained evidence field declares its role in Scala; a crossed attempt or delivery cannot explain a step in the conformance adapter, and a Case that retains fields is assessed, not refused. Task 13's typed attempt identity is read as typed data.
- [ ] An evidence kind can be declared exhaustive with a closing read; absence is inferred only under that declaration with a successful closing read, and a stale-design violation is shown from commit evidence on such a Run, live and replayed alike. Without the declaration nothing is inferred.
- [ ] Each of the seven Nexus caller Properties is listed with whether it concludes on its lowered Case and, where it does not, the reason and what would settle it; no claim was narrowed to pass. (`syncCompletion` concluding on a lowered Case is fn-107.21's acceptance.)
- [ ] What Testpilot cannot run yet is a located `unsupported` entry naming its owning task; nothing declared is silently dropped (inventory tests still close).
- [ ] No Testpilot protocol file changes; legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint pass.
## Done summary
Task 19 delivers the realization vocabulary, its admission, activity-script lowering and the conformance reading of roles, typed attempt identity and exhaustive sources. It also delivers a standalone activity realization that polls no transient state. Lowering the real activity completion and retry Cases is fn-107.21's acceptance, and Testpilot's evidence sources are fn-107.20's. Every find Query of the real activity Model is `unsupported` today, and each entry names the task that owns what is missing. Nothing is committed, the task stays `in_progress`, and no Testpilot protocol file changed.

**Where sections disagree, the latest review round is current: "Review round 3", then "Review round 2", then "Review round 1".** Round 0 named fn-107.10 and fn-107.19 as owners of the evidence-source and producer gaps, read every activity status from Describe, and refused `retry`, `cancel` and `cancelRequest` as errors. Round 1 replaced all three.

### Acceptance (as rescoped)

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | Activity realization with no transient-state poll; script lowering proven on a fixture Case `testpilot.Prepare` admits; every real Query lowered or a located `unsupported` entry naming its owner | Pass | `TestTheActivityRealizationPollsNoTransientState`, `TestAnActivityScriptLowersToItsAttemptsInOrder`, `TestTheActivityRealizationNamesWhatKeepsItFromACase` (9 Queries, owner asserted per entry) |
| 2 | A field declares its role; a crossed attempt or delivery explains no step; a Case that retains fields is assessed; typed attempt identity is read | Pass in the adapter. No lowered Case retains a field until fn-107.21 | `TestEvidenceIsAssessedLiveAndReplayedAlike` (6 identity rows), `TestTheTypedAttemptOfARunEventIsItsEvidencesIdentity` (9 cases), `TestPrepareBindsTheFieldsAndTheClosingReadsOfACase`, `TestEvidenceCarriesItsRetainedFieldsWithTheirValues` |
| 3 | Exhaustive kind with a closing read; absence inferred only then; stale-design violation from commit evidence, live and replayed | Pass | `TestEvidenceIsAssessedLiveAndReplayedAlike/A1_stale_design,_the_commit_source_exhaustive_and_closed` (violated) and five sibling rows, `TestAnExhaustiveSourceIsClosedOnlyByASuccessfulReadOfUnbrokenOrdinals` (8 cases), `TestASourceIsClosedByOneRule` (12 combinations), the unchanged `TestNoAbsenceIsInferredWithoutAnExhaustiveDeclaration` |
| 4 | Each Nexus Property listed with whether it concludes, why not, and what would settle it | Pass. None concludes on a lowered Case, and none was narrowed | `TestAClosedHistoryRulesOutOnlyWhatTheCaseCarries` (7), `TestSyncCompletionConcludesOnceItsCaseCarriesTheStartedEvent` |
| 5 | What Testpilot cannot run is a located `unsupported` entry; inventories close | Pass | `TestWhatNoCaseCarriesYetIsNamedWithItsOwner`, `TestWhatTestpilotCannotRunIsNamedWithItsOwner`, `TestEveryFieldOfARealizationHasAPlaceInTheInventory` (walks `Evidence` too), `TestAnExhaustiveKindIsCarriedWithItsClosingRead`, `TestAKindOfEvidenceACaseCannotCarryDoesNotClose` |
| 6 | No protocol change; legacy unchanged; generation, lifting, tests and lint pass | Pass | Gates below. Of 140 files under `api/`, two changed, both in `api/modelir/v1` |

### The seven Nexus Properties

The realization declares the five history kinds exhaustive and the `history` read their closing read. The seven Cases are byte-identical to before. Each Property stays inconclusive on its witness Run, with and without the read's success recorded.

| Case | Why it stays open | What would settle it |
|---|---|---|
| `syncCompletion` | A completion of an unanswered operation records started and completed, and the Case does not carry the started kind | fn-107.21 carrying exhaustive kinds off the path. With the started kind added to the Case by hand, the witness Run is `satisfied`, and a history that holds a started event is `inconclusive` while the Contract stays satisfied |
| `handlerError` | A failed reply and a non-retryable handler error record the same fact | Evidence that names an action class. The vocabulary has none |
| `asyncCompletion`, `asyncFailure` | A late completion is not found and records no fact, so no source reports it | Evidence of an action's outcome, or a closed-world declaration for a driven party |
| `retry` | The pending-attempts poll stops at its first match, so a second retryable failure can go unseen. The claim also fixes all three deadlines, and the scheduled fact is the same for all eight schedule classes | An exhaustive attempt source and deadline evidence |
| `scheduleToStartTimeout`, `startToCloseTimeout` | The three deadlines record one fact name, and the timeout type is an enum, which Testpilot does not admit as an evidence field (`dataflow.go:475`) | Enum evidence fields in Testpilot, and a role tying a field to a fact's own field |

The review recorded the last six as explicit limits of this prototype.

### What was added in round 0

- **Vocabulary and IR** (`Realize.scala`, `ir.proto`): `Evidence.fields` with `EvidenceField(id, path, role, redacted)` and roles operation, attempt and delivery; `Evidence.exhaustive`; `Command.closes`; `Instruction.AttemptFailure`; `Recorded.Single` for the one message of a response; `Activation.Activity(..., starts)` for the classes an attempt's delivery is. The lifter needed no change, since it emits declarations by name.
- **Admission** (`goir/load.go`): 14 located errors over 18 table cases, among them an exhaustive kind no command closes, a role on a redacted field, two fields of one role, a kind closed twice or by a command that does not read it, a closing read only some Cases carry, and an attempt failure outside an activity script.
- **Lowering** (`goir/testpilot`): an activity script lowers to the entrypoint's attempts in path order, and the "activity activation" gap is gone. Lowering refuses, where written, what Testpilot would refuse later: a non-answer in an activity script, a failure that is neither an application failure nor of no kind, and a poll condition that reads the Run. An exhaustive kind a Case carries must come with its closing instruction.
- **Conformance** (`goir/conformance`): a retained field is read under the role the realization gives it. The `activity_attempt` a Run Event records beside evidence supplies that evidence's attempt and delivery, and a field and the event must agree. One operation is one activity run. `sourceClosed` is the single rule for inferring absence.

### Decisions that differ from the task text

1. **The exhaustive marker is read from the Model, not from the Case.** The Case protocol has no field for it. The factory binds the Model's identity, and `Prepare` refuses a Case that carries an exhaustive kind without the instruction that closes it. The review accepted this.
2. **A field has a `redacted` flag.** The "role on a field the kind does not retain" error needs a way to not retain.
3. **`attemptStart` is the script's activation.** `Activation.Activity.starts` names it, so the path's answers are the only instructions and their count is the attempt count.
4. **`response` is not read.** It is what the worker offered, and no kind of evidence gives it a fact. The review accepted this.
5. **Expectations changed by intent**, all accepted by the review: `TestEveryQueryHasAStanding` no longer lists `activity`; `TestWhatTestpilotCannotRunIsNamedWithItsOwner` lost its "activity activation" row; the conformance refusal of an undeclared retained field has new text; `bound_test.go` counts 19 witnesses where it counted 17.

### Findings

- `specimens/activity.md` said Describe statuses were admissible through `ReadEvidence`. They are not, and the row is corrected.
- `startToCloseTimeout` needs an attempt that gives no answer, and no instruction waits. Its entrypoint would have no instruction, and Testpilot refuses an undeclared attempt. No declaration says this, so no gap entry names it.
- A poll's `until` cannot read the Run, so the `errand` fixture's listing polls select by status alone. The fixture says so.
- `model/scalav2/backends/`, `goir/composed_test.go` and edits to `README.md`, `goir/claims.go` and `goir/compose.go` appeared in the checkout at 09:00 during round 0. I took them to be task 8's merge and left them alone. My before-copies were taken after them.

### Test-first record, round 0

- Admission: 19 of 20 new cases failed before `load.go` changed, for example `an_exhaustive_kind_no_command_closes` with "An error is expected but got nil" (`red-01-admission.log`).
- Lowering: 7 tests and 15 subtests failed, for example `TestAnActivityScriptLowersToItsAttemptsInOrder` with "command fail-attempt is no instruction this reader lowers" (`red-02-lowering.log`).
- Conformance: against type-only stubs, 9 tests failed, for example `A1_stale_design,_the_commit_source_exhaustive_and_closed` (`red-03-conformance.log`).
- Every expectation was written from the specimen and the kernel before its first run. All 12 new assessment rows, the 9 typed-attempt cases, the 7 closed-history rows and the 3 carried-started cases matched on their first run against the implementation.
- Written after the code and never red: the Nexus tests, `TestAKindOfEvidenceACaseCannotCarryDoesNotClose`, `TestASourceIsClosedByOneRule` and `TestTheOrdinalsOfASourceAreUnbroken`.
- Mutation by `go test -overlay`: 54 mutants, 53 killed. The survivor showed a redundant event-kind check in `assessor.completion`, which I deleted.

### Review round 1

The review found one P1 and no other P0 to P2 defect. The P1 is fixed test-first, the gap owners are re-pointed, and every real activity Query is now a located `unsupported` entry and no longer an error.

### The P1: transient polls

`await-scheduled` and `await-started` polled `run_state` for `SCHEDULED` and `STARTED`, which a running worker moves past before a poll need see them. `await-cancel-requested` had the same flaw. All three are gone.

- Red first (`r1-red-01-transient.log`): `TestTheActivityRealizationPollsNoTransientState` failed at `activity_test.go:397`, expected `"status"`, actual `"run_state"`.
- The realization now keys each fact to something that stays true:

| Fact | Evidence | Why it is stable |
|---|---|---|
| `statusScheduled` | Run Event, the `start-activity` command's completion, guard `status == SUCCEEDED`, key the run | The Run records it once |
| `statusStarted`, `attemptCount` | Run Event, the worker's report of an activation `start-activity` carries, guard `present(activity_attempt.sdk_attempt)` and `present(activity_attempt.delivery_id)`, key the run. Fields: `attempt` (role attempt), `delivery` (role delivery), `activityRun` | A delivered attempt has both. A `NOT_NEEDED` record has neither and is excluded. No guard or field reads `response` |
| `statusCancelRequested` | Run Event, the `request-cancel-activity` command's completion | The Run records it once |
| `statusPaused` | Describe `info`, `status == PAUSED` | The controller alone leaves it |
| The five terminal statuses | Describe `info`, terminal `status` | Terminal |

- The test compares each Run Event source and each field list whole, with positions removed, and requires every poll to read `status` with one of six stable values.

### Vocabulary added this round

- `Recorded.RunEvent(kind, script, command, key, guard)` with `EventKind` (instruction completed, instruction timed out, diagnostic). `key` is the run's id or a path of the payload. `guard` is a condition over the payload, the instruction outcome. A Run Event's evidence leaves `Evidence.operation` empty.
- `Operand.All`, which the delivered-attempt guard needs. I had added and removed it in round 0 when its only use turned out to be inadmissible.
- `Instruction.AttemptCanceled`, so the worker's canceled answer is a declaration with a gap entry. Round 0 left the class unperformed, which made `cancel` and `cancelRequest` errors.
- Admission, red first (`r1-red-02-admission.log`, 11 cases and `TestARunEventSourceIsAdmitted` failing): a Run Event of no known kind, of an undeclared script or command, with a key that is neither the run nor a payload path, with a guard that reads the run, its environment or a learned value, or with an operation path beside its key; a poll of a Run Event; a canceled answer outside an activity; a conjunction of no operand.
- Lowering checks the guard, the key and the fields against `InstructionOutcome` and refuses a path the payload does not have.

### Gap owners

| Construct | Owner |
|---|---|
| Run Event evidence source; single-message evidence read | fn-107.20 |
| Retained evidence field; class confirmed more than once on a path | fn-107.21 |
| Canceled attempt answer (moved here in round 2: the protocol arm is fn-107.20's, its lowering fn-107.21's) | fn-107.20 |
| Durable-commit observation; hold-delivery control and commands | fn-107.10 |
| Authored monitor | fn-107.12 |

Round 1 named fn-107.10 for the canceled answer. The coordinator assigned it to fn-107.20 in round 2, and the code and tests name fn-107.20.

`evidence.action-repeated` is now a gap. `producerLimit` (`lower.go:466`) classifies that one refusal of `cp.Preflight` as a limit of the producer, and every other refusal stays an error (`TestOnlyARepeatedClassIsALimitOfTheProducer`, 5 cases). Round 2 checks one of the producer's later refusals behind this limit and names the three that stay unchecked. See "Review round 2", finding 2.

### For fn-107.20: what it must make pass

Each row is refused today at `model/scalav2/goir/testpilot/lower.go:288` (Run Event) or `:292` (single read). The test that flips is `TestTheActivityRealizationNamesWhatKeepsItFromACase`: the fn-107.20 entries leave every Query's list. `TestWhatNoCaseCarriesYetIsNamedWithItsOwner` flips the same way for the `tally` fixture.

| Declaration (`standaloneactivity/Realization.scala`) | What Testpilot lacks today |
|---|---|
| `accepted("statusScheduled", startActivity)`, `:73` | A Run Event source limited to one command's events under a payload guard. `evidence.go:89-105` binds a kind and a constant-true guard, and `:313-343` lifts every event of the kind |
| `accepted("statusCancelRequested", requestCancelActivity)`, `:73` | The same |
| `cancel-attempt` of the activity script (`AttemptCanceled`), refused at `lower.go:274` | An activity entrypoint instruction that answers its attempt as canceled. It blocks `cancel` and `cancelRequest` alone. Lowering the arm once it exists is fn-107.21's |
| `delivered("statusStarted")` and `delivered("attemptCount")`, `:97` | The same, under the typed guard of round 2 (attempt set, `sdk_attempt > 0`, `delivery_id` not empty), with a key that is not a payload path (the run's id), and two kinds lifted from one event. `evidence.go:51` refuses two declarations with one source and key path |
| `status("statusPaused")` and the five terminal statuses, `:53` | A read source that ends in one message. `evidence.go:121-127` requires a repeated field |

After fn-107.20, `gaps` in `lower.go` drops both entries, `adapter.source` in `realization.go` translates the two sources, and `evidenceFields` in `lower.go` accounts for `run_event` and `single` in the inventory, where both are errors today.

### For fn-107.21: what it must make pass

| Declaration | Refused today at | Test that flips |
|---|---|---|
| The three fields of `delivered(...)`, `Realization.scala:118-120` | `lower.go:300`. `caseproducer.EvidenceSource` (`producer.go:74`) has no fields, and `projectionCanonical` writes `[]` in the field slot | `TestTheActivityRealizationNamesWhatKeepsItFromACase`: the six field entries leave every Query's list. `TestAKindOfEvidenceACaseCannotCarryDoesNotClose/a_field` then needs a new expectation |
| `retry`, whose path takes `attemptStart` twice | `producer.go:419`, classified at `lower.go:466` | The same test, `retry` subtest: the last entry leaves. `TestOnlyARepeatedClassIsALimitOfTheProducer` goes with `producerLimit` |
| The exhaustive `started` kind, off the `syncCompletion` path | Not refused. `evidenceDeclarations` (`program.go:142`) declares the kinds of the rules alone | `TestAClosedHistoryRulesOutOnlyWhatTheCaseCarries/syncCompletion` becomes `satisfied`, and `carryingStarted` in `nexus_test.go` is deleted. `TestALoweredCaseIsTheComparativeGoModelsCase` changes with it |

Two things fn-107.21 meets that no entry names today, because the gaps above come first:

- `pauseResume` confirms `statusScheduled` for both `start` and `control-unpause`, and `retry` for `start` and the retryable failure. One kind in two projection rules is refused by `testpilot.Prepare` (`correlated_prepare.go:180`). The unpause and the retryable failure have no stable evidence of their own, since scheduled-again is transient. Each of the two Queries therefore needs a second stable scheduling observation, and fn-107.21 proves the result with ordinary `testpilot.Prepare`.
- `startToCloseTimeout` needs an attempt that gives no answer. No instruction of an activity entrypoint does that, and no task owns it. It is a recorded limit of the prototype.
- The review's point 3: the five Nexus history kinds share one source and key path, which `evidence.go:51` refuses once more than the path's kinds are declared. That part is fn-107.20's.

### Test-first record, round 1

- Red before the fix: the transient-poll test and the 12 admission tests, as above.
- `TestTheActivityRealizationNamesWhatKeepsItFromACase` was rewritten to assert a construct, id and owner per entry for all nine Queries. I wrote its expectation from `Realization.scala`'s declaration order before running it, and it matched on its first run. It was not run red against the round-0 code.
- Mutation by overlay on the final code: 73 mutants across both rounds, all killed. One round-1 mutant survived its first run (a key that is a path of something other than the payload). I added the admission case that kills it.

### Gates (final tree)

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (23 packages ok) | 0 |
| `... go test -tags test_dep -count=1 -race -short -v ./model/scalav2/goir/conformance/ ./model/scalav2/goir/testpilot/` (405 passing, no skip, no race) | 0 |
| `GOFLAGS=-tags=test_dep make umpire-gen-scala` | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` (3 `[error]` lines, all the refusals `run.sh` requires) | 0 |
| `make lint-scala` (0 `[error]` lines) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false` (0 issues; the first run reported 2 complexity findings, fixed by splitting `source` and `operand`) | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep mise exec -- make protoc` (twice; the second run changed nothing; only `api/modelir/v1` differs from the start of the task) | 0 |

The Scala gates ran before the last Go-only change, the split of `source` and `operand`. The Go suite, vet, lint and the mutation run are after it. The full `-race` run without `-short` was made in round 0 only (925 passing, 5 Lean-dump skips), since it takes 9 minutes.

Not run: any `tools/umpire` or `tools/canary` test, any target that reaches `lake`, any live-cluster suite, and `flowctl gate` receipts. No lowered Case has run against a server.

### Files this round

The round-0 state of every file is under `.flow/tmp/fn-107/task19-before/r1/`, first-touch copies under `task19-before/`, and logs under `task19-logs/` with the prefix `r1-`.

Changed: `ir.proto`, `api/modelir/v1/ir.pb.go`, `ir.go-helpers.pb.go`, `Realize.scala`, `standaloneactivity/Realization.scala`, `Realizations.scala.fixture`, `expected/realizations.json`, `ir/activity.json`, `goir/load.go`, `goir/realization_test.go`, `goir/testpilot/{lower.go,realization.go,descriptor.go,activity_test.go,inventory_test.go}`, `SEMANTICS.md`, `README.md`, `specimens/README.md`, `specimens/activity.md`.

Unchanged this round: `goir/conformance/**`, the Nexus realization and `ir/nexus-caller.json`.

### Files, both rounds

Added:
- `model/scalav2/scala/temporal/standaloneactivity/Realization.scala`
- `model/scalav2/goir/testpilot/activity_test.go`
- `model/scalav2/goir/conformance/identity_test.go`

Changed:
- `proto/internal/temporal/server/api/modelir/v1/ir.proto`, `api/modelir/v1/ir.pb.go`, `api/modelir/v1/ir.go-helpers.pb.go`
- `model/scalav2/scala/umpire/realize/Realize.scala`, `model/scalav2/scala/temporal/nexuscaller/Realization.scala`
- `model/scalav2/lifter/testdata/lifts/Realizations.scala.fixture`, `expected/realizations.json`, `model/scalav2/ir/activity.json`, `model/scalav2/ir/nexus-caller.json`, `model/scalav2/run.sh`
- `model/scalav2/goir/load.go`, `realization_test.go`, `bound_test.go`
- `model/scalav2/goir/testpilot/lower.go`, `realization.go`, `descriptor.go`, `fixture_test.go`, `inventory_test.go`, `lower_test.go`
- `model/scalav2/goir/conformance/evidence.go`, `assessor.go`, `candidates.go`, `plan.go`, `conformance.go`, `assessment_test.go`, `closing_test.go`, `conformance_test.go`, `fixtures_test.go`, `nexus_test.go`
- `model/scalav2/SEMANTICS.md`, `README.md`, `specimens/README.md`, `specimens/activity.md`, `specimens/nexus.md`

`README.md` changed in three places only, all lines my work made untrue. `goir/claims.go` and `goir/compose.go` were not touched by this task.


### Review round 2

All four findings are fixed, each red before the fix, and every gate passes on the final tree. The task stays `in_progress` with no commit.

### Finding 1 (P1): the delivered-attempt guard

The guard of `delivered(...)` (`standaloneactivity/Realization.scala:107-113`) is now `All(Present(activity_attempt), Greater(activity_attempt.sdk_attempt, 0), Not(Equal(activity_attempt.delivery_id, "")))`. The vocabulary gained `Operand.Greater` and `Operand.Not`, and `ir.proto` gained `Operand.greater = 11` and `Operand.not = 12`. Both lift, are admitted, and lower to Testpilot's `CompareExpression` and `NotExpression`. The errand fixture uses all three operands in a poll, and its Case passes ordinary `testpilot.Prepare`.

`conformance.Admits(source, event)` (`goir/conformance/guard.go:42`) evaluates a Run Event source against a Run Event over typed values. Presence is of a message or a oneof member, so a scalar at its proto3 default is present. A guard that cannot be evaluated returns a `*GuardError` and never `false`. A value the payload does not hold keeps the type of its descriptor, so a type error is an error on every event, and only the absence of a compared value depends on the event.

- Red: `r2-red-01-guard.log`. Three subtests of `TestOnlyADeliveredAttemptIsEvidenceOfAnAttemptStart` failed with `expected: false, actual: true`: the `NOT_NEEDED` position, `sdk_attempt = 0` with a delivery, and an attempt number with no delivery.
- Red: `r2-red-04-guard-absence.log`. Seven subtests failed while the evaluator still returned an untyped absent value, so a wrong path or type under an unset message read as `false`.
- Green: `TestOnlyADeliveredAttemptIsEvidenceOfAnAttemptStart` (9 events, both attempt kinds, read from `ir/activity.json`), `TestOnlyASucceededCallIsEvidenceOfItsAnswer` (8), `TestAGuardIsEvaluatedOverTypedValues` (20 cases and 2 events no source takes), `TestAGuardThatCannotBeEvaluatedIsAnError` (34 cases and a source of no known kind).

`Admits` is not wired into the assessment. No Case carries a Run Event kind until fn-107.20, so nothing would call it. fn-107.20 or fn-107.21 wires it where the lift happens.

### Finding 2 (P2): an error behind the producer's limit

`check` (`goir/testpilot/lower.go:392`) still classifies `evidence.action-repeated` as the limit, and then runs `behindTheLimit` (`:417`). That reports the producer's next refusal that this package can decide alone: a path that ends in steps that record nothing evidence names (`evidence.action-unmapped`). The rule is the producer's own (`caseproducer/producer.go:360-425`), isolated in `unconfirmed` (`lower.go:439`). Precedence stays in `standingOf`: the error wins over the gap.

Three producer checks stay unvalidated for a path that repeats a class, until fn-107.21 removes the limit. This package cannot make them without the producer.

1. Every other result of a row the path takes is recorded by a kind no other rule claims (`evidence.kind-ambiguous`, `evidence.kind-unknown`).
2. Each clause of the Property is placed on the schedule and answered by no earlier step (`property.clause-occurrence`, `property.clause-early-response`, `property.clauses.absent`).
3. The Program assembles (`realization.action-unplaced`, `realization.action-placed-twice`, `realization.action-unbound`).

- Red: `r2-red-03-lowering.log`, `TestAnErrorBehindTheProducersLimitIsStillAnError`. The retry path with a worker stop appended came back `unsupported` with no error.
- Green: that test now gets `*cp.Error{workerStop, evidence.action-unmapped}` and no mention of the limit. `TestThePathsLastStepsAreConfirmedAsTheProducerConfirmsThem` (9 cases) pins the rule. I wrote that table after extracting `unconfirmed`, so it was not run red. Five mutants of the rule prove it fails when the rule changes.

### Finding 3 (P2): result types of guard, key and poll condition

- `goir.Validate` (`load.go:1740`, `shaped`) types what needs no descriptor: an order of what is no number, a negation or conjunction of what is no condition, a comparison of two shapes, a guard or a poll condition that is no condition. A path is of any shape there.
- Lowering (`realization.go:186` `payloadOf`, `:256` `computes`) types paths against `InstructionOutcome` and the polled element before any gap is reported. A guard must be a condition. A key must be the run's id or one text or integer. Two enum values must be of one enum, and a written name must be one the enum has.
- Red: `r2-red-02-admission.log` (9 table cases and `TestARunEventSourceIsAdmitted`) and `r2-red-03-lowering.log` (8 guard and key cases, 2 poll cases).
- Green: 10 admission cases in `TestARealizationIsAdmittedBeforeItIsLowered`, 13 guard and key type cases in `TestTheFieldsAndTheSingleReadOfEvidenceAreCheckedAgainstTheirDescriptors`, 3 poll cases in `TestAnActivityScriptAnswersItsAttempts`. Five of the 13 were added after a mutant survived (two enums, a written name on the left, two written names, a conjunction over a text, a fanned key). They passed on first run and are proven by their mutants only.

### Finding 4 (P2): the canceled answer

- The owner is fn-107.20 (`lower.go:274`, `ownerSources`). The handover tables above are corrected.
- A command gap is path-specific. `gaps` (`lower.go:224`) asks `takes` (`:308`) whether a Case of this Query's path would carry the command: one every Case carries, one carried for a class the path takes, or the performance of a class the path takes. A gap on the path goes to `Lowering.Unsupported` and makes the Query `unsupported`. One off the path goes to the new `Lowering.OffPath` and does not change the standing. Monitors, evidence kinds and controls stay realization-wide.
- The inventory still accounts for every declaration. An off-path command has the disposition `OffPath`.
- Red: `r2-red-03-lowering.log`, `TestAnUnsupportedCommandBlocksOnlyTheQueriesWhosePathTakesIt` and 9 standings of `TestTheActivityRealizationNamesWhatKeepsItFromACase`.
- Green: `errand.withdrawn` is `unsupported` for `errand/cancel-attempt` alone. `errand.retry` lowers, lists the gap in `OffPath`, and its inventory closes. Of the nine activity Queries, `cancel` and `cancelRequest` list the canceled answer in `Unsupported`, and the other seven list it in `OffPath`. `TestAnUnsupportedCommandCarriedForAClassBlocksOnlyThePathsThatTakeIt` covers a conditional command and was added after a mutant survived.

### Additions to the lists for fn-107.20 and fn-107.21

Both are written into the sections above.

- fn-107.21: `retry` and `pauseResume` each put `statusScheduled` in two projection rules and need a second stable scheduling observation, proven with ordinary `testpilot.Prepare`.
- No owner: `startToCloseTimeout` needs an attempt that gives no answer. It is a recorded limit.
- fn-107.20: the canceled attempt answer is now in its table.

### Mutation, round 2

77 mutants of the round-2 code ran by `go test -overlay` on the final tree, and all were killed (`r2-mutants-03.log`, script `r2-mutants.py`). Nine survived the first full run (`r2-mutants-01.log`). I added tests for each, and removed one redundant branch of the evaluator. The 73 mutants of rounds 0 and 1 were not rerun: the restart wiped their script, and round 2 moved several of their patterns.

### Gates (final tree, after the last Go change)

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (23 packages ok) | 0 |
| `... go test -tags test_dep -count=1 -race -short -v ./model/scalav2/goir/ ./model/scalav2/goir/conformance/ ./model/scalav2/goir/testpilot/` (1064 passing, 5 Lean-dump skips, no race) | 0 |
| `GOFLAGS=-tags=test_dep make umpire-gen-scala` | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` | 0 |
| `make lint-scala` (0 `[error]` lines) | 0 |
| `go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false` (0 issues) | 0 |
| `GOFLAGS=-tags=test_dep make protoc` (changed nothing; only `api/modelir/v1` and `ir.proto` differ under `api/` and `proto/`) | 0 |

The first scoped lint run of this round failed with rc 2 and 10 findings: four complexity findings and six `use-errors-new`. I split `evaluate`, `shaped`, `gaps` and `operand`, and every gate above ran after that. The lint and `protoc` ran without the Lean stub on `PATH`, after the coordinator retired it. The generation and Scala check gates before that ran with it.

Not run: any `tools/umpire` or `tools/canary` test, any target that reaches `lake`, any live-cluster suite, `flowctl gate` receipts, and the full `-race` run without `-short`. No lowered Case has run against a server.

### Files this round

Logs have the prefix `r2-` under `.flow/tmp/fn-107/task19-logs/`, and the round-1 state of each file is under `.flow/tmp/fn-107/task19-before/r2/`.

New: `goir/conformance/guard.go`, `goir/conformance/guard_test.go`.

Changed: `ir.proto`, `api/modelir/v1/ir.pb.go`, `ir.go-helpers.pb.go`, `Realize.scala`, `standaloneactivity/Realization.scala`, `Realizations.scala.fixture`, `expected/realizations.json`, `run.sh`, `ir/activity.json`, `goir/load.go`, `goir/realization_test.go`, `goir/bound_test.go`, `goir/testpilot/{lower.go,realization.go,activity_test.go}`, `SEMANTICS.md`, `README.md`, `specimens/README.md`.

### Review round 3

Both findings are fixed, and the three readers of a guard now share one typed walk, so they agree by construction. Every gate passes on the final tree. The task stays `in_progress` with no commit.

### The shared walk

`goir.TypeOf` (`goir/load.go:1760`) is the one typing of an operand. `goir.GuardProblem` (`:1945`) is the one check of a Run Event guard: what the guard is made of, then `TypeOf` with the guard's path reading (`PayloadPath`, `:1919`), then that it is a condition. Each reader passes what it knows.

| Reader | Call | Descriptor |
|---|---|---|
| Admission | `GuardProblem(guard, nil)` in `runEvent` | None. Paths are checked for how they are written |
| Lowering | `goir.GuardProblem(guard, payload)` (`goir/testpilot/realization.go:193`) | `InstructionOutcome` |
| `Admits` | `goir.GuardProblem(guard, payload.Descriptor())` (`goir/conformance/guard.go:59`), then evaluation | That of the outcome it evaluates |

The lowering's own typer (`computes`, `comparable`, `written`, its value kinds) and admission's (`shaped`) are deleted. The evaluator lost every type check and is 176 lines. It can still fail for one reason the declaration cannot tell: the event lacks a value the guard compares, negates or joins, or holds an enum number its enum does not name. Poll conditions and the Run Event key use the same `TypeOf` with Testpilot's path reading (`lifted`, `realization.go:221`).

### Finding 1 (P2): nested paths

`TypeOf` types the inner path first and reads the outer one against the message it reaches, to any depth, for guards and for poll conditions.

- Red: `r3-red-02-lowering.log`, 4 subtests. A misspelled inner or outer segment of a guard came back as a gap with no error, and a poll condition over a path of a path was not checked.
- Green: both guard cases are located errors before gap classification (`TestTheFieldsAndTheSingleReadOfEvidenceAreCheckedAgainstTheirDescriptors`), and so are two poll cases (`TestAnActivityScriptAnswersItsAttempts`).

### Finding 2 (P2): `Named` in a guard

A guard may not write out a name a Case binds. `Admits` evaluates a guard on a recorded Run, which holds no such binding, so admission refuses it at the declaration: `its guard writes out a name a Case binds; a Run Event's guard reads the event's payload alone`. A written message, map, role or bytes value is refused the same way. `Named` stays legal in commands.

### One table through all three readers

`TestAGuardIsWellFormedForEveryReaderOrForNone` (`goir/conformance/readers_test.go`, 81 cases) feeds one declaration to `goir.Validate`, the lowering and `Admits`. It covers every operand kind, every written value kind, and paths one, two and three deep. A well-formed guard passes admission, lowers to the Run Event gap with no error, and evaluates to the expected value. An ill-formed guard gets a located error from the first reader that can know, and the same words from `Admits` on two events, one of which lacks the values.

- Red: `r3-red-01-readers.log`, 36 of the first 78 cases failed against the round-2 code.
- Three well-formed cases were added after two mutants survived (a flag, a number and a text compared with a different value). They were not run red.

The table found five more disagreements beyond the two reported, all now refused by every reader:

1. A path written with `[*]` in a guard passed admission and lowering, and `Admits` refused it. Admission now refuses it.
2. A guard path that ends in a repeated field, bytes or a floating-point number lowered, and `Admits` refused it. Lowering now refuses it.
3. A comparison of two messages lowered, and `Admits` refused it. `TypeOf` now refuses it, for poll conditions too.
4. Two enum values both written out passed admission and were refused at lowering. Admission now refuses them.
5. A path of a written value or of a condition passed admission and lowering, and `Admits` refused it. Admission now refuses it.

### Wording changes in existing tests

Five expectations in `TestAGuardThatCannotBeEvaluatedIsAnError` changed to the shared wording, with the same refusals. `orders a text and a number, and only numbers are ordered` became `orders a text, and only numbers are ordered`. A guard that reads the run and one of no kind now say what admission says. One round-2 lowering case (two written enum values) moved to the admission table, where it is now refused.

### Mutation, round 3

74 mutants of the shared walk, the lowering's use of it and the evaluator ran by `go test -overlay`, each against all three readers' tests where it mutates shared code. All were killed (`r3-mutants-01.log`, `r3-mutants-02.log`, script `r3-mutants.py`). Five survived the first run and got tests: the three comparisons above, a comparison with a value of no comparable kind, and `TestSeveralValuesAreNoOperand`. The round-2 mutants of `lower.go` were not rerun, since that file did not change. Earlier mutants of the code this round deleted no longer apply.

### Gates (final tree)

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (23 packages ok) | 0 |
| `... go test -tags test_dep -count=1 -race -short -v ./model/scalav2/goir/ ./model/scalav2/goir/conformance/ ./model/scalav2/goir/testpilot/` (1158 passing, 5 Lean-dump skips, no race) | 0 |
| `make lint-scala` (0 `[error]` lines) | 0 |
| `go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

`make umpire-gen-scala` and `make umpire-check-scala` were not rerun. No Scala file, fixture, IR file or proto changed this round, and their round-2 results stand.

Not run: any `tools/umpire` or `tools/canary` test, any target that reaches `lake`, any live-cluster suite, `flowctl gate` receipts, and the full `-race` run without `-short`.

### Files this round

Logs have the prefix `r3-` under `.flow/tmp/fn-107/task19-logs/`. The round-2 state of each changed file is under `.flow/tmp/fn-107/task19-before/r3/`; the test files there already hold this round's red cases.

New: `goir/conformance/readers_test.go`.

Changed: `goir/load.go`, `goir/realization_test.go`, `goir/testpilot/realization.go`, `goir/testpilot/activity_test.go`, `goir/conformance/guard.go`, `goir/conformance/guard_test.go`, `SEMANTICS.md`.

Review round 4 (conductor): SHIP with one P3, fixed by the conductor: `valueAt` in `goir/conformance/guard.go` tells the last path segment by its position, so a recursive message that names one field twice is read at the end of the path (`TestAPathThroughARecursiveMessageReadsItsEnd`).

The conductor narrowed this task's acceptance after the first review: lowering the real activity completion and retry Cases and `syncCompletion` concluding moved to fn-107.21; the Testpilot primitives the realization needs (single-message read, Run Event source under a guard, kinds sharing a source, a canceled answer) are fn-107.20. Six Nexus caller Properties are recorded limits. `startToCloseTimeout` needs an attempt that gives no answer, which no task owns.

The work is uncommitted; the owner makes the commits (HEAD is the owner's `1c4d304805 wip`). Review outputs and diffs are under `.flow/tmp/fn-107/task19/`. A machine restart during round 2 wiped the first two review output files; rounds 3 and 4 are kept.

stage: implement - ran (worker subagent, session model claude-opus-5-5; three fix rounds; one turn lost to a machine restart and resumed)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f86d-96e2-7f51-bb0d-ecbc000a4a91 over the uncommitted task diff; interim round NEEDS_WORK with one P1 and a rescoping; round 2 NEEDS_WORK with one P1 and three P2; round 3 NEEDS_WORK with two P2; round 4 SHIP with one P3, fixed)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... rc 0, pre-edit), round 0, final tree of that round: the same Go suite rc 0 (23 packages); -race -v on goir, goir/testpilot, goir/conformance rc 0 (925 pass, 5 Lean-dump skips); make umpire-gen-scala rc 0; make umpire-check-scala rc 0; make lint-scala rc 0; go vet rc 0; scoped make lint-code rc 0; make protoc rc 0, review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc 0, 23 packages ok), review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -short -v ./model/scalav2/goir/conformance/ ./model/scalav2/goir/testpilot/ (rc 0, 405 pass, no skip), review round 1: GOFLAGS=-tags=test_dep make umpire-gen-scala (rc 0), review round 1: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), review round 1: make lint-scala (rc 0, 0 [error] lines), review round 1: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc 0), review round 1: PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false (rc 0, 0 issues), review round 1: PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep mise exec -- make protoc (rc 0, only api/modelir/v1 changed), review round 1 red: r1-red-01-transient.log (TestTheActivityRealizationPollsNoTransientState: expected "status", actual "run_state"); r1-red-02-admission.log (12 failing), review round 1 mutation by go test -overlay: 73 mutants on the final code, all killed, not run: tools/umpire and tools/canary tests, lake targets, live-cluster suites, flowctl gate receipts; commit skipped per worker-overrides item 1, review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc 0, 23 packages ok; r2-gate-gotest.log), review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -short -v ./model/scalav2/goir/ ./model/scalav2/goir/conformance/ ./model/scalav2/goir/testpilot/ (rc 0, 1064 pass, 5 Lean-dump skips; r2-gate-race.log), review round 2: GOFLAGS=-tags=test_dep make umpire-gen-scala (rc 0; r2-gate-gen.log), review round 2: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0; r2-gate-check.log), review round 2: make lint-scala (rc 0, 0 [error] lines; r2-gate-lint-scala.log), review round 2: go vet -tags test_dep ./model/scalav2/... (rc 0), review round 2: GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false (rc 0, 0 issues; first run of the round rc 2 with 10 revive findings, fixed), review round 2: GOFLAGS=-tags=test_dep make protoc (rc 0, changed nothing; only api/modelir/v1 and ir.proto differ), review round 2 red: r2-red-01-guard.log (3 subtests of TestOnlyADeliveredAttemptIsEvidenceOfAnAttemptStart: expected false, actual true); r2-red-02-admission.log (9 table cases and TestARunEventSourceIsAdmitted); r2-red-03-lowering.log (19 subtests in 6 tests: guard and key types, poll types, path-specific command gaps, the error behind the producer's limit); r2-red-04-guard-absence.log (7 subtests), review round 2 mutation by go test -overlay: 77 mutants of the round-2 code on the final tree, all killed (r2-mutants-03.log); 9 survived the first run and got tests; the 73 mutants of rounds 0 and 1 were not rerun (script lost in the restart), review round 2 not run: tools/umpire and tools/canary tests, lake targets, live-cluster suites, flowctl gate receipts, full -race without -short; commit skipped per worker-overrides item 1, review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc 0, 23 packages ok; r3-gate-gotest.log), review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -short -v ./model/scalav2/goir/ ./model/scalav2/goir/conformance/ ./model/scalav2/goir/testpilot/ (rc 0, 1158 pass, 5 Lean-dump skips; r3-gate-race.log), review round 3: make lint-scala (rc 0, 0 [error] lines; r3-gate-lint-scala.log), review round 3: go vet -tags test_dep ./model/scalav2/... (rc 0), review round 3: GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false (rc 0, 0 issues; r3-gate-lint-code.log), review round 3 red: r3-red-01-readers.log (36 of 78 cases of TestAGuardIsWellFormedForEveryReaderOrForNone failing on the round-2 code); r3-red-02-lowering.log (4 subtests: nested guard paths and nested poll paths), review round 3 mutation by go test -overlay: 74 mutants of the shared walk and its three readers, all killed (r3-mutants-01.log, r3-mutants-02.log); 5 survived the first run and got tests, review round 3 not run: make umpire-gen-scala and make umpire-check-scala (no Scala, fixture, IR or proto file changed this round); tools/umpire and tools/canary tests, lake targets, live-cluster suites, flowctl gate receipts, full -race without -short; commit skipped per worker-overrides item 1, conductor final: go test -tags test_dep -count=1 -short ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc 0, before the P3 fix), conductor final: go test -short ./model/scalav2/goir/conformance/ (rc 0, after the P3 fix), conductor final: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0); make lint-scala (rc 0, 0 [error] lines, round 2 tree; no Scala changed since), conductor final: make lint-code LINT_CODE_TARGETS=./model/scalav2/goir/conformance GOLANGCI_LINT_FIX=false, codex impl-review: .flow/tmp/fn-107/task19/t19-r3.md (NEEDS_WORK), t19-r4.md (SHIP)
- PRs: