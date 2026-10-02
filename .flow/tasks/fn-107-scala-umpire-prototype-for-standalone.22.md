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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
