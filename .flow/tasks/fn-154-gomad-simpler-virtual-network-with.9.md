---
satisfies: [R1, R2, R3, R4, R9, R11, R12]
---
# fn-154-gomad-simpler-virtual-network-with.9 Pin shared-connection and in-process fault conformance

## Description
Pin shared-connection and in-process fault conformance (R1, R2, R3, R4, R9, R11, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/network_handles_toolchain_test.go`, `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3sim/fault_toolchain_test.go`, `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go`, `tools/gomad3/backends/native/execution/model_conformance_test.go`
**Touches:** [tools/gomad3sim/network_handles_toolchain_test.go, tools/gomad3sim/network_toolchain_test.go, tools/gomad3sim/fault_toolchain_test.go, tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go, tools/gomad3/backends/native/execution/model_conformance_test.go]

### Approach

- Complete the public-interface operation sequence pin for standalone and in-process domains. Add missing controls only; use preserved conformance rather than implementation-shape assertions.
- Cover short-read tail charge, local byte-full blocking, global byte error, oversize partial writes, deadline then reuse, half-close/empty I/O and selected descriptor readiness.
- Record named evidence for every existing fault/outcome in R12, including graceful queued-byte discard, crash/restart and asymmetric/group topology.
- Retain the stock-TCP comparison and declared differences. Prove stale model/handle/incarnation rejection before mutation and nested/extra/unused/reordered fault negatives.

### Investigation targets

**Required:**

- `tools/gomad3sim/network_handles_toolchain_test.go:27`
- `tools/gomad3sim/network_toolchain_test.go:335`
- `tools/gomad3sim/network_toolchain_test.go:538`
- `tools/gomad3sim/fault_toolchain_test.go:76`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go:17`
- `tools/gomad3/backends/native/execution/model_conformance_test.go:110`

### Quick commands

```bash
cd tools/gomad3 && .toolchain/bin/go test -tags test_dep internal/gomadio -run 'TestNetworkHandle'
cd /Users/stephan/Workspace/skunkworks/gomad/temporal && env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD3_CHILD_SEED=89 tools/gomad3/.toolchain/bin/go test -exec tools/gomad3/backends/native/conformance/scripts/exec.sh -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'TestNetwork|Test.*Fault'
cd tools/gomad3 && go test -tags test_dep ./backends/native/execution -run '^TestModelConformanceTCP$'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] Local and in-process operation-sequence equivalence is pinned through the public interface.
- [ ] Each byte/caller-deadline/revocation boundary has a concrete positive and negative case.
- [ ] Every R12 in-process fault/outcome has named evidence; explicit fault matching remains valid.
- [ ] Existing TCP conformance and retained validation-before-mutation controls pass on the supported candidate.

## Done summary
Actual-source portable equivalence pins public-interface local operations and byte/tail/descriptor boundaries. Preserved native fixtures cover deadline/reuse, capacity identities, tiny writes, crash/restart, EOF, reordered topology and payload waiver; portable matching/plan/validation-before-mutation and WASM asymmetric/group controls pass. Actual native in-process fault/TCP conformance remains deferred, not passed.

Acceptance uses immutable frozen960 execution evidence at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE; source manifest SHA256 8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278, terminal stages.log PASS 2026-10-14T05:36:50Z. Source-equivalent current execution inputs are joined to exact gate-only947 source checks under MILESTONES' bounded-fix rule, not relabelled as current-HEAD guest execution. Authentic fn154 completion receipt .flow/tmp/completion-review-receipt-fn-154.json records SHIP (gpt-6.1-sol/high, same GPT family), base73c507fbf09318f36ae6ddf30ad8a917d801e868, session01a13834-e701-7cb3-b524-4777fb68ff23. Native execution stays unverified with fn128.4/fn149.2 per owner hibernation amendment. Full post-rename qualification remains mandatory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4
- Tests: Actual-source portable equivalence pins public-interface local operations and byte/tail/descriptor boundaries. Preserved native fixtures cover deadline/reuse, capacity identities, tiny writes, crash/restart, EOF, reordered topology and payload waiver; portable matching/plan/validation-before-mutation and WASM asymmetric/group controls pass. Actual native in-process fault/TCP conformance remains deferred, not passed., Full frozen WASM qualification stages.log terminal PASS; source.digest retained unchanged, fn154 formal completion SHIP .flow/tmp/completion-review-receipt-fn-154.json, Corrected-source architecture/lint/validate-retry/clock/actionlint PASS /tmp/gomad-gate-integrated-*-20261014.log; native execution deferred fn128.4/fn149.2
- PRs: