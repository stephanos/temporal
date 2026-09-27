---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.11 Move the kept Evidence types off the doomed chain (B2, decision D2)

## Description
Lane B2, part one; byte-identical preparation. Two kept modules import modules B2 deletes: `Evidence/Evaluate/Types` imports `Evidence/Reading/Check`, and `Inventory/KnownGaps` imports the `Evidence.Evaluate` facade. Move whatever they use onto kept modules (`Evidence/Reading.lean`, `Evidence/Evaluate/Types.lean`) so tasks .14 and .15 can delete without touching a kept consumer.

### Owner decision
- **D2 — delete the offline Evidence evaluation chain. Recommended default: taken.** Record it as the first line of the Done summary. If declined, close this task and tasks .14 and .15 with the decision recorded and no change.

**Size:** M
**Files:** `model/Umpire/Evidence/Evaluate/Types.lean`, `model/Umpire/Evidence/Reading.lean`, `model/Umpire/Evidence/Reading/Check.lean` (source of moved definitions), `model/Umpire/Inventory/KnownGaps.lean` (narrow the import), `model/Umpire/Evidence/Evaluate.lean` (facade), any Case/Testpilot/Inventory reader whose import path changes
**Touches:** [model/Umpire/Evidence/**, model/Umpire/Inventory/KnownGaps.lean, model/Umpire/Case/**, model/Testpilot/**, model/Temporal/Tool/Inventory.lean]
**Depends on other specs:** `Case/**` is an fn-89 surface.

### Approach
- List exactly which declarations `Evaluate/Types` and `KnownGaps` use from the doomed modules (`EvidenceValue`, `ObservationStatus`, `ObservationDiagnostic`, the `Reading.lean` field/disposition types are kept by the spec). Move those definitions, unchanged, into the kept module; leave namespace names as they are so no Definition ID or rendered string moves.
- Build; goldens, INVENTORY.md and Case fixtures must be byte-identical (this task deletes nothing).
- Also grep Go (`tools/`, `common/`) for Known Gap codes whose source lives in the chain (INVENTORY.md sources 01-04, 09) and record per code whether Go emits or reads it; task .15 uses the list (DG3).

### Investigation targets
**Required:**
- `model/Umpire/Evidence/Evaluate/Types.lean:1` and its uses of `Reading.Check`
- `model/Umpire/Inventory/KnownGaps.lean:1-10,300-330`
- `model/Umpire/Evidence/Reading.lean`

### Quick commands
```sh
cd model && lake build
make umpire-check-inventory umpire-check-goldens umpire-check-regression
```
## Acceptance
- [ ] D2 recorded
- [ ] No kept module imports `Evidence/Evaluate/{Raw,Structure,Admission}`, `Evidence/Reading/Check`, `Evidence/Check` or `Evidence/PropertyStatus`, except modules kept by a declined decision (e.g. `Artifact/Result` when D3 is declined), which are listed
- [ ] Everything byte-identical, INVENTORY.md included; Known Gap code list for DG3 in the receipt
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
