---
satisfies: [R1, R2, R3, R4]
---
# fn-31-deepen-umpire-target-and-simplify.2 Deepen Target checking and approachable diagnostics

## Description
Implement the cohesive checked authoring boundary and diagnostics for R1–R4 without changing target meaning.

**Size:** M
**Files:** `model/Umpire/Target.lean`, `model/Umpire/Target/**`, `model/Umpire/Core.lean`
**Touches:** [model/Umpire/Target.lean, model/Umpire/Target/**, model/Umpire/Core.lean]

### Approach
- Move target-owned checking/canonicalization behind the public facade.
- Provide one ordinary declaration/check path plus a focused expert extension seam.
- Reuse the deterministic error ordering in `model/Umpire/Target/Language.lean:284-369`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target.lean` — current facade
- `model/Umpire/Target/Language.lean:284-390` — validation/composition
- `model/Umpire/Target/Tests/Validation.lean` — negative-case pattern
- `model/Umpire/Behavior/Language.lean` — adjacent deep checked-language pattern

### Acceptance
- [ ] Ordinary callers receive a checked target or precise deterministic diagnostic.
- [ ] Provider/connector selection and all semantic choices remain explicit.
- [ ] No partial or unchecked target enters downstream APIs.

## Acceptance
- [ ] R1–R4 public contracts and negative cases are covered through the facade.
- [ ] Existing lower-level semantics remain available only through the focused extension seam.
- [ ] Focused Target tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
