---
satisfies: [R1]
---
# fn-248-off-the-shelf-crates-and-consolidation.4 heck, darling and quote in the macro and build crates

## Description

**Size:** M
**Files:** model/umpire/catalog/syntax/**, model/umpire/catalog/build/**, model/temporal/annotate/syntax/**, model/umpire/catalog/tests/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml
**Touches:** [model/umpire/catalog/syntax/**, model/umpire/catalog/build/**, model/temporal/annotate/syntax/**, model/umpire/catalog/tests/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Replace hand-written code generation and parsing in the macro and build crates.

- `heck` for case conversion: `model/umpire/catalog/build/src/wire_schema.rs` (~456, ~473), `model/umpire/catalog/syntax/src/derive.rs` (~632), `model/temporal/annotate/syntax/src/realization.rs` (~558, ~575). The two existing `camel` copies differ (one filters empty parts); decide from the snapshot diffs and record which behavior is kept.
- `darling` 0.24 (syn 3) for catalog `ApiAttributes` (`model/umpire/catalog/syntax/src/derive.rs` ~76-299: `Parse`, `set_once`, `attributes`, `merge`) and `proposal_handler` (`model/temporal/annotate/syntax/src/realize.rs` ~812-845): flags, `key = value`, rename, duplicate detection, error accumulation. Hand-written parsing stays for source-ordered tag/constraint families, positional `also(uuid, ...)`, ordered `oneof(...)`, `rejects(...)`, the rule/mirror split, and the `Declaration` parser (`realize.rs` ~105-155). If darling makes a part worse, leave that part and record why.
- catalog/build Rust generation (`wire_schema.rs`, `services.rs`, ~85 `write!`/`push_str` calls) moves to `quote!` + `prettyplease`, for generated Rust only (protobuf schema text stays string-built).

Re-accept trybuild `.stderr` and expansion snapshots after reviewing the diffs.

### Dependency and refusal controls

`heck` 0.5.0 is already resolved through prost-build; quote and prettyplease 0.3/syn3 are already used. darling is neither declared nor resolved in the audited lock: verify its proposed 0.24/syn3 dependency and R18 admission before selecting features/resolving it. Record new direct/transitive packages and host/build versus guest contexts; do not assume all transitive packages are admitted. Use the existing dependency policy and a current source/evidence admission ledger; identify the exact fn210 R18 note if that acceptance reference requires a metadata update, without rewriting completed task evidence.

Test duplicate attributes, mixed source-ordered families, positional/oneof/rejects forms and malformed offending spans, alongside acronym/underscore/empty-part name cases and generated Rust compile controls. Keep the stated manual grammar boundaries; do not treat `deny_unknown_fields` or darling defaults as a substitute for existing admission checks. Prepare reviewed UI/expansion diffs; final reacceptance follows the join.

## Acceptance
- [ ] No hand-written snake/camel converter remains; generated names match prost-build's.
- [ ] `ApiAttributes` and `proposal_handler` parse through darling; malformed attributes still refuse at the offending span.
- [ ] catalog/build emits generated Rust through `quote` + `prettyplease`.
- [ ] fn-210's R18 note records heck and darling; Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --locked
cd model && mise exec -- cargo test -p umpire-catalog --test ui --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
