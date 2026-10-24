# Gomad: functional-suite triage sweep on WASM

## Conversation Evidence

> user: "our goal is definitely to run arbitrary functional tests with gomad (wasm)."
> user: "yes file these specs"

## Goal & Context

Nobody knows how far the cooperative WASM profile is from running the Temporal functional suite, because only one purpose-built test (`TestFrontendSystemInfo` in `tests/gomadfunctional`) has ever been run under it. The `tests` directory holds 116 test files and about 1,250 test functions, most of which stand up a full cluster through `tests/testcore`. The guest build is the slim one from `backend.TemporalGuestBuildTags()`: SQLite is the only linked SQL plugin (`tests/testcore/flag_sql_wasm_slim.go`), ringpop membership is stubbed and requires static service hosts (`temporal/membership_gomad.go`), and the cloud archivers are stubbed (`common/archiver/provider/provider_cloud_gomad.go`). Known suspects before any run: `tests/testcore/test_cluster.go` line 551 allocates real loopback ports through `common/testing/freeport`, and anything reaching `os/exec` or cgo is refused by construction under wasip1.

This spec measures the gap instead of guessing. It runs every test function in `tests` under WASM with a bounded per-test budget and classifies each outcome. The output is a retained support matrix with a reason per failure class. It is explicitly not a support claim: a pass in the sweep is an observation on one candidate, not qualification. The refusal-closing work that follows is filed as separate specs from this sweep's output.

## Architecture & Data Models

**Harness.** A sweep driver, living under `internal/backends/wasm` as a developer command or test-only producer (task .1 decides and records the location, following the `wasmmeasure` precedent of explicit flags and refused inherited variables), that enumerates test functions in `tests`, builds the guest once per package with the slim tags, and runs each function through the existing `gomad explore ... go-test <package> -- -test.run=^Name$` path with one seed, `--parallel=1`, and a per-test execution timeout and overall budget. If the one-command setup spec has not landed, the sweep uses a throwaway configuration derived from the existing integration fixtures and records that fact.

**Enumeration and compile reuse.** Test functions include registered suite methods, including nested suite registrations. Preserve the mapping from each method to its top-level wrapper and exact `-test.run` path; require the selected method's terminal verdict rather than accepting the wrapper's pass. Reconcile top-level functions with `go test -list`, and reconcile method declarations against the registration inventory with explicit exclusions. A single global deadline bounds enumeration, preparation, retries and execution. To fulfill the existing build-once requirement, the backend may reuse an authenticated module by compile-only inputs while preserving the full per-execution provenance and BuildKey. Runtime arguments and configuration must never be copied from a different cached execution. Tests must prove selector reuse avoids compilation, compilation inputs invalidate reuse, and different runtime inputs retain distinct execution provenance.

**Classification.** Each executed outcome maps to exactly one class, derived from the explore event stream, existing `backends.Termination` values, retained artifact outcome and build results:

| Class | Source |
| --- | --- |
| pass | exit termination, test exit code 0 and the selected function's verbose PASS; an explicit selected SKIP is marked skipped and reported separately |
| guest failure | nonzero test exit code, soft-assertion failure or trap |
| refused import | preparation or execution unsupported refusal naming an import or operation, with the phase retained in the reason |
| capacity exceeded | capacity termination, World/choice-trace overflow or artifact/journal capacity |
| timeout | watchdog, execution timeout or that function's explore overall timeout |
| build failure | guest compilation failure under the slim tags or the function outside the wasip1 guest file set, with the diagnostic or guest-file-set reason |
| infrastructure | helper or host failure, retried once, then recorded |
| replay divergence | observed replay, backend or World divergence; reported to fn-188 as a determinism finding |

Budget/cap cuts are recorded as `not run` with a reason and retried on resume; they are never failures or successful executions. A missing selected verdict or `no tests to run` is never a pass. The method-aware sidecar is `gomad-wasm.sweep/v2`; retained wrapper-only v1 proofs cannot establish complete method coverage.

**Artifact.** `.flow/artifacts/fn-<id>-support-matrix-<date>.md` plus a machine-readable sidecar (JSON, shape decided in task .2) listing per test function: package, name, class, reason string, seed, budget, candidate identity (helper digest, compiler, module identity). Aggregates per class and per package, and a ranked list of distinct reason strings with counts.

## API Contracts

- The harness takes explicit flags for checkout, compiler, helper, installation or throwaway config, per-test timeout, overall budget, package filter and output directory; it refuses inherited `GOMAD_*` environment variables.
- The harness never writes under the checkout other than its output directory; retained explore artifacts stay in the operator artifact root.
- The support matrix names the candidate identity it ran on and states in its first paragraph that it is an observation, not qualification.

## Edge Cases & Constraints

- Budget: one seed per test, bounded per-test timeout, and an overall wall budget; tests cut off by the overall budget are recorded as not run, never as failures.
- Each function records choices under a 64 MiB bound and uses `-test.timeout=72h`, matching the existing Temporal consumer workflow. Host per-test, explore and global watchdog budgets remain enforced and unchanged; individual test contexts are not altered. No choice-guided exploration or coverage claim is added.
- A bounded package warm-up selects no test and charges engine compilation to preparation before timed test execution. Retain the package's authenticated compilation and engine caches until its functions finish; each function still starts a fresh guest.
- Guest output is bounded at 256 MiB. Report the effective transcript capacity (currently explore's 64 MiB default), rather than borrowing the configuration's larger limit. Do not pass testcore persistence flags to unrelated packages; record their existing SQL/SQLite defaults.
- Order: packages run in a fixed sorted order so a rerun produces a comparable matrix; the sweep is not a determinism check and does not compare repetitions.
- Isolation: each test function runs in its own guest execution; a package-level fixture that cannot start is recorded once per function with the same reason.
- Host load: the sweep is heavy; run on a quiet host and record host identity, as the WASM qualification gates do.
- No native path, no WASM capability change, no slim build tag change inside this spec; discovered refusals become follow-on specs.
- No CI run, PR or push.

## Acceptance Criteria

- **R1:** The harness enumerates every test function under `tests` (count recorded and reconciled against a `go test -list` of the same packages) and runs each through the existing explore path with explicit inputs and bounded budgets. Errors: an inherited configuration variable or an unbounded run is refused.
- **R2:** Every enumerated test function has exactly one row in the retained matrix, with one of the eight execution classes or the explicit not-run state, a reason for every non-pass state and the candidate identity recorded. Errors: missing/duplicate rows, unknown classes, missing reasons or identity fields, or a row key inconsistent with its recorded identity are findings. The verifier checks the method-aware v2 inventory and candidate-key binding; v1 wrapper evidence does not satisfy this requirement.
- **R3:** The report ranks distinct reasons by count, names the known suspects' actual outcome (freeport port allocation, os/exec, cgo, ringpop, cloud archivers, non-SQLite persistence), and recommends follow-on specs per reason class with an estimate of how many tests each would unblock. Errors: a recommendation without a reason mapping is a finding.
- **R4:** The matrix states that it is an observation on one candidate and not a support claim; QUALIFICATION.md is unchanged by this spec. Errors: any support wording is a finding.

## Boundaries

- No fixing of refusals, capacities or guest failures found; those are follow-on specs.
- No qualification change; no change to `internal/backends/wasm/QUALIFICATION.md`.
- No native backend involvement.
- No CI run, PR or push.

## Decision Context

Conductor decision (2026-10-20), under the owner's instruction to follow recommendations autonomously: wrapper-only classification does not satisfy the per-function matrix, and recompiling each argument-dependent preparation does not satisfy building once per package. Task .1 therefore includes method/registration discovery and a backend compilation-cache seam, preserving the CLI, recorded provenance, capabilities, model semantics and refusal behavior. This is the implementation needed by the existing acceptance criteria; it does not reduce the sweep scope.

The same decision makes replay divergence explicit rather than misclassifying it as a guest failure, and records skipped/no-test/budget-cut dispositions separately. The 72-hour Go global virtual timeout matches the documented consumer invocation; host watchdogs retain the original execution bounds. Nested modules under `tests` are included rather than silently omitted by the parent module's `go list ./tests/...`.

Owner decision (2026-10-19): size the gap before closing it. The sweep is ordered before or alongside the one-command setup spec because its output changes what that setup must capture; a throwaway configuration is acceptable for the sweep. Refusal-closing specs are filed from the sweep's reason classes. Host-side execution cost belongs to open spec fn-185 and the determinism contract to open spec fn-188; this spec feeds both with real workload data but does not take over their scope.

## Early proof point

Task .1 runs the harness on `tests/gomadfunctional` alone and on one ordinary `tests` package with a ten-test cap, and shows each function classified. If package-level fixtures cannot be made to start at all under the throwaway configuration, record the reason and stop before the full sweep.

## Quick commands

```sh
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad sweep
cargo run --manifest-path tools/gomad/rust/Cargo.toml -p gomad --release --bin gomad-wasm-sweep -- --help
# sweep output lands under .flow/artifacts/fn221-*; branch fn221-rust-sweep carries the .2 launch plan
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Harness enumerates and runs every test function with explicit bounded inputs. | fn-221-gomad-functional-suite-triage-sweep-on.1 | — |
| R2 | Every test has one class with reason and candidate identity in the retained matrix. | fn-221-gomad-functional-suite-triage-sweep-on.2 | — |
| R3 | Ranked reasons, suspect outcomes and follow-on spec recommendations. | fn-221-gomad-functional-suite-triage-sweep-on.3 | — |
| R4 | Observation only; no support claim; QUALIFICATION.md unchanged. | fn-221-gomad-functional-suite-triage-sweep-on.2, fn-221-gomad-functional-suite-triage-sweep-on.3 | — |

## Amendments (2026-10-24, conductor)

1. **Harness owner.** The harness is the Rust `gomad-wasm-sweep` binary (`tools/gomad/rust/gomad/src/sweep/`), not a Go developer command under `internal/backends/wasm`. The `gomad explore … go-test` path it drives is the Rust verb. fn-220 has landed, so the sweep uses its installation; the throwaway-configuration clause is moot.
2. **Full rerun on the Rust binary (MILESTONES ruling 9).** Task .2's 2026-10-22 rerun of the Go-candidate journal at `1d7b37f00a` is not resumed. The Go host's 337 helper-link-count infrastructure rows and changed evidence bounds make that journal non-transferable, so the whole sweep reruns on `gomad` after the fn-232.13 cycle. Before launching, investigate the historical batch-test slowdown versus Go and exercise resume/verify on two packages, as the `fn221-rust-sweep` handover asks.
3. **Load-only failures.** Under the 2026-10-23 performance decision, rows that fail only from host load (timeouts, deadline-sensitive inspection fixtures) are classified as contention in the matrix and rechecked in the quiet-host window. The "quiet host" wording in Edge Cases is a recording requirement, not a start condition. Guest outcomes on the modeled clock are deterministic and stand as observed.
4. **Still an observation.** R4 is unchanged: no row of the matrix becomes a support claim; fn-222's cycle-published list is the only support document, and it is provisional until fn-242.
