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
