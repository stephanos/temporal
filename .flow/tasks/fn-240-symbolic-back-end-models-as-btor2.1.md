---
satisfies: [R1]
---
# fn-240-symbolic-back-end-models-as-btor2.1 Crate umpire-symbolic: BTOR2 emitter over the embedded admitted tree for Kettle-shaped Models, checked arithmetic and one-step history, emission goldens

## Description
Port the probe's emitter (model/design/native-symbolic/src/btor2.rs) into model/umpire/backends/symbolic as a workspace member with the umpire-tlc manifest conventions. Read the compiled World's embedded admitted tree (Lifts::__umpire_lifted), never source. Fields, enums with domain constraints, World fields as init-bound states in concrete mode, first-match branches, enabledness and stutter via a free act input, one-step history for postconditions with after = filters, checked arithmetic latched as its own property (overflow by widening, division and remainder by zero, MIN/-1), one bad per property with property kind metadata. Hash-cons every node, operators as well as sorts and constants (IndexSet or FxHashMap keyed by the node, insertion order giving the BTOR2 id), so a repeated subexpression is emitted once; no BTOR2 crate (flussab-btor2, btor2rs, btor2tools fail fn-210 R18). Carry a Layout (field, declaration index, width, signedness, Type) for the witness decoder. Unsupported forms refuse naming the construct. Tests: emitted-text goldens for sample_kettle and kettle_cut, a refusal per unsupported form, the two spike controls (claim99, heat2e9) emitted from mutated trees. No tool execution in this task.

## Acceptance
- umpire-symbolic builds in the model workspace under the workspace lints and emits every property of sample_kettle and kettle_cut as a BTOR2 file matching a golden.
- Postconditions with after = filters, enabledness, stutter and the arithmetic fault latch are present in the emitted text and covered by tests.
- Every node (operators included) is hash-consed: a test emits a Model with a repeated subexpression and finds it once in the output.
- Unsupported admitted forms (action inputs, sequences, non-unit outcomes, computed starts) refuse with a located message naming the construct and Model.
- Clippy, fmt and docs pass; make model-check default path unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
