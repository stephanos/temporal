---
satisfies: [R1]
---
# fn-80-close-the-model-to-case-seam-and-harden.15 Replace the Nexus3 clause-for-clause equality gate

## Description
R1's separable product defect, extracted from blocked task .4. Task .4's handover states explicitly that this is worth closing on its own and should be re-planned rather than silently substituted into .4's acceptance — this task is that re-plan.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Testpilot.lean
**Touches:** [model/Temporal/Feature/Nexus3/Testpilot.lean]

### Why this task exists
The spec's Goal section names this as defect 1: `produceCompletionCase` takes the checked values, throws on any scoped clause, then compares each input to a fixed expected model. A model edit therefore produces a **lowering error instead of a different Case**, which is the single behaviour most at odds with the vision — the model is supposed to be the source of truth.

This half needs no `ScopedEvidence` evidence path, so it does not wait on the projection task or on .4.

### Scope
Replace the clause-shape match and the hand-written success rule with lowering driven by the checked model. Of the protections the equality gate provided, preserve: witness absence stays a `LoweringError`, and an unexpressible clause stays a `LoweringError` naming the clause. A changed Target, Behavior, Query or witness must become **different Case bytes rather than an error**.

Coordinate with the ordered-list task above if both are in flight — both touch `Testpilot.lean`'s `supportsSuccessProperty`.

## Acceptance
- [ ] A changed Target, Behavior, Query or witness produces different Case bytes, not a lowering error.
- [ ] Witness absence still produces a `LoweringError`; an unexpressible clause still produces a `LoweringError` that names the clause.
- [ ] The hand-written success rule and the clause-shape equality match are gone.
- [ ] Regenerated fixture bytes are accounted for through owning targets; the async-nexus Case still satisfies live, or the reason it cannot is recorded rather than worked around.
- [ ] `make lint-model` adds nothing to the 169 baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
