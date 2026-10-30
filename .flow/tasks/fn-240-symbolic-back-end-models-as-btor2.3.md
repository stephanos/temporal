---
satisfies: [R3, R4]
---
# fn-240-symbolic-back-end-models-as-btor2.3 Witness decoding and native replay: typed frames by declaration index, checked_step replay, claim re-judged, existing fn-250.3 shared replay core, mutation controls

## Description
Port the probe's witness.rs: parse rIC3 witnesses (sat, b0, #k state sections, @k input sections, .) keyed by state/input declaration index through the Layout; decode two's-complement words and enum indices into typed Values; find the start in world.starts(), complete the whole trace in deterministic native order, backtracking across choices consistent with every assigned state, input, history, outcome and fact value (frame 0 must be complete), and check candidate exact edges through fn-250.3's umpire_tla::replay::step core and checked_step, and re-judge the violated claim natively (invariant predicate, StepClaim::holds, arithmetic refusal for the fault property). Reuse the existing core and its EdgeValues/ReplayedEdge/ReplayError contracts; retain only BTOR2 framing, completion, stutter/history and selected-property policy locally. Share neutral result types through existing backend support only where both adapters require them; do not depend on umpire-testing or introduce a second replay core. Completion must satisfy the entire trace and selected violation before committing a confirming path. Search all candidates before classifying a complete native path with no confirming violation as disagreement; if no complete native path exists, witness-reject. Explicit node/time budget exhaustion is a witness-rejected refusal without a disagreement or agreement conclusion. Tests: never_beeps witness replayed and confirmed; the claim99 control replays but does not confirm (disagrees); the heat2e9 control's witness is rejected as not a Model path; malformed, duplicate and out-of-range frames refuse.

## Acceptance
- Kettle's never_beeps witness decodes to a valid path that replays through checked_step and natively confirms the claim violation. Do not pin the probe's exact witness bytes, path choice or length.
- The claim99 and heat2e9 controls are tests that fail as disagreement and witness-rejection respectively.
- Symbolic and Apalache exact-edge replay use the fn-250.3 shared core; typed identity, successor, outcome and ordered-fact comparisons remain enforced and Apalache tests pass.
- A partial witness whose first local match dead-ends completes through a later valid choice; search exhaustion refuses with frame/context and cannot become a successful verdict. Shared result types are reused where needed without a backend-to-testing dependency.
- Malformed, duplicate, missing-frame-0 and out-of-range witnesses refuse with a message naming the frame and field.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
