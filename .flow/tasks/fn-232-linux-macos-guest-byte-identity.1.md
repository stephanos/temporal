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
- [ ] R2 evidence and documentation identify one source/toolchain and preserve historical proof evidence.
- [ ] Missing executors, failed builds or mismatches remain explicit failures with no certification claim.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
