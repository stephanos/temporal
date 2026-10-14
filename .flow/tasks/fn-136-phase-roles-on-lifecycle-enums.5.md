---
satisfies: [R9]
---
# fn-136-phase-roles-on-lifecycle-enums.5 Retire role-set states predicates; callers read roles directly (bounded IR change)

## Description
Deletes every `states` predicate whose body is now a single role test. Callers across the activity and Nexus models read the role with `in[R]`/`p.in[R]` instead. This is the one step of the spec that changes IR, within R9's bound.

**Size:** M
**Files:** model/temporal/features/activity/standalone/{product/Product.scala, system/System.scala, system/Record.scala, system/WithTaskQueue.scala}, model/temporal/features/nexus/{workflow/system/System.scala, workflow/system/TrustingCaller.scala, standalone/system/System.scala, product/Product.scala}, model/ir/**, model/cases/**, model/README.md, model/SEMANTICS.md, MILESTONES.md
**Touches:** [model/temporal/features/**, model/ir/**, model/cases/**, model/README.md, model/SEMANTICS.md, MILESTONES.md]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Inventory first. For each machine, list each `states` member and classify it: role set (retire), derived role name such as `created` (keep the name, `is` body), not a role (keep), or read by a capability field or another named-def-only caller (keep until fn-137). Known keeps: `terminal` in the activity product (Product.scala:93), Record (:272) and Nexus standalone (System.scala:149), all read by Closable's `terminal` field and retired by fn-137 task .6. Claims accept an inline lambda, but the lifter then synthesizes a new IR Function for it (irgen/Syntax.scala:273-275), which R9's diff bound doesn't allow. So a predicate a claim names stays a named definition with an `is` body: `productTerminal` (nexus Product.scala:111) is a keep.
- Callers to rewrite: about 10 in activity System.scala, 6 in nexus workflow System.scala, 5 in TrustingCaller, 5 in the activity product, 4 in nexus standalone, 3 in the nexus product, 2 in Record (from planning grep `states\.(terminal|live|held|waiting|running|terminalPhase|created|productTerminal)`). Rule headings use `in[R]`, everything else `x.phase.is[R]`. Keep the existing comments.
- WithTaskQueue's four reads of `ActivityRecord.states` (`terminal` for Closable, `running` as `started` alone) are all keeps. Don't touch them, and don't rewrite capability fields here.
- Remove a `states` object that ends up empty (the Nexus product's may, if `productTerminal` goes).
- IR diff bound (R9): snapshot model/ir and model/cases under `.flow/tmp/fn-136.5/` before editing, regenerate with `make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model`, then classify every changed line with a script: removed retired Function, inlined case set at a former call, or a Definition ID / exploration identity / Case reference following from those. Any unclassified line stops the task. Query answers and Check verdicts must be unchanged (compare receipts).
- Docs: README `states` section (what stays in it; role tests via `in[R]`/`p.in[R]`), SEMANTICS role-test forms. Update the fn-136 entry in MILESTONES.md.

### Acceptance
- [ ] No `states` predicate whose body is a single role test remains, except the listed keeps, each with its reason in the done summary.
- [ ] All former callers of retired predicates, including TrustingCaller and the compositions in activity and Nexus System.scala, use `in[R]` or `p.in[R]`. Capability fields and WithTaskQueue are untouched.
- [ ] The diff-classification script reports zero unclassified IR or Case lines. Every Query answer and Check verdict is unchanged.
- [ ] README, SEMANTICS and MILESTONES are updated; `make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model` and `make lint-model` pass.


## Done summary
Retired 11 lifecycle states aliases and rewrote their Activity and Nexus callers to direct role reads using the settled when[R] rule-heading and p.in[R] value forms. Kept the three terminal predicates still consumed by Closable, the claim-named productTerminal predicate, all non-role subsets/helpers, and every WithTaskQueue read. Added an executable inventory guard and updated authoring/semantic docs. No IR, Cases, fixtures, or capability fields were regenerated or changed; the bounded generated diff and Query/Check equivalence remain deferred to the DSL batch regeneration.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e6dc5dc033
- Tests: scala-cli test model/project.scala model/umpire model/temporal (50 passed), scala-cli test model/irgen --test-only umpire.irgen.RolesSuite (3 passed), make lint-model-models lint-model-syntax (exit 0), scalafmt check over changed Scala (exit 0), git diff --check (exit 0), batch deferral: generated IR/Case classification and Query/Check proof at the single regeneration against 96de1fd92d
- PRs:
## Acceptance
- [ ] TBD
