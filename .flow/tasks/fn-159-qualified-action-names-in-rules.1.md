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
**Touches:** [model/framework/Syntax.scala, model/framework/RuleSource.scala, model/framework/Rules.test.scala, model/framework/RuleSource.test.scala, model/irgen/Declarations.scala, model/irgen/test/QualifiedRules.test.scala, model/irgen/testdata/qualifiedRules/**, model/irgen/testdata/lifts/RuleRejects.scala, model/irgen/testdata/lifts/Grouped.scala, model/temporal/features/activity_mini/**/*.scala, model/migrate/**, model/README.md, model/framework/SEMANTICS.md]

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
  --steps-only` against `.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/` (16 tables/11,325 rows
  from the immutable pre-change task8 jar, including shared query roots, matching the gate's
  discovery prefix). Preserve the original 10-table `.flow/tmp/scala-dsl/mini-baseline-00d22597d2/`
  capture, every unchanged table within the complete capture, and the batch-1d baseline/map.
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
Native rule construction now refuses actual imported bare targets, using internal source-spelling capture while retaining original typed TASTy calls, codeOf rule names, ownership and positive behavior. Isolated lifter controls, task-named Scalafix source and copied-mini proof cover the mini/shared source closure; real-feature and consumer obligations remain held.

Tier: session (jev-unavailable(no_key)); project implementer routing retained
stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers one independent review to Scala batch end)

Baseline: green, fresh focused RulesTest, selected mini/shared compilation, and migration/source-citation suites. No baseline handoff or green receipt reuse. Gate classification was FULL (owner AGENTS dirt and executable source); the explicit owner override authorizes focused checks and defers broad validation, so no broad-gate skip/pass or SHIP is claimed.

R1/R2/R7: framework.RuleSourceTest compiles imported bare whole/class/2–4 targets inside/outside from, including later target positions, and checks declarer remedies and native file/line/column diagnostics. RulesTest retains its positive table, overlap and ownership assertions; its local-alias pin deliberately gains the required source coordinate. Native final: 39 tests passed. umpire.irgen.QualifiedRulesTest independently compiles only its isolated fixture, lifts ten qualified controls, refuses ten bare forms at actual source lines, and tests imports, foreign ownership and bare disabled inside/outside from. Lifter final: three tests passed. Qualified disabled inside from remains outside the lifter's existing grouping acceptance; bare disabled there now receives its qualified remedy before that existing refusal.

R3/R4: native selective/wildcard imports in directly inspectable from bodies refuse, including a qualified on target; foreign ownership remains checked. The lifter independently refuses from imports and foreign actions. No initializer is disabled to obtain lift evidence.

R5/R12 source boundary: authored mini sources are unchanged. Its activityWorker bindings retain the shared kind's worker IDs. Scratch selected lifts from the immutable task8 jar and the current frozen jar compare byte-for-byte, including source positions and waivers: b38f5fe304ee9d6c0de6aad646e39a68c36cc5ecad3b071cf169e57c801d92d9. inventory.mjs independently requires all 16 declared action IDs and located sources, with no activityWorker-owned ID. Final direct native equivalence compares all 16 selected tables/11,325 rows with the independently captured immutable pre-change selected universe, using an empty task-specific map. The original 10-table/9,878-row baseline and the complete baseline both pass their integrity scripts; the task8 jar retains hash 92c23185a3acd23ab00b81ce73dddc99295562c7f6c96d5a319b05e96778666e.

R6/migration: README, SEMANTICS and rule comments describe qualification and grouping-only from. All 30 source-cited mini guide excerpts remain exact (GuideSourcesTest). Fn159Task1QualifiedActions is actual compiled Scalafix 0.14.9 source with whole/class/multi/disabled transformation, idempotence, scope and ambiguity/shadowing refusal controls. The copied-mini runner introduces imported bare poll in scratch Dispatching.scala, applies the rule, saves accepted/legacy/qualified source evidence and requires exact accepted source restoration, idempotence and atomic ambiguity refusal. Final proof directory: .flow/tmp/mini-migration.lqpJyU; both full mini lifts and waivers compare raw bytes. Existing queue-alias tests also pass. Production loading and real-feature application remain held.

Evidence logs under .flow/tmp/fn159/task1:
- GREEN final: native-final.log, package-final.log, lifter-final.log, migrate-final.log, proof-final.log, equivalence-final.log, format-check.log, lift-before.log, lift-after.log.
- Intended RED: imported-red-final.log (all compiled bare controls fail refusal assertions before fix, pinned in commit 635a54cd56); owner-shadow-red.log (unsafe shadowed qualifier rewrites before refusal fix); migrate-noop-red.log (copied no-op rule sensitivity control fails transform/refusal assertions).
- Failed setup/implementation attempts: imported-red.log and imported-red-compiled.log (test helper/import naming collisions); native-first.log (quotes Tree.collect unavailable), native-second.log (new grouped helper/test flag collision), native-third.log (synthetic inline source span), native-fourth.log (definition spelling instead of caller), native-fifth.log (coordinate pin before its declared update), native-seventh.log (local Actor(this) violates safe initialization); migrate-first.log (deprecated args under Werror), migrate-second.log (top-level statement fixtures need an object context); proof-first.log (host requires absolute script path), proof-second.log (qualified member names wrongly retained a now-unused selective import; fixed without relaxing exact-source oracle).
- INCONCLUSIVE: proof-third.log returned zero but overlapped source formatting; no proof credit claimed. The stable final rerun in proof-final.log supplies the proof. Diagnostic instrumentation in capture-tests.log was removed.

Defect route:
- prior fixes: integrated source already implements lifter qualification; recent Syntax history inspected; bug-memory search found no specific native spelling fix. GitHub prior PR/issue checks unchecked because gh returned HTTP 401 (bad credentials).
- diagnosis: compiled imports are compiler-qualified by codeOf; native bare calls construct successfully before the fix. Direct argument spans also become synthetic during inline forwarding; macro expansion spans preserve the original target list. Final compiled controls and unchanged raw mini lift confirm the chosen capture mechanism.
- introduced by: skipped, no known-good native imported-bare revision was provided.
- base: 635a54cd56 compiled controls fail expected refusals; head: 2e8a060721 native controls pass.
- live: no live surface for framework rule construction; Cases/live remain held.

Protected: no real-feature/legacy/generated source or artifact, old equivalence baseline/map, Go/proto/Make/CI/schema, canonical gate or production loader was edited or regenerated. Lint/manual acceptance NOT RUN; broad validation and independent review remain deferred to the one Scala batch-end gate. Historical real-feature rename/consumer proof remains open and receives no source-complete credit. Root-owned AGENTS, IDEAS, MILESTONES, spec scopes and fn168 planning files remain excluded from staging; the root's task1 baseline-pointer update is included. No push, merge, rebase or history rewrite was performed; root owns local integration into umpire.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 635a54cd5688e79bcc7d1c083929aff0b7ac5cd2, 2e8a0607217eca5f35deb84e41536b9a5fbb198d
- Tests: timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/framework --test-only 'framework.*Rule*Test', timeout 600 mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini -f -o .flow/tmp/fn159/task1/scoped-model.jar, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.QualifiedRulesTest, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/migrate --test-only 'umpire.migrate.*Test', timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteQualifiedMini, FEATURES=activity_mini timeout 600 mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2 /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/mini.map --steps-only, timeout 600 mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/irgen -M umpire.irgen.lift -- --ir /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn166/task8/scoped-model.jar=model/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/frozen-before.classpath /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/frozen-before activity-mini, timeout 600 mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/irgen -M umpire.irgen.lift -- --ir /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/scoped-model.jar=model/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/scoped-model.classpath /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/frozen-after activity-mini, cmp .flow/tmp/fn159/task1/frozen-before/activity-mini.json .flow/tmp/fn159/task1/frozen-after/activity-mini.json; cmp .flow/tmp/fn159/task1/frozen-before/activity-mini.waivers.json .flow/tmp/fn159/task1/frozen-after/activity-mini.waivers.json, node /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/inventory.mjs /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task1/frozen-after/activity-mini.json, node /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/integrity.mjs; node /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-baseline-00d22597d2/integrity.mjs, timeout 600 mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/Syntax.scala model/framework/RuleSource.scala model/framework/RuleSource.test.scala model/framework/Rules.test.scala model/irgen/Declarations.scala model/irgen/test/QualifiedRules.test.scala model/irgen/testdata/qualifiedRules model/migrate/rules/Fn159Task1QualifiedActions.scala model/migrate/proof/RewriteQualifiedMini.scala model/migrate/test/QualifiedActions.test.scala, git diff --check; git diff --cached --check, baseline: green, fresh selected RulesTest/mini compile/migration-citation checks; no handoff, RED: imported-red-final.log; owner-shadow-red.log; copied no-op rule sensitivity in migrate-noop-red.log, FAILED SETUP/IMPLEMENTATION: imported-red.log, imported-red-compiled.log, native-first/second/third/fourth/fifth/seventh.log, migrate-first/second.log, proof-first/second.log; corrected before final gates, INCONCLUSIVE: proof-third.log overlapped formatting; stable proof-final.log supplies actual proof, NOT RUN: lint/manual acceptance, broad validation and independent review (owner batch-end deferral), canonical Make, Go/consumer, Cases/live/regeneration, production custom-rule loader and real-feature application
- PRs: