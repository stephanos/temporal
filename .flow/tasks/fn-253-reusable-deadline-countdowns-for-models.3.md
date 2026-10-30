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
Implemented the fixed native `umpire_dsl::countdown` surface: public bounded Ticks/Remaining types, Finite domains and membership, arm/tick/armed/expired, and explicit within/descends laws. The six unit tests exercise absence, armed zero, positive/maximal counts, forged raw values and N=0 through real claim frames. No front-end admission or Temporal migration is claimed; those remain later tasks.

Measured verification: 73 umpire-dsl tests pass, including six countdown tests; strict package Clippy (all targets/features), scoped rustfmt and wasm32-wasip1 build pass. Initial tests failed on the absent API before implementation.

`make model-check` was attempted and failed in unchanged existing xtask gate-budget tooling: strict Clippy reported 94 library and 30 library-test errors, including unused budget/context items and existing scheduler/count-authority lint violations. The xtask source is identical to baseline f998db36bb. Full gate success remains open; owner-paused fn-254.12 was not resumed or modified. Full output: .flow/tmp/fn2533-model-check.log.

stage: impl-review - ran; SHIP, no findings, configured claude:claude-fable-5-1:high (cross-family), one correctness reviewer for this isolated native module. Review reservation f1d16c84d2de4ab88aa16bb3742fe338.
Tier: implementer - gpt-6.1-sol at medium effort (model: gpt-6.1-sol, host spawn metadata).

Task-specific scope is complete. R2's source lowering, admission and mutant coverage remain fn-253.4/.5; final combined validation remains fn-253.8. Broader milestone execution and migration remain paused/deferred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e5e5455b937515fdd3cefb66fb35f851a7d46962
- Tests: cd model/umpire && mise exec -- cargo test -p umpire-dsl --lib countdown --locked, cd model/umpire && mise exec -- cargo test -p umpire-dsl --locked, cd model/umpire && mise exec -- cargo clippy -p umpire-dsl --all-targets --all-features --locked -- -D warnings, cd model/umpire && mise exec -- cargo build -p umpire-dsl --target wasm32-wasip1 --locked, make model-check (FAILED: unchanged xtask gate-budget Clippy errors; recorded in summary)
- PRs: