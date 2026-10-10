---
satisfies: [R1, R2, R3, R4, R9, R11, R12]
---
# fn-154-gomad-simpler-virtual-network-with.9 Pin shared-connection and in-process fault conformance

## Description
Pin shared-connection and in-process fault conformance (R1, R2, R3, R4, R9, R11, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/network_handles_toolchain_test.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3sim/fault_toolchain_test.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go`, `tools/gomad3/runner/internal/execution/model_conformance_test.go`
**Touches:** [tools/gomad3sim/network_handles_toolchain_test.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3sim/fault_toolchain_test.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go, tools/gomad3/runner/internal/execution/model_conformance_test.go]

### Approach

- Complete the public-interface operation sequence pin for standalone and in-process domains. Add missing controls only; use preserved conformance rather than implementation-shape assertions.
- Cover short-read tail charge, local byte-full blocking, global byte error, oversize partial writes, deadline then reuse, half-close/empty I/O and selected descriptor readiness.
- Record named evidence for every existing fault/outcome in R12, including graceful queued-byte discard, crash/restart and asymmetric/group topology.
- Retain the stock-TCP comparison and declared differences. Prove stale model/handle/incarnation rejection before mutation and nested/extra/unused/reordered fault negatives.

### Investigation targets

**Required:**

- `tools/gomad3sim/network_handles_toolchain_test.go:27`
- `tools/gomad3sim/network_toolchain_test.go:335`
- `tools/gomad3sim/network_toolchain_test.go:538`
- `tools/gomad3sim/fault_toolchain_test.go:76`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go:17`
- `tools/gomad3/runner/internal/execution/model_conformance_test.go:110`

### Quick commands

```bash
cd tools/gomad3 && .toolchain/bin/go test -tags test_dep internal/gomadio -run 'TestNetworkHandle'
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetwork|Test.*Fault'
cd tools/gomad3 && go test -tags test_dep ./runner/internal/execution -run '^TestModelConformanceTCP$'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] Local and in-process operation-sequence equivalence is pinned through the public interface.
- [ ] Each byte/caller-deadline/revocation boundary has a concrete positive and negative case.
- [ ] Every R12 in-process fault/outcome has named evidence; explicit fault matching remains valid.
- [ ] Existing TCP conformance and retained validation-before-mutation controls pass on the supported candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
