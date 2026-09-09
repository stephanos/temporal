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
- [x] `recordedTypePath` in `TypedUnary.lean` is derived from the Property's own `startedTypePath` through `Umpire.Case.Observed.pathOf`, not written out beside it; the two qualifying examples now use the same R8 mechanism.
- [x] Editing `startedTypePath` moves the Contract's read: a test in `Tests/TypedUnary.lean` pins the derived segments so a divergence fails rather than passing silently, mirroring `Tests/TypedNexus.lean:259-273`.
- [x] The request-side literal is left in place, with its coverage-map binding to the exact request assignment intact.
- [x] Fixture bytes moved by the change are regenerated through the owning targets and accounted for; `make umpire-check-case-runtime-conformance` is green.
- [x] `make lint-model` adds nothing to the 169-finding generated-API baseline (`Umpire.Lint` stays clean), and no new axiom dependency appears.


## Done summary
The typed unary example no longer hand-writes its Contract read path. `startedRule` derives it with
`readPathOf startedTypePath` — `Umpire.Case.Observed.pathOf` rooted at `historyEventNode`, the same
helper and the same shape `TypedNexus.readPathOf` uses — so a Property edit that moves the recorded
coordinate moves the Contract's read with it. The rule became `Except String`-valued and
`typedUnaryCase` maps its rejection to a `LoweringError` at the clause id, mirroring
`TypedNexus.operationRule`. The two qualifying examples now use one R8 mechanism.

`Tests/TypedUnary.lean` gains the sibling's pinning block: the exact derived segments of
`startedTypePath`, two edited-coordinate guards (a shortened read loses its last segment; a sibling
attribute derives `parent_workflow_namespace`), and three presence paths that derive no read path at
all. The pins were confirmed red-first — mutating the expected literal fails exactly that `#guard`,
and mutating the Property's own attribute field number fails it too.

The derivation reproduces the deleted literal byte-for-byte, so no fixture moved:
`make umpire-check-case-runtime-conformance` is green with `typed-unary-case.json` unchanged, and no
fixture was hand-edited. The request-side literal (`submittedTypeTarget`) and its coverage-map
binding are untouched, as the task required. No new axiom dependency: the `#print axioms` pins in
`Tests/TypedUnary.lean` are unchanged and green.

`make lint-model` stays at its inherited 169 generated-API findings with `Umpire.Lint` clean and no
Nexus3 finding; that gate's red is the pre-existing generated-API baseline, not this task's. No Go
was touched.

Follow-up (reviewer P3, non-blocking, deliberately not built): `segmentsOf` in the two test files
and `readPathOf` / `loweringError` in the two examples are now verbatim siblings. When a third
example appears, lift `segmentsOf` into a shared test helper and the other two into
`Temporal.Testpilot.CaseSupport`.

Swept into the commit (not this task's work, left in place rather than reverted):
`.flow/specs/fn-77-typed-operations-parameterized-actions.json` and
`.flow/tasks/fn-77-typed-operations-parameterized-actions.11.md`, both uncommitted flow-state writes
from task .11's receipt and this task's own claim.

stage: impl-review - ran [claude:claude-fable-5-1:high, 1 round, SHIP on first pass]
## Evidence
- Commits: ec1b0d8714b7f60663b34b771489b154e8dd38ce
- Tests: cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Case.CompilerTests => exit 0, mise exec -- make umpire-build-model => exit 0, mise exec -- make umpire-check-case-runtime-conformance => exit 0 (fixture bytes unchanged; typed-unary-case.json did not move), make lint-model => exit 2 at the inherited 169 generated-API findings (167 simpNF #check in Temporal/API/Types.lean + 2 unusedArguments in Temporal/API/Proto.lean); Umpire.Lint clean; no Nexus3 finding, red-first check: mutating the pinned segment literal at Tests/TypedUnary.lean:350 fails exactly that #guard; mutating startedSteps' attribute field number fails it too
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
