# TLA+ lifting and the TLC backend for Umpire in Rust (model2)

## Conversation Evidence

> user (2026-10-18), accepting the proposal "move .8 out of fn-210 into its own post-verdict spec next to fn-211, and decide there whether the second opinion is worth the machinery": "yes"
> user (2026-10-18), on dropping `#[machine]` without lifting: "we'll still need it later right?" — answered: yes for registration; token recording and the liftable-subset warnings move here.

> user (2026-10-19): "we'll need TLC for scaling our models; our own model checker is a simple, non-scalable version that won't hold"

The owner has settled the decision to **yes**: TLC is required for larger Models. Task .1 records the lifting-source choice, costs and semantic obligations; task .2 implements the backend. The earlier option to close fn-212 and fn-211 without a backend is superseded. The native checker remains the fast local engine and agreement reference; the small current fixture corpus is not a reason to defer the scaling backend.

## Goal & Context

fn-210 decides whether Umpire is built in Rust on the wasm slice alone (fn-210 R24). Lifting a Model to TLA+ and agreeing with TLC was planned as a second vertical slice inside fn-210 (its R19, task .8), but it does not inform that verdict, and it brings the most machinery: a lifted tree, a TLA+ emitter, a liftable subset that constrains what authors may write, a TLC runner and a dump decoder. This spec takes that slice out of fn-210 and starts after the verdict. Its first question is whether an independent second reading of the Model, plus TLC's temporal checks, is worth that machinery once the native checker covers progress and refinement; only if yes does it build the slice. fn-211 (Apalache) builds on this spec's emitter.

## Architecture & Data Models

- Decision first: a short design note weighs the second opinion (an independent engine reading the Model, exhaustive reachable-set agreement, TLC liveness) against its cost (the crates below, the liftable subset's constraint on authors, a JVM in the gate), using the Models that exist after fn-210. If the answer is no, this spec and fn-211 close with the note.
- Lifting source: either `#[machine]` records its module's tokens, or a build-time `syn` pass reads the Model's source files; the note chooses between those two. Not rustc's internal IR (HIR/THIR/MIR): see Decision Context, 2026-10-18. Types come from the subset's leaves (declared state fields, action enums, `const`s, a fixed `imbl` method table), with a small typer over the subset; `ra_ap_hir` (rust-analyzer as a library, stable Rust) is the recorded escalation if the subset outgrows that. Either way the lifted tree is an internal Rust enum in `umpire-lift` with no stability promise (fn-210 R28), never a golden.
- The liftable subset (fn-210 R13): arithmetic, comparisons, boolean logic, field access, `match` on enums, `imbl` operations, helpers in the same impl; anything else is warned on, naming the expression, until emission needs it, then refused.
- `umpire-lift` emits a TLA+ module and config (vars, `Init` from the starts, `Next` as the disjunction of the transitions, invariants, postconditions as action properties, assumptions and progress claims as liveness), using the author's Rust names (fn-210 R26). Under fn-214's explicit-fairness contract, each admitted `fair(A)` entry emits `WF_vars(A)` and nothing else is fair; `#[internal]` adds no fairness.
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

2026-10-18, owner, lifting source: the lifter does not operate on rustc's internal IR. HIR, THIR and MIR are reachable only through `rustc_private`, which pins a nightly toolchain and breaks across releases (every tool doing it, Clippy, Kani, Creusot, Prusti, hax, Charon, chases a nightly); that contradicts the stock-toolchain criterion of fn-210 R24 and would add a second toolchain to this gate beside the JVM. MIR is also the wrong altitude (a CFG with drops and borrows; recovering `IF a /\ b` is decompilation); THIR is the right shape but the least stable. What the IR would buy is types, which `syn` lacks and Scala's Tasty gave for free. Plan: `syn` over the Model's tokens (from `#[machine]` or a build-time pass) plus a tiny typer whose types follow from the subset's leaves (state fields, action enums, `const`s, the `imbl` table); sugar is kept out of the liftable subset by R3's lint rather than handled, so the lifter stays a tree walk with a handful of desugarings (`+=`, `if`/`else` to guarded conjuncts, `match` to `CASE`, sequential assignments to one primed relation by a small SSA pass). Escalation if the subset grows past that (generic helpers, closures over collections): `ra_ap_hir`, rust-analyzer as a library on stable Rust, weekly releases (admissible under fn-210 R18), at the cost of a very large dependency without semver. With fn-210.18's handlers (`fn name(&mut self)`, enabled iff the state changed) each handler lifts to `guard /\ x' = e /\ UNCHANGED <<..>>` and TLA+'s stuttering supplies the same semantics with no extra encoding.
