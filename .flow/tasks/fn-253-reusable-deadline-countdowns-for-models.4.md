---
satisfies: [R2]
---
# fn-253-reusable-deadline-countdowns-for-models.4 Front end: lower countdown calls in handlers, effects and claims

Touches: [model/umpire/front/src/library.rs, model/umpire/front/src/body/**, model/umpire/front/src/shape.rs, model/umpire/front/tests/**]

## Description
Admit `countdown::` calls where Models need them: assignments and conditions in `#[on]`/`#[effect]`, value-returning calls, and laws in `#[invariant]`/`#[postcondition]`. Lower them from `countdown.rs` source, with no recognition by name. The backends receive ordinary IR, so no emitter change is expected; if one is needed, record why.

**Size:** M
**Files:** `model/umpire/front/src/library.rs`, `body/{claim,expr,effects,optional,typer}.rs`, `shape.rs`, the front end's refusal goldens.
**Touches:** [model/umpire/front/src/library.rs, model/umpire/front/src/body/**, model/umpire/front/src/shape.rs, model/umpire/front/tests/**]

### Approach
- Register `countdown.rs` beside `CLOSABLE` in `library.rs`; generalize the library lookup instead of copying it.
- Extend `body/claim.rs:165`'s claim-statement path to expression position: bind arguments, inline the library body, keep the result typed as the Model's optional countdown.
- Unsupported forms refuse at their span: aliased or glob imports, qualified macro paths, nested Options and non-countdown arguments.
- Add refusal golden rows for each new refusal.

### Investigation targets
**Required:**
- `model/umpire/front/src/library.rs` — library source registration
- `model/umpire/front/src/body/claim.rs` — current library-call lowering
- `model/umpire/front/src/body/optional.rs` — optional typing
- `model/umpire/front/src/shape.rs:557` — DSL path resolution test

**Optional:**
- `model/README.md` (fn-227.11 paragraph) — the closure-law lowering contract

### Acceptance
- [ ] A Model calling every `countdown` function and law compiles, is admitted, and the interpreter and TLC agree with native on it (proved in .5).
- [ ] Each unsupported form refuses at its span with a refusal golden row.
- [ ] No backend recognizes `countdown` by name.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
