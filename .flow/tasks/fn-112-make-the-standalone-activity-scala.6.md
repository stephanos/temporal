---
satisfies: [R2, R5, R6, R7, R8, R9, R14, R15]
---
# fn-112-make-the-standalone-activity-scala.6 Migrate product, protocol and admission models to the new DSL

Touches: [model/temporal/standaloneactivity/Model.scala, model/temporal/standaloneactivity/System.scala, model/temporal/standaloneactivity/Claims.scala, model/temporal/standaloneactivity/StandaloneActivityPins.test.scala, model/gate/Roots.scala, model/ir/**, model/cases/**]

## Description
Apply the new declaration, derivation, step, evidence, input and counter surfaces to the product/protocol/admission subjects, including fn107 additions.

**Size:** M
**Files:** standaloneactivity Model/System/Claims roots before layout; feature pin tests and generated IR comparisons.

### Approach
- Start only after fn-112.11's total schema and fn-120.1's later choice schema are done. The same-spec task dependency records fn-112.11; the conductor checks fn-120.1's completion as a cross-spec entry gate. Replace copied admission machines and private step wrappers with derivation and canonical helpers. Use the settled named-choice syntax for existing branching; keep result count/order, tables and Case bytes exact. Do not wait for fn-120's lint/explorer/ITF work or strict unnamed-branch refusal.
- Capture declaration names/defaults, reduce evidence blocks to the two exceptions and migrate ProtocolState.attempts to UpTo[2] while preserving Active.
- Preserve `.results("Delivery")`, all finite catalogs and function-table rows; remove the dead Delivery enum only if unused by the frozen outputs.
- State retry saturation at attemptBound on the Property instead of changing retry behavior.

## Acceptance
- [ ] Product/protocol/admission declarations contain no duplicate steps list or private single-step helper and use the settled name/evidence/input/default forms.
- [ ] fn-112.11 and fn-120.1 are complete, in that order, before branching rewrites start; every rewritten branch uses final named-choice syntax.
- [ ] The fn107 held-delivery and response-loss machines are included.
- [ ] Result metadata, branch count/order, finite catalogs, IDs, tables, Query answers and Case bytes equal task 1. IR differences are limited to R1's exact source-position/moved-function allowances and authored Query.total, inert choice-name and queue-entity metadata deltas; entity-sensitive fingerprints match the approved delta and all other fingerprints remain exact.
- [ ] Focused feature roots, model gate and relevant Go golden tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
