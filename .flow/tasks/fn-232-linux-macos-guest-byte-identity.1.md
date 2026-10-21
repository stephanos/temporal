---
satisfies: [R1, R2]
---
# fn-232-linux-macos-guest-byte-identity.1 Run and record same-source Linux/macOS guest certification

## Description
Implements R1 and R2 using the existing guest builder after fn-230 is complete.

**Size:** M
**Files:** model/DESIGN.md, model/GUEST_BUILDS.md, .flow/artifacts/fn-232-linux-macos-guest-byte-identity/proof.md
**Touches:** `model/DESIGN.md`, `model/GUEST_BUILDS.md`, `.flow/artifacts/fn-232-linux-macos-guest-byte-identity/**`

### Approach
Reuse model/scripts/guest.sh and the retained runner pattern described in model/GUEST_BUILDS.md:106. Select one source pin including the completed fn-230 changes. Use a reachable authorized macOS executor and Linux/arm64 executor, two clean source paths per OS, initially absent independent targets, and matching pinned toolchain versions. Adapt local cache paths and macOS hash tooling without changing the build recipe. Compare both guests' bytes across all builds and retain every terminal command result. Update the existing design summary in place without exceeding 1000 lines. An absent executor or mismatch returns an incomplete handoff; do not weaken assertions or repair code under this proof-only task.

### Investigation targets
- model/scripts/guest.sh:13 - system-independent pinned build contract.
- model/GUEST_BUILDS.md:106 - regression, clean-clone proof and actual parity recipe.
- model/DESIGN.md:140 - current and historical proof summaries.

**Quick:** Bash syntax check; clean source/toolchain/target guards; actual comparisons for both guests.
**Full gates:** Actual guest-profile stock/control and Temporal host parity for the certified artifacts; retain hashes and exact commands. No new dependencies or lifecycle waivers.

## Acceptance
- [x] Actual Linux/arm64 and macOS builds satisfy R1 for both guests and all four independent build roots.
- [x] R2 evidence and documentation identify one source/toolchain and preserve historical proof evidence.
- [x] Missing executors, failed builds or mismatches remain explicit failures with no certification claim.

## Historical failed certification

The 2026-10-21 attempt at `8d429e97409b1537ecb347c359129bffafa63f78` completed
all four clean builds. Both guests matched across paths within each OS, but all
eight Linux/macOS comparisons failed. Stock first differs at zero-based offset
443 and Temporal at 635. Certification was incomplete at that revision pending a separately
scoped reproducibility fix. Fn-236 subsequently supplied the fix, and the final
[clean-checkout certification](../artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/proof.md)
passed at `3c314fb5b85427471a1525d6f190fe5d1fb9f3b2`. The [proof](../artifacts/fn-232-linux-macos-guest-byte-identity/proof.md)
records commands, platform/toolchain details, source guards, target identities,
hashes, sizes, comparisons and host-parity results. Historical fn-230 receipts
remain in `model/GUEST_BUILDS.md`.

Subsequent [diagnosis](../artifacts/fn-232-linux-macos-guest-byte-identity/diagnosis.md)
isolated Cargo metadata and generated-output path hashes. A controlled native
macOS replay matched both full Linux guests exactly. The replay reads reference
Linux values. Fn-236 supplied the self-contained builder fix; the later clean
committed-source certification above passed without those reference values.

## Done summary
Certified committed source `3c314fb5b85427471a1525d6f190fe5d1fb9f3b2` on native Linux 7.0.14/aarch64 and macOS 25.6.0/arm64. Four fresh clean detached clones used independent initially absent targets and the same pinned Rust/Cargo/Go/protoc versions. All twelve actual byte comparisons passed for stock (192271 bytes, SHA-256 f3da11cbe34e75e24ec698979058a6f300aad016e5215a63fe2d392951341219) and Temporal (3330006 bytes, SHA-256 5bd641822d7a124f10b247d022d8f6d296fe13632f418acfa23bd85152a6285e).

Both native stock/control and embedded Temporal host suites passed, including caught panic/overflow parity, repeated calls/Reset, trap invalidation and exact import denial. All four clones retained the pinned revision and clean status after testing. Every Temporal build matches its committed embedded asset.

The permanent fix was tracked separately as fn-236. The design summary remains 1000 lines and the build guide records the committed certification separately from the frozen validation and earlier failed/Linux-only proofs. Exact commands, tools, source/target guards, manifests, comparisons and native transcripts are retained at `.flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/proof.md`. This session created no commit.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3c314fb5b85427471a1525d6f190fe5d1fb9f3b2
- Tests: run-proof.sh on native Linux/aarch64 with four-guarded source/target lane checks (exit 0), run-proof.sh on native macOS/arm64 with four-guarded source/target lane checks (exit 0), compare.py executed 12 actual cmp calls and repeated all four clean-source/pin checks (exit 0), Both native stock/control and embedded Temporal host suites passed against certified artifacts, Bash syntax, scoped diff checks and Flow requirement validation
- PRs: