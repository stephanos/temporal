---
satisfies: [R1, R6]
---
# fn-49-centralize-observation-field-and.1 Introduce Observation field specifications

## Description
Add the inert field handle and direct projection/negative contract tests (R1).

**Size:** M
**Files:** `model/Umpire/Observation/Language.lean`, `model/Umpire/Observation/Tests/Compilation.lean`, `model/Umpire/Observation/Tests/Check.lean`, `model/Umpire/Observation/ImportTests.lean`
**Touches:** [model/Umpire/Observation/Language.lean, model/Umpire/Observation/Tests/Compilation.lean, model/Umpire/Observation/Tests/Check.lean, model/Umpire/Observation/ImportTests.lean]

### Approach
- Place the field specification beside existing field declaration/reference/disposition types.
- Project only existing inert values; require disposition as an explicit argument and keep checker validation authoritative.
- Add facade/import checks and prove each projection is equal to the prior record literal shape.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Language.lean:20-62,95-110` — current field and disposition vocabulary
- `model/Umpire/Observation/Language.lean:688-719` — existing field resolution and typed checking
- `model/Umpire/Observation/Tests/Compilation.lean` — public authoring examples
- `model/Umpire/Observation/Tests/Check.lean` — invalid field/type/disposition coverage

**Optional** (reference as needed):
- `model/Umpire/Observation/ImportTests.lean` — facade visibility contract

## Acceptance
- [ ] A field spec projects exact existing declaration, reference, expression, and chosen disposition records.
- [ ] No default disposition, automatic registration, macro, or coercion changes checker meaning.
- [ ] Invalid identity/type/disposition/digest cases retain existing typed errors.
- [ ] Observation compilation, check, and import tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
