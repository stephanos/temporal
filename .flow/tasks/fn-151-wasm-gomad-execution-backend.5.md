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
Prototype scope delivered. Fixture gate TestPrototypeSchedulingDeterminism: runtime_probes under 4 seeds x 10 fresh guests produced identical stdout, WASI transcript digest, choice trace and runtime terminal per seed, 4 distinct schedules across the 4 seeds, and an exact replay in a fresh guest; the choices fixture rejected missing, extra and altered tapes with their divergence reasons. The same holds for the Temporal one-box and six unchanged suites (task 8).

The earlier qualification watchdog failure (seed 4000012 instance 69) is explained as host contention against a 1-minute per-guest watchdog while every fresh guest recompiled its module: fixture guests slowed from ~2 s to 40 s under a concurrent build, and same-seed guests were otherwise identical. The helper's host-local compiled-module cache and wider prototype watchdogs remove the condition. The historical failed run and logs stay preserved in .flow/tmp/fn151-5-current; no 32x100 claim is made (moved to fn-156.1).

Fixes: select observations were validated by re-decoding the whole trace (quadratic, ~7x slowdown on Temporal guests) and are now decoded per record; choice.MaximumTraceBytes is 1 GiB for whole-program guests while the native runner keeps its 64 MiB bound and canonical request bytes. Report: .flow/artifacts/fn151-prototype-report-2026-10-10.md

## Evidence
- Commits:
- Tests: make -C tools/gomad_wasm prototype (warm: 141 s, all gates pass; .flow/tmp/fn151-prototype/prototype-full.log); GOMAD_WASM_PROTOTYPE_ONLY=smoke run of all ./tests suites (972 s incl. 321 s one-time compile; tests-smoke-all.log); make -C tools/gomad_wasm test-go test-helper; cargo fmt --check and clippy -D warnings; make lint-code-gomad-wasm, lint-code-gomad3, lint-code-fast (0 issues); tools/gomad3 ./choice ./runner/... failure set identical to baseline (125 native-toolchain-missing failures, none new)
- PRs:
