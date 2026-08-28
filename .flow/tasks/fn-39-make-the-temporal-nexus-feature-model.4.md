---
satisfies: [R3, R4, R5, R7]
---
# fn-39-make-the-temporal-nexus-feature-model.4 Mirror and name Operations tests

## Description
Reorganize Operations tests by walkthrough/planning concern and give every current assertion a descriptive declaration name (R3, R4, R5, R7). Preserve the stable aggregate and all checked-in fixtures byte-for-byte.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/OperationsTests.lean`, `model/Temporal/Feature/Nexus/Operations/AsyncStartTests.lean`, `model/Temporal/Feature/Nexus/Operations/CancellationTests.lean`, `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletionTests.lean`, `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean`, `model/Temporal/Feature/Nexus/Fixtures/Operations*.json`
**Touches:** [model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature/Nexus/Operations/*Tests.lean, model/Temporal/Feature/Nexus/Fixtures/Operations*.json]

### Approach
- Move each walkthrough's checking, Property/Behavior separation, Query JSON, deterministic/repeated run, and artifact-shape assertions to its matching test child.
- Move only shared incremental-kernel invariants, compatibility inventory, and cross-walkthrough golden assertions to `PlanningTests`.
- Keep `OperationsTests.lean` as the stable aggregate and preserve `compatibilityConsumers` in its current public namespace.
- Replace anonymous `example` declarations with behavior-oriented names while keeping propositions, proofs, fixture includes, comments, and fixture bytes unchanged.
- Treat any source-path, Query JSON, artifact checksum, or canonical artifact delta as a failure; do not regenerate fixtures to make the refactor pass.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/OperationsTests.lean:9-89` — compatibility/fixtures and Async Start coverage.
- `model/Temporal/Feature/Nexus/OperationsTests.lean:93-161` — Cancellation and Successful Completion coverage.
- `model/Temporal/Feature/Nexus/OperationsTests.lean:163-204` — shared identity and artifact goldens.
- `model/Temporal/Feature/Nexus/Fixtures/OperationsAsyncStartArtifact.json` — facade source provenance in a canonical artifact.
- `.flow/tasks/fn-38-consolidate-layered-model-helpers.7.md:13-37` — predecessor canonical/source compatibility pins.

**Optional** (reference as needed):
- `model/Temporal/Tool/InspectTests.lean` — downstream planned-artifact checks.

### Acceptance
- [ ] Stable OperationsTests import and compatibility inventory remain unchanged.
- [ ] Every existing Operations assertion is present under a descriptive declaration name in the matching test module; operation-specific run coverage stays with that walkthrough.
- [ ] PlanningTests covers only shared/cross-walkthrough planning concerns and does not become a second required file for understanding one operation.
- [ ] All six Query/Artifact fixture includes and exact bytes remain unchanged.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Temporal.Tool.InspectTests TemporalModelTests` passes.

## Acceptance
- [ ] R3, R4, R5, and R7 task-scoped checks pass.
- [ ] No fixture, assertion, or comment is lost.

## Done summary
Mirrored the Nexus Operations production split with named Async Start, Cancellation, Successful Completion, and shared Planning test modules while preserving the stable aggregate, compatibility inventory, all 30 assertion propositions/proofs, both comments, and all six canonical fixture bytes. Exact acceptance, regression, and model-lint gates pass; green receipts were not warrantable only because the two inherited false-symlink stat entries remain dirty and untouched.

stage: impl-review - ran [2026-08-28T22:38:36Z..2026-08-28T22:40:33Z] (SHIP)
## Evidence
- Commits: 6929510ff40be21c5de3b20915782436dab5dd0d
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Temporal.Tool.InspectTests TemporalModelTests failed pre-edit on transient Umpire.Query.olean missing-output race); cd model && mise exec -- lake build Umpire.Query and exact acceptance retry passed, make umpire-check-regression (pre-edit), make lint-model (pre-edit), cd model && mise exec -- lake build Temporal.Feature.Nexus.Operations.AsyncStartTests Temporal.Feature.Nexus.Operations.CancellationTests Temporal.Feature.Nexus.Operations.SuccessfulCompletionTests Temporal.Feature.Nexus.Operations.PlanningTests Temporal.Feature.Nexus.OperationsTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Temporal.Tool.InspectTests TemporalModelTests, make umpire-check-regression, make lint-model, 30 theorem / 10 #check / 6 include_str / 26 native_decide / 4 rfl / 2 comment inventory; Operations fixture SHA-256 values unchanged, gate receipts not warrantable: inherited config/development.yaml false-symlink stat remained dirty
- PRs: