---
satisfies: [R1, R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.1 Freeze Target semantics and the public/private boundary

## Description
Establish the compatibility fixtures and module boundary for R1, R5, and R6 before moving target vocabulary or callers.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Target/Language.lean`, `model/Umpire/Target/Tests/**`
**Touches:** [model/Umpire/Core.lean, model/Umpire/Target/**]

### Approach
- Inventory shared vocabulary versus target-owned composition machinery.
- Add whole-value and canonical-byte fixtures before extraction.
- Follow the authored-to-checked pattern at `model/Umpire/Target/Language.lean:8-28,354-390`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:130-200` — provider/connector vocabulary currently in Core
- `model/Umpire/Target/Language.lean:8-28` — existing declaration and checked target
- `model/Umpire/Target/Language.lean:354-390` — composition boundary
- `model/Umpire/Target/Tests/Canonicalization.lean` — identity fixtures

### Acceptance
- [ ] Compatibility fixtures cover checked values, errors, semantic identities, and canonical bytes.
- [ ] The intended public/private ownership is explicit and import-safe.
- [ ] Existing comments are preserved.

## Acceptance
- [ ] R1/R5 equivalence fixtures fail on any semantic or canonical drift.
- [ ] R6 domain-purity/import checks pass.
- [ ] Focused Target and Switch tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
