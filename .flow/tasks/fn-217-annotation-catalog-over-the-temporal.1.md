---
satisfies: [R2, R6]
---
# fn-217-annotation-catalog-over-the-temporal.1 umpire-annotations: generic kinds, tags, rules, expectations, validate and fill

## Description
New crate `model/umpire/annotations/` (`umpire-annotations`), system-agnostic:
`Kind` with protovalidate-named constraints, `Tag` trait with `Consequence`,
`Rule` (JSON Schema presence algebra: required, any_of, one_of, requires, excludes),
`Expectation`, `Catalog` keyed by field paths, `validate`, seeded `fill`,
`Violation`, and `GapReport` with deterministic readable output. Admission failures
must be typed and must retain the information needed by task .2's compile-time
catalog macro. Preserve the approved spec's full vocabulary and constraints,
including external Secret values, redaction, and expectations for request rules.

Use actual typed field access, not protobuf reflection or another validation engine.
The public surface must be consumable by task .2's exhaustive prost struct patterns.
Borrow protovalidate names and semantics; reuse existing approved dependencies only.
Test valid and invalid values, each presence operator, deterministic filling,
non-fillable and decided fields, tags and admission errors, and secret redaction.
Goldens must reflect actual validation and gap-report output, not hand-built reports.
System-specific tags and catalogs arrive in later tasks.

**Touches:** `model/umpire/annotations/**`, `model/Cargo.toml`, `model/Cargo.lock`
**Quick:** `cd model && mise exec -- cargo test -p umpire-annotations`; `cd model && mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings`
**Full gates:** `make model-check`; `make model-time` (record feedback-loop timings).
## Acceptance
- Generic crate exposes kinds, typed fields, tags/consequences, presence rules,
  expectations, catalog, validation, seeded filling, violations and gap-report types.
- Each operator and implemented kind constraint has meaningful positive and negative
  controls with deterministic report goldens; no silent validation or filling failure.
- Filling respects declared roles and constraints; identical seed/input/catalog gives
  identical output; impossible generation and missing external secrets are refused.
- Secret values cannot appear in validation or gap reports; admission refuses
  conflicting tags, unsupported kind/type pairs, ignored-field rules and missing
  request expectations. Task .2 will pin the corresponding compile-time errors.
- Generic source boundary and focused/full checks pass; no system names or new
  third-party dependencies are introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
