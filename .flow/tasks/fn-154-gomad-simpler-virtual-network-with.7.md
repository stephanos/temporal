---
satisfies: [R5]
---
# fn-154-gomad-simpler-virtual-network-with.7 Bound pending partitioned dials with independent virtual stall intervals

## Description
Bound pending partitioned dials with independent virtual stall intervals (R5). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3sim/network_process_handles_toolchain_test.go`, `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go`, `tools/gomad3/simulation_gate_selection_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3sim/network_process_handles_toolchain_test.go, tools/gomad3/runner/internal/execution/simulation_root_integration_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/Makefile]

### Approach

- Reuse the admitted strict virtual expiry mechanism for a pending disabled-link dial, without creating an established connection. Start/reset intervals only at effective directed topology changes. Atomically reserve one timeout-outcome slot before a disabled-link dial waits, within the existing history ceiling. Consume or release it on timeout/heal/caller completion without leaks or replacement-epoch release; capacity failure occurs before pending work is admitted.
- Keep ctx.Err before topology/listener admission, preserve caller deadline separately, and choose heal/expiry through existing model order with stale-epoch rejection.
- Preserve refused listener, stale endpoint, backlog/listener/connection capacity and actual process cancellation support. Do not add unsupported post-IPC cancellation guarantees.
- Add both-backend heal/deadline/stall, redundant-disable, repartition and exact-threshold tests, including partial setup cleanup and no leaked connection identity.

- Register every new process case in simulation_root_integration_test.go's test-name list and Make's Runner regex, and extend simulation_gate_selection_test.go in this task before both-backend acceptance. Run the Runner-backed case without a skip. .10 audits the final complete selection; it is not a deferred executor prerequisite.

### Investigation targets

**Required:**

- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:398`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go:73`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go:195`
- `tools/gomad3sim/network_toolchain_test.go:18`
- `tools/gomad3sim/network_process_handles_toolchain_test.go:19`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'Test.*(Dial|NetworkPartitionedDialStall)'
```

Process acceptance command: `make -C tools/gomad3 test-simulation`, with newly registered Runner-backed selectors retained in the handover. A direct seeded-wrapper process skip supplies no acceptance.

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] New TestNetworkPartitionedDialStall cases prove heal, caller deadline and internal timeout independently in both backends.
- [ ] Completed context wins before admission; tie/redundant-disable/repartition cases follow the parent ordering contract.
- [ ] Refusal/stale/capacity and supported cancellation outcomes retain existing pins.
- [ ] Pending dial work and handles clean up on every terminal path.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
