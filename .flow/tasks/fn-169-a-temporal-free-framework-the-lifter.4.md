---
satisfies: [R7]
---
# fn-169-a-temporal-free-framework-the-lifter.4 Held sweep: Go consumers follow framework.dsl IR names; Testpilot execution API isolation

## Description
Held for the post-source propagation (owner sweep). See the spec's R7. The `tools/umpire` layout, the Temporal adapter separation and the `export/quint` split moved to fn-170.2 (owner, 2026-10-12).

1. Go consumers follow the `framework.dsl` IR type names fn-169.1 introduces: `interp/eval.go` (`StepType = "framework.Step"`, `deliveryType = "framework.Delivery"`) and every Go test or fixture string naming `framework.*` DSL types; regenerate and validate.
2. Move Testpilot `internal/execution`'s Temporal API use behind `testpilot/temporal/`, with an import rule for Testpilot's generic core. The Testpilot structure research (2026-10-12) may widen this.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
