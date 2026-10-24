---
satisfies: [R11, R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.18 Authenticate Go packages without test files in static inventory

## Description
Size: S

**Touches:** `tools/gomad/rust/driver/static_stages.py`, `tools/gomad/rust/driver/ordinary_receiving.py`, `tools/gomad/rust/driver/tests/test_static_stages.py`, `tools/gomad/rust/driver/tests/test_original_primary.py`

C99 on 69fcf11c13 reached the real Go model command in validate. Both listing command-0017 and execution command-0018 exited 0. All 543 listed roots ran and passed twice. Four packages without test files emitted Go-owned package-level skip events (no Test field), with the standard `[no test files]` output. The unchanged verify_go applies its test/subtest skip refusal to those package events and aborts the stage.

Retained raw inputs: `/home/agent/wt/evidence/fn232-split-closure/resume-closure-cycle1/cycle/root/stages/validate/static/command-0017.stdout`, `command-0018.stdout`, their command/process journals, and the parent producer.stderr. Preserve them unchanged; they are failed-cycle history, not a qualification pass.

Distinguish authenticated no-test packages from skipped tests using the Go-owned listing and actual execution evidence. Tie the permitted empty-package set to the listing's package identities and exact no-test status/output. Do not silently ignore arbitrary package skips or infer inventory from observed passes. Producer and independent authenticator must reconstruct and validate the same scope. Such packages supply no test execution credit. Preserve Go-owned selection, platform/build flags, repeats, missing/duplicate execution checks, strict real test/subtest skips and source-bound existing exclusions.

The ordinary_receiving.collect recursive selection consumes the same RETAINED packages and calls the shared parser/verifier. Carry the authenticated package scope into this caller too, preserving deliberate namespace filtering and real-test credit. The same cause must not survive into a later cycle stage.

Required reads: spec R11/R12; static_stages.py Go listing/inventory/verification, Execution.go and authenticate_result; ordinary_receiving.py selectors/collect; their focused tests in test_static_stages.py and test_original_primary.py; retained C99 raw listing/run and command receipts. Follow surrounding data/receipt conventions and add no dependencies or new lanes.

Focused validation: test_static_stages.py and test_original_primary.py, including a failing reproduction before the fix. Reopen the retained C99 outputs with the corrected inventory/verification logic as a diagnostic only; do not edit C99 output or claim it qualified. Full integrated mandatory gate and actual fresh cycle remain conductor-owned.
## Acceptance
- [ ] A regression reproduces C99's refusal when an authenticated listing/run contains a real tested package plus a package with no test files; the corrected producer and independent reader accept the genuine zero-test package without crediting it as an executed test.
- [ ] Package-level skips require the matching Go listing's package identity and exact no-test classification; unknown packages, changed reasons, duplicate/missing package terminal records and a package with listed tests cannot become accepted empty-package evidence.
- [ ] Skipped real roots/subtests still refuse unless the unchanged source-bound exclusion applies; failed packages, missing/duplicate roots, repetitions and command-receipt tampering remain strict refusals.
- [ ] Restricted/unrestricted selector tests and the focused static suite pass. The retained C99 listing/run can be verified diagnostically without changing any historical files. Task .13 retains its full 96-stage cycle/evidence acceptance.
- [ ] The ordinary recursive receiving path carries the authenticated empty-package scope into the shared verifier while preserving namespace filtering and real-test credit; its caller regression and existing selector tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
