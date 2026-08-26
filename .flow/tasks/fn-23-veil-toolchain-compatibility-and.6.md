---
satisfies: [R6, R7]
---
# fn-23-veil-toolchain-compatibility-and.6 Prove model isolation and document the C11 gate

## Description
Add end-to-end isolation and repeatability coverage around the opt-in gate, then document how model engineers run and interpret it. Snapshot primary Lake/toolchain/import surfaces and ordinary build/regression behavior before and after the diagnostic, assert stable receipt identity across permitted host/timing variation, and record the resulting C11 adoption/defer decision without claiming a Temporal proof. Repair the component-roadmap link to the current optional Veil design and describe how the next formal-receipt slice consumes this explicit gate. Preserve existing comments.

**Size:** M
**Files:** `tools/umpire/veilcompat/integration_test.go`, `model/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/veilcompat/integration_test.go, model/README.md, .plans/UMPIRE4_COMPONENTS.md]

## Acceptance
Focused fake matrices and at least one opt-in real run on the supported Linux/aarch64 reference host must produce a complete canonical status-0 adopt/defer or status-2 inconclusive receipt; status-1 remains required error evidence but cannot complete this task or update the roadmap decision. Before/after checks prove byte-identical primary Lake files, unchanged `Umpire`/`Temporal` imports and default dependency graph, passing ordinary model/regression targets, no generated/product semantics, and no prohibited legacy dependency/use. Documentation distinguishes compatibility from proof, names the exact pinned candidates/closure/trust classes, records the actual reviewed roadmap decision receipt, and leaves optional adoption to the next reviewed slice.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
