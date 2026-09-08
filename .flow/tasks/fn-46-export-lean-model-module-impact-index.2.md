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
- Add an exhaustive ModuleClass-to-v1-string renderer covering all 14 constructors, including testpilot and reserved classes without current source rows. Preserve classifier precedence.
- Reconcile inputs before constructing one direct adjacency and one reverse adjacency; reject cycles and invalid roots.
- Compute reflexive reachability only from the exact 34 facade and 15 test roots over the full validated graph. Emit actual first-party direct/reverse imports only, with no shortcut edge through an external intermediary; external bridge paths still contribute root impact. Then project/sort/validate closed rows and compact JSON bytes in memory.
- Normalize harmless source/module/edge permutations and valid platform path spellings; reject duplicate identities/edges and malformed/unsafe values rather than silently deduplicating. Keep every closed JSON array even when empty; semantic-artifact omit-empty rules do not apply.
- Wire ModuleIndexTests into the existing executable modelLintTests runner. Include external-return paths, disconnected/diamond/fanout graphs, root-self/multi-root reachability, all configured roots, unknown/absent roots, cycle rejection and multiple deterministic issues. Measure pure index construction/projection at roughly 10x source count, separately from Lake/OLean work, without an all-pairs path table.

### Investigation targets
**Required** (read before coding):
- `model/Tools/LeanImportGraph.lean:37` — deterministic traversal and existing edge deduplication to avoid hiding invalid inputs.
- `model/Tools/LeanSourceInventory.lean:89` — source identities and reconciliation.
- `model/ModelLint/ImportGraph.lean:22` — 14 classification cases and current default policy.
- `model/ModelLint/ImportGraphTests.lean` — synthetic module-policy fixtures and executable test runner.
- Task 1's delivered PackageModules result — full metadata and compacted-region ownership through consumers.

**Optional** (reference as needed):
- Lean `Json.compress` usage in existing canonical artifact renderers.

### Quick commands
`cd model && mise exec -- lake -q build modelLintTests && mise exec -- lake exe modelLintTests`

### Execution constraints
Preserve all parent R1–R5 and the exact reviewed root arrays; no filename discovery or silent unknown-root omission. Preserve comments/unrelated work and existing semantic import isolation. No staging/commits/pushes, new library or cancellation work. Lean jobs serial; new tests must execute through the normal runner. Required nonfixing Go lint uses exact inherited-set comparison, never count-only allowance or new-failure suppression.
## Acceptance
- [ ] Every reconciled first-party source produces one exact closed row with correct direct/reverse edges, exact classification spelling, and explicit reflexive facade/test reachability.
- [ ] Serializer tests cover all 14 ModuleClass constructors; all 34 configured facade and 15 test roots exist/classify without heuristics.
- [ ] Root-self, descendant, disconnected, and multi-root projections are pinned.
- [ ] Duplicate/missing endpoints, cycles, unknown roots, unclassified modules and malformed noncanonical values reject atomically; harmless input permutations normalize.
- [ ] Reordered/path-normalized inputs are byte-identical; 10x fixtures avoid all-pairs traversal; no semantic/external row is emitted.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
