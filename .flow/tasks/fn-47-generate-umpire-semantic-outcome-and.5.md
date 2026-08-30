---
satisfies: [R4]
---
# fn-47-generate-umpire-semantic-outcome-and.5 Render the deterministic semantic inventory document

## Description
Aggregate, validate, and render the checked Markdown projection for R4.

**Size:** M
**Files:** `model/Temporal/Tool/SemanticInventory.lean`, `model/Temporal/Tool/SemanticInventoryMain.lean`, `model/Temporal/Tool/SemanticInventoryTests.lean`, `model/Temporal/Tool/SemanticInventoryMainTests.lean`, `model/SEMANTIC_INVENTORY.md`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/SemanticInventory.lean, model/Temporal/Tool/SemanticInventoryMain.lean, model/Temporal/Tool/SemanticInventoryTests.lean, model/Temporal/Tool/SemanticInventoryMainTests.lean, model/SEMANTIC_INVENTORY.md, model/lakefile.toml]

### Approach
- Aggregate only typed catalogs; validate all IDs, owners, ordering, families, scope/lineage, carry references, projection mappings, and sentinels before rendering.
- Render exactly: title and generated-warning preamble; ordered Outcome families; Projection sentinels; and one Known Gap flows table with Catalog ID, Owner, Lineage, Scope, Shape, Source/reference, Field mapping, and Description.
- Buffer complete Markdown before one stdout write. Validation/render errors write only stderr and return non-zero; a final OS write failure may leave a stream prefix but must return non-zero.
- Register one non-default Lake executable, use quiet Lake invocation for byte-clean stdout, and pin exact bytes, terminal LF, stderr, and exit status in focused and process-level tests.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.toml:32-59` — support/test/tool registration.
- `model/Temporal/Tool/RunEvaluation.lean:741-762` — effect-thin semantic composition/output precedent after fn-44.
- `model/README.md:342-365` — generated-view ownership rules.
- `model/Umpire/Planning/Tests/KnownGaps.lean:33-54` — exact assertion style.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests temporal-model-semantic-inventory` then `cd model && mise exec -- lake -q exe temporal-model-semantic-inventory >/tmp/semantic-inventory.md`
## Acceptance
- [ ] The generated document contains every validated outcome constructor, sentinel, Known Gap source/projection/carry row exactly once in the specified headings, columns, and canonical order.
- [ ] Reordered equivalent inputs render byte-identically with terminal LF.
- [ ] Process tests cover warm and stale builds: success stdout is only document bytes and stderr is empty; validation/render failure has empty stdout, diagnostics on stderr, and non-zero status.
- [ ] An injected final-writer failure returns non-zero; tests do not claim the OS can retract an already-written prefix.
- [ ] The checked document has no timestamp, absolute path, runtime reachability overclaim, or hand-authored semantic content.
- [ ] No additional generated file or default Lake target is added.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
