---
satisfies: [R1, R2, R6]
---

# fn-32-add-umpire-refinement-and-the-first.1 Define and check the domain-neutral Implementation Link language

## Description
Create the authored-to-checked Implementation Link facade and exhaustive domain-neutral validation fixtures for R1 and R2.

### Review reconciliation (normative)

The prototype is an exact bounded forward simulation. The inert declaration contains finite setup/state/action/outcome/observation/relation/capability tables, a complete support/Known Gap partition, and a positive semantic-transition Limit. A separate proof witness indexed by the exact declaration and checked targets supplies `initialForward`, `stepForward`, and `requiredCoverage`; the trace theorem is derived. There is no reverse/bisimulation obligation, named Behavior occurrence mapping, or serialized proof term.

**Size:** M
**Files:** `model/Umpire/ImplementationLink.lean`, `model/Umpire/ImplementationLink/**`, `model/Umpire.lean`
**Touches:** [model/Umpire/ImplementationLink.lean, model/Umpire/ImplementationLink/**, model/Umpire.lean]

### Approach
- Mirror the checked-language lifecycle used by Target and Observation.
- Keep source/destination targets, mapping tables, support/Known Gap partition, application Limit, and every forward proof obligation explicit.
- Canonicalize before exposing the checked value.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean:8-28,354-390` — checked target lifecycle
- `model/Umpire/Observation/Language.lean` — checked mapping/error pattern
- `model/Umpire/Observation/Tests/Compilation.lean` — exhaustive negative fixtures
- `model/Umpire/Core.lean` — semantic trace vocabulary

### Acceptance
- [ ] Complete valid declarations plus exact indexed witnesses check deterministically, with proof terms excluded from identity bytes.
- [ ] Stale, partial, ambiguous, wrong-kind, incomplete support/Known Gap, invalid-limit, and witness-index mismatches fail without a partial value.
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
