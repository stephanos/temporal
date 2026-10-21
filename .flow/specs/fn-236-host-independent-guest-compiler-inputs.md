# Host-independent guest compiler inputs

## Goal & Context

Make the existing guest builder self-contained on native Linux and macOS. fn-232 diagnosis isolated host-dependent Cargo crate metadata and generated OUT_DIR directory hashes. The owner authorized implementing the fix and retrying certification in this session.

## Acceptance Criteria

- **R1:** The builder derives target crate identities independently from normalized package, configuration and dependency identities, with no recorded platform hash map or binary rewriting. Distinct packages, features, profiles and target dependency variants retain distinct identities. Missing dependency identities fail explicitly.
- **R2:** Generated-source paths are remapped independently of Cargo directory hashes. The existing target, unwind profile, ABI, protocol, dependency versions, failure codes and check-mode immutability remain intact. Host tools continue executing natively.
- **R3:** Focused regression tests exercise cross-host/path equivalence, variant separation, dependency propagation, absent/corrupt identity handling and compiler failure. The full model gate and actual stock/control and Temporal hosts pass for the refreshed embedded asset.
- **R4:** Four independently materialized frozen source snapshots (base commit plus the reviewed fix), with initially absent independent targets demonstrate within-OS path and cross-OS byte identity for both guests, with source manifests, platform/toolchain details, hashes and comparisons retained. fn-232's final committed-checkout certification uses the user-created source commit; do not create a commit without authorization.

## Boundaries

Only guest build scripts, their tests, embedded Temporal guest freshness and associated proof documentation. Preserve existing comments and other agents' changes. No worktrees, remote provisioning, dependency upgrades, model semantics changes or commits.

## Implementation

Add a focused Python rustc wrapper. Keep Cargo's own cache fingerprints and filenames, derive a separate stable target symbol identity, and propagate target dependency identities through per-artifact sidecars. Record host proc-macro package/configuration identities without changing native compilation. Use a versioned guest cache so all required sidecars are created, and normalize OUT_DIR to a crate-identity path. Test the transformation and its real compiler boundary before integration, then run the actual model gate and independent cross-OS builds.
