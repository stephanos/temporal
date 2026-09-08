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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
