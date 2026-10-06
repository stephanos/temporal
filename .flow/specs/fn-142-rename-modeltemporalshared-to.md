# Rename model/temporal/shared to foundations

## Goal & Context
[user] 2026-10-06: `shared` undersells what the folder holds; its entities (the task queue, the worker) are what features build on. Rename it `foundations` (after the Foundations team, and for what the entities are). `Bounds.scala` is not an entity and moves to `model/temporal/Bounds.scala`, beside `Client.scala`.

## Acceptance Criteria
- **R1:** `model/temporal/shared/{taskqueue,worker}` live at `model/temporal/foundations/{taskqueue,worker}`, with packages `temporal.foundations.*`; no `temporal.shared` or `shared/` Model path remains in source, docs (`model/README.md`, `.plans/UMPIRE_MODULES.md`, `.plans/*`), lints or tests, except dated history.
- **R2:** `Bounds.scala` is `model/temporal/Bounds.scala` in package `temporal`.
- **R3:** After `make umpire-gen-model`, `model/ir` and `model/cases` differ only in source paths and positions; every Query answer, receipt and Definition ID is unchanged. The structure lint and the module map name the new layout.

## Boundaries
Runs after the DSL batch closes (it would otherwise mix path changes into the batch's single diff check). Mechanical move only; no declaration changes.
