---
satisfies: [R4]
---
# fn-164-gomad-backend-layout-shared-core-with.6 Fold the wasm module into backends/wasm with a stock-Go test target

Touches: tools/gomad_wasm, tools/gomad3/backends/wasm, tools/gomad3/Makefile, tools/gomad3/.gitignore

## Description
Move `tools/gomad_wasm/*` into `tools/gomad3/backends/wasm` (provider, wasi, wasmhost, overlay, installation, simulation, qualification, internal/stockgo). Delete the wasm `go.mod`, `go.sum` and both `replace` directives; the Rust helper crate moves with `wasmhost`. Point the stockgo cache at `tools/gomad3/.toolchain/stock` and gitignore it. Add `make -C tools/gomad3 test-wasm` that runs the wasm packages with stock Go and `GOMAD3_STOCK_GO`, and a guard test that fails if any wasm test references the native `.toolchain` build. Update the wasm README and CLI docs paths.

## Acceptance
- `tools/gomad_wasm` no longer exists; `find tools -name go.mod` lists one module under `tools/gomad3`.
- `make -C tools/gomad3 test-wasm` passes with stock Go and leaves no `.toolchain/go*` native build behind; the guard test fails when a wasm test imports the native toolchain builder.
- The import allowlist for wasm from fn-164.1 is replaced by the `backends/wasm` module-edge rule.
- Cooperative and exploratory integration tests pass from the new location with the Rust helper.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
