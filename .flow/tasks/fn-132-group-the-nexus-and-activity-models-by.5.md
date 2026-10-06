---
satisfies: [R2]
---
# fn-132-group-the-nexus-and-activity-models-by.5 One NexusProduct that both Nexus forms refine

## Description
**Size:** M
**Touches:** [model/temporal/features/nexus/**, model/irgen/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** task 4 executable binding/outcome decision; both Nexus forms and local levels; current kind-level admission; exact projection helper and move ledger; product-property/refinement regressions in `tools/umpire/check`.
Declare the exact allowed identity/path/refinement delta before regeneration. Unsupported Reply members must remain in the shared type and be disabled per form, with a negative check for a step that has no product image. Preserve canonical level-owned vocabulary and both forms' root realization placement.

Part B. Move `NexusProduct` from `features/nexus/workflow/product/` to `features/nexus/product/Product.scala`, and `Reply`, `Resolution` and the handler actor to `features/nexus/Nexus.scala`, with the entity binding task 4 chose. `Reply` takes both forms' members (`handlerError(retryable)` included); a form whose handler cannot answer a member disables it in its rules.

`NexusSystem` (workflow) refines the shared product as before. The standalone machine gains `object refinement extends Refinement(NexusProduct)`: `unstarted` reads as `scheduled`, and `terminated` and the outcomes follow task 4's decisions. Product Properties (`terminalIsFinal` and the rest) are checked on both forms through their refinements.

No behaviour is added: no cancel in the workflow form, no retries or deadlines in the standalone form.

## Acceptance
- [ ] `NexusProduct`, `Reply`, `Resolution` and the handler's actions are declared once under `features/nexus/`, and no copy remains in either form.
- [ ] Both forms refine `NexusProduct`; the gate checks each product Property on both.
- [ ] A before/after projection differs only in the paths of moved declarations, the standalone refinement, and what task 4 recorded; Query answers and Case bytes are otherwise identical.
- [ ] The spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
