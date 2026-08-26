---
satisfies: [R2, R6]
---
# fn-29-bounded-production-canary-execution-and.5 Admit canary evidence through canonical semantic conformance

## Description
Implement R2/R6 by admitting the exact canary runtime/evidence pair through the unchanged Lean semantic authority.

**Size:** M
**Files:** `model/Temporal/Tool/Conformance/**`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/conformance/**`, `tools/umpire/conformance/testdata/**`
**Touches:** [model/Temporal/Tool/Conformance/**, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/conformance/**, tools/umpire/conformance/testdata/**]

### Approach
- Extend closed runtime/evidence admission with the exact production-canary pair while preserving the private protocol, checker identity, child limits, and prior bytes.
- Reuse the remote public-source projection for admitted participant/history/control/cleanup facts; exclude authority, target, lease, isolation, and release fields from semantic interpretation.
- Produce the ordinary six-member v1 conformance set with byte-identical ExperimentSpec and complete configuration/run/program/mapping/query/Property/outcome bindings.
- Add paired prior-profile/canary literal fixtures plus independent mutations for missing, ambiguous, conflicting, unsupported, crossed, stale, internal-only, payload-derived, isolation-derived, and response-drift cases.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical checker and status authority
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.3.md` — multi-profile admission pattern
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.5.md` — public-remote conformance branch
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.2.md` — exact canary mapping/configuration
- `tools/umpire/regression/projection.go` — strict JSON/trailing-data precedent

### Key context
Canary safety may downgrade qualification but cannot rewrite Result. The checker sees only admitted execution evidence and semantic identity bindings.

### Acceptance
- [ ] The exact compiled canary pair reaches the same checker/evaluator as prior profiles.
- [ ] Equivalent qualified observations may share outcome identity while all operational/environment identities stay distinct.
- [ ] Authority/isolation/release fields cannot supply or override semantic coordinates.
- [ ] Every insufficiency/corruption/crossing mutation yields the exact non-satisfied or fail-closed outcome and prior protocols remain unchanged.

## Acceptance
- [ ] R2/R6 canary conformance admission and independent status preservation are complete.
- [ ] Focused Lean/Go protocol, paired-profile, corruption, and race suites pass.
- [ ] Existing checker comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
