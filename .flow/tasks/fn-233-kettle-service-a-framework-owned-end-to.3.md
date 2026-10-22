---
satisfies: [R3, R5]
---
# fn-233-kettle-service-a-framework-owned-end-to.3 derive(Catalog) Kettle catalog, config.rs, Annotate in the live run, coverage and gap goldens

## Description
Write `api.rs` as the second `#[derive(Catalog)]` catalog (fn-217.6 surface): mirror structs for every in-scope Kettle message, kinds as field types, `#[rule]`/`#[mirror]` namespaces with presence rules at item level, `config.rs` over the generic scoping for the static and per-kettle limits, the hand-written exhaustive `match` over the generated `KettleServiceMethod`, and at least one `Ignored` field with a reason. Wire Annotate into the live run so every wire message is validated and filled; produce `coverage.snap` and `gaps.txt` with the generic probe.

**Size:** L
**Touches:** `model/umpire/reference/kettle/**`, `model/umpire/hosts/go/kettle*_test.go`, `model/umpire/hosts/go/reference/kettle/*.go` (annotation/probe integration controls), `model/umpire/hosts/go/reference/kettle/v1/**`, `model/scripts/proto-kettle-go.sh` (consumer probe schema generation/freshness), `model/umpire/catalog/**`, `model/umpire/catalog-build/**`, `model/umpire/annotations/**` (generic support only when required, with both consumer regressions), `model/temporal/foundation/dc/mod.rs` (shared resolved limits extraction; policy stays local), `model/temporal/hosts/go/umpire_temporal_guest.wasm` (required freshness after shared-code changes), `model/Cargo.lock`, `model/DESIGN.md`

### Approach
This is the delivered derive's second consumer (R5); record missing generic features and close them with controls for both consumers. Keep `api.rs` readable: `cargo fmt --check` at width 100, proto declaration order.

### Generic second consumer controls

Use the shared derive/Inspect/limits/method/wire-schema machinery delivered by .1; Kettle must compile without any temporal crate. Both Kettle and Temporal are regression consumers. Follow the parent RPC projection and mutant oracle contracts for every annotation/probe. Add focused compile-fail controls for omitted field, enum variant, service method and Model action, plus coverage/gap goldens with Ignored and Unverifiable. R5 records second-consumer portability evidence, not a retroactive first spike. Quick: catalog/syntax/descriptor fixtures and the focused Go healthy test; final: make model-check.
## Acceptance
- Every in-scope message mirrored; a field, enum value, method or Model action added to the Kettle proto or Model fails the build naming it (R3).
- Annotate runs on every wire message of the live run; `coverage.snap` and `gaps.txt` goldens with at least one `Ignored` reason and one `Unverifiable` (R3).
- DESIGN.md records every derive gap found, or states that none was (R5).
- No Temporal name under `umpire/`; `make model-check` green.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

