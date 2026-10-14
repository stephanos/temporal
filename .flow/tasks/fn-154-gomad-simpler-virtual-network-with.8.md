---
satisfies: [R7, R10, R11]
---
# fn-154-gomad-simpler-virtual-network-with.8 Remove delivery history and hashes with coherent record and codec migration

## Description
Remove delivery history and hashes with coherent record and codec migration (R7, R10, R11). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/types.go`, `tools/gomad3sim/record.go`, `tools/gomad3sim/record_test.go`, `tools/gomad3sim/inspect.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go`, `tools/gomad3/simulation/schema/**`
**Touches:** [tools/gomad3sim/**, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go, tools/gomad3/simulation/schema/**, tools/gomad3/backends/native/execution/simulation_model_wire_generated.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/**, tools/gomad3/backends/native/generation/protocol/**, tools/gomad3/backends/wasm/wasi/network_v2.go, tools/gomad3/backends/wasm/wasi/network_v2_test.go, tools/gomad3/backends/wasm/wasi/broker_test.go, tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/simulation/provider.go, tools/gomad3/backends/wasm/simulation/provider_test.go, tools/gomad3/backends/wasm/installation/installation_test.go, tools/gomad3/backends/wasm/qualification/temporal_cluster_test.go, tools/gomad3/runner/internal/exploration/simulation_record.go, tools/gomad3/runner/internal/exploration/simulation_record_test.go, tools/gomad3/record/validation.go, tools/gomad3/record/record_test.go, tools/gomad3/artifact/publication_test.go, tools/gomad3/runner/minimize_operation_test.go, tools/gomad3/runner/replay_operation_test.go, tools/gomad3/runner/runner_test.go, tools/gomad3/cmd/gomad/internal/cli/cli_test.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/simulation/network/adapter.go, tools/gomad3/simulation/network/adapter_test.go, tools/gomad3/simulation/network/generated.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend_test.go]

### Approach

- Bind the concrete fn-109.14 codec inventory from task 2. Inventory every NetworkDeliver/NetworkDeliveries/NetworkDeliverySnapshot/Delivery/PayloadSHA256 network consumer before editing; separate volume hash uses from network uses. Include exact handwritten test/fixture fan-out in the task handover.
- Remove delivery count admission, fields, counters, inventories, per-delivery transitions and partition_drop. Retain byte snapshots and bounded metadata, listener/connection/history/frame ceilings. Do not delete explicit fault-match deliver strings.
- Update public/overlay record projections, validators, transition lanes/digests, inspection and the one generated semantic codec together. Bump affected identities and reject old versions before activation; retain separate volume/controller/artifact identities.
- Maintain validation before relevant mutation, complete replay consumption, nested model/controller/fault corruption rejection and structural operation mismatch controls. Mark obsolete network payload-field controls for deliberate replacement by the divided-guarantees task.
- Own all mechanical consumers and declared generated fan-out in this coherent migration. Stage the runtime/host edits as one frozen batch, validate old-format rejection and golden equality, and exclude unrelated generated drift.

### Investigation targets

**Required:**

- `tools/gomad3sim/types.go:124`
- `tools/gomad3sim/record.go:810`
- `tools/gomad3sim/record_test.go:271`
- `tools/gomad3sim/inspect.go:83`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:958`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go:147`
- `tools/gomad3/backends/native/generation/protocol/protocol.go:446`

### Quick commands

```bash
go test -tags test_dep ./tools/gomad3sim -run 'Test(ClusterRecord|DecodeClusterRecord|RuntimeNetwork|ValidateSpecRejectsReplay)'
make -C tools/gomad3 generate validate
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/backends/native/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'Test.*(Network|Fault)'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

### Revival trigger (2026-10-13)

The owner batch-first strategy admits the coherent connection/holding/expiry/dial/history source batch against integrated fn154.1 limits, fn154.3 timeout transport (c8dbb97c11) and fn154.2 generated NC2 config/vocabulary (c467490c78). Root's combined candidate passed portable simulator, protocol/projection, WASM reader/helper parity, full WASM simulation, all-platform architecture/purity, generated validation and affected lint. Source development may proceed while final independent batch review and exact native acceptance remain open. This does not waive completion dependencies, the early public-interface equivalence proof, any native gate or final current-source WASM qualification. No CI, PR, push or native qualification authority is revived. Use current backends/native paths above, one connection owner, and the accepted single generated semantic codec. Root serializes integration/identity regeneration and retains final lifecycle.

### Bound destructive-migration consumers (2026-10-13)

Task8's exact constructor, Record validation, exploration, artifact, Runner and CLI fixture fanout is included in Touches before edits. Update only fields/identities affected by the owned schema migration; do not change unrelated operation/CLI behavior or strict-reader guarantees. Preserve the existing 65,536 pending-operation/window ceiling as a separately named limit independent of deleted delivery admission. Keep World queued-event/work bounds explicit rather than deleting them or using an unbounded substitute. The accepted simulation/clock progress API remains sufficient: use the existing native timer/BrokerClock and WASM NextDeadline/Advance owners for autonomous expiry; no progress API expansion is approved.

### Native descriptor fixture migration binding (2026-10-13)

The exact native overlay descriptor_backend_test.go consumer is admitted above. Replace the retired maximumPendingChunks-based fill with a byte-capacity-derived control, preserving descriptor read/deadline/readiness behavior and adding no production descriptor interface. Native actual execution remains required; portable selected declarations do not prove this fixture executed.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] Network-only delivery/hash/count symbols and outcomes disappear from model/history consumers, while volume/generic-I/O hashes and explicit deliver fault matching remain.
- [ ] Config/record/snapshot/model/transition identities and single codec migrate coherently with old/malformed input rejection.
- [ ] Structural/order/topology/terminal negatives reject before their mutation, and full-consumption/nested-corruption controls remain.
- [ ] Complete removed-field consumer inventory, generated byte equality and focused record/codec tests pass.

## Done summary
portable.log passes retired-field/format refusal, malformed held/timeout transitions, network-limit identity, occupancy, unbound snapshots/unknown-transition preflight, actual-byte reexecution and length-mismatch rejection, nested controller/volume corruption. Generated matched consumer equality and current serialized corpus replay pass. Generic-I/O/volume hashes and explicit deliver-fault matching remain; completion SHIP checks removed-field inventory.

Acceptance uses immutable frozen960 execution evidence at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE; source manifest SHA256 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278, terminal stages.log PASS 2026-10-14T05:36:50Z. Source-equivalent current execution inputs are joined to exact gate-only947 source checks under MILESTONES' bounded-fix rule, not relabelled as current-HEAD guest execution. Authentic fn154 completion receipt .flow/tmp/completion-review-receipt-fn-154.json records SHIP (gpt-6.1-sol/high, same GPT family), base73c507fbf09318f36ae6ddf30ad8a917d801e868, session01a13834-e701-7cb3-b524-4777fb68ff23. Native execution stays unverified with fn128.4/fn149.2 per owner hibernation amendment. Full post-rename qualification remains mandatory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: portable.log passes retired-field/format refusal, malformed held/timeout transitions, network-limit identity, occupancy, unbound snapshots/unknown-transition preflight, actual-byte reexecution and length-mismatch rejection, nested controller/volume corruption. Generated matched consumer equality and current serialized corpus replay pass. Generic-I/O/volume hashes and explicit deliver-fault matching remain; completion SHIP checks removed-field inventory., Full frozen WASM qualification stages.log terminal PASS; source.digest retained unchanged, fn154 formal completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Corrected-source architecture/lint/validate-retry/clock/actionlint PASS /tmp/gomad-gate-integrated-*-20261014.log; native execution deferred fn128.4/fn149.2
- PRs: