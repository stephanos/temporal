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
- Authenticate and reopen the original executable scenario, all incarnation boot inputs and fn-176 full crash-selection operations/digest context from retained payloads; never infer context from selected IDs or observed transitions. Verify valid selections after source/cache removal and refuse malformed/tampered/missing contexts before execution; actual pending mismatch remains divergence.
- Use final fn-178 delayed-start, restart-hold and streamed-terminal behavior. Distinguish per-incarnation interrupted prefixes from exact complete tapes and keep fn-180 inspected-prefix reports distinct from ordinary artifact reproduction.

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
- [ ] Context-bearing storage-crash/restart artifacts replay from retained bytes; removed compile caches and changed external scenario/config do not affect replay
- [ ] Default absent-context/standalone payload behavior preserved and corrupt context or actual divergence cannot claim reproduction

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
