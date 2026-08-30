---
satisfies: [R2, R3]
---
# fn-47-generate-umpire-semantic-outcome-and.4 Complete Known Gap lineage and carry coverage

## Description
Complete authored, projected, carried, and test-only Known Gap coverage and stage-preserving composition checks for R2/R3.

**Size:** M
**Files:** `model/Umpire/SemanticInventory/KnownGaps.lean`, `model/Umpire/ImplementationLink/Language.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Artifact/Result.lean`, `model/Umpire/SemanticInventory/Tests/KnownGaps.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`
**Touches:** [model/Umpire/SemanticInventory/KnownGaps.lean, model/Umpire/ImplementationLink/Language.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Artifact/Result.lean, model/Umpire/SemanticInventory/Tests/KnownGaps.lean, model/Temporal/Tool/RunEvaluationTests.lean]

### Approach
- Catalog polymorphic authored ImplementationLinkKnownGap families with their own typed source shape and production/test scope, without pretending they are exact KnownGap values or enumerating arbitrary Source values.
- Model the request/raw path as two distinct rows: the Observation-admission projection maps only code and subject.toList into EvidenceGap, while Result aggregation carries the exact KnownGap kind/code/subject/detail.
- Represent exact carries as resolved references, validate the explicit lossy projection field map, and identify test-only fixtures so they cannot be mistaken for production gaps.
- Assert ResultArtifact keeps operational, Observation, Implementation Link, Property, Query, cleanup, and gaps as separate fields with unchanged bytes.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ImplementationLink/Language.lean:19-24` — polymorphic authored ImplementationLinkKnownGap declarations.
- `model/Umpire/ImplementationLink/Application.lean:360-376,420-560` — authored gap consumption.
- `model/Umpire/Observation/Evaluation.lean:74-87,968-972` — EvidenceGap shape and admission.
- `model/Temporal/Tool/RunEvaluation.lean:768-800,984-991` — lossy EvidenceGap projection versus exact Result union after fn-44.
- `model/Umpire/Artifact/Result.lean:241-265` — separate result-stage fields.
- `model/Umpire/Planning/Tests/KnownGaps.lean` — test-only fixture coverage.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Umpire.Observation.Tests Umpire.ImplementationLink.Tests Temporal.Tool.RunEvaluationTests`
## Acceptance
- [ ] Authored Implementation Link families, synthesized sources, EvidenceGap projections, exact carries, and test-only rows cover every current gap source/flow without duplication.
- [ ] Every exact Result carry resolves to a declared source and retains kind/code/subject/detail unchanged.
- [ ] Every Observation-admission row is explicitly lossy and pins code -> code plus subject.toList -> relatedDefinitionIds, with kind/detail absent rather than claimed preserved.
- [ ] Test fixtures are clearly excluded from production inventory claims.
- [ ] Result composition remains stage-separated and byte-identical with no schema field change.
- [ ] Missing lineage/scope/carry target, invalid projection mapping, and duplicate ownership fail focused and Run Evaluation tests.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
