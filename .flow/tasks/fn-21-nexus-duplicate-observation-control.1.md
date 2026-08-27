---
satisfies: [R1, R7]
---
# fn-21-nexus-duplicate-observation-control.1 Author the exact caller-closure negative-control space

## Description
Author the Temporal-owned one-axis/two-choice space and compile its selected fault point through the existing fn-16 path for R1/R7. Keep the current caller-closure target, Property, ordinary query, and artifact untouched.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/CallerClosureFault.lean`, `model/Temporal/Feature/Nexus/CallerClosureFaultTests.lean`, `model/Temporal/Feature/Nexus.lean`, `model/Temporal/Feature.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/CallerClosureFault.lean, model/Temporal/Feature/Nexus/CallerClosureFaultTests.lean, model/Temporal/Feature/Nexus.lean, model/Temporal/Feature.lean, model/TemporalModelTests.lean]

### Approach
- Compose the spec's exact space/axis/choice/fault/goal identities over the existing checked exact-action caller-closure Query and required force-close occurrence.
- Reuse fn-16 `CheckedExperimentSpace`, `lowerSpacePoint`, checked `ArtifactIntent`, and target-indexed kernel; do not construct requested-fault artifact fields directly.
- Pin the two assignments, checked metadata/digest, selected intent arrays/capability union, derived identities, and faulted ExperimentSpec bytes.
- Prove the fault choice still receives the ordinary count-one Model Trace and that the pre-existing no-fault ExperimentSpec bytes and pure Property are unchanged.
- Add reorder plus stale occurrence/action/capability, duplicate effect, invalid goal, and outcome-authoring negative fixtures; preserve existing comments and vertical imports.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-16-authored-variation-spaces-and.md:42-100` — checked space/fault/lowering contract
- `.flow/tasks/fn-16-authored-variation-spaces-and.5.md:13-34` — Temporal variation-space proof pattern
- `model/Temporal/Feature/Nexus/CallerClosure.lean:34-48` — canonical Behavior Fingerprints
- `model/Temporal/Feature/Nexus/CallerClosure.lean:441-507` — Property, Query, and occurrence
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean:232-243` — exact-identity assertion style

### Acceptance
- [ ] Exactly two canonical points compile and only the fault point carries the exact requested fault/capability intent.
- [ ] Both points use target-owned count-one output; no authored outcome/evidence/receipt enters the space.
- [ ] The existing no-fault artifact and Property remain byte/definition unchanged.
- [ ] Every R1 negative case fails at the checked-space or lowering boundary with no partial artifact.
- [ ] Focused and aggregate Lean tests pass with reusable Umpire imports remaining Temporal-free.

## Acceptance
- [ ] R1 exact space and ordinary fault-bearing ExperimentSpec are checked and deterministic.
- [ ] R7 package purity, comments, and no-general-fault boundaries hold.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
