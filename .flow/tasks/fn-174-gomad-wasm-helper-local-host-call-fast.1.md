---
satisfies: [R1]
---
# fn-174-gomad-wasm-helper-local-host-call-fast.1 Link every helper import as a typed host function

## Description
Replace the per-import dynamic `Linker::func_new` closure and the per-call string dispatch with typed host functions resolved once at link time (R1). Behaviour-neutral refactor: transcript bytes, outcomes and classification stay identical; later tasks change the payload path.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs]

### Approach
- Resolve each `(module, name)` to a closed operation enum once (extend `abi::signature` at `abi.rs:9-69` to return it). Link with `func_wrap` per operation using its exact WASI/cooperative signature, so Wasmtime passes typed params and no `String` is captured per import.
- Split `abi::invoke` (`abi.rs:196`) and `invoke_runtime` (`abi.rs:635-744`) into per-operation entry points that take the enum. Remove the `match op` on strings in `abi.rs` and the op-string locality decision in `Model::handle_input` (`model.rs:372-379`); the enum carries locality.
- Look up the `memory` export once per instance instead of per call (`abi.rs:214`).
- Keep unknown-import and signature refusal in `prepare()` (`engine.rs:484-510`) before `started`; `started.imports` is unchanged.
- Change both linker sites: production (`engine.rs:362-381`) and the unit-test harness (`engine.rs:700-715`).
- Equivalence pin: the existing literal-transcript tests (`typed_clock_*` in `engine.rs`/`model.rs`, `local_clock_abi_preserves_literal_transcript_and_pointer_validation_priority`, `runtime_imports_*` in `tests/engine.rs`) must pass unmodified, and Go `TestHelperModelPreservesGoOwnerEvidence` must stay green against the rebuilt helper.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs:362-381` (linker) and `:484-510` (import refusal)
- `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs:9-69`, `:196-625`, `:635-744`
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:367-463`

**Optional:**
- `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs:597` and `:636` (unknown import and signature tests)

### Key context
- `diagnostic-profile` gated `AbiSpan::new(module, op, local)` (`abi.rs:203`) takes strings; give it the enum (or a static name) so diagnostic builds keep their buckets. Build and test the diagnostic feature in a separate `--target-dir`.

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-profile --target-dir /tmp/gomad-wasmhost-diag
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestHelperModelPreservesGoOwnerEvidence|TestTransport' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] No `func_new` or per-call import-name string match remains on the production or test linker path.
- [ ] Unknown imports and signature mismatches are refused before `started` with unchanged classification and messages.
- [ ] Existing literal transcript, runtime-import and pointer-validation tests pass unmodified; Go owner/helper parity stays green.
- [ ] Default and `diagnostic-profile` builds pass `cargo test`; `cargo clippy` clean.

## Done summary
Every helper import is linked with `func_wrap` through one `operations!` table resolved once at link time; no `func_new` and no per-call import-name match remain. Explicit `Operation::is_model_local`/`is_namespace` tables decide routing (unit test pins them against import names), memory is looked up once per instance, and unknown/mismatched imports are still refused before `started`. Impl review (claude backend): NEEDS_WORK (per-call prefix match) -> fixed in 1d09701583 -> SHIP.

Native execution remains deferred (fn-128/fn-149). Performance proof belongs to the .6 task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fast-batch tip 08f92e58ee (frozen qualification candidate d0372e7a20)
- Tests: frozen qualification /home/agent/wt/fast-qual-20261014T1345: run-qualification.sh + continuations (stages.log ends PASS frozen-qualification module=gomad), make -C tools/gomad/internal/backends/wasm test-conformance, cargo +1.94.1 test/clippy -D warnings (default, diagnostic-profile, conformance)
- PRs: