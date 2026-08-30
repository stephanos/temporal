---
satisfies: [R1, R2]
---
# fn-47-generate-umpire-semantic-outcome-and.2 Catalog Observation, Implementation Link, and verdict outcomes

## Description
Extend the owner-local exhaustive catalogs across the semantic evaluation stages for R1/R2.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/ImplementationLink/Application.lean`, `model/Umpire/Observation/Verdict.lean`, `model/Umpire/SemanticInventory/Tests/SemanticStages.lean`
**Touches:** [model/Umpire/Observation/Evaluation.lean, model/Umpire/ImplementationLink/Application.lean, model/Umpire/Observation/Verdict.lean, model/Umpire/SemanticInventory/Tests/SemanticStages.lean]

### Approach
- Add typed ordered catalogs and total proofs beside ObservationStatus, ImplementationLinkStatus, SemanticVerdictStatus, and StrictQueryStatus.
- Add missing public name renderers only where the existing runtime has a private duplicate; migrate callers without changing bytes.
- Represent `not-evaluated` and equivalent optional-stage strings as projection sentinels separate from typed status membership.
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
- [ ] Four semantic-stage families are exhaustive and retain exact rendered names.
- [ ] Optional projection sentinels are explicit but are not constructors or reported as reachable stage outcomes.
- [ ] Private duplicate renderers are removed only when callers can reuse owner APIs byte-for-byte.
- [ ] Cross-family equal names remain distinct and tests reject accidental collapse.
- [ ] Existing comments and Result semantics are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
