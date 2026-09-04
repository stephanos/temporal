# Umpire public API

Umpire is the reusable, Temporal-independent library for composing finite semantic targets,
checking portable properties and behaviors, authoring bounded variation Spaces, planning bounded
queries, and producing portable model artifacts. For the cross-library map, see the
[model architecture](../ARCHITECTURE.md).

## Imports and modules

Most consumers should import the umbrella facade:

```lean
import Umpire
```

Focused imports are available when a consumer needs a smaller surface:

| Import | Public responsibility |
| --- | --- |
| `Umpire.Core` | Semantic vocabulary, canonical Definition identity, trace/result constructors, capabilities, laws, and finite kernels. |
| `Umpire.Target` | Finite-machine and expert Target authoring, checked composition, and canonical target projections. |
| `Umpire.Property` | Portable Property authoring, checked construction, typed diagnostics, and evaluation. |
| `Umpire.Behavior` | Setup and trace-shape authoring, checked construction, and typed diagnostics. |
| `Umpire.Query` | Checked combinations of Targets, Properties, Behaviors, Limits, and policies, plus typed diagnostics. |
| `Umpire.Json` | Ordered typed JSON construction and exact compact or pretty rendering for codec owners. |
| `Umpire.Artifact` | Exact v2 planning, runtime, Evidence, Result, set-admission, and immutable-publication contracts. |
| `Umpire.Planning` | Deterministic incremental planning over checked queries. |
| `Umpire.Promotion` | Deterministic checked source compilation from one unchanged planned Query. |
| `Umpire.Space` | Checked finite axes, request-only faults, seek-only coverage goals, metadata, point lowering, and atomic batch compilation. |
| `Umpire.Exploration` | Checked finite-universe selection, pinned precedence, and process-local one-candidate sequencing. |
| `Umpire.Observation` | Checked Evidence mappings, Observation Evaluation, Evidence Links, dispositions, Property verdicts, and strict aggregation. |
| `Umpire.ImplementationLink` | Checked forward correspondence between independently authored semantic Targets. |
| `Umpire.Case` | Temporal-independent Case, Program, Contract, Run, value, and Verdict data shared with the portable runtime. |

`Umpire.Target.Language`, `Umpire.Property.Language`, `Umpire.Property.Check`,
`Umpire.Property.Trace`, `Umpire.Property.Evaluation`, `Umpire.Behavior.Language`,
`Umpire.Query.Language`, `Umpire.Space.Language`, `Umpire.Space.Intent`,
`Umpire.Space.Metadata`, `Umpire.Space.Compiler`, `Umpire.Exploration.Language`,
`Umpire.Exploration.Candidate`, `Umpire.Exploration.Selection`, `Umpire.Exploration.Guided`,
`Umpire.Exploration.Engine`, `Umpire.Exploration.Session`, `Umpire.Observation.Declaration`,
`Umpire.Observation.Compiler`, `Umpire.Observation.Language`, `Umpire.Observation.Evaluation`,
`Umpire.ImplementationLink.Language`,
`Umpire.ImplementationLink.Application`, and `Umpire.Planning.Engine` implement their public
facades and should not normally be imported directly. `Umpire.Shared.DefinitionGraph` is likewise
an internal support module rather than a consumer facade.

## API lifecycle

The public API deliberately separates authoring from checked values:

```text
AuthoredTarget ── checkTarget / elaborateTarget ──▶ CheckedTarget
PropertyDeclaration ─ checkProperty / checkedProperty + explicit proof ─▶ CheckedProperty
BehaviorDeclaration ─ checkBehavior / checkedBehavior + explicit proof ─▶ CheckedBehavior
CheckedTarget + QueryDeclaration ─ checkQuery / checkedQuery + explicit proof ─▶ CheckedQuery
CheckedTarget + ObservationMappingDeclaration
  ─ checkObservation / checkedObservation + explicit proof ─▶ CheckedObservationPlan
CheckedQuery ─ derive planner kernel ─▶ plan ─▶ PlannerRun ─▶ ExperimentSpec?
CheckedQuery + ExperimentSpaceDeclaration ─ checkExperimentSpace ─▶ CheckedExperimentSpace
CheckedExperimentSpace ─ projectCheckedSpaceMetadata ─▶ CheckedSpaceMetadata
CheckedExperimentSpace + exact assignment ─ lowerSpacePoint ─▶ LoweredSpacePoint
CheckedExperimentSpace + base Query kernel ─ compileBatch ─▶ List ExperimentSpec
CheckedExperimentSpace + policy + Limit + pinned ExperimentSpecs + exact kernel
  ─ checkExplorationRequest / explore ─▶ ExplorationResult
ExplorationRequest + exact kernel ─ beginSession ─▶ process-local ExplorationSession
ImplementationLinkDeclaration + checked source/destination Targets + forward witness
  ─ checkImplementationLink ─▶ CheckedImplementationLink
CheckedImplementationLink + source setup + EvidenceBackedTrace
  ─ applyImplementationLink ─▶ ImplementationLinkResult
```

Target is the checked semantic substrate consumed by the distinct Property, Behavior, and Query
languages; it is not another scenario language. Space composes one checked Query rather than
introducing another Behavior, Query, Property, planner, or outcome language. Checked types freeze
canonical metadata and Behavior Fingerprints, and Planning accepts checked values rather than raw
author input.

The raw `checkProperty`, `checkBehavior`, `checkQuery`, and `checkObservation` functions are the
authoritative path when a caller needs the typed error for invalid input. Their `checked*` partners
only remove `Except` extraction (and Query's dependent Target re-ascription) after the caller
supplies an explicit proof that the same checker succeeded. They do not hide a `native_decide`
default, change validation, or add a runtime recovery path.

## Case Runtime IR

`Umpire.Case` is the Temporal-independent semantic mirror of the versioned protobuf Case contract;
its supporting types live under the `Umpire.Case` namespace (for example, `Umpire.Case.Program`).
A Case pairs one bounded Program with one deterministic Contract. Programs contain symbolic Host
roles, context-local entrypoint DAGs, immutable Slots, declared Observations, typed outcomes and
guards, an always-run cleanup graph, and explicit limits. Contracts consume only recorded Run
Events and declared Observations; their ordered finite-state transitions cite supporting monotonic
event coordinates in the resulting Verdict. Rule-local, typed scalar captures retain one declared
Observation for bounded cross-event comparison and are single-assignment by construction-time
validation.

The closed `ValueType` constructors preserve singular, repeated, and map cardinality. Instructions
and transition expressions are closed unions rather than callback or extension points. Endpoint
coordinates, clients, credentials, and opaque Host capability values are deliberately absent from
the Case data. The generated API types remain the generator-owned descriptor projection used at the
wire boundary; the `Umpire.Case.*` modules are the generic authored IR.

## Core and Target APIs

`Umpire.Core` defines the vocabulary shared by every other module. `Umpire.Target` owns target
authoring, validation, canonicalization, and checked composition.

Important value types:

- `DefinitionId` identifies semantic declarations. Public identities are expected to be
  namespaced; construct them with `DefinitionId.of`. `DefinitionId.canonicalSet`,
  `DefinitionId.firstDuplicate`, and `DefinitionId.validate` provide the shared deterministic
  identity normal form without choosing a language-specific error.
- `DefinitionMetadata` describes a state, action, outcome, observation, relation, capability,
  provider, law, connector, target, or kernel.
- `ModelValue` pairs a declaration identity with a canonical string value.
- `ModelTrace` and `ModelTraceStep` represent pure Model Traces. `ModelTraceStep.result` builds one
  step from its selected Action and the Target-owned transition result.
- `ModelCoordinate` identifies one canonical location in a Model Trace. `ModelTrace.coordinates`
  enumerates initial state followed by each step's selected Action, Model Outcome, resulting state,
  and observations in source order. `ModelTrace.valueAt?` is the sole positional lookup and rejects
  zero or out-of-range step and observation positions; every numeric position is strictly one-based.
  `ModelCoordinate.definitionKind` is the sole coordinate-kind mapping.
- `TransitionResult` represents one model-owned transition result. `TransitionResult.map` maps its
  outcome, resulting state, and observations without changing their structure or order.
- `Limit` associates one value with an explicit `LimitUnit`.

`ModelValue.named` is inert, additive shorthand for the existing public `ModelValue` record. It
pairs one caller-supplied Definition ID with one caller-supplied value string without validating,
inferring, registering, or canonicalizing either field. Raw record construction remains available,
and the checker that consumes the value remains the semantic authority.

`SourceLocation.displayPath` supplies the stable source-path fallback used by the authoring
languages. Behavior and `Umpire.Observation.Compiler` additionally import the internal
`Umpire.Shared.DefinitionGraph` module to share deterministic node, edge, order, and cycle analysis.
That module returns structural findings only: each language still consumes them at its own
validation stages and constructs its own typed diagnostic and historical cycle witness. It is not
imported by the `Umpire.Shared` facade or offered as another public authoring language.

Target composition uses:

- `FiniteMachine` — the ordinary proof-carrying adapter for complete finite Targets whose ordered
  enumerators are authoritative. It derives membership authority, a complete behavior domain, and
  finite planning from one descriptor.
- `TransitionKernel` — the direct expert route for finite initial-state and transition enumerators
  whose authoritative propositions are specified independently.
- `CapabilityProvider` — meanings and law witnesses supplied by one capability.
- `CapabilityConnector` — explicit reconciliation of meanings supplied by multiple providers.
- `TargetDefinition` — the ordinary semantic vocabulary and transition-kernel input.
- `TargetComposition` — a sealed builder that collects explicit provider and connector choices.
- `AuthoredTarget` — the sealed result of `AuthoredTarget.make`, plus optional Target-owned finite
  planning evidence and compiler-only occurrences.
- `TargetDeclaration` — the lower-level typed composition used by Target maintainers.
- `CheckedTarget` — validated and canonicalized target.

`FiniteMachine` authors still choose every setup, state, action, outcome, observation, encoder,
initial state, and transition result. They prove that emitted values stay in the declared domains
and that every advertised planning action is executable. The adapter removes only the routine
membership, completeness, and dependent planning assembly: its `kernelAvailability` and
`authoredPlanning` values enter the same `TargetDefinition` / `AuthoredTarget.make` path and the
same `checkTarget` boundary as directly authored kernels. It is typed Target convenience, not a
Behavior, Property, Query, Scenario, or macro language.

Direct `TransitionKernel` construction remains the expert route when authority is intentionally
independent of enumeration. Both routes are owned entirely by `Umpire.Target`; neither introduces
a dependency on `Shared`, Temporal families, runtime code, or optional verification modules.

The ordinary checked entry point is:

```lean
checkTarget :
  AuthoredTarget LawStatement Setup State Action Outcome Observation →
  Except AuthoringDiagnostic
    (CheckedTarget LawStatement Setup State Action Outcome Observation)
```

`elaborateTarget` emits the same typed failure at the captured Lean source occurrence.
`checkedTarget` produces the value for a declaration that compiles as valid while hiding extraction
and proof-relation re-ascription. Both `AuthoredTarget` and `CheckedTarget` have sealed constructors;
ordinary maintainers use `TargetComposition.provide`, `TargetComposition.connect`, and
`AuthoredTarget.make`. `composeTarget` remains the lower-level
`Except DefinitionError` expert seam. `canonicalCheckedTargetJson` returns the checked target's
canonical projection; compiler-only occurrence spans never enter it.

## Property API

`Umpire.Property` describes portable claims over capability-limited semantic traces.

Its internal modules form a downward implementation chain. `Umpire.Property.Language` owns the
authoring vocabulary, `Umpire.Property.Check` owns typed checking and canonicalization,
`Umpire.Property.Trace` owns capability-limited trace projection, and
`Umpire.Property.Evaluation` owns clause semantics and evaluation results. Ordinary authors should
reach all four through `Umpire.Property` rather than importing them directly.

The main authoring types are:

- `PropertyPattern`
- `ValueConstraint`
- `PropertyLimit`
- `PropertyClause`
- `PropertyDeclaration`
- `PropertyCheckContext`

`PropertyClause` supports state invariants, transition contracts, identity relations,
input/output relationships, ordering, bounded eventuality, and bounded quiescence.

Main entry points:

```lean
PropertyCheckContext.ofTarget
checkProperty
checkedProperty
PropertyPattern.exact
evaluateProperty
```

`checkProperty` returns either `PropertyError` or `CheckedProperty`. `evaluateProperty` reduces an
unrestricted semantic trace to the checked property's admitted capability view before evaluating
its clauses. `PropertyPattern.exact` is the narrow constructor for matching one trace field by
Definition ID and exact payload; it returns the existing `PropertyPattern` representation.

## Behavior API

`Umpire.Behavior` constrains setup and trace shape without assigning target outcomes.

Important types:

- `ResourceRole`, `RoleBinding`
- `SetupConstraint`
- `NamedOccurrence`, `OccurrenceBound`, `OccurrenceOrder`
- `AuthoredExactTrace`
- `BehaviorDeclaration`
- `CheckedBehavior`
- `BehaviorTrace`

Convenience constructors include:

```lean
OccurrenceBound.exactly
OccurrenceBound.atLeast
OccurrenceBound.atMost
SetupConstraint.roleEquals
BehaviorTrace.singleStep
BehaviorDeclaration.exactlyOneAction
```

The main entry point is:

```lean
checkBehavior :
  BehaviorCheckContext →
  BehaviorDeclaration →
  Except BehaviorError CheckedBehavior

checkedBehavior :
  (context : BehaviorCheckContext) →
  (declaration : BehaviorDeclaration) →
  (checkBehavior context declaration).toOption.isSome = true →
  CheckedBehavior
```

Ordinary callers derive the context with `BehaviorCheckContext.ofTarget target`; direct declaration
contexts remain confined to Behavior's focused lower-level fixtures.

Checking validates identities and references, canonicalizes constraints, rejects contradictions,
and records whether the described behavior space is statically unsatisfiable. It does not select a
target or enumerate a trace.

The semantic constructors above fill only the repeated invariant named by each function:
role-to-value equality, one transition result, or exactly one required Action. They return the
existing Behavior records, do not normalize caller collections, and leave Model Outcomes to the
Target.

`CheckedBehavior.assignOccurrences` canonically attributes selected action positions to authored
required occurrences. Behavior admission and Artifact linear extensions cross this same seam.

## Observation API

`Umpire.Observation` describes, checks, and applies bounded mappings from typed synthetic Evidence
to Evidence-backed Model Traces. Import the complete public surface with:

```lean
import Umpire.Observation
```

Ordinary authors stay on this facade. `Umpire.Observation.Declaration` internally owns the inert
authored vocabulary and field projections. `Umpire.Observation.Compiler` owns DefinitionGraph-backed
checking, Target-meaning resolution, typed errors, checked-plan contracts, and canonical plan
construction. `Umpire.Observation.Language` aggregates those modules and retains the explicit-proof
convenience; the child modules are contributor seams rather than alternative authoring surfaces.

Define each typed Evidence field once as an `ObservationFieldSpec`. This inert value is the one
field projection authority: it owns the Evidence kind identity, field identity, and
`ObservationValueType`, then projects the exact existing profile declaration, field reference,
field expression, and author-selected disposition declaration. The profile kind list, mapping
rules, closure declarations, and disposition list remain explicit, and `checkObservation` remains
their semantic authority. Direct construction of the projected records remains an expert and
compatibility path; it does not create a second field meaning.

The offline lifecycle is:

```text
EvidenceProfileDeclaration + ObservationMappingDeclaration + CheckedTarget
  ── ObservationCheckContext.ofTarget / checkObservation or checkedObservation
  ──▶ CheckedObservationPlan
CheckedObservationPlan + synthetic EvidenceBundle
  ── evaluateEvidence ──▶ ObservationResult
ObservationResult.accepted ──▶ opaque EvidenceBackedTrace
CheckedQuery + CheckedProperty + EvidenceBackedTrace
  ── evaluateObservationProperty ──▶ SemanticPropertyVerdict
CheckedQuery + property verdicts
  ── summarizeQueryVerdicts ──▶ StrictQuerySummary
```

`checkObservation` compiles the closed expression grammar, declared ordering and closures, Model
Facts, field dispositions, and positive `evidence-records` Limit into one canonical
`CheckedObservationPlan`. `evaluateEvidence` is the single Observation admission handoff. It either
returns `ObservationResult.accepted` with one complete, opaque `EvidenceBackedTrace` and an Evidence
Link for every Core-owned Model Coordinate, or one closed diagnostic without exposing a partial
trace. The accepted type has no public constructor or record-update path; documented field-style
projections provide read-only access to the semantic content needed downstream. It is a checked
handoff value, not another authoring or scenario language. The raw
`EvidenceBundle` is consumed only during Observation Evaluation and is not retained in the
Evidence-backed Model Trace or verdicts.

`Observation.Internal.analyzeStructure` is the single shared internal mechanism for normalizing
Evidence identities, sequence and causal order, closure expectations, and per-link support. Raw
Evidence admission and accepted-trace provenance each adapt their own inputs into that analysis
once, then map its findings at their own boundary. Raw admission therefore keeps its detailed
identity, ordering, causality, fault-target, and closure diagnostics, while accepted provenance
reports only its established missing-order-support or missing-closure-support diagnostics. The
internal analysis is mechanics, not a public validator or another diagnostic vocabulary.

For global sequencing, an explicit closure for a checked required kind that has no records closes
at sequence zero. A nonempty global kind still closes at its computed final sequence; source-local
closures still match source and kind and retain their ordinal, record-count, and byte-count checks.
Accepted provenance checks global required closures in the checked plan's closure order after its
existing link-consistency and ordering checks. This preserves boundary-owned first-failure
precedence without exposing any raw-Evidence diagnostic through the accepted trace.

As with the other authoring languages, `checkObservation` remains the typed diagnostic API.
`checkedObservation` accepts an explicit success proof and returns that checker's complete plan; it
does not replace or bypass checking.

`evaluateObservationProperty` accepts only the admitted trace and validates Property-owned query
membership, exact Property identity, capability access, and logical-time prerequisites before
invoking the unchanged Property evaluator. It does not repeat Observation envelope admission. An
Observation non-success carries no accepted trace, so Property evaluation cannot produce a partial
or layer-confused result.

Every consumed field has exactly one disposition:

- `retain` may preserve its approved normalized value;
- `redact` may preserve only a contribution marker;
- `hash` may preserve only a deterministic token under the mapping's named, versioned synthetic
  digest policy;
- `reject` prevents Observation Evaluation when that field is present and cannot be read by a mapping.

Observation Evaluation statuses are `accepted`, `unknown`, `conflict`, and `unsupported`. Property verdicts
are independently `satisfied`, `violated`, `unknown`, `conflict`, or `unsupported`; Observation Evaluation
failure never becomes a Property violation. Strict aggregation is `satisfied` only for exactly one
resolved satisfied verdict per required Property over the same Query, Model Trace, and Evidence Limit. It
is `violated` only when that result set is structurally complete and resolved but contains a
violation. Missing, duplicate, unexpected, divergent, wrong-query, cross-trace, cross-Limit, or
unresolved results make the summary `incomplete`.

### Runtime adapter handoff

The exact typed input a runtime adapter provides is one complete `EvidenceBundle`:

- `profile` and `profileVersion` select the checked evidence schema;
- `records` contain `id`, the same `profile` and `profileVersion`, `kind`, one-based `sequence`, and
  `causalParents`; each record's `fields` pair a field identity with a typed text, natural, or
  Boolean value plus optional `digestPolicy` and `reportedDigestToken`, while `bindingFacts` pair a
  binding identity with a typed value and `faultTarget` optionally identifies the intended target;
- `closures` pair each required evidence kind with its final sequence;
- `compatibleAlternatives` preserve alternative identities and their evidence identities instead
  of selecting one silently;
- `missingDiscriminator` names the fact needed to distinguish those alternatives.

An adapter owns translation into these fields while preserving source identity, ordering,
causality, closure, ambiguity, and declared field types. It hands the complete bundle to
`evaluateEvidence`; it does not construct an `EvidenceBackedTrace`, choose an Observation status,
or reinterpret a Property result. The fixed caller-closure Run Evaluation adapter implements this
handoff for its exact normal and duplicate-delivery profiles. The portable compiler additionally
lowers those same checked Observation plans into closed per-Test contracts for the fixed Go
interpreter. Neither path admits a generic profile, changes Property meaning, performs replay or
promotion, or establishes a non-local Claim Assessment.

## Implementation Link API

`Umpire.ImplementationLink` relates two independently checked Targets without making either Target
import or redefine the other. An author supplies one inert `ImplementationLinkDeclaration` with
finite setup, state, action, target-outcome, observation, relation, and capability mappings; an
explicit support/Known Gap partition; one positive application Limit; and an
`ImplementationLinkWitness` indexed by that exact declaration and those exact checked Targets.
`checkImplementationLink` validates the complete declaration and witness before returning one
canonical `CheckedImplementationLink`.

`ImplementationValueMapping.forward` and `ImplementationSemanticMapping.forward` are inert,
additive shorthand for their existing public source/destination records. Each records exactly the
two values supplied by the caller; neither performs lookup, checking, inference, or registration.
Raw mapping records remain available, and `checkImplementationLink` remains the semantic authority.

`KernelMorphism` is the reusable proof-free mapping of setup, state, Action, Model Outcome, and
observation values. Its step and trace translation reuse Core `TransitionResult.map` and
`ModelTraceStep.result`. `ForwardSimulation` adds only the initial-state and step-preservation laws
and derives whole-trace preservation. Both are available from the focused
`Umpire.ImplementationLink` facade. The surrounding Implementation Link remains responsible for
declaration indexing, complete mapping coverage, Known Gaps, fingerprints, Limits, and typed
diagnostics.

The prototype proves a bounded forward simulation. It does not require a reverse mapping,
bisimulation, surjectivity, or named Behavior-occurrence correspondence. `applyImplementationLink`
accepts only an admitted `EvidenceBackedTrace`; it does not repeat Observation envelope admission.
It validates its own application Limit and source-Target identity, replays the accepted trace's
source setup, initial state, and every step through the checked source kernel, and checks the
declared mapping, support/Known Gap partition, and translation. It then returns one complete
authoritative destination Model Trace with a coordinate-complete
`ImplementationLinkEvidenceLink` set, or one Link-owned diagnostic without exposing a partial
destination trace.

Implementation Link application has its own `invalid`, `unknown`, `conflict`, and `unsupported`
outcomes and canonical diagnostics. Those outcomes are distinct from Observation Evaluation
(`accepted`, `unknown`, `conflict`, or `unsupported`) and from Feature Property evaluation
(`satisfied` or `violated` for the direct evaluator). Implementation Link does not interpret raw
Evidence and does not invoke a Property; a later Run Evaluation passes the accepted trace across
this handoff and invokes later stages only after their prerequisite succeeds. It composes the three
checked stages without repeating Observation admission or collapsing stage identities,
diagnostics, or no-partial-result guarantees.

## Query and search API

`Umpire.Query` combines a checked Target, checked Properties, checked Behavior, finite Limits, and
deterministic planning policy.

Important types:

- `QueryTarget`
- `QueryForm`
- `QueryDeclaration`
- `CheckedQuery`
- `FiniteCompletenessEvidence`
- `QueryCheckContext`
- `QueryLimits`
- `PlannerPolicy`
- `SearchStrategy`

Ordinary policies use `PlannerPolicy.shortest`, `PlannerPolicy.exhaustive`, or
`PlannerPolicy.seeded`; all use Definition-ID tie-breaking and seed `17` by default. Passing a
natural number to `PlannerPolicy.seeded` preserves that exact seed, including zero. A policy's seed
is always part of canonical Query identity, but only the seeded strategy uses it to change
traversal. The public `PlannerPolicy` record remains available for exceptional strategy, seed, or
tie-break combinations.

`QueryForm` determines the meaning of the result:

- `verify` searches for bounded universal verification.
- `witness` searches for a satisfying trace.
- `counterexample` searches for a violating trace.
- `select` performs bounded exploratory selection.

The checked entry point is:

```lean
checkQuery :
  QueryCheckContext LawStatement →
  QueryDeclaration →
  Except QueryError (CheckedQuery LawStatement)

checkedQuery :
  (target : QueryTarget LawStatement) →
  (declaration : QueryDeclaration) →
  (checkQuery (.ofTarget target) declaration).toOption.isSome = true →
  CheckedQuery LawStatement
```

`QueryCheckContext.ofTarget` derives the Query view from the checked Target, including any available
finite completeness contract. An exhaustive Query rejects a Target that explicitly lacks that
Capability. `QueryLimits` keeps Behavior-space Limits separate from the planner's
candidate-evaluation budget. `checkedQuery` keeps the dependent Target re-ascription inside the
Query boundary; `checkQuery` remains the API for inspecting a `QueryError`.

`QueryLimits.bounded` is inert, additive shorthand for the existing public `QueryLimits` record.
It records caller-supplied transition, selected-Action, and candidate-evaluation bounds and fixes
only their established units. It does not validate or reconcile the bounds; raw records remain
available, and `checkQuery` retains all Limit policy and diagnostic authority.

## Space API

`Umpire.Space` composes one checked Query into a finite authored variation Space. Its axes choose a
baseline, bind at most one existing Behavior role to an existing checked value, or select declared
fault intents. Faults name one required Behavior occurrence and one target Capability; they are
requested attempts, not outcomes, observations, receipts, or success claims. Coverage goals are
seek-only metadata and do not change Property meaning or claim runtime achievement.

The ordinary leaf constructors are:

```lean
ChoiceDeclaration.baselineChoice
ChoiceDeclaration.boundValue
ChoiceDeclaration.selectedFault
VariationAxisDeclaration.faultAxis
FaultIntentDeclaration.atOccurrence
CoverageGoalDeclaration.seek
```

Each is inert, additive shorthand for its existing public record: an effect-free baseline, one
explicit binding, one explicit fault selection, a role-free fault axis, an occurrence fault with
no incompatibility declarations, or one explicit seek goal, respectively. They fill only those
established ordinary fields and defaults; they do not infer or validate identities, roles, values,
faults, occurrences, Actions, Capabilities, subjects, or minima. Raw records remain available for
non-default declarations, and `checkExperimentSpace` remains the semantic authority.

The principal entry points are:

```lean
checkExperimentSpace :
  SpaceCheckContext LawStatement →
  ExperimentSpaceDeclaration →
  Except SpaceError (CheckedExperimentSpace LawStatement)

projectCheckedSpaceMetadata :
  CheckedExperimentSpace LawStatement →
  Except SpaceMetadataError CheckedSpaceMetadata

lowerSpacePoint :
  (space : CheckedExperimentSpace LawStatement) →
  List ModelValue →
  Except SpaceCompilationError (LoweredSpacePoint space)

compileBatch :
  (space : CheckedExperimentSpace LawStatement) →
  IncrementalPlannerKernel space.baseQuery.target →
  Except SpaceCompilationError (List ExperimentSpec)
```

Checking validates the fixed finite bounds and canonical identities before any point can lower.
`lowerSpacePoint` rechecks a derived Behavior and Query, retains a proof that the Query target is
unchanged, and produces checked Artifact intent without planning. `compileBatch` transports the
caller-owned base kernel through that proof, compiles every canonical point, and returns either the
complete ordered batch or one typed error with no partial batch. Target-owned planning supplies all
outcomes.

`CheckedSpaceMetadata` is the canonical in-memory, source-backed projection consumed by catalog
aggregation and list/explain generation. Space does not persist a registry. Exploration consumes
the checked Space and its compiled points; Space itself does not select a subset, score coverage,
maintain coverage state, execute a runtime, or evaluate conformance.

## Exploration API

`Umpire.Exploration` is the reusable pure selection layer over one checked finite Space and its
exact planner kernel. `ExplorationRequest` admits only `exhaustive` or
`uncoveredCoordinate coordinate`, one positive `experiment-specs` Limit, and a list of canonical
ExperimentSpecs proposed as pinned Regressions. `checkExplorationRequest` validates those inputs
atomically and returns an opaque `CheckedExplorationRequest`; invalid Limits, unknown coordinates,
invalid or duplicate pinned Artifacts, and incompatible target or kernel bindings expose no
partial checked value.

The retained selectors are exactly:

```lean
selectExhaustive :
  CheckedExplorationRequest LawStatement → CandidateUniverse → Option ExhaustiveSelection

selectUncoveredCoordinate :
  CheckedExplorationRequest LawStatement → CandidateUniverse → Option GuidedSelection
```

Both reject crossed checked inputs. Exhaustive selection reports `exhausted` only when every
eligible candidate in the finite universe was considered. Stopping at the Limit reports
`limit-reached`, which is inconclusive and proves no absence claim. Coordinate guidance puts exact
matches first and breaks ties by ExperimentSpec semantic identity. Its
`coordinate-selected|coordinate-uncovered` outcome describes only the selected candidates; a
requested coordinate or fault remains Model intent and is not Evidence of runtime realization or
Property satisfaction.

`explore` composes checking, complete candidate compilation, and one selector into the narrow
`ExplorationResult`. Checked pinned Regressions are identity-sorted in a separate partition, appear
before exploratory candidates, consume none of the exploration Limit, and win identity overlap
through `pinned-precedence`. `beginSession`, `ExplorationSession.next`, and
`ExplorationSession.observe` expose one fixed selected order with at most one outstanding candidate
and exact Artifact-binding admission. The session is process-local and has no persisted encoding,
checkpoint, compatibility version, or restart contract.

## Promotion API

`Umpire.Promotion` accepts an unchanged checked Query, its target-indexed planner kernel, a complete
base `PlannerRun`/`ExperimentSpec` anchor, fixed fresh declaration identities, and exact expected
source bytes. It replans and rechecks the target-owned trace before returning one opaque
`CompiledPromotionSource`; drift or non-`.found` planning returns a typed error and no source.

The current Temporal caller-closure binding supplies the base Query's expected cancellation count
of one and keeps the selected duplicate-delivery `ExperimentSpec` in a separate Temporal-owned
proposal lineage. That proposal remains inert review material. Runtime reproduction, complete
reduction, Exact Replay, eligibility, publication, and installation are outside this reusable API;
fn-22 alone owns the current runtime eligibility gate.

## Planning API

`Umpire.Planning` performs deterministic, incremental enumeration of a checked target.

The principal types are:

- `IncrementalPlannerKernel`
- `FiniteKernelOrder`
- `PlanningOutcome`
- `PlanningResult`
- `PlannerRun`
- `PlannerInstrumentation`

The main entry point is:

```lean
plan :
  (query : CheckedQuery LawStatement) →
  IncrementalPlannerKernel query.target →
  PlannerRun
```

`IncrementalPlannerKernel.ofCheckedQuery?` derives indexed enumeration, Limits, soundness, and
completeness from the checked Query and Target-owned finite domain. A model maintainer supplies only
the canonical-order proofs; ordinary Query authors do not construct a planner kernel.

`IncrementalPlannerKernel` exposes indexed action, initial-state, and transition enumeration. Its
proof fields establish soundness, completeness, and canonical ordering relative to the selected
target.

`PlanningOutcome` distinguishes a selected trace, bounded verification, complete absence of a
matching trace, budget exhaustion, unsatisfiable behavior, and an invalid query. `PlannerRun`
contains the outcome, optional artifact, and instrumentation.

Planning also owns the `KnownGap` row vocabulary and its opaque `KnownGapSet`. Trusted Lean
producers use `KnownGapSet.ofUnordered` to sort and check authored rows; strict decoder adapters use
`KnownGapSet.checkCanonical`, which rejects a noncanonical external order instead of repairing it.
Both paths reject invalid identifiers, duplicates, and conflicting details. After admission,
semantic consumers accept only the checked set, compose phase-owned gaps with `KnownGapSet.union`,
and use `KnownGapSet.toList` only for a read-only canonical projection. The representation has no
public constructor or record-update path.

## Artifact API

`Umpire.Artifact` exposes its vertical modules in dependency order: Planning, Runtime, Evidence,
Result, and Set. Planning owns the portable, environment-independent products of pure model
planning; the later modules define inert transport records for runtime facts and semantic results
supplied by their owning downstream stages. Admission verifies those records but never invents an
Execution, Observation Evaluation, Implementation Link, Property, or Run Evaluation outcome.

- `DrivePlan` records bindings, selected choices and variants, requested faults, requested actions,
  model-owned outcomes, resulting states, checkpoints, Limits, selection reason, and provenance.
- `ExperimentSpec` is the portable envelope consumed by later execution, checking, replay, and
  generation work. Planning and `umpire-gen-tests` emit only byte-identical
  `umpire-experiment/v2`; runtime-specific participant, setup, ordering, termination, and cleanup
  bindings belong to the later `RuntimeConfiguration` boundary.
- `DrivePlan` and `ExperimentSpec` use one fixed-order, two-space-indented JSON spelling with stable
  escaping and base-10 naturals, no trailing spaces, and exactly one terminal LF. Each Artifact
  Checksum hashes that exact pretty preimage with only its own checksum omitted; the outer preimage
  retains the already-sealed DrivePlan.
- `artifactOfSelection` constructs an `ExperimentSpec` from a checked query and a selected,
  kernel-produced `BehaviorTrace`.
- `checkExecutionHandoff` retains reusable validation for model-owned lifecycle references without
  changing ExperimentSpec bytes or giving Space another persisted schema.

Artifact and Planning codecs own field names, field meaning, and field order. They use the focused
`Umpire.Json` module's `CanonicalJson` value to construct typed nulls, strings, naturals, arrays,
and ordered objects, then render compact text, stable pretty text, or persisted bytes with exactly
one terminal LF. `CanonicalJson` preserves supplied object order and Lean's JSON string escaping;
it is not a parser, schema engine, unordered map, or alternate Artifact format.

The retained boundary is exactly embedded `umpire-drive-plan/v2` plus persisted
`umpire-experiment/v2`, `umpire-runtime-configuration/v2`, `umpire-experiment-run/v2`,
`umpire-raw-evidence/v2`, `umpire-evidence/v2`, and `umpire-result/v2`. The latter five keep phase
Limits, Known Gaps, operational status, Observation Evaluation, Evidence Links, Implementation Link
status, Property verdicts, Run Evaluation, and cleanup status distinct.

Lean Planning, Runtime, Evidence, and Result records carry checked `KnownGapSet` values and expose
their ordered rows only as a read-only projection at explicit codec, protocol, or semantic-adapter
boundaries. The persisted representation remains the unchanged raw JSON array. Go therefore admits
every such array independently: it validates kinds, namespaced identifiers, canonical order,
uniqueness, and subject-detail consistency, then verifies the checker response projections and exact
Known Gap union without trusting Lean's opaque in-memory value.

`Umpire.Artifact.PortableEvaluationContract` is a focused build-time module outside the ordinary
`Umpire.Artifact` facade. It defines the inert version-one contract vocabulary and its canonical
ProtoJSON spelling. A Temporal-owned compiler specializes one selected checked caller-closure Test
into that value; generated protobuf code and Go admission do not add or select behavior. The runtime
artifact is deterministic `EvaluationContract` protobuf bytes, not a new member of the persisted
two-, four-, or six-member v2 JSON sets.

Canonical serialization entry points include:

```lean
canonicalCheckedTargetJson
canonicalPropertyJson
canonicalBehaviorJson
canonicalQueryJson
canonicalExperimentSpaceJson
canonicalDrivePlanJson
canonicalExperimentSpecJson
```

This deterministic pretty v2 spelling is an in-place pre-release correction that supersedes
fn-37's compact bytes and compact checksum preimages. No external or immutable published v2 set
predates the correction, and there is no compact reader, alternate writer, migration, alias, or
fallback.

Go admission rejects unsupported formats, wrong families, duplicate or case-colliding keys,
unknown fields, malformed or oversized values, noncanonical bytes, stale checksums, unsafe set
paths, and incomplete cross-document closure. Exact set admission accepts only ExperimentSpec plus
RuntimeConfiguration; those two plus ExperimentRun and RawEvidence; or those four plus Evidence and
Result.

The root `umpire-check-artifact` and `umpire-check-artifact-set` targets are read-only admission
checks. They are silent on success and never publish as a side effect. `PublishSet` remains an
explicit Go API: it stages and revalidates a complete admitted set privately, installs its immutable
manifest-digest directory with one rename, and lets `LoadSet` return only a complete revalidated
snapshot. Interrupted private staging is cleaned under the publication lock; generic artifact
management, schema migrations, receipt envelopes, runtime execution, CI, and other platform work
remain separate.

Canonical error projections are available as `canonicalDefinitionErrorJson`,
`canonicalPropertyErrorJson`, `canonicalBehaviorErrorJson`, and `canonicalQueryErrorJson`.

DrivePlan and ExperimentSpec do not claim that a runtime action occurred or that execution Evidence
was collected. ExperimentRun, RawEvidence, Evidence, and Result truthfully transport supplied
statuses and facts, but their mere admission does not prove an action, Evidence interpretation,
Property verdict, or Claim Assessment. The model-owned `umpire-gen-tests` tool accepts named
regressions, test sets, and model-selected batches without exposing discovery or explanation; Space
exposes no competing command.

## Portable Evaluation Contract

Portable Evaluation is a separate ahead-of-time path for one exact checked Test:

```text
checked Test + Observation + Implementation Link + Properties
  ── Temporal-owned Lean compiler ──▶ canonical contract ProtoJSON
  ── structural Go packer ──▶ deterministic EvaluationContract protobuf
  ── resident Go executor ──▶ detailed EvaluationResult + pass/fail/inconclusive
```

The Lean compiler is the only semantic compiler. It rejects checked constructs outside the closed
version-one vocabulary instead of asking Go to infer an approximation. The contract binds the exact
Experiment, RuntimeConfiguration, Test, Query, Observation program and profile, Implementation
Link, Properties, Definition IDs, Behavior Fingerprints, Limits, Known Gaps, and provenance. The Go
packer validates shape and canonical order and seals deterministic protobuf bytes; the interpreter
consults no Lean runtime or model registry.

The version-one vocabulary contains Observation literals, typed fields, natural rendering,
presence, equality, ordered `all`/`any`, coordinate `emit`, exact Link renaming,
`per_step_implies`, and exact-text or bounded-natural Property patterns. It has no executable,
callback, registry lookup, environment selector, credential, arbitrary network target, or extension
hook. Unknown versions, fields, enum values, operators, noncanonical bytes, crossed bindings, and
invalid Limits fail closed before execution or at their responsible evaluation stage.

Every required Evidence source must close explicitly with matching source identity, status, record
count, and byte count before absence can establish a fact. The resident executor waits for the
bounded runner's closure rather than inferring closure from wall-clock quiet. Missing, partial,
stale, crossed, or post-closure Evidence and deadline-before-closure results are inconclusive. Every
accepted Evidence Link retains source-local or causal ordering and closure support.

`EvaluationResult` keeps tooling, operational, Observation Evaluation, Implementation Link,
Property/clause, aggregate semantic, cleanup, work, Known Gap, and diagnostic fields independent.
Local `pass` requires a fully successful, closed, accepted, applied, satisfied, and cleaned-up run.
Local `fail` requires the same trustworthy closure with a Property violation. Every operational,
closure, tooling, unknown, conflict, unsupported, cancellation, Limit, or cleanup failure is
`inconclusive`.

One resident executor is single-flight and may serve sequential requests with fresh run identities.
Overlap returns typed `busy` before runtime I/O; uncertain cleanup permanently poisons that executor
instance. Its HTTP adapter is a bounded deterministic protobuf `POST /umpire/v1/execute` endpoint,
not gRPC, and adds no semantic or environment-selection surface.

The successor interface is one fully typed, caller-neutral `PortableTestPlan` containing both the
bounded execution and verification programs. Lean is its first model compiler, not its exclusive
author: an external protobuf client can establish only `plan_local` conformance, while a
`model_compiled` plan receives `model_bound` scope only after exact host-owned provenance
validation. The generated unary `UmpireExecutor.Execute` gRPC method is a thin adapter over the
same deep resident executor. It returns canonical status for failures before a result exists and
preserves admitted operational, Evidence, semantic, Limit, and cleanup outcomes in the typed
`ExecutionResult`. It adds no plan store, queue, scheduler, retry, fleet, environment selector, or
credential surface. Fn-28's HTTP descriptor, bytes, types, and behavior remain unchanged.

The tagged `testcore.NewEnv` proof keeps one disposable self-hosted cluster and one executor alive
for the pre-generated normal and duplicate-delivery contracts. It observes a local pass and a
trustworthy uniqueness-only fail with fresh run resources while the runtime `PATH` contains no Go,
Lean, Lake, Mise, Make, or shell executable. Stable comparison covers model/artifact bindings,
typed traces, Evidence Links, independent statuses, Properties, Limits, Known Gaps, cleanup, and the
local decision; executor run IDs, workflow IDs, task queues, per-run Nexus endpoint names,
correlations, transport Evidence IDs, and timestamps remain dynamic.

The companion tagged gRPC proof keeps one generated client and resident in-process server alive on
that cluster. It consumes checked-in Lean plans and a derived external plan without any runtime
toolchain, independently evaluates normal pass and trustworthy fail, and exercises ten-call
single-flight admission, malformed/crossed/forged input, exact bounds, cancellation and deadline
cleanup, poison, no retry, and fresh run isolation. Fn-29's protected controller will pin and
provenance-validate one Lean plan before using this ingress; its public Temporal gRPC target,
production authority, fencing, recovery, and publication remain downstream and separate.

This is only a local decision for the admitted Test. It is not whole-model validity, exhaustive
coverage, compiler correctness, cross-Test consistency, release eligibility, or Claim Assessment,
and it adds no fleet scheduling, leases, persistence, crash recovery, or production deployment. See
the [Portable Evaluation guide](../../tools/umpire/portableevaluation/README.md) for the exact
schema, operators, Limits, fixture workflow, executor HTTP/gRPC contracts, and test commands.

## Expert reference example

[`Umpire.Examples.Switch`](Examples/Switch.lean) is the smallest complete direct-kernel example.
Its authoritative relation is specified independently of its enumerators, so it deliberately uses
the expert `TransitionKernel` route. Read its public declarations in lifecycle order:

1. `transitionKernel`
2. `targetDeclaration`, `targetAuthoring`, and checked `target`
3. `propertyDeclaration` and `flipProperty`
4. the behavior declarations and checked behaviors
5. the query declarations and checked queries
6. `incrementalKernel` and the planner runs
7. `compiledArtifact`

The example exposes exploratory, exact-action, and exact-trace variants without depending on
Temporal-specific modules.

## API invariants

- Authored declarations remain distinguishable from checked values.
- Invalid inputs cross public boundaries as typed `Except` errors.
- Behavior Fingerprints derive from canonical projections, not source declaration order.
- Transition outcomes belong to the target model, not to behavior declarations.
- Space fault intent remains request-only, and coverage goals remain seek-only metadata.
- Planning is pure and performs no runtime execution, evidence collection, or promotion.
- Reusable Umpire modules must not depend on Temporal-specific modules.
