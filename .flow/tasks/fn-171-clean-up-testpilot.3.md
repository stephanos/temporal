---
satisfies: [R3]
---
# fn-171-clean-up-testpilot.3 Tidy: current docs, package docs, test-only exports, generated-case loaders to lower, canary preflight import

## Description
**Recut 2026-10-18 (one-language series):** Drop the loader move to `tools/umpire/lower` (deleted by fn-208); generated-case loaders go to `tools/testpilot`. The docs part stays.

From the Testpilot research (2026-10-12).
1. Docs: `temporal/README.md:8,63` and `tests/testcore/testpilot/README.md:32` still say Case 1.0, but 4.0 is the only admitted format. Move the root README's CLI exit-code section (about lines 525-593) to `tools/umpire/README.md`, and consider splitting its protocol-extension checklist (about lines 270-495) into its own doc. Add package docs to `replay`, `temporal/worker`, `temporal/control`, `temporal/internal/{delivery,activation}`, `duration` and `tests/testcore/testpilot`. Fix `.plans/UMPIRE_MODULES.md:65`, which lists `casefile.ErrNoncanonical`/`Indent`; the code has `FormatError`/`CheckVersion`/`JSONVersion`.
2. Move `tests/testcore/testpilot`'s test-only exports (`NexusCallerProfile`, `WorkflowStartProfile`, `NexusPairProfile`, `LoadModelCase`, `ResolveSettings`) to `_test.go`.
3. Move `GeneratedCases`/`LoadGeneratedCase`/`GeneratedCaseName` (which wrap `lower.GeneratedCase`) into `tools/umpire/lower`, and make `tools/canary/preflight/harness_test.go` import them from there, not from `tests/testcore/testpilot` (contradicting `UMPIRE_MODULES.md:73`).
4. Optional, size only: the oversized files `execution/scheduler.go` (1,316 lines), `execution/dataflow.go`, `delivery/ledger.go`, `assessment.go` and `execution/prepare.go`.
## Acceptance
- [ ] TBD

## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
