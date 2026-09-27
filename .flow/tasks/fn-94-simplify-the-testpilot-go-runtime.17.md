---
satisfies: [R2, R8, R10]
---
# fn-94-simplify-the-testpilot-go-runtime.17 Closing gates, deadcode sweep, measurement and docs

## Description
The finalization task: the full gate run, the `deadcode` check R2 names, the R10 measurement against fn-94.2's baseline, and the doc sweep the lanes deferred.

**Size:** M
**Files:** READMEs under `common/testing/testpilot/**` that still name removed or renamed declarations (root `README.md:27-29` aliases; `internal/verification/README.md` if touched by removals; `temporal/README.md:14-21`), `.plans/UMPIRE4_ORDER.md` (fn-70's resume note about `runCase`; fn-94's entry), any stragglers `deadcode` reports
**Touches:** [common/testing/testpilot/**/README.md, tests/testcore/testpilot/README.md, .plans/UMPIRE4_ORDER.md, common/testing/testpilot/**, tests/**]

### Approach
- Run `go run golang.org/x/tools/cmd/deadcode@v0.48.0 -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/...` and filter to Testpilot packages; each reported function is deleted or given a production caller, never suppressed. Confirm candidates with `-whylive`.
- Run every R8 gate: `make umpire-check-case-runtime-conformance`, `umpire-check-testpilot-protocol`, `umpire-check-testpilot-authoring`, `umpire-check-retired-vocabulary`, `go test -race -tags test_dep ./common/testing/testpilot/...`, `make umpire-check-regression` (live identity count equals fn-94.2's), `make lint-code-fast`. Report anything not run with the reason.
- Measure with the spec's four commands; report each against fn-94.2's baseline and the R10 floors, with the reason for any miss.
- Docs: sweep the Testpilot READMEs for removed names; update fn-70's resume note in `.plans/UMPIRE4_ORDER.md` (it cites `runCase`). Do not edit `.plans/index.json` unless fn-94's dependencies changed.

### Quick commands
```sh
go run golang.org/x/tools/cmd/deadcode@v0.48.0 -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/... | grep testpilot
make umpire-check-regression
make lint-code-fast
```

## Acceptance
- [ ] `deadcode` reports no unreachable function in Testpilot packages.
- [ ] Every R8 gate passes, or is reported not run with the reason; live identity count unchanged.
- [ ] The receipt reports the four measurements against the baseline and the R10 floors.
- [ ] No README names a removed declaration; fn-70's resume note is current.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
