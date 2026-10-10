---
satisfies: [R4, R5]
---
# fn-154-gomad-simpler-virtual-network-with.3 Preserve persistent stall timeout identity through local and process adapters

## Description
Preserve persistent stall timeout identity through local and process adapters (R4, R5). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_commands.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_commands_test.go`, `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go`, `tools/gomad3/simulation/schema/modelwire.json`, `tools/gomad3/simulation/schema/modelwire.go.tmpl`, `tools/gomad3/toolchain/runtime/overlay/src/net/gomad.go`, `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go`, `tools/gomad3/internal/gomadtool/generation/protocol/*test.go`
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_commands*, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go, tools/gomad3/simulation/schema/modelwire*, tools/gomad3/runner/internal/execution/simulation_model_wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadmodelwire/**, tools/gomad3/toolchain/runtime/overlay/src/net/gomad.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go, tools/gomad3/internal/gomadtool/generation/protocol/**]

### Approach

- This source owner is independent of .1's limits. Bind the then-current fn-155 error/status consumers before dispatch and serialize generation/gates. Add a stable stall-timeout sentinel/type at the existing error owner with errors.Is identity and Timeout classification. Prove supplied-error adapter behavior here; .4/.6 own actual persistent connection-state/reset proof.
- Extend protocol.go's closed schema Errors struct and strict 0-through-21 code validation, the modelwire schema/template and both encode/decode cases. Update the template's ErrorCapacity upper-code check to admit only the new declared code, retaining unknown-code rejection. Regenerate its four declared outputs; only changed bytes enter the commit. Preserve partial counts, bounds and existing error precedence.
- Propagate classification through processConn and net wrappers. Inspect the admitted fn-155 descriptor return path; add its exact error consumer to the write inventory if required before editing, preserving existing fd-generation checks.
- Regenerate declared consumers and test stable round trips and deadline/closed/stale/capacity controls.

### Investigation targets

**Required:**

- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_commands.go:116`
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:292`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go:245`
- `tools/gomad3/simulation/schema/modelwire.json`
- `tools/gomad3/simulation/schema/modelwire.go.tmpl`
- `tools/gomad3/toolchain/runtime/overlay/src/net/gomad.go`

### Quick commands

```bash
(cd tools/gomad3 && .toolchain/bin/go test -tags test_dep internal/gomadio -run 'Test.*(Process|Error|NetworkHandle)')
make -C tools/gomad3 generate validate
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] Stall timeout retains stable identity and Timeout classification through each selected adapter.
- [ ] Supplied timeout and caller-deadline errors remain distinguishable through error transport/wrappers; actual post-reset deadline/read/write persistence belongs to .4/.6.
- [ ] Partial response counts, frame bounds and existing error-code precedence retain their pins.
- [ ] Focused error/transport checks and generated validation pass; native adapter execution stays open until run.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
