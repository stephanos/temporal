---
satisfies: [R3, R4]
---
# fn-240-symbolic-back-end-models-as-btor2.3 Witness decoding and native replay: typed frames by declaration index, checked_step replay, claim re-judged, shared Replay types with umpire-apalache, mutation controls

## Description
Port the probe's witness.rs: parse rIC3 witnesses (sat, b0, #k state sections, @k input sections, .) keyed by state/input declaration index through the Layout; decode two's-complement words and enum indices into typed Values; find the start in world.starts(), step through checked_step choosing the successor matching the assigned fields (record the choice when fields are unassigned; frame 0 must be complete), and re-judge the violated claim natively (invariant predicate, StepClaim::holds, arithmetic refusal for the fault property). Move Replay, Confirmation, Replayed and ReplayedStep out of umpire-apalache into a shared crate (umpire-testing or a small umpire-replay) so both back ends use one definition; update umpire-apalache. Tests: never_beeps witness replayed and confirmed; the claim99 control replays but does not confirm (disagrees); the heat2e9 control's witness is rejected as not a Model path; malformed, duplicate and out-of-range frames refuse.

## Acceptance
- Kettle's never_beeps witness decodes to a 62-step path that replays through checked_step and the claim is natively confirmed.
- The claim99 and heat2e9 controls are tests that fail as disagreement and witness-rejection respectively.
- Replay types have one definition shared by umpire-symbolic and umpire-apalache; umpire-apalache tests still pass.
- Malformed, duplicate, missing-frame-0 and out-of-range witnesses refuse with a message naming the frame and field.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
