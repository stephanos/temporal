---
satisfies: [R1]
---
# fn-188-gomad-wasm-determinism-boundary-audit.1 Determinism inventory: schema, loader and completeness over the import table and seeded streams

## Description
Add the checked-in inventory `gomad.wasm-determinism-inventory/v1` (fields: id, layer, class, site, disposition, owner, rule, evidence, justification+readme_anchor for escapes; `reviewed` block with go_toolchain, wasmtime, rust). Write a Go loader in the WASM backend that rejects unknown fields and reused ids, and a Rust reader in the helper tests over the same file. First completeness checks: every `Operation` variant of the helper `operations!` table has a row (Rust side derives the set from the enum; Go side from the frozen per-module import list), and every seeded stream compared by `choice.DiffDiagnostics` (run queue, scheduler, select, runtime rand, cheap rand, timer, clock tick) has a row. Seed the rows for those two classes with dispositions and evidence tests that already exist (unknown-import refusal tests, cooperative runtime qualification). Early proof: show the test failing when a row is removed and when a variant is added without a row. Record the decision on the WASM no-op `gomadDiagnosticCheckSeededDraw` in `runtime_wasm.go.txt` as a row (implement the host-timed-draw check or disposition `escape`/`rejected` with justification). Stop and report if the import table or stream list cannot be derived without a hand-kept duplicate.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/determinism/inventory.json, tools/gomad/internal/backends/wasm/determinism/{inventory.go,inventory_test.go}, tools/gomad/internal/backends/wasm/wasmhost/tests/inventory.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs (derivation only), tools/gomad/internal/backends/wasm/toolchain/runtime_wasm.go.txt

## Acceptance
- [ ] Inventory file loads in Go and Rust; unknown field or reused id fails loading
- [ ] Every `Operation` variant and every DiffDiagnostics stream has a row; removing a row or adding a variant without one fails the test by name
- [ ] Every row names an evidence test that exists
- [ ] `gomadDiagnosticCheckSeededDraw` decision recorded as a row with rule and evidence

## Done summary
Implemented strict shared Go/Rust inventory loading, source-derived import and diagnostic-stream coverage, named evidence validation and the explicit wasip1 seeded-draw-check decision. Historical identity enforcement now preserves all 195 original pairs and the accepted 263-pair ledger; removal, rename, reassignment, swaps and ledger-prefix mutations fail. Runtime per-site corrections are shared with task .2. See the source-acceptance handover and external inventory-contract-repair reports for red/green evidence. This completes the inventory task, not stress or full-spec acceptance.

Evidence handover: `.flow/artifacts/fn188-source-acceptance-2026-10-20.md`. Integrated source `5dcd37ceaea62e95ba7e33703e247094fea6d158`: full default validate exit 0 and affected lint exit 0, zero issues.

stage: impl-review - SHIP (official SINGLE re-review, 2026-10-20T18:15:58Z; same session 01a15a74-37ad-7020-b5d4-830052a6a1a3; codex:gpt-6.1-sol:high, same-family; .flow/tmp/impl-review-fn188-source-batch.json; both findings fixed, zero open).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5dcd37ceaea62e95ba7e33703e247094fea6d158
- Tests: Go/Rust malformed inventory and duplicate/historical identity tests: exit 0; old-code negative regressions exit 1/101, Early removed operation/stream and added SockRecv mutation proof: named failures retained in t1-early-proof.log, make -C tools/gomad validate at d2410c5e: exit 0, inventory 12/12, corpus 26/26, engine refusals 3/3, Final full Go determinism package at e0cd113841: exit 0 (7.222 s), make -C tools/gomad validate on integrated5dcd37ce: exit0 (inventory12/12, corpus26/26, engine3/3, Go determinism/architecture/WASI); /home/agent/wt/evidence/integration-2026-10-20/validate-fn188-controller-tagged-fixes.log, make lint-code-gomad with Linux tools, FIX=false, basee0cd113841: exit0, zeroissues; /home/agent/wt/evidence/integration-2026-10-20/lint-fn188-controller-tagged-fixes.log
- PRs: