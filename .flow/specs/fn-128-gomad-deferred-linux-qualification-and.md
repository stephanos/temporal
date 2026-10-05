# Gomad deferred Linux qualification and repairs

## Goal & Context

The existing Gomad specs can complete their implementation and Darwin acceptance without waiting for native Linux hardware. This spec owns every remaining linux/amd64 execution obligation transferred on 2026-10-04 from fn-105, fn-109, fn-110, fn-112, fn-113 and fn-114, including D11 and D12. Linux remains unverified until the owning tasks retain native evidence.

The owner does not expect a native Linux system to become available. Keep this spec deferred and not ready. Revive it only when the owner requests Linux qualification and supplies native linux/amd64 execution through a host or CI, a supported pinned toolchain, and an identified source candidate. Developmental linux/arm64, emulation, cross-compilation, static source checks and Darwin reports cannot satisfy Linux execution requirements.

The transfer manifest at `.flow/artifacts/linux-scope-transfer-2026-10-04.md` maps the original requirements and tasks to their new owners. Original clauses and evidence remain recoverable at Git commit `10d884c6f9d97681d08aaf2636f5850407f1586a`. The old D11/D12 IDs record administrative transfers only.

## Acceptance Criteria

- **R1:** Establish a native linux/amd64 candidate, pinned toolchain/profile and exact source closure; collect runtime, process, overlay, diagnostics, time-wire, forward-clock and full-host evidence inherited from the source tasks. Errors: an unsupported host, absent runtime, stale source binding or failed build leaves the prerequisite open; a recorded gate failure is never a qualification pass. Initial collection may record the known D12 divergence while R7 remains open.
- **R2:** Fulfill the original D12 causal fix. Identify the first divergent event under native Linux host load, fix its cause, retain a failing-before/passing-after regression, qualify traced F5/F6 seeds 11 and 17 repeatedly with exact replay, and restore qualified expectations by removing nondeterministic/replay_divergence allowances from both dispatch-only and required smoke CI. Errors: diagnosis alone, disabled tracing, relaxed dispositions, an unresolved cause or an unavailable host cannot complete this requirement.
- **R3:** Preserve D11's dependency on the completed D21 findings. If those findings establish audit need and feasible scope, implement the bounded vDSO-disabled/seccomp audit with an unseeded forbidden-clock positive control and passing seeded core-linux evidence. Otherwise retain an explicit owner-approved not-triggered disposition with the D21 trigger evidence. Errors: host absence cannot establish that the trigger is false; collector/assembly restrictions and fail-closed syscall policy remain unchanged.
- **R4:** Retain native Linux model-conformance, architecture/lifecycle/network/filesystem/process/race, affected-consumer, core/smoke/representative, compatibility-pack and version-maintenance evidence required by the transferred tasks. Run host-specific discover/review/generate/qualify with exact identities and retained replay wherever required. Errors: omitted workloads, stale pins, mixed revisions or substituted Darwin results leave their Linux obligations open; D12-related replay dispositions remain informational until R2 passes.
- **R5:** Execute the actual Linux determinism soak and retain its report, ledger, cohort counts, diagnostics, load/overflow behavior and measured platform-specific bound. Errors: stand-ins, source inspection, incomplete cohorts or absent reports cannot establish a bound. Linux replay remains informational while R2 is unresolved; final evidence must match R7's candidate.
- **R6:** Qualify the real downstream target on Linux through closure and linked analyses, exact reachable adapters, refusal/drift tests, consumer-owned packs and the bounded driver. Execute seeds 11 and 17 twice each, replay retained successes exactly, and retain source reviews, identities and Linux-specific commands/support documentation. Errors: absent downstream checkout or shared D8 implementation, unsupported analysis, drift, a classified failure or missing native execution leaves this open.
- **R7:** Complete a fresh combined-candidate Linux matrix covering all transferred obligations, commands, source/toolchain/profile/module identities, dispositions, formal review and documentation. Refresh results invalidated by R2 or later source changes, or retain an explicit justified unaffected disposition without carrying invalid pack/profile bindings. Errors: missing native proof, unresolved strict replay, unexplained regressions, cross-platform artifacts or mixed-source shards prevent overall Linux qualification. No error surface beyond these and R1-R6.

## Boundaries

Source specs retain shared implementation, Darwin qualification, static coverage of both supported source sets, preservation, size, full-host, review and downstream-checkout requirements. Their completion has no dependency on this spec. fn-105 D6 and D15 remain conditional non-Linux work under their original owners. Closed fn-107's implementation-only scope and fn-108's owner waiver remain unchanged.

This transfer does not widen supported platforms, compatibility access or patch policy. It does not alter immutable reports, historical completed-task evidence, production code or CI dispositions. Every future repair follows its original policy restrictions and receives its own verification and commit.

## Decision Context

Seven tasks keep independently executable Linux evidence collection separate from the causal replay fix, conditional audit, model/consumer qualification, soak and downstream work. The final matrix reconciles one candidate after repairs.

Task 1 proves native toolchain capability and records initial gate outcomes; it does not wait for task 2's strict replay fix. Task 2 consumes those diagnostics. Tasks 3-6 become dependency-eligible after task 1. Their declared toolchain, conformance, workflow and qualification write surfaces overlap, so serialize overlapping writers and freeze the candidate during evidence collection. Parallelize only read-only qualification runs or disjoint artifact output directories with no shared-source mutations. Task 2's runtime/CI repair completes before collecting evidence claimed for its final candidate. Task 7 waits for tasks 2-6 and reconciles the final candidate. The conditional audit may finish only with verified execution or an owner-approved not-triggered disposition. The downstream task also waits for the shared D8 candidate and actual consumer checkout. Flow task dependencies are spec-local, so D21 and D8 remain named external prerequisites in the task and blocked reason; this spec does not depend on all of fn-105.

Current source identities differ from historical qualification receipts. Preserve old receipts as historical evidence; they do not qualify a later combined candidate.

## Quick commands

```bash
/home/agent/.codex/scripts/flowctl show fn-128-gomad-deferred-linux-qualification-and --json
/home/agent/.codex/scripts/flowctl validate --spec fn-128-gomad-deferred-linux-qualification-and --coverage --json
make -C tools/gomad3 validate
GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host
make -C tools/gomad3 test
make gomad3-soak
```

Use the original source-task command ledgers for additional toolchain, process, runtime/overlay, race, pack, core, smoke, representative and downstream gates. Freeze the candidate during each command and retain its identity and exit code.

## Early proof point

Task 1 must build and launch the qualified patched runtime on native linux/amd64 with the expected toolchain/profile identity. An unsupported host or failed native build blocks dependent execution. Existing source-spec work continues independently.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Establish a native linux/amd64 candidate, pinned toolchain/profile and exact source closure; collect runtime, process, overlay, diagnostics, time-wire, forward-clock and full-host evidence inherited from the source tasks. Errors: an unsupported host, absent runtime, stale source binding or failed build leaves the prerequisite open; a recorded gate failure is never a qualification pass. Initial collection may record the known D12 divergence while R7 remains open. | fn-128-gomad-deferred-linux-qualification-and.1 | — |
| R2 | Fulfill the original D12 causal fix. Identify the first divergent event under native Linux host load, fix its cause, retain a failing-before/passing-after regression, qualify traced F5/F6 seeds 11 and 17 repeatedly with exact replay, and restore qualified expectations by removing nondeterministic/replay_divergence allowances from both dispatch-only and required smoke CI. Errors: diagnosis alone, disabled tracing, relaxed dispositions, an unresolved cause or an unavailable host cannot complete this requirement. | fn-128-gomad-deferred-linux-qualification-and.2 | — |
| R3 | Preserve D11's dependency on the completed D21 findings. If those findings establish audit need and feasible scope, implement the bounded vDSO-disabled/seccomp audit with an unseeded forbidden-clock positive control and passing seeded core-linux evidence. Otherwise retain an explicit owner-approved not-triggered disposition with the D21 trigger evidence. Errors: host absence cannot establish that the trigger is false; collector/assembly restrictions and fail-closed syscall policy remain unchanged. | fn-128-gomad-deferred-linux-qualification-and.3 | — |
| R4 | Retain native Linux model-conformance, architecture/lifecycle/network/filesystem/process/race, affected-consumer, core/smoke/representative, compatibility-pack and version-maintenance evidence required by the transferred tasks. Run host-specific discover/review/generate/qualify with exact identities and retained replay wherever required. Errors: omitted workloads, stale pins, mixed revisions or substituted Darwin results leave their Linux obligations open; D12-related replay dispositions remain informational until R2 passes. | fn-128-gomad-deferred-linux-qualification-and.4 | — |
| R5 | Execute the actual Linux determinism soak and retain its report, ledger, cohort counts, diagnostics, load/overflow behavior and measured platform-specific bound. Errors: stand-ins, source inspection, incomplete cohorts or absent reports cannot establish a bound. Linux replay remains informational while R2 is unresolved; final evidence must match R7's candidate. | fn-128-gomad-deferred-linux-qualification-and.5 | — |
| R6 | Qualify the real downstream target on Linux through closure and linked analyses, exact reachable adapters, refusal/drift tests, consumer-owned packs and the bounded driver. Execute seeds 11 and 17 twice each, replay retained successes exactly, and retain source reviews, identities and Linux-specific commands/support documentation. Errors: absent downstream checkout or shared D8 implementation, unsupported analysis, drift, a classified failure or missing native execution leaves this open. | fn-128-gomad-deferred-linux-qualification-and.6 | — |
| R7 | Complete a fresh combined-candidate Linux matrix covering all transferred obligations, commands, source/toolchain/profile/module identities, dispositions, formal review and documentation. Refresh results invalidated by R2 or later source changes, or retain an explicit justified unaffected disposition without carrying invalid pack/profile bindings. Errors: missing native proof, unresolved strict replay, unexplained regressions, cross-platform artifacts or mixed-source shards prevent overall Linux qualification. No error surface beyond these and R1-R6. | fn-128-gomad-deferred-linux-qualification-and.7 | — |
