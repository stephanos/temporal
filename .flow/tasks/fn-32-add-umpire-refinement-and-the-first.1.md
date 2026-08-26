---
satisfies: [R1, R2, R6]
---
# fn-32-add-umpire-refinement-and-the-first.1 Define and check the domain-neutral Refinement language

## Description
Create the authored-to-checked Refinement facade and exhaustive domain-neutral validation fixtures for R1 and R2.

**Size:** M
**Files:** `model/Umpire/Refinement.lean`, `model/Umpire/Refinement/**`, `model/Umpire.lean`
**Touches:** [model/Umpire/Refinement.lean, model/Umpire/Refinement/**, model/Umpire.lean]

### Approach
- Mirror the checked-language lifecycle used by Target and Observation.
- Keep source/destination targets and every meaning-bearing mapping explicit.
- Canonicalize before exposing the checked value.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean:8-28,354-390` — checked target lifecycle
- `model/Umpire/Observation/Language.lean` — checked mapping/error pattern
- `model/Umpire/Observation/Tests/Compilation.lean` — exhaustive negative fixtures
- `model/Umpire/Core.lean` — semantic trace vocabulary

### Acceptance
- [ ] Complete valid declarations check deterministically.
- [ ] Stale, partial, ambiguous, wrong-kind, and obligation-broken declarations fail without a partial value.
- [ ] The public facade is Temporal-independent.

## Acceptance
- [ ] R1/R2 positive and negative matrices pass.
- [ ] Reordered equivalent declarations have identical checked identity.
- [ ] Umpire import purity is preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
