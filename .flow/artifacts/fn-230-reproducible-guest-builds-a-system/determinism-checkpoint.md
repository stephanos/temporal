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
- Head: `405889a77ac5364b53fa71acabc516778b531a80` sorts only the merged file
  list by descriptor name before `compile_fds`. The API suite passed (session 2005):
  the real codegen regression, freshness test and all four descriptor-admission
  controls. Workspace format check and focused API Clippy with `-D warnings`
  passed (session 27637). Full gates and fresh two-clone artifact proof passed;
  exact results and retained evidence paths are below.
- Live: no interactive live surface; actual WASM artifact host parity is separate.

## Fresh checkpoint audits

Both audits were dispatched read-only to fresh `gpt-6.1-sol` contexts at high
effort, one for correctness and one for standards. They are same-family audits,
not full-task formal review; neither ran builds or issued an R1 verdict.

Correctness audit, verbatim:

> Findings: no correctness defects identified in `feb4621d..405889a7`.
>
> - `model/temporal/api/codegen.rs:31` sorts only the merged file list. Message, field, enum-value and dependency-index order within each descriptor remains intact. Prost constructs its message graph before emitting files, so sorting introduces no dependency-order requirement.
> - `model/temporal/api/tests/codegen.rs:17` exercises the production helper against the real upstream closure, comparing every generated filename and byte under opposite file orders. Recorded red failure names `cel.expr.rs`; recorded API and Clippy runs pass after the fix.
>
> Disposition: bounded codegen checkpoint has no blocking findings. Full R1 remains unresolved pending artifact identity, actual WASI caught-unwind judgments and macOS proof. No SHIP/full-task verdict issued.

Standards audit, verbatim:

> No blocking standards or scope findings for `feb4621..405889a`.
>
> The shared helper preserves existing generation settings; the fix sorts descriptor files only, preserving declarations within each file. Recorded RED/GREEN logs demonstrate real generation behavior. Dependencies and protected paths remain unchanged.
>
> Checkpoint follow-up: update `determinism-checkpoint.md:57`, which still says “Head: pending.”
>
> Disposition: checkpoint acceptable on standards; task remains `in_progress`. WASI caught-unwind and macOS/cross-OS proof remain unsatisfied.

The standards follow-up is resolved by the head results immediately above.

Final documentation audit, verbatim:

> No material documentation findings. Current hashes, sizes, toolchain, commands and parity results match retained proof logs; historical evidence remains intact. `DESIGN.md` is exactly 1,000 lines.
>
> Open unwind/macOS obligations remain explicit. Optional cleanup: replace checkpoint “Full gates and fresh two-clone artifact proof follow” with final evidence pointers. No full-task verdict.

That cleanup is resolved above. The subsequent timing results are recorded below.

## Final Linux proof and gates

Proof source pin: `405889a77ac5364b53fa71acabc516778b531a80`. New scratch:
`.flow/tmp/fn2301-determinism-proof-FclJoj/`, with complete `run-proof.sh`, exact
command/timestamp/exit log `terminal-results.log` and detailed `evidence.json`.
Session 84844 exited 0 on 2026-10-21 at 00:58:56 UTC. Both clean clones stayed at
the pin and clean before/after; initially absent targets have distinct identities.

| Actual guest-profile artifact | Bytes | SHA-256 |
| --- | ---: | --- |
| Stock A and B | 143022 | `f01424be60038282bcad227110ba76b6eb34a6cb3cc934cfb7422fa48bb23a9f` |
| Temporal A and B | 2128089 | `b7b9f6f5ab77be5e177cc95001d53fae42d76207c6ca59411f88db9184198e10` |
| Control A | 103754 | `62e593deea35a7e39b562e2e416e68a20bbbef40354109b533ef566995a11647` |

Both comparisons exited 0. Actual stock/control parity passed all 19 top-level
tests (0.875 s), including report/ordered-row native goldens and panic
trap/invalidation controls. Temporal A and B each independently passed all 22
host tests (47.260 s and 47.458 s). Both actual freshness checks exited 0; the
Temporal check rebuilt before comparing. Exact platform/toolchain and reproduction
commands remain in `model/GUEST_BUILDS.md`; the DESIGN summary stays at 1000 lines
and retains the historical smaller proof as historical evidence.

`UMPIRE_BACKENDS=require make model-check` exited 0 (session 4430), with lane
`CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-determinism-fix`, four Cargo
jobs and isolated Go cache `fn2301-determinism-fix-go`. Log:
`.flow/tmp/fn2301-determinism-required-model-check.log`. This includes required
shared source/TLC controls, not Temporal TLC agreement. `make model-time` in the
same lane exited 0 (session 20204): full gate 56.738 s exceeds the 30 s soft
target, edit/test 1.886 s is under 10 s; Temporal Go-host step accounts for
46.277 s. Log: `.flow/tmp/fn2301-determinism-final-model-time.log`.
Final bash syntax and Python exit-code controls passed. No suite was rerun just
to observe its result. Gate classification required full verification; no gate
was skipped. Both gate-receipt writes returned `NO_RECEIPT` because the preserved
external `MILESTONES.md` draft is dirty; actual terminal results are authoritative.

Both original and new proof source clones, targets, artifacts and caches are
retained. The five external dirty paths pass their before/after SHA-256 checks
and were excluded from staging. Only this checkpoint's source, test and evidence
were committed; no remote or Flow lifecycle write occurred.

stage: impl-review - skipped(policy: bounded source checkpoint; unresolved full R1 review/completion belongs to conductor)
stage: checkpoint-audits - ran (fresh correctness and standards contexts; no blocking checkpoint findings)

BLOCKED: EXTERNAL_BLOCKED
Task: fn-230-reproducible-guest-builds-a-system.1
Summary: Actual WASI caught-panic/unwind judgments and macOS/cross-OS proof remain unsatisfied.
Impact: R1/task .1 remains in_progress; task .2 dependency is not satisfied.
Suggested resolution: Provide a supported caught-unwind WASI toolchain/runtime path and reachable macOS executor, or resolve the requirements explicitly with the owner before full-task review/completion.
