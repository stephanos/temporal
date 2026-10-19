# Accepted self-loops in the attribute surface (model)

## Conversation Evidence

> user (2026-10-19), after fn-223.1 moved thirteen trait-form fixtures checker-only: "explain differences and if/how to resolve them"
> assistant: self-loops matter most: an attribute handler is enabled only when it changes state, so an enabled action that leaves the state unchanged (heartbeats, idempotent requests answered OK) cannot be written except as a `reject!` self-loop; a fair self-loop changes progress verdicts. The TLA+ emitter already handles authored self-loops, so only the surface and the lifter's admission are missing. Recommended a small spec before fn-218; choice, typed outcomes/facts and computed starts stay triggered gaps.
> Filed under the owner's standing instruction (2026-10-19) to follow the root agent's own recommendations instead of asking.

## Goal & Context

`#[machine]` handlers are `fn actor(&mut self, action: Actor)`; a handler is enabled iff the state changed (fn-210.18), except that `reject!(..)` turns an unchanged handler into a `Rejected(..)` self-loop (fn-221). An *accepted* self-loop, enabled and unchanged, has no spelling. fn-223 recorded the consequence: TLC no longer pins authored self-loops anywhere in the shared corpus (`edge_checks`, `vacuity`, `shared_starts`, `progress_self_loop`, `progress_target_barrier`, `progress_excluded_prefix`, `progress_scope_exit` and one branch of `progress_action_choices` are checker-only for that reason, some among other reasons).

## Architecture & Data Models

- One statement-level macro beside `reject!`, provisionally `stay!()` (final name chosen in the task, consistent with `reject!`'s style and recorded in DESIGN.md): in a handler arm it marks the step enabled even if the state is unchanged. If the arm also changes state, the step is an ordinary changed step (the marker is redundant, not an error). In a module with `reject!` the self-loop's outcome is `Accepted`; without `reject!` it is unit.
- Precedence: `reject!` wins over the marker in the same execution (first rejection returns, as today).
- The checker sees an ordinary edge `state -> state`; no engine change is expected beyond what the macro expansion feeds `Steps`.
- `umpire-lift` admits the marker and emits the self-loop through the existing authored-self-loop observer encoding (fn-212); weak fairness stays only on authored fair actions.

## API Contracts

- `stay!()` (provisional) usable only inside `#[on(..)]` handler bodies; elsewhere a compile error, pinned by trybuild.
- No change to `Machine`, `Steps`, `Outcome` or existing goldens.

## Edge Cases & Constraints

- Existing Models without the marker keep byte-identical goldens (the activity Model, both Kettles, all fixtures).
- A marker in a `#[effect]` or claim method is refused by the read-only rule already enforced by `#[machine]`.
- R30 budgets hold; no new crates.

## Acceptance Criteria

- **R1:** The marker makes an unchanged handler arm an enabled self-loop (unit or `Accepted` outcome); trybuild pins misuse; all existing goldens are byte-identical.
- **R2:** At least `progress_self_loop` (with the other moved progress fixtures that differ from attribute fixtures only by self-loops) and `edge_checks`/`vacuity` where their remaining reasons allow, are rewritten as attribute shared fixtures with report/rows/states/paths/tlc goldens byte-identical up to the spellings fn-223 listed, and agree with TLC under `UMPIRE_BACKENDS=require`. The fn-223 disposition table in DESIGN.md is updated (rows that move back, rows that stay checker-only with their remaining gap). Errors: a fixture whose rewrite changes any verdict, count or path stays checker-only with the difference recorded.

## Boundaries

- No nondeterministic choice within one action, no typed outcomes or facts, no computed `#[starts]` block: each stays a triggered gap, built when a Temporal Model needs it.
- Does not reintroduce lifting of the hand-written trait form.

## Decision Context

2026-10-19, root agent under the owner's standing instruction: filed after fn-223 with placement after fn-213 and before fn-218, because fn-218's cover tests will want idempotent actions and the emitter support already exists.
