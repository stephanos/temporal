---
satisfies: [R4, R5]
---
# fn-47-generate-umpire-semantic-outcome-and.6 Wire the narrow semantic inventory drift gate

## Description
Publish the facade, atomic generation/check commands, and concise documentation for R4/R5.

**Size:** M
**Files:** `model/Umpire/SemanticInventory.lean`, `model/Umpire.lean`, `model/Temporal/Tool/SemanticInventoryMakeTests.lean`, `model/Temporal/Tool/SemanticInventoryMakeTestsMain.lean`, `model/lakefile.toml`, `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Umpire/SemanticInventory.lean, model/Umpire.lean, model/Temporal/Tool/SemanticInventoryMakeTests.lean, model/Temporal/Tool/SemanticInventoryMakeTestsMain.lean, model/lakefile.toml, Makefile, model/README.md, model/ARCHITECTURE.md]

### Approach
- Expose a narrow inventory facade without flattening stage implementation modules.
- Add atomic sibling-temp generation and read-only temp-render/diff targets; include only the check in `lint-model`.
- Keep the document path and renderer command overridable by private Make variables so one non-default process-test executable can exercise the real recipes in an isolated temporary tree. Cover renderer failure after partial temp output, termination before replacement, sibling-temp cleanup, missing/stale/extra deterministic diffs, check-mode immutability, and stable readable installed permissions.
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
`cd model && mise exec -- lake build temporal-model-semantic-inventory-make-tests && mise exec -- lake exe temporal-model-semantic-inventory-make-tests` then `make umpire-check-semantic-inventory && make lint-model`
## Acceptance
- [ ] Generation replaces the checked document only after successful complete render and preserves the old file on failure/interruption.
- [ ] Check mode never writes and reports deterministic diff for missing/stale/extra content.
- [ ] Isolated process tests run the actual Make recipes with injected renderer failure and termination, assert the prior document is byte-identical, verify sibling-temp cleanup, pin stable readable permissions, and prove check mode is immutable.
- [ ] `lint-model` includes only this narrow drift check and passes with focused/aggregate Lean tests.
- [ ] Docs link the inventory and state that stage types, Result schema, artifact bytes, and runtime behavior remain authoritative/unchanged.
- [ ] No GitHub Actions, broad API drift gate, public docs, or deferred spec behavior is added.
- [ ] The Make process-test executable is non-default and adds no generated repository artifact.
## Done summary
Wired the narrow semantic inventory facade, atomic generation and read-only drift checking, isolated real-Make process coverage, lint integration, and authority documentation. Generation now preserves checked bytes on renderer failure or termination, while missing, stale, and extra check states produce deterministic diffs without mutation.

baseline: green via conductor handoff (focused Quick failed only on task-owned missing targets before implementation; full lint green)
stage: impl-review - ran [02e8c1c24; SHIP session 01a05d80-69e9-7b10-b689-89c0f829d0d5]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 02e8c1c24a626ee2b21699d14896e7a497d5d463, 0823aa0292f4b671a551661949d9f59f46a32760
- Tests: baseline: green via conductor handoff (focused Quick failed only on absent task-owned Make-test target; make lint-model rc0), cd model && mise exec -- lake build temporal-model-semantic-inventory-make-tests && mise exec -- lake exe temporal-model-semantic-inventory-make-tests, cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests temporal-model-semantic-inventory temporal-model-semantic-inventory-tests temporal-model-semantic-inventory-make-tests, cd model && mise exec -- lake exe temporal-model-semantic-inventory-tests, cd model && mise exec -- lake exe temporal-model-semantic-inventory-make-tests, make umpire-check-semantic-inventory, make lint-model
- PRs: