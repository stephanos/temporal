---
satisfies: [R4, R5]
---
# fn-47-generate-umpire-semantic-outcome-and.6 Wire the narrow semantic inventory drift gate

## Description
Publish the facade, atomic generation/check commands, and concise documentation for R4/R5.

**Size:** M
**Files:** `model/Umpire/SemanticInventory.lean`, `model/Umpire.lean`, `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Umpire/SemanticInventory.lean, model/Umpire.lean, Makefile, model/README.md, model/ARCHITECTURE.md]

### Approach
- Expose a narrow inventory facade without flattening stage implementation modules.
- Add atomic sibling-temp generation and read-only temp-render/diff targets; include only the check in `lint-model`.
- Document source ownership, regeneration/check commands, dependency baseline, and non-semantic/non-schema boundaries.
- Do not edit GitHub workflows or generalize this into generated API drift infrastructure.

### Investigation targets
**Required** (read before coding):
- `Makefile:1049-1068` — temp render/diff check pattern.
- `Makefile:1304-1317` — model lint gate.
- `model/README.md:232-306,342-365` — stage and generated-view documentation.
- `model/ARCHITECTURE.md:148-160` — tooling/import-policy placement.
- `.flow/memory/declined/generated-api-drift-verification.md` — narrow reopening boundary.

### Quick commands
`make umpire-check-semantic-inventory && make lint-model`

## Acceptance
- [ ] Generation replaces the checked document only after successful complete render and preserves the old file on failure/interruption.
- [ ] Check mode never writes and reports deterministic diff for missing/stale/extra content.
- [ ] `lint-model` includes only this narrow drift check and passes with focused/aggregate Lean tests.
- [ ] Docs link the inventory and state that stage types, Result schema, artifact bytes, and runtime behavior remain authoritative/unchanged.
- [ ] No GitHub Actions, broad API drift gate, public docs, or deferred spec behavior is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
