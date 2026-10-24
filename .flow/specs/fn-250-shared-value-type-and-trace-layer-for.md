# Shared value, type and trace layer for the TLA+ back ends (model)

## Conversation Evidence

> user (2026-10-24): "are there any other major low hanging fruits we havne't considered yet to reduce code size and complexity? consider conlidation/duplication, too. and better abstractions."
> assistant: a read-only consolidation review (jscpd: ~2.5% textual duplication in the back ends; the real duplication is parallel type and value hierarchies re-lowered per crate), items A1, A2, A5, A8, A9.
> user (2026-10-24): "yes file"

## Goal & Context

The TLA+, TLC and Apalache back ends and the fixture runner each re-derive the same things from the IR: a wire `Type` that mirrors `ir::Type` (lowered five times), a `Schema` built from a `Module` (five times), a `Literal` that mirrors `umpire_dsl::Value` field for field (converted and type-checked in about eight places), the observer variable names as string literals, and a step-by-step trace replay written once for TLC and once for Apalache. This spec replaces them with one owned layer so each back end keeps only what is genuinely its own (emission syntax, tool I/O, stutter/tick encoding). Estimated 600–800 lines removed. Byte-for-byte compatibility of goldens is not required (owner, 2026-10-24); goldens are re-accepted after review.

## Compatibility policy for remaining work

Owner, 2026-10-24: backward compatibility and equality with old output bytes are not required. Internal/wire formats, generated code, diagnostics, paths and snapshots may change where needed for this consolidation; update all affected in-tree producers/readers and regenerate current guests and goldens together. No legacy decoder, dual format or old-guest support is required. Validate the new native contract, semantic correctness, typed replay, reproducibility and safety. Existing artifact SHA/checksum integrity, strict type/domain checks and actual native/SDK proof obligations remain.

## Architecture & Data Models

- **Literal in the IR (A2).** `Literal` moves from `umpire-tla` (`backends/tla/src/lib.rs:97`) into `umpire-ir`; umpire-dsl provides `From<Value> for Literal`. One deep function `Literal::admit(&self, &ir::Type) -> Result<ir::Value, Mismatch>` replaces `native/src/interpret_values.rs:9` `import`, `tla/src/bind.rs:258` `literal` and `:175` `to_literal`, `tlc/src/module.rs:116` `binding_value`, `apalache/src/replay.rs:667` `typed_literal` and `:742` `literal`, `dev/testing/src/backend.rs:276` `admits`, and the dsl→Literal copies in `dev/testing/src/bound.rs:243`, `apalache/src/replay.rs:776`, `apalache/tests/replay.rs:37`, `tla/tests/world_fairness_binding.rs:69`. `Binding::of(world, starts, catalog)` and `Concrete::of::<M>(action)` replace three `concrete::<M>()` copies (`dev/testing/src/bound.rs:188`, `apalache/src/replay.rs:766`, `apalache/tests/replay.rs:50`).
- **One type system (A1).** The wire `Type` in `tla/src/values.rs:11-174` is removed; `Schema` is keyed by `ir::Type`, with one `accepts(&ir::Type, &wire::Value)` covering bounded ranges and optional codes. The emitter owns `Module::schema()` and `Module::safety_schema()` (it knows the observers), replacing `tlc/src/module.rs:7,42-97`, `apalache/src/replay.rs:591,604,651` and the test copies in `tla/tests/{emission.rs:22,world.rs:365,optional_ticks.rs:250}`. One Snowcat type printer (`tla/src/snowcat.rs`) serves ITF `varTypes` (`apalache/src/decode.rs:206` goes). Wire `Set`/`Record` and their decoders (`tlc/src/decode.rs`, `apalache/src/decode.rs` `#set`/records, `replay.rs:816-830` `default_value`) are removed while no Model produces them; fn-245 reintroduces sets and maps on `ir::Type` directly.
- **Observers and replay (A5).** A `umpire_tla::observers` module owns the observer names (`__umpire_action`, `__umpire_choice`, `__umpire_input`, `__umpire_outcome`, `__umpire_facts`, …), today retyped in `apalache/src/replay.rs:15-23`, `dev/testing/src/backend.rs:186-290`, `tla/src/safety.rs:11-13`, `tla/src/emit.rs:13,540`, `tla/src/names.rs`, `dsl/macros/src/action.rs`, with `EdgeObservers::decode(&TraceState)`. One step-replay core (catalog entry by input ordinal with `Concrete` identity check, choice selection, `checked_step`, state/outcome/facts comparison) serves TLC (`dev/testing/src/backend.rs:157-330`) and Apalache (`apalache/src/replay.rs:205-510`); each keeps only its stutter/tick handling.
- **TLA+ lowering table (A8).** The `Op`→symbol table, leaf rendering and Div/Rem handling shared by `tla/src/emit.rs:779-878` `render` and `tla/src/safety/value.rs:46-195`. Unifying the two lowerings into one `(value, valid)` form is out of scope.
- **Pinned jar admission (A9).** `PinnedJar::admit(root, cache, jar, sha)` (cache containment, digest, pinned Java, version probe) in the process crate replaces `tlc/src/runner.rs:104-142` and `apalache/src/lib.rs:124-160`; one `identifier()` replaces `tlc/src/module.rs` and `apalache/src/lib.rs:270`. Its digest is in-process `sha2`, replacing both runners' `shasum` subprocesses (moved here from fn-248.6 so the change lands once).

## API Contracts

- `umpire-ir`: `Literal`, `Literal::admit`, `Mismatch`.
- `umpire-tla`: `Module::schema()`, `Module::safety_schema()`, `observers` module with `EdgeObservers::decode`, a replay-core entry point used by `umpire-testing` and `umpire-apalache`.
- `umpire-backend-process`: `PinnedJar::admit`.
- No Model surface or emitted TLA+ semantics change. Shared protocol representations may change where needed for the planned value/type/trace consolidation, with coordinated in-tree reader and guest regeneration; preserve the semantic contract without supporting old formats.

## Edge Cases & Constraints

- Type mismatches, out-of-range bounded values and unknown optional codes refuse as before (wording may change), now from one place.
- TLC and Apalache retain useful per-step divergence reporting. Wording may change; update tests to assert the divergence and its typed step/field context rather than an old diagnostic string.
- Removing wire Set/Record must not break fn-245: that spec adds set and map types to `ir::Type`, so `accepts` and the decoders gain them there.

## Acceptance Criteria

- **R1:** One `Literal` with `admit` replaces the listed conversion and admission copies; `Binding::of` and `Concrete::of` replace the `concrete` copies. Errors: mismatches refuse as before.
- **R2:** No wire `Type` remains; `Schema` is keyed by `ir::Type` and produced by the emitter; one Snowcat printer; no wire Set/Record code. Errors: a value outside its declared type refuses decoding.
- **R3:** Observer names exist once; one replay core serves TLC and Apalache. Errors: divergences are reported per step as before.
- **R4:** The two TLA+ lowerings share one operator table and leaf renderer.
- **R5:** TLC and Apalache admit their jars through `PinnedJar::admit` with an in-process `sha2` digest; one `identifier()`; no `shasum` subprocess. Errors: wrong digest, wrong Java or cache escape refuse before use.
- **R6:** `make model-check` passes (TLC and Apalache modes), goldens re-accepted after review. No error surface beyond the gate.

## Boundaries

- No new back end, no Set/Map support (fn-245), no BTOR2 (fn-240).
- No unification of the two TLA+ lowerings into one.
- No process supervisor changes beyond adding `PinnedJar` (fn-247 owns the supervisor).

## Decision Context

2026-10-24, owner: filed from the consolidation review. Placed after fn-248 (which changes the ITF envelope in the same back end). Recommended before fn-245, so its set/map values are added once to `Literal::admit` and `Schema` instead of in every copy; the owner orders the two.

2026-10-24, owner (throughput ordering): consolidation lands before the feature work that would otherwise repeat each change in every copy. Order: fn-247 → fn-248 → fn-250, fn-251 and fn-252 as parallel lanes → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245; fn-240 and fn-211 stay deferred behind fn-250. fn-245, fn-240 and fn-211 depend on this spec; it runs beside fn-251 and fn-252 and does not block fn-246.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | One `Literal` with `admit` replaces the listed conversion and admission copies; `Binding::of` and `Concrete::of` replace the `concrete` copies. Errors: mismatches refuse as before. | fn-250-shared-value-type-and-trace-layer-for.1 | — |
| R2 | No wire `Type` remains; `Schema` is keyed by `ir::Type` and produced by the emitter; one Snowcat printer; no wire Set/Record code. Errors: a value outside its declared type refuses decoding. | fn-250-shared-value-type-and-trace-layer-for.2 | — |
| R3 | Observer names exist once; one replay core serves TLC and Apalache. Errors: divergences are reported per step as before. | fn-250-shared-value-type-and-trace-layer-for.3 | — |
| R4 | The two TLA+ lowerings share one operator table and leaf renderer. | fn-250-shared-value-type-and-trace-layer-for.4 | — |
| R5 | TLC and Apalache admit their jars through `PinnedJar::admit` with an in-process `sha2` digest; one `identifier()`; no `shasum` subprocess. Errors: wrong digest, wrong Java or cache escape refuse before use. | fn-250-shared-value-type-and-trace-layer-for.5 | — |
| R6 | `make model-check` passes (TLC and Apalache modes), goldens re-accepted after review. No error surface beyond the gate. | fn-250-shared-value-type-and-trace-layer-for.1, fn-250-shared-value-type-and-trace-layer-for.2, fn-250-shared-value-type-and-trace-layer-for.3, fn-250-shared-value-type-and-trace-layer-for.4, fn-250-shared-value-type-and-trace-layer-for.5 | — |
