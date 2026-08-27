---
satisfies: [R3, R4, R5]
---

# fn-32-add-umpire-refinement-and-the-first.4 Compose accepted System Model Traces through the Nexus Implementation Link

## Description
Prove the Run Evaluation-facing System-trace to Feature-Property handoff and layer-specific outcomes for R3–R5.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Umpire/ImplementationLink/Tests/**`, `model/Temporal/ImplementationLinkTests/Nexus.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Umpire/ImplementationLink/Tests/**, model/Temporal/ImplementationLinkTests/Nexus.lean]

### Approach
- Consume an already-accepted System semantic trace plus explicit source setup, and re-admit its initial state and every step through the checked System kernel before translation.
- Apply the checked Implementation Link before invoking the unchanged Feature Property evaluator.
- Build independent Observation, source-admission, Implementation Link, and Property mutations under the exact non-base-System composed-test root `Temporal.ImplementationLinkTests.Nexus`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean` — Evidence-backed Model Trace boundary
- `model/Umpire/Property/Language.lean:1162-1228` — pure evaluator
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — current property fixtures

### Acceptance
- [ ] Only source-kernel-admitted accepted System Model Traces reach Feature properties through checked Implementation Link.
- [ ] Observation, Implementation Link, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.
## Acceptance
- [ ] Only source-kernel-admitted accepted System Model Traces reach Feature properties through checked Implementation Link.
- [ ] Observation, Implementation Link, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.
### Acceptance
- [ ] R3–R5 positive, source target/Behavior Fingerprint/setup/transition, and independent boundary mutation matrices pass.
- [ ] An Implementation Link failure never becomes unknown evidence or a property violation.
- [ ] Feature evaluation remains unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
