---
satisfies: [R5, R6, R7]
---
# fn-136-phase-roles-on-lifecycle-enums.3 Nexus models adopt roles; docs; close

## Description
The Nexus workflow system, Nexus standalone system and Nexus product declare roles with identical IR, their role-carrying refinements join the closedness test, and the docs describe roles. Task .5 closes the spec, after the role-set predicates retire.

**Size:** M
**Files:** `model/temporal/features/nexus/workflow/system/System.scala`, `model/temporal/features/nexus/workflow/system/TrustingCaller.scala` (callers only, if any signature needs it), `model/temporal/features/nexus/standalone/system/System.scala`, `model/temporal/features/nexus/product/Product.scala`, a closedness test beside the Nexus models, `model/README.md`, `model/SEMANTICS.md`, `model/ir/**` (positions only)
**Touches:** [model/temporal/features/nexus/**, model/README.md, model/SEMANTICS.md, model/ir/**, model/cases/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Workflow system: `unscheduled` none, `scheduled` Waiting, `backingOff` Retrying, `started` Held, closures. Standalone: `unstarted` none, `scheduled` Waiting, `started` Held, closures (cancel request stays a Boolean field). Product: per its phases.
- Same rule as task 2: keep predicate names and callers (`terminalPhase`, `running`, `waiting`, `created`, `terminal`, `over`, `live`, `productTerminal`); replace a body only when it is a single `in(...)` listing exactly a role's cases in declaration order. `created` (an OR of two predicates), `over` and `end` keep their bodies. `productTerminal` is a single `in(...)` over the product's `Closed` cases, so it keeps its `State` parameter and its body becomes a role test on `s.phase`. Leave the inline `phase == started` in the start-to-close rule as it is.
- Add the Nexus workflow and Nexus standalone refinements to the closedness test by name; the task queue refinement is not checked.
- Same identity proof as task 2, snapshot under `.flow/tmp/fn-136.3/`.
- Docs: README machine-layout section (role mixins, role tests in `states`), the supported step-function subset (type tests and type patterns against roles), the role vocabulary; SEMANTICS expression and pattern semantics (role test meaning) and the refinement rule (closedness check, Scala-side). Keep existing comments.
- Validate: `flowctl validate --spec fn-136-phase-roles-on-lifecycle-enums --json`. The MILESTONES update and the close move to task .5.

### Investigation targets
**Required** (read before coding):
- `model/temporal/features/nexus/workflow/system/System.scala:21-22,81,91-130,226-246` - phases, end, states, refinement, timer rules
- `model/temporal/features/nexus/standalone/system/System.scala:14-50,104-106` - phases, states, refinement, rules
- `model/temporal/features/nexus/product/Product.scala:13-37,82,111` - phases and the State-typed `productTerminal`
- `model/README.md:446,641-670,750` - step-function subset and machine layout
- `model/SEMANTICS.md:46-79,147-155` - expressions, patterns, refinement rule

**Optional** (reference as needed):
- `model/temporal/features/nexus/workflow/system/TrustingCaller.scala:28-67` - reads NexusSystem.states

### Key context
- fn-132.5 (a gate of this spec) moves `NexusProduct` into `features/nexus/`; use the post-fn-132 paths.

## Acceptance
- [ ] Nexus workflow, standalone and product phase enums carry roles as specified.
- [ ] Replaced predicates keep their names and callers; only single-`in(...)` bodies equal to a role's cases were replaced; `productTerminal` keeps its `State` parameter and reads `Closed` of `s.phase`; `created`, `end` and the start-to-close inline comparison are unchanged.
- [ ] The closedness test checks the Nexus workflow and Nexus standalone refinements by name, with no violations.
- [ ] Identity proof as in task 2: IR identical except positions, Cases byte-identical, every Query answer and Definition ID unchanged.
- [ ] README and SEMANTICS describe role traits, role tests and the closedness check.
- [ ] `make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model`, `make lint-model` and `flowctl validate --spec fn-136-phase-roles-on-lifecycle-enums` pass.

## Done summary
Roles are now declared on the phase enums of the Nexus workflow System, the Nexus standalone System and the Nexus Product. Each `states` predicate whose body was a single `in(...)` over exactly one role's cases is now a `p.in[R]` role test, and its name, signature and callers are unchanged. A new munit test checks closedness on both Nexus refinements, by name. README and SEMANTICS now describe roles.

stage: impl-review - skipped(config: REVIEW_MODE=none - DSL batch: reviews run once at the batch's end)
Tier: lane B, IMPLEMENTER claude-opus-5-5 at high

### What changed
- **Workflow System** (`workflow/system/System.scala`): `unscheduled` has no role. `scheduled` is Waiting, `backingOff` Retrying and `started` Held, and the closures are Succeeded, Failed, Canceled and TimedOut. `terminalPhase` is now `p.in[Closed]`, `running` is `p.in[Live]` and `waiting` is `p.in[Waiting]`. Unchanged: `created`, `end`, the `NexusCaller.end` and `TrustingCaller` callers, and the inline `phase == started` in the start-to-close rule.
- **Standalone System**: `unstarted` has no role. `scheduled` is Waiting and `started` Held, and the closures are Succeeded, Failed, Canceled and Terminated. The cancel request stays a Boolean field. `terminal` is now `p.in[Closed]` and `live` is `p.in[Live]`. Unchanged: `phase`, `over`, `created`, and the inline `in(...)` cases of the terminate rule. Those cases are not `states` predicates, so R5 leaves them alone.
- **Product**: `scheduled` is Waiting and `started` Held, and the closures are Succeeded, Failed, Canceled, TimedOut and Terminated. `productTerminal(s: State)` keeps its `State` parameter and is now `s.phase.in[Closed]`.
- **Closedness test** (`model/temporal/features/nexus/product/NexusRoleRefinements.test.scala`): it checks `workflow.system.NexusSystem.refinement` and `standalone.system.NexusSystem.refinement`. For every `Finite` System state, it collects each state whose `Closed` role differs from that of its `toProduct` image, together with the image, and requires that list to be empty. The task-queue refinement is not checked.
- **Docs**: README has a new "Phase roles" bullet in the lifted-subset list. It covers the vocabulary, the declaration, how `p.in[R]`, `when[R]` and type patterns lift, the lint on `isInstanceOf`, and the refusals. It also gets one clause on role tests in `object states` and one on the closedness test under `object refinement`. SEMANTICS defines a role test as case-set membership (after Patterns) and states the Scala-side closedness rule under Machines 6. These edits touch only those paragraphs.

### Decisions (owner unavailable; my calls)
- **The test's location and package.** The lifter's layout rule allows only `Nexus.scala` in `features/nexus/`, and it refused the test file there ("nexus holds one general feature file named after its kind"). A test in a feature package would also be held to the structure and order lints. So the test sits in `product/`, beside the Product that both refinements map onto, and is declared in `package umpire`, as `StandaloneActivityPins.test.scala` is.
- **No framework closedness check.** fn-136.2 (lane C) adds the check to `model/umpire/Refine.scala`, and it does not exist yet. As instructed, I did not implement it. The test has its own four-line `unclosed` helper. **Dependency:** once fn-136.2 lands, swap the helper for fn-136.2's framework function. That is a one-line change per test, for whoever integrates or for fn-136.5. Until then, the behaviour R6 requires (a violation fails the test and names the state and its image) holds, as the negative control shows.

### Expected IR delta at the batch regeneration
Source positions only, in four files: `nexus-workflow.json`, `nexus-workflow-control.json`, `nexus-standalone.json` and `nexus-standalone.laws.json`. Every changed line is a `"line"` value or the `:<line>` suffix of a `"position"` string (3510 `line` and 10 `position` changes). No case list, name, Definition, Function or Query changes. The shifts come from the enum lines growing into one line per case:
- System.scala lines after line 22: +7 in nexus-workflow and nexus-workflow-control.
- Product.scala lines after line 14: +6 in nexus-workflow. The Product is lifted there.
- standalone System.scala lines after line 15: +6 in nexus-standalone and its laws file.

nexus-workflow-close and the activity files have no change. The comparison script (`irsame.py`, which blanks positions only) and the raw diff (`ir-positions.diff`) are in this directory. I compared lifts at lane base 6f3aa06ce and at HEAD; the batch check compares against the batch baseline. The Cases should be byte-identical: no Case-relevant IR field changed.

### For later tasks
- **fn-136.5 (retire predicates):** the Nexus predicates that are now role tests are `terminalPhase`, `running` and `waiting` (workflow), `terminal` and `live` (standalone), and `productTerminal` (product). Two things block a plain deletion:
  - `TrustingCaller` and `NexusCaller.end` read `NexusSystem.states.terminalPhase`.
  - The standalone `terminal` feeds Closable's `terminal` field (until fn-137), and the claim `terminalIsFinal` names `productTerminal`.
- **The integrator, after fn-136.2:** point `NexusRoleRefinements` at the framework closedness check, and consider putting the activity and Nexus tests in one place.
- A role-carrying case has the type `Phase & Role`. No Nexus inference site needed an ascription.

### MILESTONES
No edit; per the task, the MILESTONES update moves to task .5.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e7ca0cbcf0, f4cbbc0457
- Tests: baseline: green via lane base 6f3aa06ce (integrator state; framework tests not re-run before edit, Models lift at base taken as the identity reference), mise exec -- scala-cli test --suppress-outdated-dependency-warning --server=false model/project.scala model/umpire model/temporal (suite_rc=0, 45 passed; includes NexusRoleRefinements x2 and IrFiles), negative control: NexusRoleRefinements with in[Held] in place of in[Closed] fails both tests, naming the state and its image (rc=1); reverted, Models lift into scratch dirs at base 6f3aa06ce and at HEAD: irsame.py (positions blanked) SAME-BUT-POSITIONS over 10 files; positive control base-vs-base SAME, negative control (swapped case pair) DIFFERENT, scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check: clean, make lint-model-models: rc=0 (the usual NoSuchFieldException noise, as in fn-136.4 logs), make lint-model-syntax: rc=0, flowctl validate --spec fn-136-phase-roles-on-lifecycle-enums --json: valid, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: no model gate per task (worker-notes.md), GATE_SKIPPED:lint-model-irgen,lint-model-irgen-lifts,lint-model-check:untouched - no irgen/check/lift-fixture change in this task
- PRs: