---
satisfies: [R2, R4]
---
# fn-221-gomad-functional-suite-triage-sweep-on.2 Run the full sweep and retain the support matrix

## Description
Run the harness over every package under tests on a quiet host with recorded host identity, fixed sorted package order, and the recorded budgets. Write .flow/artifacts/fn221-support-matrix-<date>.md plus a JSON sidecar (shape decided and recorded here) with one row per test function: package, name, class, reason, seed, budget, candidate identity (helper digest, compiler, module identity). Tests cut off by the overall budget are recorded as not run. The first paragraph states this is an observation on one candidate, not a support claim.

**Size:** L
**Files:** .flow/artifacts/fn221-*.md, .flow/artifacts/fn221-*.json

## Acceptance
- [ ] Every enumerated test function has exactly one class and, for non-pass classes, a reason string
- [ ] Candidate and host identity recorded
- [ ] Not-run tests distinguished from failures
- [ ] Observation-only statement present; QUALIFICATION.md untouched

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
