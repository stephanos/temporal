---
satisfies: [R2, R3, R4]
---
# fn-230-reproducible-guest-builds-a-system.2 After .1: commit the Temporal guest with go:embed; freshness step in model-check; umpire-guest target; docs

## Description
Commit model/temporal/hosts/go/umpire_temporal_guest.wasm built by guest.sh;
export Guest() []byte via go:embed; remove UMPIRE_TEMPORAL_WASM and Cargo
target path logic for the Temporal guest from the Temporal host tests and
slice targets (umpire-check-model-slice, umbrella) so they use the embedded
guest. Add the guest.sh --check step to make model-check (skip with warning
outside CI when the toolchain is missing, fail in CI) and make umpire-guest
to refresh. Add .gitattributes for *.wasm binary. Update MIGRATE.md,
README.md and the hosts README with the decision. Verify go test in the
Temporal host module on a shell with Rust removed from PATH, a stale-guest
negative control, and the full gate with model-time.

### Execution scope and verification

**Touches:** `.gitattributes`, `.gitignore`, `Makefile`, `MIGRATE.md`, `README.md`, `model/README.md`, `model/DESIGN.md`, `model/GUEST_BUILDS.md`, `model/temporal/hosts/go/**`, `model/temporal/realization/src/lib.rs`, `model/temporal/realization/src/probes.rs`, `model/temporal/realization/src/probes/**`, `model/temporal/realization/tests/probes.rs`, `model/umpire/hosts/go/README.md`, `model/scripts/**`, `tests/testpilot_model_slice_test.go`, `tests/testpilot_model_gaps_test.go`, `tests/testpilot_model_annotations_test.go`
**Quick:** focused embedded-guest and freshness controls; Temporal host Go tests with Rust unavailable on PATH; stale/missing output and local/CI missing pinned-toolchain controls; actual live `make umpire-check-model-slice` and `make umpire-check-model-gaps` against embedded bytes, retaining case, privacy and timeout assertions; scoped changed Go lint/vet and script checks.
**Full gates:** `make model-check`; `UMPIRE_BACKENDS=require make model-check`; `make model-time`.

The read-only caller map is `.flow/tmp/fn2302-embedding-survey.md`. It identifies
the separate live gap loader and its legacy release build prerequisite, in
addition to the slice loader. Replace every active Temporal guest lookup,
including model-check's Temporal env override. Preserve the generic stock,
stateful, caught-panic, overflow and native-report controls, and unrelated
IR-bridge behavior. The no-Rust proof is for embedded Temporal consumers,
not a claim that the Rust authoring gate or legacy bridge needs no Rust.

`model/README.md` owns active Model usage; also record the decision minimally
in root README.md, root MIGRATE.md and the existing generic hosts README.
The Temporal hosts README does not currently exist; a new one is not required.
Keep DESIGN.md below 1,000 lines; update relevant status/proof text in place.
Do not edit frozen archived documentation or add general refactors.

Build failure or stale bytes must never be relabeled as a missing-toolchain
skip. Missing pinned Rust/build prerequisites may warn/skip only with CI
unset; CI must fail. Add executable negative controls at the real gate seam.
Preserve value-free diagnostics, original Go/SDK panic behavior, Model,
ABI, protocol, Realization judgments and all independent goldens. No new
third-party dependency/version. Only Temporal's guest is committed, without
LFS; stock/control guests remain uncommitted. fn-232 retains the separate
macOS/cross-OS certification and does not gate this task.

### Bounded live-gate performance remediation

The conductor admits narrowly scoped reuse of validated, request-independent probe
metadata as part of completing the unchanged live gates. The user's milestone goal
delegates recommendations; this is the conductor's recommended implementation
remediation, not a relaxation of R1-R4 or new Model behavior.

The pre-edit full gap baseline failed at the unchanged 20-minute Go deadline
(Go 1200.527 seconds, Make exit 2). A matched selected live case took 4.21 seconds
with release bytes and 4.29 seconds with guest-profile bytes. The no-server actual
embedded-guest control then measured all twelve Prepare calls at 1.047-1.083
seconds, including repeated calls, while independently decoding two contexts and
checking repeated-output identity and different-context request isolation. Logs,
program and terminal statuses are in `.flow/tmp/fn2302/`; the source evidence is
`.flow/tmp/fn2302-baseline-performance-research.md`. These measurements motivate
removing repeated immutable work, not a new timing threshold or waiver.

- Reuse only successfully validated compiled/default-limit inventory and canonical
  expectation metadata. Never cache actual prepared requests, recipes, live slot
  values, observations or outcomes. Preserve actual-context materialization,
  validation and canonical-prediction agreement on every attempt.
- Preserve eager admission and refusal order, errors and panic behavior. Failed or
  partial initialization must not become successful metadata or a new panic.
  Account for module lifetime across Reset, independent Hosts and native parallel
  free-function tests; do not add cross-run or cross-context mutable authority.
- Keep memory within the existing 64 MiB host limit and retain the existing frame,
  allocation, operation, cleanup and 20-minute aggregate ceilings. No new
  dependency/version, protocol, Model, judgment, disposition or golden changes.
- Pin current-context isolation, repeated calls, Reset/independent-host behavior,
  initialization/error paths and bounded retained memory with focused controls.
  Retain the all-2,238-case two-context native mutation control and exact complete
  live corpus, report golden and privacy canary. Record matched before/after
  no-server diagnostics without a flaky hard timing assertion.
- Checkpoint the embedding/freshness implementation first with its original
  verified guest bytes. Commit the remediation separately. If Rust guest bytes
  change, refresh the embedded asset and renew the two-clean-checkout Linux proof
  with independent initially absent targets at the final source checkpoint;
  record the new sizes/hashes instead of reusing the earlier asset's proof.

All original Quick and full gates, review requirements and preservation constraints
remain in force. No general probe refactor, interpreter change or timeout extension
is authorized by this execution amendment.

The test harness may remove only `#![cfg(not(test))]` from the realization
facade to execute private controls against the real crate modules. Its introduction
at `fe00d08` supplies no test-exclusion rationale. Prefer this one-line test-build
change over source-tree duplication, new public testing APIs or warning
suppressions. First verify actual test collection and retain the genuine failing
reuse control before implementing the fix. Preserve all existing public integration
and exhaustive mutation calls; run newly enabled foundation controls and clippy.
If enabling the crate exposes a genuine boundary or duplicate-implementation
problem, diagnose it rather than suppressing or broadening this exception.
## Acceptance
- [ ] Temporal guest committed and embedded; no env var or target path in Go tests or slice targets
- [ ] go test passes without Rust on PATH
- [ ] Stale guest fails model-check with the summary; umpire-guest refreshes
- [ ] MIGRATE, README, hosts README, .gitattributes updated

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
