---
satisfies: [R2, R7]
---
# fn-165-gomad-architecture-checks-structural.2 Give the wasm module its own architecture test and a helper-bridge unit test

Touches: tools/gomad_wasm/architecture_test.go, tools/gomad_wasm/wasi, tools/gomad3/architecture_test.go

## Description
Add `tools/gomad_wasm/architecture_test.go` that runs `archcheck.Check` over the wasm module on darwin/arm64 and linux/amd64 with pure set `wasi/` minus `process.go` and `broker_process.go`, a pure import allowlist of the standard-library set plus the gomad3 packages those files import today, and the public-signature check against an external consumer. Replace the analyzer self-test `helper_bridge_boundary_test.go` property with a unit test in `wasi`: feed a hostile helper callback reply through the reply path and assert the environment is byte-identical afterwards. Delete `TestWASIPureEnvironmentAndPublicBoundary` from gomad3 so gomad3 no longer loads the wasm module.

## Acceptance
- `go -C tools/gomad_wasm test ./...` with stock Go runs the structural check and needs no Rust helper or `.toolchain`.
- Adding `import "os"` to `wasi/clock.go` fails the wasm module's own test with `pure-import`; a renamed `process.go` is checked as pure and fails.
- The `wasi` unit test fails when the reply path is changed to write one field of the environment.
- gomad3's `architecture_test.go` contains no reference to `../gomad_wasm`.

## Done summary
The WASM tree runs its own structural check on darwin/arm64 and linux/amd64 with wasi minus the transport files as pure set; host imports and moved transport files fail; a hostile helper reply leaves the environment byte-identical; gomad3 root tests no longer load the WASM tree.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6afac5097, d0f50ff854, b4696f1a78
- Tests: TestWASMArchitecture, TestWASMArchitectureRejectsHostImports (in make -C tools/gomad3 test-wasm, 0), TestHostileHelperCallbackReplyLeavesEnvironmentByteIdentical (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: