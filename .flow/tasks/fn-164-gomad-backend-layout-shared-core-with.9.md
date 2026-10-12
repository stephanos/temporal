---
satisfies: [R9]
---
# fn-164-gomad-backend-layout-shared-core-with.9 CI path filters for the wasm suite and shared packages

Touches: .github/workflows/gomad3.yml, .github/workflows/gomad3-smoke.yml

## Description
Add a wasm job that runs `make -C tools/gomad3 test-wasm` with stock Go and the pinned Rust toolchain on changes under `tools/gomad3/backends/wasm` and under shared gomad3 packages. Native jobs skip when the change set is confined to `backends/wasm`. Workflow edits only; this task grants no authority to dispatch or push, per the MILESTONES CI restrictions. Validate the YAML locally with `actionlint` or the workflow schema.

## Acceptance
- Path filters: a change touching only `backends/wasm` triggers the wasm job and not the native jobs; a change under `record` triggers both; a change under `backends/native` triggers native only.
- The wasm job installs the pinned Rust version and Wasmtime crate from the lockfile and does not build the native `.toolchain`.
- Workflow files pass `actionlint` with no findings.

## Done summary
CI path gates route backends/wasm-only changes to the WASM job, record to both and backends/native to native; tools/gomad3sim triggers WASM too. The WASM job pins Rust 1.94.1 with --locked and stock Go. No workflow was dispatched.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3750cab464, 2079fdcea3
- Tests: TestCIBackendGatesRoutePaths (0), actionlint v1.7.7 on gomad3.yml and gomad3-smoke.yml (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: