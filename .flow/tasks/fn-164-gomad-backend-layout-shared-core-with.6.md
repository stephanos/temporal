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
The WASM module is folded into tools/gomad3/backends/wasm with one Go module, a stock-Go test-wasm target and a guard rejecting any native backend import; the integrated compiled model identity is pinned. test-wasm passes on the final candidate.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 613fc476a5, b60b038600, 6283a4ec90, 7ac7b4bc54
- Tests: make -C tools/gomad3 test-wasm (0); root architecture selection with GOMAD3_ARCH_ALL_PLATFORMS=1 (0); make -C tools/gomad3 validate (0); make lint-code-fast base e43c7a59c7 (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: