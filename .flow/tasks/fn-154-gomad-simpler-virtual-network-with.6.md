---
satisfies: [R4, R7]
---
# fn-154-gomad-simpler-virtual-network-with.6 Expire stalled connections through autonomous virtual-time work

## Description
Expire stalled connections through autonomous virtual-time work (R4). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3/runner/internal/execution/simulation_time.go`, `tools/gomad3/runner/internal/execution/simulation_progress.go`, `tools/gomad3/runner/internal/execution/simulation_progress_test.go`, `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go`, `tools/gomad3/simulation_gate_selection_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3/runner/internal/execution/simulation_time.go, tools/gomad3/runner/internal/execution/simulation_progress.go, tools/gomad3/runner/internal/execution/simulation_progress_test.go, tools/gomad3/runner/internal/execution/simulation_root_integration_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/Makefile]

### Approach

- Arm bounded runtime-owned expiry for positive stalled occupancy. It must run after the writer returns even without a blocked reader/writer; integrate process virtual work through the existing arbiter. Consume the pair's reserved history slot for the one reset; ordinary operations cannot steal it. Release/cancel reservations by stable pair/epoch ownership. With a returned held writer and every unreserved slot consumed, expiry must still record/reset without active I/O. Preserve caller-deadline versus persistent reset classification.
- Bind connection identity, both incarnations and direction epoch. Recheck under serialized model mutation; stop/heal/close/crash/restart invalidate stale work without recording timeout.
- At strictly start+limit+1ns, commit one timeout reset, clear bytes/charge exactly once and wake both endpoints. Check both additions for overflow before affected mutation.
- Preserve seeded tie order and current terminal/progress/deadline precedence. Do not create host readiness ordering, timer tie sorting or a second process barrier.
- Any required progress-interface edit needs the concrete fn-109.15/.16 handoff and preservation evidence before changing it.

- Register every new process case in simulation_root_integration_test.go's test-name list and Make's Runner regex, and extend simulation_gate_selection_test.go in this task before both-backend acceptance. Run the Runner-backed case without a skip. .10 audits the final complete selection; it is not a deferred executor prerequisite.

### Investigation targets

**Required:**

- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:777`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go:605`
- `tools/gomad3sim/network_toolchain_test.go:225`
- `tools/gomad3/runner/internal/execution/simulation_time.go`
- `tools/gomad3/runner/internal/execution/simulation_progress.go:119`
- `tools/gomad3/runner/internal/execution/simulation_progress_test.go`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetworkStall'
cd tools/gomad3 && go test -tags test_dep ./runner/internal/execution -run 'Test.*Simulation.*Progress'
```

Process acceptance command: `make -C tools/gomad3 test-simulation`, with newly registered Runner-backed selectors retained in the handover. A direct seeded-wrapper process skip supplies no acceptance.

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] New TestNetworkStallExpiry cases cover writer-returned/no-reader expiry, persistent both-end timeout and idle survival.
- [ ] New TestNetworkStallOrdering cases cover limit-minus/equal/plus-one-ns and both heal/expiry commit orders.
- [ ] Repeated partition, stopped/crashed/restarted incarnation and stale callback controls free bytes/work once and cannot reset replacements.
- [ ] Existing process progress/cancellation/late-response pins pass, with no new clock or response barrier.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
