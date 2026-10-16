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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
