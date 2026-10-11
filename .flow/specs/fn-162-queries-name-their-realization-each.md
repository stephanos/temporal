# Queries choose their assessed Properties; each Property declared once

## Conversation Evidence

> user (turn 31): "temporal/model/temporal/features/activity/standalone/system/Heartbeat.scala has 3 separate machines; will/should we merge them?"
> user (turn 32): "why would we have both HeartbeatCompletion and HeartbeatThenCompletion spread across system.scala and its own file?"
> user (turn 34): "remind me, why not just 1 file?"
> user (turn 35): "yes, go for the real fix"
> user (turn 41): "yes both"
> user (turn 43): "apply recommendatoons"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 40% [user], 60% [paraphrase] -->

The standalone activity's system level declares 17 *claim containers*: copies of `ActivitySystem` (`Derived(ActivitySystem.unmonitored)` or `.rebind()`, with nothing changed) that exist only to hold one or a few Properties and Queries. `Heartbeat.scala` has three of them. fn-162.1 traced why (see Resolved via Codebase). A Case is already assessed against only its Query's one Property, plus the scenario machine's monitors. The copies exist because the lowering and the assessment require **exactly one realization per machine**, so a copy is how a Query says which realization runs it, or that none should. As a result, the same Property is declared again and again (the "completes" predicate six times), and the one-machine-per-file rule turns every copy into a file.

The fix: each Property is declared once, in the facet whose promise it states. Each Query names the realization that runs it, or is marked model-only, so Queries can sit on the base machine with no copies. A facet's claims then live with the facet. [paraphrase]

## Architecture & Data Models
<!-- scope: technical -->

- **Today (traced in fn-162.1):** a Case is assessed against its Query's Property plus the monitors of the scenario's machine (`tools/umpire/conformance/plan.go:134-145`). The lowering (`tools/umpire/lower/lower.go:156-175`) and the assessment (`plan.go:161-177`) pick a Query's realization as *the one realization of its scenario's machine*. Zero means "no realization", and two is an error. Claim containers exist to pick, or deliberately omit, a realization, and to decide which IR file a Query lands in.
- **Target:** a Property is declared once, on the machine or in its facet (fn-161), and identified by that declaration. A Query (or its scenario) names the realization that lowers and assesses it, or is marked model-only. The lowering and the assessment use that binding instead of "the one realization of the machine". A Case is assessed, as today, against its Query's Property plus the scenario machine's monitors. IR-file placement no longer depends on objects: fn-160's per-feature collection puts every Query of a feature in its one file, so `CompetingTimeouts`' Queries land in the activity's file. [paraphrase]
- Derived machines that change behaviour (rebind, restrict, extend, assuming, refining) stay. Only derivations that change nothing and exist as claim containers go away. [paraphrase]

## Edge Cases & Constraints
<!-- scope: technical -->

- A Query's verdict and its Case's expected outcome must not change. A Case assessed against a narrower or equal set of Properties than today must reach the same verdict. A Case whose set grows needs explaining. [inferred]
- Property and Query ids in the IR change when they move from subject objects to the facet or machine. Cases, manifests, pinned fixture trees and Go tests that name them change too. [paraphrase]
- Capability-generated Properties (`Closable`, `Pausable`, …) keep their own identities and their per-machine scope. [inferred]
- **Co-design with fn-140 (owner, 2026-10-10):** fn-140 (one-sentence witness Queries with explicit live expectations, Batch 2) also defines what a Query says. With R8, fn-162 adds nothing to the Query form: the realization names its Queries. The co-design reduces to fn-140 not adding realization information to Queries, with one owner keeping the two consistent.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Before changing anything, the way a Case's assessed Properties are selected today is traced end to end: model, IR, lowering, `umpire-run --model` and `umpire-assess --model`. It is recorded in the spec. [inferred] Errors: if selection depends on something other than the subject machine, the target design is revisited with the user before implementation.
- **R2:** *(superseded by R6: fn-162.1 found a Case is already assessed against one Property per Query)* A Query declares the Properties its Case is assessed against, by default the Property it finds or verifies, and the IR carries that set. The lowering and the Go assessment assess a Case against exactly that set. [paraphrase] Errors: a Query naming a Property of another machine fails the lift, naming both; a Case whose IR has no assessment set is refused by the assessment, not silently assessed against nothing.
- **R3:** Each Property is declared once. A predicate identical to another declaration of the same machine is refused or reported by the lifter. [paraphrase] Errors: the duplicate is named with both locations.
- **R4:** The standalone activity's claim-container derivations (derivations that change nothing and only hold Properties and Queries) are removed. Their Properties go to the facet or machine that states the promise, and their Queries keep their scenarios, limits, totals and expectations. [user] ("yes, go for the real fix") Errors: no error surface beyond R5.
- **R5:** *(superseded by R7: machine-scoped ids change too)* Every Query keeps its answer and total, and every Case keeps its expected outcome. IR and Case differences are limited to moved Property and Query ids and the assessment sets. The subject regression test is replaced by one pinning that each Case's assessment set is exactly its Query's. [inferred] Errors: any other difference is explained before it is accepted.
- **R6:** *(superseded by R8: the binding is declared on the realization side, because a system-level Query may not depend on the realization layer)* A Query names the realization that lowers and assesses it, or is marked model-only. The IR carries that binding, and the lowering and the Go assessment use it instead of requiring exactly one realization per machine. A Case is still assessed against its Query's Property plus the scenario machine's monitors. [paraphrase] Errors: a Query naming a realization of another machine fails the lift, naming both; a find Query with neither a realization nor the model-only mark is reported, as "no realization" is today; two Queries cannot make one machine's realizations ambiguous, because each Query names its own.
- **R7:** Every Query keeps its answer and total, and every Case keeps its expected outcome. IR and Case differences are limited to moved Property and Query ids, the realization bindings, derived Case names, and the target, state, action and fact ids that include a removed machine's name. All of these are listed in the rename map for fn-160.5's equivalence check. The subject regression test is replaced by one pinning each Query's realization binding. [paraphrase] Errors: any other difference is explained before it is accepted.
- **R8:** A realization names the Queries it runs (e.g. `Realizes(ActivitySystem, runs = Seq(…))`), so the binding respects the one-way layer dependency (realization → system → product, fn-160 R9) and the Query syntax is unchanged. A find Query that no realization names is model-only and is not lowered, which is today's "no-realization" standing; lint keeps reporting it. The IR carries each realization's Queries, and the lowering and the Go assessment use that instead of requiring exactly one realization per machine. A Case is still assessed against its Query's Property plus the scenario machine's monitors. [user] ("apply recommendatoons") Errors: a realization naming another machine's Query, or a Query named by two realizations, fails the lift, naming both.

## Boundaries
<!-- scope: business -->

- Changing what any Property or Query claims is out of scope. [inferred]
- Derivations that change behaviour stay. [paraphrase]
- Nexus and the task queue are migrated only if they have claim-container derivations; their behaviour-changing derivations stay. [inferred]

## Decision Context
<!-- scope: both — conditionally substructured -->

The user chose the real fix over relaxing the one-machine-per-file rule so a facet file could hold its claim subjects. [user] Relaxing would keep the clones and the duplicated Properties, and only hide them in fewer files. Scoping claims per Query removes the reason the clones exist. [paraphrase] Merging the three heartbeat subjects into one subject was rejected: every heartbeat Case would then be assessed against all three Properties, losing focus. [paraphrase]

**Decided (owner, 2026-10-10):** realizations name the Queries they run (R8), and unnamed find Queries are model-only. The merged duplicates keep their existing names, one home each: `completes` (Completion), `scheduleToStartFires` and `scheduleToCloseTimesOut` (Timeouts), `cancelRequested` (Cancellation). [user]

## Resolved via Codebase
<!-- fn-162.1 trace, 2026-10-10. Paths relative to temporal/. [inferred] marks a conclusion not read directly off code. -->

**Verdict: hypothesis corrected.** A Case is not assessed against its subject machine's Properties. Its assessment set today is exactly **the Query's one Property plus the monitors of the Query's Scenario machine**. Subject clones exist to **pick the realization** (and the IR file, and whether a Query is realized at all), not to narrow claims.

### Selection mechanism, end to end
- **Model.** A `Query` holds one `property: PropertyDecl` and one `scenario: ScenarioDecl`, and each records its `machine` (model/framework/Claims.scala:6-13, 49-56, 88-98). `QueryOn.in` lets the Scenario's machine be the Property's or a refinement of it (Claims.scala:122-126). `Realizes(machine, …)` names one machine (model/temporal/realize/Objects.scala:28-33). `DerivesFrom(base, machine)` copies the base's sections onto a different machine (Objects.scala:60-63). `CompletionExecution` … `TimeoutsExecution` are `DerivesFrom(Standalone, <subject>)` with no `changes`, so they are copies of `Standalone` (system/Realization.scala:431-436). `HeartbeatThenCompletion`, `RetryAfterHeartbeat` and `ExhaustAfterHeartbeat` are their own `Realizes(<Heartbeat subject>)`, each with its own controller, workers, evidence and observations (Realization.scala:482, 526, 580). `unmonitored` drops monitors **and the refinement**, and `rebind()` with no arguments keeps both (framework/Machine.scala:424-440, 467; lifter model/irgen/Declarations.scala:1117-1118).
- **IR** (model/ir/activity-standalone.json). A realization carries only `machine`, with no Property field (e.g. `heartbeatThenCompletion` → `heartbeatCompletion`, line ~86978). A Property carries `machine`, `name` and `holds` (`heartbeatCompletes`, line 48451). A Query carries `property {machine,name}` and `scenario {machine,name}` (`heartbeatThenCompletes`, line 56649). No IR field says "assess against these". The one claim is `query.property`, and monitors are `machine.monitors` (no activity-standalone machine declares any; `heldDispatch` in the race IR does).
- **Lowering.** `Realizable` lowers a find Query through *the one* realization with `r.machine == scenario.machine`. It refuses two (tools/umpire/lower/lower.go:156-175, also used by umpire-lint main.go:43). `check` refuses a Property of another machine or one read `through` a refinement (lower.go:855-857). The Case carries one Property: its clauses, its `DEFINITION_KIND_PROPERTY` binding, and its correlated rules (lower/internal/producer/producer.go:387-442). The manifest's expected claims are `[query.property] + expectedRun.monitors` (lower/generated.go:129-133). The check requires the assessment to conclude exactly that many claims, by id (generated.go:395-404).
- **Assessment.** `umpire-run --model` (cmd/umpire-run/run.go:171-184) and `umpire-assess --model` (cmd/umpire-assess/run.go:184-225) both call `conformance.Prepare(model, entry.Query, case, …)` (conformance/conformance.go:116). `compile` adds `bound.Property` and then each of `bound.Monitors` as claims (conformance/plan.go:134-145; doc conformance.go:29-30: "The claims are the Query's Property and the monitors of its machine"). `Bound` takes the Property from the Query and the monitors from the Scenario machine's `watched` (check/claims.go:413-441). The realization is again the unique one of the Scenario machine (plan.go:161-177). Example: `dispatchEligibility` has 3 Properties, but its Case `startDelayedCompletion` is assessed only on `completes` (model/cases/manifest.json).
- **Capability Properties** (`activitySystem.terminateSettles`, `…Deadline.*`, `…Failure.*`) are ordinary Properties of `activitySystem` with an `origin`. Each gets its own generated Query (`capabilities.bound(...)`, system/System.scala:798-811). They are assessed only as that Query's Property. `activitySystem.terminateSettles` is lowered through `Standalone`. They never enter another Case's set.
- **Definition IDs.** Property, Scenario and Query IDs are family-scoped, without the machine: `<family>.property.<name>`, `.behavior.<name>`, `.query.<name>` (tools/umpire/internal/engine/canonical.go:246-249; producer.go:430-436). Target, state, state-field, action, fact and outcome IDs embed the machine (`<family>.<kind>.<machine>.<member>`, engine/table.go:16-21, 217+). For example, the heartbeat Case binds `…target.heartbeatCompletion` and `…state.heartbeatCompletion.*`. The three `completes` Properties already share one Property ID.

### What the subject regression test guarantees
`ActivitySubjectRegression.test.scala:92-226` ("focused subjects independently own the exact complete property truth sets") pins 14 Property declarations on the 7 `rebind()` subjects (no heartbeat, reset, by-ID or TimeoutRetry subject). Each must be a distinct `PropertyDecl` object (`distinct.size == 14`) whose `decl.machine eq` its subject, with the stated `when` class, holds/holdsAcross arity, and a predicate equal to a reference predicate on every (state, binding input) of `ActivitySystem` (668304 rows). It guarantees **declaration ownership and truth tables only**. It says nothing about which Properties a Case is assessed against. Its siblings pin each subject's Query receipts (lines 240-442) and the six no-change `…Execution` realizations and the IR-file split (lines 444-480).

### Claim-container derivations (all in features/activity/standalone/system; verified: no rules, bindings, assumptions or effects added; `states` sections hold only helper values)
| Subject | Derivation | Properties | Queries | Realized by / standing |
|---|---|---|---|---|
| Completion (Completion.scala:11) | rebind() | completes | completion | CompletionExecution (= Standalone) |
| RetryFailures (RetryFailures.scala:12) | rebind() | nonRetryableFails, retryCompletes, retryExhausts | nonRetryableFailure, retry, retryExhaustionByFailures (verify) | RetryFailuresExecution (= Standalone) |
| Cancellation (Cancellation.scala:10) | rebind() | cancelRequestedWhileStarted, canceledByWorker | cancel, cancelRequest | CancellationExecution (= Standalone); both `unsupported` |
| Pausing (Pausing.scala:10) | rebind() | completes | pauseResume | PausingExecution (= Standalone) |
| DispatchEligibility (DispatchEligibility.scala:12) | rebind() | dispatchRequiresReady, scheduleToStartRequiresDispatch, completes | delayedAttemptsAreNotDispatched, scheduleToStartWaitsForDispatch (verify), startDelayedCompletion | DispatchExecution (= Standalone) |
| Timeouts (Timeouts.scala:14) | rebind() | scheduleToStartFires, startToCloseFires | scheduleToStartTimeout, startToCloseTimeout | TimeoutsExecution (= Standalone) |
| CompetingTimeouts (Timeouts.scala:46) | rebind() | scheduleToStartFires, scheduleToCloseFires | competingTimers(0..1) | none: `no-realization`, **record IR file** |
| HeartbeatCompletion (Heartbeat.scala:12) | unmonitored | heartbeatCompletes | heartbeatThenCompletes | HeartbeatThenCompletion (own) |
| HeartbeatRetry (Heartbeat.scala:29) | unmonitored | heartbeatRetryCompletes | heartbeatTimeoutRetriesThenCompletes | RetryAfterHeartbeat (own) |
| HeartbeatExhaustion (Heartbeat.scala:49) | unmonitored | heartbeatExhausts | heartbeatTimeoutExhausts | ExhaustAfterHeartbeat (own) |
| TimeoutRetry (RetryTimeouts.scala:15) | unmonitored | completesAfterTimeout, failsAfterTimeout | retryAfterTimeout, retryExhaustion | RetryAfterTimeout (own) |
| ByIDCompletion (ResponseByID.scala:14) | unmonitored | completedByID | scheduledCompletedByID | ScheduledCompletionByID (own) |
| ByIDFailure (ResponseByID.scala:26) | unmonitored | fatalFailureByID | heldFailedByID | HeldFailureByID (own) |
| ByIDCancellation (ResponseByID.scala:42) | unmonitored | activitySystem.cancelIsRequested, canceledByID | activitySystem.cancelIsRequested, heldCanceledByID | HeldCancellationByID (own) |
| ResetSettlement (Reset.scala:15) | unmonitored | resetOnFatal, resetOnExhaustion, resetOnTimeout, resetKeepsPause, completionWins, scheduleToCloseStaysTerminal, cancellationReplacesReset, pauseLeavesResetPending, resetStaysPending | resetFatality … resetRepeated (9) | none: `no-realization` (model-only by design) |
| ResetKeepingPause (Reset.scala:124) | unmonitored | resetKeptPaused | keepPausedReset | none: `no-realization` |
| DeferredReset (Reset.scala:142) | unmonitored | resetAttemptCompletes | deferredResetCompletes | ResetAfterHeartbeat (own) |

That is 17 containers. Not containers: `RecordMember` and `TrustingRecordMember` (DispatchWithTaskQueue.scala:88-90, `unmonitored` with no sections). They exist so a composition can be answered without member monitors. Every other `Derived` in model/temporal (TrustingActivityRecord, ActivityWorker, nexus ClosePolicy ×8, nexus worker, task-queue Product/System ×4) rebinds, extends, restricts or assumes. Nexus and the task queue have **no** claim containers.

### Duplicate predicates (same `when`, same body; heuristic scan of model/temporal, test files excluded)
1. `when worker.respondCompleted holds phase == completed && records(statusCompleted)`, 6×: HeartbeatCompletion.heartbeatCompletes (Heartbeat.scala:14), HeartbeatRetry.heartbeatRetryCompletes (:31), Completion.completes (Completion.scala:13), Pausing.completes (Pausing.scala:12), DispatchEligibility.completes (DispatchEligibility.scala:26), DeferredReset.resetAttemptCompletes (Reset.scala:144).
2. `when deadline.scheduleToStart holds timedOut && statusTimedOut(scheduleToStart)`, 2×: Timeouts.scheduleToStartFires (Timeouts.scala:18), CompetingTimeouts.scheduleToStartFires (:48).
3. `when deadline.scheduleToClose holds timedOut && statusTimedOut(scheduleToClose)`, 2×: CompetingTimeouts.scheduleToCloseFires (Timeouts.scala:53), ResetSettlement.scheduleToCloseStaysTerminal (Reset.scala:38).
4. `when client.requestCancel holds cancelRequested && records(statusCancelRequested)`, 2×: Cancellation.cancelRequestedWhileStarted (Cancellation.scala:12), ResetSettlement.cancellationReplacesReset (Reset.scala:42).
5. Across features, not a same-machine duplicate: ByIDCancellation `activitySystem.cancelIsRequested` (ResponseByID.scala:44) and NexusSystem `nexusSystem.cancelIsRequested` (nexus/standalone/system/System.scala:142).

