---
satisfies: [R2, R3]
---
# fn-47-generate-umpire-semantic-outcome-and.4 Complete Known Gap lineage and carry coverage

## Description
Complete authored, carried, and test-only Known Gap coverage and stage-preserving composition checks for R2/R3.

**Size:** M
**Files:** `model/Umpire/SemanticInventory/KnownGaps.lean`, `model/Umpire/ImplementationLink/Language.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Artifact/Result.lean`, `model/Umpire/SemanticInventory/Tests/KnownGaps.lean`
**Touches:** [model/Umpire/SemanticInventory/KnownGaps.lean, model/Umpire/ImplementationLink/Language.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Artifact/Result.lean, model/Umpire/SemanticInventory/Tests/KnownGaps.lean]

### Approach
- Catalog authored Implementation Link and evidence inputs by owner and production/test scope without enumerating runtime values that are not global constants.
- Represent request/raw/artifact propagation as resolved carried-from references to admitted source families.
- Identify test-only fixtures explicitly so they cannot be mistaken for production gaps.
- Assert ResultArtifact keeps operational, Observation, Implementation Link, Property, Query, cleanup, and gaps as separate fields with unchanged bytes.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ImplementationLink/Language.lean` — authored Known Gap declarations.
- `model/Umpire/ImplementationLink/Application.lean:360-376,420-560` — authored gap consumption.
- `model/Umpire/Observation/Evaluation.lean:80-90,968-972` — evidence gap admission.
- `model/Umpire/Artifact/Result.lean:241-265` — separate result-stage fields.
- `model/Umpire/Planning/Tests/KnownGaps.lean` — test-only fixture coverage.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Umpire.Observation.Tests Umpire.ImplementationLink.Tests`

## Acceptance
- [ ] Authored, synthesized, carried, and test-only rows cover every current gap source/flow without duplication.
- [ ] Every carried row resolves to a declared source and retains kind/code/subject/detail unchanged.
- [ ] Test fixtures are clearly excluded from production inventory claims.
- [ ] Result composition remains stage-separated and byte-identical with no schema field change.
- [ ] Missing lineage/scope/carry target and duplicate ownership fail focused tests.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
