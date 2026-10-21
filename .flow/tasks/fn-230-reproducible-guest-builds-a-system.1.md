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

This task proves Linux/aarch64 path independence. The owner moved macOS and
cross-OS proof to fn-232-linux-macos-guest-byte-identity; it does not gate this
task or task .2. Never claim Linux directories prove cross-OS identity. The
caught-panic requirement remains unchanged. Task .2 owns embedding and freshness wiring.

**Touches:** `model/Cargo.toml`, `model/Cargo.lock`, `model/scripts/guest.sh`, `model/scripts/tests/**`, `model/umpire/hosts/go/**`, `model/umpire/wasm/**`, `model/temporal/hosts/go/allocation_test.go`, `Makefile`, `model/DESIGN.md`, `model/GUEST_BUILDS.md`
**Quick:** bash syntax and script controls for fresh/stale/build-failure exit codes; actual caught-panic/overflow reports, reset and unexpected-trap controls through the production Go host; restricted import denial/signature controls; the two-clean-checkout proof for both guests.
**Full gates:** `make model-check`; `make model-time`; stock/control Go parity against artifacts built with profile.guest.
Use a lane-specific Cargo target and isolated proof targets. Proof source copies
are temporary artifacts, not retained worktrees. No new dependencies or Model,
ABI, protocol, export or Realization changes.
Keep DESIGN.md below the documented 1,000-line threshold: retain the actual proof
summary, hashes/sizes, source/platform/toolchain and unresolved requirements there,
with exact detailed reproduction/diagnostic commands in a linked focused
model/GUEST_BUILDS.md. This documentation layout does not waive caught-panic
obligations or the successor's cross-OS proof, and must not move unrelated older design sections.

### Proof-driven unwind implementation (2026-10-21)

The conductor's isolated proof retained the pinned Rust 1.99.1 compiler and
`wasm32-wasip1`, rebuilt `std` and `panic_unwind` with final WebAssembly exception
handling, and ran the actual existing effect-panic fixture through the existing ABI.
The full native report matched on three calls with successful Reset. The same bytes
failed under wazero 1.12's ARM64 compiler but passed under its existing interpreter.
Implement this route rather than weakening the caught-panic requirement:

- Keep the exact compiler pin and existing dependency versions. Rebuild the standard
  library with command-scoped `RUSTC_BOOTSTRAP=1`, `-Zbuild-std=std,panic_unwind`,
  explicit unwind and modern EH code generation. Preserve path remapping and locked
  builds. Document the unstable build-policy tradeoff; do not set bootstrap globally,
  switch to nightly, patch the standard library, or add an alternative WASI SDK.
- Use the existing wazero interpreter and EH feature in the transport-only Go host.
  Preserve bounds, cancellation, serialization, exact import/export admission and
  invalidation after uncaught traps. This changes no Go/SDK panic handling.
- Rebuilt std additionally imports `fd_close` and `fd_seek`. Admit only their exact
  signatures and supply unconditional WASI BADF denial stubs, with no filesystem,
  descriptor lifecycle, seek or output-memory effect. Link no general WASI module.
  Negative tests must call the registered imports with arbitrary descriptors and
  offsets and prove rejection and unchanged output memory.
- Retain permanent guest-profile caught-panic and overflow controls using existing
  checker fixtures/semantics, including a subsequent call and Reset. Keep uncaught
  trap/invalidation controls separate. No Model, ABI or protocol semantics change.
- Wire the relevant guest build/host checks so the actual tested artifacts use the
  guest profile. Do not use a release-profile green check as unwind evidence.
  Task .2 still owns committing/embedding Temporal's artifact and freshness wiring.
- Repeat the stock and Temporal two-clean-checkout proof for this code generation;
  old aborting-artifact hashes do not certify it. Record source/toolchain/platform,
  independent targets, commands, hashes, sizes and actual timing.

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
