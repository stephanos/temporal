---
satisfies: [R4]
---
# fn-47-generate-umpire-semantic-outcome-and.5 Render the deterministic semantic inventory document

## Description
Aggregate, validate, and render the checked Markdown projection for R4.

**Size:** M
**Files:** `model/Temporal/Tool/SemanticInventory.lean`, `model/Temporal/Tool/SemanticInventoryMain.lean`, `model/Temporal/Tool/SemanticInventoryTests.lean`, `model/SEMANTIC_INVENTORY.md`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/SemanticInventory.lean, model/Temporal/Tool/SemanticInventoryMain.lean, model/Temporal/Tool/SemanticInventoryTests.lean, model/SEMANTIC_INVENTORY.md, model/lakefile.toml]

### Approach
- Aggregate only typed catalogs; validate all IDs, owners, ordering, families, scope/lineage, carry references, and projection sentinels before rendering.
- Render stable stage sections and Known Gap tables with source owners, no timestamps or absolute paths.
- Buffer complete Markdown and print once; errors go only to stderr with non-zero status.
- Register one non-default Lake executable and pin exact bytes plus LF in focused tests.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.toml:32-59` — support/test/tool registration.
- `model/Temporal/Tool/RunEvaluation.lean:741-762` — effect-thin semantic composition/output precedent after fn-44.
- `model/README.md:342-365` — generated-view ownership rules.
- `model/Umpire/Planning/Tests/KnownGaps.lean:33-54` — exact assertion style.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests temporal-model-semantic-inventory && mise exec -- lake exe temporal-model-semantic-inventory >/tmp/semantic-inventory.md`

## Acceptance
- [ ] The generated document contains every validated outcome family, sentinel, Known Gap source, and carry row exactly once.
- [ ] Reordered equivalent catalogs render byte-identically with stable section/table order and LF.
- [ ] Missing/duplicate/unclassified/invalid input fails before stdout; success has empty stderr.
- [ ] The checked document has no timestamp, absolute path, runtime reachability overclaim, or hand-authored semantic content.
- [ ] No additional generated file or default Lake target is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
