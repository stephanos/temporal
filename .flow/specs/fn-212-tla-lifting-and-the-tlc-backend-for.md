# TLA+ lifting and the TLC backend for Umpire in Rust (model2)

## Conversation Evidence

> user (2026-10-18), accepting the proposal "move .8 out of fn-210 into its own post-verdict spec next to fn-211, and decide there whether the second opinion is worth the machinery": "yes"
> user (2026-10-18), on dropping `#[machine]` without lifting: "we'll still need it later right?" — answered: yes for registration; token recording and the liftable-subset warnings move here.

## Goal & Context

fn-210 decides whether Umpire is built in Rust on the wasm slice alone (fn-210 R24). Lifting a Model to TLA+ and agreeing with TLC was planned as a second vertical slice inside fn-210 (its R19, task .8), but it does not inform that verdict, and it brings the most machinery: a lifted tree, a TLA+ emitter, a liftable subset that constrains what authors may write, a TLC runner and a dump decoder. This spec takes that slice out of fn-210 and starts after the verdict. Its first question is whether an independent second reading of the Model, plus TLC's temporal checks, is worth that machinery once the native checker covers progress and refinement; only if yes does it build the slice. fn-211 (Apalache) builds on this spec's emitter.

## Architecture & Data Models

- Decision first: a short design note weighs the second opinion (an independent engine reading the Model, exhaustive reachable-set agreement, TLC liveness) against its cost (the crates below, the liftable subset's constraint on authors, a JVM in the gate), using the Models that exist after fn-210. If the answer is no, this spec and fn-211 close with the note.
- Lifting source: either `#[machine]` records its impl's tokens, or a build-time `syn` pass reads the Model's source files; the note chooses. Either way the lifted tree is an internal Rust enum in `umpire-lift` with no stability promise (fn-210 R28), never a golden.
- The liftable subset (fn-210 R13): arithmetic, comparisons, boolean logic, field access, `match` on enums, `imbl` operations, helpers in the same impl; anything else is warned on, naming the expression, until emission needs it, then refused.
- `umpire-lift` emits a TLA+ module and config (vars, `Init` from the starts, `Next` as the disjunction of the transitions, invariants, postconditions as action properties, the scope and progress claims as liveness with weak fairness on internal steps), using the author's Rust names (fn-210 R26).
- `umpire/backends/tlc/` (`umpire-tlc`) runs a pinned `tla2tools.jar` on the mise JDK with the deadlock check on and `-dump`, decodes the dump through `Spell` (fn-210 R25), and registers a TLC consumer in the shared fixture runner (fn-210 R29) owning `<Model>.tla`, `<Model>.cfg` and `tlc.txt`.

## API Contracts

- `Lift::of::<M>() -> Result<Lifted, Refusal>`; `Tla::emit(&Lifted) -> Module { tla, cfg }`.
- `Tlc::check(&Module) -> Result<TlcResult, Refusal>` with reachable states, verdicts per property, and a decoded counterexample path when one exists.
- `tlc.txt` lists reachable-state agreement with the native checker, each property's verdict and agreement, and differences if any.

## Edge Cases & Constraints

- fn-210 R30's feedback-loop budget applies: TLC runs batched across fixtures, or only with `UMPIRE_BACKENDS=require`, whichever keeps the default gate within budget.
- Integer bounds follow fn-210 R11 and are stated in the module.
- An expression outside the subset refuses emission naming it; nothing is silently dropped.

## Acceptance Criteria

- **R1:** A design note records whether the second opinion is worth the machinery, with the cost and benefit on the Models then available; on "no", this spec and fn-211 close citing it. No error surface.
- **R2:** On "yes": Kettle and the shared fixtures TLC can express are lifted and emitted; TLC's reachable set equals the native checker's (283 states for Kettle) and verdicts agree; `<Model>.tla`, `<Model>.cfg` and `tlc.txt` are goldens. Errors: an out-of-subset expression refuses emission naming it; a missing or mis-versioned jar fails before the test; a difference is listed in `tlc.txt` and fails.
- **R3:** The liftable subset is warned on by whichever lifting source the note chooses, pinned by a golden. No error surface beyond R2.

## Boundaries

- No Apalache (fn-211); no Quint.
- Does not start before fn-210 is done and its verdict is Rust.
- No change to what a Model author writes beyond the subset warnings.

## Decision Context

Moved out of fn-210 on 2026-10-18 at the owner's request, because the verdict rests on wasm alone and this slice carries the most machinery; fn-210's R13, R19 and R28 move here (fn-210 R36). `#[machine]` stays in fn-210 as a registration attribute; recording tokens for lifting is this spec's choice, against a build-time `syn` pass that needs no attribute. fn-211 depends on this spec.
