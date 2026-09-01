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
Named the eight canonical planner Known Gaps and added a validated production catalog for exact and closed synthesized Observation sources. Run Evaluation now reuses the catalog-owned Observation declaration, while raw request gaps remain carried and all artifact/protocol behavior stays unchanged.

The initial review found wildcard coverage and duplicated family-kind authority; both were fixed with a closed suffix set and descriptor-derived materialization before same-session SHIP. Memory capture was attempted after the non-trivial review fix but skipped because Flow memory is not initialized.

stage: impl-review - ran (Codex NEEDS_WORK -> SHIP; session 01a05cf9-454d-7d80-ac47-af5a859de2af; receipt /tmp/impl-review-receipt-fn-47-generate-umpire-semantic-outcome-and.3.json)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b77eb4bbe69d474e548c07dbc5dd609aa3fbb47c, abe914145b9309d04133ebf366d46865a2e6ed59
- Tests: BASELINE_RED_EXPECTED: cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Temporal.Tool.RunEvaluationTests (missing KnownGaps target; existing RunEvaluationTests built green), RED_EXPECTED: cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps (missing catalog module before implementation), RED_EXPECTED: cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps (raw interpretation code umpire.observation.raw-unknown was incorrectly covered before review fix), cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Temporal.Tool.RunEvaluationTests, make lint-model, git diff --check, impl-review Codex SHIP session 01a05cf9-454d-7d80-ac47-af5a859de2af receipt /tmp/impl-review-receipt-fn-47-generate-umpire-semantic-outcome-and.3.json
- PRs: