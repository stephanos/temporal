---
satisfies: [R4]
---
# fn-132-group-the-nexus-and-activity-models-by.8 Admit kind and form source grouping before the Model moves

## Description
Enable kind/form admission before the Part A moves (R4 prerequisite). Production Models stay in their current locations; task 3 remains the post-move strict-lint/docs checkpoint.

**Size:** M
**Files:** `model/irgen/Structure.scala`, `Order.scala` if required, `model/irgen/test/Fixtures.test.scala`, synthetic layout fixtures.
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/layout/**, model/irgen/testdata/lifts/**, MILESTONES.md]

### Approach
- Extend the existing source classifier and level validation once; do not build a parallel validator. Package-only kind scaffolds must be recognized despite `top.headOption` being empty today.
- Prove empty general headers, one/two forms, local paired Product/System levels, single-level Nexus standalone, and own-kind shared Product refinement. A kind Product needs neither a kind System nor exports.
- Preserve flat/shared behavior, independent primary machines, compound-invalid failures, vocabulary ownership, section/export rules and the exact taskqueue exception. Reject arbitrary deeper nesting, package/path disagreement and cross-kind refinement. Do not introduce a permissive intermediate mode.
- Task 3 owns the three named R4 refusal specimens and final live-tree/docs verification; add focused negative tests for any new admission/classification rule here. Keep current production path guards until the moves replace them.
- Snapshot the current tracked IR/Case/fixture inventory. Regenerate through the canonical generator and require exact unchanged outputs for the unchanged production Models; stop on unexplained differences.

### Investigation targets
**Required:**
- `model/irgen/Structure.scala:90` — flat source grouping and level contracts.
- `model/irgen/Order.scala:424` — file/section placement.
- `model/irgen/test/Fixtures.test.scala:1796` — existing layout positives and refusal groups.
- `tools/umpire/ir/layout_test.go:324` — existing current-path guards (read only).
**Optional:**
- `.flow/memory/bug/integration/paired-level-validators-must-check-each-2026-10-06.md` — independent validation regression.

### Quick commands
Run the existing Scala fixture suite (`UMPIRE_LIFTER_UPDATE= mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`), focused `go test -count=1 -tags test_dep -p 2 ./tools/umpire/ir -run 'TestRetiredModel|TestStandaloneActivityRealization|TestLiveModelDependencyGraph'`, canonical model regeneration with `MODEL_GATE_ARGS=--skip-go-checks`, and affected Scala lint. The Scala suite starts all registered fixtures even when filtered; do not call it an isolated cheap check.
Reuse still-applicable fn-124 full baseline. Broad model/Go/runtime/artifact gates are deferred to task 2's closing Part A batch; record that obligation in the done evidence, never mint a full receipt from these focused checks.

## Acceptance
- [ ] Synthetic fixtures admit the exact future scaffold/form/local-and-kind-product shapes, including empty general files; malformed nesting, package/path mismatch and foreign-kind refinement are refused by the intended diagnostic.
- [ ] Every existing flat/shared structure regression remains enforced, including independently invalid or missing Product/System primaries, canonical level vocabulary, sections/exports and the exact taskqueue exception; no blanket exemption is added.
- [ ] Production Model sources/exports stay unchanged and regenerated IR/Cases/fixtures match the recorded baseline exactly, with focused fixture/layout/lint evidence linked and broader gates explicitly deferred to task 2.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
