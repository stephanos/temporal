# fn-182 prerequisite: incremental recording-time inspection evidence

Conductor record, 2026-10-16. Supersedes the point-cache approach in
[the recovery commitment diagnosis](fn182-recovery-commitment-diagnosis-2026-10-16.md)
as the fix for the multi-instance deadline failure. Not yet integrated into `gomad`.

## Cause

fn-180's `State::inspection_snapshot` (`wasmhost/src/engine.rs`) ran on every debug
decision and broker call even with no inspection requested. One seed-7 multi-instance
recording took 84,716 snapshots; ~535 s of ~600 s execute went to inspection:
namespace state/recovery/log rebuilt each call (5.7 MB each, 323 s; namespace changed in
only 172 calls), network evidence rebuilt/hashed 2-3x per broker call (~175 s), World wire
rebuilt 2,067 times at ~17.8 MB (107 s).

## Change (branch `fn182-incremental-inspection`, base `fc40790c11`, HEAD `f0747e8ab6`)

- `2f48073ed4` perf(gomad): record inspection evidence incrementally
- `633bbe42c6` perf(gomad): reuse framed World transitions in the state digest
- `f0747e8ab6` perf(gomad): reuse encoded World transitions in inspection wires

Namespace identity+revision (advanced by every public mutator) reuses commitments while
unchanged; endpoint commitments cached by endpoint identity/revision plus shared-owner
revision; network commitments that are always overwritten before another instance can
observe them are deferred, with an exact cheap `record_check` preserving exhaustion
timing (pre-idle-handoff, boundary and terminal snapshots stay eager); World retains
framed transition encodings for the state digest (cap 256 MiB) and the wire template is
split into head + transitions emitting identical bytes, with the existing 384 MiB limit
applied identically. No format, schema, identity, budget or limit change.

Tests: test-only eager oracle plus threaded differential harness over recorded Go broker
scenarios (storage/no storage, same-length rewrites, crashes, restart incarnations, idle
handoffs, sealing/exhaustion) requiring byte-identical evidence; unit tests for
`record_check` at both encoder bounds, endpoint cache under foreign mutation, namespace
revision, World digest/wire bounds. Mutation checks (drop revision bump, ignore cached
handle, defer pre-handoff hook, drop owner bump) each fail a test.

## Gates (worker-run on HEAD `f0747e8ab6`)

| Gate | Exit |
| --- | --- |
| `cargo +1.94.1 fmt --check` | 0 |
| `cargo +1.94.1 test --release --locked` / debug | 0 / 0 |
| `cargo +1.94.1 test --release --locked --features conformance --bin gomad-wasmhost` | 0 (142) |
| clippy `--all-targets -D warnings` default / conformance / diagnostic-profile | 0 / 0 / 0 |
| `gomadtool protocol-generate -check`; generation/protocol Go test | 0 / ok |
| `make -C tools/gomad wasm-helper` (helper sha `7cf2009e…`) | 0 |
| `make -C tools/gomad test-wasm` (pinned stock Go) | 0 (502 s) |

## Measurements (same host, retained baseline module `35f2d28c…`, seed 7, original scenario, 15 min deadline, all 7 oracles checked; one recording each, sequential)

| Run | Wall | Helper user CPU | Peak RSS | Active execute |
| --- | --- | --- | --- | --- |
| base cold | 737.5 s | 718.3 s | 3.85 GB | 600.8 s |
| new cold | 193.7 s | 188.1 s | 3.85 GB | 76.8 s |
| base warm | 580.9 s | 569.4 s | 1.17 GB | 577.0 s |
| new warm | 75.9 s | 71.0 s | 1.17 GB | 72.2 s |

All four completed with 7 passing oracles; full simulation results (tapes, network,
volumes, World) identical apart from helper identity fields. This is a diagnostic A/B, not
the fn-182 R4 matched measurement.

## Open

- Independent review of the three commits.
- Real qualification cell (`TestActualTemporalClusterQualification` single-cell copy of
  `measure.py`) was started ~10:28 but its result is unknown: the sandbox root filesystem
  went read-only (EROFS) during the run. Rerun after sandbox restart.
- Residual World floor ~56 s/recording (~37 s not incremental without a format change:
  World `stateDigest` and inspection commitment hash length/header-first). Not pursued.
- Deferral relies on broker serialization (fails closed if violated). Up to 256 MiB extra
  worst-case memory for retained digest frames; measured peak RSS unchanged.
- Raw evidence was under `/home/agent/wt/evidence/inspect/` (sandbox overlay; may be lost
  on restart).

## Session state at EROFS (for the next conductor)

- fn-184 candidate `bc569fb2a3` (branch `fn184-integrate`): R1-R4 regated PASS, review
  findings fixed (CI native lint via GOFLAGS). Pending: R5 timing
  (`/home/agent/wt/evidence/fn184/measure.sh prep && AFTER_SHA_EXPECTED=bc569fb2a3 ... run`,
  script may be lost with the overlay; rewrite as plain before/after timing if so),
  full test-wasm, MILESTONES/Flow close.
- fn-189 (upstream merge) added in `2ec633859b`; order fn-182 → fn-184 → fn-189 → fn-187 → fn-185.
