---
satisfies: [R10]
---
# fn-64-umpire-case-runtime.8 Remove the legacy Umpire execution path

## Description
Remove legacy runtime code after the async Nexus Case proves the replacement (R10). This task owns
code/model/fixture deletion and the minimum build wiring needed to remain green; Task 10 owns the
normative documentation, generated-document, and complete regression-gate reconciliation. Deletion
is blocked until a reviewed migration ledger accounts for the legacy test surface.

**Size:** M
**Files:** legacy Umpire proto/runtime/command/test/model paths selected by ownership inventory and
their direct build references
**Touches:** [proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**,
tools/umpire/{portableevaluation,runevaluation,runtime,runner,executor,executorgrpc,executorhttp,testplan,temporal/nexus}/**,
tools/umpire/cmd/**, tests/umpire4_*, model/Umpire/**, model/Temporal/**, Makefile]

## Approach
- Inventory consumers by ownership; delete the Umpire4 PortableTestPlan path without blindly
  removing unrelated Umpire2/3 or general artifact functionality.
- Before deletion, ledger every removed top-level Test/Fuzz and inherited failure identity as
  `preserved`, `replaced`, or `intentionally-retired`; name its replacement/owner and record a reason
  for retirement. Treat any unaccounted entry as a cutover blocker.
- Remove the legacy executor service without adding a network/CLI replacement.
- Delete property-specific portable evaluation, fixed Run Evaluation, replaced runner/runtime/
  executor packages, scenario Temporal Nexus adapter, caller-closure model/fixtures/generated tests,
  and obsolete commands after their consumers use the Case path.
- Preserve fn-5's scenario-neutral checked-promotion source types and validation without
  caller-closure imports; remove only the fixed caller-closure candidate, command, and binding.
- Remove or redirect direct Makefile/build references so the codebase remains buildable for Task 10.

## Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/service.proto:14` — legacy public executor service
- `tools/umpire/executor/portable_executor.go:115-180` — old orchestration root
- `tools/umpire/portableevaluation/property.go:14-247` — old property interpreter
- `tools/umpire/runevaluation/run_evaluation.go:71` — fixed checker root
- `tools/umpire/temporal/nexus/binding.go:9-118` — scenario adapter root
- `tests/umpire4_caller_closure_test.go` — restored legacy integration suite to retire
- `Makefile:1063-1167` — direct legacy target/package references

## Key context
Temporary coexistence was build sequencing, not compatibility. Preserve existing comments on any
general code that moves; delete comments only with the retired ownership they describe.

## Acceptance
- [ ] Active Go/proto/Lean code has no dependency on the PortableTestPlan service, property-specific
  evaluator, fixed Run Evaluation, scenario `temporal/nexus`, or caller-closure fixtures/tests.
- [ ] The removed public executor RPC has no compatibility reader or replacement transport/CLI.
- [ ] The reviewed migration ledger accounts for every deleted top-level Test/Fuzz and inherited
  failure identity; no unaccounted row reaches deletion.
- [ ] Fn-5's generic checked-promotion types and validation build without caller-closure imports,
  while its fixed caller-closure candidate, command, and binding are gone.
- [ ] Unrelated Umpire2/3 and general artifact/model functionality remains intact or is migrated only
  where a concrete new owner consumes it.
- [ ] Focused Case Runtime unit/integration tests and `make umpire-build-model` pass after deletion.

## Done summary
Removed the legacy PortableTestPlan executor/RPC, property-specific portable evaluation, fixed Run Evaluation, superseded runtime/runner/executor/local and scenario Nexus paths, caller-closure model lineage, fixtures, tests, generated APIs, and their direct build references. Preserved Umpire2/Umpire3, generic artifact/model/promotion and ordinary Nexus functionality; regenerated protobuf/Lean APIs and published the reviewed 179-path/307-test migration ledger at `.flow/artifacts/fn-64-umpire-case-runtime/task8-migration-ledger.md`.

The canonical regression aggregate is green when its Task 10-owned legacy-vocabulary prerequisite is omitted; the untouched vocabulary gate still reports ten legitimate Case Runtime `bounds` uses and the inherited `.qualified` false positive. Model import-graph checks pass; aggregate model lint retains the inherited Task 6 generated `ActivationBinding.controller.injEq` simpNF finding.

stage: impl-review - ran [2026-09-05T12:14:09Z..2026-09-05T12:26:11Z] - SHIP (`codex:gpt-5.6-sol:high`)
## Evidence
- Commits:
- Tests: make umpire-check-live-tests (baseline: exit 0; inherited Umpire2/Umpire3 failures classified by target), make proto, make umpire-gen-lean-api, SDKROOT=$(xcrun --sdk macosx --show-sdk-path) CC=/usr/bin/clang TMPDIR=$(pwd -P)/.flow/tmp go test -count=1 -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., SDKROOT=$(xcrun --sdk macosx --show-sdk-path) CC=/usr/bin/clang TMPDIR=$(pwd -P)/.flow/tmp go test -count=1 -tags test_dep ./tools/umpire/temporal/server/... ./tools/umpire/temporal/worker/..., SDKROOT=$(xcrun --sdk macosx --show-sdk-path) CC=/usr/bin/clang TMPDIR=$(pwd -P)/.flow/tmp go test -count=1 -tags 'test_dep integration' ./tests -run TestUmpireAsyncNexusCase, make umpire-build-model, lake build TemporalModelTests TemporalExperimentalTests temporal-model-inspect modelLint modelLintTests, CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/..., make umpire-check-regression (deferred Task 10 vocabulary gate: ten pre-task8 legitimate `bounds` uses and inherited catalog.go `.qualified` false positive), make -o umpire-check-legacy-vocabulary umpire-check-regression, make -o umpire-check-semantic-inventory lint-model (import graph passes; inherited Task 6 generated `Umpire.Case.ActivationBinding.controller.injEq` simpNF finding remains), make fmt-imports, SDKROOT=$(xcrun --sdk macosx --show-sdk-path) CC=/usr/bin/clang .bin/golangci-lint-v2.13.1 run --verbose --build-tags disable_grpc_modules,,test_dep, --timeout 10m --fix=false --new-from-rev=0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf --config=.github/.golangci.yml ./tools/umpire/... ./tests/..., SDKROOT=$(xcrun --sdk macosx --show-sdk-path) CC=/usr/bin/clang TMPDIR=$(pwd -P)/.flow/tmp go vet -tags 'disable_grpc_modules,,test_dep,' -vettool=.bin/errortype -style-check=false ./tools/umpire/... ./tests, git diff --cached --check 73fabb58da5c8ed4a456952d30be3b62c30d6377, migration ledger review: codex:gpt-5.6-sol:high round 3 SHIP; 179 paths, 303 Test + 4 Fuzz, 10 inherited identities, implementation review: codex:gpt-5.6-sol:high SHIP; synthetic staged tree fd0523ae36ccf84f501b881e103195eccb64fb28
- PRs: