# Gomad: functional-suite triage sweep on WASM

## Conversation Evidence

> user: "our goal is definitely to run arbitrary functional tests with gomad (wasm)."
> user: "yes file these specs"

## Goal & Context

Nobody knows how far the cooperative WASM profile is from running the Temporal functional suite, because only one purpose-built test (`TestFrontendSystemInfo` in `tests/gomadfunctional`) has ever been run under it. The `tests` directory holds 116 test files and about 1,250 test functions, most of which stand up a full cluster through `tests/testcore`. The guest build is the slim one from `backend.TemporalGuestBuildTags()`: SQLite is the only linked SQL plugin (`tests/testcore/flag_sql_wasm_slim.go`), ringpop membership is stubbed and requires static service hosts (`temporal/membership_gomad.go`), and the cloud archivers are stubbed (`common/archiver/provider/provider_cloud_gomad.go`). Known suspects before any run: `tests/testcore/test_cluster.go` line 551 allocates real loopback ports through `common/testing/freeport`, and anything reaching `os/exec` or cgo is refused by construction under wasip1.

This spec measures the gap instead of guessing. It runs every test function in `tests` under WASM with a bounded per-test budget and classifies each outcome. The output is a retained support matrix with a reason per failure class. It is explicitly not a support claim: a pass in the sweep is an observation on one candidate, not qualification. The refusal-closing work that follows is filed as separate specs from this sweep's output.

## Architecture & Data Models

**Harness.** A sweep driver, living under `internal/backends/wasm` as a developer command or test-only producer (task .1 decides and records the location, following the `wasmmeasure` precedent of explicit flags and refused inherited variables), that enumerates test functions in `tests`, builds the guest once per package with the slim tags, and runs each function through the existing `gomad explore ... go-test <package> -- -test.run=^Name$` path with one seed, `--parallel=1`, and a per-test execution timeout and overall budget. If the one-command setup spec has not landed, the sweep uses a throwaway configuration derived from the existing integration fixtures and records that fact.

**Classification.** Each outcome maps to exactly one class, derived from the existing `backends.Termination` values and build results:

| Class | Source |
| --- | --- |
| pass | exit termination, test exit code 0 |
| guest failure | exit termination, nonzero test exit code, or trap |
| refused import | unsupported termination naming an import or operation |
| capacity exceeded | capacity termination |
| timeout | watchdog or execution timeout |
| build failure | guest did not compile under the slim tags |
| infrastructure | helper or host failure, retried once, then recorded |

**Artifact.** `.flow/artifacts/fn-<id>-support-matrix-<date>.md` plus a machine-readable sidecar (JSON, shape decided in task .2) listing per test function: package, name, class, reason string, seed, budget, candidate identity (helper digest, compiler, module identity). Aggregates per class and per package, and a ranked list of distinct reason strings with counts.

## API Contracts

- The harness takes explicit flags for checkout, compiler, helper, installation or throwaway config, per-test timeout, overall budget, package filter and output directory; it refuses inherited `GOMAD_*` environment variables.
- The harness never writes under the checkout other than its output directory; retained explore artifacts stay in the operator artifact root.
- The support matrix names the candidate identity it ran on and states in its first paragraph that it is an observation, not qualification.

## Edge Cases & Constraints

- Budget: one seed per test, bounded per-test timeout, and an overall wall budget; tests cut off by the overall budget are recorded as not run, never as failures.
- Order: packages run in a fixed sorted order so a rerun produces a comparable matrix; the sweep is not a determinism check and does not compare repetitions.
- Isolation: each test function runs in its own guest execution; a package-level fixture that cannot start is recorded once per function with the same reason.
- Host load: the sweep is heavy; run on a quiet host and record host identity, as the WASM qualification gates do.
- No native path, no WASM capability change, no slim build tag change inside this spec; discovered refusals become follow-on specs.
- No CI run, PR or push.

## Acceptance Criteria

- **R1:** The harness enumerates every test function under `tests` (count recorded and reconciled against a `go test -list` of the same packages) and runs each through the existing explore path with explicit inputs and bounded budgets. Errors: an inherited configuration variable or an unbounded run is refused.
- **R2:** Every enumerated test function has exactly one class in the retained matrix, with a reason string for every non-pass class and the candidate identity recorded. Errors: a test without a class, or a class without a reason, is a finding.
- **R3:** The report ranks distinct reasons by count, names the known suspects' actual outcome (freeport port allocation, os/exec, cgo, ringpop, cloud archivers, non-SQLite persistence), and recommends follow-on specs per reason class with an estimate of how many tests each would unblock. Errors: a recommendation without a reason mapping is a finding.
- **R4:** The matrix states that it is an observation on one candidate and not a support claim; QUALIFICATION.md is unchanged by this spec. Errors: any support wording is a finding.

## Boundaries

- No fixing of refusals, capacities or guest failures found; those are follow-on specs.
- No qualification change; no change to `internal/backends/wasm/QUALIFICATION.md`.
- No native backend involvement.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): size the gap before closing it. The sweep is ordered before or alongside the one-command setup spec because its output changes what that setup must capture; a throwaway configuration is acceptable for the sweep. Refusal-closing specs are filed from the sweep's reason classes. Host-side execution cost belongs to open spec fn-185 and the determinism contract to open spec fn-188; this spec feeds both with real workload data but does not take over their scope.

## Early proof point

Task .1 runs the harness on `tests/gomadfunctional` alone and on one ordinary `tests` package with a ten-test cap, and shows each function classified. If package-level fixtures cannot be made to start at all under the throwaway configuration, record the reason and stop before the full sweep.

## Quick commands

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/...
# harness invocation decided in task .1; sweep output lands under .flow/artifacts/
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Harness enumerates and runs every test function with explicit bounded inputs. | fn-221-gomad-functional-suite-triage-sweep-on.1 | — |
| R2 | Every test has one class with reason and candidate identity in the retained matrix. | fn-221-gomad-functional-suite-triage-sweep-on.2 | — |
| R3 | Ranked reasons, suspect outcomes and follow-on spec recommendations. | fn-221-gomad-functional-suite-triage-sweep-on.3 | — |
| R4 | Observation only; no support claim; QUALIFICATION.md unchanged. | fn-221-gomad-functional-suite-triage-sweep-on.2, fn-221-gomad-functional-suite-triage-sweep-on.3 | — |
