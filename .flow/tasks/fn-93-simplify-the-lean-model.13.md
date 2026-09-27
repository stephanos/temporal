---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.13 Delete the Run, Evidence, Result and Set artifacts (B3)

## Description
Lane B3, part two. Delete `Artifact/{RunRecord,Evidence,Result,Set}` (~1,980 production, ~1,040 tests), their goldens, Goldens-tool entries, Fingerprint derive functions and Inventory families. D3 was recorded in task .12; if it was declined, close with no change (tasks .14 and .15 then keep what the artifacts import). It runs before the observed-trace deletion (.14) and the Evidence chain deletion (.15), because `Artifact/Result.lean` imports both `ImplementationLink.Application` and `Evidence.PropertyStatus`.

**Size:** M
**Files:** the four modules and `model/Umpire/Artifact/Tests/{RunRecord,Evidence,Result,Set}.lean`; goldens `model/Umpire/Artifact/Tests/Fixtures/{RuntimeConfiguration,ExperimentRun,RawEvidence,Evidence,Result,ArtifactSet}V2.json`; `model/Umpire/Artifact/Tests/Goldens.lean` (their entries); `model/Temporal/Tool/Goldens.lean:2,48-66`; `model/Umpire/Fingerprint.lean:196-223` (their `ChecksumOf` functions); `model/Temporal/Tool/Inventory.lean` + `model/INVENTORY.md` (run-record families 01-06, Result Known Gap rows); `model/Umpire/Inventory/KnownGaps.lean`; `model/Umpire/Inventory/Tests/PlanningRuntime.lean`; `model/Umpire/Artifact.lean` (facade); `model/UmpireTests.lean`; `model/ModelLint/ModuleIndex.lean`; `model/HANDWRITTEN_INVENTORY.md`; vocabulary gate; docs (`model/ARCHITECTURE.md:335-336`, `model/Umpire/ARCHITECTURE.md:350-354`)
**Touches:** [model/Umpire/Artifact/**, model/Umpire/Artifact.lean, model/Temporal/Tool/Goldens.lean, model/Umpire/Fingerprint.lean, model/Temporal/Tool/Inventory.lean, model/INVENTORY.md, model/Umpire/Inventory/**, model/UmpireTests.lean, model/ModelLint/ModuleIndex.lean, model/HANDWRITTEN_INVENTORY.md, tools/umpire/internal/retiredvocabulary/**, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/README.md, .plans/UMPIRE4_*.md]

### Approach
- Keep `Artifact/Planning`, `Codecs`, `Types` and `SwitchPlanV2.json` (read by `tools/umpire/internal/artifactv2/artifact_test.go:17`).
- Delete the duplicate `include_str` comparisons inside the deleted modules with them (D-goldens handles the survivors).
- Vocabulary gate: module paths and compound identifiers; check `ExperimentRun`, `RawEvidence`, `ArtifactSet` tokens against kept code and `internal/artifactv2` (lowerCamel and `V<n>` variants match).
- Regenerate INVENTORY.md and goldens; diff: only deleted rows/files.

### Investigation targets
**Required:**
- `model/Temporal/Tool/Goldens.lean`
- `model/Umpire/Fingerprint.lean:190-230`
- `tools/umpire/internal/retiredvocabulary/check.go:745-760` — token matching variants

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-inventory umpire-check-retired-vocabulary umpire-check-plan-index umpire-check-regression
go test ./tools/umpire/internal/artifactv2/...
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] Four artifact modules, tests, six goldens, their tool entries and Fingerprint functions gone
- [ ] Surviving goldens byte-identical; INVENTORY.md lost only the deleted rows; `internal/artifactv2` test green
- [ ] Gate entries added without false hits; every gate green
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
