---
satisfies: [R6, R7, R8]
---
# fn-139-actor-grouped-rules-per-rpc-actions.5 Activity product and System rules in from blocks, grouped by meaning; activity on the shared Outcome

## Description
Rewrites the activity product's and System's rules into actor groups and moves the activity onto the shared `Outcome`. The rules use `from(client)`, `from(worker)` and blocks for the process, the timers and the deadlines, and the worker's answers are grouped by meaning (R6). The activity's `notFound` becomes `rejects(Rejection.notFound)` (R8, R9). The cases keep reading `in` here, and task .7 renames them.

**Size:** M
**Files:** model/temporal/features/activity/standalone/Standalone.scala (remove `Outcome` and `given Ok[Outcome]`), .../product/Product.scala, .../system/System.scala, .../system/Record.scala (`rejected = …`), .../system/WithTaskQueue.scala (outcome references only; `closedAnswer` stays until task .8)
**Touches:** [model/temporal/features/activity/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- System rules (System.scala `object rules`): `from(client) { import client.*; … }` holds `start`, then the shared not-found block `on(pause, unpause, requestCancel, terminate)`, then one block per control. `from(worker) { import worker.*; … }` holds `poll`, then the final answers that settle a held attempt (`respondCompleted`, `respondFailed(Failure.fatal)`), then the answers that hand the attempt back (`respondFailed(Failure.retryable)` per phase, `respondCanceled`). `from(process)`, `from(timers)` and `from(deadline)` hold the rest. Every `on` is a block with one case per line.
- Keep each rule's phases, guard and effect exactly as before. Only the grouping and the order of blocks change. Preserve every comment, moving each with the rule it explains.
- Product rules (Product.scala:59-81): the same grouping over the product's actions. `disabled(process.stop)` (:86) stays outside any `from` (spec, Decision Context).
- Layout: every block is brace, newline, one case per line, closing brace on its own line (spec, Architecture: rule layout).
- Name clashes: `import client.*` is ambiguous with a top-level definition of the same name in the same file (probe). Qualify the reference if one appears, and never rename an action to dodge it.
- Shared outcome: delete the activity's `enum Outcome` and its `given Ok`. Import the shared `Outcome` and `Rejection` explicitly from the framework's namespace (task .1). The product reads `Outcome` from its package in another file today, so the explicit import is what switches it. `def notFound = reject(Outcome.notFound, s)` becomes rows written `~> rejects(Rejection.notFound)`. Capability arguments `rejected = cited(Outcome.notFound, …)` (Product.scala:94) and `rejected = Outcome.notFound` (Record.scala:273) become `Outcome.rejected(Rejection.notFound)`.
- Equivalence pin (R7): the rule table per (action, class, phase) is unchanged. Add a munit test in the activity's test file that builds both machines and checks their step functions against a table recorded from the pre-task tree, over every state and class. Recording the table needs no regeneration. The batch regeneration's diff may show the outcome's type and case names (`notFound` → `rejected(notFound)`), positions, and step bindings in the new grouping's order (bindings are written in the order each action is first named: `order` → `addAllSteps`). Record the old and new binding order of both machines in the commit message, so that a Case whose witness changes only by that order can be traced to it.

### Investigation targets
**Required:**
- model/temporal/features/activity/standalone/system/System.scala:170-215 (rules) and :140-170 (effects)
- model/temporal/features/activity/standalone/product/Product.scala:40-100
- model/temporal/features/activity/standalone/system/Record.scala:265-280
- model/umpire/Syntax.scala: the shared types from task .1
## Acceptance
- [ ] The System's rules read `from(client)`, `from(worker)`, `from(process)`, `from(timers)` and `from(deadline)`, with one shared not-found block for the four controls and the worker's answers grouped as settling answers, then answers that hand the attempt back. The product's rules use the same form.
- [ ] The activity has no outcome enum of its own. Its not-found rows are `rejects(Rejection.notFound)`, and its capability `rejected` arguments are `Outcome.rejected(Rejection.notFound)`.
- [ ] The munit equivalence test shows each machine's step function unchanged over every state and class, apart from the outcome value's new name.
- [ ] The Models compile, and the activity tests pass.
## Done summary
Grouped the standalone Activity Product and System rules by actor and semantic meaning, moved Activity to the shared Outcome/Rejection model, preserved all step behavior with an exhaustive 6,435-pair equivalence pin, and migrated handwritten consumers. Product disabled process.stop remains outside from; checked generated artifacts remain deferred to the batch regeneration.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: faa90302934868962601866cef272c9883594cb0
- Tests: PASS: Model/framework Scala suite (65 tests), PASS: standalone activity pins (4 tests, included in suite), PASS: model/build/model-scala.jar packaging, PASS: tools/umpire/check compile-only surface with test_dep, PASS: make lint-model-models (known JDK 27 Scalafix warning, exit 0), DEFERRED: checked IR/Cases regeneration and generated-consumer assertions until the single batch regeneration
- PRs: