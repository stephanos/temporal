# Renew the Linux/macOS guest byte-identity certification (model)

## Conversation Evidence

> user (2026-10-22): "move veritication to seprate task outside of spec"

## Goal & Context

fn-232 certified at `3c314fb5b8` that four clean builds (two per OS) produce byte-identical stock and Temporal guests on Linux and macOS. fn-238.8 then added a per-crate lockfile closure digest to every guest crate identity, so that certification no longer covers how identities are derived. This spec renews it at a committed revision at or after the fn-238.8 merge (`cbe2c6fcb7`). It stands alone: no other spec waits on it.

## Acceptance Criteria

- **R1:** Four clean builds (two clean detached clones per OS, independent initially absent targets, one pinned toolchain) at one committed revision produce byte-identical stock and Temporal guests; all twelve comparisons pass; both host suites pass on both OSes; the Temporal build matches the committed asset. Errors: a missing executor, failed build or mismatch is an explicit failure with no certification claim.
- **R2:** The proof (commands, toolchains, source/target guards, manifests, comparisons, host logs) is retained, and the "needs renewal" caveats in `model/DESIGN.md` and `model/GUEST_BUILDS.md` are replaced by the renewed certification.

## Boundaries

- Needs a native macOS run on the owner's Mac; this sandbox is Linux-only.
- No build recipe, dependency or Wasm artifact changes to make bytes match.

## Decision Context

2026-10-22, owner: keep this verification outside fn-238 as its own spec.
