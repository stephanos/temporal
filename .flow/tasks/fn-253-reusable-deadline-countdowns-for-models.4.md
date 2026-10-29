---
satisfies: [R2]
---
# fn-253-reusable-deadline-countdowns-for-models.4 Front end: lower countdown calls in handlers, effects and claims


## Description
Admit `countdown::` calls where Models need them: assignments and conditions in `#[on]`/`#[effect]`, value-returning calls, and laws in `#[invariant]`/`#[postcondition]`. Lower them from `countdown.rs` source, with no recognition by name. The backends receive ordinary IR, so no emitter change is expected; if one is needed, record why.

**Size:** M
**Files:** `model/umpire/front/src/library.rs`, `body/{claim,expr,effects,optional,typer}.rs`, `shape.rs`, the front end's refusal goldens; `model/umpire/author/dsl/macros/src/lifted.rs` and `model/umpire/author/dsl/src/countdown.rs` for required native type/value plumbing.
**Touches:** [model/umpire/front/src/library.rs, model/umpire/front/src/body/**, model/umpire/front/src/shape.rs, model/umpire/front/tests/**, model/umpire/author/dsl/macros/src/lifted.rs, model/umpire/author/dsl/src/countdown.rs, model/umpire/author/dsl/tests/**, model/umpire/author/dsl/macros/tests/ui/action_finite_missing.stderr, model/umpire/author/umpire/tests/consumer/src/lib.rs, model/umpire/backends/tla/src/values.rs]

### Approach
- Consume .2's chosen representation and refusal-owner list before fixing the admission changes. The files above are investigation anchors. Source-local wrappers and imported library-owned const-generic wrappers require different type admission; `body/optional.rs` currently requires nongeneric source-local `Finite` types. Reanchor the affected front-end owners from the verdict rather than assuming expression lowering alone admits the chosen types.
- Register `countdown.rs` beside `CLOSABLE` in `library.rs`; generalize the library lookup instead of copying it.
- Route handler, effect and claim expression calls through shared library-call dispatch, argument binding and result typing. Keep project-library registration generic; no Temporal-specific dispatch or duplicated countdown lowering paths.
- Extend `body/claim.rs:165`'s claim-statement path to expression position: bind arguments, inline the library body, and preserve the configured/remaining nominal types and optional descriptors chosen in .2.
- Unsupported forms refuse at their span: aliased or glob imports, qualified macro paths, nested Options and non-countdown arguments.
- Add refusal golden rows for each new refusal.

- Native compilation must exercise the imported types: decode canonical instantiated bounded names as typed Rust paths rather than treating them as identifiers (`author/dsl/macros/src/lifted.rs`), and implement the existing `IntoValue` contract for the shared countdown types (`author/dsl/src/countdown.rs`). Preserve ordinary IR/value representation and add focused regression coverage. These owners were identified during .4 investigation; they close its existing compile/admit contract, not a new backend feature.

- Exercise generated decode with a real renamed facade-only consumer (`author/umpire/tests/consumer/src/lib.rs`); canonical qualified paths must remain absolute so existing dependency-root rewriting applies. Keep countdown export names independent of rustc type-name pretty-printing, including the maximal bound. Refresh only the inherited missing-Finite diagnostic implementor hints after the intentional shared Finite implementations, preserving the Opaque refusal (`author/dsl/macros/tests/ui/action_finite_missing.stderr`).

- Backend schema integration discovered in review: qualified instantiated bounded names are nominal metadata, not emitted TLA identifiers. Update `backends/tla/src/values.rs` generically to accept qualified nominal type paths, retaining malformed-name checks and strict emitted enum/field/module identifiers. Add a real admitted, World-bound countdown Model emission/schema regression. This is validator compatibility with ordinary IR, not name-specific countdown lowering; full TLC agreement/mutants remain .5.

### Investigation targets
**Required:**
- `model/design/deadline-countdowns.md` — chosen representation and admission owners
- `model/umpire/front/src/library.rs` — library source registration
- `model/umpire/front/src/body/claim.rs` — current library-call lowering
- `model/umpire/front/src/body/optional.rs` — optional typing
- `model/umpire/front/src/shape.rs:557` — DSL path resolution test

**Optional:**
- `model/README.md` (fn-227.11 paragraph) — the closure-law lowering contract

## Acceptance
- [ ] A Model calling every `countdown` function and law compiles, is admitted, and the interpreter and TLC agree with native on it (proved in .5).
- [ ] Each unsupported form refuses at its span with a refusal golden row.
- [ ] No backend recognizes `countdown` by name.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
