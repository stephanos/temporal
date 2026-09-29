# Gomad F7: any functional test, and CI

## Goal & Context
<!-- scope: business -->

Milestone F7 of `.plans/GOMAD_MILESTONES.md`. The whole `./tests` package is enumerated in the
qualification set, every test has a disposition, the unsupported count is zero on darwin/arm64
and linux/amd64, and the Temporal qualification becomes a required CI check.

## Architecture & Data Models
<!-- scope: technical -->

The manifest is generated from `go test -list` output so a new test lands with a default
expectation of `qualified`. The set is split with `gomad plan` / `execute-shard` / `gomad merge`
to fit a 90-minute CI budget. Linux still needs a host-clock escape audit to replace DTrace.

## Edge Cases & Constraints
<!-- scope: technical -->

- "Any test" means unsupported count zero, not expectations met; the report prints `supported`,
  `unsupported`, `failed`, `infrastructure_errors` separately and the gate reads
  `unsupported == 0`.
- A test that stays unsupported is fixed upstream or excluded by name with an owner and a date;
  silent skips are refused.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A generator produces the `./tests` qualification manifest from `go test -list`, and a
  check fails when the manifest is stale.
- **R2:** The Temporal qualification-set report for `./tests` shows `unsupported == 0`,
  `failed == 0`, `infrastructure_errors == 0` on darwin/arm64.
- **R3:** A smoke set of selected functional tests passes on linux/amd64 in CI with the same zero counts (`unsupported == 0`, `failed == 0`, `infrastructure_errors == 0`). The full `./tests` set is not run in CI.
- **R4:** A required CI check runs on pull requests touching `tools/gomad3`, `tests`,
  `tests/testcore`, `go.mod`, and the closure's server packages. It is a smoke test: it
  qualifies a small, named selection of functional tests (drawn from the representative set)
  on both platforms and fits well inside 90 minutes. The full `./tests` set stays an on-demand
  local gate (`make gomad3-tests-qualification`).
- **R5:** A Linux host-clock escape audit replaces DTrace for linux/amd64.
- **R6:** A newly added test in `./tests` appears in the report without a manual manifest edit.

## Boundaries
<!-- scope: business -->

- Simulation-track work, multi-P scheduling, deterministic GC research (out of scope per the
  milestone document).

## Decision Context
<!-- scope: both -->

2026-09-28: CI runs only a smoke test on selected functional tests (user decision); the full enumeration remains the local proof for R2 and the manifest generator still covers every test.

Generating the manifest makes coverage a property of the check rather than of curation.

