---
satisfies: [R6]
---
# fn-151-wasm-gomad-execution-backend.6 Strict virtual time with time skipping (prototype)

## Description
Adapt runtime/lock_wasip1.go timed waits and beforeIdle through a guest idle/timer/deadline handshake connected to World Register/Ready/Cancel/Quiesce. Advance virtual time only when the guest has no runnable work and pending completions are settled; runtime bookkeeping reads never advance logical time. Exploration frontier, resume, minimization and CPU-loop non-progress qualification move to fn-156.2.

## Acceptance
Sleeper/timer-tie/deadline/cancellation fixtures match the strict time contract. A Temporal user-timer and activity-timeout workload with long durations (e.g. hours of virtual time) completes deterministically, with virtual time far exceeding wall time; ratio recorded. No host wall clock or completion order influences choices or time.

## Done summary
Prototype scope delivered. The cooperative profile's frozen clock (zero read-step) advances only at the runtime idle handshake or a blocking poll, to the earliest deadline. TestPrototypeVirtualTime: 9h16m of virtual time per guest in 0.1-0.2 s wall with exact deadlines for a 2 h sleep, equal-deadline timers, a stopped timer, context deadline, 36 ticker periods and cancellation; runnable work and zero-time polls never move the clock; 3 seeds x 5 fresh guests plus replay identical. Temporal: the prototype workflow (two 1-minute sleeps and activities retried after 30 s backoffs) completes in 3 virtual minutes per guest with 3,104 clock jumps from server housekeeping, reproduced by replay. Integration with World's arbiter, exploration, resume and CPU-loop non-progress outcomes remain fn-156.2. Report: .flow/artifacts/fn151-prototype-report-2026-10-10.md

## Evidence
- Commits:
- Tests: make -C tools/gomad_wasm prototype (warm: 141 s, all gates pass; .flow/tmp/fn151-prototype/prototype-full.log); GOMAD_WASM_PROTOTYPE_ONLY=smoke run of all ./tests suites (972 s incl. 321 s one-time compile; tests-smoke-all.log); make -C tools/gomad_wasm test-go test-helper; cargo fmt --check and clippy -D warnings; make lint-code-gomad-wasm, lint-code-gomad3, lint-code-fast (0 issues); tools/gomad3 ./choice ./runner/... failure set identical to baseline (125 native-toolchain-missing failures, none new)
- PRs:
