---
satisfies: [R3]
---
# fn-128-close-the-activitys-precision-gaps.3 Retry policy: maxAttempts and retryable timeouts

## Description
**Batch:** active Activity Model batch (MILESTONES.md, fn-128). Do not run `make umpire-gen-model`, regenerate production fixtures or Cases, or run the full gates in this task; production IR/answer/Case comparison is checked at the shared batch's single regeneration against ee32b5fa6023c4ab8dfcc928f07d79afe5587186. Focused framework/lifter/munit and bridge/runtime regression tests run here. Commit this task separately.

R3 (comparison P1-2). `client.start` gains a `maxAttempts` input (`unlimited`, `one`, `two`), retained in `system.State`; `states.retriesRemaining(before)` reads it with the saturating attempt count bounded by the largest finite limit. Retryable failure and start-to-close timeout each have guarded retry and terminal settlement. Exhausted failure is Failed; exhausted or cancellation-requested timeout is TimedOut, matching the comparator's typed `attemptTimedOut`. A retry under a pending pause is Suspended with dispatch backoff retained. Preserve the independent Model's cancellation-requested retryable-failure -> Canceled rule; the Go comparator's Failed result is a separate owner-required disagreement, not authority to silently change that rule. Heartbeat/reset/by-ID remain fn-129.

The realization sets each start request's retry policy. The fifth input needs minimal typed Action/Rules arity support, not a generic capability or schema revival. The deadlines helper accepts an explicit Class preset for non-Timeout inputs and accumulates its catalogs, preserving every old Timeout combination and refusing ambiguous/defaulted non-Timeout or duplicate presets.

**Scope clarification approved by conductor 2026-10-08 under delegated recommendations:** timeout-retry evidence uses a mechanically Derived machine over the unchanged ActivitySystem state/rules and a separate Realizes declaration. The old second-delivery kind confirms failure1 and poll2 all-or-none, so it cannot evidence a timeout path with no failure1. Preserve the old retry Query's full-state equality and assessment. To execute the requested timeout-then-completion Case, expose the existing Testpilot `ActivityAttemptWithholding` opcode12 through the smallest typed Scala leaf and Umpire Command/admission/lifting/lowering bridge; the dispatch's earlier no-schema/lower-extension constraint is corrected only for this bridge. Withholding is an activity-script onPath item for exactly one armed positively bounded server timer, never unconditional or a performance, with focused refusals and supported ordering/runtime proof. No new Testpilot opcode/runtime, path-evidence mechanism, Deadline capability or new dependency. Full generated/live proof remains batch-deferred. R6 lists the extra derived machine/realization, additive Command arm and all expected artifact deltas.
## Acceptance
- [ ] `maxAttempts` is a start input and realized; retries stop at it; a start-to-close timeout retries while attempts remain.
- [ ] A Query shows a retryable failure that fails once attempts run out, and one that completes after a retried timeout.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
