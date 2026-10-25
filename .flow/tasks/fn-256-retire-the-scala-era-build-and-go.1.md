---
satisfies: [R1, R3]
---
# fn-256-retire-the-scala-era-build-and-go.1 Delete the Scala-era Go IR reader, Quint export and their CI/Make hooks

## Description
Delete the Scala-era Go reader: `tools/umpire/{ir,interp,check,realization,export,internal}`, the Quint helper and its pins in `export/tools_test.go`, and the `umpire-install-backends` / `umpire-check-backends` Make targets and comments (Makefile around 530–539). Narrow the CI step in `.github/workflows/umpire.yml` to the remaining packages. First confirm with `go list -mod=readonly -deps -test -tags 'test_dep integration canary_harness'` that `cmd/umpire` and `temporaldriver` import none of the deleted packages; if one does, stop and report.

**Size:** M (deletion plus retained-consumer verification)
**Files:** `tools/umpire/**`, `Makefile` (backends targets, `.PHONY` line 440), `.github/workflows/umpire.yml`, `AGENTS.md`, `tools/umpire/README.md`, `.plans/UMPIRE_MODULES.md`, `model/README.md`

### Dependency-survey evidence

The bounded pre-implementation survey found no removed package in the retained command/driver/Testpilot/canary production or test closure. CI currently has a broad tools package test, with no separate Quint CI step; narrow that test while preserving live and canary jobs. Do not delete or weaken the Testpilot synthetic forbidden-import control to satisfy a textual search.

## Acceptance
- [ ] The six packages and the backends targets are gone; A source-only tracked-file search finds no live imports/build hooks/current documentation referring to removed packages or backend targets. Preserve references in `model/archived`, historical `.flow`/`.plans` records and the Testpilot dependency-boundary negative fixture; record these explicit exemptions. Confirm the retained production and test dependency closures with `go list -mod=readonly -deps -test -tags 'test_dep integration canary_harness'` rather than treating text matches as imports.
- [ ] `go build ./...`, `go vet` on the remaining tools, `go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...` and `make canary-build` pass. Execute `make umpire-check-live-tests` (not merely its dry run), plus retained canary/workflow tests for the CI edit. Keep production build and `test_dep` test/vet modes explicit.
- [ ] Docs describe only `cmd/umpire` and `temporaldriver`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
