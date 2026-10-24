---
satisfies: [R3, R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.3 IR child traversal: ExprKind children and children_mut

## Description
Add non-allocating `ExprKind::children(&self) -> impl Iterator<Item = &Expr>` and `children_mut(&mut self) -> impl Iterator<Item = &mut Expr>` to umpire-ir (`model/umpire/ir/src/expr.rs`), one exhaustive match each (for example `([Option<&Expr>; 3], &[Expr])` flattened and chained).

Use them in the recursion-only walks:
- `Expr::infallible`: keep the explicit `false` arms (`Local`, `Unary(Neg)`, `GetOr`, `Some`, `Binary(Add|Sub|Mul|Div|Rem, ..)`, written as a pattern since clippy flags the guard form), with every other variant returning `children().all(Self::infallible)`.
- `model/umpire/backends/tla/src/names.rs` `rename`: the override check, then recursion over `children_mut`.
- `model/umpire/backends/tla/src/emit.rs` `prime`: clone plus a `prime_in_place` over `children_mut`.
- `model/umpire/dev/fixtures/interpreted.rs` `rewrite`.

Walks that do per-variant work stay as they are: TLA+ `render`, `tla/safety/value.rs::emit`, native lowering in `interpret.rs`, front `body/vector.rs` `bounded`/`bound`, `body/effects.rs::define`.

Precedes fn-245.1, which then extends the two helpers for its new variants.

## Acceptance
- [ ] `children`/`children_mut` exist with unit tests covering every variant arity.
- [ ] The four recursion-only walks use them; per-variant walks are unchanged.
- [ ] Interpreter-versus-compiled differential checks and TLC agreement pass; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
