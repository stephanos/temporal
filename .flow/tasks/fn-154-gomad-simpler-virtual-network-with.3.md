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
Completed stable stall-timeout identity and Timeout classification across local/process/net wrappers and generated error transport. Supplied stall timeout stays distinct from caller deadlines; partial counts, frame bounds, stale handles and existing code precedence are preserved. Persistent post-reset state is accepted separately by fn154.4/.6, not claimed by this adapter task.

Source implementation and authentic three-draw formal SHIP are retained in the source checkpoint above. Final fn154 completion review is SHIP in .flow/tmp/completion-review-receipt-fn-154.json (base 73c507fbf09318f36ae6ddf30ad8a917d801e868, session 01a13834-e701-7cb3-b524-4777fb68ff23). Review and writer share the GPT family. Frozen960 native-source.log records TestNetworkTimeoutPortableSourceAdapters, TestNetworkWireConsumers, TestNetworkSemanticCodecGenerated and generated schema controls PASS. Full frozen WASM qualification ended PASS; source manifest 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278 is retained unchanged. Gate-only947 changes are joined to corrected-source architecture/lint/validation/workflow checks under MILESTONES' existing bounded-fix rule; no new guest run is attributed to changed source.

The 2026-10-14 native hibernation amendment supersedes the older checkpoint's native completion blocker. Patched-runtime adapter execution is not executed or passed; it remains deferred under fn128/fn149. Current source/portable/WASM acceptance is complete.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c8dbb97c11, 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: Frozen960 native-source.log: TestNetworkTimeoutPortableSourceAdapters, TestNetworkWireConsumers, TestNetworkSemanticCodecGenerated and schema controls PASS, Full frozen WASM qualification terminal PASS: /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE/stages.log, Corrected-source947 architecture, lint, pinned-Go validate and actionlint logs PASS; exact gate-only delta preserves execution inputs, Authentic source SHIP /tmp/impl-review-receipt-657da2bc4466-fn-154-gomad-simpler-virtual-network-with.3.json; fn154 completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Native patched adapter execution explicitly deferred fn128/fn149; portable checks supply no native pass
- PRs:
## Revival trigger (2026-10-13)

The WASM rotation batch reaches the independent timeout/error surface while fn109.14 finishes the generated semantic codec predecessor. Implement a stable stall-timeout error through the existing modelwire error owner, local/process/net wrappers, and selected descriptor consumer, preserving supplied-error identity and partial counts. Current paths are under `tools/gomad3/backends/native/{generation,toolchain}`. Coordinate provisional generated interfaces with fn109.14 and serialize integration of shared generator outputs; do not replace its single codec definition or implement connection expiry here. Fn154.1's explicit limit implementation is source-reviewed, while actual persistent resets, per-operation arithmetic and final native/WASM gates retain their later owners. Local forced claim bypasses only unpushed parent-chain admission; no native qualification, CI, PR, or push authority is revived.
