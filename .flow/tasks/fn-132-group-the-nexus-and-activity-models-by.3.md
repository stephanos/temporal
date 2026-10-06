---
satisfies: [R4]
---
# fn-132-group-the-nexus-and-activity-models-by.3 Structure lint and docs learn the kind level

## Description
**Size:** M
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/layout/**, model/irgen/testdata/lifts/**, tools/umpire/ir/layout_test.go, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** prerequisite grouping/fixture handover; `model/irgen/Structure.scala:239`; `model/irgen/test/Fixtures.test.scala:1796`; `tools/umpire/ir/layout_test.go:324`; README layout sections and module map.
Reuse the prerequisite's classifier and validation seam. This task pins the three named R4 refusal specimens, verifies the actual moved tree and completes docs; do not reimplement grouping. Preserve independent/compound-invalid level guards and own-kind refinement boundaries. Repeat only affected checks after task 2's full batch unless an actual implementation change invalidates broader results.

fn-126's structure lint (R20) learns the kind level. A `features/<kind>/` folder holds one general feature file named after the kind, an optional `product/`, and its forms as subfolders (`workflow/`, `standalone/`), each a feature folder by fn-126's rules. A form's `system/` may refine a machine in its kind's `product/`. The general feature file holds types and signature and no machine.

Add one refusal fixture each: a form folder (`workflow/` or `standalone/`) outside a kind folder; a machine in a kind's general file; a second general file in a kind folder. The R10 layout test learns the new folders.

Docs: `model/README.md` ("Writing a Model", "Where things are") describes the kind level with Nexus as the example; `.plans/UMPIRE_MODULES.md` rows follow.

## Acceptance
- [ ] The lint accepts the tree tasks 1 and 2 produced, and each refusal fixture is refused at its line.
- [ ] `model/README.md` and `.plans/UMPIRE_MODULES.md` show the kind level.
- [ ] `make lint-model` and the layout test pass.

## Done summary
TBD

## Verification evidence for implementation review

The task range begins at `9144f2e07a67c8b3831662bef6d6c47dc345d60e`. The only implementation change uses the existing `Structure.formFolders` catalog to reject a form directly under `features` before flat-feature admission. Existing kind grouping, flat/shared feature rules and all independent Product/System guards are preserved.

Task-local logs are under `.flow/tmp/fn132-3/`, with `.command`, `.stdout`, `.stderr`, `.exit` and `.wall` per attempt. `r4-red` exited 1 in 136 seconds and explicitly collected all three named R4 specimens. Only `form-outside-kind` was admitted before the fix; the two existing refusal behaviors already met their exact-line assertions.

`regen-model` (`make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model`) exited 0 in 251 seconds. Its unfiltered fixture phase took 167 seconds and preserves all registered assertions, including the new exact-line specimens and prior positive/compound-invalid guards. Gate.scala captures successful fixture stdout, so GEN prints the phase and terminal result, not individual test names. The production moved tree also lifted successfully.

`layout` exited 0 in 8 seconds with 88 Go pass events across current-tree layout and path checks. `full-scala-lint` (`make lint-model`) exited 0 in 19 seconds; `check-cases` (`make umpire-check-cases`) exited 0 in 13 seconds; batch-base read-only `go-lint` exited 0 in 6 seconds. The inherited JDK27 scalafix reflection warning uses the retained `.flow/tmp/fn124-8/current/scalafix-probe-proof.md` evidence.

`artifacts-before.sha256` and `artifacts-after.sha256` match exactly across all 63 IR/Case/functional/canary/lifter-expectation files. `production-before.sha256` verifies all 35 production source files unchanged. Each hash check is retained in `artifacts-after.check` or `production-after.check`. Independently frozen expectations were not changed to accept new output.

MILESTONES source/input reuse keeps task2's unaffected full Go, runtime, publication, pinned-history and live-limitation evidence applicable; see `.flow/tmp/handovers/fn-132.2-summary.md` and `.flow/tmp/fn132-2/verification-ledger.json`. This task makes no full CHECK receipt from GEN and no new live/backend claim. The parent remains open for tasks 4-7.

## Evidence
- Commits:
- Tests:
- PRs:
