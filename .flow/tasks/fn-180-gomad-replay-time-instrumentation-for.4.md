---
satisfies: [R5]
---
# fn-180-gomad-replay-time-instrumentation-for.4 Invariant bisect over exact replays

## Description
Implement R5 over the exact tape-coordinate stop resolver, authenticated prefix verifier, closed invariant registry and dedicated inspected-prefix terminal from the .1–.3 subsystem. Default to scanning the inclusive local-choice range in ascending order; permit explicit bisect only with a retained monotonic-failure declaration. Evaluate each candidate in a fresh helper/cluster/store exact replay of the same artifact and selected incarnation, including every executed incarnation and shared World prefix. Bound attempts and elapsed wall time through startup, replay, predicate execution and cleanup; destroy timed-out stores rather than reuse them. Report evaluated positions, remaining interval and the reason for incompleteness. Reject observed monotonicity violations without returning a bisect answer. Keep predicate truth distinct from panic, timeout, instrumentation/replay divergence and fuel/capacity failures; label all bisect conclusions conditional rather than claiming samples proved monotonicity.
## Acceptance
- [ ] Scan finds the earliest failure in range for monotonic and fail/recover/fail fixtures, including passing final endpoint and single-choice ranges
- [ ] Explicit bisect requires a monotonic registration, finds the known monotonic transition, and reports the assumption
- [ ] Observed false-before-true samples invalidate bisect; tests do not claim unsampled reversals can be detected
- [ ] Already-failing and no-failure results are scoped to requested range; bisect no-failure result remains conditional
- [ ] Attempt/time exhaustion yields incomplete; invariant panic/timeout and replay divergence never become predicate results
- [ ] Each evaluation is a fresh replay; a cluster fixture targets the correct restarted incarnation and verifies its preceding shared execution
- [ ] Invalid check, target, range and nonpositive bounds refused before search
- [ ] Tests prove fresh helper/store/registry state for each attempt and apply the wall bound to startup/replay/nonreturning predicates/cleanup
- [ ] Evaluations require verified inspected-prefix terminals and compulsory witnesses; no normal full-replay success or seeded-prefix continuation substitutes
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
