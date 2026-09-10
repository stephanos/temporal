---
satisfies: [R5]
---
# fn-82-unify-the-umpire-and-testpilot.6 Evidence, Case.Projection, Variations, and Inventory

## Description
Evidence, Projection, Artifact records, Variations, and Inventory (R5, spec §R5): split
`Umpire/Observation/` by generation into `Umpire.Evidence` (offline) and `Umpire.Case.Projection`
(live seam), rename the mislabeled Known Gap and the offline verdict family, rename `Umpire.Space`
to `Umpire.Variations`, and rename the semantic inventory with its executable, targets, and document.

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Evidence/**` (from `Observation/{Declaration,Compiler,Language,Evaluation,Check,Verdict}`), `model/Umpire/Case/Projection/**` (from `Observation/Projection*`, `Evaluation/Scoped`), `model/Umpire/ImplementationLink/*.lean`, `model/Umpire/Artifact/{RunRecord,Result,Evidence,Set}.lean`, `model/Umpire/OutcomeClassification.lean`, `model/Umpire/Variations/**`, `model/Umpire/Exploration/*.lean`, `model/Umpire/Inventory/**`, `model/Temporal/Tool/Inventory.lean`, `model/INVENTORY.md`, `model/lakefile.lean`, `Makefile`, `model/ModelLint/ImportGraph.lean`, `model/Temporal/System/Nexus/Evidence.lean`, docs, gate
**Touches:** [model/Umpire/Observation*, model/Umpire/Evidence*, model/Umpire/Case/**, model/Umpire/ImplementationLink/**, model/Umpire/Artifact/**, model/Umpire/OutcomeClassification.lean, model/Umpire/Space*, model/Umpire/Variations*, model/Umpire/Exploration/**, model/Umpire/SemanticInventory*, model/Umpire/Inventory*, model/Temporal/**, model/SEMANTIC_INVENTORY.md, model/INVENTORY.md, model/lakefile.lean, model/ModelLint/**, Makefile, .flow/specs/*.md]

### Approach
- Move `Projection.lean`, `Projection/Declaration.lean`, `Projection/Coverage.lean`, `Evaluation/Scoped.lean` to `model/Umpire/Case/Projection/`; rename their `Fact` type parameter consistently (already `Fact` in `Projection.lean:17`, `Observation` elsewhere). Update the one external importer `model/Temporal/System/Nexus/Evidence.lean:1`.
- Rename the rest of `Observation/` to `Evidence/`: `ObservationMappingDeclaration` and checker to `Evidence.Reading`, `evaluateEvidence` under `Evidence.Evaluate`, `Verdict.lean` to `PropertyStatus.lean` (`SemanticVerdictStatus` to `Evidence.PropertyStatus`, `StrictQuerySummary` to `QueryStatusSummary`), `EvidenceLink` and mirrors to `EvidenceSupport`, `EvidenceBundle` to `SyntheticEvidence`.
- ImplementationLink: `ImplementationLinkKnownGap` to `UnmappedSource`, `ForwardSimulation` to `StepPreservation`, `KernelMorphism` to `ValueTranslation`, `notEvaluatedProjectionSentinel` to `stageNotRunMarker` (string `implementation-link.not-evaluated` unchanged); `ProjectionSentinelDescriptor` to `NotRunMarker`.
- Artifact: `Runtime.lean` to `RunRecord.lean` with `ObservationConfiguration` to `EvidenceSourceConfiguration`; the twenty `Artifact*` wire types in `Result.lean:11-242` drop the prefix inside `Artifact.Wire`; `KnownGapCarryMapping.{exact,observationAdmission}` to `{full,lossy}` (rendered into the inventory).
- Variations: `Umpire/Space/*` to `Umpire/Variations/*` with `VariationSpace`, `PlannedVariant`, `SpaceMetadataRows`; Exploration renames per spec (`CandidateSet`, `CandidateCursor`, `PinnedRegression`, `DroppedCandidate`).
- Inventory: `Umpire/SemanticInventory/*` to `Umpire/Inventory/*`; owner strings at `SemanticInventory/KnownGaps.lean:165,178,251,272,298,312,324,336` and `Temporal/Tool/SemanticInventory.lean:76` follow the new module names; Lake exes `temporal-model-semantic-inventory{,-tests,-make-tests}` to `umpire-inventory{,-tests,-make-tests}`; Make targets to `umpire-gen-inventory`/`umpire-check-inventory` (update `lint-model` prerequisite at `Makefile:1412` and the variables at `Makefile:135-137`); document to `model/INVENTORY.md`, regenerated. Lint rule `semanticInventoryIsolation` to `inventoryIsolation` with its label at `ImportGraph.lean:182`.
- Regenerate inventory and goldens; add compound old names to the gate; respell scanned docs and open specs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation.lean` and `model/Umpire/Observation/{Projection,Verdict,Evaluation}.lean` headers — the two generations sharing one directory
- `model/Umpire/SemanticInventory/KnownGaps.lean:150-360` — owner strings and catalog ids that render into the document
- `model/lakefile.lean:89-113` — executable definitions
- `Makefile:135-137,1108-1132,1412` — inventory variables, targets, and the `lint-model` prerequisite
- `model/ModelLint/ImportGraph.lean:66,176-190,268-272` — the inventory isolation rule and label

**Optional** (reference as needed):
- `model/Umpire/ImplementationLink/Language.lean:15-40,277,408` — the three renamed link types
- `model/Umpire/Artifact/Result.lean:11-242,269` — wire projections and carry mapping

### Key context
- Catalog ids inside the inventory (`umpire.semantic-inventory.*` rows, `implementation-link.not-evaluated`) are stable and stay; only owner strings and module names change.
- `Umpire.Case/Scoped.lean` is renamed by task .7; leave its name here even though it moves next to `Case/Projection/`.
- Retire `Umpire.Observation`, `SemanticVerdictStatus`, `ImplementationLinkKnownGap`, `Umpire.Space`, `ExperimentSpace`, `Umpire.SemanticInventory`, `SEMANTIC_INVENTORY`, `temporal-model-semantic-inventory`.

- Carried from .2: `Umpire.Artifact.Result.ArtifactModelTraceStep` now spells `outcome`/`state`/`facts`, matching `Step`; the remaining `Artifact*` wire projections in `Result.lean` still carry the prefix this task drops.
## Acceptance
- [ ] `Umpire.Evidence` holds the offline evaluator and `Umpire.Case.Projection` holds the live seam with `Fact` as its type parameter; `Umpire/Observation/` no longer exists
- [ ] `UnmappedSource`, `Evidence.PropertyStatus`, `NotRunMarker`, `Artifact.RunRecord`, `Umpire.Variations`, and the Exploration names match the spec
- [ ] `Umpire.Inventory`, `umpire-inventory`, `umpire-gen-inventory`/`umpire-check-inventory`, and `model/INVENTORY.md` replace the semantic inventory names with unchanged catalog ids; a production Umpire import of `Umpire.Inventory` fails `lint-model`
- [ ] `make lint-model`, `make umpire-check-inventory`, `lake build Umpire UmpireTests Temporal TemporalModelTests`, and regenerated goldens pass
- [ ] The gate rejects the retired compounds and passes on the tree


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
