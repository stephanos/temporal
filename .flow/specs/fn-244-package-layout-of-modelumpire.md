# Package layout of model/umpire: consistent nesting, clear names, a crate map (model)

## Conversation Evidence

> user (2026-10-24): "is the package layout in temporal/model/umpire/ appropriate? reasoanble? clear?"
> assistant: the shape matches the architecture, but: two annotation families with overlapping names (`annotations` vs `catalog*`); inconsistent nesting (`dsl/macros`, `catalog/syntax` nested, `ir-derive`, `catalog-build` top level); three confusable test crates (`testing`, `suite`, `fixtures`); the hand-written Go `runner` inside the generated protobuf tree; stale descriptions; `reference/` twice.
> user: "yes queue recommendations"

## Goal & Context

Make `model/umpire` readable from its directory names: one nesting rule, names that show which crates authors use, and generated code separated from hand-written code. Mechanical, no behaviour change.

## Architecture & Data Models

- Nesting: sub-crates live under their parent: `ir-derive` -> `ir/derive`, `catalog-build` -> `catalog/build`. Crate names may stay; directories move.
- Annotations: determine the real relationship between `umpire-annotations` (typed catalogs, validation, generation, gap reports) and the `umpire-catalog*` family. Rename so the split is obvious (for example runtime validation versus declaration and code generation), or merge if they are one concern. Record the choice.
- Test crates: keep `suite` (author-facing), `testing` (dev-only runner) and `fixtures` (shared fixtures), and document which is author-facing; rename only if a name is actively misleading.
- Go: the hand-written `runner` package moves out of the generated protobuf tree `hosts/go/umpire/*`; the generated-code freshness check loses its `runner` exception. The fn-228 spec's import path is updated to match.
- `model/README.md` gains a crate map: each crate or package, its one-line job, and whether Model authors use it. Stale Cargo descriptions (e.g. `umpire-runtime`) are fixed.

## Acceptance Criteria

- **R1:** Rust sub-crates nested consistently; the annotations/catalog split named or merged with the choice recorded; descriptions current; workspace builds; goldens, snapshots and guest bytes unchanged except paths that embed crate locations, each listed.
- **R2:** The Go Runner lives outside the generated tree; the freshness check has no special case; all importers updated; tests green.
- **R3:** `model/README.md` has the crate map; MIGRATE.md's path table and AGENTS/module-map paths updated.

## Boundaries

- No behaviour, Model, DSL surface or protocol change; no new crates beyond moves and an optional merge.

## Decision Context

2026-10-24, owner asked to queue the layout recommendations. Placement: R1 runs after fn-238 and fn-242 close (they edit these crates); R2 runs after fn-228.6 and before fn-228.7, so the write-up names the final Runner path.
