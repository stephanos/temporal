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


### Source acceptance checkpoint (2026-10-13)

Integrated implementation c8dbb97c11 and combined model pin eda3f1b007 are source accepted. Root executed full protocol/network/projection tests, all-platform architecture/purity, generated and compatibility validation, scoped lint with fixes disabled and identity check: exit 0. Three fresh-context gpt-6.1-sol/high draws (same GPT family as implementation) returned SHIP; finalized receipt /tmp/impl-review-receipt-657da2bc4466-fn-154-gomad-simpler-virtual-network-with.3.json, rid cf74bdfaa7a34cd88cbeba68acab3b52. First dispatch 8b2b797e... was not accepted because tracker-only HEAD movement violated the finalizer freeze; the second dispatch was frozen and accepted.

The exact patched-runtime adapter selector in this task's Quick commands has NOT executed: this host is linux/arm64 and has no supported patched native toolchain. The portable actual-source fixture is not that native pass. Acceptance remains open here; source-review SHIP is not Flow done. Persistent reset/timing producers remain .4/.6/.7. Task2 is admitted against this integrated interface under the owner batch-first strategy without waiving dependencies at completion or native requirements. Final current-source WASM qualification also remains outstanding.

stage: impl-review - ran (model: gpt-6.1-sol high)
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

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

## Revival trigger (2026-10-13)

The WASM rotation batch reaches the independent timeout/error surface while fn109.14 finishes the generated semantic codec predecessor. Implement a stable stall-timeout error through the existing modelwire error owner, local/process/net wrappers, and selected descriptor consumer, preserving supplied-error identity and partial counts. Current paths are under `tools/gomad3/backends/native/{generation,toolchain}`. Coordinate provisional generated interfaces with fn109.14 and serialize integration of shared generator outputs; do not replace its single codec definition or implement connection expiry here. Fn154.1's explicit limit implementation is source-reviewed, while actual persistent resets, per-operation arithmetic and final native/WASM gates retain their later owners. Local forced claim bypasses only unpushed parent-chain admission; no native qualification, CI, PR, or push authority is revived.
