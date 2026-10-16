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
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
