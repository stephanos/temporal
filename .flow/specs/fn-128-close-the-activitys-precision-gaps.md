# Close the activity's precision gaps

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

## Requirements

- **R1 Dispatch is a field, not a phase.** `system.State` gains `dispatch: Dispatch` (`now`, `startDelay`, `backoff`), replacing the `backingOff` phase. `client.start` gains a `startDelay: Timeout` input, with a `timers.startDelay` timer that makes the attempt dispatchable. Pause and unpause keep `dispatch`. Schedule-to-start is armed only while dispatchable and waiting. The refinement map to the Product follows (scheduled while waiting, whatever `dispatch` holds).
- **R2 Rejections are rows.** `Outcome` gains `failedPrecondition` and `invalidArgument`. Every pair the server answers with one of them is a rule with that outcome, citing the server code, in both the Product and the System. The seven `silent-rejection` acceptances in `model/ir/activity-standalone*.lint.json` are removed. `closedIsRejectedUniformly` keeps NotFound for closed activities. A repeated `RequestCancel` in `cancelRequested` answers `failedPrecondition`.
- **R3 Retry policy.** A `maxAttempts` start input over a small finite domain (e.g. `one`, `two`, `unlimited`) and a `states.retriesRemaining`. A retryable failure, or a retryable timeout (start-to-close today; heartbeat with the coverage spec), retries while attempts remain and settles otherwise: Failed for exhausted failure, TimedOut for exhausted or cancellation-requested timeout. The attempt bound stays finite, derived from the largest finite policy limit. A pending pause retains Suspended and backoff on retry. The independent Model's cancellation-requested retryable-failure -> Canceled disagreement with the Go comparator's Failed requires a separate owner decision.
- **R4 Stutter facts checked.** `ActivitySystem`'s `refinement` declares `visible`, so a fact recorded on a step the Product reads as a stutter is checked. Any refinement rejection this surfaces is decided and recorded.
- **R5 Adopted from the Go model:**
  - Cancel > Reset > Pause precedence as a checked Property (e.g. `cancelIsNotUndone`; reset arrives with the coverage spec).
  - An attempt-count observation in every activity Case, not only the retry Query.
  - The Go model's "nominal vs real time window" note as `because` text on the deadline rules.
- **R6 Evidence.** The done summary maps each divergence the comparison named to the rule or row that fixes it, and lists every table, Query answer and Case that changed, and why.

## Boundaries

- This is a semantic change: tables, answers, Contracts and Cases change on purpose, and each change is listed (R6).
- Runs after fn-126 closes (its R5 freeze), using its final layout, IDs and names.
- No reset, heartbeat or respond-by-ID: those belong to "Activity coverage".
- Not the Nexus Models.

R3 scope clarification, approved 2026-10-08 under delegated recommendations: a mechanically derived timeout-retry machine and separate realization preserve the existing failure-retry evidence's all-or-none occurrence contract. Realize the requested timeout-then-completion path through the existing Testpilot ActivityAttemptWithholding opcode12, with only its typed Scala/Umpire Command/admission/lifter/lowering bridge and focused armed/bounded/owner/ordering/runtime tests. No new Testpilot runtime or opcode, generic path-evidence mechanism, Deadline capability or dependency. R6 records this additive Command arm, the extra derived machine/realization and measured artifacts; production generation/live/full gates stay at the shared batch boundary.

## Verification

R3 exhaustion witness clarification, approved 2026-10-08. `retryExhaustion` and `retryAfterTimeout` are six-action finds on the same `TimeoutRetry` derived machine. Both retry after timeout on attempt1, then fail retryably with an exhausted finite-two policy or complete on attempt2. Each Property pins its full terminal State and status fact as a conjunction. `retryExhaustionByFailures` retains the two-retryable-failure six-action path as a non-vacuous pinned verify Query, checking the exact first retry OR exact exhausted settlement with no live expectation or Case. The monitor reads both failures, while the Case predicate bridge accepts conjunctions. This preserves both proofs without extending lower/Contract/core/end-selector semantics. The old failure-retry find and its assessment remain unchanged; R6 lists all three new Queries and two new Cases.

- The model gate, `make lint-model`, the full Go suite, `make umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` and `lint-code-fast` pass.
- The live generated Cases are run once. Only the known ShutdownWorker-race INCONCLUSIVEs are allowed.
- Re-run the comparison's three divergences as Queries or fixtures: each now refuses what the server refuses.
