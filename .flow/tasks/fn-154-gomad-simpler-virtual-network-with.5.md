---
satisfies: [R1, R2, R3, R7]
---
# fn-154-gomad-simpler-virtual-network-with.5 Hold partitioned bytes and release FIFO through effective topology changes

## Description
Hold partitioned bytes and release FIFO through effective topology changes (R1, R2, R3, R7). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3sim/fault_toolchain_test.go`, `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go`, `tools/gomad3/simulation_gate_selection_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3sim/fault_toolchain_test.go, tools/gomad3/runner/internal/execution/simulation_root_integration_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/Makefile]

### Approach

- Replace successful partition_drop paths with retained directed FIFO bytes. Treat elapsed readiness without actual receipt as undelivered, while preserving received readable prefixes. Atomically admit the ordinary write/topology transition and one timeout reservation for each newly stalled pair before bytes/topology change. Group reservation failure rejects every mutation. Preserve received bytes and additive record shapes admitted by .2.
- Apply effective partition/heal/delay changes to affected connection state, wake both connection and descriptor waiters, and retain existing all-or-nothing group validation.
- On effective heal, apply normal delay to held data. Queue fresh writes behind it; on repartition hold the remaining suffix and reapply delay at next heal. Redundant topology actions cannot restart intervals or add delay.
- Keep one topology cause record; no per-release history. Tests observe actual endpoint bytes and blocked/progress outcomes instead of internal queue layout.

- Register every new process case in simulation_root_integration_test.go's test-name list and Make's Runner regex, and extend simulation_gate_selection_test.go in this task before both-backend acceptance. Run the Runner-backed case without a skip. .10 audits the final complete selection; it is not a deferred executor prerequisite.

### Investigation targets

**Required:**

- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go:118`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:621`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:737`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go:73`
- `tools/gomad3sim/network_toolchain_test.go:116`
- `tools/gomad3sim/fault_toolchain_test.go:217`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetwork|Test.*Fault'
```

Process acceptance command: `make -C tools/gomad3 test-simulation`, with newly registered Runner-backed selectors retained in the handover. A direct seeded-wrapper process skip supplies no acceptance.

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] Held-before and held-after cases have complete FIFO outcomes, no stream gap and no partition_drop success.
- [ ] Delayed heal, fresh-write overtaking, repeated partition and redundant heal/disable controls pass.
- [ ] Asymmetric reverse-direction and grouped rejected-mutation controls retain their pins.
- [ ] Topology changes wake local/descriptor waiters without making undelivered bytes readable.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
