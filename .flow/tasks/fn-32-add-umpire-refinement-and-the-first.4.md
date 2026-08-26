---
satisfies: [R3, R4, R5]
---
# fn-32-add-umpire-refinement-and-the-first.4 Compose qualified System traces through the Nexus refinement

## Description
Prove the conformance-facing System-trace to Feature-Property handoff and layer-specific outcomes for R3–R5.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Umpire/Refinement/Tests/**`, `model/Temporal/RefinementTests/Nexus.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Umpire/Refinement/Tests/**, model/Temporal/RefinementTests/Nexus.lean]

### Approach
- Consume an already-qualified System semantic trace plus explicit source setup, and re-admit its initial state and every step through the checked System kernel before translation.
- Apply the checked refinement before invoking the unchanged Feature Property evaluator.
- Build independent Observation, source-admission, Refinement, and Property mutations under the exact non-base-System composed-test root `Temporal.RefinementTests.Nexus`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Qualification.lean` — qualified trace boundary
- `model/Umpire/Property/Language.lean:1162-1228` — pure evaluator
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — current property fixtures

### Acceptance
- [ ] Only source-kernel-admitted qualified System traces reach Feature properties through checked Refinement.
- [ ] Observation, Refinement, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.
## Acceptance
- [ ] Only source-kernel-admitted qualified System traces reach Feature properties through checked Refinement.
- [ ] Observation, Refinement, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.
### Acceptance
- [ ] R3–R5 positive, source target/digest/setup/transition, and independent boundary mutation matrices pass.
- [ ] A refinement failure never becomes unknown evidence or a property violation.
- [ ] Feature evaluation remains unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
