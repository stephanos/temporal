---
satisfies: [R1, R2, R3]
---
# fn-215-the-path-progression-notation-for.1 path! macro, Kettle in path! form, reading rules in README

## Description
`path!` in `umpire-dsl`: two `macro_rules!` arms, `[$p, .., $q]` and `[$p, ..= $n, $q]`, expanding to `whenever($p).eventually($q)` and `.within($n)`; catch-all arms give the two compile errors (third element; no gap), each pinned by a trybuild `.stderr` fixture in `dsl/tests/ui/`. Rewrite Kettle's `boils` as `path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())` and the `machine` macro fixture likewise; goldens byte-identical. `README.md`: the three reading rules beside Kettle (`..` = on every admitted path, `find` is the existential cousin; `..=n` = within n steps; nothing is said about between or after). DESIGN.md's comparison shows the Scala claim and the `path!` claim side by side.

**Prerequisite:** fn-214 done. **Touches:** `model/umpire/dsl/src/progress.rs` (macro), `dsl/tests/ui/`, `model/umpire/tests/fixtures/kettle/model.rs`, `dsl/macros/tests/fixtures/machine`, `model/README.md`, `model/DESIGN.md`. **Feedback loop (R30):** paste `make model-time` into the done summary.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
