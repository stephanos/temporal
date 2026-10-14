---
satisfies: [R1, R4, R9, R11]
---
# fn-154-gomad-simpler-virtual-network-with.4 Unify local connection mechanics behind a byte-bounded queue owner

## Description
Unify local connection mechanics behind a byte-bounded queue owner (R1, R9, R11). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go (new)`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go`, `tools/gomad3/backends/native/toolchain/version/version.json`, `tools/gomad3sim/network_handles_toolchain_test.go`
**Touches:** [tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection_test.go, tools/gomad3/simulation/network/**, tools/gomad3/backends/native/generation/protocol/network_connection_test.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go, tools/gomad3sim/network_handles_toolchain_test.go, tools/gomad3/backends/native/toolchain/version/version.json, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/**, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadlivecap/**]

### Approach

- Extract one effectful local connection owner in the existing gomadio package, with existing standalone/simulation domain effects at the real seam. Delete duplicated local read/write/deadline/close loops; keep processConn as transport.
- Move byte admission and retained occupancy into the shared owner. Store the admitted typed terminal reset reason on the pair; prove that clearing deadlines leaves an injected committed timeout terminal for both endpoints. .6 proves real autonomous expiry. Account for unread tails until consumption/discard and preserve partial results and global/local admission distinction.
- Migrate fn-155's final descriptor concrete casts and nonblocking/readiness calls to that owner. Preserve complete-operation direction locks, readiness generations, half-close, empty I/O, duplicate bind and error ordering.
- Before refactoring, retain public-interface outputs for local operations, short reads, half-close, deadlines, partial writes and descriptor readiness. Replay/diff the same operation sequences afterward, excluding only explicitly authorized capacity representation changes.
- Add the new overlay file to the allowlist and regenerate input-derived outputs. Preserve the existing record adapter until its later coherent history migration.

### Investigation targets

**Required:**

- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go:43`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go:24`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:139`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go:258`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go:17`
- `tools/gomad3sim/network_handles_toolchain_test.go:27`
- `tools/gomad3/backends/native/toolchain/version/version.json`

### Quick commands

```bash
(cd tools/gomad3 && .toolchain/bin/go test -tags test_dep internal/gomadio -run 'TestNetworkHandle')
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/backends/native/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetworkHandleOperationParity'
make -C tools/gomad3 generate validate
(cd tools/gomad3 && go test -tags test_dep ./backends/native/execution -run '^TestModelConformanceTCP$')
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

### Revival trigger (2026-10-13)

The owner batch-first strategy admits the coherent connection/holding/expiry/dial/history source batch against integrated fn154.1 limits, fn154.3 timeout transport (c8dbb97c11) and fn154.2 generated NC2 config/vocabulary (c467490c78). Root's combined candidate passed portable simulator, protocol/projection, WASM reader/helper parity, full WASM simulation, all-platform architecture/purity, generated validation and affected lint. Source development may proceed while final independent batch review and exact native acceptance remain open. This does not waive completion dependencies, the early public-interface equivalence proof, any native gate or final current-source WASM qualification. No CI, PR, push or native qualification authority is revived. Use current backends/native paths above, one connection owner, and the accepted single generated semantic codec. Root serializes integration/identity regeneration and retains final lifecycle.

## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] One local queue/deadline owner serves standalone, simulation and descriptor primitives; no duplicate local operation loop remains.
- [ ] Occupancy accounting includes pending short-read tails and releases each charge once.
- [ ] The old/new public-interface sequence pin matches for retained behavior, including 4 MiB each-direction partial-write boundary.
- [ ] Focused local/descriptor/handle tests, allowlist validation and existing TCP conformance pass on the required supported candidate.

## Done summary
Actual network_connection.go serves standalone, simulation and descriptor primitives. native-source.log TestNetworkConnectionPortableEquivalence pins public parity, four-MiB directional partial writes, unread tail occupancy/release, descriptor primitives, persistent terminal timeout and timer boundaries; generated overlay allowlist validation passes. Actual native handle/TCP execution remains deferred.

Acceptance uses immutable frozen960 execution evidence at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE; source manifest SHA256 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278, terminal stages.log PASS 2026-10-14T05:36:50Z. Source-equivalent current execution inputs are joined to exact gate-only947 source checks under MILESTONES' bounded-fix rule, not relabelled as current-HEAD guest execution. Authentic fn154 completion receipt .flow/tmp/completion-review-receipt-fn-154.json records SHIP (gpt-6.1-sol/high, same GPT family), base73c507fbf09318f36ae6ddf30ad8a917d801e868, session01a13834-e701-7cb3-b524-4777fb68ff23. Native execution stays unverified with fn128.4/fn149.2 per owner hibernation amendment. Full post-rename qualification remains mandatory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: Actual network_connection.go serves standalone, simulation and descriptor primitives. native-source.log TestNetworkConnectionPortableEquivalence pins public parity, four-MiB directional partial writes, unread tail occupancy/release, descriptor primitives, persistent terminal timeout and timer boundaries; generated overlay allowlist validation passes. Actual native handle/TCP execution remains deferred., Full frozen WASM qualification stages.log terminal PASS; source.digest retained unchanged, fn154 formal completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Corrected-source architecture/lint/validate-retry/clock/actionlint PASS /tmp/gomad-gate-integrated-*-20261014.log; native execution deferred fn128.4/fn149.2
- PRs: