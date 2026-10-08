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
  and both member states needed to explain the result. No error surface beyond R1 and R3. [inferred]
- **R3:** Fairness resolves to composed action classes, with documented inheritance and
  replacement rules. Reject unknown, ambiguous or unsupported fairness references. Tests cover
  a locally enabled member action blocked by its synchronization partner, and show that weak
  fairness alone does not prove a fixed response bound. [inferred]
- **R4:** Checks distinguish success within the stated scope, deadlock, fair non-progress
  cycle, deadline miss and incomplete exploration. Fixtures include holes, search exhaustion
  and multiple member starts; none is silently converted into a passing result. Existing
  single-machine progress and composition safety results remain unchanged. [inferred]
- **R5:** One concrete Temporal composition demonstrates a cross-machine safety promise and
  a bounded progress promise, with a passing design and a faulty or missing-prerequisite
  variant. Documentation explains their bounds and assumptions and states that these are
  model checks. An unsupported runtime realization remains unsupported. [inferred]

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
