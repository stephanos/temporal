# fn-240.1 dependency survey and crate cut table (2026-10-23)

Task: `fn-240-gomad-context-sized-rust-crates-with.1` (R1). Base: `gomad` 520c28ef12. This
survey changes no code. Nothing was built; the only tooling run was `cargo metadata`.

## Summary

- **Size.** The spec's figures (35k + 57k lines) are stale. Today the workspace holds:

  | Crate | Production lines | Test lines | Integration tests |
  | --- | ---: | ---: | --- |
  | `gomad` | 73.7k | 36.7k | 1.5k under `tests/` |
  | `gomad-wasmhost` | 43.3k | 14.0k | 13.0k under `tests/` |

  `wasmhost` already sits in the `rust/` workspace, and the Makefile builds the helper with cargo. The spec's
  "keep the Go cgo/build.rs wiring unchanged" condition no longer applies; the
  `wasm-test-helper`/`publish-executable.sh` path does.
- **Neither crate has a layered module graph.** Counting every gated module, all of
  `wasmhost` except `clock` forms one strongly connected component: 26 modules,
  36.9k production lines. In the default build, with feature-gated and test-only modules
  removed, 19 modules and 29.0k lines remain in it. In `gomad`, 23 modules (67.2k lines) form one
  component; in the default build it is 13 modules and 63.8k lines. Beyond pure
  `use`-path edits, every cut needs the relocations listed below.
- **The owner's 8–12 target fits** with 11 crates: three on the helper side, eight on the host
  side. Most crates hold 4–17k production lines. `gomad-measure` is the outlier at about 19.7k.
- **Both approved first extractions carry blockers the spec did not expect.**
  `gomad-evidence` has three upward edges, so its early proof ("use-path edits only")
  will fail as written. `wasm-sim` needs part of `linked_simulation` and two
  feature-gated hooks. Each blocker is small; details below.

## Method

`tools/gomad/rust/{gomad,wasmhost}/src` was grepped for `crate::<module>` and
`gomad_wasmhost::<module>` paths, including grouped `use crate::{…}` and inline paths.
Each match is attributed to the referencing file's top-level module. A reference counts as
test-only when it sits in a `*_tests.rs`/`tests.rs` file or inside a `#[cfg(test)] mod` block.
`wasmtime::` and `wasmtime_wasi::` references are counted the same way.

Limits:
- Grouped imports nested more than one level can attribute a submodule name; those were
  dropped by keeping only top-level module names.
- `#[cfg(feature)]`-gated references count as production; the Gate column marks the
  modules that exist only under a feature.
- `cargo-modules` is not installed.

Strongly connected components were computed with Tarjan's algorithm over production edges. The
script ran from the session scratchpad and is reproduced in Appendix B.

## Proposed crates (8–12 target)

Production/test lines are current module sums. The listed relocations are the moves needed
to make the crate graph acyclic. Dependency order runs bottom to top.

| # | Crate | Contents (top-level modules) | Prod / test lines | Depends on | Relocations required |
| --- | --- | --- | --- | --- | --- |
| 1 | `wasm-sim` | `clock`, `network`, `volume`, `event_order` (feature), `stress_jitter` (feature); later `world` (see note) | 8.2k / 1.7k; with `world` 11.2k / 2.9k; plus test-only corpora `clock_conformance`, `world_conformance`, `network_corpus` (1.0k) | serde, sha2 | the pure controller half of `linked_simulation` (`Dimension`, `Override`, `Plan`, `Decision`, `Divergence`, `Controller`, `domain_hash`, `Shared`; lines 111–496, about 385 lines); a diagnostic owner-span hook (`volume` → `diagnostic` ×26, diagnostic-profile only); a determinism-fixture owner hook (`event_order` → `model` ×2) |
| 2 | `wasm-model` | `abi`, `protocol`, `transcript`, `model`, `local_model`, `namespace`, `storage_transport`, `storage_evidence_bounds`, `metrics`, `diagnostic` (feature), `profile_inventory`, `module_metadata`, `determinism_fixture` (feature), `conformance` (feature) | 14.3k / 4.9k | wasm-sim, wasmtime | invert `abi`→`engine` (1), `protocol`→`guest_profile` (1), `determinism_fixture`→`linked_conditions` (2), `storage_transport`→`linked_simulation` (4; resolved by the sim move), `model`→`linked` (2), `storage_evidence_bounds`→`linked` (1) |
| 3 | `gomad-wasmhost` (residual) | `engine`, `cache`, `guest_profile`, `linked_conditions`, `linked_fixture`, `cluster`, `linked`, `linked_*`, the execute half of `linked_simulation`, `main`; test-only `model_preservation`, `storage_preservation` | 16.9k / 5.6k | wasm-model, wasm-sim | none. `engine`↔`cluster` (`engine`→`cluster` ×7, `cluster`→`engine` ×13) stay together on purpose |
| 4 | `gomad-evidence` | `evidence` plus `campaign/origin{.rs,/cache.rs}`, `campaign/exploration/site_scope.rs`, `minimize/source.rs` and the `choice::hex` helper | about 6.0k / 2.1k | gomad-wasmhost (`linked::ChoiceIdentity`, `inspect_choices`, `NativeOutputAccounting`, `wasmparser` re-export) | the three back-edges `evidence/store/open.rs:492-493` (`minimize::source::read`, `campaign::origin::authenticate`) and `evidence/search_context.rs` (`site_scope`, `choice::hex`); move the cross-module tests `evidence/store/record_intake_tests.rs` (790) and `write_failure_tests.rs` (294) to `gomad` |
| 5 | `gomad-install` | `install`, `config`, `backstop`, `slots`, `execute`, plus the option parser `cli/explore.rs`+`cli/exploration.rs` (`ExploreOptions`, `parse`, `parse_measurement`) and the strict codec `qualify/spec*` | about 5.5k / 1.4k | gomad-evidence, gomad-wasmhost | `install`→`qualify::spec` (5) and `install`→`campaign::prepare::process` (3); `backstop`→`fixture` hooks (feature) |
| 6 | `gomad-exec` (campaign core) | `campaign/{prepare*, preparation, execution, scenario*, ledger, soft_assertions, launcher, seeds, shards}`, `qualify/workloads`, `qualify/usage` | about 12.4k prod+test (the prod split needs file-level counting in .2) | gomad-install, gomad-evidence | `campaign::exploration::base64` (52 uses from `replay`, 8 from `qualify`, 2 from `install`) and `qualify::cells::base64` move into one evidence codec helper; `fixture::execution_started` and `live_backend::*` hooks (feature) |
| 7 | `gomad-replay` | `replay`, `minimize` | 5.2k / 6.0k | gomad-exec | the verb bodies `cli::replay::run`/`retain_observed` and `cli::install::run` move into their library owners (`replay`→`cli` ×4, `campaign`→`cli` ×5, `measurement`→`cli` ×2); `minimize`→`live_backend` hooks (feature) |
| 8 | `gomad-qualify` | `qualify` (minus `spec`, `workloads`, `usage`), `runtime_qualification` | about 6.3k / 2.1k | gomad-replay, gomad-exec | `qualify`→`cli::explore::parse` (1) resolved by crate 5 |
| 9 | `gomad-campaign` | `campaign/{mod, exploration (minus site_scope), counterfactual*, simulation*, choice, inspect, retention, failure_policy}` | about 9.0k prod+test | gomad-qualify, gomad-replay, gomad-exec | `campaign`→`stress::compare_counterfactual` (1) moves into `counterfactual` |
| 10 | `gomad-measure` | `measurement`, `stress`, `profile` | 19.7k / 9.9k | gomad-campaign and below | none known; size is the risk (see .4) |
| 11 | `gomad` (bin and glue) | `cli` (verbs), `shim`, `sweep`, `fixture*`/`live_*` (determinism-fixture), `main`, `lib`, the three bins, examples | about 13.8k / 9.3k | all | owns the fixture/live hook implementations that crates 5–7 call through hooks |

Notes:
- **Owner decisions kept.** At most two extractions come before .4: `gomad-evidence` first, then
  `wasm-sim`. `wasm-sim` starts as the owner's clock/network/volume plus `event_order`, which
  `network` and `volume` call through it under fixture/stress features. `namespace` stays out
  of the sim crate because it imports `wasmtime`. `campaign` sits above `qualify` and `replay`.
  That needs its execution core (crate 6) below them, because today `replay`→`campaign` has
  73 references and `qualify`→`campaign` has 22.
- **`world`** (3.0k lines, no `wasmtime`; its only production edge is `event_order`) fits `wasm-sim`
  and is recommended as the first .4 addition. The spec listed it under `wasm-engine`.
- **Three helper-side crates, not four.** Separating `engine` from `cluster`/`linked` would need
  about 20 inversions, `engine`→`cluster` ×7 among them. Merging them gives a 16.9k-line crate,
  inside the "larger crates" allowance.
- **Only `wasm-sim` escapes `wasmtime`.** Every gomad crate depends on `gomad-wasmhost`
  transitively, so the compile-time gain is parallel rustc over smaller crates, not avoiding
  `wasmtime`. Measure it at .4.

## Seams each cut removes

| Cut | Cross-boundary references removed or made explicit |
| --- | --- |
| `wasm-sim` | 131 downward references from helper crates become public API: wasm-model→sim 33, wasmhost→sim 85, engine→sim 13. The upward references fixed by relocation are `volume`→`linked_simulation` ×2, `volume`→`diagnostic` ×26 (gated), `event_order`→`model` ×2 (gated) |
| `wasm-model` | 244 references become public API (engine→model 94, wasmhost→model 150); 11 upward references inverted |
| `gomad-evidence` | 188 downward references (campaign 62, replay 21, qualify 24, measurement 39, gomad 35, install 7); 5 upward references relocated |
| `gomad-install` / `gomad-exec` | the option parser and campaign core now sit below every verb; this removes the `cli` back-edges (`campaign`→`cli` ×16, `replay`→`cli` ×4, `measurement`→`cli` ×4, `runtime_qualification`→`cli` ×2, `qualify`→`cli` ×1) |
| `gomad-replay` / `gomad-qualify` / `gomad-campaign` | turns the `replay`↔`campaign`↔`qualify` triangle (`replay`→`campaign` 73, `minimize`→`campaign` 14 and `qualify`→`campaign` 22 references upward; `campaign`→`replay`/`qualify` 28 downward) into one direction |

## Cycle and integration risks (with recommendations)

1. **Implementation identity coverage (highest risk).**
   - `wasmhost/build.rs` hashes only the helper crate's own `src/` plus a fixed list for `GOMAD_LINKED_IMPLEMENTATION_SHA256`.
   - `gomad/build.rs` does the same for `GOMAD_HOST_IMPLEMENTATION_SHA256`.
   - After a move, the extracted sources would silently leave the identity closure. That is an identity/determinism contract hole.
   - **Recommendation:** each extraction extends the consumer's build-identity closure to every crate it now depends on. The `measurement-source-inputs.tsv` closure must follow too. Add a regression: change one byte in the extracted crate and require the identity to change. This is mandatory in .2 and .3.
2. **The determinism inventory is path-bound.**
   - `tools/gomad/internal/backends/wasm/determinism/inventory.json` keys host sites as `host.rust.<crate>.src.<module>…`: gomad/evidence has 4 such rows, wasmhost/network 1, wasmhost/world 1.
   - Evidence entries name paths such as `rust:src/event_order.rs` (2), `rust:src/clock_conformance.rs` (2), `rust:src/network_corpus.rs` (1) and `rust:src/diagnostic.rs` (8).
   - The `tests/inventory/host.rs` scanner walks only the two existing crate roots. Code moved out would escape the scan.
   - **Recommendation:** extend the scanner to every workspace crate before moving code, then re-key the affected rows in the same commit. Never drop a row.
3. **Feature propagation and exclusivity.**
   - `wasmhost` has the features `conformance`, `determinism-fixture`, `diagnostic-profile`, `diagnostic-fuel-off`, `guest-profile`, `reference-helper` and `stress-jitter`, plus `compile_error!` exclusivity checks (lib.rs:5–20).
   - The `conformance` build is already a `wasmtime`-module-free subset: `namespace`, `volume`, `transcript`, `local_model`, `conformance`.
   - **Recommendation:**
     - Forward each feature through every new crate.
     - Repeat the exclusivity checks in `wasm-sim` and `wasm-model`.
     - Keep `conformance` in `wasm-model`.
     - Extend the Makefile `lint-rust` matrix, which today covers `-p gomad`/`-p gomad-wasmhost` only, to each new package.
4. **Hooks for gated test fixtures.** `backstop`, `campaign` and `minimize` call `fixture`/`live_backend` functions under determinism-fixture. A lower crate cannot call `gomad`.
   - **Recommendation:** one feature-gated hook registry in `gomad-install`, with function pointers set by `gomad`'s fixture modules. The determinism-fixture role stays release-excluded (the existing release-exclusion lane checks this).
5. **Splitting inside a module is required twice.** One split is `linked_simulation`'s pure controller (for `wasm-sim`). The other is `cli/explore.rs`'s option parser together with the `cli::{install,replay}::run` bodies, used as library functions.
   - The spec's Boundaries forbid cutting inside a module. **Recommendation:** an owner decision allowing these named splits, recorded in .2 and .3. Without it, `wasm-sim` must exclude `volume`, leaving clock/network/world.
6. **The early-proof wording for .2 is wrong for today's code.** "Use-path edits only" cannot pass: `evidence/store/open.rs` calls `minimize::source::read` and `campaign::origin::authenticate`.
   - **Recommendation:** amend .2's early proof to "use-path edits plus the four listed relocations (origin, site_scope, minimize/source, hex)". The first-hour stop rule then holds.
7. **The `pub` surface will grow.** Crossing references currently rely on `pub(crate)`: for example `linked_simulation::Shared`, `qualify::spec::strict` and `campaign::exploration::choice::hex`. Each becomes `pub` with docs under `missing_docs`. Record the count per extraction (R2/R3 target: zero items made `pub` only for tests).
8. **Golden and fixture paths.** Tests resolve `testdata/` through `CARGO_MANIFEST_DIR` (for example `wasmhost/tests/testdata/replay_plan` used by `make validate-replay-plan`). Moved tests must keep their fixture paths or update the Makefile recipes in the same commit.
9. **Frozen-candidate identities.** Any extraction rotates host and helper identities (MILESTONES rulings 7–10). Start .2 only after the post-freeze evidence windows that bind the frozen candidate.
10. **Branch conflicts.** Every extraction is a large rename diff. Before claiming .2, list open branches touching `gomad/src/evidence`, `campaign/origin*` and `minimize/source.rs`; before .3, those touching `wasmhost/src/{clock,network,volume,event_order,linked_simulation}.rs`.

## Proposed task amendments (for the conductor)

- **.2:** add the four relocations, the cross-module test move, identity-closure extension, the inventory scanner/re-key, `lint-rust` matrix entries and a `pub` count. Size M.
- **.3:** add the `linked_simulation` controller split (needs an owner ruling), the two feature hooks, feature forwarding and exclusivity, identity closure and inventory re-key. Make `world` an explicit .4 candidate. Size M.
- **.4:** decide crates 2 and 5–11 from the measurements, including whether `gomad-measure` (19.7k) needs a twelfth crate.

## Appendix A: module table and edges

Production edges count references to other top-level modules; `wasmhost::X` marks
`gomad`→helper edges. Gate shows feature- or test-only modules (from `lib.rs` `cfg`).

### wasmhost

| Module | Prod lines | Test lines | Files | wasmtime refs | Gate | Production edges (refs) | Test-only edges |
| --- | ---: | ---: | ---: | --- | --- | --- | --- |
| `cluster` | 7868 | 2089 | 20 | 7 |  | event_order (35), diagnostic (25), model (24), metrics (17), network (16), engine (13), protocol (13), linked_simulation (10), namespace (7), storage_transport (7), linked_network (4), world (4), stress_jitter (3), determinism_fixture (2), clock (1) | abi, linked_cluster, volume |
| `model` | 3820 | 2187 | 10 | 7 |  | transcript (22), diagnostic (12), event_order (12), namespace (9), determinism_fixture (4), protocol (4), local_model (3), world (3), linked (2), abi (1), storage_transport (1) | engine, linked_inspection, metrics |
| `volume` | 3865 | 1346 | 8 |  |  | diagnostic (26), linked_simulation (2), event_order (1) | linked, linked_storage_inspection |
| `linked_cluster` | 1835 | 2539 | 4 | 1 |  | world (9), cluster (5), metrics (5), diagnostic (4), linked_simulation (3), engine (1), linked (1), linked_conditions (1), model (1), namespace (1), protocol (1) | storage_transport, volume |
| `world` | 2981 | 1199 | 16 |  |  | event_order (1) |  |
| `network` | 3166 | 246 | 7 |  |  | event_order (1) |  |
| `namespace` | 2572 | 562 | 4 | 10 |  | event_order (11), determinism_fixture (1), protocol (1), transcript (1), volume (1) |  |
| `engine` | 2620 | 0 | 1 | 6 |  | model (25), diagnostic (22), namespace (12), metrics (9), cluster (7), event_order (6), linked_conditions (3), linked_fixture (3), storage_transport (3), abi (1), cache (1), guest_profile (1), linked_network (1), protocol (1), stress_jitter (1), transcript (1) |  |
| `storage_transport` | 1334 | 653 | 5 | 1 |  | diagnostic (10), namespace (6), linked_simulation (4), protocol (1), volume (1) |  |
| `conformance` | 1540 | 433 | 1 |  | test or conformance | namespace (3), local_model (1), transcript (1), volume (1) |  |
| `diagnostic` | 1138 | 443 | 1 |  | diagnostic-profile | abi (3), metrics (2), profile_inventory (2) |  |
| `abi` | 1466 | 28 | 1 | 3 |  | diagnostic (13), model (3), namespace (3), transcript (2), engine (1), protocol (1) |  |
| `linked_inspection` | 1277 | 0 | 1 | 1 |  | model (4), world (1) |  |
| `linked` | 874 | 267 | 2 | 23 (+3 test) |  | metrics (5), diagnostic (4), model (4), storage_evidence_bounds (4), linked_network (3), guest_profile (2), namespace (2), protocol (2), cache (1), cluster (1), determinism_fixture (1), engine (1), linked_cancellation (1), linked_cluster (1), linked_conditions (1), linked_storage (1), module_metadata (1), stress_jitter (1) | abi |
| `cache` | 436 | 417 | 1 | 2 (+1 test) |  | diagnostic (10), engine (2), metrics (1) | linked |
| `clock` | 768 | 63 | 1 |  |  |  |  |
| `protocol` | 651 | 47 | 1 | 1 |  | diagnostic (6), namespace (6), model (5), abi (3), guest_profile (1), metrics (1), storage_transport (1) |  |
| `linked_storage_inspection` | 372 | 250 | 3 |  |  | volume (11), storage_transport (3), linked (2) | namespace |
| `linked_simulation` | 606 | 0 | 1 | 6 |  | linked_cluster (8), protocol (2) |  |
| `model_preservation` | 594 | 0 | 1 |  | test | cluster (4), abi (1), model (1), namespace (1), protocol (1), transcript (1) |  |
| `event_order` | 374 | 95 | 1 |  | determinism-fixture or stress-jitter | model (2) |  |
| `linked_conditions` | 294 | 173 | 1 | 1 |  | event_order (6), determinism_fixture (5), diagnostic (3), linked (2), engine (1), model (1) |  |
| `network_corpus` | 451 | 0 | 1 |  | test | network (1) |  |
| `linked_inspection_predicate_tests` | 0 | 378 | 1 |  | test |  | linked_cluster, model |
| `determinism_fixture` | 179 | 167 | 3 |  | determinism-fixture | linked_conditions (2), model (2), event_order (1) |  |
| `transcript` | 259 | 35 | 1 | 1 |  | abi (2) |  |
| `clock_conformance` | 287 | 0 | 1 |  | test | clock (1) |  |
| `world_conformance` | 285 | 0 | 1 |  | test | world (1) |  |
| `metrics` | 137 | 110 | 1 |  |  |  |  |
| `guest_profile` | 225 | 0 | 2 | 2 | guest-profile |  |  |
| `storage_evidence_bounds_tests` | 0 | 217 | 1 |  | test |  | linked, volume |
| `storage_preservation` | 205 | 0 | 1 |  | test | namespace (2), storage_transport (2), volume (2) |  |
| `linked_network` | 141 | 0 | 1 | 1 |  | cluster (3), namespace (3), network (3), abi (1), engine (1), linked_cluster (1), model (1), world (1) |  |
| `linked_cancellation` | 98 | 39 | 1 | 1 (+1 test) |  | cluster (1), linked (1), protocol (1) |  |
| `local_model` | 121 | 0 | 1 |  |  | namespace (1), transcript (1) |  |
| `lib` | 121 | 0 | 1 | 2 |  |  |  |
| `storage_evidence_bounds` | 106 | 0 | 1 | 1 |  | linked (1), model (1), storage_transport (1), volume (1) |  |
| `module_metadata` | 101 | 0 | 1 | 2 |  |  |  |
| `stress_jitter` | 42 | 44 | 1 |  | stress-jitter |  | engine, linked_conditions |
| `linked_storage` | 70 | 0 | 1 | 1 |  | storage_transport (3), engine (1), linked_simulation (1), namespace (1), protocol (1) |  |
| `profile_inventory` | 32 | 0 | 1 |  |  | abi (1) |  |
| `linked_fixture` | 25 | 0 | 1 | 1 | determinism-fixture | engine (1) |  |
| `main` | 9 | 0 | 1 |  |  |  |  |

### gomad

| Module | Prod lines | Test lines | Files | wasmtime refs | Gate | Production edges (refs) | Test-only edges |
| --- | ---: | ---: | ---: | --- | --- | --- | --- |
| `measurement` | 16356 | 8448 | 56 |  |  | wasmhost::linked (43), qualify (38), evidence (28), campaign (16), install (15), stress (12), profile (7), cli (4), backstop (2), wasmhost::linked_cluster (2), wasmhost::profile_inventory (1) | replay |
| `campaign` | 17446 | 5122 | 48 |  |  | evidence (62), wasmhost::linked (30), cli (16), qualify (14), install (13), wasmhost::linked_cluster (12), replay (9), backstop (7), fixture (3), live_backend (3), execute (2), slots (2), installed_lifecycle_tests (1), runtime_qualification (1), stress (1), wasmhost::linked_inspection (1), wasmhost::linked_simulation (1) | measurement |
| `qualify` | 9054 | 2031 | 28 |  |  | evidence (22), campaign (18), wasmhost::linked (15), replay (9), install (8), wasmhost::linked_cluster (2), cli (1), wasmhost::linked_inspection (1) |  |
| `sweep` | 4016 | 4649 | 11 |  |  | campaign (8), install (3), wasmhost::linked (3), replay (2), backstop (1), evidence (1) |  |
| `replay` | 3278 | 4962 | 16 |  |  | qualify (12), evidence (10), campaign (8), wasmhost::linked (6), wasmhost::linked_inspection (5), cli (4), backstop (3), execute (2), fixture (1), wasmhost::linked_cluster (1), wasmhost::linked_storage_inspection (1) | fixture_cluster, fixture_execute, installed_lifecycle_tests |
| `evidence` | 4762 | 3206 | 22 |  |  | wasmhost::linked (4), campaign (3), minimize (1) | cli, fixture, install |
| `cli` | 4219 | 1853 | 22 |  |  | campaign (43), evidence (20), install (11), replay (11), qualify (6), minimize (5), backstop (3), execute (3), config (1), fixture (1), stress (1), wasmhost::linked (1), wasmhost::linked_storage_inspection (1) | installed_lifecycle_tests |
| `stress` | 2842 | 1447 | 8 |  |  | qualify (20), replay (16), evidence (10), wasmhost::linked (9), measurement (8), campaign (2), wasmhost::linked_inspection (2), backstop (1), cli (1) |  |
| `install` | 2863 | 973 | 11 |  |  | qualify (6), campaign (3), config (3), evidence (3), wasmhost::linked (3), wasmhost::linked_cluster (2) | shim |
| `shim` | 1900 | 1643 | 5 |  |  | install (5), cli (4), config (4), campaign (3), backstop (2), evidence (2) | wasmhost::linked |
| `minimize` | 1908 | 1058 | 10 |  |  | campaign (13), evidence (11), wasmhost::linked (5), live_backend (3) | cli |
| `installed_lifecycle_tests` | 0 | 875 | 1 |  | test |  | campaign, cli, evidence, install, qualify, replay, wasmhost::linked, wasmhost::linked_cluster |
| `fixture` | 640 | 0 | 1 |  |  | fixture_execute (5), campaign (4), cli (4), evidence (2), fixture_cluster (2), fixture_replay (2), qualify (2), fixture_role (1), fixture_storage (1), live_broker (1), live_cluster (1), live_runtime (1), minimize (1), replay (1), slots (1) |  |
| `fixture_cluster` | 586 | 0 | 1 |  | determinism-fixture | fixture (3), evidence (2), fixture_execute (2), replay (2), backstop (1), cli (1), wasmhost::linked (1) |  |
| `fixture_execute` | 560 | 0 | 1 |  | determinism-fixture | fixture (4), evidence (2), replay (1), slots (1), wasmhost::linked (1) |  |
| `config` | 346 | 131 | 3 |  |  | install (1) | campaign |
| `profile` | 467 | 0 | 3 |  |  | measurement (2), wasmhost::diagnostic (2), campaign (1), wasmhost::profile_inventory (1) |  |
| `fixture_replay` | 388 | 0 | 1 |  | determinism-fixture | evidence (1), fixture (1), wasmhost::linked (1) |  |
| `live_cluster` | 318 | 0 | 1 |  | determinism-fixture | fixture_cluster (3), evidence (2), fixture (2), backstop (1), wasmhost::linked (1) |  |
| `runtime_qualification` | 221 | 40 | 1 |  |  | wasmhost::linked (7), campaign (4), cli (2), evidence (2), qualify (1) |  |
| `main` | 219 | 0 | 1 |  |  |  |  |
| `live_runtime` | 217 | 0 | 1 |  | determinism-fixture | evidence (1), execute (1), wasmhost::linked (1), wasmhost::linked_inspection (1) |  |
| `fixture_role` | 181 | 0 | 1 |  | determinism-fixture | evidence (1), fixture (1), wasmhost::linked (1) |  |
| `fixture_storage` | 181 | 0 | 1 |  | determinism-fixture | evidence (1), fixture (1), replay (1), wasmhost::linked (1) |  |
| `backstop` | 161 | 0 | 1 |  |  | fixture (3) |  |
| `fixture_cluster_lifecycle_tests` | 0 | 159 | 1 |  |  |  | campaign, cli, evidence |
| `live_broker` | 124 | 0 | 1 |  | determinism-fixture | replay (2), backstop (1), evidence (1), wasmhost::linked (1) |  |
| `slots` | 91 | 24 | 1 |  |  |  |  |
| `bin:measure` | 92 | 0 | 1 |  |  |  |  |
| `fixture_cluster_metrics_tests` | 0 | 87 | 1 |  |  |  | campaign |
| `live_backend` | 78 | 0 | 1 |  | determinism-fixture | wasmhost::linked (1) |  |
| `execute` | 66 | 0 | 1 |  |  | evidence (4), wasmhost::linked (2), backstop (1) |  |
| `lib` | 45 | 0 | 1 |  |  |  |  |
| `bin:profile` | 28 | 0 | 1 |  |  |  |  |
| `bin:gomad-wasm-sweep` | 3 | 0 | 1 |  |  |  |  |

## Appendix B: survey script

Run as `python3 -I survey.py tools/gomad/rust graph.json`. Strongly connected components then
come from a Tarjan pass over production edges to known top-level modules.

```python
import re, sys, json, os
from pathlib import Path
from collections import defaultdict
ROOT = Path(sys.argv[1])
CRATES = {'gomad': ROOT/'gomad/src', 'wasmhost': ROOT/'wasmhost/src'}
EXT = {'gomad': 'gomad_wasmhost', 'wasmhost': None}
TEST = re.compile(r'(^|/)(tests?|.*_tests?)\.rs$|/tests/')
def topmod(crate, path):
    rel = path.relative_to(CRATES[crate]).as_posix()
    if rel.startswith('bin/'): return 'bin:' + rel.split('/')[1].removesuffix('.rs')
    first = rel.split('/')[0].removesuffix('.rs')
    return first
def strip_comments(text):
    text = re.sub(r'/\*.*?\*/', '', text, flags=re.S)
    return re.sub(r'//[^\n]*', '', text)
def cfg_test_spans(text):
    # crude: lines after '#[cfg(test)]' followed by 'mod x {' until matching brace
    spans=[]; i=0
    for m in re.finditer(r'#\[cfg\(test\)\]\s*(?:#\[[^\]]*\]\s*)*mod\s+\w+\s*\{', text):
        depth=1; j=m.end()
        while j < len(text) and depth:
            if text[j]=='{': depth+=1
            elif text[j]=='}': depth-=1
            j+=1
        spans.append((m.start(), j))
    return spans
lines = defaultdict(lambda: [0,0])  # prod, test
edges = defaultdict(lambda: defaultdict(lambda: [0,0]))  # (crate,mod) -> (crate,target) -> [prod,test]
wasm = defaultdict(lambda: [0,0])
files = defaultdict(int)
for crate, src in CRATES.items():
    mods = {p.name.removesuffix('.rs') for p in src.iterdir()}
    for path in sorted(src.rglob('*.rs')):
        raw = path.read_text()
        m = topmod(crate, path)
        rel = path.relative_to(src).as_posix()
        whole_test = bool(TEST.search(rel)) 
        text = strip_comments(raw)
        spans = cfg_test_spans(text)
        n = raw.count('\n')
        if whole_test: lines[(crate,m)][1]+=n
        else:
            tl = sum(text[a:b].count('\n') for a,b in spans)
            lines[(crate,m)][0]+=n-tl; lines[(crate,m)][1]+=tl
        files[(crate,m)]+=1
        def in_test(pos): return whole_test or any(a<=pos<b for a,b in spans)
        pats = [(crate, re.compile(r'\bcrate::(\w+)'))]
        if crate=='gomad': pats.append(('wasmhost', re.compile(r'\bgomad_wasmhost::(\w+)')))
        # also grouped use crate::{a, b::c}
        for tgt_crate, pat in pats:
            for mm in pat.finditer(text):
                t = mm.group(1)
                if tgt_crate==crate and t==m: continue
                edges[(crate,m)][(tgt_crate,t)][1 if in_test(mm.start()) else 0]+=1
        for mm in re.finditer(r'\b(?:crate|gomad_wasmhost)::\{([^}]*)\}', text):
            prefix = 'gomad_wasmhost' if text[mm.start():mm.start()+14]=='gomad_wasmhost' else 'crate'
            tc = 'wasmhost' if prefix=='gomad_wasmhost' else crate
            for item in re.split(r',', mm.group(1)):
                item=item.strip().split('::')[0].strip()
                if item and item!='self' and re.match(r'\w+$',item) and not (tc==crate and item==m):
                    edges[(crate,m)][(tc,item)][1 if in_test(mm.start()) else 0]+=1
        for mm in re.finditer(r'\bwasmtime(?:_wasi)?::', text):
            wasm[(crate,m)][1 if in_test(mm.start()) else 0]+=1
out = {'lines': {f'{c}/{m}': v for (c,m),v in lines.items()}, 'files': {f'{c}/{m}':v for (c,m),v in files.items()},
       'wasmtime': {f'{c}/{m}': v for (c,m),v in wasm.items()},
       'edges': {f'{c}/{m}': {f'{tc}/{t}': v for (tc,t),v in d.items()} for (c,m),d in edges.items()}}
json.dump(out, open(sys.argv[2],'w'), indent=1, sort_keys=True)
```
