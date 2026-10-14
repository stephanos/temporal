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

## Overview

Replace the pilot's combined regression declaration with four independently testable layers: a
shared semantic foundation, pure portable properties, constrained behaviors, and explicit queries.
The fn-3 slice stops at deterministic model-only planning and emits inspectable `DrivePlan` and
`ExperimentSpec` values for later runtime and evidence work.

## Scope

- Establish the stable semantic identity, vocabulary, capability, connector, and pure trace
  contracts consumed by all later Umpire languages.
- Deliver Property, Behavior, and Query as separate typed Lean authoring forms.
- Implement the first deterministic bounded planner and its canonical artifacts.
- Prove the design with the Workflow–Nexus caller-closure scenario and a finite two-state switch
  scenario that uses the same public entry points.
- Remove the prior combined authoring path and retain one focused, root-owned model check.

## Approach

Build the shared semantic foundation first, then implement Property and Behavior independently.
Add the Query contract only after both inputs are stable, place the lazy enumerator behind a small
planner interface, and cut the existing pilot over only when the replacement path is complete. The
second scenario and focused regression suite form the final reuse and integration proof.

## Quick commands

```bash
cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect
make umpire-check-regression
```

## Architecture & Data Models
<!-- scope: technical -->

Implement three separate typed authoring forms—Property, Behavior, and Query—over one stable,
namespaced semantic vocabulary. Portable property declarations are typed data with a Lean
denotation and executable evaluator. Capability records carry stable identities and checked laws;
type-class syntax may provide authoring convenience, but persisted artifacts state requirements
explicitly.

The initial public surface is ordinary typed Lean declarations, constructors, combinators, and
lightweight notation. A custom parser or command elaborator is not part of this slice. Capability
law obligations are Lean proof fields on checked providers and connectors; portable declarations
and artifacts retain the stable law identities and semantic digests, not proof terms.

A behavior declares typed symbolic resource roles, semantic setup constraints, admissible actions,
occurrence bounds, and a partial order. `actionsExactly` fixes the controllable action schedule while
allowing model outcomes to vary. `traceExactly` denotes one fully selected semantic witness for
replay and promotion. The query binds or searches resources, selects a compatible target, declares
its quantifier and typed phase bounds, and invokes a deterministic bounded planner.

The first planner is a lazy pure-Lean enumerator behind a small backend interface. It produces a
concrete `DrivePlan` and an environment-independent `ExperimentSpec`. A trace contains an initial
semantic state followed by typed steps that retain selected actions, model outcomes, resulting
states, and semantic observations.

`SemanticTrace` is pure model data. Its observation values are model-emitted semantic values, not
raw execution evidence, and it carries no qualification or evidence derivations. Property
evaluation over that trace returns a model-only clause result that later evidence work can gate and
wrap without redefining the property's denotation.

Every checked target also provides a pure semantic transition kernel. It enumerates valid initial
states for resolved setup and, for a state plus selected action, valid model-owned outcomes,
resulting states, and model-emitted observations. Lean soundness and completeness fields connect
those finite enumerators to the target's authoritative transition relation. Planning and exact-trace
validation consume this kernel; they never infer valid steps from a Cartesian product of finite
value domains or retain callback-based semantic projections.

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
- A target supplies finite role/action domains plus sound-and-complete initial-state and step
  enumerators for every choice relevant to an exhaustive query. Missing evidence or a kernel without
  its required relation proofs rejects the query before enumeration.
- An empty behavior returns `unsatisfiable`; it never silently becomes a successful universal
  verification.
- Capability conflicts are rejected unless an explicit connector reconciles them. Declaration order
  never selects semantic meaning.
- Every authoring error is structured by stable kind, declaration identity, source path, offending
  value, and related identities.
- Query identity covers resolved semantic digests, expanded bounds, strategy, seed, and target
  composition while ignoring incidental source order and documentation text.
- Planning results distinguish a found selection, verified completion, complete absence of a
  requested trace, budget exhaustion, unsatisfiable behavior, and invalid authoring. Every result
  records explored counts, expanded bounds, and whether completeness was established.
- Vocabulary/capability metadata and every portable Property, Behavior, Query, `DrivePlan`, and
  `ExperimentSpec` value have deterministic in-memory and canonical JSON projections in this slice.
  Meaning-bearing mutations change the corresponding bytes/digest. Public readers, compatibility
  policy, and migrations remain separate follow-up work.

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

Canonical ordering covers vocabulary and capability sets, related diagnostic identities, partial
order tie-breaking, planner candidates, digest inputs, and artifact collections. Fixed strategy,
seed, target, and checked declarations must produce byte-identical inspection output.

`traceExactly` is checked in two phases: Behavior validates that the witness is structurally
complete, then Query/planning replays every step through the selected target kernel. A structurally
complete but semantically invalid witness is rejected rather than treated as a singleton trace.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Stable typed vocabulary, proof-backed capability records, sound-and-complete target
  transition kernels, and explicit connectors accept the Workflow–Nexus composition independent of
  declaration order. Errors: unknown identities, wrong kinds, missing laws, incomplete kernels,
  missing capabilities, ambiguous connectors, and conflicting providers produce structured
  declaration errors with stable source and related-identity fields.
- **R2:** A portable property is inspectable data with a Lean denotation, a structurally proved
  agreeing executable evaluator for the whole portable core, typed temporal bounds, a
  capability-limited pure trace view, and a model-only clause result. Errors: opaque callbacks,
  undeclared vocabulary access, unit mismatches, and evidence-derived fields cannot enter planning
  or persisted artifacts.
- **R3:** A reusable behavior supports symbolic setup, allowed/required/forbidden actions,
  occurrence bounds, partial ordering, `actionsExactly`, and singleton `traceExactly`.
  `actionsExactly` permits distinct model-owned outcomes, while `traceExactly` admits exactly one
  complete pure semantic trace. Errors: cycles, impossible counts, invalid bindings, contradictory
  constraints, and malformed exact traces fail checking.
- **R4:** Property-led and behavior-led queries distinguish exhaustive verification, witness search,
  counterexample search, and bounded selection with typed phase bounds and deterministic strategy
  policy. Errors: unsatisfiable spaces, exhausted budgets, and missing finite-completeness evidence
  remain distinct and never report verification.
- **R5:** The pure-Lean planner consumes only checked target kernels and lazily produces
  deterministic `DrivePlan` and `ExperimentSpec` values with expanded bounds, explored counts,
  semantic identity, provenance, selection reason, and explicit omissions; fixed checked inputs
  produce byte-identical canonical output without generating or retaining candidates beyond the
  consumed frontier. Errors: incomplete finiteness evidence rejects exhaustive mode, and eager,
  nondeterministic, or incomplete artifact construction fails closed.
- **R6:** A real cross-domain caller-closure property and a finite two-state switch scenario reuse
  the same target-kernel, Property, Behavior, Query, planner, and artifact concepts without changing
  core modules. Errors: removing the Workflow–Nexus connector makes the cross-domain query fail
  before planning.
- **R7:** The new languages cleanly replace the existing combined `Regression` authoring structures
  when they land; no permanent compatibility facade, callback-bearing target projection, or second
  public authoring path remains. Errors: old and new public semantics cannot coexist silently.
- **R8:** The implementation has no dependency on, reuse of, or behavioral reference to Umpire3.
  Errors: any such dependency, import, reuse, or semantic coupling fails verification.

## Early proof point

Task fn-3-umpire-semantic-authoring-and-planning.1 validates that stable typed vocabulary and
proof-backed capability/transition-kernel composition can express the shared semantic foundation
and reject an unreconciled provider conflict or incomplete kernel deterministically. If that
interface cannot support both the Workflow–Nexus connector and the independent switch capability,
re-evaluate the target/capability boundary before continuing with later language tasks.

## Boundaries
<!-- scope: business -->

No live Temporal execution, evidence interpretation, qualified verdict aggregation, Go authoring
facade, external solver, advanced coverage strategy, multiple evidence profiles, custom Lean parser
or command elaborator, or user interface is included. Artifact readers, compatibility migrations,
promotion, generated glossary/discovery surfaces, generated-API drift gates, and CI workflow work
are deferred. The language is for Umpire/model engineers, not a replacement for SDK workflow tests
or ordinary application testing. Syntax remains experimental while concepts and stable semantic
identities are preserved.

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

## References

- Umpire semantic authoring languages design contract
- fn-1-lean-regression-dsl-and-nexus — implemented compiler and artifact substrate being replaced
- fn-4-umpire-observation-and-semantic-verdicts — downstream evidence interpretation and verdicts
- fn-5-umpire-discovery-promotion-and-artifact — downstream glossary, promotion, and evolution

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Typed vocabulary, capabilities, connectors, and structured diagnostics | fn-3-umpire-semantic-authoring-and-planning.1, fn-3-umpire-semantic-authoring-and-planning.6, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R2 | Portable pure properties, trace views, denotation, and evaluator | fn-3-umpire-semantic-authoring-and-planning.1, fn-3-umpire-semantic-authoring-and-planning.2, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R3 | Reusable behavior constraint algebra and exactness levels | fn-3-umpire-semantic-authoring-and-planning.3, fn-3-umpire-semantic-authoring-and-planning.5, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R4 | Explicit query modes, finite completeness, and result semantics | fn-3-umpire-semantic-authoring-and-planning.4, fn-3-umpire-semantic-authoring-and-planning.5, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R5 | Deterministic lazy planning and inspectable artifacts | fn-3-umpire-semantic-authoring-and-planning.4, fn-3-umpire-semantic-authoring-and-planning.5, fn-3-umpire-semantic-authoring-and-planning.6, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R6 | Caller-closure connector proof and independent switch reuse | fn-3-umpire-semantic-authoring-and-planning.6, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R7 | Clean replacement of the combined regression path | fn-3-umpire-semantic-authoring-and-planning.5, fn-3-umpire-semantic-authoring-and-planning.6, fn-3-umpire-semantic-authoring-and-planning.7 | — |
| R8 | Explicit exclusion boundary | fn-3-umpire-semantic-authoring-and-planning.1, fn-3-umpire-semantic-authoring-and-planning.2, fn-3-umpire-semantic-authoring-and-planning.3, fn-3-umpire-semantic-authoring-and-planning.4, fn-3-umpire-semantic-authoring-and-planning.5, fn-3-umpire-semantic-authoring-and-planning.6, fn-3-umpire-semantic-authoring-and-planning.7 | — |
