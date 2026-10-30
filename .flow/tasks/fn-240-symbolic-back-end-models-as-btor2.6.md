---
satisfies: [R7]
---
# fn-240-symbolic-back-end-models-as-btor2.6 Rest of the admitted subset: action inputs, non-unit outcomes, bounded sequences, finite sets/maps from fn-245; input, return and collection goldens

## Description
Extend the emitter: action inputs over finite domains as free inputs constrained to their domain (replay carries the decoded input into checked_step); non-unit outcomes as part of the one-step history so after = filters on outcomes work; sequences with a declared bound as fixed-length arrays of words with a length state (Length, Append, GetOr), refusing an unbounded sequence by name. Use the Rust width/signedness metadata supplied by .1, keeping declared ranges as independent constraints. Complete fn-245's finite-collection handoff: sets as one membership bit per canonical element, maps as canonical key slots with value words and presence bits, and the admitted set/map/list operations including Forall/Exists, Front/PopFront and Contains. Quantifiers preserve canonical traversal order and short-circuit checked-failure semantics; eager GetOr/Front defaults retain their arithmetic checks. Bound sequence storage from the admitted authored length guard or an explicit internal derived bound; refuse if a finite sound bound cannot be established. Goldens for action_inputs, shared_returns, finite_sets, finite_maps and bounded_queue; the remaining refusals name the construct.

## Acceptance
- action_inputs and shared_returns have expected/symbolic.txt with every property agreeing and replayed violations carrying the decoded inputs.
- Widths use .1's Rust backing metadata for every fixture; narrowed ranges remain independent constraints.
- finite_sets, finite_maps and bounded_queue agree with exhaustive native checking and replay false claims through the shared core, including absent map slots, duplicate/no-op updates, ordered quantifier short-circuit arithmetic, eager defaults and empty queue operations. Unbounded or unsupported forms refuse by location/name.
- A bounded sequence Model emits and agrees; an unbounded one refuses naming the field.
- Earlier fixtures keep the same verdicts and agreement; their goldens may be re-accepted after review.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
