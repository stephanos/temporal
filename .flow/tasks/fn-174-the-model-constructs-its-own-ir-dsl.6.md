---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.6 Claims construct their messages: Properties, Scenarios, Queries, progress; the lifter's fold goes

## Description
**Recut 2026-10-18 (one-language series):** fn-174 R5's oracle is now Scala checker answers, fn-203 Cases and fn-206/207 backends; the IR stays in-process (fn-201).

Depends on .4. Property and progress predicates, `holds`/`holdsAcross`, `leadsTo` `from`/`to` and Scenario functions are capture points calling `Reify.function`. Properties, Scenarios, Queries (including those yielded by comprehensions and helper defs, which now export: fn-141 R13) and progress claims build their messages. The lifter's `Claims.scala` and its `fold` evaluator go.
## Acceptance
- Every claim comes from constructed values; `export/Claims.scala` and the fold are deleted (R3).
- A fixture yields Queries from a comprehension, each with its own name (R4, fn-141 R13).
- Names equal today's (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
