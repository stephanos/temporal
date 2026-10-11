---
satisfies: [R7]
---
# fn-151-wasm-gomad-execution-backend.7 Simulated in-guest network (prototype)

## Description
Add an interception seam in the guest net implementation before fake TCP delivery, reusing gomadio network semantics through a narrow guest adapter. Model stable listener/connection identities, deterministic ordered delivery, delay, partition/heal and reset, driven by the same virtual clock and seed. Single guest only; reference-model conformance, backpressure/half-close/deadline completeness, storage faults and multi-node brokering move to fn-156.

## Acceptance
Real gRPC/SDK traffic in the one-box guest flows through the model; same seed gives identical delivery transcripts. Injected delay and a connection reset/partition during a workflow produce a Temporal retry/recovery that replays with matching choice, network and outcome evidence.

## Done summary
Prototype scope delivered. The overlay patches net/net_fake.go so stream connects and writes consult an optional host network model (gomad_wasm_v1 net import, wasi/network.go): seeded per-operation delays (writer held, preserving order), and reset, partition and heal faults on virtual time; decisions form the NetworkTranscript evidence stream; without a model the guest never calls the host. Fixture gate: TCP echo reset at 3 s, dials refused during the 6-8.5 s partition, different network seed changes decisions, 3 seeds x 5 fresh guests plus replay identical. Temporal: all connections reset at 40 s and everything partitioned 100-110 s during the workflow; 12 resets and 3 refused dials, the workflow completes and the replay reproduces the network transcript. One guest only; backpressure, half-close, deadlines, reference-model checks and the broker remain fn-156.4/.5. Report: .flow/artifacts/fn151-prototype-report-2026-10-10.md

## Evidence
- Commits:
- Tests: make -C tools/gomad_wasm prototype (warm: 141 s, all gates pass; .flow/tmp/fn151-prototype/prototype-full.log); GOMAD_WASM_PROTOTYPE_ONLY=smoke run of all ./tests suites (972 s incl. 321 s one-time compile; tests-smoke-all.log); make -C tools/gomad_wasm test-go test-helper; cargo fmt --check and clippy -D warnings; make lint-code-gomad-wasm, lint-code-gomad3, lint-code-fast (0 issues); tools/gomad3 ./choice ./runner/... failure set identical to baseline (125 native-toolchain-missing failures, none new)
- PRs:
