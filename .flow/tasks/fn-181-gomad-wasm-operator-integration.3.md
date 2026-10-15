---
satisfies: [R3]
---
# fn-181-gomad-wasm-operator-integration.3 Inspect and replay newly produced cluster campaign artifacts

## Description
Extend existing retained cluster replay/inspection for task .2 artifacts, implementing R3 without another replay engine.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/command/command.go, command_test.go, cluster/retained.go, cluster/replay.go, tools/gomad/internal/runner/inspect/inspect.go
**Touches:** [tools/gomad/internal/backends/wasm/command/**, tools/gomad/internal/backends/wasm/cluster/retained*, tools/gomad/internal/backends/wasm/cluster/replay*, tools/gomad/internal/runner/inspect/**]

### Approach
- Feed new payloads through RetainedClusterReplay, VerifyRetained and ReplayBoundRetained. Preserve schema/identity refusal ordering.
- Resolve every boot, captured file and recovery image from retained bytes; preserve per-incarnation tape completeness and shared lifecycle/network/storage evidence. Do not claim a global exact-choice tape.
- Extend inspection only for missing scenario/boot metadata. Preserve fn-180 metadata if present, without creating a feature dependency.
- Extend detached operator tests to remove source/compiler/mounts and preparation caches before replay, retaining the compatible installed helper.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/command/command.go:56
- tools/gomad/internal/backends/wasm/command/command_test.go:211
- tools/gomad/internal/backends/wasm/cluster/retained.go:382
- tools/gomad/internal/backends/wasm/cluster/replay.go
- tools/gomad/internal/runner/inspect/inspect.go

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/cluster ./internal/runner/inspect` with helper inputs for actual replay gates.

## Acceptance
- [ ] Success/failure artifacts inspect and replay without original source/compiler/mounts/preparation cache
- [ ] Missing/tampered scenario, module, boot/image payloads and incompatible installed identities fail closed
- [ ] Evidence and completeness reports match the per-incarnation replay contract
- [ ] Existing standalone and historical schema refusals remain intact

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
