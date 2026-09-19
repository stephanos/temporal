---
satisfies: [R2, R3]
---
# fn-47-generate-umpire-semantic-outcome-and.4 Complete Known Gap lineage and carry coverage

## Description
Complete authored, projected, carried, and test-only Known Gap coverage and stage-preserving composition checks for R2/R3.

**Size:** M
**Files:** `model/Umpire/SemanticInventory/Types.lean`, `model/Umpire/SemanticInventory/KnownGaps.lean`, `model/Umpire/ImplementationLink/Language.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Artifact/Result.lean`, `model/Umpire/Planning/Tests/KnownGaps.lean`, `model/Umpire/SemanticInventory/Tests/KnownGaps.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`
**Touches:** [model/Umpire/SemanticInventory/Types.lean, model/Umpire/SemanticInventory/KnownGaps.lean, model/Umpire/ImplementationLink/Language.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Artifact/Result.lean, model/Umpire/Planning/Tests/KnownGaps.lean, model/Umpire/SemanticInventory/Tests/KnownGaps.lean, model/Temporal/Tool/RunEvaluationTests.lean]

### Approach
- Catalog polymorphic authored ImplementationLinkKnownGap families with their own typed source shape and production/test scope, without pretending they are exact KnownGap values or enumerating arbitrary Source values.
- Model request/raw origin as a non-semantic admitted-input row: it carries arbitrary validated values without claiming a wildcard definition. The Observation-admission projection resolves to that origin and maps only code and subject.toList into EvidenceGap.
- Represent exact Result propagation with separate resolved carry rows for request/raw origin and the synthesized Observation source; neither exact carry resolves to the lossy projection. Validate the explicit lossy projection field map.
- Identify test-only fixtures so they cannot be mistaken for production gaps. A test use that has the same exact code as a production source is a resolved test-only reference, not a duplicate exact declaration.
- Keep the private Planning test fixtures private, but add one-way assertions in their owning test module that each actual fixture resolves to its test-only source row or the intended production-source reference.
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
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Umpire.Observation.Tests Umpire.ImplementationLink.Tests Temporal.Tool.RunEvaluationTests`
## Acceptance
- [ ] Authored Implementation Link families, synthesized sources, non-semantic request/raw admitted input, EvidenceGap projections, exact carries, and test-only rows cover every current gap source/flow without duplication.
- [ ] Separate exact Result carries resolve to the request/raw origin and synthesized Observation source and retain kind/code/subject/detail unchanged; neither resolves to the lossy projection.
- [ ] Every Observation-admission row is explicitly lossy and pins code -> code plus subject.toList -> relatedDefinitionIds, with kind/detail absent rather than claimed preserved.
- [ ] Test fixtures are clearly excluded from production inventory claims; expected scope is authoritative, exact codes are unique across scopes, and a test reuse of a production exact code is represented as a reference.
- [ ] Owner-local assertions bind every private Planning test fixture to its catalog source/reference so fixture drift fails the focused build.
- [ ] Result composition remains stage-separated and byte-identical with no schema field change.
- [ ] Missing lineage/scope/carry target, invalid projection mapping, and duplicate ownership fail focused and Run Evaluation tests.
## Done summary
Completed the 24-row Known Gap catalog with typed Implementation Link families, a non-semantic request/raw admitted-input boundary, a separately lossy Observation projection, distinct exact Result carry origins, and test-only fixtures/references. Added closed validation and owner-local regressions for lineage, scope, exact-code uniqueness, resolved carries, fixture bindings, and byte-identical stage separation; the non-trivial review findings were fixed before same-session SHIP. Memory capture was attempted after the review fix but skipped because Flow memory is not initialized.

Baseline: recovered strict TDD RED in `Umpire.SemanticInventory.Tests.KnownGaps`; final exact Quick and `make lint-model` passed.

stage: impl-review - ran (Codex NEEDS_WORK -> SHIP; session 01a05d27-ea22-7a30-aac0-41fc56b236c2; receipt /tmp/impl-review-receipt-fn-47-generate-umpire-semantic-outcome-and.4.json; 0 open findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 462a8beed399e72365ea3f449842725d0ebd04c1, 62226e0c915d1b8435e4efd2ddb23b5eb9c52325, 3eabd1e7b9d8a582baae31deb13d343604ef6cc5
- Tests: cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Umpire.Observation.Tests Umpire.ImplementationLink.Tests Temporal.Tool.RunEvaluationTests, make lint-model, impl-review Codex SHIP session 01a05d27-ea22-7a30-aac0-41fc56b236c2 receipt /tmp/impl-review-receipt-fn-47-generate-umpire-semantic-outcome-and.4.json
- PRs: