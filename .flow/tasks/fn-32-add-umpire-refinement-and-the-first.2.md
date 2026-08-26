---
satisfies: [R2, R3]
---
# fn-32-add-umpire-refinement-and-the-first.2 Apply checked refinements with complete derivations

## Description
Implement total checked trace correspondence and derivations for R2 and R3.

### Review reconciliation (normative)

`applyRefinement checked sourceSetup qualifiedTrace` must replay the exact initial state and each step through the bound source kernel before translation. It uses positional coordinates only. The failure mapping is exhaustive: stale target/digest/setup, non-authoritative source transitions, and invalid coordinates are `invalid`; absent coordinates and bound exhaustion are `unknown`; duplicate/contradictory coordinates, multiple mappings, and derivation mismatch are `conflict`; explicit omissions or out-of-partition vocabulary are `unsupported`. Every diagnostic has a canonical identity over refinement, kind/status, coordinate, related identities, bounds/counts, and omission provenance.

**Size:** M
**Files:** `model/Umpire/Refinement/**`, `model/Umpire/Refinement/Tests/**`
**Touches:** [model/Umpire/Refinement/**]

### Approach
- Preserve `initialState`, step-index, and observation-position coordinates across repeated equal values.
- Return no partial Feature trace on any non-success.
- Keep refinement outcomes separate from Observation and Property outcomes.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Qualification.lean` — qualification and no-partial-trace precedent
- `model/Umpire/Observation/Tests/Derivation.lean` — coordinate derivation pattern
- `model/Umpire/Property/Language.lean` — downstream pure trace consumer

### Acceptance
- [ ] Every destination coordinate has one complete refinement derivation.
- [ ] Every named failure kind maps to exactly one invalid/unknown/conflict/unsupported outcome and canonical diagnostic identity.
- [ ] Failed refinement cannot invoke Feature Property evaluation.
## Acceptance
- [ ] Every destination coordinate has one complete refinement derivation.
- [ ] Every named failure kind maps to exactly one invalid/unknown/conflict/unsupported outcome and canonical diagnostic identity.
- [ ] Failed refinement cannot invoke Feature Property evaluation.
### Acceptance
- [ ] R2/R3 source-kernel admission, application, bound, exhaustive status, and derivation matrices pass.
- [ ] Repeated-value coordinates remain auditable.
- [ ] No partial destination trace is observable.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
