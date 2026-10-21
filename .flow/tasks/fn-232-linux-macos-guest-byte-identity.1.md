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
- [ ] Actual Linux/arm64 and macOS builds satisfy R1 for both guests and all four independent build roots.
- [x] R2 evidence and documentation identify one source/toolchain and preserve historical proof evidence.
- [x] Missing executors, failed builds or mismatches remain explicit failures with no certification claim.

## Incomplete handoff

The 2026-10-21 attempt at `8d429e97409b1537ecb347c359129bffafa63f78` completed
all four clean builds. Both guests matched across paths within each OS, but all
eight Linux/macOS comparisons failed. Stock first differs at zero-based offset
443 and Temporal at 635. Certification remains incomplete pending a separately
scoped reproducibility fix. The [proof](../artifacts/fn-232-linux-macos-guest-byte-identity/proof.md)
records commands, platform/toolchain details, source guards, target identities,
hashes, sizes, comparisons and host-parity results. Historical fn-230 receipts
remain in `model/GUEST_BUILDS.md`.

## Done summary
Blocked:
Certification failed at source `8d429e97409b1537ecb347c359129bffafa63f78` on
2026-10-21. All four clean builds completed, and both guests were path-independent
within each OS. All eight Linux/macOS comparisons failed. Stock first differs at
zero-based offset 443 and Temporal at 635. See the retained
proof at `.flow/artifacts/fn-232-linux-macos-guest-byte-identity/proof.md`,
platform transcripts, manifests and `comparisons.json`.

R1 remains unmet. The spec requires a separately scoped reproducibility fix
before another certification attempt. No build recipe, dependency version or
Wasm artifact was changed to hide the mismatch. Keep fn-232 open.

## Evidence
- Commits:
- Tests:
  - `bash model/scripts/guest.sh` for stock and Temporal at all four clean roots; builds passed.
  - `python3 .flow/artifacts/fn-232-linux-macos-guest-byte-identity/compare.py "$PWD"`; expected exit 1, four path comparisons passed and eight cross-OS comparisons failed.
  - `bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/run-proof.sh` on Linux; exit 0 including both actual host suites.
  - `bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/run-parity.sh` on macOS; exit 0 including both actual host suites, using a separate test copy for the emitted Temporal guest.
- PRs:
