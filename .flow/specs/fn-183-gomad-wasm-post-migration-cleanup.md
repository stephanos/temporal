# Gomad: WASM post-migration cleanup

## Conversation Evidence

> user: "as well as 8. Post-migration cleanup"

## Goal & Context

Simplify the WASM execution architecture after fn-176 and fn-178 by removing redundant recovery reconstruction and obsolete migration plumbing, while keeping Go models as differential references and retaining supported stock/native behavior. [paraphrase]

fn-176 records a specific redundant operation: recovery-image decoding constructs namespace nodes and byte accounting that recovery immediately discards while rebuilding the durable state. fn-178 already owns production cluster routing to one Rust helper and retention of the per-process Go broker as a reference. This cleanup addresses remaining duplication after those contracts are delivered.

## Architecture & Data Models

Production cooperative clusters use fn-178's Rust helper. The per-process Go broker reference remains accessible to differential tests through an explicit test selection; supported Go-owned standalone, stock and native paths retain their existing owners. Shared Go model sources, generated projections and strict evidence decoders remain available where those consumers need them.

Recovery-image decoding and recovery construct each necessary state representation once. Removing discarded intermediate state must preserve all decode-time validation, bounds, image identity, operation counters, fault occurrence counters and durable restart semantics.

Dependencies are fn-176 and fn-178. Their unfinished migration requirements are not transferred into this spec.

## API Contracts

Public commands, model semantics and current retained evidence formats remain unchanged. Remove transitional selectors or duplicate routes only when their supported callers are gone; keep any reference-only selection outside normal production configuration. Existing incompatible historical artifact handling remains an explicit refusal.

## Edge Cases & Constraints

Reference execution is needed to detect semantic drift. Code shared with the stock profile, standalone Go network owner, native execution, conformance generation or historical artifact inspection is not dead solely because cooperative clusters moved to Rust. Package or source changes can rotate implementation identities and require truthful requalification.

## Implementation Design

### Route inventory and proven removals (.1)

Re-anchor the inventory to the completed fn-176/fn-178 source, including the final reopened fn-176 crash-selection correction. `cluster/provider.go` and `wasi/cluster.go` own cooperative cluster routing and the one-helper client; `wasi/process.go`, Go broker/network adapters and shared generated models still have standalone/stock/reference consumers. Inspect all normal installation/provider/config/replay/resume callers, build-tag/source sets and differential/qualification entry points before choosing removals. Record each candidate branch/adapter/selector, remaining consumers and why its production responsibility is gone. A path used only by differential tests or conformance generation is intentionally retained, not dead code.

fn-178 already defaults cluster execution to the Rust helper and keeps an explicit Go-broker test selector; verify the actual final seam rather than introduce another transport selector or redo that migration. Remove only residual normal-production selection of a reference route when the final supported-profile contract proves it obsolete. Keep the test seam unexported/test-only and reachable by the existing parity tests, without making the general standalone Go owner inaccessible. If a path still provides a shipped capability (including any final supported isolation/profile behavior), preserve it or retain its required consumer before removing the adapter; changing a supported request into a refusal is not cleanup. If fn-178 already removed a proposed leftover, record the proof and leave it removed rather than invent new deletion work.

Pin production/reference routing before/after with actual final fn-178 fixtures: one-helper clusters, delayed/never-started nodes, incarnation crash/stop/restart holds and recovery images, frame admission, per-incarnation evidence, global World/lifecycle order, network faults, capacity and helper reaping. Keep standalone/stock/default/native source behavior, source-less retained replay/recovery and explicit historical-format refusals. No new operator feature or current evidence-format change belongs here; fn-181 and fn-180 owners retain their own contracts if integrated before cleanup.

### Decode validation without discarded materialization (.2)

The concrete redundant work is `volume/image.rs`'s `RecoveryImage::decode` materializing `fs.nodes` and `fs.used_bytes` after reading durable volumes, immediately followed by `recover` calling `Fs::clone_durable` in `volume/crash.rs` to construct the final namespace again. Separate that discarded materialization from structural/accounting validation. Retain the authenticated durable core, full image fields, counters and fault occurrences; build the final recovered namespace once. Survey every `RecoveryImage.fs`/encode/recover consumer before changing representation so round-trip, evidence and conformance callers remain correct.

Keep exact current image bytes/schema/digest and refusal ordering: bounded frame/section lengths, model/config matching, ordering, object/inode uniqueness across volumes and reserved nodes, normalized paths, object references, valid effects and dependencies, generation/operation counters, fault occurrences, and SQLite process-owned-path exclusions. Preserve next-inode/rebuild-allocation headroom and overflow checks even if they previously ran beside reconstruction. Go permits dependency-closed partial persistence with a durable child before its parent namespace operation, and permits a recovered durable state larger than the volume's volatile-growth capacity; keep those valid cases. Do not invent a generic parent-directory requirement or substitute volume capacity for the existing `recovered-namespace` bound. Recompute digests on malformed test images so checksum rejection cannot hide missing structural checks; include valid orphan/unreferenced durable state controls alongside rejected aliases/counters/paths.

fn-176's full pending-operation crash-selection context, its digest, strict Go/Rust pre-start validation, dependency closure, chunked descriptor transport and actual-pending mismatch classification remain required. They are separate trust-boundary and retained-state contracts, not redundant copies to remove because the helper owns execution. Do not discard pending/history/effect material just because recovery resets pending operations: image authentication/round-trip, enumeration, selection and replay consumers still need it. Preserve absent-context/default nil bytes. Keep image bytes, complete storage-state evidence, refused-history evidence, crash-context and transcript capacities independent, including saturating arithmetic and header admission; an image-size bound cannot stand in for the others.

### Integrated proof and documentation (.3)

Before cleanup pin current canonical outputs and classification for representative production/reference and recovery scripts; after cleanup compare operation replies, namespace/volume snapshots, image bytes/identity, operation/fault occurrence counters, per-incarnation choice/WASI/network/storage evidence and shared World/lifecycle order. Source/helper/model implementation digests may rotate from changed files; distinguish those expected identity changes from unchanged semantic evidence and never relabel old qualifications. Keep all Go namespace/volume/network/World oracles, generated native-source projections, conformance corpora and source-drift/architecture checks. Do not replace the independent Go oracle with a comparison of two Rust paths or delete callback/strict-decoder consumers used by source-less replay.

Run the affected Rust image/volume and Go decoder/model/differential controls, explicit malformed-image and context-selection controls, actual storage crash/reopen and cluster lifecycle parity, and detached retained replay plus interrupted recovery/resume with source/compiler/mounts/preparation caches unavailable. Preserve fn-176's independent storage-evidence admission regressions and final full-context coverage, fn-178's callback/refusal/lifecycle controls and shared resource budgets. Run affected lint/host-source/architecture/generated identity validation, full `test-wasm` and relevant retained qualification selectors once on the integrated candidate; verify none silently skipped. No performance improvement is required or claimed merely from deleting allocations.

Tasks .1 and .2 may develop in parallel with disjoint declared write surfaces: .1 owns cluster/client/installation routing and `wasi/{cluster*.go,process.go,integration_test.go}`; .2 owns Rust recovery image/volume/namespace helpers plus `wasi/volume_test.go` and storage-evidence tests. The routing task does not own the broad `wasi/*test.go` glob. If a new common owner appears, serialize it or consolidate into one subsystem batch; .3 integrates before task acceptance. Keep one R1–R5 handover listing the proven removals and preserved consumers, exact candidate/tool identities, commands/exits and independent review. Update existing architecture/capability/qualification documents with the final owners and actual evidence. Native execution remains deferred under fn-128/fn-149 while portable native-source controls remain required. This planning pass performs no source edits, build/runtime tests, Flow completion, PR, push or CI action.

## Acceptance Criteria

- **R1:** Review the completed fn-176/fn-178 execution routes and remove any residual production selection of the per-process Go broker reference, with an explicit test-only path retained for parity coverage. Errors: unsupported reference selection through normal installation/configuration fails rather than silently changing owners; supported stock/native and standalone routes remain functional.
- **R2:** Eliminate recovery-image namespace and byte-accounting reconstruction that recovery discards, while retaining strict image validation, bounds and deterministic durable restart behavior. Errors: malformed, inconsistent, truncated or over-limit images retain their existing refusal classifications before execution.
- **R3:** Remove obsolete migration adapters, branches or configuration plumbing only when caller and supported-profile analysis proves them redundant; document the removed responsibilities and remaining execution owners. Errors: code with a required stock/native, replay-inspection or differential-test consumer is retained or deliberately relocated without losing that consumer.
- **R4:** Differential namespace/volume/network/World tests and drift guards continue to run against the Go reference models; affected recovery, cluster lifecycle and retained replay gates pass on the cleaned candidate. Errors: lost reference coverage, evidence divergence or changed failure classification blocks completion.
- **R5:** Refresh architecture and capability documentation for the final owners, recording any source-identity rotation and current-candidate qualification evidence without relabeling historical results. Errors: unavailable native execution remains explicitly deferred, and missing WASM acceptance is not claimed as passed.

## Boundaries

- Keep Go differential implementations, generated native projections and their provenance; this is not removal of the Go backend or stock support.
- No new operator features, evidence schemas, model semantics, non-UTF-8 volume-path support or execution-performance target.
- No broad package reorganization or unrelated cleanup; remove only demonstrated post-migration duplication and obsolete paths.

## Decision Context

The Rust migration removes production callbacks but intentionally keeps independent Go references. Clarifying that boundary and eliminating the known discarded recovery state reduces maintenance work without weakening the parity checks that justify the Rust models.

Planning confirmed the discarded recovery reconstruction and fn-178's ownership of the actual cluster migration. Cleanup verifies delivered routing and removes only residual duplication. Authenticated malformed-image tests preserve inode/path validation; complete storage evidence, refused history and recovery images retain independent limits. Tasks .1 and .2 can proceed in parallel after prerequisites; task .3 validates their integrated candidate.

Planning refinement (2026-10-15): no shipped capability or source-less replay/recovery consumer can be removed as migration debris; preserve fn-176's corrected full crash-selection context and both trust-boundary validators. Separate discarded namespace materialization from required validation and retain Go's legitimate partial-persistence orphan/capacity behavior.

## Early proof point

fn-183-gomad-wasm-post-migration-cleanup.1 establishes the supported production/reference route inventory; task .2 independently proves recovery-state equivalence. If either proof fails, retain the affected path or validation before the integrated gate in task .3.

## Quick commands

Use pinned tool versions and the documented helper inputs for actual guest gates.

```sh
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Review the completed fn-176/fn-178 execution routes and remove any residual production selection of the per-process Go broker reference, with an explicit test-only path retained for parity coverage. Errors: unsupported reference selection through normal installation/configuration fails rather than silently changing owners; supported stock/native and standalone routes remain functional. | fn-183-gomad-wasm-post-migration-cleanup.1 | — |
| R2 | Eliminate recovery-image namespace and byte-accounting reconstruction that recovery discards, while retaining strict image validation, bounds and deterministic durable restart behavior. Errors: malformed, inconsistent, truncated or over-limit images retain their existing refusal classifications before execution. | fn-183-gomad-wasm-post-migration-cleanup.2 | — |
| R3 | Remove obsolete migration adapters, branches or configuration plumbing only when caller and supported-profile analysis proves them redundant; document the removed responsibilities and remaining execution owners. Errors: code with a required stock/native, replay-inspection or differential-test consumer is retained or deliberately relocated without losing that consumer. | fn-183-gomad-wasm-post-migration-cleanup.1 | — |
| R4 | Differential namespace/volume/network/World tests and drift guards continue to run against the Go reference models; affected recovery, cluster lifecycle and retained replay gates pass on the cleaned candidate. Errors: lost reference coverage, evidence divergence or changed failure classification blocks completion. | fn-183-gomad-wasm-post-migration-cleanup.3 | — |
| R5 | Refresh architecture and capability documentation for the final owners, recording any source-identity rotation and current-candidate qualification evidence without relabeling historical results. Errors: unavailable native execution remains explicitly deferred, and missing WASM acceptance is not claimed as passed. | fn-183-gomad-wasm-post-migration-cleanup.3 | — |
