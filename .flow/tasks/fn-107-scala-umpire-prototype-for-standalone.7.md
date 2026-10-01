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
A Testpilot Run can now be assessed against an admitted IR Model beside its Contract. `conformance.Prepare(model, queryKey, case, limits)` returns a `testpilot.AssessmentFactory`; bound with `PreparedCase.WithAssessment`, it gives a conformance conclusion and one conclusion per claim (the Query's Property and each monitor of its machine), live and on replay, as one value. Nothing is committed, the task stays `in_progress`, and no file outside `model/scalav2/goir/conformance/` changed.

**Read this first: on the witness Run of every one of the seven lowered Nexus caller Cases the Contract's Verdict is satisfied and the model assessment leaves the Query's Property inconclusive (conformance is conformant for all seven).** This follows from the Model and the evidence each Case carries, not from a defect in either:
- `syncCompletion`, `handlerError`, `scheduleToStartTimeout`, `startToCloseTimeout`: another modeled execution records the same evidence without taking the step the Property is about (a completed event is also what a completion records; a timed-out event is recorded by all three deadlines).
- `asyncCompletion`, `asyncFailure`: a completion that arrives after the operation is over is not found and records nothing, so no evidence excludes one, and `completionSucceeds` / `completionFails` fail on that step as written.
- `retry`: the pending-attempts evidence carries no count, so attempt 2 explains it as well as attempt 1.

The Contract reads the evidence as the witness, step by step; the assessment keeps every execution that explains it. Whether these Properties should be reachable as satisfied (by carrying the started event in every Case, a typed attempt count, or narrower claims) is the owner's call.

### Files

Workspace: `/private/tmp/claude-501/-Users-stephan-Workspace-temporal-umpire/3649feab-fa2b-4588-b529-974aeb4c1742/scratchpad/w7`. `git status` there differs from the pre-edit snapshot by one untracked directory. No existing file changed, so there are no before-copies.

Added, all under `model/scalav2/goir/conformance/` (1,372 non-test lines, 1,574 test lines):
- `conformance.go`: package doc, `Prepare`, `Limits`, `Factory`, `LimitError`, `EvidenceError`, the Model and Query identities.
- `plan.go`: the immutable snapshot: the machine's steps and hole rows by state, and what each claim reads on each step.
- `evidence.go`: the one place a Run's evidence is read.
- `candidates.go`: the order between observations and the bounded exploration.
- `conclude.go`: the one place a conclusion is decided.
- `assessor.go`: the `testpilot.Assessor`.
- `assessment_test.go`, `nexus_test.go`, `closing_test.go`, `conformance_test.go`, `fixtures_test.go`.

### What it decides

- Evidence is the `CorrelatedEvidence` values of the Case's evidence observation. One observation is one fact of one step. The kinds a Case carries are the ones its correlated Contract gives a projection rule; each is resolved through the Case's local names to the realization's declaration.
- One machine instance per Run scope and operation key. An instance is violated or nonconformant when any is; satisfied or conformant only when all are.
- Order is the evidence's causal parents and its ordinal within its source, and nothing else: not Run Event order and not a clock. Unordered observations are tried both ways.
- A step whose fact no observation reports may still have happened. The exception is a durable kind the Case carries, once the Run has closed complete with that kind's source ordinals unbroken and every named parent recorded: then a step that records such a fact has its observation.
- `Observe` uses only the first reading and reports a nonconformance or violation at the event that proves it. `Close` adds the durable reading and the end-of-path monitors.
- Conformance: nonconformant when no execution is left and no hole row was in reach of one tried; a hole in reach makes it inconclusive.
- A claim: violated when every execution left violates it; satisfied when every one reads it and none violates it; inconclusive otherwise, including when a hole row was in reach of an execution that had not violated it. Nothing is conformant or satisfied unless the Run closed `COMPLETED` with no execution-incomplete event and no Contract evaluation failure.
- Events from the first `execution_incomplete` one on are not read, as for the Contract.
- Ceilings: `MaxCandidates` per reading, `MaxWork` over the assessment, 64 observations per operation. One that is reached is a `*LimitError` from `Observe` or `Close`, which Testpilot reports as the Assessment's failure with the ceiling named; what was established stands.

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | A trace conforms while a history-sensitive property stays inconclusive | pass | `TestAWitnessRunConformsWhileItsPropertyStaysOpen/retry` (and the six other lowered Cases), on the real lowered Case and its real Contract |
| 2 | Missing or crossed evidence and reachable holes keep the conclusion limits | pass | `TestEvidenceIsAssessedLiveAndReplayedAlike`: rows `missing admission commit`, `crossed operation`, `crossed run`, `A10 commit evidence removed`, `a hole in reach of an explained trace`, `a hole that may account for a mismatch`, with the two hole-free contrasts; `TestABrokenRecordSaysNothingOfWhatIsAbsent` |
| 3 | Live and offline agree, including violation before failure, candidate limits, timeout after commit and lost acknowledgment; pinned with and without commit evidence | pass | `TestEvidenceIsAssessedLiveAndReplayedAlike` (20 rows, each: live equals the worked-out value, replay equals live, whole value, Verdict equals the unassessed Case's); `TestAViolationStandsWhenACeilingIsReachedAfterIt`; `TestACeilingThatIsReachedIsReported` |
| 4 | Timestamp skew with unchanged causal facts keeps the verdicts | pass | `TestClockSkewChangesNoAssessment` (the same 20 Runs with every elapsed coordinate moved) |
| 5 | The public factory gives fresh bound state without internal imports or a Testpilot-to-Scala dependency; existing callers keep the Contract; mismatched replay identities reject | pass | `TestConcurrentRunsAndReplaysShareNoState` (8 goroutines, race detector), `TestThePackageImportsNoInternalPackage`, `TestAnAssessmentIsReplayedOnlyUnderItsOwnBinding` (another Model, other ceilings, another Query's Case, and a moved declaration that is the same Model), the Verdict comparison in every row, and the unchanged `./common/testing/testpilot/...` suite |

Also: `TestAMismatchTheContractFailsOnIsANonconformance` (a second completed event: the Contract's evaluation fails there and its Verdict is inconclusive; the assessment reads that event and reports the nonconformance), `TestEvidenceThatCannotBeReadIsALocatedError` (13 malformed inputs, each an `*EvidenceError` at its event, no panic), `TestPrepareRefusesWhatItCannotAssess` (15 cases), and the precedence tables `TestAClaimIsConcludedOnlyFromAgreement` (64 combinations against the rule written out), `TestConformanceIsConcludedFromWhatExplainsTheEvidence`, `TestInstancesAreConcludedTogether`, `TestOnlyARunThatClosedCompleteIsConcludedPositively`.

### The Run fixtures

- **Recorded from a fake** (the 20 rows, the ceiling test, the concurrency test): a carrier Case written in `fixtures_test.go` reads one piece of evidence per instruction from a fake Driver written against the public facade, so Testpilot's own executor and recorder make the Run. Its correlated Contract gives every kind the meaning `IRRELEVANT`, so the Contract checks the evidence's form and concludes nothing.
- **Constructed** (`nexus_test.go`): Runs of the seven lowered Nexus Cases written out event by event and replayed only. The mismatch Run is marked the way the recorder marks a Contract evaluation failure, after the test first shows the unmarked Run fails at that event.
- The activity rows run on the lifter's `admission` fixture (both designs, with its monitors). It declares no realization, so the tests declare one per design from the specimen's evidence matrix: the three statuses reported, `dispatchEnqueued`, `attemptAdmitted` and `admissionRejected` durable. The hole rows do the same for the `declarations` fixture's `disk` and `store`. These are test fixtures in Go; a realization for these machines belongs in Scala (task 9 or 10).

Expectations were written from `specimens/activity.md` (A1, A1', A2, A4, A7, A8, A10), `Claims.scala` and the Nexus kernel before each first run, with the reasoning in a comment beside each row. Of the 28 whole-Assessment expectations (16 activity rows, 4 hole rows, 7 lowered Cases, 1 mismatch), 27 matched on their first run. The one that did not was mine: `crossed run` expected a completed Run, and the Contract refuses evidence whose scope changes (`verification/correlated.go:224`), so the Run is incomplete; I corrected the row from that rule.

### Decisions that differ from the task text

1. **Completeness comes from `commitment`.** A durable kind is complete on a closed, unbroken Run; a reported kind never is. I took this from the specimen (A7 differs from A8 "by the absence of the commit observation") because the vocabulary has no way to declare a source complete. Without it nothing separates "with commit evidence" from "without".
2. **The Contract's evaluation-failure event is read.** Only `execution_incomplete` freezes the assessment. The recorder flags events after the failing one, not the failing one, and that event is where a visible mismatch is. A Run with a recorded evaluation failure still concludes nothing positive.
3. **A ceiling is reported as the Assessment's failure** (`observe_failed` or `close_failed`, detail `run event N: conformance <resource> ceiling of K reached`), since Testpilot's `limit_exceeded` code is for its own ceilings.
4. **The Scenario gives only the start state.** Its pinned schedule does not restrict the executions kept: the system may take any modeled step.
5. **The claims are the Query's Property and every monitor of its machine**, as a monitor watches every Query of its machine.
6. **A satisfied claim must be read on every execution left.** One that never reaches the claim's evaluation point gives no verdict (SEMANTICS, Monitors), which I apply to Properties too.
7. **Not assessed, refused at `Prepare`:** a Query of a composition, a Query read through a refinement, a transition Property with a `when`, a machine with two realizations or none.
8. **One observation explains one fact.** A fact reported twice needs two steps, as for the Contract.
9. **The ceilings are part of the Query identity**, so a Run assessed under other ceilings is refused on replay.
10. **Model identity** is a hash of the Model with every source position removed.

### Where typed identities replace text

All in `evidence.go`: `observation.instance` (today the scope text and operation key), `observation.after` and `identityOf` (parents and source ordinals as text), and `reader.observed`, which builds them. Typed operation, attempt and delivery fields from task 13 are read there; `candidates.go`, `conclude.go` and `assessor.go` see only `observation`. An attempt identity would also let `retry` and A10's "the poll returned attempt 1" be settled, which they cannot be today.

### Gaps outside the Touches

1. **goir exports no claim bound to a step.** `Realizer.Find` returns a `umpire.Query` whose Property predicate and watching monitors are unexported. `plan.go` therefore reads `holds`, `next`, `violated` and `after` with `Interpreter.Call` under the rules of SEMANTICS (a non-Boolean or an out-of-domain monitor state is a located error, a hole is unknown). About 120 lines would go if `Realizer` exposed the bound Property and monitors per step. Nothing was copied from goir.
2. **No way to declare an evidence source complete or sampled** in `umpire.realize` or the IR (decision 1).
3. **A lowered Case carries only the kinds on its path.** The off-path kinds the realization declares, such as the started event in `syncCompletion`, never reach the Run as correlated evidence, although the `history-event` observation carries the raw events.
4. **No realization for the activity and close/reset Models** (task 9), and no durable-commit kind can be lowered (task 10), so R8's commit evidence exists only in the carrier fixtures.
5. `AssessedCase.Evaluate` returns no Assessment when the Contract's replay errs. A constructed Run must carry the evaluation-failure coordinate, as a recorded one does.

### Test-first record

I did not work test-first for the first shape. The exploration, conclusions and Assessor were drafted before any test, because the semantics had to be settled against the seam first.
- Red against a stubbed exploration: 15 of the then 16 rows failed, for example `A1 stale design admits after the pause` at `assessment_test.go:315` (expected `violated`, got `inconclusive`). Log `task7-logs/red-01-admission.log`.
- One real defect was found red: an empty causal parent was accepted, since a nil element comes back from the wire as an empty message. `TestEvidenceThatCannotBeReadIsALocatedError/an_empty_parent` at `conformance_test.go:242`, "An error is expected but got nil" (`red-02-empty-parent.log`), then fixed in `evidence.go`.
- Mutation, foreground with restore: 17 mutants. 12 died on the first pass; the 5 survivors each got a test in `closing_test.go`, and all 17 now die (`mutation.log`, `mutation-2.log`).

### Gates (final tree)

| Command | rc |
|---|---|
| Baseline before editing: `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` | 0 |
| Baseline before editing: `... go test -tags test_dep -count=1 ./common/testing/testpilot/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/...` (15 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -v ./model/scalav2/goir/conformance/...` (115 passing tests and subtests, no skip, no race) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/conformance/...' GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

Not run: `make umpire-check-scala`, `make lint-scala` (no Scala file changed), any `tools/umpire` or `tools/canary` test, any live-cluster suite, and every target that could reach `lake`. `~/.elan/toolchains` is empty and the workspace has no `model/lean/.lake`. No `time.Sleep` in the package. No `flowctl gate` receipt was attempted. Logs are under `<workspace>/.flow/tmp/fn-107/task7-logs/`.

### Left undone

- No tenfold probe beyond the ceiling tests: `TestACeilingThatIsReachedIsReported` pins both ceilings at their exact counts and at ten times them on a three-candidate machine, and nothing larger was measured.
- A `FAULT_INJECTED` Run Event is not read as evidence; the worker stop is a step that records nothing either way.
- An Assessment's `Detail` names the machine, the instance and one reason. It carries no witness execution and no counts.
- Unordered observations cost exponentially in their number within one operation; the two ceilings bound it, and nothing reduces it.
- `model/scalav2/SEMANTICS.md` and `README.md` do not describe runtime assessment; both are outside the Touches.

### Review round 1

The Codex review returned NEEDS_WORK with three P1, one P2 and one P3. All five were valid, each failed a test before its fix, and all are fixed. Durable evidence no longer rules anything out, a ceiling is consulted before the work it bounds, a Case's scope and field policies are enforced, monitor states are 32-bit, and the claim binding now comes from goir. Nothing is committed and the task stays `in_progress`.

**The sections above describe round 0. Where they speak of durable completeness, the "durable reading" at `Close`, `unbroken`, nonconformance for a missing commit, or `plan.go` reading claim functions itself, this section replaces them.**

### Files

Changed, outside the original Touches by the widening for finding 5:
- `model/scalav2/goir/claims.go` (before-copy at `<workspace>/.flow/tmp/fn-107/task7-before/model/scalav2/goir/claims.go`; 198 changed lines).

Added:
- `model/scalav2/goir/bound_test.go`
- `model/scalav2/goir/conformance/bounds_test.go`, `monitor_test.go`

Changed since round 0, all in `model/scalav2/goir/conformance/`: `conformance.go`, `plan.go`, `evidence.go`, `candidates.go`, `assessor.go`, `assessment_test.go`, `closing_test.go`, `conformance_test.go`, `fixtures_test.go`. `conclude.go` and `nexus_test.go` are unchanged.

The full list to merge: `model/scalav2/goir/claims.go` (changed), `model/scalav2/goir/bound_test.go` (added), and the 13 files of `model/scalav2/goir/conformance/` (added): `conformance.go`, `plan.go`, `evidence.go`, `candidates.go`, `conclude.go`, `assessor.go`, `assessment_test.go`, `nexus_test.go`, `closing_test.go`, `conformance_test.go`, `fixtures_test.go`, `bounds_test.go`, `monitor_test.go`. The package is now 1,438 non-test and 1,893 test lines.

### 1. P1, matchings were listed before a ceiling was checked (valid)

`expand` no longer lists anything. `explaining` builds a set one observation at a time and charges one unit of work before each observation is added and before each step is taken. A set reached two ways (two facts of one name) is taken once; the record of sets taken grows only after a charge.

- Red first (`task7-logs/r1-red-01-lazy.log`): `TestACeilingIsReachedBeforeTheWorkItBoundsIsDone/22_facts` allocated 386,886,392 bytes under `MaxWork=1`, `MaxCandidates=1` ("is not less than 65536"). I ran only the 22-fact case red; the 64-fact case (2^64 sets) could not have returned.
- Green: both cases return `LimitError{work, 1}` with `spent == 1` and under 64 KB allocated, measured by `runtime.MemStats.TotalAlloc`, not by time. `TestEverySetOfObservationsAStepCanExplainIsTried` pins the whole count for three facts (27 steps and 19 tries, worked out in the comment), and `TestASetReachedTwoWaysIsTakenOnce` the deduplication.
- **Work is counted differently now:** one unit per observation tried and one per step taken, where it was one per step taken. `TestACeilingThatIsReachedIsReported` moved from 2 to 3 units for the store, with the count written out.

What I checked for a collection sized by evidence or Model data before a ceiling:

| Place | Size depends on | Result |
|---|---|---|
| `expand` / `explaining` | sets of observations per step | **Fixed** (above). |
| `tried` in `expand` | sets taken per step | Grows only after a charge. |
| `seen`, the queue, `reached` | candidates | Checked against `MaxCandidates` before each insert. One candidate is built before its key can be compared; that is the only allocation ahead of the check. |
| `order` | observations of one operation, squared | At most 64, refused before the 65th is appended. |
| `assessor.observe`, re-ordering every other operation on each event | operations times events | **Fixed.** It was uncharged and quadratic in the Run. A parent of another operation is now found through an index by identity, in constant time, in whichever order the two arrive. Red-free refactor; `a parent of another operation, recorded after its child` and `two operations naming one parent` pin both directions. |
| `named`, `awaited`, `instances` | Run Events | One entry per event, bounded by Testpilot's `MaxEvents`. |
| `Close` | operations times claims | One exploration per operation, charged; the status slices are operations times claims. |
| `plan.index` at `Prepare` | the machine's rows | The table goir built within its own ceilings. |
| `compileProperty`, `compileMonitor` at `Prepare` | steps, and monitor states times steps | **Fixed.** Each reading is charged against the new `Limits.MaxReadings` before it is made, and a monitor is read only from the states a step can take it to. One row of `next` (one entry per step) is allocated ahead of its charges. |
| `newReader` | the Case's projection rules and field policies | The Case as given. |
| `explaining` recursion depth | facts of one step | The table's. |

### 2. P1, absence was inferred from durable evidence (valid)

The inference is removed. `regime.complete`, `assessor.unbroken`, `plan.durable` and `kind.durable` are gone; every step may go unobserved on every Run; `Close` re-reads a completed Run only so that end-of-path monitors are read. `commitment` is no longer read by the adapter at all.

- Red first (`r1-red-02-absence.log`): I re-derived the expectations and they failed against the old code in 7 subtests, for example `missing admission commit is no mismatch` (expected conformant, got nonconformant).
- Rows that changed, each re-derived from the specimen: `A1 stale` (was violated/satisfied/satisfied, now all three inconclusive: an unobserved earlier admission explains it), `A2 stale` (both monitors now inconclusive), `timeout after commit` on the stale design (both monitors now inconclusive), `missing admission commit` (was nonconformant, now conformant with all claims inconclusive), `crossed operation` (moved to the corrected design: conformant, the admission monitor inconclusive where the same commit on the same activity satisfies it). New row: `timeout after commit, corrected design` (all satisfied, because that design admits at most once on every execution, observed or not).
- The pin: `TestNoAbsenceIsInferredWithoutAnExhaustiveDeclaration` (a reported start with the commit source unbroken and silent is conformant; one observed commit leaves the stale design's admission monitor inconclusive and satisfies the corrected design's).
- Consequence for R8: with no exhaustive declaration, commit evidence proves a commit and can establish a mismatch (`A1'`, `A8 stale`), and "with versus without commit evidence" shows only where the Model itself guarantees the claim. No fixture can now show a stale-design violation from commit evidence.

**The declaration a later task needs** (outside my Touches):
- `model/scalav2/scala/umpire/realize/Realize.scala`: a field on `Evidence`, for example `exhaustive: Boolean = false` ("the source reports every occurrence of the facts this kind records, for the operations a read of it covers"), and on a read command that lifts it a marker that the read is final, for example `ResponseRead(..., closes = Vector(evidenceId))`, placed after the system is quiescent.
- `proto/internal/temporal/server/api/modelir/v1/ir.proto`: `bool exhaustive = 9` on `message Evidence`, and the kinds a `ResponseRead` or `Poll` closes; `goir/load.go` admission must refuse `exhaustive` on a kind no command closes.
- The lowered Case must carry both, where the assessment can see them: which kinds are exhaustive (on `EvidenceDeclaration`, or in provenance) and which instruction is the closing read of each. The Run already records whether that instruction completed. The adapter would then allow absence only for an exhaustive kind whose closing read succeeded, with its source's ordinals dense, on a Run that closed complete.
- The plug-in point here is `explaining` in `candidates.go`: the skip branch of a fact is the one line that would become conditional.

### 3. P1, scope, field policies and attempt correlation (valid)

- **Scope.** `Prepare` requires the Case's `scope_fields` to be exactly the realization's Run field and its `operation_field` the realization's operation field, through the Case's local names. Each piece of evidence, and each of its parents, must carry exactly those scope fields in order with values, from a source the Case names; and all evidence of a Run is under one scope. Anything else is an `*EvidenceError` at its event, so nothing is established from it. The `crossed run` row is now an assessment failure at that event, where it was read as a second operation.
- **Field policies.** Evidence must carry its kind's fields as the Case declares them: no undeclared field, none twice, a redacted field present without a value, a rejected field absent.
- **Roles.** `observation` has typed `attempt` and `delivery` roles, and observations that name different attempts or deliveries are never facts of one step (`sameStep`, enforced while a set is built). Nothing can fill them today: the IR declares a Run field and an operation field and no other role. So a Case whose evidence **retains** any field is refused at `Prepare` as unsupported, located at the kind, because the field could be an attempt or delivery key and can be neither matched nor safely ignored. The seven lowered Nexus Cases declare no fields and are unaffected.
- Red first (`r1-red-03-correlation.log`, against type-only stubs): 20 subtests, for example `TestOneStepIsNotExplainedAcrossAttemptsOrDeliveries/two_attempts`, `TestEvidenceThatCannotBeReadIsALocatedError/a_scope_that_changes_within_the_Run`, `TestPrepareRefusesWhatItCannotAssess/a_kind_that_retains_a_field_of_no_declared_role`, `TestEvidenceCarriesItsKindsFieldsAsDeclared/the_rejected_field`.
- Limit: role agreement is checked within one step only. The IR gives a step no attempt, so "this fact belongs to attempt 2 of the machine" cannot be said or checked across steps.

**Where task 13's typed fields plug in after the merge**, all in `evidence.go`:
- `reader.observed` fills `observation.attempt` from `InstructionOutcome.activity_attempt.sdk_attempt` (with `activity_run_id`) and `observation.delivery` from `activity_attempt.delivery_id`, read from the Run Event that carried the evidence; `reader.read` passes the event's payload to it.
- The `RETAIN` refusal in `newReader` is where a declared role lifts the refusal, once a realization can name a field's role.
- `role.agrees` and `observation.sameStep` are already what `candidates.go` calls; nothing there changes.
- The `activity_attempt_failure` instruction arm is an outcome, not correlated evidence; it becomes evidence only if a realization declares a kind for it.

### 4. P2, monitor states in 16 bits (valid)

Monitor states are `int32`, numbered in the order they are found from the initial state, and `lost` is `-1`, which is no position. The candidate key carries the state in 32 bits.

- Red first (`r1-red-04-monitor-index.log`): `TestAMonitorStateIsItselfAtEveryIndexOfTheLargestCatalog/32767` panicked, "index out of range [-32768]". The test puts a 65,536-state counter on the store and checks one step from 0, 32766, 32767 and 65534, each to the one state that violates it, and one step that stops short of the target.
- `TestCandidatesAreKeyedByTheWholeMonitorState` keeps `lost`, 0, 32767, 32768 and 65535 apart in the key.

### 5. P3, the claim binding is goir's (valid)

`goir/claims.go` exports `Realizer.Bound(key) (*Bound, error)`, `Bound{Table, Start, Property, Monitors}`, `BoundProperty{Name, About, Holds}`, `BoundMonitor{Name, Initial, Next, Violated, AtEnds, Read}` and `Unknown(err)`.

- The functions are the ones `Check` declares. `binding.property` and the monitor binding were split so that the closures are built once (`propertyReads`, `watch`) and handed both to `umpire.KeyProperty` / `KeyTransitionProperty` / `KeyMonitor` and to `Bound`. They read steps through the same `subject.step` and `keyedStep`, and the same `when` selector. The table is the one the Query's Scenario is declared on, with hole rows as unknown pairs.
- `Bound` refuses what `Check` does not answer on one machine's own table: a composition, a Query read through a refinement, an unsupported or malformed declaration.
- `plan.go` no longer imports the interpreter or reads any IR function. `compileProperty` and `compileMonitor` only tabulate the bound functions' answers.
- Red first (`r1-red-05-seam.log`, against a stub): `TestABoundQueryReadsStepsAsCheckDoes` panicked on the stub's nil table.
- Agreement with `Check`, step by step: for every Query receipt of the `admission`, `closereset`, `declarations` and `realizations` fixtures that carries a witness (17: five, eight, one and three), the test walks the witness through `Bound` alone and requires what the receipt says: a found trace holds on every step the Property is about and is about one; a counterexample fails on its last step and none before, or leaves the named monitor violated; and every monitor ends in the state and with the verdict the receipt reports. It also checks that hole rows are the table's unknown pairs and that an end-of-path monitor is read after no step.
- `Check`'s behaviour is unchanged: every existing goir and goir/testpilot test passes unedited.
- `Bound`'s functions share the binding's interpreter and are not safe for two goroutines at once; conformance calls them only inside `Prepare`.

### What each lowered Nexus Case would need to be concluded (for task 9)

| Case | Property | Evidence that would discriminate |
|---|---|---|
| `syncCompletion` | `syncSucceeds` | The started event carried by the Case and its history source declared exhaustive with a final read after close: no started event then means no asynchronous start, so the completed event is the synchronous reply's. |
| `handlerError` | `handlerErrorFails` | The same exhaustive history source (rules out a completion after an asynchronous start), plus evidence that tells a handler error from a failed reply: the failure's kind as a retained field with a declared role, or two kinds. |
| `asyncCompletion`, `asyncFailure` | `completionSucceeds`, `completionFails` | Completion-result evidence: a kind for the completion call's own answer (`accepted` or `notFound`), keyed to the operation and exhaustive, so that a late not-found completion is either observed or ruled out. Alternatively the claim narrowed to an accepted completion. |
| `retry` | `retrySucceeds` | A typed attempt count on the pending-attempts evidence (or on the completed event), matched against the state's `attempts`. |
| `scheduleToStartTimeout`, `startToCloseTimeout` | `scheduleToStartFires`, `startToCloseFires` | The timeout type as distinct evidence: one kind per `TimeoutType`, or the type as a retained field with a declared role, so that a timed-out event names the deadline that fired. For `scheduleToStartTimeout`, also the schedule's deadlines or an exhaustive history source. |

None of these works until a retained field can declare its role (finding 3) and a source can be declared exhaustive (finding 2).

### Mutation

Foreground with restore over `conformance/` and `goir/claims.go` (`r1-mutation.log`, `r1-mutation-2.log`): 22 mutants. 18 died on the first pass. Three of the four survivors each got a test (a set taken twice, the 16-bit key, an end-of-path monitor's `Read`) and now die. The last is equivalent: dropping the `through` test in `Bound` changes nothing, because such a Query also reads a Property of another machine, which the same condition refuses.

### Gates (final tree)

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./model/go/...` (21 packages ok) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -v ./model/scalav2/goir/conformance/...` (158 passing tests and subtests, no skip, no race) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/...` | 0 |
| `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

Not run: any `tools/umpire` or `tools/canary` test, any Scala target (no Scala file changed), anything that reaches `lake`. `~/.elan/toolchains` is empty.

### Decisions this round

- `Limits.MaxReadings` is a new required ceiling, for the claim readings `Prepare` makes. It is not part of the Query identity, since it changes whether `Prepare` succeeds and nothing an assessment concludes.
- Refusal messages for a composition and a through Query are now goir's ("is not read on the steps of one machine").
- One scope per Run, as the Contract requires, replaces "one machine instance per scope and operation".

### Still open

- No exhaustive-source declaration and no role declaration for evidence fields exist; both are specified above.
- A lowered Case still carries only the kinds on its path.
- `SEMANTICS.md` does not describe `Realizer.Bound` or runtime assessment.

Review round 2 (conductor): SHIP with one P3, fixed by the conductor after merging: `assessor.admit` finds an instance through a name index, not by scanning (`assessor.go`). The task was implemented in a directory copy; its files (`goir/conformance/**`, `goir/claims.go`, `goir/bound_test.go`) were merged into the main checkout by path, beside task 13's concurrent Testpilot changes. On the merged tree the conformance package passes under `-race`, `./model/scalav2/goir/...` passes with `-short`, and the scoped lint has 0 issues.

Carried to later work: no realization can declare a source exhaustive or give a retained evidence field a role (attempt, delivery), so absence is never inferred and a Case that retains an evidence field is refused at Prepare; the seven lowered Nexus Cases conform with their Property inconclusive until their realizations carry discriminating evidence (table above); task 13's typed `activity_attempt` identity fields plug into `evidence.go` at the places listed above.

The work is uncommitted; the owner makes the commits. The review outputs are under `.flow/tmp/fn-107/task7/`.

stage: implement - ran (worker subagent in a directory copy, session model claude-opus-5-5; one fix round)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f7c2-c84d-73f0-bd20-0dc11f41d1cd; round 1 NEEDS_WORK with three P1, one P2 and one P3, all fixed; round 2 SHIP with one P3, fixed)
stage: wave-join - ran (files copied by path into the main checkout; integrated gates rc 0)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0; ... ./common/testing/testpilot/... rc=0, both before any edit), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... (rc=0, 15 packages ok), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -v ./model/scalav2/goir/conformance/... (rc=0, 115 passing tests and subtests, no skip, no data race), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc=0), PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/conformance/...' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), mutation check, foreground with restore: 17 mutants, 17 killed after 5 survivors each got a test (task7-logs/mutation.log, mutation-2.log), not run: make umpire-check-scala, make lint-scala (no Scala file changed); any tools/umpire or tools/canary test; any target that reaches lake; any live-cluster suite, CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./model/go/... (rc=0, 21 packages ok), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -race -v ./model/scalav2/goir/conformance/... (rc=0, 158 passing tests and subtests, no skip, no data race), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/... (rc=0), PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/goir/...' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), red before each fix: task7-logs/r1-red-01-lazy.log, r1-red-02-absence.log, r1-red-03-correlation.log, r1-red-04-monitor-index.log, r1-red-05-seam.log, mutation check, foreground with restore: 22 mutants, 21 killed, 1 equivalent (task7-logs/r1-mutation.log, r1-mutation-2.log), conductor, merged tree: go test -tags test_dep -count=1 -race ./model/scalav2/goir/conformance/ (rc 0), conductor, merged tree: go test -tags test_dep -count=1 -short ./model/scalav2/goir/... (rc 0), conductor, merged tree: make lint-code over goir and goir/conformance, GOLANGCI_LINT_FIX=false (0 issues), codex impl-review rounds: .flow/tmp/fn-107/task7/t7-r1.md, t7-r2.md (final SHIP)
- PRs: