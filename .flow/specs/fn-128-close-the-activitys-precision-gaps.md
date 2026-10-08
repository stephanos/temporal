# Close the activity's precision gaps

> HTML render lens: `.flow/artifacts/fn-128-close-the-activitys-precision-gaps/spec.html` (local open) - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal

Make the standalone activity Model as precise as upstream's Go model (`chasm/lib/activity/model`) where both cover the same behaviour, and fix the three places where it permits what the server forbids. Source: `.plans/ACTIVITY_MODEL_COMPARISON.md` (recommendations P1-1 to P1-4, P2-8, and section 5's adoptions). Owner decision 2026-10-05.

## Why

The comparison found:

- **Unpause after a pause during backoff dispatches at once.** The server keeps the backoff (`model/temporal/features/activity/standalone/system/System.scala` resume rule vs `chasm/lib/activity/model/model.go:239-244`).
- **Schedule-to-start can fire during backoff.** The server does not arm it then (`model.go:312-322`).
- **A repeated `RequestCancel` in `cancelRequested` is accepted.** The server answers FailedPrecondition (`model.go:201-202`).
- **Rejections are silent.** The seven FailedPrecondition pairs are accepted `silent-rejection` lint findings, not rows, so a change in the server's error kind is invisible.
- **Retries are unlimited and timeouts never retry.** The Go model has `MaxAttempts`, `retriesRemaining` and retryable timeouts.
- **Stutter facts are unchecked.** `requestPause` records `statusPaused` while the Product reads it as a stutter, and no `visible` declaration is checked.

The first two have one cause: `backingOff` is a *phase*, while the Go model keeps whether a worker can get the attempt (`Dispatchability`) as a field orthogonal to the status.

## Current source context

Current source after fn-132.2: `model/temporal/features/activity/standalone/Standalone.scala` owns the form; `product/Product.scala` and `system/System.scala` each own their local `Phase`, `State` and `Fact`. Actions are `client.start` and `client.control` on the `activity` entity. All three System realizations remain in `system/Realization.scala`. The kind header `features/activity/Activity.scala` declares `Timeout`, `TimeoutType`, `AttemptResult`, the worker's `poll` and `respond`, `timers` and `deadline` (fn-132.6); the form binds the worker actions to its `activity` entity. IR stems are `activity-standalone`, `activity-standalone-record` and `activity-standalone-race`.

## Acceptance Criteria

- **R1:** Dispatch is a field, not a phase. `system.State` gains `dispatch: Dispatch` (`now`, `startDelay`, `backoff`), replacing the `backingOff` phase. `client.start` gains a `startDelay: Timeout` input, with a `timers.startDelay` timer that makes the attempt dispatchable. Pause and unpause keep `dispatch`. Schedule-to-start is armed only while dispatchable and waiting. The refinement map to the Product follows (scheduled while waiting, whatever `dispatch` holds).
- **R2:** Rejections are rows. `Outcome` gains `failedPrecondition` and `invalidArgument`. Every pair the server answers with one of them is a rule with that outcome, citing the server code, in both the Product and the System. The seven `silent-rejection` acceptances in `model/ir/activity-standalone*.lint.json` are removed. `closedIsRejectedUniformly` keeps NotFound for closed activities. A repeated `RequestCancel` in `cancelRequested` answers `failedPrecondition`.
- **R3:** Retry policy. A `maxAttempts` start input over a small finite domain (e.g. `one`, `two`, `unlimited`) and a `states.retriesRemaining`. A retryable failure, or a retryable timeout (start-to-close today; heartbeat with the coverage spec), retries while attempts remain and settles otherwise: Failed for exhausted failure, TimedOut for exhausted or cancellation-requested timeout. The attempt bound stays finite, derived from the largest finite policy limit. A pending pause retains Suspended and backoff on retry. The independent Model's cancellation-requested retryable-failure -> Canceled disagreement with the Go comparator's Failed requires a separate owner decision.
- **R4:** Stutter facts checked. `ActivitySystem`'s `refinement` declares `visible`, so a fact recorded on a step the Product reads as a stutter is checked. Any refinement rejection this surfaces is decided and recorded.
- **R5:** Adopted from the Go model.
  - Cancel > Reset > Pause precedence as a checked Property (e.g. `cancelIsNotUndone`; reset arrives with the coverage spec).
  - An attempt-count observation in every activity Case, not only the retry Query.
  - The Go model's "nominal vs real time window" note as `because` text on the deadline rules.
- **R6:** Evidence. The done summary maps each divergence the comparison named to the rule or row that fixes it, and lists every table, Query answer and Case that changed, and why. The map includes R7/R8, ordered receipts, identities and exact old/new hash inputs. No error surface beyond the required checks and the retained, explained artifact differences.
- **R7:** Accepted fatal settlement has an independently declared classification Fact alongside `statusFailed`. The existing fatal Property and its satisfied expectation stay unchanged. Only accepted fatal rows gain this Fact; states, actions, guards, outcomes, next states and old fact subsequences stay unchanged. Product refinement hides only the classification. A bounded typed FAILED Describe observation requests last failure and retains present application failure with `nonRetryable=true`; a singleton Taking gives the existing facts-only judge one decisive primary source. Errors and boundaries include retryable exhaustion, absent failure or application arm, false or absent flag, wrong operation, malformed fields and offered-fatal/server-other races. None may establish fatal settlement. Source proofs and final recording, live execution and offline replay must satisfy the unchanged fatal Property.
- **R8:** An explicit caller-owned, Case-scoped Profile authorizes the bounded pauseResume evidence using the existing Profile API. Preflight, execution and replay consume the same immutable Profile and identities. The policy proves the final Case's emitted records, byte sizes, output fanout, captured cardinalities and complete projection-plus-obligation reservation within retained hard ceilings. Cubic accepted-set rescanning, including duplicate stages, stays unchanged. Errors and boundaries include unauthorized Case or shape drift, one-below actual stage charge, excess events, duplicate stages, buffered multi-output release and arithmetic overflow. Rejection precedes external I/O where admission can decide it. Defaults, other resource caps and independent canary authorization stay unchanged.

## Shared source and closure order

The activity batch is active. Source tasks fn-128.1-5 are done. The conductor runs fn-138.1-3 next and seals its original versus adopted-source R3 comparison before coverage or fatal observation changes touch Source. That comparison retains old Query truth, limits, order, answers, ordered receipts, Case bytes, edited-span correspondence and independently recomputed hash-input ledgers. R7 creates no exception to fn-138 R3.

The conductor then runs fn-129.1-4. The two correction tasks can run in parallel against those settled sources with disjoint file ownership. Their focused source proofs precede the existing close task. Cross-spec task gates remain conductor-owned; spec-close dependencies inside this batch would deadlock its source work.

The close task owns one production regeneration and one shared full-gate, independent review, live-and-replay boundary with fn-129.5. It closes fn-128, fn-138 and fn-129 only after each spec's requirements pass completion review. Deferred artifacts and full gates do not turn an unresolved recording or replay failure into passing evidence. Every generated Run must match its authored disposition, cleanup, Contract, conformance and each Property's exact status/reason. Preserve the explicitly retained Property-only `explanationsDisagree` expectations on `retry`, `retryAfterTimeout` and `retryExhaustion`, and every other unchanged authored expectation. These are exact expected-assessment checks, not permission to accept an unexpected inconclusive. Only the previously named ShutdownWorker-race exception permits an additional inconclusive; all other mismatches remain failures. R7 still requires fatal Property satisfaction and R8 still requires bounded pauseResume execution and replay.

## Decision Context

The pre-138 source capture proves a conditional five-record, 512-byte pauseResume envelope with T=14, Q=23, R=1, F=2 and no ordinary evidence rules. Complete projection plus obligation allowance is 12,723,923 units. This is an offline upper bound, not an observed minimum or authority for post-coverage Case bytes. The caller policy must inventory the final source and regenerated Case, recompute its authorization and reject unexplained shape changes.

The accepted settlement observation supplies the distinction that `statusFailed` alone cannot carry. SDK-offered failure remains separate from server acceptance. Real conformance disagreement goes to a human under the project mandate.

Rejected global default increases, evaluator Query-ID switches and generic budget frameworks. Existing caller-owned Profile authorization is sufficient. Broad generated API drift verification and new CI coverage remain declined under the existing decision in `.flow/memory/declined/generated-api-drift-verification.md`.

## Early proof point

The fatal correction's focused recording and replay prove that accepted server classification satisfies the unchanged fatal Property. If it fails, revisit the typed public observation before the shared close. The budget correction independently proves admission and actual charged-stage bounds for the final supported Case shape.

## Quick commands

```bash
python3 /home/agent/.codex/scripts/flowctl.py validate --spec fn-128 --coverage --json
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/conformance -run Activity
```

## Boundaries

- This is a semantic change: tables, answers, Contracts and Cases change on purpose, and each change is listed (R6).
- Runs after fn-126 closes (its R5 freeze), using its final layout, IDs and names.
- No reset, heartbeat or respond-by-ID: those belong to "Activity coverage".
- Not the Nexus Models.

R3 scope clarification, approved 2026-10-08 under delegated recommendations: a mechanically derived timeout-retry machine and separate realization preserve the existing failure-retry evidence's all-or-none occurrence contract. Realize the requested timeout-then-completion path through the existing Testpilot ActivityAttemptWithholding opcode12, with only its typed Scala/Umpire Command/admission/lifter/lowering bridge and focused armed/bounded/owner/ordering/runtime tests. No new Testpilot runtime or opcode, generic path-evidence mechanism, Deadline capability or dependency. R6 records this additive Command arm, the extra derived machine/realization and measured artifacts; production generation/live/full gates stay at the shared batch boundary.

## Verification

R3 exhaustion witness clarification, approved 2026-10-08. `retryExhaustion` and `retryAfterTimeout` are six-action finds on the same `TimeoutRetry` derived machine. Both retry after timeout on attempt1, then fail retryably with an exhausted finite-two policy or complete on attempt2. Each Property pins its full terminal State and status fact as a conjunction. `retryExhaustionByFailures` retains the two-retryable-failure six-action path as a non-vacuous pinned verify Query, checking the exact first retry OR exact exhausted settlement with no live expectation or Case. The monitor reads both failures, while the Case predicate bridge accepts conjunctions. This preserves both proofs without extending lower/Contract/core/end-selector semantics. The old failure-retry find and its assessment remain unchanged; R6 lists all three new Queries and two new Cases.

- The model gate, `make lint-model`, the full Go suite, `make umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` and `lint-code-fast` pass.
- The live generated Cases are run once. Each Run matches every authored expected assessment and reason exactly, including the retained retry Property-only `explanationsDisagree` expectations. Only the named ShutdownWorker-race exception permits an additional inconclusive; resource-limit, neverEvaluated and any other unexpected status/reason fail.
- Re-run the comparison's three divergences as Queries or fixtures: each now refuses what the server refuses.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Dispatch is a field, not a phase. `system.State` gains `dispatch: Dispatch` (`now`, `startDelay`, `backoff`), replacing the `backingOff` phase. `client.start` gains a `startDelay: Timeout` input, with a `timers.startDelay` timer that makes the attempt dispatchable. Pause and unpause keep `dispatch`. Schedule-to-start is armed only while dispatchable and waiting. The refinement map to the Product follows (scheduled while waiting, whatever `dispatch` holds). | fn-128-close-the-activitys-precision-gaps.1 | — |
| R2 | Rejections are rows. `Outcome` gains `failedPrecondition` and `invalidArgument`. Every pair the server answers with one of them is a rule with that outcome, citing the server code, in both the Product and the System. The seven `silent-rejection` acceptances in `model/ir/activity-standalone*.lint.json` are removed. `closedIsRejectedUniformly` keeps NotFound for closed activities. A repeated `RequestCancel` in `cancelRequested` answers `failedPrecondition`. | fn-128-close-the-activitys-precision-gaps.2 | — |
| R3 | Retry policy. A `maxAttempts` start input over a small finite domain (e.g. `one`, `two`, `unlimited`) and a `states.retriesRemaining`. A retryable failure, or a retryable timeout (start-to-close today; heartbeat with the coverage spec), retries while attempts remain and settles otherwise: Failed for exhausted failure, TimedOut for exhausted or cancellation-requested timeout. The attempt bound stays finite, derived from the largest finite policy limit. A pending pause retains Suspended and backoff on retry. The independent Model's cancellation-requested retryable-failure -> Canceled disagreement with the Go comparator's Failed requires a separate owner decision. | fn-128-close-the-activitys-precision-gaps.3 | — |
| R4 | Stutter facts checked. `ActivitySystem`'s `refinement` declares `visible`, so a fact recorded on a step the Product reads as a stutter is checked. Any refinement rejection this surfaces is decided and recorded. | fn-128-close-the-activitys-precision-gaps.4 | — |
| R5 | Adopted from the Go model. | fn-128-close-the-activitys-precision-gaps.5 | — |
| R6 | Evidence. The done summary maps each divergence the comparison named to the rule or row that fixes it, and lists every table, Query answer and Case that changed, and why. The map includes R7/R8, ordered receipts, identities and exact old/new hash inputs. No error surface beyond the required checks and the retained, explained artifact differences. | fn-128-close-the-activitys-precision-gaps.6 | — |
| R7 | Accepted fatal settlement has an independently declared classification Fact alongside `statusFailed`. The existing fatal Property and its satisfied expectation stay unchanged. Only accepted fatal rows gain this Fact; states, actions, guards, outcomes, next states and old fact subsequences stay unchanged. Product refinement hides only the classification. A bounded typed FAILED Describe observation requests last failure and retains present application failure with `nonRetryable=true`; a singleton Taking gives the existing facts-only judge one decisive primary source. Errors and boundaries include retryable exhaustion, absent failure or application arm, false or absent flag, wrong operation, malformed fields and offered-fatal/server-other races. None may establish fatal settlement. Source proofs and final recording, live execution and offline replay must satisfy the unchanged fatal Property. | fn-128-close-the-activitys-precision-gaps.6, fn-128-close-the-activitys-precision-gaps.7 | — |
| R8 | An explicit caller-owned, Case-scoped Profile authorizes the bounded pauseResume evidence using the existing Profile API. Preflight, execution and replay consume the same immutable Profile and identities. The policy proves the final Case's emitted records, byte sizes, output fanout, captured cardinalities and complete projection-plus-obligation reservation within retained hard ceilings. Cubic accepted-set rescanning, including duplicate stages, stays unchanged. Errors and boundaries include unauthorized Case or shape drift, one-below actual stage charge, excess events, duplicate stages, buffered multi-output release and arithmetic overflow. Rejection precedes external I/O where admission can decide it. Defaults, other resource caps and independent canary authorization stay unchanged. | fn-128-close-the-activitys-precision-gaps.6, fn-128-close-the-activitys-precision-gaps.8 | — |
