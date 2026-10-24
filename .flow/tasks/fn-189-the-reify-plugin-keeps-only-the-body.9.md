---
satisfies: [R5, R10]
---
# fn-189-the-reify-plugin-keeps-only-the-body.9 After .1: root-only positions inside the body lifter; drop the at/pos threading (R5, R10)

## Description
**Note 2026-10-18 (one-language series):** `export` callers become `lift` callers (fn-201).

Touches: model/framework/reify/body (Expressions, BodyEngine, BodySyntax, BodyContext, Types, Roles), model/framework/reify/plugin, model/framework/export (callers of the shared engine)

### Description
Finish R5 inside the body lifter. Once a stored `ir.Model` carries a position on its root function only, no lifted `ir.Expr` needs one: stop writing `ir.Expr.position` and remove the `at: Tree` parameter that every helper (`expr`, `lit`, `text`, `list`, `binary`, `step`, `inbox`, `copyOf`, `typeRef`, `declareType`, `optionType`, ...) threads only to fill it. Remove `placing`, `unplacing`, `argumentAt`, `boundValuePositions`, `placedAt` and the position arguments of `ruleGuard`'s `phasePosition`. Root positions stay on `ir.Function.position`; refusals keep reporting at the tree through `fail(t, ...)`, which is unchanged. Where `export` still shares the engine, its callers drop the same arguments; the old lifter's own position fixtures that assert per-expression positions are retired with fn-174's "equivalence except positions" rule, not kept green. Rough size of the removal: the `at`/`pos` plumbing is spread across most helper signatures in `reify/body`.

**Size:** M

### Verification
- The six samples compile with -Xplugin and lift; stored IR differs from before only in `ir.Expr.position` fields, which are now absent.
- Compile-time refusal fixtures report at the same positions as before.
- grep shows no `position = Some(pos(` on an `ir.Expr` in `reify/body` and no `placedAt`/`placing`/`argumentAt`.
## Acceptance
- [ ] No `ir.Expr` built in `reify/body` carries a position; only `ir.Function.position` is written.
- [ ] The `at: Tree` parameter exists only where a refusal needs it; `placing`, `unplacing`, `argumentAt` and `boundValuePositions` are gone.
- [ ] Refusal positions are unchanged (existing refusal fixtures pass).


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: