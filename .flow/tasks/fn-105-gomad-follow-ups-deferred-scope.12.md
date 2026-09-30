---
satisfies: [R12]
---
# fn-105-gomad-follow-ups-deferred-scope.12 D12: fix Linux replay divergence and restore strict CI qualification

## Description
Origin: fn-106.1 (2026-09-30). Identify and fix the linux/amd64 replay channel: about one tier-3 seed-run in 26 diverges (nondeterministic or replay_divergence, at choice ordinals from 8 to ~85k, either seed, a different suite each run). Bisect ruled out the FIPS DRBG draw (6bc11ef7d) and the mark-start greying (440552d2c): fork runs with each reverted still diverged (36741951399, 36741956374, 36741960556, 36741965198). A Rosetta linux/amd64 container reproduced it once in 32 seed-17 repetitions and not in 100 sequential replays, so the recorded evidence points to host timing under load. Each fork iteration took an hour at that rate.

Decision on 2026-09-30: this must be fixed. Remove the previous host-availability/rate-rise deferral. Obtain a native linux/amd64 host, reproduce under load, and use buffered per-event runtime logging to identify the first divergent event. Fix the cause and retain a regression reproducer. Keep this task open if the investigation only diagnoses or classifies the cause, or if the host is unavailable.

## Acceptance
- Identify the first divergent event and its causal runtime path on native linux/amd64.
- Fix the cause without widening the deterministic boundary or suppressing divergence evidence. A retained regression reproducer demonstrates the original failure and passes with the fix.
- Repeated qualification with choice tracing enabled and exact replay of affected F5/F6 suites pass on seeds 11 and 17 on native linux/amd64, including host-load runs. Retain commands, platform identity, repetition counts, and outcomes; an environment blocker is not a passing result. Disabling tracing cannot satisfy this acceptance criterion.
- After verification passes, restore the affected Linux expectations to `qualified` and remove both `nondeterministic` and `replay_divergence` allowances from the dispatch-only and required smoke gates. The gates still reject target failures, unsupported targets, and infrastructure errors.
- Diagnosis, classification, or a recorded unresolved cause alone cannot satisfy R12 or close this task.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
