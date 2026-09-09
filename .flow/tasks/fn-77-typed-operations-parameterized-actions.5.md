---
satisfies: [R3, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.5 Evaluate checked same-step field Properties

## Description
Evaluate checked same-step field Properties for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Property/**; model/Umpire/Property.lean; model/Umpire/Operation/**; model/Umpire/Value/**
**Touches:** [model/Umpire/Property/**, model/Umpire/Property.lean, model/Umpire/Operation/**, model/Umpire/Value/**]

### Approach
- Extend existing closed Property data with typed operands for immutable request, prior/resulting state, outcome and semantic event values; leave existing literal predicates unchanged.
- Admit compatible equality/inequality, declared scalar ordering, Boolean composition, presence and oneof tests using task3 access; track branch-local availability and reject missing values rather than comparing absent sentinels.
- Keep raw denotation behind checked evaluation input and authoring errors at the actual operand/clause source. Use ordinary typed constructors or focused existing authoring syntax, never callbacks.
- Add independent request/state and request/result requirements whose clauses are not derived from the transition under test; prove evaluation agreement with typed operands and canonicalization compatibility.

### Investigation targets
**Required:**
- model/Umpire/Property/Language.lean:132 — existing literal-only contract.
- model/Umpire/Property/Check.lean — checked admission owner.
- model/Umpire/Property/Evaluation.lean — existing evaluator owner.
- model/Umpire/Property/Authoring.lean — source-owned authoring diagnostics.
- common/testing/testpilot/internal/ir/expression.go:101 — guarded availability reference.

### Quick commands
`cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Property.ImportTests Umpire.Query.Tests`

`cd model && mise exec -- lake build Umpire.Property.Tests.Fields`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Property.Tests.Fields into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Independent request/state and request/result clauses evaluate through the existing checked Property owner and preserve partial-field meaning.
- [ ] Wrong types/context/schema, absent required values and unselected oneof branches reject at source; Boolean availability does not leak across branches.
- [ ] Typed and surface forms have identical new meaning while old literal canonical bytes and evaluator results remain unchanged.
- [ ] Load-bearing operand/evaluator correspondence and negative authoring tests pass without new axiom dependencies.

## Done summary
Checked same-step field Properties now admit heterogeneous evidence: `PropertyFieldEvidence` erases
the generated owner/witness index at input admission (its private constructor is reachable only
through a real `PropertyFieldProjection`), the global `requireFieldOwner` homogeneity guard is gone,
and admission rests on the per-operand `(reference, schema)` binding check already owned by
`Umpire/Property/Check.lean`. Independently authored request, prior/resulting-state, outcome and
event operands can therefore share one same-step context, and the field tests bind model state and
semantic events to a second `RpcOwner`/`Schema` so request/state, request/result and
resulting-state/event relations are genuinely mixed-schema.

stage: impl-review - ran [round 1 NEEDS_WORK (copilot/gpt-5.4) .. round 2 SHIP (copilot/gpt-5.4)]
## Evidence
- Commits: 8ecf132a160d7bea386afc130f2551fb80f2b813, 0491652266a384f2d1c4bd358cc3e2f3cdf319cb, 47fb8ac98e8d78d71d87b895ebd74fb637afc49a, 7a18621bee3998ae94126c9be08e2fa9ead145bd, 9829a291cdc6f66f241bb4f755a5aebd366c114e, e670f137367d965dd121d6f6d9bef4c0931a088a
- Tests: cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Property.ImportTests Umpire.Query.Tests, cd model && mise exec -- lake build Umpire.Property.Tests.Fields, mise exec -- make umpire-build-model
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
