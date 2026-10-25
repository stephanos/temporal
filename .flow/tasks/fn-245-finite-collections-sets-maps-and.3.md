---
satisfies: [R4]
---
# fn-245-finite-collections-sets-maps-and.3 TLA+ emission and typed TLC decoding for sets, partial maps with the Absent sentinel, quantifiers and list Head/Tail

## Description

After .1, parallel with .2 only on disjoint files; front-consuming tests wait for .2. Extend the TLA emitter and its fn-250 shared value layer: emitter-produced `Module::schema()`/`safety_schema()`, shared `umpire_tla::values::Value`, `accepts`, `validate_type` and typed admitted-value conversion, all keyed by the existing `ir::Type`. Extend existing exhaustive consumers (including Snowcat) with supported forms or explicit unsupported-backend refusal; do not recreate wire Type, private Schema builders or Literal admission copies, and do not claim Apalache collection support.

Emit finite sets with their type invariant, map partial values as total key-domain functions with one private collision-free Absent sentinel, and contains_key/MapGetOr through sentinel-aware lookup. OrdSet predicates iterate canonical element order; map key/value predicates iterate present keys in canonical key order and preserve repeated values. Lower all/any result and arithmetic validity with ordered short-circuit prefixes in normal and safety emission. Use direct logical quantifiers only when predicate evaluation is infallible; do not evaluate a skipped failing predicate or collapse values into a set.

Front evaluates the fallback eagerly and selects `Head` only for nonempty sequences. PopFront uses `IF Len(sequence) = 0 THEN sequence ELSE Tail(sequence)`; the discarded native Option return creates no new IR Option surface. Shared Length retains fn-242 semantics.

In TLC typed decode, admit canonical sets and total functions against the emitter Schema, enforcing the exact finite key domain, exactly one entry per key, typed member ranges and Absent only as the map slot marker; remove absent slots before native projection. Refuse duplicate set elements/map keys, missing or extra function-domain entries, wrong member types and leaked sentinels. Use the fn-250 shared replay core with compiled checked_step callbacks for typed state/outcome/fact comparisons rather than a collection-specific replay implementation. Preserve existing Models' semantic results and review changed goldens.

Touches: model/umpire/backends/tla/**, model/umpire/backends/tlc/**; coordinate shared testing adapters with .4.

## Acceptance

- [ ] Emitter/schema/decode controls cover empty and nonempty sets/maps/sequences, map replacement and sentinel-safe reads; malformed typed values, duplicate/domain violations and leaked sentinels refuse.
- [ ] Normal and safety emission agree with native ordered all/any on same-type map keys/values, repeated values, read-only captures, empty sources and skipped versus reached arithmetic failures.
- [ ] Empty front/pop and eager default failures agree across compiled/interpreted/TLC controls; Head/Tail are guarded.
- [ ] Round trips through shared Schema and native replay cover every collection kind; forged traces are witness-rejected, and all existing collection-free TLC results retain their meaning.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
