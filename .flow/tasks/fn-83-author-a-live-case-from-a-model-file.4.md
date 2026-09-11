---
satisfies: [R4]
---
# fn-83-author-a-live-case-from-a-model-file.4 Generator reads the registry; artifact and live-test helpers

## Description
Delete the Go generator's functional Case table and its fixed-count guard and drive it from `umpire-case --list` (R4); make the shared artifact test table-driven over the testdata directory; give `runCase` a default binding derived from the fixture name; and lift the async-Nexus evidence helpers into the shared live-test file so a new live test is one function of Verdict assertions. Depends on .7 because both edit the shared live-test file and the Makefile.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` (functional table and count guard removed, `--list` consumed), its tests, `tests/testcore/testpilot/artifact_test.go` (table-driven decode/prepare/identity), `tests/testcore/testpilot/README.md`, `tests/testpilot_run_case_test.go` (default binding), `tests/testpilot_live_case_test.go` (lifted helpers), `tests/testpilot_async_nexus_case_test.go`, `Makefile` targets that name the renderer
**Touches:** [tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/artifact_test.go, tests/testcore/testpilot/README.md, tests/testpilot_run_case_test.go, tests/testpilot_live_case_test.go, tests/testpilot_async_nexus_case_test.go, Makefile]

### Approach
- Replace `functionalManifest()` with a call to the renderer's `--list`, parsing `<case-id> <fixture-name>` lines; drop the guard that requires exactly six functional entries; keep `renderLeanCase`/`renderExecutable`. The conformance builder with its expected Verdicts stays as is and keeps naming synthetic and conformance Cases by renderer argument. The generator already renders twice and compares; keep that as the determinism assertion for `--list` order.
- Artifact tests: one table-driven test walks `testdata/*-case.json`, decodes strictly, derives a Profile via `DeriveProfile` where the Case admits it (typed fixtures use their existing builders and are skipped by the table), prepares without Driver I/O, and pins identity. The per-Case semantic tests (outage Deadline, typed tenfold load, run isolation, checked Provenance) stay as their own small functions; do not fold them.
- `runCase(t, env, name)` derives `Identity`, `Namespace`, `TaskQueue`, and `NexusEndpoint` from `name` (for example `umpire-<name>`, `umpire-<name>-queue`, `umpire-<name>-endpoint`) and creates the endpoint only if the Case declares an endpoint binding; keep an explicit-binding variant for tests that vary bindings.
- Move `requireCorrelatedNexusHistoryEvidence`, `requireScheduledNexusEndpoint`, and `requireRunHasOutcome` to `testpilot_live_case_test.go`, on top of .7's provisioning change to that file.
- Go conventions: `require` only, `-tags test_dep` (plus `integration` for live tests), no `time.Sleep`.

### Investigation targets
**Required:**
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:121, 281, 291-296, 350-383, 467-511` — flag set, the six-entry guard, double-render check, functional table and its consumers, conformance builder that stays
- `tests/testcore/testpilot/artifact_test.go:30-336` — the existing shared tests to fold into the table
- `tests/testcore/testpilot/{worker_outage,typed_unary,typed_nexus}_artifact_test.go` — per-Case semantic tests that stay
- `tests/testpilot_run_case_test.go:14-58` — `CaseBinding`, `bindCase`, `runCase`
- `tests/testpilot_async_nexus_case_test.go:125-206` — helpers to lift
- `Makefile:95, 597-618, 669-695` — renderer name, generation/check targets, live-test gate

### Key context
- Memory: conformance tests must not import functional adapters; keep `common/testing/testpilot` free of `tests/` imports.
## Acceptance
- [ ] `generate.go` contains no hand-written functional Case list and no fixed entry count; `make umpire-gen-case-runtime-conformance` reproduces every checked-in fixture
- [ ] One table-driven artifact test covers decode, prepare, and identity for every fixture; per-Case semantic tests remain and pass
- [ ] `runCase(t, env, "async-nexus")` runs with no explicit binding; the async-Nexus live test uses lifted helpers
- [ ] `go test -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/...` and `make umpire-check-live-tests` pass
## Done summary
Blocked:
Blocked 2026-09-10 pending a redesign of the `case` abstraction (decided with the user, not yet a spec).

The per-Case `case` block (one Query, one hand-picked realization template, per-Case evidence lines) is being replaced by:

- **Sets per purpose.** A developer declares query sets by kind: functional and canary sets list Queries explicitly; exploratory sets state a coverage goal and a budget over a variation space.
- **One Case per Query.** A set compiles to many Cases run together; "Case = one Program + one Contract" stays.
- **A separate Temporal binding.** Runtime metadata (how an Action is caused, which recorded event confirms a step result, which resources a role needs) lives in a Temporal-owned binding declaration beside the behavioral Model, so Programs and Contracts are assembled from the Model plus its binding rather than from a whole-Program template chosen per Case.

This task builds on the `case` block, whole-Program templates, per-Case evidence, or the Case-registry shape that redesign replaces. Unblock or rewrite it once the redesign spec exists.
## Evidence
- Commits:
- Tests:
- PRs:
