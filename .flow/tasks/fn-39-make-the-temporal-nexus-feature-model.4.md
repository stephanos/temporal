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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
