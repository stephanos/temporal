---
satisfies: [R2, R3]
---
# fn-32-add-umpire-refinement-and-the-first.2 Apply checked refinements with complete derivations

## Description
Implement total checked trace correspondence and derivations for R2 and R3.

**Size:** M
**Files:** `model/Umpire/Refinement/**`, `model/Umpire/Refinement/Tests/**`
**Touches:** [model/Umpire/Refinement/**]

### Approach
- Preserve stable semantic coordinates across repeated equal values.
- Return no partial Feature trace on any non-success.
- Keep refinement outcomes separate from Observation and Property outcomes.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Qualification.lean` — qualification and no-partial-trace precedent
- `model/Umpire/Observation/Tests/Derivation.lean` — coordinate derivation pattern
- `model/Umpire/Property/Language.lean` — downstream pure trace consumer

### Acceptance
- [ ] Every destination coordinate has one complete refinement derivation.
- [ ] Invalid/unknown/conflict/unsupported outcomes are deterministic and distinct.
- [ ] Failed refinement cannot invoke Feature Property evaluation.

## Acceptance
- [ ] R2/R3 application, bound, and derivation matrices pass.
- [ ] Repeated-value coordinates remain auditable.
- [ ] No partial destination trace is observable.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
