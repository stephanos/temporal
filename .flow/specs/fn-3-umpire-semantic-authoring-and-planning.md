# Umpire semantic authoring and planning core

## Goal & Context
<!-- scope: business -->

Give Umpire model engineers an exploratory-first, Lean-owned language for pure properties,
constrained behaviors, and explicit queries. The first deliverable is a proving ground inside the
Temporal fork: it must compile finite model questions into deterministic artifacts without a live
Temporal server, while keeping the concepts teachable to future Temporal feature engineers.

Success means the bounded Workflow–Nexus scenario and a second small scenario use the same public
concepts without changing the compiler boundary. Trustworthiness outranks learnability, and
learnability outranks feature breadth.

## Architecture & Data Models
<!-- scope: technical -->

Implement three separate typed authoring forms—Property, Behavior, and Query—over one stable,
namespaced semantic vocabulary. Portable property declarations are typed data with a Lean
denotation and executable evaluator. Capability records carry stable identities and checked laws;
type-class syntax may provide authoring convenience, but persisted artifacts state requirements
explicitly.

A behavior declares typed symbolic resource roles, semantic setup constraints, admissible actions,
occurrence bounds, and a partial order. `actionsExactly` fixes the controllable action schedule while
allowing model outcomes to vary. `traceExactly` denotes one fully selected semantic witness for
replay and promotion. The query binds or searches resources, selects a compatible target, declares
its quantifier and typed phase bounds, and invokes a deterministic bounded planner.

The first planner is a lazy pure-Lean enumerator behind a small backend interface. It produces a
concrete `DrivePlan` and an environment-independent `ExperimentSpec`. A trace contains an initial
semantic state followed by typed steps that retain selected actions, model outcomes, resulting
states, and semantic observations.

## API Contracts
<!-- scope: technical -->

- Properties can access only the vocabulary exposed by their declared capability view. A portable
  property is serializable and inspectable; an opaque expert escape hatch is excluded from planning,
  persisted artifacts, and promotion.
- Bounds carry explicit units such as semantic transitions, selected actions, observation positions,
  or logical time. Named profiles may supply defaults, but checked queries and artifacts expand all
  values and units.
- `verify` requires finite completeness evidence. `find witness`, `find counterexample`, and
  behavior-led selection retain their explicit existential or exploratory meaning.
- An empty behavior returns `unsatisfiable`; it never silently becomes a successful universal
  verification.
- Capability conflicts are rejected unless an explicit connector reconciles them. Declaration order
  never selects semantic meaning.
- Every authoring error is structured by stable kind, declaration identity, source path, offending
  value, and related identities.
- Query identity covers resolved semantic digests, expanded bounds, strategy, seed, and target
  composition while ignoring incidental source order and documentation text.

## Edge Cases & Constraints
<!-- scope: technical -->

Exhaustive queries fail before claiming completeness when relevant types are not finitely enumerable
or complete bounds cannot be established. Budget exhaustion is distinct from proof. Invalid bounds,
wrong-kind references, missing capabilities, connector ambiguity, constraint cycles, impossible
occurrence counts, and nondeterministic output fail closed.

Partial orders form directed acyclic graphs. The planner records one deterministic linear extension
in each `DrivePlan`; it does not claim control over goroutine, storage, transport, or scheduler
order. Enumeration is lazy and budgeted so memory is proportional to the active frontier rather
than the full trace set.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Stable typed vocabulary and capability records accept the Workflow–Nexus composition and reject missing or ambiguous connectors. Errors: unknown identities, wrong kinds, missing laws, and conflicting providers produce structured declaration errors. [paraphrase]
- **R2:** A portable property is inspectable data with a Lean denotation, executable evaluator, typed temporal bounds, and a capability-limited view. Errors: opaque or undeclared access cannot enter planning or persisted artifacts. [paraphrase]
- **R3:** A reusable behavior supports symbolic setup, allowed/required/forbidden actions, occurrence bounds, partial ordering, `actionsExactly`, and singleton `traceExactly`. Errors: cycles, impossible counts, invalid bindings, and malformed exact traces fail checking. [paraphrase]
- **R4:** Property-led and behavior-led queries distinguish exhaustive verification, witness search, counterexample search, and bounded selection. Errors: unsatisfiable spaces and exhausted budgets never report verification. [paraphrase]
- **R5:** The pure-Lean planner lazily produces deterministic `DrivePlan` and `ExperimentSpec` values with expanded bounds, semantic identity, provenance, and explicit omissions. Errors: incomplete finiteness evidence rejects exhaustive mode. [paraphrase]
- **R6:** One real cross-domain caller-closure property and one second small scenario reuse the same language and compiler concepts. Errors: removing the connector capability makes the cross-domain query fail before planning. [paraphrase]
- **R7:** The new languages cleanly replace the existing combined `Regression` authoring structures when they land; no permanent compatibility facade or second public authoring path remains. Errors: old and new public semantics cannot coexist silently. [user]
- **R8:** The implementation has no dependency on, reuse of, or behavioral reference to Umpire3. Errors: any such dependency or reuse fails verification. [user]

## Boundaries
<!-- scope: business -->

No live Temporal execution, Go authoring facade, external solver, advanced coverage strategy,
multiple evidence profiles, or user interface is included. The language is for Umpire/model
engineers, not a replacement for SDK workflow tests or ordinary application testing. Syntax remains
experimental while concepts and stable semantic identities are preserved.

## Decision Context
<!-- scope: both — conditionally substructured -->

### Motivation
<!-- scope: business -->

A concise semantic authoring path is the next proof point after the Protobuf-to-Lean structural
catalog. It must demonstrate useful exploration and exact regression without waiting for runtime
machinery.

### Implementation Tradeoffs
<!-- scope: technical -->

Separate typed forms avoid recreating one universal DSL. Explicit capabilities enable cross-domain
reuse without a global state bag. A native bounded enumerator keeps the first trusted core small;
later planners can implement the same contract. Distinguishing exact actions from an exact trace
prevents a drive schedule from pretending to control model outcomes.
