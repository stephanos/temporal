---
satisfies: [R5]
---
# fn-151-wasm-gomad-execution-backend.5 Deterministic cooperative scheduling and exact replay (prototype)

## Description
Migrated on 2026-10-10. Implementation and acceptance are incomplete; work is stopped pending owner direction. Retained partial handover: .flow/tmp/fn151-5-current/summary.md and evidence.json. Required qualification failed after 400 retained fresh records and 4 separate exact replays, with shard4 seed4000012 instance69 exceeding the wall watchdog; no aggregate/full support bound or destination qualification is claimed. Preserve the old source-bound manifest and failed logs without rebinding them as new evidence.

Prototype scope (owner split 2026-10-10): port stable goroutine/timer identities and seeded run-queue/ready-select choices into the WASM guest runtime through bounded versioned host imports, reusing choice tape owners. Same seed must give the same run; a recorded tape must force the same run. Map/GC/runtime-entropy-stream qualification, the optional checkpoint profile and the 32x100 repeat budget move to fn-156.1.

## Acceptance
Fixture and one-box Temporal smoke guests: same seed yields identical canonical choice/I-O transcripts across 4 seeds x 10 fresh instances; at least two seeds give different schedules; a recorded tape replays exactly and missing/extra/altered decisions are rejected. The prior watchdog failure is diagnosed (fixed here or attributed to .6).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
