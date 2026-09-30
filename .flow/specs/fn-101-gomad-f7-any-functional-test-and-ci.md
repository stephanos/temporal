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
to fit a 90-minute CI budget. Linux host-clock escapes are pinned statically instead of traced:
on linux/amd64 the runtime reads the clock through the vDSO, which seccomp and ptrace cannot
observe, and the interception is platform-neutral Go that the darwin DTrace audit already
exercises, so what differs per platform is who reaches the host clock without passing through it.

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
- **R2:** The named smoke selection of functional tests (defined in task .4, drawn from the
  representative set) shows `unsupported == 0`, `failed == 0`, `infrastructure_errors == 0` on
  darwin/arm64, and every divergence found so far in `./tests` suites is fixed in Gomad or
  demonstrated to be a test bug with a named, owned exclusion. The full `./tests` set is not run
  as a gate (2026-09-29 user decision: not scalable).
- **R3:** A smoke set of selected functional tests passes on linux/amd64 in CI with the same zero counts (`unsupported == 0`, `failed == 0`, `infrastructure_errors == 0`). The full `./tests` set is not run in CI.
- **R4:** A required CI check runs on pull requests touching `tools/gomad3`, `tests`,
  `tests/testcore`, `go.mod`, and the closure's server packages. It is a smoke test: it
  qualifies a small, named selection of functional tests (drawn from the representative set)
  on both platforms and fits well inside 90 minutes. The full `./tests` set stays an on-demand
  local gate (`make gomad3-tests-qualification`).
- **R5:** A static host-clock inventory in the toolchain tier pins every standard-library reference
  to `nanotime1`, `walltime`, `time_now`, and the vDSO clock symbols on each qualified platform,
  each classified (implementation, guarded, host-by-design, escape with a finding, unrelated), and
  asserts that `nanotime` and `time_runtimeNow` return on `gomadEnabled` before the host call. Any
  new, removed, or recounted reference fails `make test-toolchain` on either host. The darwin DTrace
  audit stays. A dynamic Linux audit (seccomp with the vDSO disabled in the fixture) is recorded as
  a follow-up if a Linux-only escape is ever observed.
- **R6:** A newly added test in `./tests` appears in the generated manifest (and so has a disposition) without a manual edit; the staleness check fails otherwise.

## Boundaries
<!-- scope: business -->

- Simulation-track work, multi-P scheduling, deterministic GC research (out of scope per the
  milestone document).

## Decision Context
<!-- scope: both -->

2026-09-29: the full `./tests` set is no longer run for anything — not in CI and not as a local gate; validation is the smoke selection plus the suites a change affects (user decision: not scalable). The generator keeps every test enumerated with a disposition.

2026-09-28: CI runs only a smoke test on selected functional tests (user decision); the full enumeration remains the local proof for R2 and the manifest generator still covers every test.

Generating the manifest makes coverage a property of the check rather than of curation.


