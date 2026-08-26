---
satisfies: [R1, R3]
---
# fn-23-veil-toolchain-compatibility-and.1 Freeze the Veil candidate and acquisition contract

## Description
Implement the closed compatibility manifest, Linux/aarch64 reference profile, and pre-execution acquisition validator in `tools/umpire/veilcompat`. Pin the preview/main commits, exact 13-package Git closure, 229-entry npm lock closure, three solver archives, Node/npm bundle, Lean bundle, Zig compiler/sysroot/linker bundle, Ubuntu runtime manifest/config/layers, constant `clang` wrapper, complete merged/unpacked-tree identities, current/declared Lean versions, candidate order, numeric process/resource ceilings, and cost thresholds from the spec. Populate content-addressed Git/Lake/npm/solver/tool/rootfs caches in a caller-owned temporary root with ambient Git/SSH/proxy credentials/config disabled; verify commits, manifests, SRI/checksums, paths, symlinks, OCI extraction, counts, sizes, URLs, bundle trees, wrapper, supported host, recursive ELF resolution, and hermetic compile/link/load preflight before dependency code executes. Separate transport/integrity/tooling errors from conclusive candidate incompatibility and preserve existing comments.

**Size:** M
**Files:** `tools/umpire/veilcompat/manifest.go`, `tools/umpire/veilcompat/reference_linux_aarch64.go`, `tools/umpire/veilcompat/acquire.go`, `tools/umpire/veilcompat/manifest_test.go`, `tools/umpire/veilcompat/acquire_test.go`
**Touches:** [tools/umpire/veilcompat/manifest.go, tools/umpire/veilcompat/reference_linux_aarch64.go, tools/umpire/veilcompat/acquire.go, tools/umpire/veilcompat/manifest_test.go, tools/umpire/veilcompat/acquire_test.go]

## Acceptance
The exact two candidates and entire reference closure validate deterministically; reordered, duplicate, moving, unknown, unresolved, unsafe, mutable, cache-missing, rootfs/bundle/wrapper/preflight-mismatched, ambient runtime/compiler-path-resolving, or undeclared inputs fail before dependency execution with no repository write. Fake acquisition covers success, unsupported host status 2, supported-host mismatch status 1, transport failure, every Git/npm/archive/OCI/tool-bundle/tree digest family, bad manifest, branch-only dependency, OCI whiteout/path escape, symlink/path escape, submodule, every numeric N/N+1 closure bound, credential stripping, offline-cache layout, missing/escaping ELF interpreter or `DT_NEEDED`, compiler include/resource/crt/libc++/linker/archive-tool escape, and complete cleanup.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
