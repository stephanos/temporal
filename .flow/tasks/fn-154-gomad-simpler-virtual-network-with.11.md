---
satisfies: [R8, R10, R12]
---
# fn-154-gomad-simpler-virtual-network-with.11 Verify framed streams and the divided payload replay guarantees

## Description
Verify framed streams and the divided payload replay guarantees (R8, R10, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/network_framed_toolchain_test.go (new)`, `tools/gomad3sim/network_process_handles_toolchain_test.go`, `tools/gomad3sim/record_test.go`, `tools/gomad3sim/fault_toolchain_test.go`, `tools/gomad3/runner/internal/execution/simulation_replay_toolchain_test.go` (new), `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3sim/network_framed_toolchain_test.go, tools/gomad3sim/network_process_handles_toolchain_test.go, tools/gomad3sim/record_test.go, tools/gomad3sim/fault_toolchain_test.go, tools/gomad3/runner/internal/execution/simulation_replay_toolchain_test.go, tools/gomad3/runner/internal/execution/simulation_root_integration_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/Makefile]

### Approach

- Add length-framed stream workloads for short/long and repeated partitions in both backends. Assert actual complete frames or timeout; retain caller-deadline and explicit crash/stop outcomes without labelling those framing gaps.
- Replace obsolete PayloadSHA256-field assertions with a composed simulation plus generic-I/O replay test in the new simulation_replay_toolchain_test.go. Follow IOCapability/Expected/ReplayDivergence at io_entropy_toolchain_test.go:76, combining it with the existing simulation capability. Use the configured bound and mismatch ordinal. Pair it with the direct network-only waiver control; coordinator-only tests are not a live replay harness.
- Retain network structure/topology/write-length/order/reset/full-consumption and nested fault/controller corruption controls; rehashing the outer record must not excuse corrupt nested evidence.
- Register new process/replay cases in the Runner list, Make selection and selection guard. This is mechanical registration after task 10, not another barrier or replay layer.

### Investigation targets

**Required:**

- `tools/gomad3sim/network_process_handles_toolchain_test.go:338`
- `tools/gomad3sim/record_test.go:271`
- `tools/gomad3sim/fault_toolchain_test.go:76`
- `tools/gomad3/runner/internal/execution/io_entropy_toolchain_test.go:76`
- `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go:33`
- `tools/gomad3/Makefile:154`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go:468`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetworkFramedPartition'
make -C tools/gomad3 test-simulation
(cd tools/gomad3 && go test -tags test_dep ./runner/internal/execution -run '^TestSimulationReplayPayloadDivergence')
go test -tags test_dep ./tools/gomad3sim -run 'TestClusterRecord|Test.*Fault'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] New TestNetworkFramedPartition cases run in both backends and yield complete frames or timeout, never a partition-created stream-gap error.
- [ ] Composed-profile payload mismatch still rejects at the retained generic-I/O ordinal/bound; direct network-only waiver is explicit and tested.
- [ ] Structural/terminal/full-consumption and nested fault/controller corruption negatives remain effective.
- [ ] All new process cases actually execute through registered gates.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
