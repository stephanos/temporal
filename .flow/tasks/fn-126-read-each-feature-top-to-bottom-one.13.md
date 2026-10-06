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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
