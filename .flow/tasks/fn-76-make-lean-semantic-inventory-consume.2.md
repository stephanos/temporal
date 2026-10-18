---
satisfies: [R2, R7]
---
# fn-76-make-lean-semantic-inventory-consume.2 Enforce inventory and neutral-classification dependency boundaries

## Description
Extend the existing graph policy for the migrated ownership boundary.

**Size:** M
**Files:** ModelLint.ImportGraph and ModelLint.ImportGraphTests.
**Touches:** [model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean]

### Approach
- Reuse existing production/test classification and fn75 external-aware traversal. Reject SemanticInventory reachability from every production Umpire source outside that namespace, including root umbrella, newly added helpers, Artifact.Result and ImplementationLink; do not enumerate only current offenders.
- Enforce neutral OutcomeClassification independence from concrete owners, Core/KnownGap, inventory, Temporal and aggregation, allowing only its selected minimal foundation. Do not disguise it as inventory.
- Add direct, helper, facade, external-bridge and test-fixture-bridge rejection; allowed inventory-to-owner and dedicated Planning KnownGap test cases; cycles and deterministic shortest paths. Preserve fn75 Target isolation and missing-owned-metadata reconciliation.

### Investigation targets
**Required:**
- model/ModelLint/ImportGraph.lean:237
- model/ModelLint/ImportGraph.lean:258
- model/ModelLint/ImportGraphTests.lean:314
- model/Tools/LeanImportGraph.lean
- model/Tools/LeanImportGraph/Metadata.lean
- model/Umpire/Planning/Tests/KnownGaps.lean:1

### Verification
From model, serial mise exec -- lake build modelLintTests modelLint; then lake exe modelLintTests and lake exe modelLint. Preserve the exact controlled-violation diagnostic. Run required non-fixing Go lint and classify inherited output precisely.

## Acceptance
- [ ] Universal production inventory isolation and minimal neutral-module independence are enforced transitively with deterministic diagnostics.
- [ ] Direct/indirect/external/test-bridge negatives and reverse-direction/test-consumer positives pass.
- [ ] Existing graph policies, inventory reconciliation and controlled diagnostic remain unchanged.
- [ ] Real whole-source modelLint passes in addition to the focused synthetic regressions.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: