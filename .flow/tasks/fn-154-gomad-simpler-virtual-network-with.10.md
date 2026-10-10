---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R11, R12]
---
# fn-154-gomad-simpler-virtual-network-with.10 Execute process network cases through registered isolated Runner gates

## Description
Execute process network cases through registered isolated Runner gates (R1, R2, R3, R4, R5, R6, R7, R11, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/network_process_handles_toolchain_test.go`, `tools/gomad3sim/network_handles_toolchain_test.go`, `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go`, `tools/gomad3/simulation_gate_selection_test.go`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3sim/network_process_handles_toolchain_test.go, tools/gomad3sim/network_handles_toolchain_test.go, tools/gomad3/runner/internal/execution/simulation_root_integration_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/Makefile]

### Approach

- Run process cases equivalent to the local acceptance table and cover bounded request/result allocation, partial counts, incarnation/domain ownership and fresh processes.
- Audit and complete registrations already owned by .5/.6/.7 before their acceptance, then register process-parity additions here. Verify Runner list, Make selection and selection guards execute every required case. Compilation or a skipped detached process test is insufficient.
- Preserve existing arbiter response/cancellation/late-response accounting and hard isolation independently of local model parity.
- Retain the strict node-clock watchdog finding under its existing owner; do not hide it with a forward-mode pass or weaken its gate.

### Investigation targets

**Required:**

- `tools/gomad3sim/network_process_handles_toolchain_test.go:19`
- `tools/gomad3sim/network_handles_toolchain_test.go:65`
- `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go:33`
- `tools/gomad3/simulation_gate_selection_test.go`
- `tools/gomad3/Makefile:154`

### Quick commands

```bash
make -C tools/gomad3 test-simulation
cd tools/gomad3 && go test -tags test_dep -run '^TestSimulationGateSelectsProcessNetworkHandles$' .
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] Every R12 process fault/outcome and new hold/stall/dial/byte boundary runs with named evidence.
- [ ] Runner/Make selection includes each required case, with a guard against missing selectors.
- [ ] Fresh-process isolation, partial bounded replies and stale-incarnation negatives pass.
- [ ] Canonical test-simulation retains its disposition, including separately reported existing strict-clock finding.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
