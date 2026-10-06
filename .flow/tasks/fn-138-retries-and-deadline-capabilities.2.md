---
satisfies: [R2]
---
# fn-138-retries-and-deadline-capabilities.2 Deadline capability: kit Properties, per-declaration names and lifter refusals

## Description
Framework/kit work only: the Deadline capability, its Properties and refusals, proven on irgen fixtures, following task 1's shape. Split from task 1 so each capability's owner question and refusals are proven on their own; it follows task 1 because both edit the same kit and lifter files.

**Size:** M
**Files:** `model/temporal/capabilities/Deadline.scala` (new), `model/temporal/capabilities/Capabilities.scala`, `model/irgen/Capabilities.scala`, `model/irgen/testdata/lifts/Capabilities.scala` + `lifts/CapabilityRejects.scala` + `lifts/expected/*`, `model/irgen/test/Fixtures.test.scala`, `tools/umpire/ir/framework_test.go`
**Touches:** [model/temporal/capabilities/**, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala, tools/umpire/ir/framework_test.go, .flow/specs/fn-138-retries-and-deadline-capabilities.md]
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- **Ask the owner first** (record in the spec's Decision Context; update R2 if it changes): after fn-128.3 the activity's start-to-close timeout retries while attempts remain, so its firing lands in `Waiting`, not `TimedOut`; fn-129.1's heartbeat deadline (later in the same deferred batch) will do the same. Proposed: Deadline binds an optional retryable flag; a retryable Deadline's Property reads "a firing with retries remaining lands where Retries' retryable failure lands, otherwise in `TimedOut` recording its type", reading Retries' retries-remaining field when both are declared (fn-134's multi-capability rule brings it only where both are), and a non-retryable one keeps "lands in `TimedOut`".
- Fields: the deadline input (`ClassRef`), the covered role (a role type, carried as a type parameter with its `TypeTest` witness, not as a value), a set-in-this-state predicate (`S => Boolean`, e.g. `_.scheduleToClose == Timeout.expires`; after fn-128.1 the activity's schedule-to-start predicate also requires `dispatch = now`), and the timeout type recorded on firing (the fact it records). Phase from `Phased` (fn-137.1).
- Properties, in forms Check supports (a transition Property with a `when` is `unsupported`; see task 1):
  - Window: a transition Property with no `when`, keyed on the recorded timeout-type fact. Its shape is: if the step records this Deadline's timeout type, then the state before it was set and in the covered role. Keying on the fact is sound only because duplicate timeout types are refused, so say so in the Scaladoc.
  - Landing: a `when <deadline> holds` over the state after the step. The phase is `TimedOut` and the fact is recorded, with the retryable variant above if the owner accepts it.
- Names: one machine declares three Deadlines, and fn-134 names generated Queries `<machine>.<property>`, so three Deadlines would collide. Make the generated Property and Query name include the declaring `val` (e.g. `<machine>.<val>.<property>`, or the property name suffixed by the val) and record the chosen form in the spec; a collision that still occurs is refused naming both declarations. Check this against fn-134.2's naming before writing it.
- Refusals (lifter, reusing fn-137's missing-role check): covered role no phase case has (names the role); no `TimedOut` case; non-`Phased` machine; two Deadlines on one machine recording the same timeout type, naming both declarations and positions (carry over fn-134's duplicate-capability check: two Deadlines of different types are allowed, so the duplicate rule keys on kind plus timeout type).
- Add `Deadline(` to `TestFrameworkNamesNoTemporal`. Keep the name distinct from fn-133.4's realization `deadlines(…)` binding and the feature's `deadline` section: say so in the companion's Scaladoc.

### Investigation targets
**Required:**
- task 1's `model/temporal/capabilities/Retries.scala` (shape to mirror)
- `model/temporal/features/activity/standalone/system/System.scala:176-177, 215-231, 278-290` (timeOut effect, timer rules, `*Fires` Properties)
- `model/temporal/features/nexus/workflow/system/System.scala:209-247`
- `model/irgen/Capabilities.scala` (duplicate-capability check, ~316-353 before fn-134 moves it)
**Optional:**
- `.flow/tasks/fn-133-lean-typed-realizations.4.md` (`deadlines(…)` naming)

### Key context
- Relies on: task 1, fn-134.2 (shape, duplicate-capability refusal), fn-136.1 (roles), fn-137.1/.6 (`Phased`, witnesses, shared missing-role check), fn-128.1 (dispatch field in the schedule-to-start window), fn-128.3 (retryable start-to-close).
- The Deadline's covered role is a framework role or a model role extending one (fn-136 R1); `Live` for schedule-to-close includes `Suspended`, matching today's `states.live`.
## Acceptance
- [ ] The owner's answer on retryable deadlines is recorded in the spec, and R2 matches it.
- [ ] A fixture machine with two Deadlines of different timeout types lifts, each bringing its own Properties with `origin`, bounded from `queries`.
- [ ] Refusal fixtures: missing covered role, missing `TimedOut`, non-`Phased` machine, duplicate timeout type naming both declarations.
- [ ] `scala-cli test model/irgen` and `scala-cli test model/temporal` pass with no unchecked warning; `TestFrameworkNamesNoTemporal` lists `Deadline(`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
