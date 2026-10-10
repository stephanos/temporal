---
satisfies: [R6]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.4 Discover exported machines for generic law checks

## Description

Implement R6 with one reusable exported-subject traversal and generic law suite. This follows convention source fixes because both alter Model tests and framework root traversal.

**Size:** M
**Files:** `model/framework/IrFile.scala`, `model/framework/Machine.scala`, `model/framework/Refine.scala`, `model/temporal/IrFiles.test.scala`, activity/Nexus role-refinement tests and new generic law tests.
**Touches:** [model/framework/IrFile.scala, model/framework/Machine.scala, model/framework/Refine.scala, model/framework/*test.scala, model/temporal/IrFiles.test.scala, model/temporal/**/*test.scala, .flow/tmp/fn156/source/**]

## Approach

- Reuse IrFile.construct's closure, retaining typed instances/witnesses for law checks rather than returning only names. Initialize export owners through discovered compiled declarations or the existing compilation/lifting inventory; adding a machine to an export requires no test registry edit. Keep the actual declaration universe independent from the artifact under validation.
- Include Queries' scenario/property machines, capability/realization/progress owners, composed members, refinement targets and every derivation source. Deduplicate by full declaration identity, not only simple name. Detect same-name collisions and missing/dropped export or machine subjects with seeded negatives.
- Preserve Scala role and refinement map evidence for totality and closedness, since IR carries no roles. Convert named role checks into generic instances without losing their existing assertions. Iterate every finite source state and verify image membership; record role-less/inapplicable refinements explicitly.
- Check deterministic repeated ordered result lists for each finite state/class and unique bindings. Preserve intentional named alternatives and failure/negative-control subjects. Binding order means declared actor-group/binding order, separately from sorted interpreter action classes; establish an independently pinned order inventory rather than comparing a list with itself.
- Report discovered counts/identities and every failing or inapplicable machine/law with its reason. New semantic failures remain visible for their owning batch; no Model, expected status or law is fitted to pass.

## Investigation targets

**Required:**
- `model/framework/IrFile.scala:29` - complete root traversal.
- `model/framework/Machine.scala:363` - finite witnesses and derivation reachability.
- `model/framework/Refine.scala:36` - refinement discovery/map access.
- `model/temporal/IrFiles.test.scala:13` - hand-listed exports and lifted completeness check.
- `model/temporal/features/activity/standalone/RoleRefinements.test.scala:9` - existing role laws.
- `model/temporal/features/nexus/product/NexusRoleRefinements.test.scala:16` - other refinement instances.
- `model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala:1050` - declaration order pin.

## Quick commands

Run `mise exec -- scala-cli test --server=false model/project.scala model/framework model/temporal` with full exported-law scope. Focus new discovery/negative fixtures during development. Keep complete diagnostics and law dispositions, not just aggregate counts.

- [ ] A seeded new exported machine and a new export owner enter every applicable law without test registration edits; dropping a whole subject or duplicate identity fails completeness checks.
- [ ] Totality, closedness, repeatable relations and independent binding-order assertions cover the full exported closure, including derived, composed, failure and negative-control subjects.
- [ ] Existing activity/Nexus role assertions survive as generic instances with Scala role evidence; no IR/schema/Go changes supply substitute semantics.
- [ ] Every inapplicable or failing machine/law is listed with its reason; named choices remain intact and no semantic expectation is weakened.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Acceptance
- [ ] TBD
