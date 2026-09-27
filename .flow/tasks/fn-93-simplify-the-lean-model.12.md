---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.12 Move ArtifactBinding and the carry mapping off the doomed artifacts (B3, decision D3)

## Description
Lane B3, part one; byte-identical preparation. Production Exploration (`Campaign.lean:62`, `Session.lean:50`) uses `RunRecord`'s `ArtifactBinding` (~`:29`) and `Plan.artifactBinding` (~`:448`); `Inventory/KnownGaps` uses `ResultArtifact.knownGapCarryMapping` (`:307-330`). Move the binding into a kept Artifact module and decide the carry-mapping rows (DG3), so task .13 deletes nothing a kept module needs.

### Owner decision
- **D3 — delete the Run, Evidence, Result and Set artifacts. Recommended default: taken.** Record it first in the Done summary; if declined, close this task and task .13 with no change.

**Size:** S
**Files:** `model/Umpire/Artifact/RunRecord.lean` (source), `model/Umpire/Artifact/Planning.lean` or `model/Umpire/Artifact/Types.lean` (destination), `model/Umpire/Exploration/Campaign.lean`, `model/Umpire/Exploration/Session.lean`, `model/Umpire/Inventory/KnownGaps.lean`
**Touches:** [model/Umpire/Artifact/**, model/Umpire/Exploration/**, model/Umpire/Inventory/KnownGaps.lean]
**Depends on other specs:** `Exploration/**` is an fn-88.10 surface.

### Approach
- Move `ArtifactBinding` and `Plan.artifactBinding` unchanged (same namespace) into the destination module; point Exploration at it directly instead of the `Umpire.Artifact` facade if the facade will lose modules.
- `knownGapCarryMapping`: its Known Gap rows describe the Result artifact; per DG3 grep Go for those codes. If unused, prepare `KnownGaps` to drop the rows in task .13 (keep this task byte-identical); if used, move the mapping instead.

### Investigation targets
**Required:**
- `model/Umpire/Artifact/RunRecord.lean:20-40,440-460`
- `model/Umpire/Exploration/Campaign.lean:55-70`, `model/Umpire/Exploration/Session.lean:45-55`
- `model/Umpire/Inventory/KnownGaps.lean:300-330`

### Quick commands
```sh
cd model && lake build
make umpire-check-inventory umpire-check-goldens
```
## Acceptance
- [ ] D3 recorded
- [ ] No kept module needs anything from RunRecord, Evidence, Result or Set artifacts
- [ ] Everything byte-identical, INVENTORY.md included
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
