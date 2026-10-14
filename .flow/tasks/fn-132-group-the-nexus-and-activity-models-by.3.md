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
Structure rejects a form directly under `features/` using the existing form catalog and kind classifier. Three named R4 fixtures pin exact refusal lines; the current-tree layout checks, Model README and module map describe the kind/form layout while preserving existing validation boundaries.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-06T14:06:43.841602Z..2026-10-06T14:08:29.489018Z] (model: gpt-6.1-sol at high) - SHIP. Three fresh read-only same-Codex-family draws found zero findings. Native writer actual model is not evidenced.
stage: memory - skipped(clean first-pass SHIP).
stage: plan-sync - skipped(config: planSync.enabled=false).
baseline: green via applicable source/input-identical task2 evidence under MILESTONES policy; no formal BASELINE_HANDOFF or honored gate receipt.

### Verification

`.flow/tmp/fn132-3/verification-ledger.json` indexes captured commands, separate stdout/stderr, numeric exits and independent walls. The named R4 red run collected all three tests and exposed the outside-kind admission gap. After the minimal guard, canonical unfiltered generation passed its complete fixture phase and actual moved Model tree; Gate suppresses successful individual fixture names.

| Current observation | Exit | Wall seconds | Log stem |
| --- | ---: | ---: | --- |
| Three named R4 regressions before the fix | 1, intended orphan admission failure | 136 | r4-red |
| Focused Go layout checks, nine top-level tests and 88 pass events | 0 | 8 | layout |
| `make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model`, full fixture phase167s | 0 | 251 | regen-model |
| `make lint-model` | 0 | 19 | full-scala-lint |
| `make umpire-check-cases` | 0 | 13 | check-cases |
| Read-only batch-base Go lint, zero issues | 0 | 6 | go-lint |

All 63 complete IR/Case/fixture-expectation/publication artifacts retain identical inventory and bytes; all 35 production Model sources retain their hashes. The before/after manifests and checks are under `.flow/tmp/fn132-3`. No changed golden was recaptured. These equalities bound reuse of task2's unaffected Go/runtime/publication/canary/history evidence in `.flow/tmp/handovers/fn-132.2-evidence.json`. The inherited JDK27 scalafix warning retains the existing fn124-8 rule-execution proof. Task2's supplemental default-cluster worker-stop probe remains red/inconclusive under its approved MILESTONES owner exception. No fresh full CHECK receipt, whole-live green or backend result is claimed.

Review receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-132-group-the-nexus-and-activity-models-by.3.json`; sidecars: `.flow/review-fanout/b0517ee6911744e4a735265146335d5b/`. Reviewed range: `9144f2e07a67c8b3831662bef6d6c47dc345d60e..128ac8210c72fae26c1b21c1925e66a888b84a33`. The canonical completion receipt will close only task3; parent fn132 stays open and tasks4–7 stay pending.
## Verification evidence for implementation review

The task range begins at `9144f2e07a67c8b3831662bef6d6c47dc345d60e`. The only implementation change uses the existing `Structure.formFolders` catalog to reject a form directly under `features` before flat-feature admission. Existing kind grouping, flat/shared feature rules and all independent Product/System guards are preserved.

Task-local logs are under `.flow/tmp/fn132-3/`, with `.command`, `.stdout`, `.stderr`, `.exit` and `.wall` per attempt. `r4-red` exited 1 in 136 seconds and explicitly collected all three named R4 specimens. Only `form-outside-kind` was admitted before the fix; the two existing refusal behaviors already met their exact-line assertions.

`regen-model` (`make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model`) exited 0 in 251 seconds. Its unfiltered fixture phase took 167 seconds and preserves all registered assertions, including the new exact-line specimens and prior positive/compound-invalid guards. Gate.scala captures successful fixture stdout, so GEN prints the phase and terminal result, not individual test names. The production moved tree also lifted successfully.

`layout` exited 0 in 8 seconds with 88 Go pass events across current-tree layout and path checks. `full-scala-lint` (`make lint-model`) exited 0 in 19 seconds; `check-cases` (`make umpire-check-cases`) exited 0 in 13 seconds; batch-base read-only `go-lint` exited 0 in 6 seconds. The inherited JDK27 scalafix reflection warning uses the retained `.flow/tmp/fn124-8/current/scalafix-probe-proof.md` evidence.

`artifacts-before.sha256` and `artifacts-after.sha256` match exactly across all 63 IR/Case/functional/canary/lifter-expectation files. `production-before.sha256` verifies all 35 production source files unchanged. Each hash check is retained in `artifacts-after.check` or `production-after.check`. Independently frozen expectations were not changed to accept new output.

MILESTONES source/input reuse keeps task2's unaffected full Go, runtime, publication, pinned-history and live-limitation evidence applicable; see `.flow/tmp/handovers/fn-132.2-summary.md` and `.flow/tmp/fn132-2/verification-ledger.json`. This task makes no full CHECK receipt from GEN and no new live/backend claim. The parent remains open for tasks 4-7.

## Evidence
- Commits: a7ec275ec98153d2e49c6647893ce324cc18a9f9, cb45ed41b84a476df5fb40c2fa9b390584521c12, 128ac8210c72fae26c1b21c1925e66a888b84a33
- Tests: env UMPIRE_LIFTER_UPDATE= mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen --test-only '*Fixtures' -- '*R4*' (intended pre-fix red: exit1, three tests collected; outside-kind form admitted), go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/ir -run 'TestNexusFormsLayout|TestKindGeneralFilesAndForms|TestStandaloneActivityRealizationFollowsSystem|TestRetiredModel|TestRetiredFeatureFiles|TestModelFilesLeaveOutTheBuildCaches', make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model, make lint-model, make umpire-check-cases, env GOLANGCI_LINT_BASE_REV=21b9964965c8f6e383ee0a751a5fc3cb32d172db GOLANGCI_LINT_FIX=false make lint-code-fast, cmp .flow/tmp/fn132-3/artifacts-before.sha256 .flow/tmp/fn132-3/artifacts-after.sha256, sha256sum -c .flow/tmp/fn132-3/artifacts-before.sha256, sha256sum -c .flow/tmp/fn132-3/production-before.sha256, git diff --check
- PRs: