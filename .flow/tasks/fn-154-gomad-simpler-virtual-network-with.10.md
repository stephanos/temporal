---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R11, R12]
---
# fn-154-gomad-simpler-virtual-network-with.10 Execute process network cases through registered isolated Runner gates

## Description
Execute process network cases through registered isolated Runner gates (R1, R2, R3, R4, R5, R6, R7, R11, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/network_process_handles_toolchain_test.go`, `tools/gomad3sim/network_handles_toolchain_test.go`, `tools/gomad3/runner/internal/exploration/integration_test.go`, `tools/gomad3/internal/archcheck/integration_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3sim/network_process_handles_toolchain_test.go, tools/gomad3sim/network_handles_toolchain_test.go, tools/gomad3/runner/internal/exploration/integration_test.go, tools/gomad3/internal/archcheck/integration_test.go, tools/gomad3/Makefile]

### Approach

- Run process cases equivalent to the local acceptance table and cover bounded request/result allocation, partial counts, incarnation/domain ownership and fresh processes.
- Audit and complete registrations already owned by .5/.6/.7 before their acceptance, then register process-parity additions here. Verify Runner list, Make selection and selection guards execute every required case. Compilation or a skipped detached process test is insufficient.
- Preserve existing arbiter response/cancellation/late-response accounting and hard isolation independently of local model parity.
- Retain the strict node-clock watchdog finding under its existing owner; do not hide it with a forward-mode pass or weaken its gate.

### Investigation targets

**Required:**

- `tools/gomad3sim/network_process_handles_toolchain_test.go:19`
- `tools/gomad3sim/network_handles_toolchain_test.go:65`
- `tools/gomad3/runner/internal/exploration/integration_test.go:33`
- `tools/gomad3/internal/archcheck/integration_test.go`
- `tools/gomad3/Makefile:154`

### Quick commands

```bash
make -C tools/gomad3 test-simulation
cd tools/gomad3 && go test -tags test_dep -run '^TestSimulationGateSelectsProcessNetworkHandles$' ./internal/archcheck
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] Every R12 process fault/outcome and new hold/stall/dial/byte boundary runs with named evidence.
- [ ] Runner/Make selection includes each required case, with a guard against missing selectors.
- [ ] Fresh-process isolation, partial bounded replies and stale-incarnation negatives pass.
- [ ] Canonical test-simulation retains its disposition, including separately reported existing strict-clock finding.

## Done summary
Architecture TestSimulationGateSelectsProcessNetworkHandles pins Runner/Make registration for all sixteen handle cases and held FIFO, expiry, ordering, partitioned dial and framed families. Native source fixtures retain listener/connection/byte capacity, revocation, grouped faults and changed/unused/reordered actions. Native fresh-process isolation and bounded replies remain deferred; existing strict-clock watchdog disposition is preserved separately.

Acceptance uses immutable frozen960 execution evidence at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE; source manifest SHA256 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278, terminal stages.log PASS 2026-10-14T05:36:50Z. Source-equivalent current execution inputs are joined to exact gate-only947 source checks under MILESTONES' bounded-fix rule, not relabelled as current-HEAD guest execution. Authentic fn154 completion receipt .flow/tmp/completion-review-receipt-fn-154.json records SHIP (gpt-6.1-sol/high, same GPT family), base73c507fbf09318f36ae6ddf30ad8a917d801e868, session01a13834-e701-7cb3-b524-4777fb68ff23. Native execution stays unverified with fn128.4/fn149.2 per owner hibernation amendment. Full post-rename qualification remains mandatory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: Architecture TestSimulationGateSelectsProcessNetworkHandles pins Runner/Make registration for all sixteen handle cases and held FIFO, expiry, ordering, partitioned dial and framed families. Native source fixtures retain listener/connection/byte capacity, revocation, grouped faults and changed/unused/reordered actions. Native fresh-process isolation and bounded replies remain deferred; existing strict-clock watchdog disposition is preserved separately., Full frozen WASM qualification stages.log terminal PASS; source.digest retained unchanged, fn154 formal completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Corrected-source architecture/lint/validate-retry/clock/actionlint PASS /tmp/gomad-gate-integrated-*-20261014.log; native execution deferred fn128.4/fn149.2
- PRs: