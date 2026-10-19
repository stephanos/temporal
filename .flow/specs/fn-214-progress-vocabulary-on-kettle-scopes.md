# Progress vocabulary on Kettle: scopes, bounds, fairness and verdicts (model)

> Local HTML render lens: `.flow/artifacts/fn-214-progress-vocabulary-on-kettle-scopes/spec.html`. Regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-19): "do we have the Temporal logic API well flhes out?" — answered: no; Rust has `whenever/eventually` and `without` only, and `#[internal]` implies weak fairness where Scala and TLA+ make fairness explicit.
> user (2026-10-19), after the design walkthrough with Kettle examples: "yes, write new spec".

## Goal & Context

Umpire in Rust checks one progress form today: `whenever(p).eventually(q)` inside a `without` scope, answering verified, refuted or unknown. Scala's progress vocabulary (archived `SEMANTICS.md` §11, `REFERENCE.md` §9 and §10) also has `only` and `fair` scopes, a bound on the claim, and named verdicts (deadlock, cycle, missed deadline, verified with `longest`, unknown) with hint lines that say which fix applies. Kettle's own archived test (`samples/Kettle.test.scala`) exercises every one of these on Kettle, so this is Kettle parity, not a new sample: the questions fn-213.4 must ask need this vocabulary. The spec also settles one semantic divergence: Rust currently makes every `#[internal]` action weakly fair; the owner chose Scala's rule, nothing is fair unless a scope says `fair`.

## Architecture & Data Models

- `Assumption` is the existing plain DSL value, with `without`, `only` and `fair` constructors and receiver chaining over actor-level actions. The macro maps those actions into the generated action catalog. Admission is the only-selection minus exclusions, independently of chaining order. Repeated `only` replaces the selection; repeated `without` and `fair` accumulate stable deduplicated entries. Fair membership admits nothing by itself. Nothing is fair by default. `#[internal]` means environment-independent only. Preserve the manual-trait `Scope` alias and its static `without` constructor; one extension trait supplies compatible receiver chaining.
- Author-facing `LeadsTo` carries trigger, target, assumption and an optional positive numeric bound. `.within(n)` counts transitions from an obligation state to its first target, excluding the incoming transition that made the trigger true. A target at exactly n transitions satisfies the bound. A claim names its assumption with `.assuming(Self::never_switched_off())`; named assumptions compose by chaining. Generated metadata must be fixed across reached states, otherwise the claim is Unknown. The legacy registered `Progress<M>` and its scope-taking `.within(scope)` remain compatible; registered numeric bounds use a distinct operation.
- Checker progress results carry `ProgressVerdict` in this order: `Deadlock` (a region state has no admitted successor), `Cycle` (a target-avoiding cycle not excluded by explicit weak fairness), `MissedDeadline` (bounded only, an n-step target-avoiding witness or a cycle excluded only by fairness), `Verified { longest }`, `Unknown`. Longest counts transitions to the first target, is zero without an outstanding obligation, and is absent for a complete unbounded claim whose region can cycle before a fair exit. Refuting logical results have paths and hints; Verified and Unknown need no refutation path. Unknown replaces unproved success on cuts or holes, while concrete refutations remain authoritative. The aggregate Outcome stays Verified/Refuted/Unknown.
- Hint lines, Scala's wording with Rust names: `deadlock: <a>, <b> ends in <state>`; `cycle <a>, <b>: <action> stays enabled and is never taken; add fair(<action>) to the assumption, or admit fewer actions with only`; `cycle <a>, ..: every action enabled on it is taken; admit fewer actions with only`; `a cycle can avoid a target state for any number of steps`; `a path of <n> steps reaches no target state`.
- This spec checks per-question variants through one narrow registered-progress query over an existing exploration. A fixture adapter supplies room/cold starts and delegates original behavior and callbacks. It preserves the canonical sample sources and goldens. Fn-213 later provides ergonomic `tests.rs` claim selection and the illustrated `Model::boils` notation; this spec does not depend on that future surface.

## API Contracts

- Assumption constructors and chains provide `without`, `only`, `fair`, `allows` and `is_fair`. Preserve the existing static `Assumption::without` / `Scope::without`; receiver chaining is supplied by an extension trait in scope in generated modules.
- `LeadsTo::within(self, n: u32) -> LeadsTo` and `.assuming(self, assumption) -> LeadsTo`. Registered progress values expose assumption replacement and a separately named numeric-bound operation. An explicit replacement discards the original metadata completely, including its consistency checks; a bound override changes only the bound.
- `ClaimResult::verdict: Option<ProgressVerdict>` and `ClaimResult::hint: Option<String>`, with `ProgressVerdict::{Deadlock, Cycle, MissedDeadline, Verified { longest: Option<u32> }, Unknown}`. Safety results and caught predicate/effect defects keep existing outcomes without a fabricated temporal classification. The DSL's predicate `Verdict` remains distinct. Refuting cycles retain `Path::cycle_start`. Default report text stays byte-identical; supplementary fixtures render the new fields.
- `Result::verify_progress(&self, world, &registered_claim) -> Result<ClaimResult, Refusal>` reuses the retained graph and progress algorithm, without re-exploration, retained-row cloning or mutation of cached claim results. Fn-213 can wrap this operation in its future suite.

## Edge Cases & Constraints

- `within(0)` or an assumption naming an unknown action is a `Refusal` naming it (compile-time where `#[machine]` can see it, runtime otherwise).
- Validate entries in every assumption list, including excluded fair entries, before deciding admission. An explicit restriction with no admitted catalog action is an empty-assumption Refusal. An unrestricted empty catalog remains valid; an outstanding obligation there is a Deadlock. A press-only catalog that admits PressButton remains valid when that action is disabled in the current state.
- Validate available registration metadata and explicit overrides even when there are no starts. Method-produced metadata is checked only when a real reached state invokes its evaluator. A zero-start Model invokes no evaluator and remains vacuously verified when its available metadata is valid; do not fabricate a state to inspect a callback's return value.
- A bounded claim never benefits from fairness: any cycle avoiding the target fails it (as a cycle when no fair action leaves it, as a missed deadline when one does).
- `end` states are not part of this spec; when `end` arrives, a deadlock test ignores `end` states as Scala did.
- fn-210 R14 determinism and R30 budgets hold; Kettle's existing goldens must not change (the fairness rule change is invisible on Kettle inside `never_switched_off`).

## Acceptance Criteria

- **R1:** `Assumption` (renamed from `Scope`, declared by `#[assumption]` methods, used by `.assuming(..)`) chains `without`, `only` and `fair` over actor-level actions; nothing is fair by default and `#[internal]` no longer implies fairness. Errors: an unknown action in an assumption is refused naming it; `only` and `without` of the same action admits nothing and is reported as an empty assumption.
- **R2:** `LeadsTo::within(n)` makes a claim bounded; n must be positive and reaching the first target in exactly n transitions passes. Errors: zero is refused; inconsistent reached bound metadata is Unknown unless replaced by a per-question bound.
- **R3:** Progress verdicts are `Deadlock`, `Cycle`, `MissedDeadline`, `Verified { longest }` or `Unknown`, tested in that order, with paths and the specified hints for refuting logical results; the aggregate `Outcome` is unchanged. Safety and caught-defect results have no temporal classification. Errors: none beyond R1/R2; cuts and reached holes prevent unproved success while concrete refutations remain visible.
- **R4:** Kettle answers its archived test's progress questions: `boils` in `never_switched_off` is verified with longest 81 from room temperature and 141 from the coldest start; on the whole kettle from room temperature it is a cycle `depress, press` with the `only` hint; assuming `only([User::PressButton])` it is a deadlock ending in the switched-on room-temperature state; with `within(80)` from room temperature it is a missed deadline; with budget 10 it is unknown. Pinned as checker fixtures and, once fn-213 exists, asked from Kettle's `tests.rs`. Kettle's existing goldens are byte-identical. No error surface.
- **R5:** fn-212's emitter contract is recorded: each `fair` entry becomes `WF_vars(A)` and nothing else is fair; stated in `DESIGN.md`. No error surface.

## Boundaries

- No `end`, monitors, strong fairness, `starts(..)`, pinned schedules or refinement: `starts` and schedules belong to fn-213's `Space`/`Run`; the rest to the samples that need them (parked).
- Keep the flat attribute surface; add only the assumption/bound builders and the registered-progress query needed by these questions. No new attributes, suite crate or starts DSL.
- Starts after fn-210's committed local closure. The implementation tasks run .1, then .2, then .3 because they share macro and checker files.

## Decision Context

2026-10-19, owner: fairness is explicit (`fair([..])` in the scope), never implied by `#[internal]`; this matches Scala's rule and TLA+'s `WF_vars(A)`, and makes the assumption visible where a reader looks for it. Kettle's goldens hide the divergence (no internal step stays enabled on its cycle), which is why it went unnoticed. Named verdicts and hint lines reuse Scala's wording. The spec is Kettle parity (its archived test covers deadlock, cycle, missed deadline and verified with longest), so no new sample is needed; the samples port stays parked.

2026-10-19, owner: `Scope` becomes `Assumption` (`#[assumption]` methods, `.assuming(..)` on the claim) so the sentence reads 'boils, assuming never switched off'; the word is SEMANTICS §10's. The `#[within(..)]` attribute is gone; `.within(n)` is only the bound. fn-215 adds the `path!` progression notation on top of this spec's `Progress`.

2026-10-19, naming: the author-facing progress claim value type is `LeadsTo`. The existing registered trait-engine `Progress<M>` remains a separate compatibility surface.

2026-10-19, delegated implementation choices: an optional ProgressVerdict avoids misclassifying failed safety claims. An extension trait preserves the static legacy assumption constructor while adding chains; a Rust 1.99.1 disposable probe confirmed method resolution. The registered query and a delegating fixture adapter deliver R4 without depending on fn-213 or changing canonical Kettle behavior. The archived fair(heat) hint is generic advice; heat is disabled during the off part of the press/depress cycle, so explicit weak fairness alone does not remove it.

## Quick commands

```bash
make model-check
make model-time
```

The feedback-loop targets remain 10 s inner and 30 s full, both soft. Record minor deviations without blocking; investigate material overruns.

## Early proof point

Task .1 proves explicit fairness can replace internal-action fairness while retaining canonical Kettle parity and the independent brute-force oracle. If it fails, resolve assumption admission and fairness representation before bounds or query fixtures are built.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `Assumption` (renamed from `Scope`, declared by `#[assumption]` methods, used by `.assuming(..)`) chains `without`, `only` and `fair` over actor-level actions; nothing is fair by default and `#[internal]` no longer implies fairness. Errors: an unknown action in an assumption is refused naming it; `only` and `without` of the same action admits nothing and is reported as an empty assumption. | fn-214-progress-vocabulary-on-kettle-scopes.1 | — |
| R2 | `LeadsTo::within(n)` makes a claim bounded; n must be positive and reaching the first target in exactly n transitions passes. Errors: zero is refused; inconsistent reached bound metadata is Unknown unless replaced by a per-question bound. | fn-214-progress-vocabulary-on-kettle-scopes.2 | — |
| R3 | Progress verdicts are `Deadlock`, `Cycle`, `MissedDeadline`, `Verified { longest }` or `Unknown`, tested in that order, with paths and the specified hints for refuting logical results; the aggregate `Outcome` is unchanged. Safety and caught-defect results have no temporal classification. Errors: none beyond R1/R2; cuts and reached holes prevent unproved success while concrete refutations remain visible. | fn-214-progress-vocabulary-on-kettle-scopes.2 | — |
| R4 | Kettle answers its archived test's progress questions: `boils` in `never_switched_off` is verified with longest 81 from room temperature and 141 from the coldest start; on the whole kettle from room temperature it is a cycle `depress, press` with the `only` hint; assuming `only([User::PressButton])` it is a deadlock ending in the switched-on room-temperature state; with `within(80)` from room temperature it is a missed deadline; with budget 10 it is unknown. Pinned as checker fixtures and, once fn-213 exists, asked from Kettle's `tests.rs`. Kettle's existing goldens are byte-identical. No error surface. | fn-214-progress-vocabulary-on-kettle-scopes.3 | — |
| R5 | fn-212's emitter contract is recorded: each `fair` entry becomes `WF_vars(A)` and nothing else is fair; stated in `DESIGN.md`. No error surface. | fn-214-progress-vocabulary-on-kettle-scopes.3 | — |
