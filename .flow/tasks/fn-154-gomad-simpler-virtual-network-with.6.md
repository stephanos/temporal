---
satisfies: [R4, R7]
---
# fn-154-gomad-simpler-virtual-network-with.6 Expire stalled connections through autonomous virtual-time work

## Description
Expire stalled connections through autonomous virtual-time work (R4). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3/backends/native/execution/simulation_time.go`, `tools/gomad3/simulation/clock/simulation_progress.go`, `tools/gomad3/simulation/clock/simulation_progress_test.go`, `tools/gomad3/runner/internal/exploration/integration_test.go`, `tools/gomad3/internal/archcheck/integration_test.go`, `tools/gomad3/Makefile`
`tools/gomad3/simulation/network/adapter.go`, `tools/gomad3/simulation/network/stall_test.go`, `tools/gomad3/simulation/network/generate/main.go`, `tools/gomad3/simulation/network/generate/main_test.go`, `tools/gomad3/simulation/network/generated.go`
**Touches:** [tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3/backends/native/execution/simulation_time.go, tools/gomad3/simulation/clock/simulation_progress.go, tools/gomad3/simulation/clock/simulation_progress_test.go, tools/gomad3/runner/internal/exploration/integration_test.go, tools/gomad3/internal/archcheck/integration_test.go, tools/gomad3/Makefile, tools/gomad3/simulation/network/adapter.go, tools/gomad3/simulation/network/stall_test.go, tools/gomad3/simulation/network/generate/main.go, tools/gomad3/simulation/network/generate/main_test.go, tools/gomad3/simulation/network/generated.go]

### Approach

- Arm bounded runtime-owned expiry for positive stalled occupancy. It must run after the writer returns even without a blocked reader/writer; integrate process virtual work through the existing arbiter. Consume the pair's reserved history slot for the one reset; ordinary operations cannot steal it. Release/cancel reservations by stable pair/epoch ownership. With a returned held writer and every unreserved slot consumed, expiry must still record/reset without active I/O. Preserve caller-deadline versus persistent reset classification.
- Bind connection identity, both incarnations and direction epoch. Recheck under serialized model mutation; stop/heal/close/crash/restart invalidate stale work without recording timeout.
- At strictly start+limit+1ns, commit one timeout reset, clear bytes/charge exactly once and wake both endpoints. Check both additions for overflow before affected mutation.
- Preserve seeded tie order and current terminal/progress/deadline precedence. Do not create host readiness ordering, timer tie sorting or a second process barrier.
- Any required progress-interface edit needs the concrete fn-109.15/.16 handoff and preservation evidence before changing it.

- Register every new process case in runner/internal/exploration/integration_test.go's test-name list and Make's Runner regex, and extend internal/archcheck/integration_test.go in this task before both-backend acceptance. Run the Runner-backed case without a skip. .10 audits the final complete selection; it is not a deferred executor prerequisite.

### Investigation targets

**Required:**

- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:777`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go:605`
- `tools/gomad3sim/network_toolchain_test.go:225`
- `tools/gomad3/backends/native/execution/simulation_time.go`
- `tools/gomad3/simulation/clock/simulation_progress.go:119`
- `tools/gomad3/simulation/clock/simulation_progress_test.go`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/backends/native/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetworkStall'
cd tools/gomad3 && go test -tags test_dep ./simulation/clock ./backends/native/execution -run 'Test.*(Simulation|Broker|Progress)'
```

Process acceptance command: `make -C tools/gomad3 test-simulation`, with newly registered Runner-backed selectors retained in the handover. A direct seeded-wrapper process skip supplies no acceptance.

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

### Revival trigger (2026-10-13)

The owner batch-first strategy admits the coherent connection/holding/expiry/dial/history source batch against integrated fn154.1 limits, fn154.3 timeout transport (c8dbb97c11) and fn154.2 generated NC2 config/vocabulary (c467490c78). Root's combined candidate passed portable simulator, protocol/projection, WASM reader/helper parity, full WASM simulation, all-platform architecture/purity, generated validation and affected lint. Source development may proceed while final independent batch review and exact native acceptance remain open. This does not waive completion dependencies, the early public-interface equivalence proof, any native gate or final current-source WASM qualification. No CI, PR, push or native qualification authority is revived. Use current backends/native paths above, one connection owner, and the accepted single generated semantic codec. Root serializes integration/identity regeneration and retains final lifecycle.
### Current expiry projection binding

Use the existing native runtime timer and WASM Model.NextDeadline/Advance owners. The exact projected adapter, stall tests and generator files above belong to this task within the already declared connection batch. No clock/progress interface edit is required or authorized by this binding. Preserve native seeded timer ownership and existing broker ordering; a no-op projected timer seam must be guarded by generator drift/shape tests and autonomous model expiry tests.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] New TestNetworkStallExpiry cases cover writer-returned/no-reader expiry, persistent both-end timeout and idle survival.
- [ ] New TestNetworkStallOrdering cases cover limit-minus/equal/plus-one-ns and both heal/expiry commit orders.
- [ ] Repeated partition, stopped/crashed/restarted incarnation and stale callback controls free bytes/work once and cannot reset replacements.
- [ ] Existing process progress/cancellation/late-response pins pass, with no new clock or response barrier.

## Done summary
portable.log passes autonomous reserved-slot expiry, idle/epoch invalidation, simultaneous pair expiry replay, accepted prefixes, reset reservations and clock-ceiling atomic refusal/boundaries. ordinary.log passes scheduled partition/heal/expiry ordering. Native expiry/ordering/progress/cancellation fixtures and process registrations are retained; actual patched execution remains deferred.

Acceptance uses immutable frozen960 execution evidence at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE; source manifest SHA256 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278, terminal stages.log PASS 2026-10-14T05:36:50Z. Source-equivalent current execution inputs are joined to exact gate-only947 source checks under MILESTONES' bounded-fix rule, not relabelled as current-HEAD guest execution. Authentic fn154 completion receipt .flow/tmp/completion-review-receipt-fn-154.json records SHIP (gpt-6.1-sol/high, same GPT family), base73c507fbf09318f36ae6ddf30ad8a917d801e868, session01a13834-e701-7cb3-b524-4777fb68ff23. Native execution stays unverified with fn128.4/fn149.2 per owner hibernation amendment. Full post-rename qualification remains mandatory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: portable.log passes autonomous reserved-slot expiry, idle/epoch invalidation, simultaneous pair expiry replay, accepted prefixes, reset reservations and clock-ceiling atomic refusal/boundaries. ordinary.log passes scheduled partition/heal/expiry ordering. Native expiry/ordering/progress/cancellation fixtures and process registrations are retained; actual patched execution remains deferred., Full frozen WASM qualification stages.log terminal PASS; source.digest retained unchanged, fn154 formal completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Corrected-source architecture/lint/validate-retry/clock/actionlint PASS /tmp/gomad-gate-integrated-*-20261014.log; native execution deferred fn128.4/fn149.2
- PRs: