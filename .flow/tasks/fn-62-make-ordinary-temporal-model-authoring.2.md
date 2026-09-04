---
satisfies: [R3, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.2 Hide finite planner representation transport

## Description
Satisfy R3 and R8 by moving finite incremental-kernel derivation behind an existing checked Target/Query boundary and deleting Temporal-authored dependent equality plumbing.

**Size:** M
**Files:** `model/Umpire/Planning/Engine.lean`, `model/Umpire/Planning/VisibilityTests.lean`, `model/Umpire/Planning/Tests/Enumeration.lean`, `model/Temporal/Feature/Nexus/Operations/Planning.lean`, `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean`
**Touches:** [model/Umpire/Planning/Engine.lean, model/Umpire/Planning/VisibilityTests.lean, model/Umpire/Planning/Tests/Enumeration.lean, model/Temporal/Feature/Nexus/Operations/Planning.lean, model/Temporal/Feature/Nexus/Operations/PlanningTests.lean]

### Approach
- Start from `IncrementalPlannerKernel.ofCheckedQuery?` and `ofFinite` in `model/Umpire/Planning/Engine.lean:79-160`; reuse an existing seam if it can express the ordinary Nexus flow.
- Add only the narrowest proof-carrying adapter needed when the target-dependent return type prevents reuse; lack of finite completeness must remain explicit.
- Replace the representation-specific `simp` and equality reconstruction in `model/Temporal/Feature/Nexus/Operations/Planning.lean:15-43` with the public adapter.
- Keep indexed order, Limits, soundness, deterministic selection, and Model Outcomes unchanged.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Engine.lean:35-160` — kernel contract and existing finite/query derivation.
- `model/Temporal/Feature/Nexus/Operations/Planning.lean:15-43` — dependent proof burden to encapsulate.
- `model/Umpire/Planning/VisibilityTests.lean` — public API boundary.
- `model/Umpire/Planning/Tests/Enumeration.lean` — enumeration/soundness regressions.
- `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean` — Temporal plan identity checks.

**Optional** (reference as needed):
- `model/Umpire/ARCHITECTURE.md:578-610` — intended incremental-planner and Known Gap seams.

### Acceptance
- [ ] Ordinary Nexus planning contains no author-written `Eq.mpr`, target-internal unfolding, or representation-specific `simp` proof to obtain its kernel.
- [ ] Missing finite completeness and Target mismatches fail explicitly with no inferred/partial fallback.
- [ ] Kernel enumeration, Limits, soundness, selected traces, runs, and deterministic output remain exact.
- [ ] The adapter is public only through the established Planning facade and has generic negative tests.
- [ ] Focused Planning and Nexus planning targets pass.

## Acceptance
- [ ] R3 and R8 are satisfied without widening planner authority.
- [ ] `cd model && mise exec -- lake build Umpire.Planning.Tests Umpire.Planning.VisibilityTests Temporal.Feature.Nexus.Operations.PlanningTests` passes.
- [ ] Existing deterministic plan fixtures remain byte-identical unless an intentional delta is documented by the parent spec.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
