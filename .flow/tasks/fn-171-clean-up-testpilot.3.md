---
satisfies: [R3]
---
# fn-171-clean-up-testpilot.3 Tidy: current docs, package docs, test-only exports, generated-case loaders to lower, canary preflight import

## Description
From the Testpilot research (2026-10-12).
1. Docs: `temporal/README.md:8,63` and `tests/testcore/testpilot/README.md:32` still say Case 1.0, but 4.0 is the only admitted format. Move the root README's CLI exit-code section (about lines 525-593) to `tools/umpire/README.md`, and consider splitting its protocol-extension checklist (about lines 270-495) into its own doc. Add package docs to `replay`, `temporal/worker`, `temporal/control`, `temporal/internal/{delivery,activation}`, `duration` and `tests/testcore/testpilot`. Fix `.plans/UMPIRE_MODULES.md:65`, which lists `casefile.ErrNoncanonical`/`Indent`; the code has `FormatError`/`CheckVersion`/`JSONVersion`.
2. Move `tests/testcore/testpilot`'s test-only exports (`NexusCallerProfile`, `WorkflowStartProfile`, `NexusPairProfile`, `LoadModelCase`, `ResolveSettings`) to `_test.go`.
3. Move `GeneratedCases`/`LoadGeneratedCase`/`GeneratedCaseName` (which wrap `lower.GeneratedCase`) into `tools/umpire/lower`, and make `tools/canary/preflight/harness_test.go` import them from there, not from `tests/testcore/testpilot` (contradicting `UMPIRE_MODULES.md:73`).
4. Optional, size only: the oversized files `execution/scheduler.go` (1,316 lines), `execution/dataflow.go`, `delivery/ledger.go`, `assessment.go` and `execution/prepare.go`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
