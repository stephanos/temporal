# Retries and Deadline capabilities

> HTML render lens (local): open `.flow/artifacts/fn-138-retries-and-deadline-capabilities/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 2, part 1): "actually think a little harder; could we provide more reusability evne?"
> user (turn 2, part 2): "there are a few patterns across phases; like retry, failure, timeout etc."
> user (turn 2, part 3): "is there a way we could have several pre-defined reusable states like \"is terminal\""
> user (turn 3): "could we do sth more composable instead? we don't want to define all these types head. we want each of these to be a trait and then connect the actual enum with them"
> user (turn 6): "no problem; I accept the proposal. write a flow next spec"
> user (turn 7, selected): "Roles + caps + Retries"
> user (turn 8, selected): "split-as-proposed"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [paraphrase], 20% [user], 30% [inferred] -->

The owner pointed at patterns that repeat across phases: "like retry, failure, timeout etc." Phase roles (sibling spec fn-136 ("Phase roles on lifecycle enums")) cover what a phase is. Retry and timeout are about how phases change: a retryable failure backs an attempt off and a later timer retries it, and a deadline covers a window of phases and closes the entity as timed out when it fires. The activity system and the Nexus workflow system each write these transitions, and the phase windows each deadline covers, by hand.

This spec adds two capabilities that state those patterns once, in role terms, so a machine declares them and receives their Properties. The Properties are checked against the machine's own steps. Declaring a capability doesn't write the transitions for the machine.

## Architecture & Data Models
<!-- scope: technical -->

**Retries.** A capability that reads the phase from the machine's `Phased` declaration (fn-137) and binds its attempt count, the attempt bound and the worker's failure answer. Its Properties are stated in roles, without the `Retrying` role: a retryable failure of a `Held` attempt lands in a `Waiting` phase; a failure that is not retryable lands in a `Failed` phase; the attempt count never exceeds the bound. Both models saturate the attempt count at the bound and keep retrying, and after fn-128.1 the activity's back-off is a field rather than a phase, so the Properties name neither a bound-reached failure nor a back-off phase.

**Deadline.** [paraphrase] A capability that reads the phase from the machine's `Phased` declaration (fn-137) and binds the role the deadline covers, whether the deadline is set in a state, and the timeout type it records. Its Properties: the deadline fires only while it is set and the phase has the covered role; a firing lands in a `TimedOut` phase and records its timeout type. [paraphrase] A machine declares one Deadline per timer. The activity's three map to roles as schedule-to-close covering `Live`, schedule-to-start covering `Waiting` and start-to-close covering `Held`.

**Shape.** [inferred] Both capabilities follow fn-134's shape (companion-defined Properties, declared in a machine's `capabilities` section, bounded in its `queries` section) and read roles through the type witnesses of fn-137 ("Capabilities read phase roles").

## API Contracts
<!-- scope: technical -->

- [paraphrase] **Retries**: reads the phase from `Phased`; binds the attempt-count projection, the attempt bound and the failure input with its retryable classification.
- [paraphrase] **Deadline**: reads the phase from `Phased`; binds the covered role, a set-in-this-state predicate and the timeout type recorded on firing.

## Edge Cases & Constraints
<!-- scope: technical -->

- [inferred] A Retries declared for a phase with no `Held`, `Waiting` or `Failed` case, or a Deadline for a phase with no `TimedOut` case or no case with its covered role, is refused instead of generating a vacuous Property.
- [inferred] A machine with more than one deadline of the same timeout type is refused, since the recorded type would not tell them apart.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A Retries capability exists whose Properties state that a retryable failure of a `Held` attempt lands in a `Waiting` phase, a non-retryable failure lands in a `Failed` phase, and the attempt count never exceeds the bound. The activity system and the Nexus workflow system declare it and its Properties hold there. Errors: a Retries declared for a phase lacking a `Held`, `Waiting` or `Failed` case is refused by the IR generator, naming the missing role.
- **R2:** [paraphrase] A Deadline capability exists whose Properties state that the deadline fires only while set and in a phase with its covered role, and that a firing lands in a `TimedOut` phase recording its timeout type. The activity system and the Nexus workflow system declare one per timer, and their Properties hold there. Errors: a Deadline whose covered role no phase case has, or whose phase has no `TimedOut` case, is refused naming the role; two Deadlines of one machine with the same timeout type are refused naming both.
- **R3:** [inferred] The hand-written phase windows these machines' timer rules read (live, waiting, held) are role tests, and declaring Retries and Deadline changes no existing Query's answer, receipt or Definition ID. Errors: any other difference stops the regeneration.

## Early proof point

Task fn-138-retries-and-deadline-capabilities.1 validates the core approach: a capability that reads roles but whose Properties read its own fields lifts on a machine without Pollable, and its missing-role refusals fire. If it fails, revisit how fn-137's owned-role rule treats a capability that reads a role another capability owns before continuing with .2+.

## Quick commands

```bash
mise exec -- scala-cli test model/irgen
mise exec -- scala-cli test model/temporal
```

## Open Questions

Found in planning (2026-10-06) against the upstream task shapes. Tasks 1 and 2 ask the owner before writing the Properties.

- **Exhausted attempts (R1).** fn-128.3 makes the activity's retryable failure, and its retryable start-to-close timeout, fail once `retriesRemaining` is false. The Decision Context says both models keep retrying at the bound. Proposed: Retries binds an optional retries-remaining predicate, and a retryable failure lands in `Waiting` only while retries remain.
- **Source role (R1).** The Nexus workflow's retryable handler error, and `network.fault`, fire from `scheduled`, a `Waiting` phase, not a `Held` one. Proposed: the source is any `Live` phase, or Retries states no source role.
- **Retryable deadlines (R2).** After fn-128.3 the activity's start-to-close firing lands in `Waiting` while attempts remain, and fn-129.1's heartbeat deadline will do the same. Proposed: an optional retryable flag on Deadline, whose Property defers to Retries while retries remain.
- **Request phases (R1).** In the activity a retryable failure from `pauseRequested` lands in `paused` (`Suspended`) and from `cancelRequested` in `canceled` (`Closed`). fn-136 makes both request phases `Held`, so no choice of source role makes "lands in `Waiting`" hold. Proposed: the retryable Property promises only "a retryable failure with retries remaining does not land in `Failed`".
- **Vacuous bound (R1).** "The attempt count never exceeds the bound" can't fail: the activity's `attempts: UpTo[2]` and the Nexus workflow's `Finite.upTo(attemptBound)` already guarantee it by type. Proposed: bound the count against fn-128.3's `maxAttempts` where it is finite, or drop the Property and amend R1.
- **Expressible forms (R1, R2).** Check answers a transition Property that has a `when` as `unsupported`, and `Step` doesn't carry the action. So a Property must be either a `when … holds` over the state after the step, or a transition Property with no `when`. The Deadline window is the second kind, keyed on the recorded timeout-type fact. That form depends on refusing duplicate timeout types. A Retries condition on the state before the failure has no supported form.
- **Generated names (R2).** Three Deadlines on one machine would collide under `<machine>.<property>`. Task 2 settles a per-declaration name.
- **R3 overlap with fn-136.5.** fn-136.5 already retires the `live`, `waiting` and `held` predicates. What is left is the Nexus start-to-close rule's inline `phase == started`. R3's "no Definition ID change" covers existing Queries only: fn-138's own delta adds Properties and Queries.

## Boundaries
<!-- scope: business -->

- [paraphrase] Retries and Deadline check the machine's steps. They don't generate the steps.
- [inferred] Backoff timing, retry policy fields (initial interval, coefficient, maximum interval) and non-retryable error types beyond the retryable classification are not modeled.

## Decision Context
<!-- scope: both -->

Retries reads `Held`, `Waiting` and `Failed` rather than `Retrying` (owner's choice during planning, 2026-10-06): a retryable failure at the bound saturates and keeps retrying in both models, and fn-128.1 turns the activity's back-off into a field, so a `Retrying` phase is not a portable target. The `Retrying` role stays in fn-136 for phases that are backing off, such as the Nexus workflow's.


[paraphrase] Retry and timeout were first raised as candidate phase classifications, but they describe transitions, not phases: a phase is `Retrying` or `TimedOut`, and the retry or the timeout is the step into it. So they are capabilities that read roles, beside Close and Pause, and not more roles.

Maintainability (plan review): duplication - task .3 keeps the hand-written timeout Properties (`scheduleToStartFires`, `scheduleToCloseFires`, `startToCloseFires` and the Nexus equivalents) beside Deadline Properties that state the same landing claim, because R3 forbids changing existing Queries; retiring them is an owner decision after this spec; structure - none identified

[paraphrase] This spec is one of three split from one conversation. It depends on fn-137 ("Capabilities read phase roles"), which depends on fn-136 ("Phase roles on lifecycle enums"). The approved activity-batch conductor gate starts implementation only after fn-128.5 is done: the precision source and realization work must be available before Retries reads it. Flow cannot express cross-spec task dependencies, so this source gate supplements its metadata rather than waiting for fn-128 to close. fn-128.6 and fn-129.5 share regeneration, review and live-run evidence at the batch boundary; no activity spec closes prematurely.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A Retries capability exists whose Properties state that a retryable failure of a `Held` attempt lands in a `Waiting` phase, a non-retryable failure lands in a `Failed` phase, and the attempt count never exceeds the bound. The activity system and the Nexus workflow system declare it and its Properties hold there. Errors: a Retries declared for a phase lacking a `Held`, `Waiting` or `Failed` case is refused by the IR generator, naming the missing role. | fn-138-retries-and-deadline-capabilities.1, fn-138-retries-and-deadline-capabilities.3 | — |
| R2 | [paraphrase] A Deadline capability exists whose Properties state that the deadline fires only while set and in a phase with its covered role, and that a firing lands in a `TimedOut` phase recording its timeout type. The activity system and the Nexus workflow system declare one per timer, and their Properties hold there. Errors: a Deadline whose covered role no phase case has, or whose phase has no `TimedOut` case, is refused naming the role; two Deadlines of one machine with the same timeout type are refused naming both. | fn-138-retries-and-deadline-capabilities.2, fn-138-retries-and-deadline-capabilities.3 | — |
| R3 | [inferred] The hand-written phase windows these machines' timer rules read (live, waiting, held) are role tests, and declaring Retries and Deadline changes no existing Query's answer, receipt or Definition ID. Errors: any other difference stops the regeneration. | fn-138-retries-and-deadline-capabilities.3 | — |

