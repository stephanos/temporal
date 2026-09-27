---
satisfies: [R14]
---
# fn-93-simplify-the-lean-model.31 DefinitionId ordering, first-duplicate and required-ID helpers (A8)

## Description
Lane A8, part one. Public `LE DefinitionId` (with `DecidableRel`) on `.value` replaces the surviving private `idLe`/`definitionIdLe` copies (`Shared/DefinitionGraph:66`, `Artifact/Planning:5`, `Case/Projection:79`, plus any kept by declined decisions), the 24 `xLe := idLe` wrappers and the ~18 `decide (….id.value ≤ …)` bodies; `DefinitionId.canonicalSet` replaces surviving `canonicalIds` copies; one `stringLe` replaces the survivors of three. One `List.firstDuplicateBy?` (input order) and the existing after-sort `firstDuplicate` in `Umpire/Core` replace the recursive copies, each site keeping its variant. One generic `requireDefinitionId`/`requireUniqueIds` pair over A5's `Diagnostic` kind (keeps `"<empty>"`).

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Evaluation.lean` (existing `firstDuplicate [BEq α]` ~132, merge), survivors: `model/Umpire/Property/Check.lean`, `model/Umpire/Scenario/Check.lean`, `model/Umpire/Model/Check.lean`, `model/Umpire/Model/Canonical.lean`, `model/Umpire/Query/Check.lean`, `model/Umpire/Shared/DefinitionGraph.lean`, `model/Umpire/Artifact/Planning.lean`, `model/Umpire/Case/Projection.lean`, `model/Umpire/Inventory/KnownGaps.lean`, `model/Umpire/ImplementationLink/**`, `model/Temporal/System/Callback/Configuration.lean`, `model/Temporal/System/Configuration/Core.lean`, `model/Umpire/Search.lean`, `model/Umpire/Search/Product/Monitor.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/Evaluation.lean, model/Umpire/Property/Check.lean, model/Umpire/Scenario/Check.lean, model/Umpire/Model/**, model/Umpire/Query/Check.lean, model/Umpire/Shared/**, model/Umpire/Artifact/Planning.lean, model/Umpire/Case/Projection.lean, model/Umpire/Inventory/KnownGaps.lean, model/Umpire/ImplementationLink/**, model/Temporal/System/**, model/Umpire/Search.lean, model/Umpire/Search/**]

### Approach
- Re-count after lane B (`grep -rnE 'private def (idLe|definitionIdLe|canonicalIds|stringLe)'`, recursive `firstDuplicate`-shaped helpers).
- Lookalikes that compare more than `.value` keep their own relation (`Property/Check` `meaningLe`, `Canonical.definitionLe`).
- Sorted output and duplicate diagnostics are pinned by goldens and `#guard_msgs`; each changed declaration keeps its axiom inventory under E2.

### Investigation targets
**Required:**
- `model/Umpire/Core.lean:30-50`
- `model/Umpire/Evaluation.lean:125-145`
- `model/Umpire/Property/Check.lean:330-360`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] No private `DefinitionId` comparator, `canonicalIds` copy or recursive first-duplicate helper remains (lookalikes excepted, listed)
- [ ] Required-ID checks are one generic pair
- [ ] Sorts and duplicate diagnostics byte-identical; no axiom inventory widened


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
