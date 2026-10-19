---
satisfies: [R2, R6]
---
# fn-217-annotation-catalog-over-the-temporal.1 umpire-annotations: generic kinds, tags, rules, expectations, validate and fill

## Description
New crate `model/umpire/annotations/` (`umpire-annotations`), system-agnostic: `Kind` with protovalidate-named constraints, `Tag` trait with `Consequence`, `Rule` (JSON Schema presence algebra: `required`, `any_of`, `one_of`, `requires`, `excludes`), `Expectation`, `Catalog` keyed by field paths, `validate`, `fill` (seeded), `Violation`, `GapReport` types and their `Spell` output; admission rules as typed errors; fixtures for each operator and kind constraint (goldens of violation reports). No Temporal name (fn-216 R5 gate). **Touches:** `model/umpire/annotations/**`, `model/Cargo.toml`. **Feedback loop (R30):** paste `make model-time`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
