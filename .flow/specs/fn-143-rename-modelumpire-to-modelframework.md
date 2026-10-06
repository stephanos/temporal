# Rename model/umpire to model/framework

## Goal & Context
[user] 2026-10-06: the DSL framework folder `model/umpire` becomes `model/framework`, and its Scala package `umpire` becomes `framework`, so the folder says what it is beside `model/temporal` and `model/irgen`. fn-114.9 had left the framework's name out of scope "unless the owner asks"; the owner asked.

## Acceptance Criteria
- **R1:** The framework lives in `model/framework/` with package `framework` (sub-packages `framework.*`); every Model, kit file, lift fixture, test and lint imports `framework.*`. No `model/umpire` path or `umpire.` framework package reference remains in source, the Makefile (`MODEL_SOURCES`, scalafix lists), `.scalafix.conf`, the syntax lint, the lifter's fully qualified name matches, or the docs (`model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_SPEC.md`), except dated history. Names that mean the Umpire product or tooling (`tools/umpire`, `make umpire-*`, the IR's `umpire.v1` proto package) are unchanged.
- **R2:** After `make umpire-gen-model`, `model/ir` and `model/cases` differ only in source paths and positions (and any fully qualified framework name the IR records, listed); every Query answer, receipt and Definition ID is unchanged.
- **R3:** The lifter, gate and Model tests, `make lint-model`, `make umpire-check-model` and the Go tooling suite pass.

## Boundaries
Runs after the DSL batch closes and after fn-142 (both move Model source paths; one at a time keeps each diff check paths-only). Mechanical rename only; no declaration changes.
