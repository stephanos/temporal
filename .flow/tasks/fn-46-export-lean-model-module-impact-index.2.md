---
satisfies: [R2, R3, R5]
---
# fn-46-export-lean-model-module-impact-index.2 Build the pure deterministic module impact index

## Description
Implement the pure `temporal-model-module-index/v1` projection for R2/R3.

**Size:** M
**Files:** `model/ModelLint/ModuleIndex.lean`, `model/ModelLint/ModuleIndexTests.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/ModelLint/ImportGraph.lean`
**Touches:** [model/ModelLint/ModuleIndex.lean, model/ModelLint/ModuleIndexTests.lean, model/ModelLint/ImportGraphTests.lean, model/ModelLint/ImportGraph.lean]

### Approach
- Define `IndexPolicy` with exactly the v1 facade/test arrays in the parent spec and reuse `defaultPolicy` classification; no filename heuristics.
- Add an exhaustive ModuleClass-to-v1-string renderer covering all 13 constructors.
- Reconcile inputs before constructing one direct adjacency and one reverse adjacency; reject cycles and invalid roots.
- Compute reflexive reachability only from configured roots, then project/sort/validate closed rows and compact JSON bytes in memory.

### Investigation targets
**Required** (read before coding):
- `model/Tools/LeanImportGraph.lean:28-84` — deterministic graph conventions.
- `model/Tools/LeanSourceInventory.lean:43-117` — unique sorting and reconciliation.
- `model/ModelLint/ImportGraph.lean:21-54,78-146` — classification vocabulary/policy.
- `model/ModelLint/ImportGraphTests.lean:185-333` — synthetic module-policy fixtures.

**Optional** (reference as needed):
- Lean `Json.compress` usage in existing canonical artifact renderers.

### Quick commands
`cd model && mise exec -- lake -q build modelLintTests && mise exec -- lake exe modelLintTests`
## Acceptance
- [ ] Every reconciled first-party source produces one exact closed row with correct direct/reverse edges, exact classification spelling, and explicit reflexive facade/test reachability.
- [ ] Serializer tests cover all 13 ModuleClass constructors and all configured roots exist/classify.
- [ ] Root-self, descendant, disconnected, and multi-root projections are pinned.
- [ ] Duplicate/missing endpoints, cycles, unknown roots, unclassified modules, and noncanonical input reject atomically.
- [ ] Reordered/path-normalized inputs are byte-identical; 10x fixtures avoid all-pairs traversal; no semantic/external row is emitted.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
