# Samples follow the machine sections and are linted in the framework check

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-16): "why is there no `object properties` or sth?"
> user: "update the kettle example separately. and let's add a lint check for the samples to run routinely when we validate/check the framework itself."

## Goal & Context

A machine object has named sections in a fixed order. Today's order is `states`, `refinement`,
`effects`, `monitors`, `rules`, `properties` (with `safety`/`liveness` groups inside),
`capabilities`, `queries`; the structure and order lints enforce it for feature sources. The owner
replaced the claim grouping (2026-10-16): claims go in sections named for their kind,
`invariants`, `postconditions`, `monitors`, `progress`, design-by-contract words a developer
already knows, with the guards of the transitions as the preconditions. The order becomes `states`,
`refinement`, `effects`, `rules`, `invariants`, `postconditions`, `monitors`, `progress`,
`capabilities`, `queries`; fn-190 then renames `rules` to `transitions`, and the lint follows it
there. The six framework samples predate any of it: they keep claims,
scenarios, limits and queries as loose vals in the feature object and collect queries in a
`Vector`. No lint runs over them, and the Go model check of their answers is held in the
framework-only loop, so nothing says so.

This spec applies the convention to the samples and makes the framework check lint them, so the
samples stay the reference they are meant to be and a drift is caught by the routine check rather
than by a reader. Kettle is done first, on its own, as the model for the other five.

## Architecture & Data Models

- Each sample's Machine gains `invariants`, `postconditions` (and `progress` where it has a
  claim) and `queries`. Until fn-190 lands, the members keep today's vocabulary (`property.expect`,
  `property.across.expect`, `leadsTo(name)(…)`); fn-190 removes the words, not the sections.
  Scenarios, limits and queries are named vals in `queries`, so a query is named after its val and
  the `Vector` goes. Values of the state that are not claims, such as Kettle's `coldest`, stay in
  the feature object.
- The structure and order lints learn the new section set and drop the `safety`/`liveness`
  groups; SEMANTICS.md §8 and §11 say "in `invariants`/`postconditions`" and "in `progress`".
  The Temporal features move at the sweep.
- Kettle gains a progress claim, `boils` (`leadsTo("boils")(heatingOn, beeping, within = 81)` in
  today's vocabulary), checked on a derived `HeatingOnly = Derived(Model.restrict(heater.heat, heater.failover))`
  until fn-190 moves the restriction onto the scenario (`scenario.only(...)`); a comment says why
  the user's actions are restricted rather than assumed away; the intended answer (Verified) goes
  in the inventory.
- IR discovery already reads a `queries` section reflectively; the discovery roots in the reify
  sample test move from `samples.<Name>$.queries` to `samples.<Name>$.Model$.queries` (or the
  sample's machine name).
- A **samples model-check stage** in the framework check: the two `GoHeld` cases in
  `check/Samples.test.scala` already lift the samples, build the Go checker and compare every
  answer with the inventory; the hold reason is only "runs the Go toolchain". `checkFramework`
  runs that path as a stage after the lint stage, failing on any answer that differs from the
  inventory, and the `GoHeld` wrapper is removed from those two cases.
- A **samples lint stage** in the framework check (`checkFramework`, fn-188.4): compile the
  samples against the fresh framework jar with the plugin, index their TASTy, and run the section
  rules of the structure and order lints over them with the feature-folder rules exempt (a sample
  is one file, not a feature folder). A finding fails the check with the lint's own text and
  position. The stage runs on every `checkFramework`, between the plugin build and the test modules.

## API Contracts

The section layout a sample follows:

```scala
object Model extends Machine[...]:
  val init = ...
  def end(...) = ...
  object effects: ...
  object rules extends Rules: ...
  object invariants:
    val someInvariant = property.expect: ...            // fn-190: no word
  object postconditions:
    val someStep = property.across.expect: ...          // fn-190: no word; after(action): stays
  object progress:
    val someProgress = Model.leadsTo("someProgress")(…) // fn-190: leadsTo(from=, to=, within=); environment on the scenario
  object queries:
    val someScenario = scenario.starts(init).free
    val bounds = Limits(...)
    val someQuery = query verify invariants.someInvariant in someScenario limits bounds
```

Check output lines for the model-check stage: `PASS samples check: <n> queries, <m> progress claims`
or `FAIL samples check: <sample>.<query>: expected <answer>, got <answer>` per difference.

Check output lines, exact shapes: `PASS samples lint: <n> files` or
`FAIL samples lint: <file>:<line>: <lint text>` per finding.

## Edge Cases & Constraints

- A sample whose claims span two machines (Voting, TwoPhaseCommit, TwoLayeredCache) puts each
  Property in the machine it is about and each Query in the machine of its Scenario.
- `NativeInputs` already has an `object properties`; it gains `queries` if it has any.
- The reify sample oracle under `export/testdata/reify/samples` changes function names from
  `model.property.x` to the section-qualified names; it is regenerated once the export suite is
  re-anchored (fn-164.3), not here. The export suite is red for that reason already.
- The lint stage exempts the feature-folder rules only; every other rule a feature source gets, a
  sample gets.

## Acceptance Criteria

- **R1:** Kettle's claims, scenarios, limits and queries are in `invariants`, `postconditions`
  and `queries` sections of its Machine, each query named after its val, no `Vector`; the native
  sample tests pass. Errors: none beyond compile errors.
- **R2:** The other five samples and `NativeInputs` follow the same layout, with `progress` where
  a sample has a claim; Kettle gains `HeatingOnly` and the `boils` progress claim with its intended
  answer in the inventory; the structure and order lints and SEMANTICS.md carry the new section
  set; the native sample tests pass; the reify sample test's discovery roots name the new sections.
  Errors: none beyond compile errors.
- **R3:** `checkFramework` runs a samples lint stage that compiles the samples with the fresh
  framework jar and plugin, indexes their TASTy and applies the structure and order section rules
  with feature-folder rules exempt; it prints the PASS or FAIL lines in API Contracts and fails the
  check on any finding. A focused test proves a sample with a misplaced or misnamed section fails
  with the lint's text and that the six samples pass. Errors: a compile failure of the samples
  fails the stage with the compiler output; zero indexed files fails the stage.

- **R4:** `checkFramework` runs the samples model check as a stage: lift the six samples, build
  and run the Go checker, compare every query answer and progress verdict with the inventory;
  print the PASS or FAIL lines in API Contracts; fail on any difference; the two `GoHeld` wrappers
  in `check/Samples.test.scala` are removed so the same cases run in the framework loop. Errors: a
  Go build or run failure fails the stage with its output; a sample with no queries fails the
  stage naming it.

## Boundaries

- No change to the lint rules beyond the exemption switch and the new section set.
- No change to the Go checker; the stage runs the existing generated checker.
- No regeneration of export fixtures or the reify sample oracle.
- The Temporal kit stays disabled.

## Decision Context

Grouping into sections was already the convention and the lints already exist; the samples were
simply never brought under them. Applying the lint in the framework check, rather than only in the
Model gate, is what keeps the samples honest in the framework-only phase, where the Model gate does
not run. Kettle goes first because its rewrite was in hand and is the smallest.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | applied on capture (Kettle) | — |
| R2 | .1 | — |
| R3 | .2 | — |
| R4 | .3 | — |
