---
satisfies: [R6]
---
# fn-151-wasm-gomad-execution-backend.6 Strict virtual time with time skipping (prototype)

## Description
Adapt runtime/lock_wasip1.go timed waits and beforeIdle through a guest idle/timer/deadline handshake connected to World Register/Ready/Cancel/Quiesce. Advance virtual time only when the guest has no runnable work and pending completions are settled; runtime bookkeeping reads never advance logical time. Exploration frontier, resume, minimization and CPU-loop non-progress qualification move to fn-156.2.

## Acceptance
Sleeper/timer-tie/deadline/cancellation fixtures match the strict time contract. A Temporal user-timer and activity-timeout workload with long durations (e.g. hours of virtual time) completes deterministically, with virtual time far exceeding wall time; ratio recorded. No host wall clock or completion order influences choices or time.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
