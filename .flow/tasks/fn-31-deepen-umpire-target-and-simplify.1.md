---
satisfies: [R1, R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.1 Freeze Target semantics and the public/private boundary

## Description
Establish the compatibility fixtures and module boundary for R1, R5, and R6 before moving target vocabulary or callers.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Target/Language.lean`, `model/Umpire/Target/Tests/**`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/Target/**, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Inventory shared vocabulary versus target-owned composition machinery.
- Preserve `composeTarget` and its deterministic validation/canonicalization as the low-level semantic baseline rather than introducing a second checker.
- Add separate whole-value, stable-`SemanticSource` canonical-metadata, semantic-digest, and typed-error fixtures under the import-pure `Umpire.Target.Tests` boundary. Freeze the downstream stable role/action-domain tokens and persisted Query/artifact bytes in `Umpire.Examples.SwitchTests`, which may legitimately import Query, Planning, and Artifact.
- Follow the generalized authored-to-checked pattern at `model/Umpire/Target/Language.lean:8-32,369-401`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:130-200` — provider/connector vocabulary currently in Core
- `model/Umpire/Target/Language.lean:8-32` — existing generalized declaration and checked target
- `model/Umpire/Target/Language.lean:352-401` — deterministic composition and canonical projection boundary
- `model/Umpire/Target/Tests/Canonicalization.lean` — identity fixtures
- `model/Umpire/Examples/SwitchTests.lean` — downstream Query/Planning/artifact compatibility boundary

### Acceptance
- [ ] Import-pure Target fixtures independently cover checked values, typed errors, semantic identities/digests, and stable provenance-bearing canonical metadata; downstream Switch fixtures cover existing role/action-domain token strings and persisted Query/artifact bytes.
- [ ] The intended public/private ownership is explicit and import-safe.
- [ ] The existing pure checker is retained as the sole semantic implementation and focused expert seam.
- [ ] Existing comments are preserved.

## Acceptance
- [ ] R1/R5 equivalence fixtures fail on any semantic or canonical drift.
- [ ] R6 domain-purity/import checks pass; no `Umpire.Target.Tests.*` module imports Query, Planning, or Artifact.
- [ ] Focused Target and Switch tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
