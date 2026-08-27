---
satisfies: [R2, R5, R6]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.5 Emit v2-only Lean artifacts with exact checksums

## Description
Implement the Lean half of R5 and every checked-in Lean fixture affected by R2/R5/R6. Replace the identity/digest wire contract with canonical v2 DrivePlan and ExperimentSpec Artifacts using the approved names and exact Artifact Checksum semantics.

**Size:** L
**Files:** `model/Umpire/Artifact.lean`, `model/Temporal/Tool/Inspect.lean`, inspector tests, and every current Switch/Nexus Artifact fixture and producer
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Examples/Switch*.lean, model/Umpire/Examples/Fixtures/SwitchCompiledArtifact.json, model/Umpire/Examples/testdata/switch-experiment-spec.json, model/Temporal/Tool/Inspect*.lean, model/Temporal/Feature/Nexus/Operations*.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure*.lean, model/Temporal/Feature/Nexus/Fixtures/OperationsAsyncStartArtifact.json, model/Temporal/Feature/Nexus/Fixtures/OperationsCancellationArtifact.json, model/Temporal/Feature/Nexus/Fixtures/OperationsSuccessfulCompletionArtifact.json, model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json]

### Approach
- Rename Artifact record fields by meaning, including Definition ID references, Behavior Fingerprints, Model preconditions, expanded Limits, exact Known Gap rows, and Artifact Checksums.
- Emit only `umpire-drive-plan/v2` and `umpire-experiment/v2`; remove v1 constants, keys, expectations, and compatibility comments.
- Define canonical bytes as the canonical Lean JSON object followed by exactly one LF. Use a fixed field order, no insignificant whitespace, canonical string escaping, and canonical base-10 natural numbers.
- Compute each Artifact Checksum over its complete canonical object with only its own checksum field absent; ExperimentSpec includes the complete nested DrivePlan representation and its checksum.
- Keep deterministic field/set ordering and current modeled trace content; Source Locations and complete Known Gap rows participate in the Artifact Checksum.
- Regenerate all six Artifact goldens from their existing authoritative Lean producers: `SwitchCompiledArtifact.json`, `switch-experiment-spec.json`, three `Operations*Artifact.json` files, and `nexus-caller-closure-experiment-spec.json`.
- Add one-at-a-time mutation tests for every checksum-bearing content category and domain separation between the two Artifact kinds.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact.lean:30-110,185-245,285-355` — current structures, JSON boundary, lowering, and identity derivation.
- `model/Temporal/Tool/Inspect.lean` and `InspectTests.lean` — authoritative fixture emitter and contract tests.
- `model/Umpire/Examples/SwitchTests.lean` — Switch compiled and ExperimentSpec fixture producers.
- `model/Temporal/Feature/Nexus/OperationsTests.lean` — three Nexus Operations Artifact producers.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean` — Nexus caller-closure Artifact producer.
- All six JSON output paths listed in **Touches** — complete golden set that must be replaced, never hand-edited.

### Key context
There is no Lean v1 reader today and this task must not add one. The checksum covers provenance and the complete nested plan because it identifies exact Artifact content, unlike a Behavior Fingerprint.
## Acceptance
- [ ] Lean emits only the two exact v2 format versions, replacement JSON keys, and canonical bytes ending in exactly one LF.
- [ ] DrivePlan and ExperimentSpec checksums are reproducible, domain-separated, and cover all canonical content except their own checksum field.
- [ ] `SwitchCompiledArtifact.json`, `switch-experiment-spec.json`, all three `Operations*Artifact.json` files, and `nexus-caller-closure-experiment-spec.json` are regenerated from authoritative Lean producers and retain the same selected Model Traces and Properties.
- [ ] Mutation tests prove every content category—including complete Known Gap rows, provenance, nested plan content, and format version—changes the owning checksum.
- [ ] No v1 constant, serializer branch, reader, or migration exists in Lean.
## Done summary
Replaced the Lean Artifact boundary with v2-only DrivePlan and ExperimentSpec records, context-qualified Definition ID fields, exact Known Gap rows, typed domain-separated Artifact Checksums, and canonical byte encoders that append exactly one LF. All six Switch/Nexus goldens were regenerated from the pinned Lean inspector producers while retaining the selected Model Traces and Properties.

Baseline: Lean, pinned Go, and the current regression target were green before editing; the future `umpire-check-regression-views` target was absent as the declared fn37.6 sequencing boundary. Verification: the checksum/category matrix and five-producer inspector surface passed (27 jobs), and the full Lean Quick passed (133 jobs). Pinned Go failed only in `TestProductionFixtureProjectsCanonicalMetadata`, `TestWorkflowNexusQueryExactActionCallerClosure`, and `TestRequireProjectionIsIndependentOfWorkingDirectory` because the task-.6-owned v1 Projection consumer rejects `umpire-experiment/v2`; `umpire-check-regression` reached the same stable unsupported-format classification. No Go or Generated View consumer change belongs to this Lean-half task.

Codex review: first-pass SHIP with no findings, session `01a04440-15c2-75a1-a23c-bd7131150c0d`.

stage: impl-review - ran [2026-08-27T17:24:05Z..2026-08-27T17:27:17Z] (SHIP) (model: gpt-5.6-sol)
## Evidence
- Commits: 99f894d7555acee77b5ef8230849755712329e64
- Tests: baseline GREEN: cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect (131 jobs), baseline GREEN: mise exec -- go test ./tools/umpire/..., baseline RED inherited sequencing gap: mise exec -- make umpire-check-regression-views (target is introduced by fn37.6), baseline GREEN: mise exec -- make umpire-check-regression, RED: cd model && mise exec -- lake build Umpire.Examples.SwitchTests Temporal.Feature.Nexus.Experimental.CallerClosureTests (v2 format assertions rejected v1 producers), mise exec -- lake build Umpire.Planning.Tests.Artifacts Umpire.Examples.SwitchTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests Temporal.Tool.InspectTests (27 jobs), authoritative regeneration: mise exec -- lake exe temporal-model-inspect <scenario> for Switch x2, Operations x3, CallerClosure x1; all six end in exactly one LF, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect (133 jobs), sequencing RED owned by fn37.6: mise exec -- go test ./tools/umpire/... (three v1 Projection consumer tests reject umpire-experiment/v2), sequencing RED owned by fn37.6: mise exec -- make umpire-check-regression (v1 Projection extractor rejects umpire-experiment/v2), Codex impl-review SHIP: session 01a04440-15c2-75a1-a23c-bd7131150c0d
- PRs: