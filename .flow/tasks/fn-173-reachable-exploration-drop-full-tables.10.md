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
- Preserve native Scala `Int` authoring as approved in design §§6.5/9.6; do not add a numeric wrapper, alias or replacement arithmetic syntax. D6's checked signed-int64 and overflow-hole semantics are the interpreted-IR contract (§6.4), not direct callback execution. Test and document the host's actual 32-bit boundaries, wrapping arithmetic and arithmetic exceptions as smoke behavior only. Native tests must not become a checking oracle or claim checked-int64 semantics.
- Record the held .4/.5 numeric proof explicitly: checked signed64 endpoints, overflow holes, division by zero/minimum divided by -1, and preservation or explicit refusal of host constant folding and initialized numeric values. No admitted native `Data[Int]` or host arithmetic result closes R5 or cross-interpreter parity.

### Investigation targets
**Required:** model/framework/dsl/Domain.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Channel.scala; model/framework/dsl/Compose.scala; model/framework/dsl/Refine.scala; model/design/reachable-exploration.md.

### Verification
Run native domain/schema/machine/Inbox/composition/refinement selections with positive counts. No Go/schema/export/capture, whole-state tables or Temporal source inputs.

## Acceptance
- [ ] Native Set/Map/List and Inbox state constructs without whole-state enumeration; outcomes/facts/inputs retain finite-domain checks.
- [ ] Independent finite-order and owner/type regressions pass; missing-map-key behavior is explicit in native tests.
- [ ] Native Scala Int authoring and actual host boundaries are tested and documented as non-authoritative smoke behavior; no wrapper/alias or native checked-int64 claim is introduced. Checked interpreted arithmetic, constant-fold/initializer preservation or refusal, cross-interpreter parity and full R5/R6 remain explicitly open with .4/.5.

## Done summary
Blocked:
Owner-ordered native DSL queue. Release only after fn-165.9's native record caller/API checkpoint is committed on umpire. Cross-spec task edges are unsupported by flowctl; this task admission hold substitutes for that edge without requiring full IR-bearing spec closure. Native order and boundary are recorded in MILESTONES.md. No IR/Go/backend gate or Temporal source input may be added.
## Evidence
- Commits:
- Tests:
- PRs:
