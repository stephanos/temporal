---
satisfies: [R4]
---
# fn-188-gomad-wasm-determinism-boundary-audit.4 First-divergence report across incarnations and World order, with reproducer

## Description
Build the comparison and report used by the stress producer and by hand. Input: two executions of the same workload, seed and identities (refuse otherwise). Compare in helper execution order: choice tape, diagnostic trace counters (reuse `choice.DiffDiagnostics`), WASI transcript ops and clock-run folding, entropy counters, model transition digests (World, network, volume, namespace), per-incarnation outputs and the shared World order. Output the earliest differing event with kind (`choice`, `draw-counter`, `clock`, `entropy`, `wasi-op`, `model-transition`, `output`, `world-order`), instance and incarnation, local ordinal, shared execution-order coordinate and virtual time (fn-180 coordinates), both values, and a reproducer block (workload, seed, perturbation cell, both evidence digests, identities, exact rerun commands, prefix cut usable as a `gomad replay` stop point). Runs without a diagnostic trace report `undiagnosable` and the flag to enable it. Fixtures: a seeded single-guest and cluster pair where one run injects one perturbed draw, one shifted clock read and one reordered cluster handoff (test-only hooks, compiled out of release), asserting the expected first event for each. Expose as `gomadtool`-style subcommand or a Go package consumed by the stress producer; no evidence format change.

**Size:** L
**Files:** tools/gomad/internal/backends/wasm/determinism/{diverge.go,diverge_test.go,testdata/**}, tools/gomad/internal/model/choice/diagnostic.go (read), tools/gomad/internal/model/cluster/wasm.go (read), tools/gomad/internal/backends/wasm/wasi/runtime.go (read)

## Acceptance
- [ ] Report names the earliest differing event with kind, instance/incarnation, ordinal, shared coordinate, virtual time and both values
- [ ] Reproducer block carries rerun commands and a prefix cut accepted by `gomad replay` as a stop point
- [ ] Fixtures for perturbed draw, shifted clock and reordered handoff each yield the expected first event
- [ ] Different seeds or identities refused; missing trace reports `undiagnosable`

## Done summary
Implemented first-divergence comparison in genuine helper event order, with instance/incarnation, local/shared coordinates, actual model time, both values, identities and authenticated preceding prefix cuts. Incomparable inputs are refused and missing diagnostics are undiagnosable. Four real Runner fixtures (draw, clock, handoff and output-before-clock) retain complete supplemental logs and failure artifacts. Provider inspection and the exact emitted external CLI prefix replay both verify the prefix and witness for every case. Earliest helper events are draw 102, clock 87, handoff 449 and output 84; draw/clock/output stop at choice 7 and handoff at b/1 choice 23. Proof roots retain their own original identities; they are not relabeled as full current-candidate qualification. No canonical evidence format change.

Final scoped repair bc3d00b5d6 accepts only the reserved controller/0 event identity. A real native Broker.command producer matches the checked-in Go parser fixture; guest zero incarnations and controller inspection boundaries still fail.

Evidence handover: `.flow/artifacts/fn188-source-acceptance-2026-10-20.md`. Integrated source `5dcd37ceaea62e95ba7e33703e247094fea6d158`: full default validate exit 0 and affected lint exit 0, zero issues.

stage: impl-review - SHIP (official SINGLE re-review, 2026-10-20T18:15:58Z; same session 01a15a74-37ad-7020-b5d4-830052a6a1a3; codex:gpt-6.1-sol:high, same-family; .flow/tmp/impl-review-fn188-source-batch.json; both findings fixed, zero open).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5dcd37ceaea62e95ba7e33703e247094fea6d158
- Tests: Actual TestDeterminismExecutedDivergenceArtifacts four-case execution: exit 0 (36.461 s), retained external fixture roots, All four prefix-receipt.json files: Report diverged/helper-event; Inspection and CLIInspection inspected_prefix, prefix_verified=true, witness_match=true, Different-seed/identity and missing/incomplete/framing/prefix diagnostic negative tests: exit 0, Integrated determinism/observed and transport tests in default validation: exit 0, make -C tools/gomad validate on integrated5dcd37ce: exit0 (inventory12/12, corpus26/26, engine3/3, Go determinism/architecture/WASI); /home/agent/wt/evidence/integration-2026-10-20/validate-fn188-controller-tagged-fixes.log, make lint-code-gomad with Linux tools, FIX=false, basee0cd113841: exit0, zeroissues; /home/agent/wt/evidence/integration-2026-10-20/lint-fn188-controller-tagged-fixes.log
- PRs: