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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
