# Task 2 metadata reuse and renewed guest proof

Source freeze checkpoint for static audits, not a review verdict or task completion.
Full live gaps and all three full Model gates are still pending at this checkpoint.

## Reproduction and remediation

The complete pre-edit gaps baseline failed at the unchanged 20-minute Go timeout:
Go 1200.527 s, Make exit 2. The earlier outer-600-second run was inconclusive.
Matched selected live diagnostics took 4.21 s with release bytes and 4.29 s with
guest-profile bytes. Their filtered parent aggregate checks deliberately failed;
neither diagnostic is a passing full gate. Twelve no-RPC PREPARE calls, including
warm calls, each took 1.047–1.083 s with the original embedded guest.

Commit `cb3adae845b9340b3c47c8e245be590178fcc075` pins the genuine private
inventory-reuse RED: one real crate test collected and failed on distinct live
metadata allocation identities, exit 101. Two existing foundation tests were
filtered. The earlier zero-test observation is inconclusive; the abandoned broad
source-inclusion harness is not the accepted regression proof. Removing only
the realization facade's crate-wide test exclusion enables genuine private tests.

Commit `4c0fea3b5de1bc6e02f8d15503c9a636811581d5`, integrated normally by
`061bfbe8e7`, publishes only complete successful immutable inventories and lazy
canonical expectation metadata. Request limits and eager refusal order remain;
actual-context materialization, validation and canonical agreement still execute
per attempt. No request bytes, recipes, live slots, observations or outcomes are
cached. Initialization errors, partial work and panics publish nothing. Native
parallel callers share a completed immutable publication without widened error
traits. Canonical validation remains at its original call sites.

Final lane controls: 6 real unit tests, including both newly enabled foundation
tests, and 12 public integration tests passed, exit 0. All original 11 public
controls are unchanged, including all 2238 generic mutations across two contexts.
The added public test checks parallel repeated actual-context requests and invalid
current baselines. Private controls cover identity, failure/panic retry, concurrent
first publication and lazy canonical validation/reuse. Clippy all-targets with
`-D warnings`, scoped Rust formatting and diff checks passed without suppressions.
The lane was retired normally after evidence archival and integration.

## Refreshed bytes and actual host controls

`make umpire-guest` regenerated the asset through the unchanged generic builder.
Committed source/asset pin: `594c13b237b3e3e5defcd67e7457257fa1db21ae`.
Temporal: 3326027 bytes, SHA-256
`1d52b188f7f582a3d1b78d5ed19f4ba8af4753bf43343aff0524854f1d713800`.

The actual guest instantiates under the unchanged host import allowlist, interpreter
and 64 MiB ceiling. Matched no-RPC diagnostics retained all twelve complete response
hashes and sizes. First PREPARE took 1.102 s; warm valid-start preparation took
about 50 ms and selected generic preparation 1.28–5.14 ms. These observations
have no hard timing assertion. Actual two-host/current-context/Reset controls pass,
with 52 retained pages per warmed host, stable across repeats. The full Temporal
host module passed in 9.534 s using absolute Go with cargo/rustc absent from PATH,
automatic Go toolchain switching disabled and guest-path overrides unset.
Changed-Go lint, integration-tag lint and Temporal host vet passed.

## Renewed Linux proof

The proof ran 08:36:24–08:38:13 UTC on 2026-10-21, exit 0, source pin above.
Linux 7.0.14/aarch64; Rust/Cargo 1.99.1, compiler commit
`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1; Go 1.27.0; protoc 29.5.
Two different-path clean clones used initially absent independent targets with
identities `58:453392182` and `58:453408168`. Both source statuses remained clean.
Each Temporal output matched that clone's committed embedded asset and the main
asset; both output pairs compared byte-identically. Stock remains 192255 bytes,
SHA-256 `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1`.
This renews the Linux proof for changed Rust bytes, not macOS/cross-OS certification.
That separate proof remains open under fn-232.

Exact runner, regular logs, statuses, no-RPC source and archived lane evidence are
under `.flow/tmp/fn2302/`; clone/target/output proof is retained at
`.flow/tmp/fn2302-reproduction-proof.JrgaIS/`. Commands are documented in
`model/GUEST_BUILDS.md`. Foreign milestone/fn-233 plans remain excluded from commits.
