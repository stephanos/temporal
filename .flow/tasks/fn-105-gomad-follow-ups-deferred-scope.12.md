---
satisfies: [R1, R2]
---
# fn-105-gomad-follow-ups-deferred-scope.12 D12: identify the linux replay-divergence channel

## Description
Origin: fn-106.1 (2026-09-30). Identify the linux/amd64 replay channel: about one tier-3 seed-run in 26 diverges (nondeterministic or replay_divergence, at choice ordinals from 8 to ~85k, either seed, a different suite each run). Bisect ruled out the FIPS DRBG draw (6bc11ef7d) and the mark-start greying (440552d2c): fork runs with each reverted still diverged (36741951399, 36741956374, 36741960556, 36741965198). A Rosetta linux/amd64 container reproduced it once in 32 seed-17 repetitions and not in 100 sequential replays, so it depends on host timing under load. Deferred because each fork iteration takes an hour at that rate. Revive when a linux/amd64 host is available for the buffered per-event runtime logging F7 used on darwin, or the rate rises.

## Acceptance
- the channel is identified and fixed, or recorded; the F5/F6 linux expectations return to qualified and the smoke gate stops accepting nondeterministic


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
