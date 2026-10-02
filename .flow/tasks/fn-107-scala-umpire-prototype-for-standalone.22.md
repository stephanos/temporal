---
satisfies: [R6, R7, R10]
---
# fn-107-scala-umpire-prototype-for-standalone.22 Generate lowered Case files and run them with one generic live runner

## Description
**Touches:** [model/scalav2/run.sh, model/scalav2/goir/testpilot/**, model/scalav2/cases/**, model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/**, model/scalav2/lifter/**, model/scalav2/ir/**, model/scalav2/goir/load.go, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/README.md, model/scalav2/SEMANTICS.md, Makefile, tests/testcore/testpilot/**, tests/testpilot_scala*_test.go, tools/umpire/cmd/umpire-run/**]

Owner's direction (2026-10-01): lowered Cases are generated files run by a generic runner, as the checked-in `*-case.json` Cases already are; no Go test is written per scenario. Task 9 (and task 10's live scenario) wrote hand-made Go tests with per-Query branches and assertions (`tests/testpilot_scala_{activity,nexus}_test.go`); this task replaces them.

**Size:** M

### Approach
- **Generate.** `make umpire-gen-scala` writes every Case that lowers from the checked-in IR as a canonical Case file (a directory such as `model/scalav2/cases/`, one file per Query, deterministic bytes), with a manifest of every Query's standing (lowered, nothing-to-realize, no-realization, unsupported with its located reasons). `make umpire-check-scala` fails when a file or the manifest is stale.
- **Expected results are declared, not coded.** What a live Run of a Case is expected to conclude beyond a satisfied Contract (the model assessment: conformance and the Query's Property as satisfied or inconclusive, with the reason) is declared on the Scala side and generated next to the Case; Go holds no per-Query expectation.
- **One generic live runner.** A single table-driven live test discovers the generated Cases and runs each through the existing bind, Run, Evaluate and `WithAssessment`/`conformance.Prepare` path on the in-process cluster, on independent resources, twice concurrently, and compares the Verdict and the assessment with the generated expectation, live and replayed. No Query name appears in Go. `umpire-run` runs the same files against an external endpoint where a Case needs nothing the harness alone provides.
- **Delete** the per-Query Go tests and their per-Query assertions. If an assertion they made is not implied by the Case's Contract, that is a gap in the Scala declaration or the lowering: fix it there or record it, do not keep it in Go. The canary harness seam tests stay (they test the boundary, not a feature) but read the generated completion Case file.
- Capabilities a Case needs that a consumer lacks (the canary profile and hold-delivery) are rejected before I/O and reported by the runner as skipped-with-reason, from the Case's own requirements.

### Quick commands
`GOFLAGS=-tags=test_dep make umpire-gen-scala`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `cd tests && go test -tags 'test_dep integration canary_harness' -run '^TestTestpilotScala' -count=1 .`; `go test -tags test_dep ./model/scalav2/... ./tests/testcore/testpilot/... ./tools/umpire/...`.

## Acceptance
- [ ] Every lowerable Query of the checked-in IR has a generated, checked-in Case file with deterministic bytes, and a manifest names every other Query's standing; the check gate fails on a stale file.
- [ ] One generic live test runs every generated Case with no Query-specific Go code, on independent resources, and compares Verdict and model assessment, live and replayed, with expectations generated from Scala declarations.
- [ ] `tests/testpilot_scala_activity_test.go` and `tests/testpilot_scala_nexus_test.go` and their per-Query assertions are gone; anything they asserted is implied by a Contract or recorded as a gap. The canary seam tests use the generated Case file.
- [ ] Task 10's hold-delivery scenario runs through the same generic runner, and a consumer lacking a required capability reports it before I/O.
- [ ] Existing live Testpilot tests and the Scala gates pass.


## Done summary
Implemented generated canonical Cases, Scala-declared Run expectations, and one generic live runner. The version-1 strict manifest accounts for all 260 Queries: 14 lowered, 148 verify-only, 95 without realization, and 3 unsupported. Generation checks deterministic bytes and publishes a fully staged managed tree with locking and rollback. Unknown manifest fields/versions, duplicate Queries/paths, invalid expectations, stale/missing/obsolete files, and symlink drift have focused rejection tests.

The generic runner executes all 14 Cases (6 activity, 1 held-delivery, 7 Nexus) through both Nexus implementations, with two independent bindings and two concurrent rounds each. It checks Contract, cleanup, every Scala-declared assessment, recorded faults, replay equality, immutable source bytes, and declaration-driven missing-durable-evidence ambiguity. The two handwritten per-Query suites are removed. Canary seam tests and the separate umpire-run process consume generated files. CLI capability admission now precedes opening a target; the negative delivery-control test proves opener calls remain zero.

The held race's selected Property remains satisfied. Its atMostOneActiveAttempt monitor expects inconclusive because its readAfter(attemptAdmitted) evaluation point is never reached on the rejected-admission path; this is declared in Scala with the evaluator's exact reason, not a Go exception.

Verification: baseline green (all four Quick commands, baseline-0.log through baseline-3.log). Final generation/check, generated live suite (46.301s), focused unit packages, focused tests after lint helper extraction, canary preflight unit tests, and existing TestTestpilot compatibility tests (351.744s) passed. The compatibility invocation excludes Scala (separately verified) and exactly one legacy Lean replay command test requiring the absent model/lean/.lake/build/bin/umpire-replay-bridge. That test is unrun, not passed; .11 owns its Scala replacement. The existing external CLI success test now runs a generated Nexus Case; its unreachable-endpoint test supplies the required Nexus binding so it reaches the intended connection failure.

Required default make lint-code-fast was run and leaves one inherited finding: realizing.declarations complexity 27 > 25. That function is byte-identical to the saved base (inherited-lint.txt). Task-scoped make lint-code-fast with integration and canary_harness tags passes with 0 issues; git diff --check passes. Initial final builds exhausted disk; after completion, cleaning reproducible Go cache allowed sequential reruns to pass. No evidence logs were removed.

The removed tests' assertions beyond current Contracts are recorded in README.md: exact returned done payload, UUID spelling/unique delivery IDs, whole SDK response sequence, cross-namespace learned-ID rejection, raw Nexus endpoint/history attribute equality, and workflow-backed activity parity. Existing correlated path ordering, terminal/supporting evidence, both Nexus implementation switches, and generic ambiguity/fault controls remain covered. Model-budget live faults are .23 scope, not claimed here.

Base: 8fff42d9a8f77a07e8f351f3871cc8030dfe1db6. Exact 48 task paths: .flow/tmp/fn107-22/changed-paths.json. Gate commands and logs: .flow/tmp/fn107-22-evidence.json. Changes are uncommitted by user instruction; no staging, commit, branch change, or worktree. Conductor verified the changed paths and test outputs before completion.

Independent review: gpt-6.1-sol at high reviewed the exact uncommitted 48-file scope, then the lint helper extraction and external CLI fixture changes, returning SHIP both times with no Critical or Important finding. Full digest: .flow/tmp/fn107-22-review.md. Native worktree review replaces the commit-only wrapper because user instructions reserve commits for the owner. Its optional SyncCases admission-hardening suggestion is deferred: current production callers use checked generation.

stage: impl-review - ran (native independent-context reviewer; SHIP, initial and incremental)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: PASS: GOFLAGS=-tags=test_dep make umpire-gen-scala (exit 0; .flow/tmp/fn107-22/generate-final.log), PASS: GOFLAGS=-tags=test_dep make umpire-check-scala (exit 0; .flow/tmp/fn107-22/check-final-2.log), PASS: cd tests && go test -tags 'test_dep integration canary_harness' -run '^TestTestpilotScala' -count=1 . (exit 0; .flow/tmp/fn107-22/live-retry.log), PASS: go test -tags test_dep ./model/scalav2/... ./tests/testcore/testpilot/... ./tools/umpire/... (exit 0; .flow/tmp/fn107-22/unit-final.log), PASS: go test -tags test_dep ./model/scalav2/goir ./model/scalav2/goir/testpilot (exit 0; .flow/tmp/fn107-22/unit-lint-fixes.log), INHERITED FAILURE: GOLANGCI_LINT_FIX=false make lint-code-fast (exit 2; .flow/tmp/fn107-22/lint-final.log), PASS: cd tests && go test -tags 'test_dep integration canary_harness' -run '^TestTestpilot' -skip '^TestTestpilot(Scala|NexusControlReplaysThroughTheCommand)' -count=1 . (exit 0; .flow/tmp/fn107-22/compatibility.log), PASS: go test -tags 'test_dep canary_harness' ./tools/canary/preflight (exit 0; .flow/tmp/fn107-22/canary-unit.log), PASS: GOLANGCI_LINT_BASE_REV=8fff42d9a8f77a07e8f351f3871cc8030dfe1db6 GOLANGCI_LINT_FIX=false TEST_TAG=integration,canary_harness make lint-code-fast (exit 0; .flow/tmp/fn107-22/lint-scoped.log), PASS: git diff --check, UNRUN: TestTestpilotNexusControlReplaysThroughTheCommand: model/lean/.lake/build/bin/umpire-replay-bridge is absent; replacement belongs to fn-107.11
- PRs: