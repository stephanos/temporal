---
satisfies: [R10]
---
# fn-155-gomad-syscall-level-io-boundary-from.4 Serve multi-node simulation traffic through virtual descriptors in both backends

## Description
Back virtual descriptors with the simulation network for multi-node runs: per-node addresses, incarnations and revocation in the in-process backend, and forwarding to the coordinator in the process backend.

**Size:** M
**Files:** overlay descriptor layer, `internal/gomadio/simulation_network.go`, `simulation_handles.go`, `process_network.go`, `process_commands.go`, `tools/gomad3sim/runtime_network.go`; multi-node workload/tests in `tools/gomad3sim` and `runner/internal/execution`.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/**, tools/gomad3sim/**, tools/gomad3/runner/internal/execution/**]

### Approach
- In-process backend: key descriptors by the calling goroutine's network domain (`CurrentNetworkDomain`) and map to `simulationConn`/listeners; revocation of an incarnation fails its descriptors with the existing outcome.
- Process backend: descriptor operations forward through the existing process RPC (`process_network.go:155-289`) to the coordinator's model; readiness returns through the same channel.
- Reuse the existing simulation tests as the behavior pin (`tools/gomad3sim/*network*_toolchain_test.go`) and add boundary-on variants.
- Use .2's single TCP stand-aside switch; do not add a second one for simulation networks.
- Coordinate with fn-154 if it has started: build on its connection model rather than duplicating it.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:398-857`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go`
- `tools/gomad3sim/runtime_network.go`
- `tools/gomad3sim/network_toolchain_test.go`

## Acceptance
- [ ] Existing simulation network tests pass with the boundary off and, as boundary-on variants, with it on, in both backends.
- [ ] A crashed/stopped node incarnation's descriptors fail with the existing revocation outcome; dials to unknown addresses are refused.
- [ ] A multi-node gRPC workload runs with the boundary on and its network adapters (gRPC, `x/net`, `sockaddr` if selected) excluded, same seed twice → identical traces and transcripts.
- [ ] `make -C tools/gomad3 test-simulation overlay-test` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
