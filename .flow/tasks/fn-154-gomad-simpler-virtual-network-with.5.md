---
satisfies: [R1, R2, R3, R7]
---
# fn-154-gomad-simpler-virtual-network-with.5 Hold partitioned bytes and release FIFO through effective topology changes

## Description
Hold partitioned bytes and release FIFO through effective topology changes (R1, R2, R3, R7). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3sim/fault_toolchain_test.go`, `tools/gomad3/runner/internal/exploration/integration_test.go`, `tools/gomad3/internal/archcheck/integration_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3sim/fault_toolchain_test.go, tools/gomad3/runner/internal/exploration/integration_test.go, tools/gomad3/internal/archcheck/integration_test.go, tools/gomad3/Makefile]

### Approach

- Replace successful partition_drop paths with retained directed FIFO bytes. Treat elapsed readiness without actual receipt as undelivered, while preserving received readable prefixes. Atomically admit the ordinary write/topology transition and one timeout reservation for each newly stalled pair before bytes/topology change. Group reservation failure rejects every mutation. Preserve received bytes and additive record shapes admitted by .2.
- Apply effective partition/heal/delay changes to affected connection state, wake both connection and descriptor waiters, and retain existing all-or-nothing group validation.
- On effective heal, apply normal delay to held data. Queue fresh writes behind it; on repartition hold the remaining suffix and reapply delay at next heal. Redundant topology actions cannot restart intervals or add delay.
- Keep one topology cause record; no per-release history. Tests observe actual endpoint bytes and blocked/progress outcomes instead of internal queue layout.

- Register every new process case in runner/internal/exploration/integration_test.go's test-name list and Make's Runner regex, and extend internal/archcheck/integration_test.go in this task before both-backend acceptance. Run the Runner-backed case without a skip. .10 audits the final complete selection; it is not a deferred executor prerequisite.

### Investigation targets

**Required:**

- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go:118`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:621`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go:737`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go:73`
- `tools/gomad3sim/network_toolchain_test.go:116`
- `tools/gomad3sim/fault_toolchain_test.go:217`

### Quick commands

```bash
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/backends/native/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetwork|Test.*Fault'
```

Process acceptance command: `make -C tools/gomad3 test-simulation`, with newly registered Runner-backed selectors retained in the handover. A direct seeded-wrapper process skip supplies no acceptance.

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

### Revival trigger (2026-10-13)

The owner batch-first strategy admits the coherent connection/holding/expiry/dial/history source batch against integrated fn154.1 limits, fn154.3 timeout transport (c8dbb97c11) and fn154.2 generated NC2 config/vocabulary (c467490c78). Root's combined candidate passed portable simulator, protocol/projection, WASM reader/helper parity, full WASM simulation, all-platform architecture/purity, generated validation and affected lint. Source development may proceed while final independent batch review and exact native acceptance remain open. This does not waive completion dependencies, the early public-interface equivalence proof, any native gate or final current-source WASM qualification. No CI, PR, push or native qualification authority is revived. Use current backends/native paths above, one connection owner, and the accepted single generated semantic codec. Root serializes integration/identity regeneration and retains final lifecycle.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] Held-before and held-after cases have complete FIFO outcomes, no stream gap and no partition_drop success.
- [ ] Delayed heal, fresh-write overtaking, repeated partition and redundant heal/disable controls pass.
- [ ] Asymmetric reverse-direction and grouped rejected-mutation controls retain their pins.
- [ ] Topology changes wake local/descriptor waiters without making undelivered bytes readable.

## Done summary
portable.log pins held-before/after FIFO, received tails, delayed heal, repartition, half-close, redundant topology and atomic reservation refusal. ordinary.log pins broker reverse-direction preservation and atomic grouped topology/reservation/expiry. Actual native waiter notification execution remains deferred; source and portable descriptor controls preserve notification ownership.

Acceptance uses immutable frozen960 execution evidence at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE; source manifest SHA256 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278, terminal stages.log PASS 2026-10-14T05:36:50Z. Source-equivalent current execution inputs are joined to exact gate-only947 source checks under MILESTONES' bounded-fix rule, not relabelled as current-HEAD guest execution. Authentic fn154 completion receipt .flow/tmp/completion-review-receipt-fn-154.json records SHIP (gpt-6.1-sol/high, same GPT family), base73c507fbf09318f36ae6ddf30ad8a917d801e868, session01a13834-e701-7cb3-b524-4777fb68ff23. Native execution stays unverified with fn128.4/fn149.2 per owner hibernation amendment. Full post-rename qualification remains mandatory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: portable.log pins held-before/after FIFO, received tails, delayed heal, repartition, half-close, redundant topology and atomic reservation refusal. ordinary.log pins broker reverse-direction preservation and atomic grouped topology/reservation/expiry. Actual native waiter notification execution remains deferred; source and portable descriptor controls preserve notification ownership., Full frozen WASM qualification stages.log terminal PASS; source.digest retained unchanged, fn154 formal completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Corrected-source architecture/lint/validate-retry/clock/actionlint PASS /tmp/gomad-gate-integrated-*-20261014.log; native execution deferred fn128.4/fn149.2
- PRs: