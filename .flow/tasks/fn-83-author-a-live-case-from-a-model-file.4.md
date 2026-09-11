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
Blocked 2026-09-10; superseded by fn-85 ("Model side effects as typed interfaces and run query sets").

The per-Case `case` block (one Query, one hand-picked whole-Program template, per-Case evidence lines) is replaced by:

- **Side effects in the Model.** Entities with structured state, and interfaces with a kind (`call`, `command`, `reply`, `observation`), a party, input classes with representatives, and result classes. Request fields that decide the outcome are Model behavior, not binding detail.
- **A Temporal Realization** that binds interfaces, result classes, observations, setup parameters and parties to RPCs, workflow commands, handler replies, history events and dynamic config. The Producer assembles Program and Contract from the witness; whole-Program templates and the `case` command are removed.
- **Query sets per purpose.** A set binds each party to test or environment; a functional set compiles to one Case per Query; canary and exploratory sets are admitted for fn-70/fn-29 and fn-33.

fn-85's final task closes this task as superseded and names where its concern went. Design record: `model/Temporal/Feature/Nexus/DESIGN.md`.
## Evidence
- Commits:
- Tests:
- PRs:

## Work that landed before the block

This task was claimed and implemented before the `case`-abstraction block was recorded, and its
Go-side half is committed in `dda17feda8`. It is left `blocked` rather than `done`, because the
block is a human decision and the redesign may still change what `--list` enumerates.

What is already in the tree, and what the redesign has to reckon with:

- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` no longer carries a functional
  Case table or a fixed entry count. It calls the renderer's `--list`, reads it twice and compares
  (the same determinism discipline it already applied per fixture), and renders each entry by Case
  ID. **This depends only on `--list` printing `<case-id> <fixture-name>` -- not on the `case`
  block, the templates, or per-Case evidence.** A set-per-purpose design that still enumerates the
  Cases it compiles keeps this half unchanged.
- The one functional fixture the registry does not model (`synthetic`) stays named by its own
  renderer argument, in `syntheticEntry()`.
- `tests/testcore/testpilot/fixture_table_test.go` (new) enumerates `testdata/*-case.json` and
  checks decode, identity, Case-ID uniqueness, preparation over unchanged bytes where
  `DeriveProfile` reads the Profile, and rejection of a mutated role. It names no Case, so it
  survives any change to how Cases are authored. The per-Case semantic tests are untouched.
- `runCase(t, env, "async-nexus")` derives its binding from the fixture name and creates a Nexus
  endpoint only when the Case declares one; `runCaseWithBinding` keeps the explicit form. The
  async-Nexus evidence helpers moved to `tests/testpilot_live_case_test.go`.

Verified at that commit: `make umpire-check-case-runtime-conformance` reproduces every checked-in
fixture, `go test -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/...` is green, and
`make umpire-check-live-tests` passes across 9 identities. An impl-review of the same diff on
`claude:claude-sonnet-4-5:high` returned SHIP with no introduced findings.
