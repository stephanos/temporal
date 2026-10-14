---
satisfies: [R19]
---
# fn-93-simplify-the-lean-model.37 Retire the handwritten-inventory ledger and dead lint policy (D-lint)

## Description
Lane D-lint. Retire `HANDWRITTEN_INVENTORY.md`: `Temporal.Testpilot` joins `authoringPathRoots` with `CaseSupport` and `Conformance` as `authoringPathExceptions`; delete the reconcile code, its tests and the Markdown file. Delete the dead `nexusExperimentalIsolation` rule (`ImportGraph.lean` ~89, 213, 304-306) and its tests, and the stale `TemporalExperimentalTests` and `GenerateTestsIOTestsMain` policy entries. Move `Temporal/Tool/Inventory.lean` (imports only Umpire) under `Umpire/`.

**Size:** M
**Files:** `model/ModelLint/ImportGraph.lean` (`Rule` enum ~79-101; ledger readers ~246-248, 449, 464, 480-505; policy ~132, 148, 155-178), `model/ModelLint.lean:25,48-50`, `model/Tools/LeanSourceInventory.lean:39,67`, `model/ModelLint/ImportGraphTests.lean` (~115-160, 236, 505, 597-627, 383, 404), `model/HANDWRITTEN_INVENTORY.md` (delete), `Makefile:1115` (ledger check), `model/Temporal/Tool/Inventory.lean` → `model/Umpire/Tool/Inventory.lean` (or similar) with its `InventoryMain`/tests, `model/lakefile.lean` exe roots (~112-119), `Makefile` inventory targets, `model/ModelLint/ModuleIndex.lean`
**Touches:** [model/ModelLint/**, model/ModelLint.lean, model/Tools/**, model/HANDWRITTEN_INVENTORY.md, Makefile, model/Temporal/Tool/**, model/Umpire/Tool/**, model/lakefile.lean]
**Depends on other specs:** fn-88.5 (`searchBackendIsolation` in the same enum) and fn-92.1 (`feature-entity-uniqueness` lint, a ledger row) — rebase on both.

### Approach
- Before deleting the ledger, confirm the new `authoringPathRoots` policy rejects every module the ledger rejected (run lint on the tree plus the existing controlled violations); a module the ledger rejected that the policy admits fails the task.
- Keep every surviving rule's controlled violation in `lint-model`.
- The inventory move keeps `make umpire-check-inventory` output byte-identical.

### Investigation targets
**Required:**
- `model/ModelLint/ImportGraph.lean:75-180,240-250,440-510`
- `Makefile:1100-1145`
- `model/lakefile.lean:84-158`

### Quick commands
```sh
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-inventory umpire-check-model-module-index
```

## Acceptance
- [ ] Ledger, reconcile code, its tests and dead rule gone; policy covers `Temporal.Testpilot` with the two exceptions
- [ ] Inventory tool lives under `Umpire/`; INVENTORY.md byte-identical
- [ ] `lint-model` green with every controlled violation still firing


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
