---
satisfies: [R1, R2, R3, R4, R5, R6, R7]
---
# fn-69-extract-testpilot-from-umpire.8 Remove old Umpire runtime owners and close the migration

## Description
Delete the now-unreferenced Umpire proto/runtime owners, remove every temporary coexistence path, update active architecture and downstream dependencies, and run the complete compatibility gates (R1-R7).

**Size:** M
**Files:** former Umpire proto/generated/runtime/facade trees including `tools/umpire/carrier_test.go`, active Umpire/Testpilot docs including `model/Umpire/ARCHITECTURE.md`, regression/vocabulary enforcement, Make/workflow selectors, downstream Flow specs/tasks, migration ledger
**Touches:** [proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, tools/umpire/internal/{ir,execution}/**, tools/umpire/verification/**, tools/umpire/{prepare,prepared_case,profile,host}*.go, tools/umpire/carrier_test.go, tools/umpire/{CONTEXT,CLEANUP_INVENTORY}.md, tools/umpire/internal/{legacyvocabulary,retiredvocabulary}/check.go, tools/umpire/cmd/umpire-check-{legacy,retired}-vocabulary/**, tools/umpire/vocabulary/{legacy_gate,retired_vocabulary}_test.go, tools/umpire/regression/**, model/{README,ARCHITECTURE}.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_*.md, .github/workflows/umpire.yml, Makefile, .flow/specs/fn-{22,26,29,33,68}-*.md, .flow/tasks/fn-{22,26,29,33,68}-*.md, .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Prove zero active consumers before deleting old generated/source proto, facade, private core, verification, tests, and temporary compatibility scaffolding; preserve historical Flow artifacts and fn-64/fn-66 ledgers.
- Replace old ownership/import assertions with stronger Testpilot dependency-direction, destination-coverage, generated-identity, and exact live-selector checks.
- Retain the active retired-vocabulary guard, but rename its `legacyvocabulary` package, command, Make target, and test file to `retiredvocabulary` / `umpire-check-retired-vocabulary` terminology.
- Update active specs/docs narrowly to the Testpilot protocol/Driver boundary and dependency; retain each downstream spec's execution, assessment, canary, replay, and exploration ownership.
- Freeze the final source before serial full gates and reconcile every path, package, test, fixture, descriptor, byte, diagnostic, and expected inherited failure against the migration ledger.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/regression/ci_workflow_test.go:141-430` — executable architecture enforcement
- `.plans/UMPIRE4_SPEC.md:139-150` — normative public/runtime authority rules
- `model/README.md:115-159` — active package and command ownership
- `.flow/specs/fn-68-minimal-nexus3-success-demonstration.md` — immediate downstream consumer
- `.flow/specs/fn-29-bounded-production-canary-execution-and.md` — future Driver consumer boundary
- `tools/umpire/CLEANUP_INVENTORY.md` — retained Umpire tooling baseline

### Acceptance

## Acceptance
- [ ] Former Umpire proto source/generated package, public facade, private IR/execution/verification, adapter, caseartifact, aliases, converters, and temporary duplicate paths have zero active references and are absent; Testpilot is the sole owner.
- [ ] Executable dependency tests reject Testpilot imports of Umpire/adapters/canary, cross-adapter imports, private evaluator replacement, stale proto identities, and gate selectors that omit moved tests.
- [ ] The vocabulary guard still rejects retired public Umpire tokens, and no active package, command, Make target, or test path for that guard uses a `legacy*` name.
- [ ] Active docs and fn-22/fn-26/fn-29/fn-33/fn-68 dependencies/interfaces name Testpilot without changing their separate domain ownership; historical evidence remains immutable.
- [ ] Focused Testpilot/Umpire/testcore tests, `make proto`, Lean/API generation, conformance and regression checks, model build/lint, proto/API/code lint, full live selector, and exact migration-ledger reconciliation pass or match explicitly verified inherited failures.
- [ ] Final evidence proves preserved comments, errors, authority, bytes except allowlisted namespace identities, cancellation/crash/cleanup behavior, security boundaries, bounded 10x cost, package/test coverage, and no empty former directories.

## Done summary
Closed the Testpilot extraction by removing every former Umpire protocol/runtime owner and obsolete comparison path, retaining Umpire Lean authoring and Producer commands, and making common Testpilot plus the testcore Driver the sole active owners. Renamed the active vocabulary guard to retired-vocabulary terminology, updated executable dependency checks, workflow/Make wiring, active architecture and downstream plans, and recorded exact migration evidence.

Verification passed the 18-package serial selector, proto and Lean API regeneration, conformance and retired-vocabulary gates, exact live selector, 325-target aggregate regression, 330-job model build, 261-target model lint, proto/API lint, final-patch scoped no-fix Go lint, fixture/descriptor/import/reference/empty-directory reconciliation, and git diff checks. A cold whole-repository lint loader was stopped after exhausting temporary disk in unrelated tools/fairsim compilation; the package-scoped final-patch run covered all active Umpire/Testpilot packages and reported zero issues.

stage: impl-review - ran 2026-09-06T22:09:17Z..2026-09-06T22:14:44Z | SHIP (corrected synthetic range 2d9b1cea04f73635e877ccefc4a74ffb2de77dac..7bb7c883a37a1df02d5c8100f323c074deaaaa69)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/..., GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tests/testcore/testpilot/..., GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make proto, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make -B proto/image.bin, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-gen-lean-api, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-case-runtime-conformance, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-retired-vocabulary, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-live-tests, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-regression, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-build-model, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make lint-model, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make lint-protos, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp make lint-api, GOFLAGS=-p=1 CGO_ENABLED=0 TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression, GOLANGCI_LINT_FIX=false ./.bin/golangci-lint-v2.13.1 run --build-tags test_dep,integration --timeout 10m --fix=false --new-from-patch=/private/tmp/fn69-task8-final.patch --config=.github/.golangci.yml ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/..., fixture SHA-256 reconciliation: pass, descriptor/import/reference/live-selector/empty-directory reconciliation: pass, git diff --check
- PRs:
