# Quick consolidation wins in model code, tests and tooling (model)

## Conversation Evidence

> user (2026-10-24): "are there any other major low hanging fruits we havne't considered yet to reduce code size and complexity? consider conlidation/duplication, too. and better abstractions."
> assistant: a read-only consolidation review, quick-win items A3, A4, A6, A7, A10, B5–B7, B9, B11, B12.
> user (2026-10-24): "yes file"

## Goal & Context

Small, independent consolidations found in the same review: a second shortest-path search beside the pattern search, a legacy `Path` representation, committed spike output, test fixtures and harnesses copied across files, a duplicated TLC replay in the activity tests, near-identical scripts and per-guest lint tables, and a test fixture exported from the runtime library. Each task stands alone. Estimated 1,100–1,300 lines of Rust, ~250 lines of shell/Makefile/TOML and ~12.9k lines of generated output removed. Byte-for-byte compatibility is not required (owner, 2026-10-24); goldens are re-accepted after review.

## Architecture & Data Models

- **One search (A3).** `author/suite/src/search.rs:85-290` (`Along`, `Through`, `arrive`, `trace`, `settled`) duplicates `pattern_search.rs:64-296`. `space.rs:254` and `run.rs:123` replace their `is_state_only()` branch with `Graph::new(&explored)?.shortest(&path.alternatives(1)?[0], true)` (`Path::alternatives`, `dsl/src/path.rs:~830`, already turns a state-only path into `[State, Gap, State…]`). Reconcile: the cut-frontier rule (first cut row vs minimum over all), `Unbridged.waypoint` mapped through `ordinals`, and speed (FIFO age domination vs a map of non-dominated labels; measure).
- **Path stores a Pattern (A4).** `dsl/src/path.rs:686-870` drops `first`/`leg`/`more` beside `Option<Pattern>`; `first()`, `legs()`, `len()`, `is_empty()` no longer panic on the pattern case; `claim()` is derived from the pattern; `IntoWorldPath<kind::Legacy>` (`dsl/src/path_input.rs:18`) goes. After A3.
- **Spike output (A6).** Delete committed `design/native-symbolic/out/` and `out-big/` (~12.9k lines, regenerable with `run.sh`) and ignore them; `design/native-symbolic/src/sample_kettle.rs` and `kettle_big.rs` include `dev/fixtures/fixtures/sample_kettle/model.rs` by `#[path]` (or a parameterized variant) instead of copies. The spike stays for fn-240.
- **Native fixtures (A7).** `backends/native/tests/fixtures/{choices,edge_checks,shared_starts,vacuity}/model.rs:1-53` reuse `backends/native/tests/support/mod.rs` (`Value`, `Action`, Display impls, `Tiny`); `edge_checks` and `shared_starts` become `Tiny(Mode::Edges)`/`Tiny(Mode::SharedStarts)`, with new Modes for `choices` and `vacuity` if needed.
- **SpaceBinding (A10).** The two `SpaceBinding` impls in `author/suite/src/space.rs:153-203` differ only in `world()`: one impl generic over a small `WorldSource` trait.
- **trybuild cases (B5).** Of ~90 UI cases in scope, proc-macro admission errors with no rustc stage become rows in table tests: 14 of 23 in `temporal/annotate/tests/ui` into `temporal/annotate/syntax/tests/admission.rs:31` (or its successor after fn-251 removes the facade), `temporal/realization/tests/ui/supplemental_*` (4), duplicate `added_enum_variant`/`omitted_field` between `catalog/tests/ui` and `annotate/tests/ui`, Kettle's overlapping `omitted_field`/`omitted_enum`; most `realization_*` proc-macro errors become `expand()` unit tests. Keep cases that need rustc (`source_scope`, `external_alias`, and any with a type-check stage).
- **Guest test harness (B6).** A `testing` feature on umpire-runtime exposes `GuestExt::{call, judge, reset, propose}` and `ResponseExt::{refusal(code), explanation(), unexplained()}`, replacing copies in runtime `tests/{proposal,stateful,catalog,selected_start}.rs`, realization `tests/{realization,proposal,admission,annotations}.rs` and `kettle/tests/realization.rs`. A `protocol::Method` enum replaces the magic numbers 1–6 (`runtime.rs:270-313`, `live.rs` `prepare_method`).
- **Catalog-syntax tests (B7).** `temporal/annotate/syntax/tests/{admission,descriptor,expansion}.rs` (565 lines) test `umpire_catalog_syntax` through a re-export: move them to `catalog/syntax/tests`. One `assert_additions_refused(bytes, coverage, …)` replaces the four "added field/enum/method is refused" tests (`kettle/tests/descriptor.rs`, `annotate/syntax/tests/descriptor.rs:182`, `temporal/api/tests/services.rs:231,323`, `catalog/build/tests/neutral.rs:110`).
- **Activity TLC replay (B9).** `temporal/feature/activity/proof_checks.rs:206-338` (`state_value`, `matches_state`, `replay_tlc`) uses an exposed `umpire_testing::replay_trace(world, starts, module, trace) -> Result<(), Divergence>` (today private `dev/testing/src/backend.rs:157`); the two state comparisons (`Values` vs Display text) are reconciled on `Values`. If fn-250's replay core lands first, this uses it.
- **Tooling (B11).** One parameterized proto-go script replaces `scripts/proto-go.sh`, `proto-kettle-go.sh`, `proto-temporal-go.sh`; delete unreferenced `scripts/tests/guest-parity.sh`; one Makefile variable for the repeated `UMPIRE_*_WASM` environment block (Makefile ~1294-1295); the workspace sets `unsafe_code = "deny"` so the six guest manifests use `lints.workspace = true` instead of copying the table (crates without unsafe add `#![forbid(unsafe_code)]` or the gate checks that only the audited modules allow it).
- **Runtime fixture (B12).** `umpire/protocol/runtime/src/stock.rs:3` includes `backends/native/tests/fixtures/sample_kettle_trait/kettle.rs` and exports it as `umpire_runtime::Model`; it moves into the stock guest crate.

## API Contracts

- `umpire-runtime` (feature `testing`): `GuestExt`, `ResponseExt`; `protocol::Method`.
- `umpire-testing`: `replay_trace`, `Divergence`; `assert_additions_refused` in the catalog-syntax test support.
- `umpire_runtime::Model` is no longer exported.
- No Model surface, checker semantics or protocol change.

## Edge Cases & Constraints

- A3 must keep every existing search result (shortest witnesses, Unbridged reports); pattern-search goldens are the oracle, and a measured slowdown above 25 percent on the default gate keeps the old path.
- B5 keeps every diagnostic covered: each removed UI case maps to a table row or unit test asserting the same message and span.
- B11's lint change must not let unsafe into crates that forbid it today: the gate proves it.
- fn-247 owns scripts it ports to xtask: B11 touches only scripts fn-247 leaves in Bash.

## Acceptance Criteria

- **R1:** One shortest-path search in the suite; `search.rs`'s parallel types are gone. Errors: unbridged and cut-frontier reports as before.
- **R2:** `Path` stores only a `Pattern`; no panicking accessors.
- **R3:** No committed spike output; no copied Kettle model in the spike.
- **R4:** Native fixtures reuse `Tiny`; `SpaceBinding` has one impl.
- **R5:** UI cases without a rustc stage are table or unit tests; the guest test harness, `protocol::Method` and `assert_additions_refused` exist once; catalog-syntax tests live in their crate.
- **R6:** The activity TLC replay uses `umpire_testing::replay_trace`.
- **R7:** One proto-go script, no unreferenced guest-parity script, one wasm environment variable, guest manifests inherit workspace lints; unsafe stays confined to audited modules.
- **R8:** The runtime library exports no test fixture.
- **R9:** `make model-check` passes, goldens re-accepted after review. No error surface beyond the gate.

## Boundaries

- No checker or Model semantics change; no protocol change.
- No xtask work (fn-247), no back-end layer work (fn-250), no probe engine work (fn-251).

## Decision Context

2026-10-24, owner: filed from the consolidation review. Placed after fn-248. Tasks are independent unless their dependencies say otherwise; B5/B7 follow fn-251's facade removal where they touch `temporal/annotate`.

2026-10-24, owner (throughput ordering): consolidation lands before the feature work that would otherwise repeat each change in every copy. Order: fn-247 → fn-248 → fn-250, fn-251 and fn-252 as parallel lanes → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245; fn-240 and fn-211 stay deferred behind fn-250. fn-246 depends on this spec. Its .5 follows fn-251.6 where it touches `temporal/annotate`; its .6 uses fn-250.3's replay core when that has landed.
