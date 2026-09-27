---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.15 Delete the offline Evidence evaluation chain (B2)

## Description
Lane B2, last part. It runs after .13 and .14 removed the chain's remaining consumers (Application's observed-trace path, the Nexus evaluate path, `Artifact/Result`); any other importer found now blocks the task (R9). Delete `Evidence/Evaluate/{Raw,Structure,Admission}` (~1,850), `Evidence/Reading/Check` (946), `Evidence/Check` (281), `Evidence/PropertyStatus` (450) and their tests (`Evidence/Tests/*` ~4,000, `Evidence/ImportTests`). Decision D2 was recorded in task 11; if it was declined, close with no change. **Decision interaction:** if D3 was declined (task .12), `Artifact/Result` survives and imports `Evidence.PropertyStatus`, which pulls in the `Evidence.Evaluate` facade; this task then deletes only the chain modules outside the kept artifacts' import closure and lists the kept ones in the receipt. That is the spec's decision-interaction rule, not an R9 block. Likewise if D1 was declined, anything the kept Variations package imports stays.

**Size:** M
**Files:** the modules above; `model/Umpire/Evidence.lean` (facade), `model/Umpire/Evidence/Evaluate.lean`, `model/UmpireTests.lean`, `model/Umpire/Case/Tests/ProjectionBoundary.lean`, `model/Umpire/Inventory/Tests/SemanticStages.lean`, `model/Temporal/Tool/Inventory.lean` (drop the Property status family, ~`:61-62`), `model/INVENTORY.md` (regenerated: those rows leave), `model/Umpire/Inventory/KnownGaps.lean` (Known Gap rows whose source is deleted, per DG3), `model/ModelLint/ModuleIndex.lean`, `model/HANDWRITTEN_INVENTORY.md`, `tools/umpire/internal/retiredvocabulary/check.go`, `tools/umpire/CONTEXT.md:49-51` (Evidence structure term), `.plans/UMPIRE4_SPEC.md` (Evidence glossary ~591-594 names `Umpire.Evidence.PropertyStatus`), `model/README.md:114-151`, `model/ARCHITECTURE.md:30,98,335`, `model/Umpire/ARCHITECTURE.md:31,112`, E2 entries (`Evidence/Tests/Compilation` pins)
**Touches:** [model/Umpire/Evidence/**, model/Umpire/Evidence.lean, model/UmpireTests.lean, model/Umpire/Case/Tests/**, model/Umpire/Inventory/**, model/Temporal/Tool/Inventory.lean, model/INVENTORY.md, model/ModelLint/ModuleIndex.lean, model/HANDWRITTEN_INVENTORY.md, tools/umpire/internal/retiredvocabulary/**, tools/umpire/CONTEXT.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_*.md, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Re-verify importers; any unlisted production importer blocks (R9).
- Regenerate INVENTORY.md with the existing target and diff it: only rows of deleted families and DG3-dropped Known Gaps may leave; any other change fails (R10).
- Vocabulary gate: module paths (dotted + slash) and compound identifiers; narrow any token that would hit kept code (`RawEvidence` vs `internal/artifactv2`).
- `UMPIRE4_SPEC.md` Evidence glossary: rewrite the description (D9) so `spec_names_test` passes; no rule text changes here.

### Investigation targets
**Required:**
- `model/Temporal/Tool/Inventory.lean:1-70` — families and imports
- `model/Umpire/Inventory/KnownGaps.lean` — Known Gap sources
- Task 11's receipt (DG3 list)

### Quick commands
```sh
cd model && lake build
make umpire-check-inventory umpire-check-goldens umpire-check-retired-vocabulary umpire-check-plan-index
go test ./tools/umpire/vocabulary/... ./tools/umpire/internal/retiredvocabulary/...
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] Every listed module outside the import closure of modules kept by a declined decision is deleted with its tests; retained modules and their Inventory families are listed in the receipt (none when D1 and D3 were taken)
- [ ] No surviving module imports a deleted one
- [ ] INVENTORY.md lost exactly the rows of the families and DG3 Known Gaps whose modules were deleted; every other golden byte-identical
- [ ] Gate entries added; `spec_names_test` green; docs that described the chain updated
- [ ] Every gate in Quick commands green
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
