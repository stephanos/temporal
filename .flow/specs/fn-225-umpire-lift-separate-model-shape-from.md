# umpire-lift: separate Model shape from body lowering (model)

## Conversation Evidence

> user (2026-10-19): "is / should temporal/model/umpire/lift/src be separated into lifting the model itself and separately the method/func bodies?"
> assistant: only partly today; recommended an explicit split inside the crate (shape vs body lowering, tree split into declarations and `Expr`), byte-identical goldens, right after fn-224; not two crates yet; a parser shared with `#[machine]` deferred until drift bites.
> user (2026-10-19): "yes, file"

## Goal & Context

After fn-223 `umpire-lift` reads one source form, the `#[machine]` module, in about 3,000 lines. The two jobs are interleaved:

- **Shape:** locate the single `#[machine]` module, admit imports and attributes (`canonical_machine`, conditional compilation, aliases), read state fields and `#[starts]`, action enums and actors, `#[internal]`, claim/assumption/progress registrations and their order. Lives in the first ~360 lines of `lift/src/parse.rs`.
- **Bodies:** lower Rust statements and expressions to the typed `Expr`: the subset typer (`leaf.rs`), expressions (`parse/expr.rs`), vectors (`parse/vector.rs`), handler effects and guards, claim bodies, `path!`/builder progress bodies. Spread over the rest of `parse.rs`, `parse/expr.rs`, `parse/vector.rs` and `parse/progress.rs`.

The shape is fixed by the attribute surface and rarely changes; body lowering grows with every admitted construct (fn-224's self-loop marker, typed outcomes and facts, choice, the activity Model's `Phased`/`reject!` shape). Making the seam explicit keeps each future admission local and reviewable. The Scala history went the same way (fn-189: the reify plugin keeps only the body lifter).

## Architecture & Data Models

- `lift/src/shape.rs` (or `shape/`): from `syn::File` to the declaration tree (fields, starts, actors/actions, claim, postcondition, assumption and progress declarations with their order and spans). It calls body lowering for each body it finds and never inspects statements or expressions itself.
- `lift/src/body/`: `expr`, `effects` (handler arms to guarded branches), `vector`, `claim` (claim and progress bodies, including `path!`), and the typer. Entry points take a typed environment (state fields, locals, action catalog, file for diagnostics) plus a `syn` node and return `Expr` or `Vec<Branch>`; they never walk items.
- `tree.rs` splits into declarations (`Lifted`, `FieldDef`, `ActionDef`, `Branch`, `ClaimDef`, `PostDef`, `ProgressDef`, `AssumptionDef`) and `expr.rs` (`Expr`, `ExprKind`, `Op`, `Unary`, constructors). `emit.rs` may split the same way if that falls out naturally; not required.
- `names.rs`, `build.rs`, `lib.rs`'s public API (`Lift`, `Lifted`, `Tla`, `Diagnostic`, `Metadata`, `Module`) are unchanged.

## API Contracts

- No public API change; no author-visible change; no change to any diagnostic text or location.

## Edge Cases & Constraints

- Every golden is byte-identical: lift refusal goldens, TLA+/cfg, TLC agreement, trybuild, expansion goldens.
- `make model-check` and the required TLC run (`UMPIRE_BACKENDS=require`) pass; R30 budgets hold; no new crates.
- Pure refactor: behaviour changes found along the way are recorded, not fixed here.

## Acceptance Criteria

- **R1:** `umpire-lift` has a shape module that never lowers statements or expressions and a body module that never walks items, with the interface above; `tree.rs` is split into declarations and expressions; `model/README.md`/`model/DESIGN.md` describe the seam in a few lines. All goldens byte-identical; default and required TLC gates pass. Errors: none new; every existing diagnostic keeps its text and location.

## Boundaries

- Not two crates. A crate boundary pays off only with a second consumer.
- Deferred, not in this spec: one shape parser shared by `#[machine]` (`dsl/macros`, ~1,000 lines, which today enforces order and attributes itself) and the lifter, so the two cannot drift. File it when a divergence actually shows up; it touches trybuild and expansion goldens.
- No new admitted constructs.

## Decision Context

2026-10-19, owner approved filing. Placement (root agent): after fn-224, which reworks handler parsing for the self-loop marker, and before fn-218, so later admissions land in the separated body module.
