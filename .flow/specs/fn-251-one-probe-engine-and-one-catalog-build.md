# One probe engine and one catalog build path for Kettle and Temporal (model, temporal)

## Conversation Evidence

> user (2026-10-24): "are there any other major low hanging fruits we havne't considered yet to reduce code size and complexity? consider conlidation/duplication, too. and better abstractions."
> assistant: a read-only consolidation review of catalog, runtime, Kettle and Temporal, items B1–B4, B8, B10.
> user (2026-10-24): "yes file"

## Goal & Context

Kettle (the reference system) and Temporal realization each implement the same probe engine, the same descriptor-to-Rust graph builder, the same annotation/probe transport dispatch and their own build-configuration layers; Temporal also hand-writes a `Wire` enum that walks catalogs the `#[derive(Catalog)]` `Inspect` impl already walks, and keeps two facade crates that only re-export umpire-catalog. This spec moves each into one system-neutral place in `model/umpire` so Kettle and Temporal provide only their system-specific parts. Estimated 850–950 lines and two crates removed. Byte-for-byte compatibility is not required (owner, 2026-10-24); goldens are re-accepted after review.

## Architecture & Data Models

- **Descriptor graph (B3).** `umpire_catalog_syntax::proto::Graph::new(bytes, rust_root)` replaces `catalog/build/src/wire_schema.rs:136-330` (`Schema`: collect, kind, optional, map-entry member, oneof) and `temporal/annotate/syntax/src/realization.rs:467-556`. It removes the hardcoded `::umpire_temporal_api` (`realization.rs:168`), so `#[realization]` becomes system-neutral.
- **Build configuration (B10).** `umpire_catalog_build::System { service, prefix, rust_root, request_roots, catalog_root }.generate(descriptor, out_dir)` writes both generated files. It replaces `services::Config` (8 string fields derived from three), the one-call wrappers `temporal/api/{services,wire_schema,codegen}.rs` and `annotate/syntax/src/services.rs`, and the repeated descriptor build in `realization/macros/build.rs:14-40` vs `temporal/api/build.rs`.
- **Generated findings (B2).** catalog/build generates `request_findings(method, bytes)` per service, with hooks for supplemental intent catalogs, replacing the 8-variant `Wire` in `temporal/realization/src/probes.rs:311-551` (`bytes`, `decoded_from`, `findings`, recursive `failure_findings`); Kettle already uses its generated equivalent (`kettle/api.rs:360`). Check whether `mutate`/`mutate_start` (`probes.rs:561-782`) can use the generated `mutate_workflow_request`; record the answer.
- **Probe engine (B1).** `umpire_catalog::generic_probe::Engine<S: ProbeSystem>` with `ProbeSystem { type Method; fn baseline(m, ctx); fn mutate(..); fn findings(m, bytes); fn rpc_path(m); fn prediction(&Violation) }` and `run(op, ..) -> ProbeResponse` owning entries (`method`, `prerequisite`, `case`), `generic::…` ids, materialization (baseline, mutate, Unchanged detection, findings, predictions), classification (observation check, applicability → mutation entries, prediction correlation), the inventory/prepare/classify dispatch and Expectation→ProbeExpectation mapping. Replaces `kettle/probes.rs:19-292` and `temporal/realization/src/probes/generic.rs:21-345` plus `report.rs:144` `observation_result`. Temporal's fingerprint-keyed strict-entry predictions are supported by the trait, not special-cased in the engine.
- **Transport dispatch (B8).** catalog/build generates `annotate_<svc>(method, response, bytes, seed, limits)`, and `AnnotationRuntime::with_probes(rt, probe_fn, annotate_fn)` routes `ProbeRequest` vs `AnnotationRequest`, strips the service prefix (or accepts a bare name), maps `classify` to OutOfScope and packs `AnnotationResponse`. Replaces `kettle/transport.rs:13-61`, `temporal/feature/activity/api_transport.rs:19-58` and `temporal/realization/src/annotations.rs:25-40`.
- **Facade crates (B4).** `temporal/annotate` (`pub use umpire_catalog::*` plus tests) and `temporal/annotate/macros` (a copy of `catalog/macros/src/lib.rs` with hardcoded context/api defaults) are deleted. A crate-root `CatalogContext` alias plus an api re-export covers the defaults (`DeriveCatalog::parse_with` already accepts `#[mirror(context=…, api=…)]`); the four importers (`foundation/{api,views,enum_inventory}.rs`, `feature/activity/api.rs`) switch; the ~140 lines of view tests move to `catalog/tests`.

## API Contracts

- `umpire-catalog-syntax`: `proto::Graph`.
- `umpire-catalog-build`: `System::generate`; generated `request_findings` and `annotate_<svc>` per service.
- `umpire-catalog`: `generic_probe::{Engine, ProbeSystem}`; `AnnotationRuntime::with_probes`.
- Protocol messages (`ProbeRequest`, `ProbeResponse`, `AnnotationRequest`, `AnnotationResponse`) unchanged; Go hosts unchanged.
- No Temporal detail enters `model/umpire` (owner rule): everything Temporal-specific stays behind `ProbeSystem` and the realization crate.

## Edge Cases & Constraints

- Probe results (classification, predictions, Unchanged detection, OutOfScope) stay the same for Kettle and Temporal; the probe goldens are the oracle.
- Supplemental intent catalogs in Temporal keep their findings through the generated hooks.
- Removing the facade crates must not change `#[derive(Catalog)]` diagnostics beyond path names; trybuild goldens are re-accepted.

## Acceptance Criteria

- **R1:** One descriptor graph builder serves catalog/build and realization; `#[realization]` names no Temporal crate.
- **R2:** `System::generate` replaces the build-configuration layers and wrappers.
- **R3:** Temporal's `Wire` enum is replaced by generated `request_findings`; the `mutate` question is answered in the task summary.
- **R4:** One `Engine<S: ProbeSystem>` serves Kettle and Temporal; their probe goldens hold. Errors: unknown method, bad prerequisite and unapplicable cases refuse as before.
- **R5:** Generated annotation dispatch and `AnnotationRuntime::with_probes` replace the three transport copies. Errors: unknown service prefix or method refuses as before.
- **R6:** `temporal/annotate` and `temporal/annotate/macros` are gone from the workspace.
- **R7:** `make model-check` passes, including Go host tests. No error surface beyond the gate.

## Boundaries

- No protocol, Go host or probe semantics change.
- No change to fn-246's action registry or refinement design; where fn-246 tasks touch the same files, the later spec rebases.

## Decision Context

2026-10-24, owner: filed from the consolidation review. Placed after fn-248 (which changes catalog/syntax, catalog/build and annotate/syntax for `heck`, `darling` and `quote`). fn-246 (realization as checked refinement) touches Temporal realization too; doing this spec first gives fn-246 one probe engine to build on, and the owner orders the two.

2026-10-24, owner (throughput ordering): consolidation lands before the feature work that would otherwise repeat each change in every copy. Order: fn-247 → fn-248 → fn-250, fn-251 and fn-252 as parallel lanes → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245; fn-240 and fn-211 stay deferred behind fn-250. fn-246 depends on this spec. fn-252.5 follows this spec's .6 where it touches `temporal/annotate`.
