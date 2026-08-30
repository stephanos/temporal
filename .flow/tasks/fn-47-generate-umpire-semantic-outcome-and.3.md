---
satisfies: [R3]
---
# fn-47-generate-umpire-semantic-outcome-and.3 Catalog fixed and synthesized Known Gap sources

## Description
Create the production Known Gap source catalog and reuse named fixed/synthesized declarations for R3.

**Size:** M
**Files:** `model/Umpire/SemanticInventory/KnownGaps.lean`, `model/Umpire/Artifact/Types.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Umpire/SemanticInventory/Tests/KnownGaps.lean`
**Touches:** [model/Umpire/SemanticInventory/KnownGaps.lean, model/Umpire/Artifact/Types.lean, model/Temporal/Tool/RunEvaluation.lean, model/Umpire/SemanticInventory/Tests/KnownGaps.lean]

### Approach
- Name and catalog the eight canonical planner gaps without changing their values/order or KnownGapKind.
- Catalog runtime/Observation synthesized families using typed exact/prefix shapes and make the producer reuse the catalog-owned declaration after fn-44.
- Validate stable catalog IDs, namespaced exact codes/prefixes, lineage, scope, and canonical order.
- Keep request-provided unknown codes out of synthesized definitions; their later flow is carried, not invented.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Types.lean:7-33,86-113` — KnownGap type, validation, and bytes.
- `model/Umpire/Artifact/Types.lean:130-139` — eight canonical planner gaps.
- `model/Temporal/Tool/RunEvaluation.lean:696-727,751-762,913-960` — input, synthesis, and Result flow after fn-44.
- `.flow/tasks/fn-44-seal-observation-traces-and-centralize.5.md` — dependency-owned Run Evaluation migration.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Temporal.Tool.RunEvaluationTests`
## Acceptance
- [ ] All fixed planner gaps appear exactly once with unchanged KnownGap values/order.
- [ ] Each synthesized production family has one typed namespaced source descriptor reused by its producer.
- [ ] Duplicate IDs/codes, invalid namespaces/prefixes, wrong lineage/scope, and noncanonical order fail atomically.
- [ ] Request/raw unknown codes are not turned into wildcard semantic definitions.
- [ ] Existing Run Evaluation behavior, protocol, artifacts, canonical bytes, and comments remain unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
