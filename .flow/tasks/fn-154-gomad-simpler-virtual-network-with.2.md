---
satisfies: [R6, R7, R11]
---
# fn-154-gomad-simpler-virtual-network-with.2 Propagate limits and admit held/timeout vocabulary in the generated network codec

## Description
Propagate explicit limits and admit held/timeout vocabulary through fn109.14's single generated network codec (R6, R7, R11). Root owns integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/simulation/schema/networkwire.json`, `tools/gomad3/simulation/schema/networkwire.go.tmpl`, `tools/gomad3/backends/native/generation/protocol/network.go`, `tools/gomad3/backends/native/generation/protocol/*test.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go`, `tools/gomad3sim/runtime_network_wire.go`, `tools/gomad3sim/runtime_network_wire_test.go`, `tools/gomad3sim/types.go`, `tools/gomad3sim/record.go`, `tools/gomad3sim/record_test.go` and existing activation/config callers needed to supply the two explicit limits.
**Touches:** [tools/gomad3/simulation/schema/networkwire*, tools/gomad3/backends/native/generation/protocol/**, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network*, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/bootstrap*, tools/gomad3sim/runtime_network*, tools/gomad3sim/types.go, tools/gomad3sim/record*, tools/gomad3sim/runtime.go, tools/gomad3sim/runtime_backend.go, tools/gomad3/simulation/network/**, tools/gomad3/backends/wasm/wasi/network_v2.go, tools/gomad3/backends/wasm/wasi/*_test.go, tools/gomad3/backends/wasm/simulation/provider.go, tools/gomad3/backends/wasm/simulation/*_test.go, tools/gomad3/backends/wasm/qualification/temporal_cluster_test.go, tools/gomad3/backends/wasm/installation/installation_test.go]

### Approach

- The narrow fn109.14 admission gate is satisfied: Flow reports done, integrated commit 0b0971431c owns the single semantic definition, paired codecs, cross-consumer malformed vectors, drift tests and source review. Exact inputs are networkwire.json/networkwire.go.tmpl plus generation/protocol/network.go; outputs are gomad3sim/runtime_network_wire.go and native overlay internal/gomadio/simulation_wire.go. Do not write a second codec or add a whole-fn109 dependency.
- Consume fn154.1 public limits and fn154.3 stable timeout error transport. Propagate NetworkConnectionBytes and NetworkStallNanos through both activation paths. Reject absent/zero/invalid/overflowing values and old config identity before activation; serialize changes to generated outputs.
- Admit additive held-write and timeout outcomes, connection-reset kind and exact endpoint/lane validation in matched host/overlay consumers before runtime producers. Preserve every currently produced delivery shape until task8. Keep semantic network identity distinct from IPC transport identity.
- Retain host/overlay golden equality, synthetic valid records, malformed/unknown variants, old-version and overflow negatives. Regenerate the shared network projection from accepted native inputs; its output is not another format owner.
- Bind discovered activation call sites before editing if outside the declared inventory. Do not implement partition timing, expiry, queue simplification or destructive delivery/hash removal in this task.
- Bound WASM consumers (2026-10-13): network_v2.go must validate both new fields before its projected model activates. Broker already forwards its whole Limits. Provider.prepare must reject a selected nonnil NetworkV2 boot whose two new limits differ from the simulation Spec, including every selected node; never overwrite identity-bound boot inputs silently. Network-free boots remain valid. Existing WASI/simulation fixtures must supply explicit new fields, with only necessary literal changes and focused invalid/mismatch coverage. Cluster.go/process_node.go already share the single encoder and need no changes.
- Exact additional test-only write binding: `tools/gomad3/backends/wasm/qualification/temporal_cluster_test.go` and `tools/gomad3/backends/wasm/installation/installation_test.go` each construct the same projected Limits. Permit only required explicit 4 MiB/10-second fixture values in those existing constructors; preserve task20 strict-reader tests and do not change their production readers.

### Quick commands

```bash
make -C tools/gomad3 generate validate
go test -tags test_dep ./tools/gomad3sim -run 'TestRuntimeNetwork|TestClusterRecord|TestDecodeClusterRecord'
(cd tools/gomad3 && go test -tags test_dep ./backends/native/generation/protocol ./simulation/network ./simulation/network/generate)
```

Use stock Go1.27.1 for portable source checks on linux/arm64. Native execution remains deferred to fn128/fn149; no portable result substitutes for it. Final current-candidate WASM acceptance belongs to the coherent rotation batch.

### Revival trigger (2026-10-13)

The WASM shared-owner rotation batch has accepted fn109.14's generated semantic codec and fn154.1's explicit limits; fn154.3 timeout propagation is integrated and undergoing source review. Work the declared generated config/vocabulary surface next, consuming those interfaces and preserving existing delivery producers until task8. This revives source implementation only, with no native qualification, CI, PR or push authority.
## Acceptance
- [ ] Fn-109.14 source acceptance and exact generated input/output inventory are retained before dispatch.
- [ ] Both activation paths receive identical byte/stall configuration, and additive held/timeout synthetic records pass matched endpoint/lane validation before their runtime producers land.
- [ ] Generated host/overlay golden vectors match and malformed/old config fails before activation.
- [ ] Generate/validate and focused codec source checks pass without a second codec or unrelated generated drift.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
