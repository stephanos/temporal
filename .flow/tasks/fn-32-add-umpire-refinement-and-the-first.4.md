---
satisfies: [R3, R4, R5]
---
# fn-32-add-umpire-refinement-and-the-first.4 Compose qualified System traces through the Nexus refinement

## Description
Prove the conformance-facing System-trace to Feature-Property handoff and layer-specific outcomes for R3–R5.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Umpire/Refinement/Tests/**`, `model/Temporal/System/Nexus/RefinementTests.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Umpire/Refinement/Tests/**]

### Approach
- Consume only an already-qualified System semantic trace.
- Apply the checked refinement before invoking the unchanged Feature Property evaluator.
- Build independent Observation, Refinement, and Property mutations.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Qualification.lean` — qualified trace boundary
- `model/Umpire/Property/Language.lean:1162-1228` — pure evaluator
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — current property fixtures

### Acceptance
- [ ] Qualified System traces reach Feature properties only through checked Refinement.
- [ ] Observation, Refinement, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.

## Acceptance
- [ ] R3–R5 positive and independent mutation matrices pass.
- [ ] A refinement failure never becomes unknown evidence or a property violation.
- [ ] Feature evaluation remains unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
