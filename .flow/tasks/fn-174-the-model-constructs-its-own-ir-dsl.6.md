---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.6 Claims construct their messages: Properties, Scenarios, Queries, progress; the lifter's fold goes

## Description
Depends on .4. Property and progress predicates, `holds`/`holdsAcross`, `leadsTo` `from`/`to` and Scenario functions are capture points calling `Reify.function`. Properties, Scenarios, Queries (including those yielded by comprehensions and helper defs, which now export: fn-141 R13) and progress claims build their messages. The lifter's `Claims.scala` and its `fold` evaluator go.

## Acceptance
- Every claim comes from constructed values; `export/Claims.scala` and the fold are deleted (R3).
- A fixture yields Queries from a comprehension, each with its own name (R4, fn-141 R13).
- Names equal today's (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
