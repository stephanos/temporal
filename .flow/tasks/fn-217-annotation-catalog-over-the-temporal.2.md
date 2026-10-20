---
satisfies: [R1, R2]
---
# fn-217-annotation-catalog-over-the-temporal.2 annotate! with four-level exhaustiveness and the descriptor-set build check

## Description
Implement the approved typed annotate! surface under model/temporal/annotate.
Generate executable Catalog factories, checked enum conversion and action/RPC
bindings from the same declarations used by descriptor coverage. No metadata-only
facade, reflection, upstream proto edits or duplicate hand-maintained classifications.
Keep the parser/descriptor checker independent of generated API dependencies so
model/temporal/api/build.rs can call it without a dependency cycle.

Four independent build-failing boundaries:
- Message fields: exhaustive borrowed struct patterns without .., typed field
  accessors/projections and SupportsKind witnesses. Every field is retained,
  including explicitly Ignored fields. Nested messages and prost oneofs must not
  hide a new descriptor field or known oneof variant.
- Enums: TryFrom<i32> first refuses unknown integers, then an exhaustive match
  classifies known variants. Allowed discriminants come from that declaration.
- RPCs: the API build script loads the real proto/api.binpb and checks every
  method of each named service against explicit Action, Observed or
  OutOfScope(reason) declarations. Reject stale/duplicate entries and empty
  reasons. Emit rerun-if-changed for both descriptors and catalog inputs.
- Model actions: generate a callable binding with exhaustive outer Action and
  inner actor matches. A catch-all actor arm cannot conceal new actor variants.
  Each leaf names an RPC, revealing observation or environmental reason.

Use a restricted declaration grammar wherever macro-time admission requires it:
one kind; supported kind/type pairs; named literal constraints; rules and covers;
every request constraint/rule has an expectation; ignored fields cannot appear
in rules; all contradictory roles and kind/tag pairs refuse compilation.
Custom Temporal tags remain types implementing Tag, but compile-time-guaranteed
custom tags need a declarative compatible-kind/consequence definition. Arbitrary
opaque Rust expressions whose dynamic methods determine applicability must not
silently receive a compile-time guarantee. Reuse the generic engine at runtime.

Existing carriers cover strings, bytes, integral counters/enums, time and their
Option/Vec forms. Opaque bool/map/message fields need an intentional typed path
when explicitly ignored: retain actual type, kind and reason, preserve redaction,
and never fake a supported String field or silently omit metadata. Extend the
generic typed adapter narrowly if needed; do not introduce Temporal imports there.

Task .3 owns the actual mini-activity annotations and live fill/validate route.
This task owns the reusable macro/checker, real API build hook and honest controls.

**Touches:** `model/temporal/annotate/**`, `model/temporal/api/build.rs`, `model/temporal/api/Cargo.toml`, `model/temporal/testing/**`, `model/umpire/annotations/**`, `model/Cargo.toml`, `model/Cargo.lock`
**Quick:** focused macro/checker tests, trybuild fixtures and all-target clippy.
**Full gates:** `make model-check`; `make model-time` (paste actual timing).
Use a lane-specific CARGO_TARGET_DIR. Reuse workspace syn/quote/trybuild/prost
facilities; no new external validation engine or dependencies.

## Acceptance
- [ ] Generated executable catalogs and bindings share their declarations with descriptor metadata.
- [ ] trybuild refuses omitted fields, added enum variants, new inner actor variants and new outer actors, naming each omission.
- [ ] Runtime unknown enum integer is refused; known enum additions cannot match a wildcard.
- [ ] The actual API build hook checks named services and tracks catalog changes.
- [ ] In-memory mutations of the real descriptor set independently add a field, enum value and RPC; the same production checker fails naming each added symbol.
- [ ] Admission compile-fail controls cover kind/type mismatch, tag compatibility, contradictory pairs, ignored-rule references, absent request expectations and absent reasons/consequences.
- [ ] Positive controls include typed projections, oneof coverage, custom Temporal tags and explicitly ignored opaque fields; output is snapshot-tested using existing patterns.
- [ ] Full gate, clippy and feedback timing pass without changing independent Models, goldens, ABI or protocol.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
