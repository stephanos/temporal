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
All five acceptance bullets are met and every gate is at or under its baseline.

Landed in commits 61bc4cc11e, ba0885cd26, e6ab89152a, 0cbc1c33c4, dc7f6e7af1, 953e10da56,
b4ad2eccd0, 6a5df0c8df:

- `Umpire/Observation/` held two generations under one name. The live seam --
  `Projection.lean`, `Projection/{Declaration,Coverage}.lean` and
  `Evaluation/Scoped.lean` -- moves to `Umpire/Case/Projection/` with its tests and keeps
  `Fact` as its type parameter; `Case/Projection/Scoped.compile` now names
  `Property.Scoped.Limits` explicitly, because under `Umpire.Case.Projection` a bare
  `Limits` resolves to the projection's own abbrev.
- The rest becomes `Umpire/Evidence/`: `Declaration.lean` + `Compiler.lean` +
  `Language.lean` are `Evidence/Reading.lean` and `Evidence/Reading/Check.lean`,
  `Evaluation/` is `Evaluate/`, `Verdict.lean` is `PropertyStatus.lean`. The reading
  vocabulary follows the module (`Evidence.{Reading, ReadingSpec, CheckedReading,
  ReadingContext, ReadingError, ReadingErrorKind, checkReading, checkedReading}`),
  `SemanticVerdict*` is `Evidence.PropertyStatus*`, `StrictQuery{Status,Summary}` is
  `Query{Status,StatusSummary}`, `EvidenceLink` and its two mirrors are `EvidenceSupport`,
  and `EvidenceBundle` is `SyntheticEvidence`. `Observation` keeps its one meaning -- a
  declared typed Run value -- so the profile, kind, rule, field, expression and evaluation
  diagnostic records keep their names.
- `ImplementationLinkKnownGap` is `UnmappedSource` (it is a hole in a mapping's source
  domain, not a Known Gap), `ForwardSimulation` is `StepPreservation`, `KernelMorphism` is
  `ValueTranslation`, `ProjectionSentinelDescriptor` is `NotRunMarker`, and
  `notEvaluatedProjectionSentinel` is `stageNotRunMarker` with its rendered
  `implementation-link.not-evaluated` string unchanged.
- `Artifact/Runtime.lean` is `RunRecord.lean` with `ObservationConfiguration` as
  `EvidenceSourceConfiguration`; the twenty-two wire projections in `Result.lean` drop the
  `Artifact` prefix inside one `Artifact.Wire` namespace, and `EvidenceArtifact` moves
  beside `ResultArtifact`; `KnownGapCarryMapping.{exact, observationAdmission}` are
  `{full, lossy}` with their rendered strings unchanged.
- `Umpire/Space/` is `Umpire/Variations/` with `VariationSpace`, `PlannedVariant` and
  `SpaceMetadataRows`; Exploration is `CandidateSet`, `CandidateCursor`,
  `PinnedRegression` and `DroppedCandidate`.
- `Umpire/SemanticInventory/` is `Umpire/Inventory/`, `Temporal/Tool/SemanticInventory*` is
  `Inventory*`, the Lake executables are `umpire-inventory{,-tests,-make-tests}`, the Make
  targets are `umpire-gen-inventory`/`umpire-check-inventory` (with the `lint-model`
  prerequisite and `umpire-check-regression` following), and the document is
  `model/INVENTORY.md`. The `ModelLint` rule is `inventoryIsolation`, so a production
  Umpire import of `Umpire.Inventory` still fails `lint-model`; `ImportGraphTests` covers
  the direct, bridged and helper cases. Catalog ids -- the `umpire.semantic-inventory.*`
  rows and `implementation-link.not-evaluated` -- are byte-unchanged; only owner strings
  and module names moved.
- The gate rejects thirty new compounds covering every retired name above, and drops the
  `Umpire.Observation.Qualification` entry the new `Umpire.Observation` rule subsumes.

Two things a reader should know:

1. `evidenceLinks` and `evidenceLinkBehaviorFingerprint` are JSON member keys inside
   `umpire-evidence/v2` and `umpire-result/v2`; renaming the Lean types renamed them, which
   rewrote three fixture checksums, the artifact-set identity and
   `evaluationOutcomeChecksum`. The `formatVersion` identifiers are untouched, no Go code
   reads those keys, and task .2 already crossed this line for `outcome`/`state`/`facts`.
   The rendered diagnostic strings `evidence-link-mismatch` and `inconsistent-evidence-link`
   are deliberately kept for catalog stability.
2. Carried debt, from task .2 and outside this task's files: `Umpire/Core.lean:307-329`
   still names the finite-domain record's model-fact type parameter `Observation`
   (`observationDomain`, `observations`, `encodeObservation`, `observationSound/Complete`).
   The spec's vocabulary table retires "the `Observation` type parameter" in favour of
   `Fact`. It reaches `Umpire.Model`, `Umpire.ImplementationLink` and the Nexus features,
   so it belongs in one deliberate pass rather than as a side effect here.

Review: NEEDS_WORK then SHIP. The first round found `PinnedPlan` where the spec says
`PinnedRegression` (task .5's `ExperimentSpec` -> `Plan` sweep had renamed it in passing),
`ArtifactIntent*` compounds the gate's boundary regex cannot see, stale Space-era names in
`UMPIRE4_DSL.md`, a relabelled inventory-hash row, and a gate rule subsumed by a new one;
all are fixed in b4ad2eccd0. The second round found open Flow records half-respelled to
paths that exist in neither tree; fixed in 6a5df0c8df.
## Evidence
- Commits: 61bc4cc11e, ba0885cd26, e6ab89152a, 0cbc1c33c4, dc7f6e7af1, 953e10da56, b4ad2eccd0, 6a5df0c8df
- Tests: cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot TemporalExperimentalTests +Umpire.PromotionTests (pass, 401 jobs), make umpire-check-regression (pass; goldens, regression-views, testpilot-protocol, testpilot-authoring, case-runtime-conformance, inventory, retired-vocabulary, lean-api, live-tests, the model layout assertions, and go test ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...), make umpire-check-inventory (pass), make umpire-build-model (pass, includes the Makefile package-layout check), cd model && lake exe umpire-inventory-tests (pass); TMPDIR=<physical> lake exe umpire-inventory-make-tests (pass), make lint-model (169 diagnostics, all generated Temporal/API; equals baseline), make lint-code GOLANGCI_LINT_FIX=false (126 findings; baseline 128), CC=/usr/bin/cc go vet -tags test_dep ./... (15 diagnostics; equals baseline), go run ./tools/planindex (48 lines; equals baseline)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
