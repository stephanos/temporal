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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
