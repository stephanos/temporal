---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.3 The signature constructs its messages: actions, inputs, entities, observations, channels, assumptions, holes, Limits, types

## Description
`ActionDecl` → `ir.Action` (it already has nearly every field; add the derived ID and schemas), inputs and their `Param`s, entities and observations, channels, assumptions, holes, Limits. Types come from the `Finite` instances each machine holds for its state, outcome and fact, and from the input domains, without trees; where `Types.scala` lifted a type from a tree, the same description must come from the run-time value. The driver takes the signature from the objects; the lifter's `Declarations` arms for these kinds go.

## Acceptance
- Every signature kind in the IR comes from constructed values; the corresponding lifter arms are deleted (R3).
- Names and derived IDs equal today's (R4).
- Types equal today's as messages (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
