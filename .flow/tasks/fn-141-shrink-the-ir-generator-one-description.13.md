---
satisfies: [R6, R11, R12]
---
# fn-141-shrink-the-ir-generator-one-description.13 Export capability expansions

## Description
A capability's shared Property is a def that takes the model and the capability's fields. At run time that is a call. Expand capabilities by calling, and delete the lifter's expansion.

**Size:** M
**Files:** `model/umpire/{Capabilities,Catalog}.scala`, `model/temporal/capabilities/**`, `model/irgen/Capabilities.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/capabilities/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Plan against the shape fn-134 leaves (a `capabilities` section, Properties in the kind's companion, bounds in `queries`, `origin` on generated Properties). The 2026-10-06 tree still has `Law`, `Catalog` and `Implements`.
- Generated names stay `<machine>.<property>`; `origin` and positions stay as fn-134 defines them.
- Waivers (`except`, `overriding`) and their reasons reach wherever fn-134 sends them.
- A function-valued capability field is a recorded function, so the rule that it names a def and never a lambda may fall away. Follow the ledger and put it to the owner if it changes what authors may write.

### Investigation targets
**Required:**
- `model/irgen/Capabilities.scala` (`declare`, `expand`, `lawOf`, `declaredOf`, `staticTotal`)
- `model/umpire/Capabilities.scala`, `Catalog.scala` and `model/temporal/capabilities/*.scala`
- fn-134's spec, for the shape and the sidecar's fate

## Acceptance
- [ ] Generated Properties, Scenarios and Queries in the IR come from the exporter; the lifter's capability reader is deleted.
- [ ] Every generated name, answer and `origin` is unchanged.
- [ ] The ledger's capability rows have their outcomes.
- [ ] `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
