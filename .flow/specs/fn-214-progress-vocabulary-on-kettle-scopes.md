# Progress vocabulary on Kettle: scopes, bounds, fairness and verdicts (model)

## Conversation Evidence

> user (2026-10-19): "do we have the Temporal logic API well flhes out?" — answered: no; Rust has `whenever/eventually` and `without` only, and `#[internal]` implies weak fairness where Scala and TLA+ make fairness explicit.
> user (2026-10-19), after the design walkthrough with Kettle examples: "yes, write new spec".

## Goal & Context

Umpire in Rust checks one progress form today: `whenever(p).eventually(q)` inside a `without` scope, answering verified, refuted or unknown. Scala's progress vocabulary (archived `SEMANTICS.md` §11, `REFERENCE.md` §9 and §10) also has `only` and `fair` scopes, a bound on the claim, and named verdicts (deadlock, cycle, missed deadline, verified with `longest`, unknown) with hint lines that say which fix applies. Kettle's own archived test (`samples/Kettle.test.scala`) exercises every one of these on Kettle, so this is Kettle parity, not a new sample: the questions fn-213.4 must ask need this vocabulary. The spec also settles one semantic divergence: Rust currently makes every `#[internal]` action weakly fair; the owner chose Scala's rule, nothing is fair unless a scope says `fair`.

## Architecture & Data Models

- `Assumption` (umpire-dsl; renamed from `Scope`, owner 2026-10-19): a plain value with three chaining builders, `without([a, ..])`, `only([a, ..])`, `fair([a, ..])`, over actor-level action values; `#[machine]` maps them into the generated `<mod>::Action`. `only` and `without` select the admitted actions (`only` wins when both are present: admitted = only minus without); `fair` lists weakly fair actions and admits nothing by itself. Nothing is fair by default. `#[internal]` keeps its meaning "not driven by the environment" and no longer implies fairness.
- `Progress` (umpire-dsl): `{ trigger: bool, target: bool, within: Option<u32> }` evaluated per state; `.within(n)` sets the bound, `n` a positive constant. A claim names its assumption in its expression, `.assuming(Self::never_switched_off())` (owner 2026-10-19); several compose by chaining on a named one, `Self::reporting().fair([..])`.
- Checker verdicts (umpire-checker): a progress claim's result carries a `Verdict` kind in Scala's order of testing: `Deadlock` (a region state has no admitted successor), `Cycle` (a cycle avoiding the target that no fair action leaves; a fair action leaves a cycle when it is enabled at every state of it and never taken inside it), `MissedDeadline` (bounded only: a path of `n` steps reaches no target, or a cycle that a fair action leaves can still avoid the target), `Verified` with `longest: Option<u32>` (the longest path to the first target state; `None` when an unbounded claim's region cycles), `Unknown` (cut exploration or a hole). The aggregate `Outcome` keeps Verified/Refuted/Unknown; the kind is additional.
- Hint lines, Scala's wording with Rust names: `deadlock: <a>, <b> ends in <state>`; `cycle <a>, <b>: <action> stays enabled and is never taken; add fair(<action>) to the assumption, or admit fewer actions with only`; `cycle <a>, ..: every action enabled on it is taken; admit fewer actions with only`; `a cycle can avoid a target state for any number of steps`; `a path of <n> steps reaches no target state`.
- Questions beyond `verify_all` are asked from fn-213's `tests.rs`: `Model::boils.assuming(Model::press_only())` re-states a claim under another assumption for one question (Scala's `verify(boils) in scenario`), and `Model::boils.within(80)` asks a bounded variant. fn-213.4 depends on this spec.

## API Contracts

- `Assumption::without(actions) / only(actions) / fair(actions)`, each `(self, actions) -> Assumption` and as free constructors; `Assumption::allows(&action) -> bool`, `Assumption::is_fair(&action) -> bool`; `Progress::assuming(self, Assumption) -> Progress`.
- `Progress::within(self, n: u32) -> Progress`.
- `ClaimResult::verdict: Verdict` with `Verdict::{Deadlock, Cycle, MissedDeadline, Verified { longest: Option<u32> }, Unknown}`; `ClaimResult::hint: Option<String>`; the refuting `Path` keeps `cycle_start` for cycles.
- fn-213 suite: `verify(&space, claim) -> ClaimResult`, the claim being any `Progress` value, so `Model::boils.assuming(..)` and `.within(n)` compose per question.

## Edge Cases & Constraints

- `within(0)` or an assumption naming an unknown action is a `Refusal` naming it (compile-time where `#[machine]` can see it, runtime otherwise).
- A bounded claim never benefits from fairness: any cycle avoiding the target fails it (as a cycle when no fair action leaves it, as a missed deadline when one does).
- `end` states are not part of this spec; when `end` arrives, a deadlock test ignores `end` states as Scala did.
- fn-210 R14 determinism and R30 budgets hold; Kettle's existing goldens must not change (the fairness rule change is invisible on Kettle inside `never_switched_off`).

## Acceptance Criteria

- **R1:** `Assumption` (renamed from `Scope`, with `#[assumptions]` and `.assuming(..)`) chains `without`, `only` and `fair` over actor-level actions; nothing is fair by default and `#[internal]` no longer implies fairness. Errors: an unknown action in an assumption is refused naming it; `only` and `without` of the same action admits nothing and is reported as an empty assumption.
- **R2:** `Progress::within(n)` makes a claim bounded; `n` must be positive. Errors: `within(0)` is refused.
- **R3:** Progress verdicts are `Deadlock`, `Cycle`, `MissedDeadline`, `Verified { longest }` or `Unknown`, tested in that order, each with a path and the hint line above; the aggregate `Outcome` is unchanged. Errors: none beyond R1/R2; a cut exploration answers `Unknown`.
- **R4:** Kettle answers its archived test's progress questions: `boils` in `never_switched_off` is verified with longest 81 from room temperature and 141 from the coldest start; on the whole kettle from room temperature it is a cycle `depress, press` with the `only` hint; assuming `only([User::PressButton])` it is a deadlock ending in the switched-on room-temperature state; with `within(80)` from room temperature it is a missed deadline; with budget 10 it is unknown. Pinned as checker fixtures and, once fn-213 exists, asked from Kettle's `tests.rs`. Kettle's existing goldens are byte-identical. No error surface.
- **R5:** fn-212's emitter contract is recorded: each `fair` entry becomes `WF_vars(A)` and nothing else is fair; stated in `DESIGN.md`. No error surface.

## Boundaries

- No `end`, monitors, strong fairness, `starts(..)`, pinned schedules or refinement: `starts` and schedules belong to fn-213's `Space`/`Run`; the rest to the samples that need them (parked).
- No change to the attribute surface beyond `only`, `fair` and `.within(n)`.
- Does not start before fn-210.19.

## Decision Context

2026-10-19, owner: fairness is explicit (`fair([..])` in the scope), never implied by `#[internal]`; this matches Scala's rule and TLA+'s `WF_vars(A)`, and makes the assumption visible where a reader looks for it. Kettle's goldens hide the divergence (no internal step stays enabled on its cycle), which is why it went unnoticed. Named verdicts and hint lines reuse Scala's wording. The spec is Kettle parity (its archived test covers deadlock, cycle, missed deadline and verified with longest), so no new sample is needed; the samples port stays parked.

2026-10-19, owner: `Scope` becomes `Assumption` (`#[assumptions]`, `.assuming(..)` on the claim) so the sentence reads 'boils, assuming never switched off'; the word is SEMANTICS §10's. The `#[within(..)]` attribute is gone; `.within(n)` is only the bound. fn-215 adds the `path!` progression notation on top of this spec's `Progress`.
