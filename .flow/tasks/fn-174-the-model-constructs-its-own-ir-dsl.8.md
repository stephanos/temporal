---
satisfies: [R5, R6, R7, R8, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.8 Retire the lifter for IR: driver only, inspector for lints, fixtures in-process, rules of record, counts

## Description
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
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
