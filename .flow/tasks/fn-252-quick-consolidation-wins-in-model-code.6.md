---
satisfies: [R6, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.6 Activity TLC replay through umpire_testing::replay_trace

## Description
Implement **Activity TLC replay (B9)**: expose `umpire_testing::replay_trace` (or fn-250's replay core if landed) and remove `proof_checks.rs` `state_value`/`matches_state`/`replay_tlc`, comparing on `Values`.

## Acceptance
- [ ] The activity proof checks replay through the shared function.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
