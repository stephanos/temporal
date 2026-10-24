---
satisfies: [R5, R8, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.6 Hooks instead of features: wasm-hooks, wasm-diagnostic, gomad-fixture

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
Create crates/wasm-hooks (Observer + FaultHook traits, zero-sized no-op defaults; no logic), crates/wasm-diagnostic (diagnostic.rs + guest_profile schema; implements Observer) and crates/gomad-fixture (fixture_*, live_*; implements FaultHook; installs via install_hooks()). Replace every cfg(feature = diagnostic-profile|determinism-fixture|stress-jitter) reference in library code (abi 24, engine 19, broker 12, volume 26, cache, protocol, reply, storage_transport, linked, linked_cluster, cluster execution, backstop, campaign, minimize) with a call on the hook held in state. Also route the test-only environment hooks read deep in library code through the same FaultHook registry instead of std::env: GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER (campaign/exploration/round.rs, campaign/ledger.rs), GOMAD_FIXTURE_CAMPAIGN_PANIC_SEED (campaign/mod.rs), GOMAD_TEST_UNACCEPTED_MODULE/RECIPE/SHA256 (campaign/prepare/driver.rs), GOMAD_TEST_BLOCKED_STDERR (campaign/launcher.rs), GOMAD_CORE_COUNTERFACTUAL_* (campaign/counterfactual.rs), GOMAD_WASM_EVENT_ORDER, GOMAD_STRESS_JITTER_TEST_CHILD, GOMAD_WASMHOST_PEAK_CHILD, GOMAD_INSPECTION_RAW_DIR; operator-facing variables (GOMAD_WASM_INSTALLATION, GOMAD_SHIM_*, GOMAD_WASM_STRESS*) stay but are read once in the bin crate and passed down. Keep compile_error! exclusivity only where a feature remains. Decision: if release binaries must provably exclude diagnostic code, keep one cfg at the hook installation site in the bin crate and record it. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); typed error kinds and documented refusal behavior remain tested; message text and goldens may change. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.
## Acceptance
grep -rn 'cfg(feature' over crates/*/src shows only wasm-engine guest-profile; diagnostic, determinism-fixture and stress-jitter builds produce current profile/fixture behavior with authenticated deterministically regenerated outputs on the existing fixture corpus; release build identity recorded; cargo test --workspace and the determinism lanes green; each new crate has the contract doc and tests/ boundary tests. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
