---
satisfies: [R6, R7, R11]
---
# fn-154-gomad-simpler-virtual-network-with.2 Propagate limits and admit held/timeout vocabulary in the generated network codec

## Description
Propagate explicit limits through the admitted generated network codec (R6, R11). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/simulation/schema/**`, `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3sim/runtime_network_wire.go`, `tools/gomad3sim/runtime_network_wire_test.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go`, `tools/gomad3sim/types.go`, `tools/gomad3sim/record.go`, `tools/gomad3sim/record_test.go`
**Touches:** [tools/gomad3/simulation/schema/**, tools/gomad3/runner/internal/execution/simulation_model_wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadmodelwire/**, tools/gomad3/internal/gomadtool/generation/protocol/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go, tools/gomad3sim/runtime_network_wire*, tools/gomad3sim/types.go, tools/gomad3sim/record.go, tools/gomad3sim/record_test.go]

### Approach

- Admission gate: root must retain source acceptance of fn-109.14's single generated semantic codec, substitute its exact schema/template/outputs into Files and Touches, and bind that inventory before worker dispatch. Flow cannot encode this narrow cross-spec task edge; do not add a whole-fn-109 dependency or write another codec.
- Modify the admitted config schema/template and generator inventory, then regenerate every declared host/overlay consumer. Today's handwritten wire files are investigation anchors, not a license to keep them as separate format owners.
- Consume integrated public limits from .1 and stable timeout transport from .3. Propagate fields through both activation paths. Admit additive held-write/timeout outcome and connection-reset kind, endpoint requirements, matched host/overlay lanes and the admitted codec's vectors/version before .5/.6 producers. Round-trip synthetic valid records and reject unknown/malformed variants. Preserve every still-produced delivery shape until .8. Keep IPC transport identity distinct from semantic network versions.
- Retain host/overlay golden equality and malformed/old-version/overflow negatives. Inventory generated fan-out and serialize generation with other owners.

### Investigation targets

**Required:**

- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:446`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:139`
- `tools/gomad3sim/types.go:124`
- `tools/gomad3sim/record.go:1078`
- `tools/gomad3sim/record_test.go:271`
- `tools/gomad3sim/runtime_network_wire_test.go`
- `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.14.md`

### Quick commands

```bash
make -C tools/gomad3 generate validate
go test -tags test_dep ./tools/gomad3sim -run 'TestRuntimeNetwork|TestClusterRecord|TestDecodeClusterRecord'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

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
