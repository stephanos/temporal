---
satisfies: [R1, R5]
---
# fn-97-gomad-f3-qualify-the-frontend.1 Qualify TestFrontendSystemInfo on darwin/arm64 with seeds 11 and 17

## Description
Analyze in closure/guarded modes on darwin, then `gomad qualify --seed {11,17} --repeat 2 --choices --replay-successes`. Trace any divergence to its channel and fix in Gomad.

## Acceptance
- both seeds identical evidence, replay_match, choice_replay_exact
- transcript bytes and decisions recorded

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
