---
satisfies: [R1, R2]
---
# fn-47-generate-umpire-semantic-outcome-and.2 Catalog Observation, Implementation Link, and verdict outcomes

## Description
Extend the owner-local exact-one constructor classifiers across the semantic evaluation stages for R1/R2.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/ImplementationLink/Application.lean`, `model/Umpire/Observation/Verdict.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Umpire/SemanticInventory/Tests/SemanticStages.lean`
**Touches:** [model/Umpire/Observation/Evaluation.lean, model/Umpire/ImplementationLink/Application.lean, model/Umpire/Observation/Verdict.lean, model/Temporal/Tool/RunEvaluation.lean, model/Umpire/SemanticInventory/Tests/SemanticStages.lean]

### Approach
- Add typed ordered constructor descriptors and exact-one classifiers beside ObservationStatus, ImplementationLinkStatus, SemanticVerdictStatus, and StrictQueryStatus.
- Add missing public name renderers only where the existing runtime has a private duplicate; migrate callers without changing bytes.
- Represent `not-evaluated` and equivalent optional-stage strings as projection sentinels separate from typed status classification.
- Pin identical words in different families as distinct rows rather than normalizing them.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:92-125` — Observation status/diagnostics.
- `model/Umpire/ImplementationLink/Application.lean:13-28` — Implementation Link status/name.
- `model/Umpire/Observation/Verdict.lean:11-17,62-80,93-97` — Property and strict Query distinctions.
- `model/Temporal/Tool/RunEvaluation.lean:102-114,876-919` — private names and optional-stage projections after fn-44.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.SemanticStages`
## Acceptance
- [ ] Four semantic-stage families have exact-one constructor classifiers and retain exact rendered names.
- [ ] Optional projection sentinels are explicit but are not constructors or reported as reachable stage outcomes.
- [ ] Private duplicate renderers are removed only when callers can reuse owner APIs byte-for-byte.
- [ ] Cross-family equal names remain distinct and tests reject accidental collapse.
- [ ] Existing comments and Result semantics are preserved.
## Done summary
Added owner-local ordered constructor catalogs and exact-one proofs for Observation, Implementation Link, semantic Property, and strict Query outcomes. Run Evaluation now reuses the byte-identical owner renderers, while optional Implementation Link absence is a separate typed projection sentinel and cross-family equal names remain distinct.

Baseline: staged red for future semantic-inventory executable/test and Make targets; `make lint-model` was green at 201/201.

stage: impl-review - ran [SHIP at 2026-09-01T12:16:23Z; session 01a05ce2-8279-7451-bc72-0daae4278e76; 0 open findings]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 25acfaa1a85dc36541fccd2c627aca64bd4d03ca, 85b4874c35a24271d021e68b9342e975586009a6
- Tests: baseline: staged red (future semantic-inventory executable/test and Make targets absent); make lint-model green 201/201, TDD RED: cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.SemanticStages (missing owner catalog/proof and projection-sentinel APIs), cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.SemanticStages, cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, make lint-model
- PRs: