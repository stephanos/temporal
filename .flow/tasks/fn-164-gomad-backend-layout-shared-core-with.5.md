---
satisfies: [R6]
---
# fn-164-gomad-backend-layout-shared-core-with.5 Make native a Provider behind backends/backend.go

Touches: tools/gomad3/backends/backend.go, tools/gomad3/backends/native, tools/gomad3/runner

## Description
Create `backends/backend.go` with `Provider` (today's four methods plus `Capabilities() Capabilities` as specified) and a registry. Move `runner/internal/execution`, `runner/coordinator_process_unix.go` and the native branches of `runner.go`, `backend.go` and `replay_operation.go` into `backends/native`, which implements `Provider`. `runner` takes a `Provider` and keeps no `config.Backend == nil` default path. If a native path needs a Provider method beyond those, add it and record the reason in fn-164's Decision Context.

## Acceptance
- `go list -deps ./runner` contains no `backends/` package.
- `TestRunnerExecutionInjectionIsPrivate` and the external Runner consumer compile test still pass.
- Native campaign, replay and minimize integration tests pass through the Provider path with artifacts identical in content to a pre-change run on the same seed.
- Any Provider method beyond `Capabilities` plus three is listed in the spec's Decision Context with its reason.

## Done summary
Native is a Provider behind backends/backend.go; runner imports only the backends root. Guest verification found and fixed resumed-plan, fresh-campaign and shard validation environments and restored the backend-selector error order.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b97c0e9c63, c46b5058e8, a2b42872f8, 7299fb6be9, 4e9140c637, 8ea822b67b, c9486b7272
- Tests: make -C tools/gomad3 test-wasm (0); root architecture selection with GOMAD3_ARCH_ALL_PLATFORMS=1 (0); make -C tools/gomad3 validate (0); make lint-code-fast base e43c7a59c7 (0), TestWASMChoiceCampaignInterruptedRoundRecoveryNoDuplicates, TestWASMCooperativeCampaignWithExplicitEnvironment, ./runner failure set identical to base (native-host only), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: