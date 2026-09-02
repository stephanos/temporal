---
satisfies: [R2, R6]
---
# fn-29-bounded-production-canary-execution-and.5 Admit canary Evidence through canonical Run Evaluation

## Description
Admit R2/R6's exact model-scoped UmpireExecutor result and public Evidence closure into the canary artifact/Claim Assessment pipeline. Reuse fn-52's portable evaluator result; do not run a second semantic evaluator in the canary.

**Size:** M
**Files:** `tools/canary/evaluation/**`, `tools/canary/artifact/**`, `tools/canary/testdata/**`, focused canary evaluation tests
**Touches:** [tools/canary/evaluation/**, tools/canary/artifact/**, tools/canary/testdata/**]

### Approach
- Require the exact pinned plan checksum, run identity, validated model scope, ExperimentSpec/model bindings, source closures, Evidence Links, stage statuses, Known Gaps, and external-obligation set from the fn-52 ExecutionResult.
- Preserve the existing ordinary Run/RawEvidence/Result artifact meanings and exact evaluation outcome rather than reconstructing or reinterpreting semantic facts.
- Exclude authority, target, lease, isolation, and release fields from the portable semantic result; admit them only into later canary Claim Assessment provenance.
- Add paired prior-profile/canary fixtures and mutations for plan-local scope, missing/ambiguous/conflicting/unsupported/crossed/stale Evidence, unresolved obligations, and result drift.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-52-caller-neutral-grpc-portable-test-plans.md` — ExecutionResult and claim-scope contract
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical semantic status meanings
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.2.md` — exact canary plan/profile
- `tools/umpire/portableevaluation/evaluator.go` — evaluator whose result is consumed unchanged
- `tools/umpire/artifact/result.go` — ordinary Result admission pattern

### Key context
The canary validates provenance and operational policy around the portable result. It cannot upgrade plan-local output, satisfy an unresolved external obligation, or rewrite any semantic status.
## Acceptance
- [ ] Only the exact model-scoped ExecutionResult for the pinned canary plan enters Claim Assessment.
- [ ] Equivalent accepted observations retain canonical semantic outcome identity while operational/environment identities remain distinct.
- [ ] Plan-local, crossed, stale, incomplete, unresolved-obligation, authority-derived, isolation-derived, internal-only, and payload-derived mutations cannot qualify.
- [ ] No second evaluator or semantic reconstruction is introduced.
- [ ] R2/R6 focused protocol, artifact, corruption, and race suites pass.
- [ ] Existing checker comments are preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
