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
- Aggregate only typed catalogs; validate all IDs, owners, owner-local ordering, families, scope/lineage, carry references, projection mappings, and sentinels before rendering. Sort the otherwise unordered assembled family/source lists by validated canonical keys.
- Render exactly: title and generated-warning preamble; ordered Outcome families; Projection sentinels; and one Known Gap flows table with Catalog ID, Owner, Lineage, Scope, Shape, Source/reference, Field mapping, and Description.
- Buffer complete Markdown before one stdout write. Validation/render errors write only stderr and return non-zero; a final OS write failure may leave a stream prefix but must return non-zero.
- Register the non-default inventory executable and a non-default process-test executable, use quiet Lake invocation for byte-clean stdout, and pin exact bytes, terminal LF, stderr, and exit status in focused and process-level tests. Neither executable becomes a default target.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.toml:32-59` — support/test/tool registration.
- `model/Temporal/Tool/RunEvaluation.lean:741-762` — effect-thin semantic composition/output precedent after fn-44.
- `model/README.md:342-365` — generated-view ownership rules.
- `model/Umpire/Planning/Tests/KnownGaps.lean:33-54` — exact assertion style.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests temporal-model-semantic-inventory temporal-model-semantic-inventory-tests` then `cd model && mise exec -- lake exe temporal-model-semantic-inventory-tests` and `cd model && mise exec -- lake -q exe temporal-model-semantic-inventory >/tmp/semantic-inventory.md`
## Acceptance
- [ ] The generated document contains every validated outcome constructor, sentinel, Known Gap source/projection/carry row exactly once in the specified headings, columns, and canonical order.
- [ ] Reordered equivalent assembled family/source inputs render byte-identically with terminal LF, while noncanonical owner-local catalog order is rejected.
- [ ] Process tests cover warm and stale builds: success stdout is only document bytes and stderr is empty; validation/render failure has empty stdout, diagnostics on stderr, and non-zero status.
- [ ] An injected final-writer failure returns non-zero; tests do not claim the OS can retract an already-written prefix.
- [ ] The checked document has no timestamp, absolute path, runtime reachability overclaim, or hand-authored semantic content.
- [ ] No additional generated file or default Lake target is added; the process-test executable is non-default.
## Done summary
Aggregated the typed outcome, projection-sentinel, and 24-row Known Gap catalogs behind complete validation and deterministic canonical Markdown rendering. Added non-default renderer/process-test executables, stale/warm and stream-failure regressions, and the byte-exact checked document.

Baseline: red before edits only because `temporal-model-semantic-inventory` and `temporal-model-semantic-inventory-tests` did not yet exist; baseline `make lint-model` was green (202/202).

stage: impl-review - ran [2026-09-01T14:31:02Z..2026-09-01T14:35:01Z] (Codex SHIP; session 01a05d61-73be-7812-8425-978ae178c14e; 0 open findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ce72755d96339c271184994b77aaaacf486159e9
- Tests: baseline: red (task Quick commands failed pre-edit only because temporal-model-semantic-inventory and temporal-model-semantic-inventory-tests did not exist), baseline: make lint-model (green, 202/202), cd model && mise exec -- lake env lean Temporal/Tool/SemanticInventoryTests.lean (strict RED: missing Temporal.Tool.SemanticInventory), cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests temporal-model-semantic-inventory temporal-model-semantic-inventory-tests, cd model && mise exec -- lake exe temporal-model-semantic-inventory-tests, cd model && mise exec -- lake -q exe temporal-model-semantic-inventory >/tmp/semantic-inventory.md, cmp -s /tmp/fn47-task5-final-render.md model/SEMANTIC_INVENTORY.md, make lint-model, impl-review Codex SHIP session 01a05d61-73be-7812-8425-978ae178c14e receipt /tmp/impl-review-receipt-fn-47-generate-umpire-semantic-outcome-and.5.json
- PRs: