---
satisfies: [R1, R2, R3]
---

# fn-215-the-path-pattern-progressions-for.1 Path and path!: pattern, claim restriction, Kettle in path! form, reading rules
# fn-215-the-path-progression-notation-for.1 path! macro, Kettle in path! form, reading rules in README

## Description
`Path<S>` and `path!` in `umpire-dsl` (`progress.rs` or a new `path.rs`): waypoints from `bool` / `Fn(&S) -> bool` / `S`, gaps `..` and `..=n`; `macro_rules!` arms for two or more waypoints, catch-all arms for the refusals (no gap; leading or trailing gap), each pinned by a trybuild `.stderr` fixture in `dsl/tests/ui/`. `Progress` wraps a two-waypoint `Path`; `whenever(..).eventually(..)` / `.within(n)` build the same value and stay public. `#[machine]` refuses a claim with more than two waypoints (`a progress claim has one trigger and one target; ask a longer progression with find or Run::along`), pinned by a trybuild fixture in `dsl/macros/tests/ui/`. Rewrite Kettle's `boils` as `path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())` and the `machine` macro fixture likewise; goldens byte-identical. `README.md`: the reading rules beside Kettle (claim: every admitted path; `find`/`Run::along`: some path, shortest per leg; `..=n` bounds a gap; nothing is said between or after waypoints). DESIGN.md's comparison shows the Scala claim and the `path!` claim side by side. `find`/`Run::along` themselves are fn-213.3/.2 (R4), not this task.

**Prerequisite:** fn-214 done. **Touches:** `model/umpire/dsl/src/**`, `dsl/tests/ui/`, `dsl/macros/src/machine.rs`, `dsl/macros/tests/ui/`, `model/umpire/tests/fixtures/kettle/model.rs`, `dsl/macros/tests/fixtures/machine`, `model/README.md`, `model/DESIGN.md`. **Feedback loop (R30):** paste `make model-time` into the done summary.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
