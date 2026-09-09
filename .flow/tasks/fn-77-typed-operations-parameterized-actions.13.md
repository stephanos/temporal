---
satisfies: [R8]
---
# fn-77-typed-operations-parameterized-actions.13 Derive the unary example's Contract read path

## Description
Derive the unary example's Contract read path from the Property instead of hand-writing it.

**Size:** S
**Files:** model/Temporal/Feature/Nexus3/TypedUnary.lean; model/Temporal/Feature/Nexus3/Tests/TypedUnary.lean
**Touches:** [model/Temporal/Feature/Nexus3/TypedUnary.lean, model/Temporal/Feature/Nexus3/Tests/TypedUnary.lean]

### Why this task exists
Raised as a P2 (confidence 100, classification introduced, R8) by this spec's completion review,
which shipped with R8 recorded as *partial* for exactly this reason.

`model/Temporal/Feature/Nexus3/TypedUnary.lean:434` hand-writes `recordedTypePath` — the oneof
selector plus `workflow_type` plus `name` — beside the Property, and `startedRule` compares it
against a literal. Editing the Property's `startedTypePath` does not move the Contract's read, and
nothing checks that the two agree. That is a duplicate assertion that can silently diverge.

The sibling example already solves this: `TypedNexus.readPathOf` applies
`Umpire.Case.Observed.pathOf` to the Property's own `PropertyFieldPath`. So the two qualifying
examples currently disagree on the R8 mechanism, which is the inconsistency worth removing.

The literal on the REQUEST side is defensible and should stay — the coverage map binds it to the
exact request assignment. Only the response-side path is duplicated.

### Approach
- Derive `recordedTypePath` as `Umpire.Case.Observed.pathOf startedTypePath historyEventNode`,
  mirroring `TypedNexus.readPathOf`.
- Pin the derived segments in `model/Temporal/Feature/Nexus3/Tests/TypedUnary.lean` the way
  `Tests/TypedNexus.lean:259-273` pins them, so a future divergence fails a test rather than
  passing silently.
- Regenerate any fixture bytes the change moves through the owning targets; never hand-edit them.

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Case.CompilerTests`
`mise exec -- make umpire-build-model`
`mise exec -- make umpire-check-case-runtime-conformance`

## Acceptance
- [ ] `recordedTypePath` in `TypedUnary.lean` is derived from the Property's own `startedTypePath` through `Umpire.Case.Observed.pathOf`, not written out beside it; the two qualifying examples now use the same R8 mechanism.
- [ ] Editing `startedTypePath` moves the Contract's read: a test in `Tests/TypedUnary.lean` pins the derived segments so a divergence fails rather than passing silently, mirroring `Tests/TypedNexus.lean:259-273`.
- [ ] The request-side literal is left in place, with its coverage-map binding to the exact request assignment intact.
- [ ] Fixture bytes moved by the change are regenerated through the owning targets and accounted for; `make umpire-check-case-runtime-conformance` is green.
- [ ] `make lint-model` adds nothing to the 169-finding generated-API baseline (`Umpire.Lint` stays clean), and no new axiom dependency appears.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
