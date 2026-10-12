---
satisfies: [R7]
---
# fn-169-a-temporal-free-framework-the-lifter.4 Held sweep: Go tools/umpire temporal adapter and Testpilot execution API isolation with an import rule

## Description
Held for the post-source propagation (owner sweep). See the spec's R7.

1. Go consumers follow the `framework.dsl` IR type names fn-169.1 introduces: `tools/umpire/interp/eval.go` (`StepType = "framework.Step"`, `deliveryType = "framework.Delivery"`) and every Go test or fixture string naming `framework.*` DSL types; regenerate and validate.
2. Separate a `tools/umpire/temporal/` adapter from the generic packages: API builders (`lower/realization.go`), activity/nexus lowering and realization specifics (`lower/activity_*.go`, `realization/activity_*.go`, `realization/carriers.go`), Temporal lint binding kinds, conformance evidence/guard specifics, descriptor registration (the `workflowservice` blank imports in the commands).
3. Move Testpilot `internal/execution`'s Temporal API use behind `testpilot/temporal/`.
4. Add an import rule: generic Go packages may not import `go.temporal.io/api`.
5. **Quint out of the generic export (owner, 2026-10-12).** `tools/umpire/export` (package `backends`) keeps the backend-neutral parts: slice selection (`slice.go`), reader-side confirmation (`checked.go`), the `Tool` type (`tool.go`) and composition reading (`composed.go`). The Quint-specific parts move to `tools/umpire/export/quint`: the exporter (`quint.go`, `quint.sh`), the agreement (`agreement.go`), `quint verify`/Apalache (`verify.go`) and the ITF trace readers (`itf.go`, `dump.go`, `raw.go`), with their tests colocated. Update `make umpire-check-backends`, the README and `ir/ownership_test.go`; agreement receipts stay identical.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