### Target design: what changes
- **Kept:** each Property declared once on `ActivitySystem` (or its facet). Removing the claim containers is sound with respect to *assessment*, because no Case's assessed set grows: it stays {its Property} ∪ monitors, and ActivitySystem declares no monitors.
- **Correction 1, assessment sets:** "a Query declares its assessment set" is already true in effect: a Query has exactly one assessed Property. A new IR assessment-set field (R2) adds no capability unless a Query should be assessed against *several* Properties. Machine-wide claims already have a form: monitors, assessed on every Case of the machine. [inferred] R2 can shrink to keeping today's rule (Property + monitors) and making it explicit, or be dropped.
- **Correction 2, realization selection (the real blocker):** once the Queries move onto `ActivitySystem`, it would have 9 realizations (Standalone, HeartbeatThenCompletion, RetryAfterHeartbeat, ExhaustAfterHeartbeat, RetryAfterTimeout, ScheduledCompletionByID, HeldFailureByID, HeldCancellationByID, ResetAfterHeartbeat). `lower.Realizable` and `conformance.realizationOf` refuse more than one. The design therefore needs **a Query (or realization) to name its realization**, carried in the IR and read by lower, lint and conformance. The six no-change `…Execution` objects can simply go: their Queries lower through `Standalone`, as `activitySystem.terminate` does today.
- **Correction 3, model-only Queries:** ResetSettlement, ResetKeepingPause and CompetingTimeouts are `no-realization` *because* their machine has none. On `ActivitySystem` they would be realized through `Standalone`, and lowered or refused. A Query needs an explicit "model-only" (no realization) marker, or their standing changes.
- **Correction 4, IR-file membership:** IR roots are per-object `queries` sections (Standalone.scala `exports`). CompetingTimeouts goes to `activity-standalone-record`, not `activity-standalone`. With one `ActivitySystem.queries` section, that split needs another mechanism, e.g. a separate root object for the record's Queries.
- **Correction 5, diff scope (R5):** Property/Scenario/Query Definition IDs are family-scoped, so merging the 6 `completes` into one `completes` keeps that Property ID. Renamed ones (heartbeatCompletes, …) change. Every machine-owned ID in a moved Case changes from `<subject>` to `activitySystem` (target, states, state-fields, actions, facts, outcomes, evidence facts), and so do target and query fingerprints and the manifest key `Owner`. R5's "limited to moved Property and Query ids" understates the diff.
- **Refinement caveat** [inferred]: the 10 `unmonitored` subjects drop ActivitySystem's refinement of ActivityProduct, and the 7 `rebind()` subjects keep it. Moving the former's Queries onto ActivitySystem runs them on a refining machine. That should not change answers or totals (same table and states), but fn-162.2 must confirm it.
- **The pin:** the subject regression test (ownership plus truth tables of 14 decls) should become a test that each Property is declared once, plus a test that each Case lowers through its named realization with claims {Property} ∪ monitors.

### Needs owner decision
1. How a Query names its realization (on the Query, or the realization lists the Queries it runs), and how a Query is marked model-only. Without this, R4 cannot remove the heartbeat, TimeoutRetry, by-ID, DeferredReset, ResetSettlement, ResetKeepingPause or CompetingTimeouts subjects without changing standings. This touches fn-140's Query form.
2. Whether R2's explicit assessment set is still wanted, given that today's set is already exactly the Query's Property plus monitors. If kept: may a Query name *several* Properties, which would grow sets and change expected claims?
3. How the record IR file keeps CompetingTimeouts' Queries once there is no CompetingTimeouts object.
4. Whether R5 accepts that machine-owned Definition IDs and fingerprints change in every moved Case.
5. Names for the merged duplicates. Keeping `completes`, `scheduleToStartFires`, `scheduleToCloseFires` and `cancelRequestedWhileStarted` (or another single name each) decides which Property IDs survive unchanged.

## Nexus (Batch 1d nexus lane, 2026-10-10)

- **No migration.** Re-surveyed after fn-159: every nexus derivation changes behaviour. `HandlerWorker` restricts the shared worker, and the eight close-policy designs derived from `RejectAfterClose` rebind effects or add assumptions. None is `Derived(X.unmonitored)` or a bare `.rebind()`, so nexus has no claim container and its derivations stay (Edge Cases: behaviour-changing derivations stay). Each nexus machine has at most one realization (standalone `Standalone`, workflow `AsyncNexus`, control `ForgedControl`), so nexus Queries need no realization name to lower.
- **`cancelIsRequested` stays in both features.** `nexusSystem.cancelIsRequested` (nexus standalone, the System machine's `properties`) and `activitySystem.cancelIsRequested` share a name and a shape (a cancel request records its status fact), but they are Properties of different machines with different facts (`statusCancelRequested` of each feature's own `Fact`) and different Queries (nexus expects `inconclusive(explanationsDisagree)`, total 28). They are not a same-machine duplicate, so fn-162.3's merge leaves the nexus one alone. Their common shape is a capability-sized pattern ("a cancel request is recorded while live"); if a third entity needs it, it is a candidate for a cancellation capability, not for a shared Property here.
