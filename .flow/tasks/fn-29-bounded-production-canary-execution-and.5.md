---
satisfies: [R2, R6]
---
# fn-29-bounded-production-canary-execution-and.5 Admit canary Evidence through canonical Run Evaluation

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R2/R6 by admitting the exact canary runtime/evidence pair through the unchanged Lean semantic authority.

**Size:** M
**Files:** `model/Temporal/Tool/RunEvaluation/**`, `model/Temporal/Tool/RunEvaluationTests.lean`, `tools/umpire/runevaluation/**`, `tools/umpire/runevaluation/testdata/**`
**Touches:** [model/Temporal/Tool/RunEvaluation/**, model/Temporal/Tool/RunEvaluationTests.lean, tools/umpire/runevaluation/**, tools/umpire/runevaluation/testdata/**]

### Approach
- Extend closed runtime/evidence admission with the exact production-canary pair while preserving the private protocol, checker identity, child limits, and prior bytes.
- Reuse the remote public-source Generated View for admitted participant/history/control/cleanup facts; exclude authority, target, lease, isolation, and release fields from semantic interpretation.
- Produce the ordinary six-member v2 Run Evaluation set with byte-identical ExperimentSpec and complete configuration/run/program/mapping/query/Property/outcome bindings.
- Add paired prior-profile/canary literal fixtures plus independent mutations for missing, ambiguous, conflicting, unsupported, crossed, stale, internal-only, payload-derived, isolation-derived, and response-drift cases.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical checker and status authority
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.3.md` — shared Run Evaluation parity pattern
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.5.md` — public-remote Run Evaluation branch
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.2.md` — exact canary mapping/configuration
- `tools/umpire/regression/generated_view.go` — strict JSON/trailing-data precedent

### Key context
Canary safety may downgrade Claim Assessment but cannot rewrite Result. The checker sees only admitted execution evidence and Behavior Fingerprint bindings.

### Acceptance
- [ ] The exact compiled canary pair reaches the same checker/evaluator as prior profiles.
- [ ] Equivalent accepted observations may share outcome identity while all operational/environment identities stay distinct.
- [ ] Authority/isolation/release fields cannot supply or override semantic coordinates.
- [ ] Every insufficiency/corruption/crossing mutation yields the exact non-satisfied or fail-closed outcome and prior protocols remain unchanged.
## Acceptance
- [ ] R2/R6 canary Run Evaluation admission and independent status preservation are complete.
- [ ] Focused Lean/Go protocol, paired-profile, corruption, and race suites pass.
- [ ] Existing checker comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
