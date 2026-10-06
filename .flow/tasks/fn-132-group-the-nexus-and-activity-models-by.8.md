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
Extended the existing Structure classifier to admit the closed kind/form catalog before the production moves. Synthetic fixtures prove package-only headers, one or two forms, local paired levels, single-level Nexus standalone, and a shared kind Product without a kind System or exports; focused refusals retain strict ownership and placement.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-06T10:42:32Z..2026-10-06T11:13:11Z]

### Verification

The pre-edit baseline was green. It ran freshly because the conductor's reuse direction arrived after that baseline sequence had started. Persisted task base is 21b9964965c8f6e383ee0a751a5fc3cb32d172db; the original whole-goal spec base remains untouched.

Final affected verification logs are under .flow/tmp/fn132-8/review-fixed-*. Scala exited 0 in 166 seconds with 86 passing tests and one explicit platform skip. Formatting exited 0, affected Scala lint exited 0 in 24 seconds, canonical skip-Go regeneration exited 0 in 201 seconds, and make umpire-check-cases exited 0 in 27 seconds. The named Go IR layout/path tests passed at the checked checkpoint and remain applicable because their source inputs, production paths and generated outputs are unchanged. That Go result is reused, not a new run.

The exact tracked 69-file IR/Case/functional/canary/lifting-golden inventory equals the pre-edit inventory. All 33 production source hashes also match. Evidence is artifacts-before.sha256, artifacts-review-fixed.sha256, review-fixed-artifacts-check.log and review-fixed-production-check.log in .flow/tmp/fn132-8. Production Models and exports were not moved or edited.

Existing flat/shared structure groups, independent primary/refinement checks, compound-invalid refusals, canonical Phase/State/Fact ownership, sections/exports and the exact taskqueue exception remain enforced. Diagnostic red covers each new admission boundary. The review found three additional bypasses; review-guards-red-corrected exited 1 in 169 seconds because the previous classifier admitted the wrong-parent, unrefined sibling, valid derived sibling and type-only wrong-folder specimens. Their final assertions are green. The earlier bare Derived(RelayProduct) probe was confounded by invalid DSL and is not banked as admission red.

This filesystem aliases Nexus.scala and nexus.scala to one inode. The case-variant duplicate specimen is explicitly skipped here, while the ordinary Extra.scala duplicate refusal runs everywhere. The earlier alias-confounded duplicate probe is inconclusive. The fixture suite launches every registered concurrent fixture even when filtered; an earlier zero-collected attempt is also inconclusive and minted no receipt.

Affected scalafix retains the inherited JDK 27 ClassPathOps warning with exit 0. The existing fn124 scoped-rule execution proof is reused for that unchanged environment issue; no diagnostic rule was silenced.

### Review and remaining boundaries

SHIP receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-132-group-the-nexus-and-activity-models-by.8.json. The same primary session 01a110ce-d4bd-7b90-af47-d7d4687ed98d resumed after the repairs; all three merged findings are marked fixed. The required memory capture updated the existing paired-level validator entry through flowctl and preserved prior lessons.

BROAD_GATE_DEFERRED:fn-132.2: the full Model/Go/runtime/artifact/lint boundary runs after both Part A moves. Final gate classification is full, but this explicit batching obligation overrides broad execution here. Focused checks minted no full receipts. Offline checks make no new backend or deployment claim; fn124 evidence is not a formal baseline handoff.

Package-only units emit no TASTy. The new fixture calls use the proven direct --server=false packaging mode because Bloop printed errors despite exit 0. Error parsing and Tools.orFail remain unchanged; the original lamp fixture keeps its compiler mode and exact six-file inventory. Task .1 must carry the necessary canonical packaging option within its approved scope before empty production headers are introduced. Task .3 retains its three named R4 specimens, final docs and live-tree verification. The parent spec remains open; planSync is false and tracker projection is inactive.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9afcce03cad0159d403fd35037d6fa78c25d1466, d014ca09711a9fa5033a5fcef4b61bd4d3d33935, dbe220bf48f97ad5dd14aa772347077688bb78cf
- Tests: UMPIRE_LIFTER_UPDATE= mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen, go test -count=1 -tags test_dep -p 2 ./tools/umpire/ir -run 'TestRetiredModel|TestStandaloneActivityRealization|TestLiveModelDependencyGraph', mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/irgen, make lint-model-irgen lint-model-irgen-lifts, make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model, make umpire-check-cases, cmp .flow/tmp/fn132-8/artifacts-before.sha256 .flow/tmp/fn132-8/artifacts-review-fixed.sha256, sha256sum -c .flow/tmp/fn132-8/artifacts-before.sha256, sha256sum -c .flow/tmp/fn132-8/production-before.sha256, BROAD_GATE_DEFERRED:fn-132.2: full Model/Go/runtime/artifact/lint boundary after both Part A moves; focused checks mint no full receipts
- PRs: