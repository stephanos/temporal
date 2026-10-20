---
satisfies: [R3, R4]
---
# fn-218-test-space-patterns-cover-and-the-full.3 Kettle and mini activity cover tests; README grammar table

## Description
Add the real Kettle and activity cover tests with expansion and coverage goldens. Use actual paths fixtures/sample_kettle and temporal/features/activity. Keep Cases(81 * 2) for Kettle's repeated Heat prefix crossed with Depress/Failover endings, inserting an explicit gap before the ending alternatives as the spec explains; keep the literal adjacent impossible form as a refusal control. Preserve Cases(2 * 4) for activity's polled-or-not × four closings with an explicit gap where needed and the appropriate declared deadline actions. Do not change Model enabledness to make alternatives pass. Convert the archived Kettle enabledness table into one cover test while preserving every enabled/disabled assertion, including explicit disabled controls. Add README 'Describing a test space' with grammar and the three confidence mechanisms. **Touches:** `model/umpire/tests/fixtures/sample_kettle/tests.rs`, its `expected/`, `model/temporal/features/activity/tests.rs`, its `expected/`, `model/README.md`. **Quick:** `cd model && mise exec -- cargo test -p umpire-tests -p umpire-temporal-tests`; full gates `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`.

## Acceptance
- [ ] Kettle's 162 and activity's eight authored alternatives each have witnesses and checked expansions; literal impossible adjacency is refused.
- [ ] All archived Kettle enabledness table cases retain their assertions, including disabled actions, in the cover test.
- [ ] Expansion/coverage goldens and README grammar/confidence explanation are checked in.
- [ ] Focused tests, normal/required-TLC model gates and make model-time pass; Kettle/activity cover feedback fits the inner-loop budget, with any gate soft-target overrun recorded separately.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
