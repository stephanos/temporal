# Reusable deadline countdowns for Models (model)

## Conversation Evidence

> user (2026-10-25): "can this be written more clearly?" (the activity's `#[on(Deadline)]` handler)
> assistant: a rewrite using Option equality, `time_out`/`time_out_attempt` effects and an `armed()` helper; all 33 activity tests passed, including TLC, then reverted.
> user (2026-10-25): "should this be split into separate invariants with good names?" (`state_bounds`)
> user (2026-10-25): "should we have a re-usable component/facet (do wwe have this design alrady?) for the various deadline concerns? we'll like need those in other places. a great abstraction would go a long way."
> assistant: no Rust design exists; the archived Scala `temporal/capabilities/Deadline.scala` held laws only, and fn-122 deferred `Deadlined`. Proposed a generic countdown in `umpire-dsl` plus Temporal deadline semantics in `model/temporal`, starting with a spike.
> user (2026-10-25): "yes write new flow spec to capture this abstraction"

## Goal & Context

<!-- scope: business -->

Every deadline in the activity Model is written out by hand, once per budget. Each of its four budgets (`schedule_to_start_remaining`, `start_to_close_remaining`, `schedule_to_close_remaining`, `heartbeat_remaining`) repeats the same five pieces of code:

- arm it from the configured `Option<Ticks>`,
- count it down by one per Tick,
- treat zero as expired,
- the invariant that an armed budget lies in `1..=configured`,
- the law that a Tick lowers an armed budget by exactly one.

Together with the `is_some() && unwrap_or(Remaining { ticks: 0 }).ticks` pattern, this makes up most of `decrement`, `state_bounds`, `countdowns_decrement`, `tick_priority` and the expiry chain in `deadline` (`model/temporal/feature/activity/model.rs:244-340`, `:342-420`, `:481-600`).

Deadlines will recur. fn-246.13 authors an independent protocol activity Model with its own timers and budgets. Workflow run/execution timeouts, Nexus schedule-to-close and user timers are the next likely Models. The goal is one named, reusable deadline abstraction, so a Model states *which* budgets exist and *how they interact*, and never re-derives the countdown mechanics or their laws.

Following the owner's naming rule, the abstraction is named now with the activity as its only user.

## Architecture & Data Models

<!-- scope: technical -->

Two layers, split by what is generic and what is Temporal's.

**1. Countdowns: generic, in `umpire-dsl`.** A `countdown` module beside `closable` (`umpire/author/dsl/src/`) with no Temporal vocabulary, as the no-Temporal-in-`model/umpire` rule requires. It holds pure value functions over an optional countdown and laws in the `closable` style. Pure functions, not `&mut` helpers, so they map directly to TLA+ operators (and Quint `pure def`s) and stay plain Rust:

```rust
self.heartbeat_remaining = countdown::arm(world.heartbeat_timeout);
self.start_to_close_remaining = countdown::tick(self.start_to_close_remaining);
if countdown::expired(self.schedule_to_close_remaining) { .. }

#[invariant]
fn countdowns_within_budget(&self, world: &World) {
    countdown::within(self.heartbeat_remaining, world.heartbeat_timeout);
    ..
}
```

The spike (task .2) settles the exact surface and the representation:
- either library-owned configured/remaining types,
- or functions generic over a small trait that each Model's `Finite` wrappers implement.

The range (`0..=3` today) stays Model-selected.

**2. Temporal deadline semantics: in `model/temporal`.** This layer covers:
- the named timeouts (ScheduleToStart, StartToClose, ScheduleToClose, Heartbeat),
- which phase arms which budget,
- expiry priority (global, then attempt, then heartbeat, then schedule-to-start),
- re-arming on retry and resetting on heartbeat,
- the API-aligned pending-cancellation failure/expiry outcomes from fn-246.13,
- the global budget never being replenished.

It is a named pattern under `model/temporal/capabilities/system/` (fn-163's system/product split; new directory in the Rust tree). If the lifter cannot read shared Temporal source (see constraints), this layer is instead a documented authoring pattern with named per-Model effects such as `time_out` and `time_out_attempt`, plus shared laws only where `umpire-dsl` can hold them generically.

**Lifter.** Today the front end reads `umpire-dsl`'s `closable.rs` as a library (`umpire/front/src/library.rs:4`) and admits library calls only as claim statements (`umpire/front/src/body/claim.rs:165`). A `countdown` library additionally needs:
- calls in handler/effect assignments and conditions,
- calls returning a value,
- generic optional-countdown arguments beyond today's rule that an optional value is an `Option` of a source-local `Finite` wrapper (`umpire/README.md#bounded-optional-values`).

All checkers (compiled native, interpreter, TLC emission) lower the library from its source, with no recognition by name, as `closable` does today.

## API Contracts

<!-- scope: technical -->

Provisional; task .2 fixes it.

- `umpire_dsl::countdown::{arm, tick, expired, armed}`: pure functions over `Option<countdown>`.
- `umpire_dsl::countdown::{within, descends}`: laws, callable from `#[invariant]`/`#[postcondition]` like `closable::` laws.
- `model/temporal/capabilities/system/deadline.rs` (or the authoring pattern): the Temporal deadline semantics above.
- `#[on]`, `#[effect]`, `check!`, `reject!` and the Model surface are otherwise unchanged.

## Edge Cases & Constraints

<!-- scope: technical -->

- **Zero vs absent.** "Absent" (`None`) and "armed at zero" stay distinct in values, decoding and replay (`umpire/README.md#bounded-optional-values`). `tick` never decrements below zero or arms an absent budget.
- **Expiry in the same step.** The activity clears every budget on a terminal transition in the step where one reaches zero, so `within` asserts `1..=configured` for armed budgets. A Model that keeps an expired budget armed needs a different law; the library documents which one it states.
- **Independence for refinement.** fn-246 checks a feature Model against an independently authored protocol Model (fn-246.13). Sharing the generic countdown mechanics between them is acceptable. Sharing the Temporal deadline *semantics* would let one misreading pass both sides of the refinement check. Whether the protocol Model may use the Temporal layer is an owner decision (see Parked unknowns).
- **Shared Temporal source.** The lifter reads the machine module plus `umpire-dsl` libraries only. Lifting a `model/temporal` helper needs a registered project library, which is a front-end change. The spike decides whether that is worth it or the authoring-pattern form suffices.
- **Refactor equivalence.** Tasks .1 and .7 preserve the committed fn-246.15/fn-246.13 corrected baseline's states, edges, facts, outcomes and claim coverage. Task .9 may change World representation while retaining valid-World behavior and native forged-value validation. Task .10 captures cooperative twin baselines before deleting the flag, then applies the separately approved compatibility-control decision. Its final selected Worlds retain those twins' corrected behavior after removing the obsolete World field from comparison. Default atomic-world goldens may legitimately change to cooperative behavior and must regenerate against those captured twins. Source-mutation anchors in `proof_checks.rs` (e.g. `global_mutants`) are reanchored, not dropped.
- **Concurrent owner.** fn-246.15 delivered the timeout-retry correction; fn-246.13 owns the approved pending-cancellation outcome correction and independent protocol proof. Activity edits start from both committed closures, after the conductor checks same-file ownership. The spike and generic library may proceed independently.

## Acceptance Criteria

<!-- scope: both -->

- **R1:** A spike rewrites the activity's deadline code against the proposed `countdown` surface (in a worktree, not merged) and records the resulting admission refusals. It also writes a short verdict in `model/design/deadline-countdowns.md` choosing the representation and the Temporal-layer form. Errors: every refusal is listed with its source location and the front-end owner that must change.
- **R2:** `umpire_dsl::countdown` exists with pure arm/tick/expired/armed functions and within/descends laws, documented with what each does not promise. It is admitted by the shared front end and lowered from its own source by every checker. Errors: fixture-source mutants replace countdown calls with inline broken bodies for no decrement, absent-countdown corruption, zero-boundary corruption and an off-by-one `within` check. The interpreted mutant differs from the compiled good transition/claim behavior, TLC refutes a named law, and its witness replays through the interpreted mutant. No runtime compiled-library mutation or library-source override hook is required. Unsupported uses (aliases, glob imports, nesting) refuse at their span.
- **R3:** The Temporal deadline semantics exist once under the form R1 chose, named, and documented with their authority (pinned API and docs references already used by fn-237/fn-246.15).
- **R4:** The activity Model uses R2 and R3:
  - no `unwrap_or(Remaining { ticks: 0 })` remains in its deadline code;
  - `state_bounds` is split into named invariants (policy in range, attempt within policy, countdowns within budget);
  - the expiry chain reads as one priority list;
  - task .9 conditionally adopts a bounded `Attempts` wrapper only if World-selected assumption comparisons and the `#[domain] fn attempts(world)` nominal payload read compile and are admitted. It preserves valid-World behavior and native forged-value validation, or records non-adoption with the concrete refusal.

  Errors: native/TLC agreement, exact transitions and facts, the closure-law and deadline mutants, and `compatibility/checkpoint.rs` all still pass through .7 against the corrected baseline. Any .9 representation change preserves those valid-World semantics and native invalid-World controls. R6 governs the final compatibility-control form after .10. Reanchored source mutants still discriminate.
- **R5:** `make model-check` passes and the TLC-required `make model-validate` passes. Goldens are re-accepted after review. README and DESIGN describe the abstraction and how a new Model declares its countdowns.
- **R6:** The activity World has no cancellation-mode flag (`asynchronous_cancel` removed). Cancellation is always cooperative, matching the pinned request-cancel contract: immediate with no running attempt, pending while an attempt runs. The fn-237 R3 compatibility control is either re-expressed as a projection or retired, by owner decision recorded in fn-237's Decision Context after reviewing a concrete total projection with pending-edge accounting and a changed-old-edge control, or the documented limits that motivate retirement. Errors: the reintroduced atomic transition (Cancel in `Started` closing at once) is refuted by native and TLC; a changed old lifecycle edge still fails the control if it is kept.

## Boundaries

<!-- scope: business -->

- No new timeout semantics. fn-246.15 owns the timeout-retry correction and fn-246.13 owns the approved pending-cancellation failure/expiry correction; this spec preserves their corrected results.
- No workflow, Nexus or timer Models. They are future users, not scope.
- No new deadline realization or observation semantics. Tasks .9/.10 update exact World parameter/adaptor plumbing as required, and .10 updates cancellation-mode admission in coordination with fn-237.4. Worker evidence production remains with fn-237.4; the independent refinement semantics remain with fn-246.
- No real-time clock or backoff modelling; Ticks stay abstract.
- No capability registry or attribute that adds claims implicitly. Laws stay explicit delegating calls, as with `closable` (fn-221).

## Decision Context

<!-- scope: both -->

2026-10-25, naming: the generic library is `countdown`, not `budget`, because "budget" already names the checker's exploration/search budget in `model/umpire`. Rejected: `timer` (Temporal user Timers) and `deadline` (the activity's action/fact names; a point in time, not a count). The Temporal layer keeps Temporal's words: timeouts and deadline.

2026-10-25: filed from the conversation above. Rejected: a Scala-style capability object with closure parameters (`Deadline.scala`). Its laws took `S => Boolean` bindings the Rust lifter cannot lower, and it left the mechanics duplicated. Also rejected: automatic inheritance of laws from a trait, for the visibility reasons given for `closable`.

Current placement follows MILESTONES: .2–.5 may run as a disjoint lane. Activity edits in .1/.7/.9/.10 wait for committed fn-246.15 and fn-246.13 closures. Serialize the activity changes as .1 → .7 → .9 → .10, then .8 validates the final result. The unresolved compatibility-control decision still gates .10; this review does not authorize changing that control.

2026-10-25, owner: remove the cancellation-mode flag and always model cooperative cancellation (R6, task .10). The atomic World matched no live behaviour; it existed for fn-227's retained lifecycle comparison and the atomic live binding. Reopens fn-237 R3's control shape.

2026-10-25, owner question "where do we file this": the tested deadline-handler cleanup and the `state_bounds` split belong to task .1 and remain independent of the spike, before the abstraction migration. The original placement waited for fn-246.15; current activity gates additionally wait for fn-246.13's approved correction, as stated above. fn-252 (quick wins) is closed.

Maintainability (plan review): duplication - World constructor and claim rewrites reuse one reanchored call-site inventory; refactor, representation and cooperative-twin equivalence remain distinct. Structure - library-call expression dispatch shares argument/result typing; project-library registration stays generic and Temporal semantics stay in their owning tree.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A spike rewrites the activity's deadline code against the proposed `countdown` surface (in a worktree, not merged) and records the resulting admission refusals. It also writes a short verdict in `model/design/deadline-countdowns.md` choosing the representation and the Temporal-layer form. Errors: every refusal is listed with its source location and the front-end owner that must change. | fn-253-reusable-deadline-countdowns-for-models.2 | — |
| R2 | `umpire_dsl::countdown` exists with pure arm/tick/expired/armed functions and within/descends laws, documented with what each does not promise. It is admitted by the shared front end and lowered from its own source by every checker. Errors: fixture-source mutants replace countdown calls with inline broken bodies for no decrement, absent-countdown corruption, zero-boundary corruption and an off-by-one `within` check. The interpreted mutant differs from the compiled good transition/claim behavior, TLC refutes a named law, and its witness replays through the interpreted mutant. No runtime compiled-library mutation or library-source override hook is required. Unsupported uses (aliases, glob imports, nesting) refuse at their span. | fn-253-reusable-deadline-countdowns-for-models.3, fn-253-reusable-deadline-countdowns-for-models.4, fn-253-reusable-deadline-countdowns-for-models.5 | — |
| R3 | The Temporal deadline semantics exist once under the form R1 chose, named, and documented with their authority (pinned API and docs references already used by fn-237/fn-246.15). | fn-253-reusable-deadline-countdowns-for-models.6 | — |
| R4 | The activity Model uses R2 and R3: | fn-253-reusable-deadline-countdowns-for-models.1, fn-253-reusable-deadline-countdowns-for-models.7, fn-253-reusable-deadline-countdowns-for-models.9 | — |
| R5 | `make model-check` passes and the TLC-required `make model-validate` passes. Goldens are re-accepted after review. README and DESIGN describe the abstraction and how a new Model declares its countdowns. | fn-253-reusable-deadline-countdowns-for-models.8 | — |
| R6 | The activity World has no cancellation-mode flag (`asynchronous_cancel` removed). Cancellation is always cooperative, matching the pinned request-cancel contract: immediate with no running attempt, pending while an attempt runs. The fn-237 R3 compatibility control is either re-expressed as a projection or retired, by owner decision recorded in fn-237's Decision Context after reviewing a concrete total projection with pending-edge accounting and a changed-old-edge control, or the documented limits that motivate retirement. Errors: the reintroduced atomic transition (Cancel in `Started` closing at once) is refuted by native and TLC; a changed old lifecycle edge still fails the control if it is kept. | fn-253-reusable-deadline-countdowns-for-models.10 | — |

The final proof in .8 consumes .9's recorded outcome and .10's approved implementation. A refused .9 wrapper proposal is recorded as a scoped non-adoption, never as a successful schema migration. .10 remains pending until its compatibility-control decision is recorded.

## Parked unknowns

- May the independent protocol Model (fn-246.13) use the Temporal deadline layer, or only the generic countdown?
- Library-owned countdown types vs a trait over each Model's `Finite` wrappers. Owner, 2026-10-25, asked whether `Ticks` should be a shared Umpire concern. Proposed answer: the abstract tick and the configured/remaining pair are system-agnostic and belong with `countdown` in `umpire-dsl`; the bound stays Model-selected (it sizes the state space), ideally one const parameter shared by both types. .2 decides.
- Whether a registered project library (shared `model/temporal` source read by the front end) is worth its cost, or the authoring-pattern form suffices.
