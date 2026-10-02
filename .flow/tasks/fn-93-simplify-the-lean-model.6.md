---
satisfies: [R4]
---
# fn-93-simplify-the-lean-model.6 WireName for the remaining enums and the lists that restate names (A2)

## Description
Lane A2, last task. Convert the surviving name functions outside tasks 4 and 5, and delete the lists that restate them. Skip every module lane B deletes (`Artifact/{RunRecord,Evidence,Result,Set}`, `Evidence/PropertyStatus`, `Evidence/Reading/Check`, `Search/Branches`, `Variations/**`, the `*AuthoringRole` blocks inside the B4 syntax tails); `ImplementationLink/Application`'s two functions wait for B2.3 to decide what survives.

**Size:** M
**Files:** `model/Umpire/ImplementationLink/Language.lean` (3), `model/Umpire/Evidence/Reading.lean` (2), `model/Umpire/Evidence/Evaluate/Types.lean` (1), `model/Umpire/Exploration/Ledger.lean` (2), `model/Temporal/System/Configuration/Core.lean` (1), `model/ModelLint/*.lean` (2), `model/Umpire/Inventory/KnownGaps.lean` (`observationKnownGapSuffixes`, `observationFailureKnownGapSuffix`), `model/Umpire/Fingerprint.lean` (if its parser has no aliases)
**Touches:** [model/Umpire/ImplementationLink/Language.lean, model/Umpire/Evidence/Reading.lean, model/Umpire/Evidence/Evaluate/Types.lean, model/Umpire/Exploration/Ledger.lean, model/Temporal/System/Configuration/Core.lean, model/ModelLint/**, model/Umpire/Inventory/KnownGaps.lean, model/Umpire/Fingerprint.lean, model/Umpire/Tests/**]
**Depends on other specs:** fn-88.10 edits `Exploration/**`.

### Approach
- Same loop as task 4. For the two Known Gap suffix lists, replace each with `T.all.map T.name` (or the derived name directly) after a `#guard` that the list equals the derived one, in order.
- `ModelLint` is outside `Umpire`; it may use the derivation only if its import policy admits the module chosen in task 4 — otherwise keep and list.
- `gapKindTerm`'s parse in `Command/Syntax.lean` belongs to task 7 (A3), not here.

### Investigation targets
**Required:**
- `model/Umpire/Inventory/KnownGaps.lean:200-215,500-512` — the restating lists
- `model/Umpire/Fingerprint.lean` — `ofName?`/`parse?` pair
- Task 4's receipt

### Quick commands
```sh
cd model && lake build
make umpire-check-inventory && make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] Every surviving hand-written enum-to-string function outside lane B's modules is derived or listed with its reason
- [ ] No list restates a name function (Known Gap suffix lists gone)
- [ ] INVENTORY.md, goldens, Fingerprints byte-identical; `lint-model` green


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
