---
satisfies: [R1, R2, R4]
---
# fn-36-nexus-lifecycle-cleanup.1 Make start cancel and completion the core Nexus surface

## Description
Replace the Examples modules with the root Lifecycle and Operations modules (R1, R2, R4). This task owns the core semantic seam and removes the dependency from ordinary Nexus behavior to AutoClose.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/**`, `model/Temporal/Feature/Nexus/Lifecycle.lean`, `model/Temporal/Feature/Nexus/LifecycleTests.lean`, `model/Temporal/Feature/Nexus/Operations.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`, `model/Temporal/Feature.lean`, `model/Temporal.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Examples/**, model/Temporal/Feature/Nexus/Lifecycle.lean, model/Temporal/Feature/Nexus/LifecycleTests.lean, model/Temporal/Feature/Nexus/Operations.lean, model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature.lean, model/Temporal.lean, model/TemporalModelTests.lean]

### Approach
- Move the checked-target machinery from BasicLifecycle to Lifecycle, replace AutoClose types/step with the exact four-state, three-event focused relation, and update finite domains and semantic contract digests.
- Preserve stable start/completion declaration identities and existing comments; add canceled state/action/outcome/observation values and proofs.
- Move BasicOperations to Operations, retain AsyncStart and SuccessfulCompletion, and add a Cancellation walkthrough with the same Property/Behavior/Query/planning shape.
- Move and expand focused tests, delete Examples completely, and make the ordinary Feature/Temporal/test facades expose only the new core modules.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:1-371` — checked target and finite planning seam to retain.
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean:1-294` — two walkthrough patterns to migrate and extend.
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean:1-91` — target/error regression style.
- `model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean:1-67` — positive/negative artifact assertions.
- `model/Temporal/Feature/Nexus/AutoClose.lean:181-446` — semantic reference only; do not import or move it in this task.

### Key context
- `cancel` means accepted cancellation and resulting state `canceled`; `succeed` remains handler-reported completion.
- Every unsupported focused state/event pair must return no transition.

### Acceptance
- [ ] Lifecycle compiles independently from Experimental/AutoClose and exposes exactly the specified transition surface.
- [ ] Operations exposes deterministic start, cancellation, and completion walkthroughs with positive and negative checks.
- [ ] Examples is deleted with no import-only compatibility facade.
- [ ] Focused core and aggregate Lean targets pass.

## Acceptance
- [ ] R1 focused lifecycle and negative transitions are covered.
- [ ] R2 has three checked/planned operation paths.
- [ ] R4 ordinary facades contain no experimental import.
- [ ] Existing comments in moved/refactored code are preserved.

## Done summary
Implemented focused Nexus Lifecycle and Operations root modules for start, cancel, and successful completion; added focused tests; updated core facades; removed Nexus/Examples. Preserved lifecycle comments and declaration identities where applicable. stage: plan-sync - skipped(config: planSync.enabled != true). No commit was created per repository instructions.
## Evidence
- Commits:
- Tests: cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests TemporalModelTests Temporal
- PRs: