---
satisfies: [R1, R4, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.4 Check activity parity, admission race, and scoped queue refinement

Touches: [model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/ir/activity*.json, model/scalav2/goir/activity*_test.go, model/scalav2/run.sh]

## Description
Deliver the executable activity proof point using the existing Go activity baseline and the reviewed provider/race controls.

**Size:** M
**Files:** model/scalav2/scala/temporal/standaloneactivity/Model.scala and Claims.scala, proposed System.scala beside them, generated activity IR, differential fixture/test.

### Approach
- Lift the existing activity behavior and compare all states/actions/results/claims in the explicit intersection domain, including disabled actions. Keep the baseline Go policy unchanged.
- Add independently identified operation/attempt/delivery state, current-eligibility admission, and the deliberately faulty stale-eligibility control.
- Replace the opaque durable queue with the detailed provider from the sketch; explore crash/commit/ack cuts and competing timers. Include a faulty provider.
- Author ordinary process crash and committed-storage loss as separate transitions with distinct fault IDs and receipt entries. Ordinary crash preserves committed queue state; enable destructive storage loss only through its explicitly selected fault assumption. Include a provider mutation that wrongly loses committed state on an ordinary crash.
- Pin expected witnesses and report source-specific coverage/exclusions. Stop downstream integration if parity or corrected refinement fails.

### Investigation targets
**Required:** model/go/standaloneactivity/model.go:301; model/go/standaloneactivity/claims.go:24; model/go/standaloneactivity/pins_test.go:122; model/scalav2/scala/temporal/standaloneactivity/Model.scala; chasm/lib/activity/tasks.go:75.
**Optional:** tests/activity_parity_test.go:544; chasm/lib/activity/statemachine.go:423.

### Quick commands
`make umpire-check-scala`; `make lint-scala`; `mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/standaloneactivity/...`.

## Acceptance
- [ ] Exhaustive correspondence within the declared baseline domain covers disabled behavior and results.
- [ ] Generic search finds the stale-delivery control and excludes it for the current-eligibility design.
- [ ] Scoped queue substitution passes and the violating provider fails with a replayable witness.
- [ ] Duplicate delivery, pre-pause admission, timer ordering, and crash cuts match the trace oracles; ordinary crash preserves committed records and separately enabled storage loss has its own transition/receipt/control.

- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing with that tree absent.

## Done summary
The lifted standalone activity Model now equals `model/go/standaloneactivity` on every state, class, row, result, disabled pair, refinement row, action declaration and on ten of its eleven Queries. The eleventh, `stoppedWorkerStartsNothing`, is outside the compared domain because goir's admission refuses its Property (gap 1). The system contract in `System.scala` finds the stale-delivery control, excludes it for the current-eligibility design, passes the scoped queue substitution and rejects both violating providers with witnesses that replay. Nothing is committed and the task stays `in_progress`.

### Files

Changed (before-copies under `.flow/tmp/fn-107/task4-before/`):
- `model/scalav2/scala/temporal/standaloneactivity/Claims.scala`: the private varargs helper `path` is gone and its eight Scenarios are declared directly, which is the form the lifter folds. Names, schedules and comments are unchanged.
- `model/scalav2/run.sh`: lifts the activity roots into `ir/activity.json` and `ir/activity-system.json`, compares or rewrites them with the fixtures, and prints the stale message it used to skip (decision 10).

Added:
- `model/scalav2/scala/temporal/standaloneactivity/System.scala` (1061 lines, 779 without blanks and comments): the record, both admission designs, two monitors, the opaque queue, the detailed queue, two violating providers, the storage-loss pair, seven compositions and their Queries.
- `model/scalav2/ir/activity.json` (292,385 bytes) and `model/scalav2/ir/activity-system.json` (786,609 bytes), both written by `make umpire-gen-scala`.
- `model/scalav2/goir/activity_parity_test.go` (296 lines, 8 tests) and `model/scalav2/goir/activity_system_test.go` (605 lines, 13 tests).

`Model.scala`, `model/go/standaloneactivity`, the lifter, the Scala framework, `goir`'s non-test files and `model/scala` are byte-unchanged by this task. `git status` shows `model/scala/temporal/test/Lean.test.scala` modified, as it was before the task started.

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | Exhaustive correspondence within the declared baseline domain covers disabled behavior and results | pass, with one Query excluded (gap 1) and one ordering difference (finding 1) | `TestActivityTablesEqualTheGoModel`, `TestActivityDisabledBehaviorIsTheBaselines`, `TestActivityCompositionEqualsTheGoModel`, `TestActivityRefinementEqualsTheGoModel`, `TestActivityActionsEqualTheGoModel`, `TestActivityClaimsEqualTheGoModel`, `TestActivityEvidenceIsInCatalogOrder`, `TestActivityCrossEntityClaimIsOutsideTheDomain` |
| 2 | Generic search finds the stale-delivery control and excludes it for the current-eligibility design | pass | `TestActivitySystemResults`, `TestActivityStaleDeliveryAfterPause` |
| 3 | Scoped queue substitution passes and the violating provider fails with a replayable witness | pass | `TestActivityQueueSubstitution`, `TestActivityViolatingProviders` |
| 4 | Duplicate delivery, pre-pause admission, timer ordering and crash cuts match the trace oracles. Ordinary crash preserves committed records, and storage loss has its own transition, receipt and control | pass | `TestActivityDuplicateDelivery`, `TestActivityAdmittedBeforePause`, `TestActivityCompetingTimers`, `TestActivityCrashCuts`, `TestActivityFailedCommit`, `TestActivityTerminalFinality`, `TestActivityStorageLossIsAssumed`, `TestActivityFreeSearchesReachEveryState`, `TestActivitySystemExclusionsAreDisabled` |
| 5 | Legacy `model/scala` files unchanged, and v2 generation, lifting, native tests and lint pass with that tree absent | pass | `final-iso.log`: a copy of the tree without `model/scala` and without `model/scalav2/gen` ran `make umpire-check-scala` (rc 0) and `make lint-scala` (rc 0, no `[error]` line). `TestNoInputNamesTheLegacyTree` and `TestIRSourcePositionsResolveInsideScalav2` read the two new IR files |

### Parity coverage (AC 1)

| Compared | Size |
|---|---|
| `activityProduct` | 9 states, 11 classes, 43 rows, 56 disabled pairs |
| `activityProtocol` | 288 states, 22 classes, 1788 rows, 4548 disabled pairs, 238 reachable |
| `activityWorker`, `polling` | 2 states each, 2 and 3 rows |
| `standaloneActivity` composition | whole composed table |
| Refinement of the protocol machine | every row |
| Actions | 12 declarations: party, entity, schemas, inputs, results, examples |
| Queries | 8 `find` and 2 `verify`: outcome, explored and expanded counts, exercised flag, rows and witness with its Definition IDs |

Each table comparison is one whole-value equality over states, classes, outcomes, facts, starts, ends, reachable states, rows with results and explanations, the disabled pairs, stuck state, state fields, entity, family, Definition IDs and target fingerprint.

Outside the domain, and compared nowhere: the functional, canary and exploratory sets, `productWithoutControls`, the `attemptCount` observation, which timers are unobservable, the Definition IDs of Properties and Scenarios, and `stoppedWorkerStartsNothing`.

### What the system contract reports

`receipts.log` holds all 96 receipts with kind, explored count, limits and assumptions.

- **A1, A1', A5.** `staleAdmission.staleDelivery` is a counterexample `dispatch, control-pause, attemptStart` ending in `started-one-owed`. `currentAdmission` verifies it, and its step there is a stutter that records only `admissionRejected`. The stale design's refinement is `refinement-rejected` (unmatched) at row `paused-none-settled-attemptStart`, and its `through` Query carries that same rejection.
- **A2.** `admittedBeforePause` verifies in both designs, alone and over both queues.
- **A3, A8.** The stale design's second admission reaches `started-two-owed`. The corrected design answers a redelivery with `admissionRejected` and admits nothing. Over the detailed queue the redelivery takes `queue_ackLoss` or a `queue_crash` before the acknowledgment.
- **A4.** The stale design starts an activity that is over. The `terminalFinality` monitor reports it with state `reopened`.
- **A6.** On `activityProtocol` and on both designs, `scheduleToStart` and `scheduleToClose` are two rows of one state with distinct facts, and each `find` is found.
- **A7.** `scheduled-none-settled_committed-admit` has a second result with `activity_admissionCommitFailed`, no `attemptAdmitted`, the record still scheduled and the message still `deliveredOnce`. `settle` is disabled there.
- **A9.** `matchingQueue` refines `dispatchQueue` over all 81 rows. Its four crash cuts are `found`, and every crash row keeps a held message held. `forgetfulQueue` (V1) is rejected with witness `enqueue, addActivityTask, crash`, `volatileQueue` (V2) with `enqueue, addActivityTask, persistTask, crash`. Neither receipt names `storageLoss`.
- **Storage loss.** `storageLoss` is an action of `lossyMatchingQueue` and of `dispatchQueueUnderStorageLoss` only, with a Definition ID apart from `crash` and `ackLoss`. Every receipt over `lossyMatchingQueue` and `currentOverLossyMatching` names `storageLoss`, and no other receipt does.
- **Substitution.** `currentOverMatching` and `staleOverMatching` verify the replacement. Queries over `currentOverQueue` name `dispatchQueue.opaque`, and the same Queries over `currentOverMatching` name nothing. The stale design's counterexample survives as `dispatch, queue_addActivityTask, queue_persistTask, activity_control-pause, admit`.
- **Free searches.** The 13 free searches of the corrected design run within limits past the depth of their tables (3, 4, 9 and 10 steps), so no bound cuts them.

### Test-first record

- I wrote both test files before any activity IR existed. All 18 tests then failed on the missing files (`red-01-no-ir.log`).
- The first lift was refused at admission (`gen-01.log`, `System.scala:888` and `Claims.scala:235`: "a Property of a composition is about every step"). That is gap 1.
- The first run against admitted IR failed 7 tests (`gen-02.log`). Four were my comparison reading the wrong representation: `Table.Owner` instead of `OwnerName()`, nil against empty slices twice, and an action ID compared across two tables. Each fix is in the test's reading, and no expected behavior changed.
- The rest were two wrong expectations of mine (decisions 11 and 12), one real difference from Go (finding 1) and one behavior of the binding (gap 3).
- `TestActivitySystemResults`, which pins all 96 kinds, passed on its first run against admitted IR.
- `TestActivityFreeSearchesReachEveryState` was written with the limit change it checks, so it was never red. The mutant that lowers one limit to seven fails it.
- Mutation over the checked-in IR, in the foreground with the bytes restored after each mutant (`mutants-03.log`, script beside it): 25 mutants, 23 fail a test. The two survivors drop `visible` from `currentAdmission` and from `matchingQueue`. Both designs hold under either rule, so no result changes (follow-up 3).

### Decisions that differ from the task text or the specimen

1. **The record and the queue are separate machines, checked together by composition.** The specimen's supported sketch folds the message into the record's state. Row keys therefore differ from the specimen's, and I pin no explored-state count from it. The sketch's own counts stay pinned on `Admission.scala.fixture` by `checking_test.go`.
2. **Each design exists twice.** `currentAdmission` and `staleAdmission` take any delivery, refine `activityProduct` and name the monitors. `currentRecord` and `staleRecord` are the same steps without monitors or refinement, because goir answers no Query over a composition whose member names a monitor.
3. **The delivery's stamp is not data.** A sync passes nothing between members, so admission cannot read the message. With no unpause in scope the phase check is the whole eligibility check.
4. **`Answer` is a record field.** It couples the acknowledgment to admission's answer, so a failed commit cannot be acknowledged.
5. **A redelivery that meets the committed attempt records `admissionRejected`.** The specimen says "answered idempotently". Both admit nothing, and the Model keeps one fact.
6. **Both deadlines are always armed in the admission designs.** A6 is checked there and on `activityProtocol`.
7. **V1 is a crash that loses an invocation or a sync match.** `RefineTables` reads every row, reachable or not, so the good provider's state type cannot hold a state it would lose a message from. The specimen's "history discards its obligation at the invocation" is what V1's crash at `invoked` or `reserved` stands for. The rejection is at the crash step, as the specimen says.
8. **Storage loss is a pair of machines.** `lossyMatchingQueue` refines `dispatchQueueUnderStorageLoss`. No machine without the assumption binds the action.
9. **The interface's storage-loss assumption has its own name, `dispatchQueue.storageLoss`.** See gap 3.
10. **`run.sh` beyond the roots.** The stale-IR message never printed, because `diff | head` fails under `pipefail` before it. Both occurrences now end in `|| true`, and the unused `fixtures` variable is gone. The gate failed on stale IR before and after (`final-iso-stale.log`, rc 2).
11. **A free search of a design alone reports the monitor's violation.** `staleAdmission.any.notAdmittedWhilePaused` is a counterexample `attemptStart, attemptStart` naming `atMostOneActiveAttempt`, which SEMANTICS makes a counterexample of a verify. I expected `control-pause, attemptStart`. The Property's own free-search witness is pinned over `staleOverQueue`, whose members name no monitor.
12. **A composed row carries no explanation.** I expected the commit failure's `because` on the composed row. It is on the member's row, and the test reads it there.
13. **The free searches over the detailed queue run within twelve steps,** past the table depth of ten. Within seven steps the search explored 79 product states of a table that reaches 114 states.
14. **`failedCommitKeepsTheMessage` is about every step,** written as an implication, because of gap 1.

### Finding

1. **Evidence order.** For `activityProtocol` the IR lists evidence in fact-catalog order, as SEMANTICS Machines 5 says, with `attemptCount` last. Go lists it as declared, with `attemptCount` first. The lines are equal as a set, and Definition IDs and the target fingerprint are equal. `TestActivityEvidenceIsInCatalogOrder` pins the difference.

### Gaps outside the Touches (not changed)

1. **`goir/load.go:969-972` refuses a `when` on a composition's Property,** and `admission_test.go:650` pins that. `model/go` and the Scala framework answer `WhenAction` on a composition. So `stoppedWorkerStartsNothing` cannot be lifted and admitted, and its Go answer (`verified-within-limits`, exercised) is compared with nothing. `TestActivityCrossEntityClaimIsOutsideTheDomain` fails when admission starts to take such a Property.
2. **The lifter folds neither `Composition.own(member, class)` in `actionKeys` nor a declaring helper with varargs.** The second forced the `Claims.scala` rewrite. The first means `stoppedBeforeRetry` needs literal keys before it can be a root.
3. **A replacement discharges an assumption the replacing member also names.** SEMANTICS says the member's own assumptions stay. With one `storageLoss` on both machines, the composition's receipts lose it. The mutant "interface and provider share the loss assumption" shows this. Two names keep the receipts right today.
4. **A sync carries no value between members, a composition declares no refinement, and a member's monitors make its Queries unsupported.** Decisions 2 and 3 follow from these.
5. **`make lint-scala` exits 0 while it prints `[error]`.** It printed two "unused explicit parameter" errors for my monitor functions with rc 0 (`iso.log:99-116`). I fixed them. The gate's rc alone proves nothing, so I also counted `[error]` lines: 0.
6. **Stale prose.** `model/scalav2/README.md` still says the standalone activity is not lifted, and the Makefile comment names only `ir/nexus-caller.json`.

### Left undone

1. One logical activity. The task's optional scope of two is not modeled.
2. Unpause, cancel request, terminate, and failed or canceled answers are disabled in the admission designs. `TestActivitySystemExclusionsAreDisabled` pins them as disabled pairs with no hole.
3. No control in the feature Model isolates the visible-projection rule. Task 3's fixture controls cover the rule itself.
4. No Scala-native test reads `System.scala`. `scala/temporal/test` is outside the Touches, and the Scala search answers no Query over a monitored machine.
5. The specimen's proposed `observe`, `scope` and scenario-DAG declarations (E8 to E10) belong to later tasks.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (baseline, before any edit) | 0 |
| same, final (goir: 394 passing tests and subtests, 0 failed, 5 Lean-dump skips) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/standaloneactivity/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala` (run after the last `umpire-gen-scala`, so regeneration is byte-identical) | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` (0 `[error]` lines) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false` (0 issues, after one import-order fix) | 0 |
| snapshot without `model/scala` and `gen/`: `make umpire-check-scala`, `make lint-scala` | 0, 0 |
| same snapshot with one byte appended to `ir/activity-system.json`: `make umpire-check-scala` | 2, as intended |

No `flowctl gate` receipt was attempted, since the tree is dirty outside the ignore set. Logs are under `.flow/tmp/fn-107/task4-logs/`.

### Authoring measurements (R1)

Warm, one run each: compiling `scala/` 3.8 s, lifting the six baseline roots 2.8 s, the activity Go tests 0.7 s with a built test binary and 6 s with the build.

### Review round 1

The Codex review returned NEEDS_WORK with one introduced P2. It is valid, and it was red before the fix.

**P2, `cancelRequestedWhileStarted` was silently outside the comparison (valid).** Go and Scala both declare the Property on the protocol machine and neither declares a Query that asks it, so no root lifted it and the ten compared Queries never read it.

- **The Property is lifted and compared.** `Claims.scala` declares `cancelRequest`, a `find` of `cancelRequestedWhileStarted` over the baseline's own `cancelRequestedThenCanceled` path within `four`. `run.sh` lifts it as a root of `ir/activity.json`. The parity test builds the same pairing from Go's declarations (`CancelRequestedThenCanceled.Find(..., CancelRequestedWhileStarted, Four)`) and compares the two answers as it compares every other Query. The Go baseline is unchanged, and `cancelRequest` is in no Scala set.
- **The compared domain checks itself.** `TestActivityClaimDomainIsWholeOrExcluded` parses the source of `model/go/standaloneactivity` for every `Property`, `Scenario`, `path`, `Find`, `Verify` and `VerifyRefined` call with a literal name. It fails on a declaration that is neither compared nor in `excludedClaims`, on an exclusion Go does not declare, on a compared claim Go does not declare other than `query cancelRequest`, and when the lifted Model's Properties, Scenarios and Queries differ from the compared set. It also requires exactly one declaration whose name is no literal, the Scenario inside Go's `path` helper, so a new indirection fails it too.
- **Nothing else was omitted the same way.** Against the tests as they stood, the new test reports exactly one unaccounted declaration, `property cancelRequestedWhileStarted` (`r1-red-omission.log`). Go declares 31 claims: 10 Properties, 8 Scenarios and 10 Queries on the product and protocol machines, all now compared, and the cross-entity claim's Property, Scenario and Query, which are the three entries of `excludedClaims`.
- **A second weakness found while fixing.** A mutant that repoints `cancelRequest` at `canceledByWorker` survived, because that pairing answers exactly as `cancel` does. The comparison now includes each Query's Property and Scenario names on both sides, which fails that mutant and one that repoints `completion` at another Scenario.

Red before the fix:
- `TestActivityClaimDomainIsWholeOrExcluded`, "declared by Go, and neither compared nor excluded: [property cancelRequestedWhileStarted]" (`r1-red-omission.log`, the test run against the ten-Query comparison).
- `TestActivityClaimsEqualTheGoModel`, `no receipt "query activityProtocol cancelRequest"`, and `TestActivityClaimDomainIsWholeOrExcluded` on the lifted set (`r1-red.log`, before the Scala Query and the root existed).
- The surviving pairing mutant, in `r1-mutants.log`.

Green after: both tests pass, and `cancelRequest` is `found` on both sides with equal witness, rows, explored and expanded counts. Six IR mutants of the new comparison all fail a test (`r1-mutants.log`).

**A bound that remains.** A Property is compared by the answers of the Queries that ask it, over the baseline's paths. The predicate itself is a Go function on one side and an IR expression on the other, and nothing compares the two on every step of the table.

Acceptance item 1 now reads: 11 Queries compared (8 `find` and 2 `verify` of the baseline, and `cancelRequest`), every Property and Scenario of the product and protocol machines compared through them, and the cross-entity claim excluded by name.

Files this round: `model/scalav2/goir/activity_parity_test.go` (432 lines, 9 tests), `model/scalav2/scala/temporal/standaloneactivity/Claims.scala` (one Query added), `model/scalav2/run.sh` (one root added), and `model/scalav2/ir/activity.json` regenerated. `ir/activity-system.json` is byte-identical. Before-copies of the round are under `.flow/tmp/fn-107/task4-before-r1/`. The two pre-existing defects the review classified are untouched.

Gates after the fix:

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (goir: 395 passing tests and subtests, 0 failed, 5 Lean-dump skips) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/standaloneactivity/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala` (after `umpire-gen-scala`) | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` (0 `[error]` lines) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

The snapshot run without `model/scala` was not repeated this round. The round added no path and no build input.

### Review round 2

The Codex re-review accepted the `cancelRequest` fix and the inventory, and left one introduced P2: the bound I named at the end of round 1. It is valid. The round-1 suite passed on the reviewer's mutant, and the new test fails on it.

**P2, Properties were compared only on the baseline's paths (valid).** `TestActivityPropertiesAgreeOnEveryRow`, in the new file `goir/activity_properties_test.go`, compares every compared Property on every row of its machine.

- **How a row is evaluated.** Each row becomes a one-step pinned Scenario from the row's state by the row's class, and each Property is verified over it within one step. Go answers through `Scenario.Verify` and `Scenario.VerifyRefined` with `Query.Answer`. The IR answers through `Check`, on a copy of the lifted Model to which the test adds the same Scenarios and Queries. The test reads no private field and holds no predicate of its own.
- **What is compared per row.** Whether the Property is about the step (the answer's exercised flag), whether it holds there (verified or counterexample), and the counterexample's witness.
- **Coverage.** 17,966 answers per side: the 8 protocol Properties and the 2 product Properties read through the refinement on all 1,788 protocol rows, and the 2 product Properties on the 43 product rows. The rows include unreachable states. The answers are not all alike: `retryCompletes` holds on 3 rows and fails on 69, and `terminated` and `cancelRequestedWhileStarted` each hold on 144 and fail on 120.
- **The IR side names no machine and no Property.** It asks every Property the Model declares, and each product Property again through every machine that refines its machine. A Property on one side only shows as a missing answer. The round-1 inventory already requires the lifted Properties to be exactly the compared ones.
- **The path comparisons stay as they were,** in `TestActivityClaimsEqualTheGoModel`.

Proof that it bites, `TestActivityPropertyRowsCatchWhatThePathsMiss`. Each subtest mutates a copy of the lifted Model, requires that the mutant is admitted and that all eleven path Queries still answer as Go does, and then requires the named rows among the row disagreements.

| Mutant | Kind | Path Queries | A row that differs |
|---|---|---|---|
| `cancelRequestedWhileStarted` also requires `attempts == 1` (the reviewer's example) | same-step predicate | all agree | `scheduled-0-unset-unset-unset-control-requestCancel` |
| `canceledByWorker` is about every class of `attemptResult` | trigger | all agree | `started-1-unset-unset-unset-attemptResult-completed` |
| `pausedIsNotDispatched` also forbids reaching `terminated` | transition predicate | all agree | `scheduled-control-terminate` on the product machine, and `scheduled-0-unset-unset-unset-control-terminate` through the refinement |

Red before the fix: the same three mutants, applied to `ir/activity.json` on disk with the bytes restored after each, all survived the round-1 suite (`r2-red-survivors.log`). Green after: each fails `TestActivityPropertiesAgreeOnEveryRow` and no other test (`r2-green-killed.log`, script `r2-mutants.py`). Both new tests passed on their first run against the unmutated IR, so the two sides did agree on every row.

Limits of the new check:
- **One answer per row stands for one result because every baseline row has exactly one.** Both sides require that. A row with several results would fail the test and need a Scenario per result, which the public search does not offer: a pinned Scenario selects a class, not one of its results.
- **The Go side lists the ten Property values by name,** since the typed API takes typed Properties. A Property added to Go and lifted, and not added to that list, fails as "Go has no answer".
- **Still outside:** `startedByPollingWorker`, excluded with the cross-entity claim, and the system contract's Properties, which have no Go counterpart.

Files this round: `model/scalav2/goir/activity_properties_test.go` (new, 364 lines, 2 tests). No Scala, no IR and no `run.sh` change, and both IR files are byte-identical to round 1.

Gates after the fix:

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (goir: 400 passing tests and subtests, 0 failed, 5 Lean-dump skips, 12.7 s) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/standaloneactivity/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala` | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` (0 `[error]` lines) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false` (0 issues, after one switch-style fix) | 0 |

The snapshot run without `model/scala` was not repeated. The round added one Go test file.

stage: implement - ran (worker subagent, session model claude-opus-5-5; two fix rounds)

Follow-up scheduled as task 18: replacement dropping a same-named assumption of the replacing member (`model/go/umpire/compose.go`), and goir admission refusing a `when` on a composition's Property, which keeps `stoppedWorkerStartsNothing` outside parity. Until it lands the interface's assumption is named `dispatchQueue.storageLoss` and the parity exclusion list holds the three declarations of that cross-entity claim.

The work is uncommitted; the owner makes the commits. The task diff and the three review outputs are under `.flow/tmp/fn-107/task4/`.

stage: implement - ran (worker subagent, session model claude-opus-5-5; two fix rounds)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f625-bb93-7f62-8bf4-28e619ec8609 over the uncommitted task diff; rounds 1-2 NEEDS_WORK with one introduced finding each, fixed; round 3 SHIP; two pre-existing findings scheduled as task 18)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0 before any edit; .flow/tmp/fn-107/task4-logs/baseline-go.log), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0; goir 394 passed, 0 failed, 5 Lean-dump skips; final-go-test.log), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/standaloneactivity/... (rc=0; final-go-test-task.log), GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala (rc=0, after the last umpire-gen-scala; final-check-scala.log), GOFLAGS=-tags=test_dep make lint-scala (rc=0, 0 [error] lines; final-lint-scala.log), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc=0; final-vet.log), GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false (rc=0, 0 issues; final-lint-goir.log), snapshot without model/scala and model/scalav2/gen: make umpire-check-scala rc=0, make lint-scala rc=0 (final-iso.log), same snapshot, one byte appended to ir/activity-system.json: make umpire-check-scala rc=2 with 'ir/activity-system.json is stale' (final-iso-stale.log); restored, rc=0 (final-iso-2.log), red before the IR existed: 18 activity tests failed on the missing ir/activity*.json (red-01-no-ir.log), first run on admitted IR: 7 of 18 tests failed (gen-02.log); see the summary's test-first record, IR mutation, foreground with restore: 25 mutants, 23 killed, 2 result-equivalent survivors (mutants-03.log, mutants.py), ir/activity.json sha1 803ad77fc3d7984cd322f8d8c9deeaa531436841; ir/activity-system.json sha1 76e5512468340c38f48e0d7bfbfd0475c4d544b6, Review round 1, red: TestActivityClaimDomainIsWholeOrExcluded reports [property cancelRequestedWhileStarted] against the ten-Query comparison (r1-red-omission.log); TestActivityClaimsEqualTheGoModel has no receipt for cancelRequest (r1-red.log), Review round 1, green: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0; goir 395 passed, 0 failed, 5 Lean-dump skips; r1-go-test.log), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/standaloneactivity/... (rc=0; r1-go-test-task.log), Review round 1: GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala (rc=0; r1-check-scala.log), Review round 1: GOFLAGS=-tags=test_dep make lint-scala (rc=0, 0 [error] lines; r1-lint-scala.log), Review round 1: go vet -tags test_dep ./model/scalav2/... (rc=0; r1-vet.log); make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false (rc=0, 0 issues; r1-lint-goir.log), Review round 1: six IR mutants of the new comparison, all killed after the pairing is compared; one survived before it (r1-mutants.log), Review round 1: ir/activity.json sha1 9cad09be49a0f467c14a3324f671f143af64642e; ir/activity-system.json unchanged at 76e5512468340c38f48e0d7bfbfd0475c4d544b6, Review round 1: the snapshot run without model/scala was not repeated, Review round 2, red: the reviewer's mutant and two more, applied to ir/activity.json, all survive the round-1 suite (r2-red-survivors.log), Review round 2, green: each mutant fails TestActivityPropertiesAgreeOnEveryRow and nothing else (r2-green-killed.log, r2-mutants.py); TestActivityPropertyRowsCatchWhatThePathsMiss shows in-test that the path Queries agree on each mutant while the rows differ, Review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0; goir 400 passed, 0 failed, 5 Lean-dump skips; r2-go-test.log), Review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/standaloneactivity/... (rc=0; r2-go-test-task.log), Review round 2: GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala (rc=0; r2-check-scala.log), Review round 2: GOFLAGS=-tags=test_dep make lint-scala (rc=0, 0 [error] lines; r2-lint-scala.log), Review round 2: go vet -tags test_dep ./model/scalav2/... (rc=0; r2-vet.log); make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false (rc=0, 0 issues; r2-lint-goir.log), Review round 2: IR files unchanged (activity.json 9cad09be49a0f467c14a3324f671f143af64642e, activity-system.json 76e5512468340c38f48e0d7bfbfd0475c4d544b6); the snapshot run without model/scala was not repeated, conductor final: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc 0), conductor final: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), conductor: make lint-scala (rc 0, 0 [error] lines; before the test-only round 2), conductor final: make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT_FIX=false (0 issues), codex impl-review rounds: .flow/tmp/fn-107/task4/t4-r1..r3.md (final SHIP)
- PRs: