---
satisfies: [R6]
---
# fn-242-gomad-shrink-the-qualification.8 Requalification: one fresh full cycle under the verb with verdict parity

## Description
Run the full 96-stage cycle with the Rust verb on gomad head (after fn-240 .2/.3 have merged if the fn-242-before-fn-240 ordering holds, so one cycle covers both). Retain command exits, signals, identities and the verdict per stage under .flow/artifacts/fn242-cycle-<date>/; compare verdicts side by side with the fn-232.13 closing cycle. Wall time is recorded; a timing-only miss is deferred to the quiet-host window, a correctness difference blocks. Touches: .flow/artifacts only.

## Acceptance
- [ ] All 96 stages ran under the verb with retained exits and identities
- [ ] Verdict parity table against the fn-232.13 closing cycle with zero correctness differences
- [ ] Wall time recorded against the 3,600 s limit; any miss filed as a quiet-window item, not waived

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
