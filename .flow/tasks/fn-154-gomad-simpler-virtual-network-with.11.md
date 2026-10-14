---
satisfies: [R8, R10, R12]
---
# fn-154-gomad-simpler-virtual-network-with.11 Verify framed streams and the divided payload replay guarantees

## Description
Verify framed streams and the divided payload replay guarantees (R8, R10, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/network_framed_toolchain_test.go (new)`, `tools/gomad3sim/network_process_handles_toolchain_test.go`, `tools/gomad3sim/record_test.go`, `tools/gomad3sim/fault_toolchain_test.go`, `tools/gomad3/backends/native/execution/simulation_replay_toolchain_test.go` (new), `tools/gomad3/runner/internal/exploration/integration_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3sim/network_framed_toolchain_test.go, tools/gomad3sim/network_process_handles_toolchain_test.go, tools/gomad3sim/record_test.go, tools/gomad3sim/fault_toolchain_test.go, tools/gomad3/backends/native/execution/simulation_replay_toolchain_test.go, tools/gomad3/runner/internal/exploration/integration_test.go, tools/gomad3/internal/archcheck/integration_test.go, tools/gomad3/Makefile, tools/gomad3sim/testdata/simulation_replay/main.go]

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
- `tools/gomad3/backends/native/execution/io_entropy_toolchain_test.go:76`
- `tools/gomad3/runner/internal/exploration/integration_test.go:33`
- `tools/gomad3/Makefile:154`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go:468`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/backends/native/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetworkFramedPartition'
make -C tools/gomad3 test-simulation
(cd tools/gomad3 && go test -tags test_dep ./backends/native/execution -run '^TestSimulationReplayPayloadDivergence')
go test -tags test_dep ./tools/gomad3sim -run 'TestClusterRecord|Test.*Fault'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

### Provisional disjoint fixture admission (2026-10-13)

The owner batch-first strategy admits only the new network_framed_toolchain_test.go and backends/native/execution/simulation_replay_toolchain_test.go fixture surface while task8 migrates history. Existing consumers, record/fault tests and Runner/Make registrations remain serialized with the network worker/root. Develop against the approved R8/R10 behavior and stable public connection/profile interface; re-anchor and validate against final source before acceptance. Compile/source evidence never replaces actual supported-native execution. The full task scope, dependencies, both-backend framing, divided replay guarantees and all execution gates remain open.

### Source-bound replay fixture binding (2026-10-13)

Also admit `tools/gomad3sim/testdata/simulation_replay/main.go` as the replay fixture worker's third disjoint file. Preparing the entire gomad3sim test variant is intentionally refused for syscall imports; preserve that capability refusal. Prepare this production-import-only stream workload normally, then reuse the same prepared/source-bound binary for live simulation plus generic-I/O recording/replay. Never fabricate Prepared metadata or bypass source/capability/identity preflight. Framed process test names must join the existing Runner list, Make selector and selection guard at serialized integration.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

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
