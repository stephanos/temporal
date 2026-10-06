---
satisfies: [R3]
---
# fn-128-close-the-activitys-precision-gaps.3 Retry policy: maxAttempts and retryable timeouts

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R3 (comparison P1-2). `client.start` gains a `maxAttempts` input over a small finite domain (e.g. `one`, `two`, `unlimited`), kept in `system.State`, and `states.retriesRemaining(s)` reads it with the attempt count. A retryable failure, or a start-to-close timeout, retries (dispatch `backoff`, attempts saturating) while attempts remain and fails otherwise, as the Go model's `retriesRemaining` and `NonRetryableTimeouts` say; each split is two guarded rules. The attempt bound stays finite and is derived from the domain (comparison section 5 item 4). Heartbeat timeouts join with fn-129.

The realization sets the start request's retry policy from the input.
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
