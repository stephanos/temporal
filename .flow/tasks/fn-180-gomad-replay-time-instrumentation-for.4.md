---
satisfies: [R5]
---
# fn-180-gomad-replay-time-instrumentation-for.4 Invariant bisect over exact replays

## Description
Implement R5 over the replay request and stop resolver from tasks .1–.3. Register invariant checks with an optional monotonic-failure declaration. Default to scanning the inclusive local-choice range in ascending order; permit explicit bisect only with that declaration. Evaluate each candidate in a fresh exact replay of the same artifact and selected incarnation, including the cluster prefix. Bound attempts and elapsed wall time, including each evaluation, and report the evaluated range/remaining interval and the reason for an incomplete result. Reject observed monotonicity violations without returning a bisect answer. Keep predicate failures distinct from evaluation failures and replay divergence; label bisect conclusions as conditional rather than claiming samples proved monotonicity.
## Acceptance
- [ ] Scan finds the earliest failure in range for monotonic and fail/recover/fail fixtures, including passing final endpoint and single-choice ranges
- [ ] Explicit bisect requires a monotonic registration, finds the known monotonic transition, and reports the assumption
- [ ] Observed false-before-true samples invalidate bisect; tests do not claim unsampled reversals can be detected
- [ ] Already-failing and no-failure results are scoped to requested range; bisect no-failure result remains conditional
- [ ] Attempt/time exhaustion yields incomplete; invariant panic/timeout and replay divergence never become predicate results
- [ ] Each evaluation is a fresh replay; a cluster fixture targets the correct restarted incarnation and verifies its preceding shared execution
- [ ] Invalid check, target, range and nonpositive bounds refused before search
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
