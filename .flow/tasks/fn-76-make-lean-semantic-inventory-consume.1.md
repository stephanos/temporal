---
satisfies: [R1, R2, R3, R4, R6, R7]
---
# fn-76-make-lean-semantic-inventory-consume.1 Extract semantic classification and Known Gap carry contracts and migrate consumers

## Description
Relocate the existing shared contracts and complete their import migration in one compilable change. Capture the original trust and output evidence before editing.

**Size:** M
**Files:** Umpire.OutcomeClassification, KnownGap, SemanticInventory.Types, five semantic owners, Umpire umbrella, affected explicit inventory consumers and import tests.
**Touches:** [model/Umpire.lean, model/Umpire/**, model/Temporal/**/*.lean]

### Approach
- Before source edits, capture original moved declaration bodies, generated inventory bytes, relevant canonical fixture hashes, and full raw transitive axiom inventories. Include all ten concrete ExactlyOne proofs, carry mappings/renderings, projection sentinel, and generated auxiliaries; retain complete multiline arrays and qualified-name mapping.
- Extract classifier/projection declarations from SemanticInventory.Types into OutcomeClassification using only the minimal established Lean/Std foundation. Move KnownGapCarryMapping/name to KnownGap. Preserve names, signatures, bodies and comments; leave concrete classifiers/proofs/mappings at their owners.
- Replace all five production Types imports with semantic owners; add explicit KnownGap imports to Result/Evidence consumers where needed. Remove inventory from the Umpire umbrella and migrate exposed catalog consumers to explicit imports. Wider Umpire/Temporal Touches permit these import-only repairs, not unrelated body changes. Do not reexport through Core or leave broken consumers for task 2.
- Add a focused import qualification proving the neutral public contract compiles without inventory or concrete-stage imports.

### Investigation targets
**Required:**
- model/Umpire/SemanticInventory/Types.lean:13
- model/Umpire/KnownGap.lean:1
- model/Umpire/Planning/Engine.lean:576
- model/Umpire/ImplementationLink/Application.lean:3
- model/Umpire/Artifact/Result.lean:268
- model/Umpire.lean:14
- model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean:104

### Verification
Use serial LEAN_NUM_THREADS=1 and mise. From model, build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests and the new neutral import test. Run make umpire-check-semantic-inventory without publishing. Record required non-fixing Go lint against the inherited baseline.

## Acceptance
- [ ] Original raw trust inventories and output/fixture hashes are captured before edits and remain immutable.
- [ ] Shared contracts have their planned semantic owners; all production imports and affected catalog consumers compile with unchanged declaration bodies and qualified APIs.
- [ ] Focused outcome/carry/catalog tests and neutral import qualification pass; checked inventory bytes remain unchanged.
- [ ] No new semantic behavior, proof trust, generated expectations, or fn75/fn79 scope changes are introduced.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: