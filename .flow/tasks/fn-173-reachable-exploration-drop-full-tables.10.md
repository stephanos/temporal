---
satisfies: [R6]
---
# fn-173-reachable-exploration-drop-full-tables.10 Native collection state without whole-state enumeration

## Description
Separate native state data support from finite input/outcome/fact enumeration. Original .2-.9 retain engine, interpreter, lifting, sample-isomorphism and backend obligations.

**Size:** M
**Files:** framework DSL Domain, Machine, Channel, Compose, Refine and focused native data/Inbox tests.
**Touches:** [model/framework/dsl/**]

### Approach
- Apply design/reachable-exploration.md's Data versus Finite distinction to native construction. Preserve Finite inputs/outcomes/facts and finite catalog ordering; admit Set/partial Map/List state and Inbox without a hand-written whole-state Finite.
- Remove only native construction checks that require enumerating all S. Preserve type/owner checks and reject unsupported domains explicitly. Prove construction with a witness that throws on attempted whole-state enumeration.
- Settle and test the numeric authoring contract before completing this native checkpoint. Scala Int is 32-bit/wrapping; admission alone cannot prove the requested checked-int64/overflow-hole behavior. Establish the agreed native representation and boundary behavior with a focused proof before migrating numeric callers. Cross-interpreter parity remains later; do not substitute ordinary wrapping Int silently or close native DSL with this authoring decision unresolved.

### Investigation targets
**Required:** model/framework/dsl/Domain.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Channel.scala; model/framework/dsl/Compose.scala; model/framework/dsl/Refine.scala; model/design/reachable-exploration.md.

### Verification
Run native domain/schema/machine/Inbox/composition/refinement selections with positive counts. No Go/schema/export/capture, whole-state tables or Temporal source inputs.

## Acceptance
- [ ] Native Set/Map/List and Inbox state constructs without whole-state enumeration; outcomes/facts/inputs retain finite-domain checks.
- [ ] Independent finite-order and owner/type regressions pass; missing-map-key behavior is explicit in native tests.
- [ ] Numeric authoring representation and boundary behavior are settled, tested and documented before native completion; cross-interpreter parity and full R5/R6 remain open.

## Done summary
Blocked:
Owner-ordered native DSL queue. Release only after fn-165.9's native record caller/API checkpoint is committed on umpire. Cross-spec task edges are unsupported by flowctl; this task admission hold substitutes for that edge without requiring full IR-bearing spec closure. Native order and boundary are recorded in MILESTONES.md. No IR/Go/backend gate or Temporal source input may be added.
## Evidence
- Commits:
- Tests:
- PRs:
