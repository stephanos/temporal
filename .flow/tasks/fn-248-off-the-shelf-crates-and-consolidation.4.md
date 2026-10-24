---
satisfies: [R1, R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.4 heck, darling and quote in the macro and build crates

## Description
Replace hand-written code generation and parsing in the macro and build crates.

- `heck` for case conversion: `model/umpire/catalog/build/src/wire_schema.rs` (~456, ~473), `model/umpire/catalog/syntax/src/derive.rs` (~632), `model/temporal/annotate/syntax/src/realization.rs` (~558, ~575). The two existing `camel` copies differ (one filters empty parts); decide from the snapshot diffs and record which behavior is kept.
- `darling` 0.24 (syn 3) for catalog `ApiAttributes` (`model/umpire/catalog/syntax/src/derive.rs` ~76-299: `Parse`, `set_once`, `attributes`, `merge`) and `proposal_handler` (`model/temporal/annotate/syntax/src/realize.rs` ~812-845): flags, `key = value`, rename, duplicate detection, error accumulation. Hand-written parsing stays for source-ordered tag/constraint families, positional `also(uuid, ...)`, ordered `oneof(...)`, `rejects(...)`, the rule/mirror split, and the `Declaration` parser (`realize.rs` ~105-155). If darling makes a part worse, leave that part and record why.
- catalog/build Rust generation (`wire_schema.rs`, `services.rs`, ~85 `write!`/`push_str` calls) moves to `quote!` + `prettyplease`, for generated Rust only (protobuf schema text stays string-built).

Re-accept trybuild `.stderr` and expansion snapshots after reviewing the diffs.

## Acceptance
- [ ] No hand-written snake/camel converter remains; generated names match prost-build's.
- [ ] `ApiAttributes` and `proposal_handler` parse through darling; malformed attributes still refuse at the offending span.
- [ ] catalog/build emits generated Rust through `quote` + `prettyplease`.
- [ ] fn-210's R18 note records heck and darling; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
