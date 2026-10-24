---
satisfies: [R5, R6, R7, R8, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.8 Retire the lifter for IR: driver only, inspector for lints, fixtures in-process, rules of record, counts

## Description
**Recut 2026-10-18 (one-language series):** `export` is `lift` (fn-201); 'expected IR regenerated' becomes fn-198 case folders over the in-process IR; the oracle is the Scala checker, fn-203 Cases and fn-206/207 backends.

Depends on .5, .7 and fn-141.6 (refusal ledger). Delete `Lift.scala`'s inspector and target handling for IR, `Lifting`, `Context`'s tree parts and `Syntax.scala`; `export` is the driver and its plumbing. `inspect`/`lint` keep the inspector for the order, structure and marker lints. Migrate the 331 fixtures: each is a Model that compiles and runs in-process; refusal fixtures are checked through `typeCheckErrors` or a compile; expected IR is regenerated and reviewed. Apply the refusal ledger: every refusal kind has its outcome and its home. Rewrite the rules of record (`.plans/DSL_OPERATORS.md` rule 5, `.plans/SCALA.md`, `model/README.md` TASTy paragraph, `model/framework/SEMANTICS.md` §21, `Lift.scala` header) and update `model/design/scala-macros-export.md` §7/§8 with the outcome. Record source lines of `export`, `reify`, `inspect` and `lint` before and after. Close fn-141 (its .5, .8–.14 are superseded here).
## Acceptance
- `model/framework/export` emits no IR from trees; the inspector runs for lints only (R8).
- Every fixture compiles and runs in-process; every refusal fixture refuses at its line (R7, R8).
- Every ledger row has an outcome and a home (R7).
- The rules of record no longer state the retired reason (R8).
- Two-run determinism passes on every fixture (R6).
- Line counts before/after recorded; receipts, Cases and backend answers unchanged (R5, R8).
- No Model edit beyond `@lifted` and the `records` decision (R9).

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: