---
satisfies: [R6]
---
# fn-173-reachable-exploration-drop-full-tables.10 Native collection state without whole-state enumeration

## Description
Separate native state data support from finite input/outcome/fact enumeration. Original .2-.9 retain engine, interpreter, lifting, sample-isomorphism and backend obligations.

**Size:** M
**Files:** framework DSL Domain, Machine, Channel, Compose, Refine, Monitor, native data/Inbox tests and framework-local guidance.
**Touches:** [model/framework/dsl/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Apply design/reachable-exploration.md's Data versus Finite distinction to native construction. Preserve Finite inputs/outcomes/facts and finite catalog ordering; admit Set/partial Map/List state and Inbox without a hand-written whole-state Finite.
- Structural Data comparison must not read Finite.values. Preserve explicit finite action-domain enumeration order even when noncanonical (`Finite.of(2, 0)` stays `[2, 0]`); numeric comparison remains `0 < 2`. Unordered Inbox insertion uses structural type order, so that custom message catalog yields canonical contents `[0, 2]`, an explicit representation delta rather than arbitrary catalog/key parity. Pin both orders independently alongside FIFO/duplicate/capacity/owner controls. Parent's 2026-10-16 clarification supersedes a blanket type-order-equals-custom-catalog claim; held IR/backend parity remains open.
- Remove only native construction checks that require enumerating all S. Preserve type/owner checks and reject unsupported domains explicitly. Prove construction with a witness that throws on attempted whole-state enumeration.
- Both declared and derived rule registration currently enumerate state and evaluate guards to check overlap. Remove that eager checking at both sites; registration must not invoke guards/effects, and declaration ownership, typing, rule order and metadata must remain intact. Do not replace it with a native explorer or claim overlap safety; static/reached checking stays held.
- Carry supported state evidence through Machine, Built/Derived, Monitor and state-valued gap annotations, preserving selected metadata and exact Inbox ownership without requiring enumerable state. Generic Data[String] supports text equality/order but does not admit direct or nested String state; functions and unsupported leaves also refuse at the native construction boundary, including through custom Finite instances. Existing optional retained schemas alone cannot enforce this on every native declaration: provide a native admissibility proof or equivalent early boundary, with retained metadata as a consistency check rather than an exporter prerequisite.
- Re-anchor StepTables/MachineLaws compilation coupling after the Data change. Any temporary finite-only compatibility helper requires explicit finite evidence, is not installed as a state-construction fallback and earns no retirement/checking credit. Do not add fake Data.values, enumerate collection/Int state, or run complete whole-state tables as this task's gate. Full carrier retirement remains held with .3.
- Preserve native Scala `Int` authoring as approved in design §§6.5/9.6; do not add a numeric wrapper, alias or replacement arithmetic syntax. D6's checked signed-int64 and overflow-hole semantics are the interpreted-IR contract (§6.4), not direct callback execution. Test and document the host's actual 32-bit boundaries, wrapping arithmetic and arithmetic exceptions as smoke behavior only. Native tests must not become a checking oracle or claim checked-int64 semantics.
- Record the held .4/.5 numeric proof explicitly: checked signed64 endpoints, overflow holes, division by zero/minimum divided by -1, and preservation or explicit refusal of host constant folding and initialized numeric values. No admitted native `Data[Int]` or host arithmetic result closes R5 or cross-interpreter parity.
- Document the Data/state admission and custom-order boundary in framework-local guidance, with a short migration recipe; do not add a Scalafix rule or Temporal proof.

### Investigation targets
**Required:** model/framework/dsl/Domain.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Channel.scala; model/framework/dsl/Compose.scala; model/framework/dsl/Refine.scala; model/design/reachable-exploration.md.

### Verification
Run native domain/schema/machine/Inbox/composition/refinement selections with positive counts. No Go/schema/export/capture, whole-state tables or Temporal source inputs.

## Acceptance
- [ ] Native Set/Map/List and Inbox state constructs without whole-state enumeration; outcomes/facts/inputs retain finite-domain checks.
- [ ] Construction/derivation/monitor and state-gap controls retain owners/types/order without state enumeration or callback evaluation; unsupported direct/nested state leaves refuse before evaluation without requiring exporter execution.
- [ ] Independent finite-order and owner/type regressions pass; missing-map-key behavior is explicit in native tests.
- [ ] Noncanonical finite domains retain authored class enumeration order while structural comparison and unordered Inbox contents use type order without catalog access; any custom-channel representation delta is documented without claiming IR/key parity.
- [ ] Native Scala Int authoring and actual host boundaries are tested and documented as non-authoritative smoke behavior; no wrapper/alias or native checked-int64 claim is introduced. Checked interpreted arithmetic, constant-fold/initializer preservation or refusal, cross-interpreter parity and full R5/R6 remain explicitly open with .4/.5.

## Done summary
TBD
## Evidence
- Commits:
- Tests:
- PRs:
