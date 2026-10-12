---
satisfies: [R1]
---
# fn-171-clean-up-testpilot.1 Generic core self-contained: remove adapter back-edges, absolute import rule, offline readers off the runtime, Temporal files named in execution

## Description
From the Testpilot research (2026-10-12).
1. Remove the two back-edges into `temporal/binding`, today whitelisted by `permittedHelperDriverImport` in `dependency_boundary_test.go`. Move `campaign.CampaignBinder` (`campaign/run.go:29-37`) into `temporal/binding` or the CLIs. Replace `binding.IsPreparationRejection` with `errors.As` on `*testpilot.PreparationError`. Move `binding.ReleaseAll` (used by `replay/report.go:173`) into `campaign`. Then delete the whitelist.
2. Extend `internal/execution/dependencies_test.go` `TestPrivateCoreImportBoundary` to forbid `go.temporal.io/api` and `go.temporal.io/sdk` in `ir`, `verification`, `cel`, `predicate`, `duration`, `casefile`, `recordedrun`, `evaluation`, `publish`, `campaign` and `replay`.
3. Offline readers (`recordedrun`, `evaluation`) import the root facade only for `ConcludeVerdict` (`internal/execution/verdict.go`, 35 lines) and `DriverIdentity`. Move these into a leaf so umpire-assess and the canary assessment no longer link the runtime and the Temporal API.
4. Rename the Temporal-specific files in `internal/execution` (`typed.go`, `carrier.go`, `activity_reset.go`, `activity_external.go`, `activity_external_runtime.go`) to `temporal_*.go`, and add one README paragraph stating that the instruction set is Temporal's.
Behaviour unchanged: Testpilot tests, the conformance corpus and the recorded-run replays pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
