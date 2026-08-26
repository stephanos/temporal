---
satisfies: [R1, R2, R3, R4]
---
# fn-31-deepen-umpire-target-and-simplify.2 Deepen Target checking and approachable diagnostics

## Description
### Review reconciliation (normative)

This task owns only the Target-side declaration/checker and the elaboration-only diagnostic adapter. Preserve the existing `DeclarationErrorKind` set and pin validation order: identity syntax; duplicate provider/connector identities; referenced declaration kind/existence; law witnesses and digest agreement; provider coverage/conflict; connector membership/ambiguity; kernel availability. Bounds, query-level finite completeness, and planner-kernel ordering remain in Query/Planning and are assigned to Task `.7`. Keep stable serialized `SemanticSource` separate from an authored-occurrence table captured from Lean source information. Each syntax occurrence receives a nonsemantic source-span/ordinal token and carries its declaration identity; diagnostic lookup source-sorts matching occurrences so duplicate IDs report an unambiguous original/offending pair independent of input-list order. `AuthoringDiagnostic` may expose file/line/column but occurrence data must never enter checked values, semantic digests, or artifact bytes. Add the checked kernel's explicit finite action list plus focused `actionSound`/`actionComplete` obligations here; they are semantic maintainer proofs, not runtime Target errors.

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
