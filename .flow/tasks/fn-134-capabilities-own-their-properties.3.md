---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-134-capabilities-own-their-properties.3 Migrate the Temporal kit and every Model to capabilities sections; regenerate

## Description
Move every law into its capability's companion and every declaration site into a `capabilities` section, with bounds in `queries`, then run the one regeneration. This is the task R7's equivalence proof belongs to. The tests that read law data move here too, so the gate is green at the end.

**Size:** M
**Files:** `model/temporal/capabilities/{Capabilities,Close,Pause,Terminate,Cancel}.scala`, `model/temporal/capabilities/Catalog.test.scala` (the two-machines test, rewritten), `model/temporal/features/activity/standalone/{Standalone,product/Product,system/System,system/Record,system/WithTaskQueue}.scala`, `model/temporal/features/nexus/standalone/{Standalone,system/System}.scala`, `tools/umpire/check/{capabilities_test,activity_parity_test}.go`, `model/ir/**`, `model/cases/**`
**Touches:** [model/temporal/**, model/ir/**, model/cases/**, tools/umpire/check/capabilities_test.go, tools/umpire/check/activity_parity_test.go]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Kit: each `object x extends Law(...)` (Close.scala:19,40; Pause.scala:12; Terminate.scala:13; Cancel.scala:12) becomes a def in its kind's companion (Capabilities.scala:16-65), with `promises`/`doesNotPromise` and the cites moved to Scaladoc. `pausedIsNotDispatched` goes into `Pausable`'s companion and reads Pollable's `running`. Pausable and Pollable stay separate.
- Sites: Product.scala:120, System.scala:312 and nexus System.scala:155 become `capabilities` sections, keeping nexus's `overriding`. Record.scala:276-290 and WithTaskQueue.scala:101-117 and :189-205 become shared sets (task 2's form), keeping each `except`. Rewrite the `.claim` uses (Record.scala:256-264, 346; WithTaskQueue.scala:87-94, 181-183) and the `irFile` roots (activity Standalone.scala:144-145, nexus Standalone.scala:53).
- Bounds move unchanged: three (product, activity system, nexus), five (record, over-queue), twelve (over-matching).
- Two-machines test: rewrite Catalog.test.scala over the capability companions and the declared machines (its hand-written list at :18-43 updates).
- Go tests that read `activity-standalone.laws.json` (`capabilities_test.go:54-97`, `activity_parity_test.go:90-104`, which also has a regex on `object implements`) now read the IR `origin` and the new section form.
- Equivalence harness: before editing, capture a projection on the current tree with the gate's own outputs: every Query name and answer, every Check receipt, every Definition ID, the exploration identity map, `model/cases` bytes, and `model/ir/*.lint.json`. After `make umpire-gen-model`, diff it. The only allowed IR differences are `origin` fields, generated-Property positions and the three deleted `*.laws.json`. `.lint.json` keys and reasons must be unchanged.

### Investigation targets
**Required:**
- `model/temporal/capabilities/*.scala`
- `model/temporal/features/activity/standalone/system/{Record,WithTaskQueue}.scala`
- `model/temporal/features/nexus/standalone/system/System.scala:155-175`
- `tools/umpire/check/capabilities_test.go`, `tools/umpire/check/activity_parity_test.go:90-110`
**Optional:**
- `model/temporal/IrFiles.test.scala:29`

### Key context
This runs after fn-132 and fn-133 close, rebased on their regenerated `model/ir`, so the baseline is clean. No other regeneration may run alongside it.

## Acceptance
- [ ] No `Law` object and no `implements` section remain under `model/temporal`; Pausable and Pollable are separate capabilities.
- [ ] Every machine's generated Queries are bounded in `queries` with the bounds they had.
- [ ] The two-machines test passes over the companions.
- [ ] Equivalence diff: Query names, answers, receipts, Definition IDs, exploration identity, Case bytes and `.lint.json` unchanged; the IR diff shows only `origin`, generated-Property positions and the removed `*.laws.json`.
- [ ] `make umpire-gen-model`, `make umpire-check-cases`, `make umpire-check-lint` and the Go suite pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
