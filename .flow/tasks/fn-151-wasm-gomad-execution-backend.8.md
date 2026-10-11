---
satisfies: [R8]
---
# fn-151-wasm-gomad-execution-backend.8 Determinism prototype gate and go/no-go decision

## Description
Run unchanged one-box smoke suites (workflow start/complete, activity, child workflow, update, user timers) with scheduling, time skipping and simulated network all enabled. Measure per-run wall time, CPU, peak guest/host memory, virtual-to-wall ratio and module/compile cost. Record findings and a go/no-go recommendation for fn-156.

## Acceptance
Each admitted suite passes under two seeds; the first seed's recorded run is reproduced byte-for-byte (stdout, WASI transcript digest, choice trace, runtime terminal) by a fresh guest forced through its tape, and the seeds produce different schedules. The 10-fresh-guest budget of the fixture gates is unaffordable for Temporal guests (about 15 minutes each on the shared host), so it is recorded as a production-phase budget in fn-156.1. Unsupported suites fail closed with their first cause. A retained report states measured costs, open determinism gaps and the go/no-go recommendation; the owner decision is recorded in MILESTONES.md.

## Done summary
Integrated gate and decision delivered. Default validation (make prototype, warm compile cache) runs all fixture gates, the Temporal workflow gate with network faults, and the unchanged tests/gomadfunctional TestFrontendSystemInfo, each under two seeds with an exact fresh-guest replay of the first: 141 s wall at load ~13 on the shared 12-core host. The unchanged ./tests suites workflow, activity, child, update (41 tests, unqualified on the stock profile) and timers also pass under two seeds plus replay (selectable; the 338 MB module needs ~6 GiB once to compile). Measured: Temporal guests 7-176 s wall, 110-320 MiB guest memory, ~2,100 host calls per virtual second; one-time Cranelift compile 168-321 s. Wall time is dominated by per-call JSON round trips between helper and model process.

Recommendation GO, recorded in MILESTONES.md under the owner's instruction to follow Claude's recommendations; fn-156 is unblocked, starting with in-helper decision handling (fn-156.8). Report: .flow/artifacts/fn151-prototype-report-2026-10-10.md

## Evidence
- Commits:
- Tests: make -C tools/gomad_wasm prototype (warm: 141 s, all gates pass; .flow/tmp/fn151-prototype/prototype-full.log); GOMAD_WASM_PROTOTYPE_ONLY=smoke run of all ./tests suites (972 s incl. 321 s one-time compile; tests-smoke-all.log); make -C tools/gomad_wasm test-go test-helper; cargo fmt --check and clippy -D warnings; make lint-code-gomad-wasm, lint-code-gomad3, lint-code-fast (0 issues); tools/gomad3 ./choice ./runner/... failure set identical to baseline (125 native-toolchain-missing failures, none new)
- PRs:
