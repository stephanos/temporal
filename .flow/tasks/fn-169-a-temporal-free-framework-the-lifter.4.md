---
satisfies: [R7]
---
# fn-169-a-temporal-free-framework-the-lifter.4 Held sweep: Go consumers follow framework.dsl IR names; Testpilot's reusable parts separated from Temporal-specific code

## Description
Held for the post-source propagation (owner sweep). See the spec's R7. The `tools/umpire` layout, the Temporal adapter separation and the `export/quint` split moved to fn-170.2 (owner, 2026-10-12).

1. Go consumers follow the `framework.dsl` IR type names fn-169.1 introduces: `interp/eval.go` (`StepType = "framework.Step"`, `deliveryType = "framework.Delivery"`) and every Go test or fixture string naming `framework.*` DSL types; regenerate and validate.
2. Testpilot is mostly Temporal-specific and is not a framework (owner, 2026-10-12): it keeps its shape (a root plus `temporal/`). Separate its reusable parts (likely the Case format, preparation, Contract evaluation, offline replay, recorded runs, CEL, predicates, durations) from Temporal-specific code, starting with `internal/execution`'s Temporal API imports, and add an import rule for the reusable part. The Testpilot structure research (2026-10-12) will name the exact packages.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
