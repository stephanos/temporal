---
satisfies: [R8]
---
# fn-151-wasm-gomad-execution-backend.8 Determinism prototype gate and go/no-go decision

## Description
Run unchanged one-box smoke suites (workflow start/complete, activity, child workflow, update, user timers) with scheduling, time skipping and simulated network all enabled. Measure per-run wall time, CPU, peak guest/host memory, virtual-to-wall ratio and module/compile cost. Record findings and a go/no-go recommendation for fn-156.

## Acceptance
Each admitted suite passes under two seeds; the first seed's recorded run is reproduced byte-for-byte (stdout, WASI transcript digest, choice trace, runtime terminal) by a fresh guest forced through its tape, and the seeds produce different schedules. The 10-fresh-guest budget of the fixture gates is unaffordable for Temporal guests (about 15 minutes each on the shared host), so it is recorded as a production-phase budget in fn-156.1. Unsupported suites fail closed with their first cause. A retained report states measured costs, open determinism gaps and the go/no-go recommendation; the owner decision is recorded in MILESTONES.md.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
