# Bounded liveness across composed machines

## Goal & Context

Authors should be able to state and check a progress promise involving several machines on
their composition. Cross-machine safety already has this owner; bounded liveness should use
the same composed state and transitions. This follows the owner's agreement to track the
extension separately from fn-149's safety/liveness grouping. [paraphrase]

## Architecture & Data Models

A composition owns progress claims whose source and destination predicates read its whole
state, including multiple members. The checker evaluates the claim against the composition's
reachable transition graph. Member-only actions and synchronized actions count as composed
steps. A synchronized action counts once, regardless of how many members it advances. [inferred]

Reuse the existing bounded progress outcomes and weak-fairness interpretation. Fairness must
refer to the actions actually enabled in the composed graph. A member's local enabledness
does not establish that a synchronized action is enabled. [inferred]

## API Contracts

- Extend the existing progress declaration to accept a composition with predicates typed to
  its composed state, an explicit positive step bound, and explicit assumptions. [inferred]
- Expose composed action references for fairness, including synchronized actions. Reuse the
  existing composed action selection conventions; reject unknown or ambiguous references. [inferred]
- Claims fit under properties.liveness when fn-149's grouping is present. Their evaluated
  meaning is independent of that authoring reorganization. [paraphrase]

## Edge Cases & Constraints

- Define how member assumptions are inherited, scoped through member replacement and mapped
  to composed action classes. Do not silently drop an assumption or strengthen its guarantee
  while composing. Record the resulting assumptions in the check result. [inferred]
- Weak fairness alone provides no fixed scheduling delay. A bound counts all composed steps,
  including unrelated member actions, and a deadline miss remains possible on a fair path.
  A positive example must justify its bound from the actual graph and assumptions. [inferred]
- Preserve deadlock, fair non-progress cycle and deadline-miss distinctions. Holes and search
  limits retain their existing incomplete or unresolved meanings. [inferred]
- A progress claim must not pass vacuously because only the first member start was checked.
  Use the existing composition semantics over every combination of member starts. [inferred]

## Acceptance Criteria

- **R1:** An author can declare and check a bounded progress claim on a composition whose
  source and destination predicates read at least two members. Reject an unknown owner,
  incompatible predicate state type and a missing or non-positive bound with source attribution.
  [paraphrase]
- **R2:** The bound counts composed transitions. Tests cover a synchronized step counting
  once and an unrelated member step consuming the bound. A witness reports the composed actions
  and both member states needed to explain the result. No error surface beyond R1 and R3.
- **R3:** Fairness resolves to composed action classes, with documented inheritance and
  replacement rules. Reject unknown, ambiguous or unsupported fairness references. Tests cover
  a locally enabled member action blocked by its synchronization partner, and show that weak
  fairness alone does not prove a fixed response bound.
- **R4:** Checks distinguish success within the stated scope, deadlock, fair non-progress
  cycle, deadline miss and incomplete exploration. Fixtures include holes, search exhaustion
  and multiple member starts; none is silently converted into a passing result. Existing
  single-machine progress and composition safety results remain unchanged.
- **R5:** One concrete Temporal composition demonstrates a cross-machine safety promise and
  a bounded progress promise, with a passing design and a faulty or missing-prerequisite
  variant. Documentation explains their bounds and assumptions and states that these are
  model checks. An unsupported runtime realization remains unsupported.

## Early proof point

The first task checks a hand-authored two-member Progress declaration through the existing
composition subject and progress engine, with a passing and replayed failing case. If that requires
a second checking algorithm, reconsider the binding approach before extending authoring.

## Boundaries

- No unbounded liveness operators, stronger fairness guarantees, or automatic inference of
  environmental recovery assumptions. [inferred]
- Generating executable Cases for compositions and proving cross-machine progress from live
  evidence are separate work. [inferred]
- fn-149 owns safety/liveness grouping, declaration-kind enforcement and classification in
  existing reports. This spec owns the additional composition progress capability. [paraphrase]

## Decision Context

- A composition is the owner because the promise depends on several machines and their
  interactions. Independent checks on each machine do not establish the combined promise.
  [paraphrase]
- Related spec: [fn-149, safety and liveness groups for object properties](fn-149-safety-and-liveness-groups-for-object.md).
  Coordinate authoring and reporting without making this extension a prerequisite for fn-149.
  Execution remains unscheduled. [inferred]
- Coordinate with fn-141's composition and progress declaration work and the schema migrations
  already in the delivery queue. Re-anchor implementation to the completed interfaces. [inferred]

- Planning confirmed R2-R5 against existing composed tables, fairness remapping, progress outcomes
  and replay. The implementation extends those mechanisms rather than defining new fairness.
- Fairness references preserve the composed owner, member field or synchronization, and whether
  the author selected an action or one parameterized class. Repeated instances of one machine are
  distinguished by member field. Invalid or ambiguous references fail before graph checking.
- Reuse existing inherited fairness mapping, same-name unions and replacement-member assumptions
  for claim-specific fairness too. Enabledness is decided on the composed graph. Reports retain
  assumption names and show the composed actions in supporting witnesses.
- Monitored-member compositions remain explicitly unsupported, matching existing Query support.
  Check predicate state compatibility on the resolved owner; arity alone is insufficient.
- The Temporal example must have a reachable source that does not already satisfy its destination.
  Any action/start restrictions that justify a finite bound are stated in the Model and docs.
- No hard prerequisite on fn-149 or reverse prerequisite was found. These specs can ship separately;
  serialize their overlapping authoring and regeneration surfaces and consume the landed grouping
  contract. Keep the approved delivery chain unchanged and re-anchor task paths before execution.
- Retain the existing decision against broad generated-API drift verification and new CI coverage.

## Quick commands

```bash
go test -tags test_dep ./tools/umpire/ir ./tools/umpire/check ./tools/umpire/internal/engine
mise exec -- scala-cli test model/irgen --test-only Fixtures
```

The final task owns the integrated Model example, documentation, artifact and lint gates and the
single full Go run.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | An author can declare and check a bounded progress claim on a composition whose source and destination predicates read at least two members. Reject an unknown owner, incompatible predicate state type and a missing or non-positive bound with source attribution. | fn-150-bounded-liveness-across-composed.1, fn-150-bounded-liveness-across-composed.2, fn-150-bounded-liveness-across-composed.3, fn-150-bounded-liveness-across-composed.5 | — |
| R2 | The bound counts composed transitions. Tests cover a synchronized step counting once and an unrelated member step consuming the bound. A witness reports the composed actions and both member states needed to explain the result. No error surface beyond R1 and R3. | fn-150-bounded-liveness-across-composed.1, fn-150-bounded-liveness-across-composed.2, fn-150-bounded-liveness-across-composed.4, fn-150-bounded-liveness-across-composed.5 | — |
| R3 | Fairness resolves to composed action classes, with documented inheritance and replacement rules. Reject unknown, ambiguous or unsupported fairness references. Tests cover a locally enabled member action blocked by its synchronization partner, and show that weak fairness alone does not prove a fixed response bound. | fn-150-bounded-liveness-across-composed.2, fn-150-bounded-liveness-across-composed.3, fn-150-bounded-liveness-across-composed.4, fn-150-bounded-liveness-across-composed.5 | — |
| R4 | Checks distinguish success within the stated scope, deadlock, fair non-progress cycle, deadline miss and incomplete exploration. Fixtures include holes, search exhaustion and multiple member starts; none is silently converted into a passing result. Existing single-machine progress and composition safety results remain unchanged. | fn-150-bounded-liveness-across-composed.1, fn-150-bounded-liveness-across-composed.4, fn-150-bounded-liveness-across-composed.5 | — |
| R5 | One concrete Temporal composition demonstrates a cross-machine safety promise and a bounded progress promise, with a passing design and a faulty or missing-prerequisite variant. Documentation explains their bounds and assumptions and states that these are model checks. An unsupported runtime realization remains unsupported. | fn-150-bounded-liveness-across-composed.5 | — |
