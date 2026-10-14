---
satisfies: [R7]
---
# fn-169-a-temporal-free-framework-the-lifter.4 Held sweep: Go consumers follow the framework.dsl IR type names

## Description
Held for the post-source propagation (owner sweep). See the spec's R7. The `tools/umpire` layout, the Temporal adapter separation and the `export/quint` split moved to fn-170.2; Testpilot's separation of its reusable parts moved to fn-171 (owner, 2026-10-12).

Go consumers follow the `framework.dsl` IR type names that fn-169.1 introduces: `interp/eval.go` (`StepType = "framework.Step"`, `deliveryType = "framework.Delivery"`) and every Go test or fixture string naming `framework.*` DSL types. Regenerate and validate.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
