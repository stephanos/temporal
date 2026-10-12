---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R12]
---
# fn-159-qualified-action-names-in-rules.1 Require qualified on targets; from(X) groups without importing; rename the bound worker to activityWorker

## Description

Complete the source/lift proof of qualified action targets and grouping-only `from`. The main source
implementation is integrated; re-anchor to it rather than replaying the historical standalone edit.
The owner Scope limits current implementation and proof to the mini feature, framework, lifter and
shared code. Real-feature source, IR/Cases and historical pins stay untouched for the sweep.

**Size:** L
**Files:** framework rule construction and focused tests, IR rule lifting and isolated fixtures,
mini feature only where a missing contract requires it, author docs and the shared migration-rule source.
**Touches:** [model/framework/Syntax.scala, model/framework/RuleSource.scala, model/framework/Rules.test.scala, model/framework/RuleSource.test.scala, model/irgen/Declarations.scala, model/irgen/test/QualifiedRules.test.scala, model/irgen/testdata/qualifiedRules/**, model/irgen/testdata/lifts/RuleRejects.scala, model/irgen/testdata/lifts/Grouped.scala, model/temporal/features/activity_mini/**/*.scala, model/migrate/**, model/README.md, model/SEMANTICS.md]

### Approach

- Inspect actual integrated `Rules.block`/`derivedRules`, inline `on` variants, the lifter's
  `qualified`/`fromBlock`/`declaredBy`, and existing refusal fixtures before changing production.
  Qualified whole-action, input-class and two-to-four-target forms must continue to work. Class
  argument dots do not qualify a bare action. Apply the qualification contract inside/outside
  `from` and retain ownership checks. The owner decision also includes `disabled` targets.
- First pin the gap in current R2 evidence: native `Rules.test.scala` covers a local alias but its
  comment assigns imported bare names to the lifter, because `codeOf` compiler-qualifies them.
  Add actual compiled native imported-bare controls inside/outside `from`, for the applicable
  whole-action/class/multiple-target forms, before fixing the source spelling capture. Preserve
  R2's framework construction refusal; do not redefine it as lifter-only or waive these controls.
  Keep any narrowly needed source-spelling helper internal to the framework. Do not add a public
  authoring construct or break original TASTy rule collection.
- The lifter must independently refuse bare references and imports in a `from`, with the action,
  actual source position and qualified remedy. Test whole/class/multi-target forms and foreign
  ownership; do not satisfy refusal tests by disabling native initialization or modifying unrelated
  expected snapshots. Any native source-capture adjustment must leave qualified effects unchanged.
- The mini already declares the bound `activityWorker` and qualified action uses. Prove its action
  IDs still come from the shared kind declaration and its selected native tables/lift are unchanged.
  Do not edit standalone/Nexus/legacy sources or invent extra mini constructs to justify real-only work.
- Add the actual fn-159 rule source under the convention established by fn-166.9, with focused
  transformation, idempotence, ambiguous/refusal and compiled copied-mini controls. Prove the
  transformed qualified source agrees with the accepted mini's identities/meaning. Real-feature
  application and production custom-rule loading remain held; a direct rule-source test is not
  loader evidence. Avoid a competing general-purpose transformation engine.
- Keep docs accurate and show only qualified actions/grouping-only `from`; preserve the complete
  mini examples and their checked source citations. Update cited line numbers when needed, not
  the source-matching assertion or refusal oracle.

### Source and consumer boundary

Current source closure proves framework/lifter refusals, qualification, selected identities and native
meaning on mini/shared input. Historical real-feature rename proof, managed IR/Case regeneration,
Go/lint consumer gates and the canonical Make gate remain at the owner's sweep. Preserve original
R1–R7/R12 requirements collectively: this source receipt is not consumer/full-set completion.

### Verification

- Fresh targeted baseline, then `scala-cli --server=false` mini/shared compile and focused native
  rule tests plus the new isolated lifter qualification class. The broad `Fixtures` class starts all
  fixtures in `beforeAll`; do not invoke it just to filter one test by name.
- Scratch-only pre/post selected mini lift comparison, independently checked action inventory and
  source-coordinate/identity preservation. Do not regenerate checked-in outputs or inspect/lift
  excluded real features to earn current proof.
- Direct `FEATURES=activity_mini` Scala `--check-equivalence <absolute mini baseline> <mini map>
  --steps-only`, preserving `.flow/tmp/scala-dsl/mini-baseline-00d22597d2/` and the batch-1d baseline/map.
  An empty map is appropriate when no mini declaration identity changes; every nonempty rename
  must be justified and used, never broad normalization. This is not canonical Make credit.
- Focused migration-rule tests and isolated copied-mini compiled proof; source-citation tests for
  changed docs. Formatting/diff checks scoped to changed files. Lint is NOT RUN under owner override.
- Broad validation and independent review are deferred to the ONE mini batch-end gate. Preserve
  earlier findings/receipts; do not claim an unrun suite passed or invent SHIP.

## Acceptance

- [ ] The IR generator refuses every applicable bare `on`/`disabled` target inside/outside `from`,
  names its action and source position, and spells out the qualified form; class arguments are not
  treated as action qualification. Accepted whole-action/class/multi-target forms remain valid.
- [ ] Framework construction refuses the corresponding bare `on` targets, including actual
  imported names inside/outside `from`, rather than proving only local aliases. Native and lifted
  positive/negative controls independently exercise the contract.
- [ ] `from(X)` retains its foreign-action ownership refusal and imports no members; imports in
  a `from` are refused by the lifter and natively where the framework can inspect them.
- [ ] Mini/shared source and focused fixtures use qualified targets, mini binds `activityWorker`
  to the kind's existing action identities, its scratch lift/native tables preserve meaning, and
  excluded real sources/artifacts, pins and the old baseline/map are unchanged.
- [ ] README/framework documentation uses qualified actions and grouping-only `from`, with any
  updated mini guide excerpts still checked against their cited source.
- [ ] Actual migration-rule source, isolated mini transformation/idempotence/refusal controls and
  compiled identity/meaning proof exist under the shared convention; production loader and
  real-feature applications are explicitly held.
- [ ] Focused source commands and direct mini native equivalence are green, with all failed setup
  attempts retained honestly. Canonical Make, Go/consumer, regeneration/live, lint, broad validation
  and independent review credit is not claimed for this source closure.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
