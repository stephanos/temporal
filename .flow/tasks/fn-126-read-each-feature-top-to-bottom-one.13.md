---
satisfies: [R5, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.13 Require both level machines and preserve loss-budget regression coverage

## Description
Close the remaining original R20 and R5 regression-coverage gaps found by the periodic quality audit. Structure.scala currently accepts canonical Product/System files without a refinement pair, then skips primary-machine and level-ownership checks when systemDef is absent. A System.scala containing only types and Product-only exports must not evade the missing-System check. Resolve and require both primary machine declarations independently before checking their names, prefix, refinement and vocabulary. Preserve allowed Product elaborations and the named taskqueue shared-vocabulary exception; multiple Product-file machines must not disable missing-refinement checks.

Restore the lost-response regression in StandaloneActivityPins.test.scala. The old repeated-loss-disabled assertion was replaced with end(state), but end does not forbid transitions. For both resulting states, assert through the existing rules/table machinery that another loss is disabled, independently of the end assertion. Do not add a production API only for this test or change Model behavior.

**Touches:** model/irgen/Structure.scala, model/irgen/testdata/**, model/irgen/*.test.scala, model/irgen/test/**, model/temporal/features/standaloneactivity/StandaloneActivityPins.test.scala, model/README.md, model/SEMANTICS.md, tools/umpire/model/*_test.go, tools/umpire/model/testdata/**, model/ir/**, model/cases/**, MILESTONES.md

Limit fixture, documentation and generated-artifact changes to those directly affected by the checks. Flow lifecycle files remain writable. Add red/green refusals for a types-only missing System, the corresponding Product case, and missing refinement with Product elaborations, plus a passing elaboration case. Model tables, Queries and Cases must remain unchanged.

This is the final batch gate after .12. Regenerate and inspect required artifacts, then run the full model gate with --skip-go-checks, Scala lint, read-only batch-base Go lint, the instrumented full Go tooling suite with -json -tags test_dep -p 2 -timeout 30m, and Case/fixture/canary smoke checks under the shared heavy-suite lock. Reuse evidence when relevant inputs still apply; after fixes repeat only affected checks unless broader results are invalidated. Per-task review, outer completion review and the quality-audit fix decisions must all be resolved before the conductor closes fn-126.
## Acceptance
- [ ] Canonical level files require both primary Product and System machine declarations independently of the presence of a refinement pair. Types-only missing-level fixtures and missing-refinement-with-elaborations fixtures are refused at their source; legitimate elaborations still pass.
- [ ] Existing name/prefix/refinement and level-owned-vocabulary checks cannot silently skip on a missing or ambiguous primary definition; the documented taskqueue exception remains narrow.
- [ ] Both lost-response resulting states explicitly disable another loss through rules/table evaluation. The end-state check remains an independent assertion, and removing the loss-budget guard would fail the restored regression.
- [ ] Regeneration/projection evidence proves unchanged Model tables, Query answers, Case behavior and verdicts; only new refusal-fixture artifacts or declared source metadata may differ.
- [ ] Full .12/.13 batch gates pass with retained commands, logs, exit statuses and separate Go wall time. Per-task review reaches SHIP and Flow done is verified; the parent stays open for the conductor's completion review and final audit resolution.


## Done summary
Both canonical level files now require an independently selected, unambiguous primary machine before pair checks. The standalone activity regression evaluates the existing guarded loss binding for both resulting states, independently of its retained end-state assertion; no production Model or API changed.

Tier: session (jev-unavailable(no_key))

### Coverage and reproduction

R20: source-exact refusals cover types-only missing System and Product files, independent malformed names on the surviving level, ambiguous Product/System declarations, and missing System refinement with a Product elaboration. The layout template exports and lifts elaborations from both canonical files. Primary selection is restricted to two-level layouts, preserving all three prior structure-refusal expectations and exactly the temporal.shared.taskqueue vocabulary exception.

R5: umpire.StandaloneActivityPins checks the loss binding initially returns both choices, then returns Nil for each spent-budget state. The separate end assertion remains unchanged. In the ignored scratch mutant, replacing where(_.lossAvailable) with always still passed the original test (one selected test), but failed the restored rule-evaluation assertion. Production Record.scala was never edited.

Baseline: green focused structure (three selected tests) and original loss (one selected test). No full baseline handoff was claimed. The first zero-selection structure invocation and the first misplaced-fixture reproduction are inconclusive and retained, not passes. The corrected new structure test failed because the original lifter returned exit 0 for the malformed tree. Checkpoint 2b2e7a3c40 records the failing reproduction before fix 32fe892a19. Final focused checks selected four structure tests, one loss test and one multi-machine layout test; all exited 0.

The conductor authorized model/irgen/test/** as the existing intended test surface; its exact Touches correction was written through flowctl and committed with the reproduction. Prior-fix history and memory were checked. GitHub PR/issue lookup was unchecked (HTTP 401); bisect was skipped because no known-good revision of the newly exposed structure cases was available. See .flow/tmp/fn-126/task13/baseline.md and regressions.md.

### Behavior freeze

make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model exited 0 in 162 wall seconds. Strict comparisons of the frozen base snapshot against current IR, Cases, generated fixtures, canary files and generated names are byte-identical; no generated-artifact diff shipped.

The existing rename wrapper was inapplicable: idmap reported duplicate local State/record:phase type shapes despite zero renamed IDs/types. Its proof.log and exit 1 remain as inconclusive evidence. The conductor approved direct identity proof with the same projtool, Case generator and compare machinery, without normalization or diagnostic drops. proof-identity.log records exit 0 in 32 seconds: byte-identical 88,856,698-byte projections; seven IR models, 21/21 Case artifacts, 9/9 generated fixtures, 2/2 canary files, laws, lint results and names equal. Script: .flow/tmp/fn-126/task13/prove-identity.sh; frozen inputs: task13/before; outputs: task13/prove.

### Full .12/.13 batch gates

All logs below are in .flow/tmp/fn-126/task13. Exit files record 0. The shared heavy-suite lock serialized every heavy gate; Go used test_dep and -p 2.

- make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model: full-model.log, 147 wall seconds; receipt .flow/tmp/green-receipts/32fe892a-model.json.
- make lint-model: full-scala-lint.log, 17 seconds. Inherited JDK 27/scalafix NoSuchFieldException:path warnings remain in the successful log; none were silenced.
- GOLANGCI_LINT_BASE_REV=4e0c4c5b3783e5cfcd4e038d113d615d9b259474 GOLANGCI_LINT_FIX=false make lint-code-fast: full-go-lint.log, two seconds, zero findings. The base includes both tasks; no main-relative result or autofix is claimed.
- go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/...: full-go.jsonl, full-go.stderr.log (empty), full-go.exit and full-go.elapsed-seconds (108 wall seconds, not summed package times). full-go-summary.json records 17 passing packages, 2,866 passing tests/subtests, ten skips and zero failures. Receipt: .flow/tmp/green-receipts/32fe892a-unittest.json.
- make umpire-check-cases && make umpire-check-fixtures && make canary-check-case: full-smoke.log, 46 seconds. These quiet check-mode generators validated the populated artifact sets; receipt .flow/tmp/green-receipts/32fe892a-smoke.json.

No full receipt was minted from a focused command. The full classification remains executable (gate-classify.log); no suites were rerun after the unchanged-artifact SHIP review.

### Review and handoff

stage: impl-review - ran [2026-10-06T07:51:52Z..2026-10-06T07:54:05Z], SHIP. Explicit CLI spec codex:gpt-6.1-sol:high overrides the environment default. Same-family review, not a cross-family independence claim. All three axes returned SHIP with no findings or unaddressed R-IDs; no fix loop or memory capture was triggered. Receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-126-read-each-feature-top-to-bottom-one.13.json. Draws: .flow/review-fanout/c42897aa85c0479da09bbab2dca1e690; commands, route, merge plan and finalize output: task13/review-*.

stage: plan-sync - skipped(config: planSync.enabled=false).
stage: tracker-sync - skipped(policy: inactive bridge; .flow/tmp/fn-126/task13-sync-active.json).

The task closes the two original audit gaps recorded in .flow/tmp/fn-126/quality-audit-round1.md. The conductor owns the separate spec-completion review and final audit resolution. Keep the fn-126 MILESTONES block and parent spec open until that review is verified SHIP; no push or history rewrite occurred.
## Evidence
- Commits: 2b2e7a3c4063889f27eb24b28e48295412167105, 32fe892a19eaae8d6e096bb5b91000668b9c9c64, 322758581325b6562c5f627daefca2c841feb9a8
- Tests: mise exec -- scala-cli test model/irgen --test-only umpire.irgen.Fixtures -- --tests '*structure*', mise exec -- scala-cli test model/project.scala model/umpire model/temporal --test-only temporal.features.standaloneactivity.StandaloneActivityPins, mise exec -- scala-cli test model/irgen --test-only umpire.irgen.Fixtures -- --tests '*independent unambiguous*', mise exec -- scala-cli test .flow/tmp/fn-126/task13/loss-mutant/model/project.scala .flow/tmp/fn-126/task13/loss-mutant/model/umpire .flow/tmp/fn-126/task13/loss-mutant/model/temporal --test-only temporal.features.standaloneactivity.StandaloneActivityPins, mise exec -- scala-cli test .flow/tmp/fn-126/task13/loss-mutant/model/project.scala .flow/tmp/fn-126/task13/loss-mutant/model/umpire .flow/tmp/fn-126/task13/loss-mutant/model/temporal --test-only umpire.StandaloneActivityPins, mise exec -- scala-cli test model/project.scala model/umpire model/temporal --test-only umpire.StandaloneActivityPins, mise exec -- scala-cli test model/irgen --test-only umpire.irgen.Fixtures -- --tests '*layout template*', make fmt-model, make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model, bash .flow/tmp/fn-126/task13/prove-identity.sh, make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model, make lint-model, GOLANGCI_LINT_BASE_REV=4e0c4c5b3783e5cfcd4e038d113d615d9b259474 GOLANGCI_LINT_FIX=false make lint-code-fast, go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/..., make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
- PRs: