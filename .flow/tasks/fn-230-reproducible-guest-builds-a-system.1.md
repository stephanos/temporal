---
satisfies: [R1]
---
# fn-230-reproducible-guest-builds-a-system.1 Guest profile and guest.sh in Umpire; reproducibility proven on two clean checkouts

## Description
Add profile.guest inheriting release with debug=false, strip="symbols", lto="fat",
codegen-units=1, overflow-checks=true, panic="unwind". Add the system-agnostic
model/scripts/guest.sh <crate> <output> [--check] using wasm32-wasip1, --locked,
the pinned mise toolchain, and remapped workspace/registry/sysroot paths. The script
must not encode system crate names or paths. Exit 0 for successful build/freshness,
1 for build failure, 2 for stale bytes with sizes and first differing offset.
Keep checker release profiling unchanged and preserve caught-panic judgments.

Prove both umpire-temporal-guest and umpire-stock-guest byte-identical in two clean
source checkouts at different absolute paths with independent target directories.
Record exact commands, hashes, sizes, platform and toolchain in DESIGN.md. Verify
the stock guest's actual Go-host reports still match the native goldens under this
new profile, including panic controls. Do not substitute a release-profile parity
run as evidence about profile.guest.

This environment is Linux/aarch64. Prove what is actually available and retain the
spec's macOS cross-platform proof as explicitly unverified if no macOS executor
is reachable; never claim two Linux directories prove cross-OS identity. Exhaust
available safe executor checks and deliver the Linux implementation/proof rather
than stopping before making progress. Task .2 owns embedding and freshness wiring.

**Touches:** `model/Cargo.toml`, `model/scripts/guest.sh`, `model/scripts/tests/**`, `model/DESIGN.md`, `model/GUEST_BUILDS.md`
**Quick:** bash syntax and script controls for fresh/stale/build-failure exit codes; the two-clean-checkout proof for both guests.
**Full gates:** `make model-check`; `make model-time`; stock/control Go parity against artifacts built with profile.guest.
Use a lane-specific Cargo target and isolated proof targets. Proof source copies
are temporary artifacts, not retained worktrees. No new dependencies or Model,
ABI, protocol, export or Realization changes.
Keep DESIGN.md below the documented 1,000-line threshold: retain the actual proof
summary, hashes/sizes, source/platform/toolchain and unresolved requirements there,
with exact detailed reproduction/diagnostic commands in a linked focused
model/GUEST_BUILDS.md. This documentation layout does not waive caught-panic or
macOS/cross-OS obligations, and must not move unrelated older design sections.

## Acceptance
- [ ] profile.guest and guest.sh present with the specified exit codes
- [ ] Two clean checkouts produce identical bytes for both guests; sizes in DESIGN.md
- [ ] Stock guest report byte-identical to goldens; gate passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
