# Temporal guest determinism checkpoint

Task `fn-230-reproducible-guest-builds-a-system.1` remains in progress. This checkpoint
addresses the current Linux code-generation defect; actual WASI caught-panic/unwind
judgments and macOS/cross-OS identity remain unsatisfied. It is not an R1 completion
or review verdict, and does not satisfy task .2's dependency.

Tier: session (jev-unavailable(no_key))

## Authorized scope exception

The conductor explicitly authorized `model/temporal/api/build.rs` and the minimum
generation helper/tests under `model/temporal/api/**`, plus reuse of already-present
workspace dependencies in its Cargo manifest. The exception allows canonical file
ordering at the consuming API generation boundary. It does not allow reordering
messages, fields or enum values within a descriptor, editing upstream descriptors
or `cmd/tools/getproto`, or changing Model, goldens, ABI, protocol or Realization.
`prost-build` is reused as a dev dependency without a new dependency/version/lock change.

## Pre-edit evidence

Base: `feb4621de85e1889f4dd071ef3875b97a2ad8d22`. Script syntax and guest builder
exit controls passed. Fresh isolated `make model-check` passed (session 30882).
`make model-time` exited 0 (session 67030): 76.864 s full gate, 5.845 s edit/test;
the full gate exceeded its 30 s soft budget, with 67.886 s in Temporal Go-host tests.
Receipt checks required execution because the external `MILESTONES.md` draft is dirty.

The original two-clean-clone proof failed before this checkpoint's edits. Stock
matched at 143022 bytes, SHA-256
`f01424be60038282bcad227110ba76b6eb34a6cb3cc934cfb7422fa48bb23a9f`.
Temporal A/B each had 2128089 bytes but differed at zero-based offset 26293:
A `10d3e0af5c4f805bf44b3622d63c9da83a4356000fa90a5357d8476b0430e6c2`,
B `5a83e093fd50a410f61cf1e2089e717389c45bdee4d44e1f6c44dbad1045935c`.
All four builds succeeded; both real Temporal artifacts passed their 22-test host
suites. Those runtime passes did not prove byte identity. Immutable original
commands, artifacts and evidence remain in `.flow/tmp/fn230-current-proof-mgeeue/`.

## Defect route

- Prior fixes: local recent/all-branch history found no ordering fix. The original
  generation path was introduced by `fe00d08becc0d1c73413244acf289ff42a86a5fa`.
  Bug memory search found no matching fix. GitHub open PR/issue checks were unchecked
  because both returned HTTP 401 (bad credentials).
- Diagnosis: the original WASM code/data sections and generated same-package Rust
  block order differed; no path strings explained the mismatch. `getproto` ranges
  a Go map to emit descriptor files. The real-codegen regression reverses only file
  order and fails because generated `cel.expr.rs` bytes change, confirming the
  file-order mechanism without mocking prost or scraping source.
- Introduced by: the API closure consuming unordered file inputs began in
  `fe00d08becc0d1c73413244acf289ff42a86a5fa`; bisect skipped because the older
  small API proof predates the current multi-file/package input and supplies no
  known-good revision for the same regression fixture.
- Base: regression command `mise exec -- cargo test -p umpire-temporal-api --test
  codegen --locked` exited 101 with one behavior assertion failure. Logs:
  `.flow/tmp/fn2301-determinism-red.log`. The unchanged generation path was
  extracted solely to share it between build.rs and this regression.
- Head: pending the minimal fix and verification.
- Live: no interactive live surface; actual WASM artifact host parity is separate.
