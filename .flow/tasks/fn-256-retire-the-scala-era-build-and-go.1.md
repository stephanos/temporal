---
satisfies: [R1, R3]
---
# fn-256-retire-the-scala-era-build-and-go.1 Delete the Scala-era Go IR reader, Quint export and their CI/Make hooks

## Description
Delete the Scala-era Go reader: `tools/umpire/{ir,interp,check,realization,export,internal}`, the Quint helper and its pins in `export/tools_test.go`, and the `umpire-install-backends` / `umpire-check-backends` Make targets and comments (Makefile around 530–539). Narrow the CI step in `.github/workflows/umpire.yml` to the remaining packages. First confirm with `go list -mod=readonly -deps -test -tags 'test_dep integration canary_harness'` that `cmd/umpire` and `temporaldriver` import none of the deleted packages; if one does, stop and report.

**Size:** M (deletion plus retained-consumer verification)
**Files:** `tools/umpire/**`, `Makefile` (backends targets, `.PHONY` line 440), `.github/workflows/umpire.yml`, `AGENTS.md`, `tools/umpire/README.md`, `.plans/UMPIRE_MODULES.md`, `model/README.md`
**Touches:** [tools/tdbg/dlq_v2_service.go, tools/umpire/**, Makefile, .github/workflows/umpire.yml, AGENTS.md, tools/umpire/README.md, .plans/UMPIRE_MODULES.md, model/README.md]

### Dependency-survey evidence

The bounded pre-implementation survey found no removed package in the retained command/driver/Testpilot/canary production or test closure. CI currently has a broad tools package test, with no separate Quint CI step; narrow that test while preserving live and canary jobs. Do not delete or weaken the Testpilot synthetic forbidden-import control to satisfy a textual search.

### Required vet gate repair

The required broad vet command exposes pre-existing QueueInfo protobuf mutex copies in `tools/tdbg/dlq_v2_service.go`. Repair that small retained-tool issue in this lane using pointer-preserving collection/iteration; preserve sorted JSON/table behavior. Run existing DLQ list controls and broad vet in the declared modes, retaining the original failure as baseline evidence. This exact file is admitted so the required gate can pass without suppressing copylocks or excluding unrelated packages.

### Deleted-directory lint selection

The removal exposes `lint-code-fast` returning a failure when its final changed directory no longer exists. Its Makefile directory-selection loop is admitted for a minimal fix that skips deleted directories successfully while retaining actual linter failures. Verify the wrapper against this task baseline and run the retained changed-package lint; no broad lint-policy rewrite.

## Acceptance
- [ ] The six packages and the backends targets are gone; A source-only tracked-file search finds no live imports/build hooks/current documentation referring to removed packages or backend targets. Preserve references in `model/archived`, historical `.flow`/`.plans` records and the Testpilot dependency-boundary negative fixture; record these explicit exemptions. Confirm the retained production and test dependency closures with `go list -mod=readonly -deps -test -tags 'test_dep integration canary_harness'` rather than treating text matches as imports.
- [ ] `go build ./...`, `go vet` on the remaining tools, `go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...` and `make canary-build` pass. Execute `make umpire-check-live-tests` (not merely its dry run), plus retained canary/workflow tests for the CI edit. Keep production build and `test_dep` test/vet modes explicit.
- [ ] Docs describe only `cmd/umpire` and `temporaldriver`.


## Done summary
Blocked:
Source/proof checkpoint d9a58c62ddbe931083bebb368a24065480ec0e85 is preserved on work/fn2561-retirement in .worktrees/fn2561-retirement. It is not yet reviewed, integrated or complete.

The required live gate failed twice at raw/SDK judgment-path equality (seeds21/3). Slice failure prevents gaps/umbrella execution. A whole-prefix consumer diagnostic and exact regression follow-up are recorded in existing fn-246.9; no Model refutation was reported, and no assertion is waived. Resume task1 after that consumer obligation is resolved and the actual full live gate passes. The final current guest/policy producer renewal from fn-254.13 is also required before full canary tests can pass. Neither a source/import identity argument nor a dry run counts as a green gate.

Build, broad vet in both modes, retained tests, DLQ controls, canary harness/build and positive/negative lint controls passed at the checkpoint. Keep private Go cache idle/protected for fn-254.13's coordinated policy/preflight checks; retain unfinished branch/checkout and all committed proof. Continue fn-254 now, then the existing fn-246 order; this is a task prerequisite, not a block of the milestone goal.
## Evidence
- Commits:
- Tests:
- PRs:
