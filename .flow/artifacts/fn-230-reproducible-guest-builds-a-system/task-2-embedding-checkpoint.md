# Temporal embedding checkpoint

Temporal Go consumers now read `temporal.Guest()` from the committed wasm asset.
The slice, gaps, umbrella and Model gate no longer select Temporal Cargo output.
`make umpire-guest` refreshes through the existing generic builder. The freshness
step compares bytes and distinguishes unavailable prerequisites from compiler,
metadata and stale-output failures. Generic stock/control guests stay source-built.

The initial asset is 3,330,375 bytes, SHA-256
`0d4e1b112097ba62f62f53c6a95ecc7776bfbf907df8594592586d3fe767ece0`.
It matches task 1's integrated Linux two-clone proof. No Rust guest source or build
recipe changed in this checkpoint. The authorized metadata remediation follows
in a separate commit and requires a renewed proof if it changes these bytes.

Focused evidence under `.flow/tmp/fn2302/`:

- `no-rust-before.log` fails on the old Cargo release lookup. `no-rust-after.log`
  passes the complete Temporal host module in 7.015 seconds with `rustc` and Cargo
  unavailable on PATH, absolute Go execution and `GOTOOLCHAIN=local`.
- Six Python controls pass. They exercise the real wrapper and builder, the real
  Make gate seam, missing embedded-file compilation, freshness/build exits,
  local versus empty-CI prerequisite policy, and command/malformed-metadata errors.
  Red observations are `freshness-source-red.log`, `embed-contract-red.log` and
  `metadata-red.log`. `controls-metadata-fixed.log` is the latest pass.
- `stale-model-check.log` records Make exit 2 with existing/built sizes of
  3,330,375 and first differing offset 8. `refresh-after-stale.log` and
  `stale-control.log` show `umpire-guest` restoring the original bytes.
- Real absent-rustup controls return local 0 with a warning and empty-CI 1.
  `real-missing-local.log` and `real-missing-ci.log` retain both observations.
- Root lint with integration tags passes; complete Temporal host vet passes.
  Unfiltered host lint exposed inherited aliases and ObserveProbe complexity.
  The touched parity imports were cleaned; unchanged-file findings remain scoped
  out of this task's changed-file lint. No broad lint refactor was made.
- DESIGN remains 999 lines. Git reports wasm as binary and no LFS filter.

The live baseline is red. The first combined wrapper exited 124 before gaps
completed. Its retry reached the unchanged 20-minute Go deadline after 389 of
2,317 cases, with Go FAIL 1200.527 seconds and Make exit 2. The earlier slice
baseline passed in 1.111 seconds. These are inherited observations, not a live
gate pass for this checkpoint. Full post-remediation gates remain required.

Matched selected live diagnostics passed their selected case in 4.21 seconds
for release bytes and 4.29 for guest-profile bytes. Both filtered parents failed
the expected full-inventory count. They are diagnostics, not green gates.
The actual embedded no-RPC control in `probe-diagnostic.go` ran twelve Prepare
calls across two contexts in 1.047-1.083 seconds each, including repeated calls.
It decoded actual requests, checked current fields and mutations, compared full
repeated responses and proved distinct-context bytes differ. Memory stabilized
at 44 pages. `probe-diagnostic.log` records sizes, hashes and timings.

No implementation review or Flow completion is claimed at this checkpoint.
The task remains in progress for metadata remediation and the unchanged final gates.
