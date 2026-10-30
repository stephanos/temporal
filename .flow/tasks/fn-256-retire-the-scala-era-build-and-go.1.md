---
satisfies: [R1, R3]
---
# fn-256-retire-the-scala-era-build-and-go.1 Delete the Scala-era Go IR reader, Quint export and their CI/Make hooks

## Description
Delete the Scala-era Go reader: `tools/umpire/{ir,interp,check,realization,export,internal}`, the Quint helper and its pins in `export/tools_test.go`, and the `umpire-install-backends` / `umpire-check-backends` Make targets and comments (Makefile around 530–539). Narrow the CI step in `.github/workflows/umpire.yml` to the remaining packages. Before resuming deletion/integration, confirm the retained production and test closures in the separate modes below with `go list -mod=readonly`; if a retained package imports a deleted tree, stop and report.

**Size:** M (deletion plus retained-consumer verification)
**Files:** `tools/umpire/**`, `Makefile` (backends targets, `.PHONY` line 440), `.github/workflows/umpire.yml`, `AGENTS.md`, `tools/umpire/README.md`, `.plans/UMPIRE_MODULES.md`, `model/README.md`
**Touches:** [tools/tdbg/dlq_v2_service.go, tools/umpire/**, Makefile, .github/workflows/umpire.yml, AGENTS.md, tools/umpire/README.md, .plans/UMPIRE_MODULES.md, model/README.md]

### Dependency-survey evidence

The bounded checkpoint survey used combined `test_dep integration canary_harness` tags and found no removed package in that selected closure. Preserve that evidence and renew the following inventories against the admitted integrated inputs. Run from the repository root, record commands, source identity and complete package output, and reject any of the six removed package prefixes.

- Production: `go list -mod=readonly -deps ./tools/umpire/cmd/umpire ./tools/umpire/temporaldriver ./common/testing/testpilot/... ./tools/canary/...`.
- Package tests and production seams: `go list -mod=readonly -deps -test -tags test_dep ./tools/umpire/cmd/umpire ./tools/umpire/temporaldriver ./common/testing/testpilot/... ./tools/canary/...`.
- Live integration: `go list -mod=readonly -deps -test -tags 'test_dep integration' ./tests`.
- Canary harness: `go list -mod=readonly -deps -test -tags 'test_dep canary_harness' ./tools/canary/cmd/umpire-canary ./tools/canary/testharness`.

The separate production/test modes include `seams.go` and `seams_test.go`, whose `!canary_harness` guards exclude them from the old combined survey. CI currently has a broad tools package test, with no separate Quint CI step; narrow that test while preserving live and canary jobs. Serialize the workflow selector with fn-254.10's framework/Model job changes and the shared guidance files with fn-254.13. Do not delete or weaken the Testpilot synthetic forbidden-import control to satisfy a textual search.

### Required vet gate repair

The required broad vet command exposes pre-existing QueueInfo protobuf mutex copies in `tools/tdbg/dlq_v2_service.go`. Repair that small retained-tool issue in this lane using pointer-preserving collection/iteration; preserve sorted JSON/table behavior. Run existing DLQ list controls and broad vet in the declared modes, retaining the original failure as baseline evidence. This exact file is admitted so the required gate can pass without suppressing copylocks or excluding unrelated packages.

### Deleted-directory lint selection

The removal exposes `lint-code-fast` returning a failure when its final changed directory no longer exists. Its Makefile directory-selection loop is admitted for a minimal fix that skips deleted directories successfully while retaining actual linter failures. Verify the wrapper against this task baseline and run the retained changed-package lint; no broad lint-policy rewrite.

## Acceptance
- [ ] The six packages and the backends targets are gone; A source-only tracked-file search finds no live imports/build hooks/current documentation referring to removed packages or backend targets. Preserve references in `model/archived`, historical `.flow`/`.plans` records and the Testpilot dependency-boundary negative fixture; record these explicit exemptions. Confirm all four separate retained dependency inventories listed above; textual matches and a combined-tag survey do not replace these mode-specific closures.
- [ ] `go build ./...`, `go vet` on the remaining tools, `go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...` and `make canary-build` pass. Execute `make umpire-check-live-tests` (not merely its dry run), plus retained canary/workflow tests for the CI edit. Keep production build and `test_dep` test/vet modes explicit.
- [ ] Current Go tooling ownership and instructions describe `cmd/umpire` and `temporaldriver`. Preserve explicitly historical Scala/Go migration records and mark them historical; their old paths and commands do not describe current ownership or build hooks.


## Done summary
Reviewed and removed six Scala-era Go reader/interpreter/checker/realization/export/internal trees, Quint helper/pin and backend hooks. CI retains current tools packages; documentation separates current and historical ownership. Pointer-preserving DLQ iteration repairs required vet; deleted-directory lint wrapper retains actual linter failures. Four independent dependency modes exclude removed trees. Current production build, both broad vet modes, retained Umpire/Testpilot/canary and DLQ tests, canary build/harness, actual fast lint and failure control pass. Current guest8b9ea freshness passes. Full actual make umpire-check-live-tests passes all prerequisites and1251umbrella identities after explicit source-attributed gap golden renewal (81/81authored Confirmed,0mismatch). Snapshot header normalized without payload change; old/new/failure/success evidence and3SHIPreviews retained. Orphan duration package preserved and ownership accurately records no current importer; Scala Make/pin/archive cleanup remains task2.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d9a58c62dd, 1888b3f702, 442e068641
- Tests: Four separate readonly production/test_dep/integration/canarydependencyinventories:pass,no removedprefix, go build ./...; broad vet production+test_dep; retained Umpire/Testpilot/canarytests; DLQV2controls:pass, make canary-build,canaryharness,actualfastlint:pass; false-lintercontrol exits2, Current native driver Temporal guest--check:pass,unchanged8b9ea, ExplicitRustgapcandidate importer+no-updateverification:pass; exact1178removals2additions4strengthenedpredictions accounted against reviewedproducerinventory, Actual make umpire-check-live-tests:exit0,slice+gaps+umbrella1251pass,emptyfailures; exactsourceequivalence andloghash verified, Postreviewdocsandheader-onlycorrections:source-scopediffcheckpass,parsedpayloadbyte-identical
- PRs: