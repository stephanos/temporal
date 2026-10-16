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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
