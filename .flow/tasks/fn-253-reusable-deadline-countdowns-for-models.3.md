---
satisfies: [R2]
---
# fn-253-reusable-deadline-countdowns-for-models.3 umpire_dsl::countdown: functions and laws in plain Rust

Touches: [model/umpire/author/dsl/src/countdown.rs, model/umpire/author/dsl/src/lib.rs]

## Description
Add the `countdown` module to `umpire-dsl` with the surface .2's verdict fixed. Native only in this task: it compiles, has unit tests and docs. The front end comes in .4.

**Size:** S
**Files:** `model/umpire/author/dsl/src/countdown.rs` (new), `model/umpire/author/dsl/src/lib.rs`.
**Touches:** [model/umpire/author/dsl/src/countdown.rs, model/umpire/author/dsl/src/lib.rs]

### Approach
- Mirror `closable.rs`: generic, no Temporal vocabulary, laws use `check!`, doc comments state what each law does *not* promise.
- Write bodies only in the subset the front end will lower (spec Edge Cases): no combinators, no pattern matching on Option, eager `unwrap_or`.
- `tick` never arms an absent countdown or goes below zero; `within` asserts `1..=configured` for armed countdowns and says why (expiry clears in the same step).
- Unit tests cover absent, armed, zero and maximum values for every function and law.

### Investigation targets
**Required:**
- `model/design/deadline-countdowns.md` — the fixed surface
- `model/umpire/author/dsl/src/closable.rs` — style and doc pattern
- `model/umpire/author/dsl/src/lib.rs:8-15` — module exports

## Acceptance
- [ ] `umpire_dsl::countdown` exports exactly the verdict's functions and laws, documented.
- [ ] Unit tests pass for absent/armed/zero/maximum cases; wasm32-wasip1 build of `umpire-dsl` still passes.
- [ ] No Temporal names in the module (gate check for model/umpire).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
