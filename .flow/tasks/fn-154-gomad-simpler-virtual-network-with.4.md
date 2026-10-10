---
satisfies: [R1, R4, R9, R11]
---
# fn-154-gomad-simpler-virtual-network-with.4 Unify local connection mechanics behind a byte-bounded queue owner

## Description
Unify local connection mechanics behind a byte-bounded queue owner (R1, R9, R11). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go (new)`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go`, `tools/gomad3/toolchain/version/version.json`, `tools/gomad3sim/network_handles_toolchain_test.go`
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go, tools/gomad3sim/network_handles_toolchain_test.go, tools/gomad3/toolchain/version/version.json, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadlivecap/**]

### Approach

- Extract one effectful local connection owner in the existing gomadio package, with existing standalone/simulation domain effects at the real seam. Delete duplicated local read/write/deadline/close loops; keep processConn as transport.
- Move byte admission and retained occupancy into the shared owner. Store the admitted typed terminal reset reason on the pair; prove that clearing deadlines leaves an injected committed timeout terminal for both endpoints. .6 proves real autonomous expiry. Account for unread tails until consumption/discard and preserve partial results and global/local admission distinction.
- Migrate fn-155's final descriptor concrete casts and nonblocking/readiness calls to that owner. Preserve complete-operation direction locks, readiness generations, half-close, empty I/O, duplicate bind and error ordering.
- Before refactoring, retain public-interface outputs for local operations, short reads, half-close, deadlines, partial writes and descriptor readiness. Replay/diff the same operation sequences afterward, excluding only explicitly authorized capacity representation changes.
- Add the new overlay file to the allowlist and regenerate input-derived outputs. Preserve the existing record adapter until its later coherent history migration.

### Investigation targets

**Required:**

- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go:43`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go:24`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:139`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go:258`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go:17`
- `tools/gomad3sim/network_handles_toolchain_test.go:27`
- `tools/gomad3/toolchain/version/version.json`

### Quick commands

```bash
(cd tools/gomad3 && .toolchain/bin/go test -tags test_dep internal/gomadio -run 'TestNetworkHandle')
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetworkHandleOperationParity'
make -C tools/gomad3 generate validate
(cd tools/gomad3 && go test -tags test_dep ./runner/internal/execution -run '^TestModelConformanceTCP$')
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] One local queue/deadline owner serves standalone, simulation and descriptor primitives; no duplicate local operation loop remains.
- [ ] Occupancy accounting includes pending short-read tails and releases each charge once.
- [ ] The old/new public-interface sequence pin matches for retained behavior, including 4 MiB each-direction partial-write boundary.
- [ ] Focused local/descriptor/handle tests, allowlist validation and existing TCP conformance pass on the required supported candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
