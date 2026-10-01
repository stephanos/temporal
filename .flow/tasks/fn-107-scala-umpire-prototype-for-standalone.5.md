---
satisfies: [R1, R5, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.5 Check Nexus close/reset outcome and ownership contracts

Touches: [model/scalav2/scala/temporal/nexuscaller/closepolicy/**, model/scalav2/ir/nexus-close*.json, model/scalav2/goir/nexus_close*_test.go, model/scalav2/run.sh]

## Description
Add a bounded linked-run Nexus design specimen using small close/reset/handler contracts. Keep current runtime cancellation work deferred.

**Size:** M
**Files:** proposed model/scalav2/scala/temporal/nexuscaller/closepolicy/Model.scala and Claims.scala, generated IR, design-check test/trace fixtures.

### Approach
- Reuse existing Nexus action vocabulary; keep logical operation/request IDs distinct from run ownership.
- Implement the reviewed cancel/handler/retention/ack cuts and two deliberately faulty policies as Scala declarations.
- Check safety monitors and conditional progress through the generic checker; vary terminal outcomes, rejection types, duplicates, and deadline/retention assumptions.
- Compare only behavior represented by the existing Go baseline; label new close/reset claims as authored design promises.

### Investigation targets
**Required:** model/scalav2/scala/temporal/nexuscaller/Model.scala; model/scalav2/scala/temporal/nexuscaller/Claims.scala; model/scalav2/scala/temporal/nexuscaller/kernel/Nexus.scala; model/go/nexuscaller/model.go; model/scalav2/scala/temporal/nexuscaller/Realization.scala:68.
**Optional:** .flow/specs/fn-79-deferred-nexus-operation-cancellation.md; tests/nexus_workflow_test.go:2078.

### Quick commands
`make umpire-check-scala`; `make lint-scala`; `mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/nexuscaller/...`.

## Acceptance
- [ ] Both pinned faulty designs produce replayable counterexamples and corrected ownership/retention excludes them within scope.
- [ ] Cancellation receipt/effect/knowledge and immutable history/detached work remain distinct.
- [ ] Terminal outcomes, duplicate reporting, reset cuts, and principal reconstruction satisfy the declared controls.
- [ ] Timeout resolution and finite open prefixes never become unsupported indefinite-hang claims.

- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing with that tree absent.

## Done summary
The Nexus caller close and reset designs are now a feature Model in `scala/temporal/nexuscaller/closepolicy/`, lifted to `ir/nexus-close.json` through `run.sh` and checked through `goir.Check` alone. Both pinned faulty policies give replayable counterexamples, the corrected policy excludes them, and every oracle N1 to N10 of `specimens/nexus.md` is reproduced; N11 is covered in part (decision 8). Nothing is committed and the task stays `in_progress`.

### Files

Changed (before-copy under `.flow/tmp/fn-107/task5-before/`):
- `model/scalav2/run.sh`: the nineteen Nexus close roots, their lift into `ir/nexus-close.json`, and the stale-IR comparison for it. The usage comment and the comment on where IR is checked in name the new file.

Added:
- `model/scalav2/scala/temporal/nexuscaller/closepolicy/Model.scala` (702 lines, 444 without blanks and comments): vocabulary, step functions, four monitors, six assumptions, nine machines.
- `model/scalav2/scala/temporal/nexuscaller/closepolicy/Claims.scala` (465 lines, 383 without blanks and comments): 22 Properties, three Query helpers, ten progress claims.
- `model/scalav2/ir/nexus-close.json` (974,588 bytes), written by `make umpire-gen-scala`: 9 machines, 4 monitors, 6 assumptions, 110 Properties, 99 Scenarios, 156 Queries, 10 progress claims.
- `model/scalav2/goir/nexus_close_test.go` (746 lines, 17 tests) and `model/scalav2/goir/nexus_close_baseline_test.go` (432 lines, 8 tests).

`model/scala`, `model/go`, the lifter, the Scala framework, `goir`'s non-test files, the fixture and the specimen are untouched by this task. No file under `model/scala` has an mtime after my baseline run; `git status` shows `model/scala/temporal/test/Lean.test.scala` and `specimens/nexus.md` modified, as they were before I started.

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | Both pinned faulty designs produce replayable counterexamples, and corrected ownership and retention excludes them within scope | pass | `TestNexusCloseResults`, `TestNexusCloseRejectionAfterCloseLosesTheOutcome` (N1), `TestNexusCloseAcknowledgmentByTheOriginalRun` (N2), `TestNexusCloseRetainedOutcomeIsReapplied` (N1'), `TestNexusCloseWitnessesReplay`, `TestNexusCloseFreeSearchesReachEveryState`, `TestNexusClosePinnedSearchesExploreTheSpecimensStates` |
| 2 | Cancellation receipt, effect and knowledge, and immutable history and detached work, stay distinct | pass | `TestNexusCloseCancellationAcrossReset`, `TestNexusCloseFrozenHistoryAndDetachedWork` |
| 3 | Terminal outcomes, duplicate reporting, reset cuts and principal reconstruction satisfy the declared controls | pass | `TestNexusCloseDuplicateCompletion` (N5), `TestNexusCloseResetAfterAcknowledgment` (N4 and its mutation), `TestNexusClosePrincipalLossIsItsOwnAssessment` (N7), the N6 rows of `TestNexusCloseResults`. Succeeded, failed and canceled are N1, N2 and N3 |
| 4 | Timeout resolution and finite open prefixes never become unsupported indefinite-hang claims | pass | `TestNexusCloseTimeoutResolvesTheLostOutcome` (N8), `TestNexusCloseDeadlockIsTheLostOutcome` (N10), `TestNexusCloseFairNonProgressCycle` (N9), `TestNexusCloseRetainedOutcomeNeedsRecovery` (N11, in part) |
| 5 | Legacy `model/scala` unchanged, and v2 generation, lifting, native tests and lint pass with that tree absent | pass | `final-iso.log`: a copy of the tree without `model/scala` and `model/scalav2/gen` ran `make umpire-check-scala` (rc 0) and `make lint-scala` (rc 0, no `[error]` line). `TestIRSourcePositionsResolveInsideScalav2` and `TestNoInputNamesTheLegacyTree` read the new IR |

Baseline comparison and inventory (task notes): `TestNexusCloseOpenCallerAcceptsACompletionAsTheGoModel`, `TestNexusCloseCompleteIsTheBaselinesAction`, `TestNexusCloseCompletionClaimsEqualTheGoModel`, `TestNexusCloseEvidenceNamesTheBaselinesEvents`, `TestNexusCloseLateCompletionDiffersFromTheGoModel`, `TestNexusCloseBaselineClaimsAreComparedOrExcluded`, `TestNexusCloseEveryDeclarationIsLiftedAndAsked`, `TestNexusCloseProgressClaimsAreTheSpecimens`, `TestNexusCloseReceiptsNameTheirAssumptions`.

### What the feature Model adds over the fixture

The fixture is the specimen's supported block with two monitors. The feature Model keeps its vocabulary (`Caller`, `Handler`, `Completion`, `Retained`, `Knowledge`, `Answer`, `Policy`, the step function names, `ownerKnows`, `outcomePreserved`, `ackOnlyWhenKept`, `settled`, the three design names, the seven Query names) and adds:

- **Evidence.** Twelve facts, which the fixture left to this task. An owner's history records an outcome and the deadline by the baseline's event names.
- **The cancel request's principal and its delivery.** `Intent.requested(by)` replaces the Boolean, and `deliverCancel` moves the handler to `cancelReceived`. A canceled result needs that receipt.
- **Two faulty resets as controls.** `forgetsCancelOnReset` (N7) and `truncatesOnReset` (N4's mutation, which the specimen left hand-reviewed).
- **A channel that redelivers once** (`Redelivery.once`, `Completion.retried`), for N9's two variants.
- **A schedule-to-close deadline** (`Knowledge.expired`, the baseline's `scheduleToClose` timer), for N8.
- **Two monitors,** `singleOutcome` and `cancelPrincipal`, beside the fixture's two.
- **Assumptions and progress claims,** `outcomeReachesOwner` and `retainedReachesOwner`, from the specimen's unsupported block.
- **An entity of its own,** `nexusRequest` keyed by `requestId` (finding F9).

### What the checks report

186 receipts: 156 Queries and 30 progress parts. `TestNexusCloseResults` pins every kind and the monitor a counterexample names, in one whole-map equality.

- **N1.** `rejectAfterClose.closedThenFinished` is a counterexample over the specimen's three rows, answered `rejectedPermanent`, ending in `closed-none-done-succeeded-none-none-none`, 7 states explored. The find `lostAfterReset` gives step 4 and the stuck state `resetOpen-none-done-succeeded-none-none-none`. The free search ends with `failed`, as the specimen says. `any.ackOnlyWhenKept` names the `retainedOutcome` monitor.
- **N1'.** `retainAndRoute` answers step 3 `retained` and the reset records `workflowReset, outcomeReapplied, nexusOperationCompleted`. Verified, 9 states.
- **Transient against permanent.** Two finds over one path end with `rejectedTransient` (report still in flight) and `rejectedPermanent`.
- **N2, N2'.** `ackByOriginal` gives the specimen's three rows for both promises, pinned and free, 4 states. The corrected design ends in `resetOpen-none-done-failed-none-none-successor-failed`, 6 states.
- **N3.** `ackByOriginal` is violated at the last step with the intent kept (`resetOpen-requested-callerWorkflow-done-canceled-none-none-none`). Request, receipt, effect and knowledge are four steps, each found apart.
- **N4.** Verified in the three policies, 8 states. `truncatesOnReset` loses the outcome at the reset.
- **N5.** The second delivery is `accepted`, changes no knowledge and records nothing.
- **N6.** `resetAfterRetention` and `resetBeforeRetention` are two finds with different witnesses.
- **N7.** `forgetsCancelOnReset.canceledAcrossReset` is a counterexample naming `cancelPrincipal`, state `lost`, at the reset, with the three outcome monitors held on the same path.
- **N8.** With the deadline, both faulty designs keep their safety counterexample at the same rows, the timeout is found with nothing owed, no design has a stuck state, and all nine progress parts verify. The corrected design verifies `noUnnecessaryWait`.
- **N9.** `retainAndRoute outcomeReachesOwner fair-cycle` is a counterexample whose loop is `complete` answered `rejectedTransient`. `retainAndRouteBoundedRetry`, which assumes `transientRejectionEventuallyAccepted`, verifies all three parts.
- **N10.** The deadlock witnesses of both faulty designs end in the lost-outcome state after four steps. Explored to the start only, all three parts are `unresolved`. Explored three steps deep, the deadlock is `unresolved`.
- **N11.** `retainedReachesOwner` under the recovery assumption has no fair cycle. `retainedWaitsWithoutRecovery` has one. Over the bounded channel all three parts verify within two steps.

### Baseline comparison

- **Compared.** The first delivery of each completion to an open caller: outcome `accepted` and the fact `nexusOperationCompleted`, `Failed` or `Canceled`, equal to the product's and the protocol's `started` rows, on all nine designs. The `complete` action declaration. The baseline's `asyncCompletion` and `asyncFailure` Queries with its `completionSucceeds` and `completionFails` Properties, by their last step. Four evidence lines.
- **Excluded by name, with reasons.** The other 23 baseline claims. The test reads the Go source and fails on a claim that is neither compared nor excluded.
- **Labeled.** Each of the 20 other Properties is listed in `closePromises` as an authored design promise with the oracle it states. A Property that is neither the baseline's nor labeled fails the test.

### Test-first record

- Both test files were written before any Model. 21 tests failed on the missing IR (`red-01-no-ir.log`).
- The first run against lifted IR failed 4 tests (`gen-01.log`). All four were my reading of a representation: Go's empty facts are `[]string{}`, the Model lists monitors sorted by id, and `Table.Stuck` is one state (two tests). No expected behavior changed.
- One wrong expectation of mine (`gen-02.log`): I expected all three progress parts `unresolved` one step deep. The cycle is a self-loop at depth one and is found there, which is a witness and no open prefix. The test now explores to the start (all `unresolved`) and three steps deep (deadlock `unresolved`).
- `TestNexusCloseResults` passed on its first run against lifted IR, all 186 kinds and monitor names.
- Never red: `TestNexusClosePinnedSearchesExploreTheSpecimensStates`, `TestNexusCloseEvidenceNamesTheBaselinesEvents` and `TestNexusCloseProgressClaimsAreTheSpecimens`, written after the Model. Mutants fail each.
- Mutation over the checked-in IR, in the foreground with the bytes restored after each mutant (`mutants-01a.log` to `mutants-02a.log`, script beside them): 58 mutants. Three survived the first run. Dropping the delivery's fairness and swapping a progress claim's `from` and `to` changed no verdict, so `TestNexusCloseProgressClaimsAreTheSpecimens` now pins the declarations and fails both. Dropping the timer's fairness changed nothing because no claim needs it, so I removed the declaration (decision 7). All 58 now fail a test.

### Decisions that differ from the task text or the specimen

1. **The state is richer than the sketch's, so keys and free-search counts differ.** Free searches explore 64, 79 and 74 states against the sketch's 61, 76 and 71. The pinned paths the additions do not touch explore the specimen's counts (7, 9, 4, 6, 8), which are pinned. The sketch's own counts stay pinned on the fixture.
2. **N3 has five steps and runs within `five`.** The delivery of the cancel request is a step. The specimen's "a canceled result needs a cancel intent" is here "needs the handler's receipt".
3. **Free searches run within twelve steps, not six.** Every verified one is checked to be past its table's depth.
4. **One principal.** A second costs half again the catalog, and the suite's time follows the catalog (finding 2).
5. **The deadline designs use the bounded channel,** and `outcomePreserved` and `settled` accept an operation the deadline resolved.
6. **`Redelivery.once` bounds lost acknowledgments too,** not only transient rejections. Otherwise the bounded machine keeps a cycle.
7. **Three assumptions are names only.** `handlerReportsUntilAckOrPermanent` and `retentionSurvivesCrash` stand for what the steps already do and no modeled fault tests them. `scheduleToCloseExpires` makes nothing fair: the machines with the timer have no cycle.
8. **N11 is partial.** Recovery is the reset of a closed run. No owner crash is modeled, and `inconclusive` is a runtime assessment for tasks 7 and 11.
9. **A duplicate delivery records no second history event.**
10. **No Property states N7.** Any path that could fail it passes the reset first, where the monitor reports, so the monitor is the assessment.
11. **The channel is hand-written alternatives,** as in the sketch. As I read SEMANTICS Channels, a derived delivery cannot answer a rejection that keeps the message.
12. **`Claims.scala` imports `umpire.{Answer as _, *}`.** The framework's `Answer` otherwise shadows the designs'.
13. **`run.sh` comments edited** where they name the IR files.

### Findings

1. **The baseline and the specimen disagree on a completion after the operation is over.** Go answers `notFound`; N5 answers the duplicate `accepted`. `TestNexusCloseLateCompletionDiffersFromTheGoModel` pins both sides. The designs do reject a completion after the deadline, as `rejectedPermanent`.
2. **The `goir` suite takes 24 s, up from 12.7 s.** Each machine's rows are read over its whole catalog (6,720 states), nine machines, three times: twice in `Check` and once for the tables the tests read. A profile shows 42% in garbage collection.
3. **On a faulty design a free verify of any Property reports the monitor,** as task 4 found. Five Properties that hold of every step are therefore shown on the corrected design only.
4. **`singleOutcome` is never violated.** No design delivers a second result, so only the monitor list and a mutant of its verdict exercise it.

### Gaps outside the Touches (not changed)

1. `Report` exposes no tables, so a test that reads rows pays a third `Build`.
2. An `Entity`'s key is not lifted. The IR carries `nexusRequest` and not `requestId`, so F9 is addressed in Scala only.
3. `goDeclaredClaims` in `activity_parity_test.go` names its directory, so `goClaims` repeats it for `model/go/nexuscaller`.
4. `model/scalav2/README.md` and the Makefile comment do not name `ir/nexus-close.json`.
5. No Scala-native test reads the new package. `scala/temporal/test` is outside the Touches, and the Scala search answers no Query over a monitored machine.
6. Neither task-18 defect was met: the Model has no composition.

### Left undone

1. Operation and request identity roles, observations and scenario DAGs (E8 to E10).
2. A second principal, a close-initiated cancellation, and more than one operation, handler or successor.
3. Storage loss of a retained outcome and an owner crash.

### Incident

`scala-cli fmt <dir>` run on the package directory used scalafmt's defaults and rewrapped both new files at 80 columns. I reflowed them with the project configuration and `make lint-scala` passes. Two `.scala-build` directories it created in the new package are removed.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (baseline, before any edit) | 0 |
| same, final | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/nexuscaller/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-gen-scala` (other IR byte-identical) | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala` | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` (0 `[error]` lines) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false` (0 issues, after one testifylint fix) | 0 |
| snapshot without `model/scala` and `gen/`: `make umpire-check-scala`, `make lint-scala` | 0, 0 |
| same snapshot with one byte appended to `ir/nexus-close.json`: `make umpire-check-scala` | 2, as intended |

No `flowctl gate` receipt was attempted, since the tree is dirty outside the ignore set. Logs are under `.flow/tmp/fn-107/task5-logs/`.

### Authoring measurements (R1)

Warm, one run each: compiling `scala/` 1.3 s, packaging and lifting the nineteen roots 3.4 s, the Nexus close Go tests 11 s.

stage: implement - ran (worker subagent, session model claude-opus-5-5)

### Review round 1

Codex returned SHIP with two P3 findings. Both are closed. No mutation run, no review, no git mutation.

1. **P3, the catalogs were interpreted three times (valid).** `checkedOnce` in `nexus_close_test.go` runs `check`'s own sequence over one `newChecker` and keeps that checker's machines beside the report, so the row assertions read the interpretation the checks read. No non-test `goir` file changed. `TestCheckedOnceIsCheck` holds it to `Check` on two small Models (whole plain receipts, and Build's rows and fingerprints) and to Build's error for a machine with no table. `activity_system_test.go` reuses it in one line.
   - **Suite time:** `goir` 24.1 s before, 21.1 s after. The Nexus close loader went from 9.1 s to 7.0 s.
   - **What remains is not in the tests.** `Check` itself interprets twice, once for the checks and once for the witness replay, and a profile still shows about 39% in garbage collection. Removing more needs a change in `goir`'s non-test code or smaller catalogs.
   - **A cost of the fix:** the helper repeats `check`'s loop, about 25 lines. A receipt kind added to `check` and not to the helper shows in `TestCheckedOnceIsCheck` only if one of its two Models produces it.
2. **P3, the docs named one IR (valid).** `model/scalav2/README.md` (diagram, run commands, layout table) and the Makefile comment above `umpire-check-scala` now name `ir/nexus-caller.json`, `ir/activity.json`, `ir/activity-system.json` and `ir/nexus-close.json`, each with a line on what it is. Nothing else in either file changed. The README's "Results" section still says Properties and the standalone activity are not lifted; that was outside what the widened Touches named, and I left it.

Files this round: `goir/nexus_close_test.go`, `goir/activity_system_test.go`, `model/scalav2/README.md`, `Makefile`. Before-copies are under `.flow/tmp/fn-107/task5-before-r1/`.

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (goir 21.1 s) | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

The conductor also brought the README's Results section up to date (what lifts now, what does not).

The work is uncommitted; the owner makes the commits. The task diff and the review output are under `.flow/tmp/fn-107/task5/`.

stage: implement - ran (worker subagent, session model claude-opus-5-5)
stage: impl-review - ran (codex:gpt-5.6-sol:high, session 01a0f66c-ce7b-7ff1-94a4-bbe29005b00a over the uncommitted task diff; round 1 SHIP with two P3 findings, both closed afterwards and not re-reviewed)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0, before any edit), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/nexuscaller/... (rc=0), GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala (rc=0, after make umpire-gen-scala rc=0 with byte-identical IR), GOFLAGS=-tags=test_dep make lint-scala (rc=0, 0 [error] lines), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc=0), GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false (rc=0, 0 issues), snapshot without model/scala and model/scalav2/gen: make umpire-check-scala (rc=0), make lint-scala (rc=0); with one byte appended to ir/nexus-close.json: make umpire-check-scala (rc=2, as intended), IR mutation, foreground with restore: 58 mutants, 58 fail a test (3 survived the first run; see summary), review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0; goir 24.1 s before, 21.1 s after), review round 1: GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0), review round 1: GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false (rc=0, 0 issues), conductor final: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc 0), conductor final: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), conductor: make lint-scala (rc 0, 0 [error] lines; before the test/doc-only P3 round), codex impl-review: .flow/tmp/fn-107/task5/t5-r1.md (SHIP)
- PRs: