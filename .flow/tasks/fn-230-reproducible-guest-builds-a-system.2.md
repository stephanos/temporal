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

**Touches:** `.gitattributes`, `.gitignore`, `Makefile`, `MIGRATE.md`, `README.md`, `model/README.md`, `model/DESIGN.md`, `model/GUEST_BUILDS.md`, `model/temporal/hosts/go/**`, `model/umpire/hosts/go/README.md`, `model/scripts/**`, `tests/testpilot_model_slice_test.go`, `tests/testpilot_model_gaps_test.go`, `tests/testpilot_model_annotations_test.go`
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
