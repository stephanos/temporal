---
status: done
---

# Plan: Move Gomad v3 Runtime Additions into an Overlay

## Context

Refactor Gomad v3 construction so the versioned Go patch contains only edits
to upstream runtime files. Net-new runtime source lives under
`tools/gomad3/overlay/src/runtime/`, is validated and hashed as an immutable
build input, and is copied into the extracted Go source without changing any
deterministic behavior.

The approved design is a compile-time seam inside `package runtime`, not a
late package-initialization registration mechanism. The overlay must retain the
same strict runtime scope and failed-build publication guarantees as the patch.

## Pattern Survey

### Analogous Features
- `tools/gomad3/build.sh:73` — Copies the patch into generated toolchain state, validates and hashes that immutable snapshot, includes its digest in the cache key, then applies the same snapshot with zero fuzz.
- `tools/gomad1/ctrl/program.go:334` — Walks a static overlay tree, preserves relative paths, creates destination directories, and copies file contents into a generated source tree.
- `tools/gomad2/gomadmain/selftest.go:130` — Builds an explicit source-file set, recreates its relative directory layout in generated state, and copies each file into that layout.
- `tools/gomad3/test.sh:81` — Negative builder testing verifies invalid source customization fails without replacing the stable published toolchain.
- `tools/gomad3/build.sh:79` — Immutable builds are addressed by content-derived keys and published through a stable wrapper only after successful construction.

### Reusable Utilities
- `tools/gomad3/build.sh:43` — `sha256_file` — Provides the existing portable SHA-256 convention for Gomad v3 build inputs.
- `tools/gomad3/test.sh:10` — `validate_patch` — Encodes the existing runtime path allowlist plus prohibited filename, platform-file, generated-output, and binary rules relevant to equivalent overlay validation.
- `tools/gomad2/internal/translate/cache.go:35` — `NewHasher` / `addFile` — Existing precedent for hashing both a relative path and that file’s content, preventing equal-content files at different paths from sharing an identity.
- `tools/gomad2/gomadmain/main.go:98` — `hashFile` — Streams file contents through SHA-256.
- `tools/gomad2/gomadmain/main.go:111` — `copyFile` — Existing regular-file copy helper preserving caller-selected permissions.
- No existing shell utility performs a strict, symlink-rejecting immutable directory snapshot; Gomad v3 currently snapshots only one regular patch file.

### Convention Anchors
- Runtime scope ownership: `GOMAD_ALT.md:48` and `GOMAD_ALT.md:178` keep v3 behavior confined to `gomad.go`, `proc.go`, and `rand.go`, with activation and early initialization owned by the net-new runtime module.
- Approved artifact boundary: `docs/superpowers/specs/2026-08-10-gomad3-runtime-overlay-design.md:14` places net-new runtime source at `tools/gomad3/overlay/src/runtime/` while retaining modifications to upstream files in `go1.26.4.patch`.
- Validate before identity or mutation: `tools/gomad3/build.sh:73` validates the snapshot before computing the build key, extracting source, or publishing a toolchain.
- Path-and-content cache identity: `tools/gomad2/internal/translate/cache.go:59` hashes file paths with contents; ordered aggregate inputs are established at `tools/gomad2/internal/translate/cache.go:102`.
- Explicit build inputs: `tools/gomad3/Makefile:3` names source customization artifacts as prerequisites while `build.sh` computes their dynamic content identity.
- Collision behavior differs by domain: static Gomad v1 overrides intentionally overwrite generated files at `tools/gomad1/ctrl/program.go:334`; the approved v3 design instead requires upstream collisions to fail at `docs/superpowers/specs/2026-08-10-gomad3-runtime-overlay-design.md:37`.
- Generated-state boundary: `tools/gomad3/.gitignore:1` excludes all snapshots, downloads, locks, and built toolchains under `.toolchain`, while authored patch and overlay inputs remain tracked.

### Proposed Alignment

Blend the existing Gomad v3 immutable patch-snapshot/cache-key flow with the
repository’s relative-tree overlay and path-plus-content hashing patterns.
Implement the v3-specific strict regular-file validation, symlink rejection,
and no-overwrite collision rule locally.

## Implementation Steps

1. **Pin the overlay contract with failing builder tests**
   - Extend `tools/gomad3/test.sh` with negative `GOMAD3_OVERLAY_DIR` cases for an empty overlay, a prohibited runtime file, a symlink, and an upstream `src/runtime/proc.go` collision.
   - Assert each rejected build leaves the existing stable GOROOT unchanged.
   - Add a valid Git new-file patch fixture and require patch validation to reject it, proving new runtime files belong in the overlay.
2. **Separate authored runtime additions from the upstream patch**
   - Add `tools/gomad3/overlay/src/runtime/gomad.go` with the exact current net-new runtime source and preserved comments.
   - Remove the `src/runtime/gomad.go` creation hunk from `tools/gomad3/go1.26.4.patch`, retaining only the existing `proc.go` and `rand.go` modifications.
3. **Validate and identify the overlay as an immutable input**
   - Add `GOMAD3_OVERLAY_DIR` and `validate_overlay` to `tools/gomad3/test.sh`, sharing the existing runtime path rules with `validate_patch` and rejecting empty trees, non-regular entries, platform/prohibited names, generated markers, and NUL-containing source.
   - Extend `tools/gomad3/build.sh` cleanup and snapshot flow to copy and revalidate the overlay before hashing it.
   - Compute a deterministic aggregate digest from sorted relative paths and file content digests, and include it in `build_key`.
4. **Copy the overlay without overwriting upstream source**
   - In `tools/gomad3/build.sh`, scan every validated overlay destination after archive extraction and fail before copying if any destination already exists.
   - Copy the immutable overlay snapshot into the extracted Go tree before applying the modifications-only patch.
   - Add overlay files to `tools/gomad3/Makefile` toolchain prerequisites while retaining validation as a mandatory dependency.
5. **Update construction documentation and verify unchanged behavior**
   - Update `tools/gomad3/README.md`, `GOMAD_ALT.md`, and the completed Gomad v3 plan to describe patch-plus-overlay construction and cache identity.
   - Rebuild the custom toolchain, confirm the patch contains no new-file hunk, and run the complete Gomad black-box/upstream runtime suite plus root workflows and repository standards checks.

## Verification

- Run the new negative overlay cases before implementation and observe failure because `GOMAD3_OVERLAY_DIR` is ignored and new-file patches are accepted.
- `tools/gomad3/test.sh validate` accepts the default overlay and modifications-only patch, while alternate invalid overlays fail with actionable errors.
- `make -C tools/gomad3 toolchain` builds a new immutable key containing the overlay digest and copies `gomad.go` into the custom GOROOT.
- `make -C tools/gomad3 test` passes all 100-process repeatability, cross-seed diversity, cache, watchdog, generated-test-binary, and upstream `runtime` checks with `-tags test_dep`.
- Root `gomad3-run` and `gomad3-test` workflows retain their output, `bash -n` and `git diff --check` pass, and `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=HEAD lint-code` reports no new issues.

## Context Files

- `docs/superpowers/specs/2026-08-10-gomad3-runtime-overlay-design.md` — approved overlay behavior and failure contract.
- `GOMAD_ALT.md` — runtime scope and deterministic behavior contract.
- `tools/gomad3/build.sh` — immutable build inputs, locking, extraction, patching, and publication.
- `tools/gomad3/test.sh` — patch scope validation and black-box builder/runtime tests.
- `tools/gomad3/go1.26.4.patch` — current runtime additions and upstream modifications to separate.
- `tools/gomad3/Makefile` — toolchain dependency and validation ordering.
- `tools/gomad3/README.md` — user-facing construction and cache documentation.
