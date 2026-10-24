---
satisfies: [R7]
---
# fn-240-symbolic-back-end-models-as-btor2.6 Rest of the admitted subset: action inputs, non-unit outcomes, bounded sequences, Rust widths on Type::Int; action_inputs and shared_returns goldens

## Description
Extend the emitter: action inputs over finite domains as free inputs constrained to their domain (replay carries the decoded input into checked_step); non-unit outcomes as part of the one-step history so after = filters on outcomes work; sequences with a declared bound as fixed-length arrays of words with a length state (Length, Append, GetOr), refusing an unbounded sequence by name; Rust integer widths carried on Type::Int by the front end so the emitter stops deriving widths from ranges (the declared range stays a domain constraint). Goldens for action_inputs and shared_returns; the remaining refusals name the construct.

## Acceptance
- action_inputs and shared_returns have expected/symbolic.txt with every property agreeing and replayed violations carrying the decoded inputs.
- Type::Int carries the Rust width; emitted widths match the Rust type for every fixture, with the range as a constraint.
- A bounded sequence Model emits and agrees; an unbounded one refuses naming the field.
- Earlier goldens are byte-identical or each change is justified in the task evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
