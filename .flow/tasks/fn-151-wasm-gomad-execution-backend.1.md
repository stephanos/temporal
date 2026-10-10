---
satisfies: [R1]
---
# fn-151-wasm-gomad-execution-backend.1 Implement isolated WASM engine and typed execution protocol

## Description
Add pinned Wasmtime helper and typed execution/call/result protocol, validate module/import/memory identities, run tiny real guests and bound infinite execution. Preserve native owners. Tests cover malformed requests, unknown imports, fuel/memory limits and fresh instances.

## Acceptance
Add pinned Wasmtime helper and typed execution/call/result protocol, validate module/import/memory identities, run tiny real guests and bound infinite execution. Preserve native owners. Tests cover malformed requests, unknown imports, fuel/memory limits and fresh instances.

## Done summary
Implemented the isolated pinned Wasmtime 47.0.3 helper with typed WASI callbacks,
pre-instantiation authorization, explicit limits and distinct termination classes.
All 33 actual import names have an explicit callback or denial; Go owns models.

The final release helper compiled the exact main-based module, instantiated it,
and executed 11 Go runtime callbacks before explicitly rejecting the scratch
environment's unimplemented stdin flags. The helper then reaped normally.
This proves the R1 engine gate, without claiming help completion, server startup,
scheduler control, replay or native qualification.

Fresh independent same-family Codex review verified all execution fixes. Root
reran release tests: 32 guest tests plus two unit tests passed, verified the
module/helper/frame identities and source manifests, and checked the sole later
documentation correction. The reviewed helper binary is unchanged. Exact worker
commands/results and red regressions are retained in handover.json; actual
runtime evidence is server-probe-release.json and its durable frames.

Delivered source manifest: 125d300730c7edbf3ec5dc5dc63f3424369b22a090284459027a10b6b875576e.
Helper: 1306e745fbc9a3dc607bb428a5ebc786132e09959cc3f60a19b40ce684ca5d50.
Module: 551734fd82cb854ad03651f8bbb6bdb66f75acb6786ab106bb3b207f48a45743.
No existing Go, native overlay or generator input changed. New Rust code does not
require repeating unrelated Go host gates. Commits remain with the user; source
and evidence are uncommitted. Native fn-128/fn-149 qualification stays deferred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: cargo +1.94.1 fmt --manifest-path tools/gomad3/wasmhost/Cargo.toml -- --check, cargo +1.94.1 clippy --manifest-path tools/gomad3/wasmhost/Cargo.toml --locked --all-targets -- -D warnings, cargo +1.94.1 test --manifest-path tools/gomad3/wasmhost/Cargo.toml --locked, cargo +1.94.1 build --manifest-path tools/gomad3/wasmhost/Cargo.toml --release --locked, cargo +1.94.1 test --manifest-path tools/gomad3/wasmhost/Cargo.toml --release --locked, python3 .tmp/wasm-backend-engine/probe-release.py, root source/module/helper/frame digest verification, git diff --check
- PRs: