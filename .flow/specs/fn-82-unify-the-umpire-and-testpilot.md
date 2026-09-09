## Goal & Context
<!-- scope: business -->

Status: implementation specification derived from the 2026-09-08 vocabulary investigation of
`model/`, the Testpilot protocol, and the Go facade. [UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md)
remains normative and is itself rewritten by this spec under GOV-02.

The investigation found that the model works but its vocabulary does not. A Temporal engineer who
reads a Nexus3 model, then the Umpire types behind it, then the Case that comes out, meets the
same idea under several names and the same name for several ideas. Measured on the current tree:

| Word | Distinct meanings today | Examples |
| --- | --- | --- |
| Observation | 6 | model fact (`ModelTraceStep.observations`), `DefinitionKind.observation`, the offline `Umpire.Observation` evaluator, the live `Umpire.Observation.Projection` seam, the Testpilot declared Run value, `Artifact.ObservationConfiguration` |
| Projection | 9 | `Target.Projection` (sorting and JSON), `Observation.Projection` (live evidence reader), proto `ResponseProjection`, `ScopedProjectionRule`, `SpaceMetadataProjection`, `ProjectionSentinelDescriptor`, the Go golden comparison |
| Evidence | 8 | `RawEvidence`, `EvidenceArtifact`, `EvidenceBundle`, `EvidenceProfileDeclaration`, `EvidenceLink`, `Shared.ScopedProjection.Event`, proto `ScopedEvidence`, `EVIDENCE.md` (test coverage) |
| Target | 6 types | `TargetDefinition`, `TargetDeclaration`, `FiniteTargetDefinition`, `AuthoredTarget`, `CheckedTarget`, `TargetComposition`, plus `QueryTarget` and `CheckedQueryTarget` |
| Outcome | 5 | model outcome, `InstructionOutcome`, `CleanupOutcome`, `ScopedTransition.outcome`, `PlanningOutcome` |
| Capability | 5 | Go opcode enum `testpilot.Capability`, Profile capabilities, `OpaqueCapability`, `CapabilityBridge`, Umpire `CapabilityContract` |
| Case | 3 | the Testpilot Case, `PropertyCase` branches, `Umpire.Planning.CaseAnalysis` |
| Run | 4 | Testpilot `Run`, `Umpire.Property.Scoped.Run`, `Testpilot.Scoped.Run`, `PlannerRun` |
| Scoped | 1 undocumented | 24 proto names and 40 Lean files carry the prefix; nothing states that it means "tracked per operation, correlated by a key" |

The reverse problem is as common. The transition relation is a `TransitionKernel`, a
`FiniteMachine`, a `FiniteTable`, or a `ValidatedFiniteModel` depending on the file. A model fact is
`Fact` in `FiniteTable`, `Observation` in `FiniteMachine`, `observations` on `TransitionResult`,
and `expectationFact` in a Property. Something that must hold is a Property, a Clause, an
Obligation, a Requirement, a Claim, a Law, or a Rule. A budget is a Limit, a Bound, a Ceiling, a
Horizon, or `bounds`. The three-valued satisfied/violated/inconclusive result is `Verdict`,
`PropertyEndpointAnswer`, or `ScopedObligation.Answer`. The spec's own word for a constrained set of
traces is Scenario; the type is `Behavior`, which also names the Behavior Model and the Behavior
Fingerprint.

Some vocabulary is dead weight: `Umpire.ExecutionHandoff` (no production consumer),
`Umpire.Target.Language` (zero declarations), `Shared/Transition.lean` and `Shared/TraceReplay.lean`
(no importer), the `Umpire.Case` alias family (test-only consumers, each file says "remove this
alias"), five `DefinitionKind` constructors that exist only to be rejected, a 142-structure duplicate
Testpilot mirror inside `Temporal.API` with zero consumers, `TieBreakPolicy` and `PropertyScopedClock`
with one constructor each, and `QueryQuantifier` plus `QueryClaim`, both derivable from `QueryForm`.

This spec fixes one vocabulary, applies it across Lean, protobuf, Go, documentation, tool names,
and the author-facing command syntax, deletes the dead names, and makes the result enforceable.
Breaking changes are accepted throughout. Nothing is versioned, aliased, or deprecated: old names
are retired and the retired-vocabulary gate rejects them.

Sequencing against open work: fn-77 tasks .9 to .11 are in flight in `Umpire.Operation`,
`Umpire.Value`, and the Nexus3 Producer; fn-80 rewrites the Nexus3 command syntax, the Nexus3
Producer, `contract.proto`, `run.proto`, `instruction.proto`, and the Go facade; fn-81 deletes the
pre-Testpilot Go trees and their Makefile blocks; fn-67 has one open documentation task on
`Nexus3/Nexus.md` and `Integration.md`. fn-80 tasks .4 to .8 also edit `Umpire/Target`,
`Umpire/Property`, `Umpire/Query`, `Umpire/Space`, and `Umpire/Case`, so no area of this spec can
land without a byte conflict while those are open. This spec therefore depends on fn-77, fn-80,
fn-81, and fn-67 at the spec level and starts after they close. It does not touch fn-77's five
in-flight terms (typed operation, parameterized Action, field-level Property, occurrence, capture).

## Architecture & Data Models
<!-- scope: technical -->

### The vocabulary

One word per concept, used identically in Lean, proto, Go, docs, and the command syntax. This
table is normative for the whole spec; the per-area sections below say where each word lands.

| Concept | Word | Retires |
| --- | --- | --- |
| The checked behavior model authors write | **Model** (`DraftModel` before checking, `CheckedModel` after, `ModelSpec` for the author record) | Target, `AuthoredTarget`, `CheckedTarget`, `TargetDefinition`, `TargetDeclaration`, `QueryTarget`, `TargetComposition` (becomes `Providers`) |
| The transition relation with its proofs | **Machine** | `TransitionKernel`, `KernelMetadata`, `KernelAvailability`, `DefinitionKind.kernel` |
| The finite row form of a Machine | **Table** (`FiniteTable`, `CheckedTable`) | `ValidatedFiniteTable`, `ValidatedFiniteModel`, `FiniteTargetDefinition` (becomes `TableModelSpec`) |
| The enumerated finite vocabulary of a Model | **Vocabulary** | `TargetBehaviorDomain`, `TargetBehaviorDomainAvailability`, `TargetBehaviorDescription` (becomes `BehaviorTable`), `TargetBehaviorClosure` (deleted) |
| One model step and what it produced | **Step** with `outcome`, `state`, `facts` | `TransitionResult` with `modelOutcome`, `resultingState`, `observations`; `ModelTraceStep` |
| A claim the model makes at a step | **Fact** | the `Observation` type parameter, `facts`/`observations`/`expectationFact` spellings, `DefinitionKind.observation` (becomes `.fact`) |
| A sequence of steps | **Trace**, `TraceAddress` | `ModelTrace`, `BehaviorTrace`, `ModelCoordinate` |
| A constrained set of traces | **Scenario** | `Umpire.Behavior`, `BehaviorDeclaration`, `CheckedBehavior`, `BehaviorSpec`, `ExactSequenceSpec` (becomes `Scenario.exactly`) |
| A pass/fail rule over traces | **Property** with **Clauses**; guarded alternatives are **Branches** | `PropertyDeclaration`, `PropertySpec`, `PropertyAuthoring`, `Resolved*`, `PropertyCase`, `PropertyCaseGroup`, `PropertyException` (becomes `Unless`), Obligation, Requirement, Claim, Law as synonyms |
| A rule tracked separately per operation, correlated by a key | **Correlated** | the `Scoped` prefix in Lean, proto, and Go |
| A bounded-time ceiling on a rule | **Deadline** | `ContractHorizonDefinition`, "horizon" |
| A ceiling on any stage | **Limit** with units `steps`, `actions`, `logicalTime`, `search`, `plans` | Bound, Ceiling, Budget, `bounds`, `semanticTransitions`, `selectedActions`, `candidateEvaluations`, `experimentSpecs`; `observationPositions` is deleted |
| A bounded question | **Query** with forms `verify`, `find`, `findViolation`, `pick` | `QueryDeclaration`, `QuerySpec`, `QueryAuthoringInput`, forms `witness`/`counterexample`/`select`, `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy` |
| Answering a Query | **Search** (`Umpire.Search`, `SearchView`, `SearchStats`, `PlanResult`) | `Umpire.Planning`, `Planning.Engine`, `IncrementalPlannerKernel`, `PlannerInstrumentation`, `PlannerRun` |
| The planned model-level test | **Plan** with `Plan.Steps` | `ExperimentSpec`, `DrivePlan`, "Planning Artifact", "Execution Plan" |
| The three-valued result | **Verdict** `satisfied`, `violated`, `inconclusive` | `PropertyEndpointAnswer`, `ScopedObligation.Answer`, `unresolved` |
| Per-run rule state | **Monitor** | `Property.Scoped.Run`, `Testpilot.Scoped.Run`; the Lean `Testpilot.Authoring.Monitor` namespace becomes `Contract` |
| Offline evaluation of captured runtime records | **Evidence** (`Umpire.Evidence`) | `Umpire.Observation.{Declaration,Compiler,Language,Evaluation,Check,Verdict}`, `SemanticVerdict*` (becomes `Evidence.PropertyStatus`) |
| Reading declared Run values into model steps and fields | **Projection** (`Umpire.Case.Projection`) | `Umpire.Observation.Projection*`, `Umpire.Observation.Evaluation.Scoped`; every other use of "projection" is renamed |
| A hole in a mapping's source domain | **UnmappedSource** | `ImplementationLinkKnownGap`, which is not a Known Gap |
| Umpire's own Case-lowering surface | **Case** (`Umpire.Case` with `Compiler`, `Coverage`, `Correlated`, `Projection`) | `Umpire.Case.Scoped`; the `Umpire.Case.{Program,Contract,Run,Value,ProtoJSON}` aliases are deleted |
| Producer-owned identity bytes inside a Case | **Provenance** (`Umpire.Provenance`) | the provenance half of `Umpire/Case.lean`, `CaseMetadata`, `CaseDefinitionBinding`, `CaseDefinitionKind`, `CaseKnownGap` |
| A finite space of authored variations | **Variations** (`Umpire.Variations`, `VariationSpace`) | `Umpire.Space`, `ExperimentSpace`, `ExperimentSpaceDeclaration` |
| The generated status and gap catalog | **Inventory** (`Umpire.Inventory`, `model/INVENTORY.md`) | `Umpire.SemanticInventory`, `SEMANTIC_INVENTORY.md` |
| A declared, provided behavior with laws | **Capability**, **Provider**, **Law**, **LawProof**, **Meaning**, **Connector** | `CapabilityContract`, `CapabilityProvider`, `LawDefinition`, `LawWitness`, `MeaningProvision`, `CapabilityConnector` |
| A source position for a diagnostic | **SourceRef**, **SourceSpan**, **LocatedError** | `AuthoringOccurrence*`, `AuthoringDiagnostic` |
| The Case instruction kind | **Opcode** | Go `testpilot.Capability`, `ProfileSpec.Capabilities` (becomes `Opcodes`) |

Words that keep their current meaning and spelling everywhere: `Case`, `Program`, `Contract`,
`Rule` (a Contract rule), `Profile`, `Driver`, `Run`, `Run Event`, `Slot`, `Observation` (a declared
typed Run value, nothing else), `Verdict`, `Definition ID`, `Known Gap`, `Implementation Link`,
`Promotion`, `Exploration`, `Producer` (the role that emits a Case, never a module name), `role`,
`Feature`, `System`, `DynamicConfig`, `Property`, `Query`, `Limit`, `ModelValue`.

`step` is one word for one concept on purpose: the `steps` block of a model lists the allowed
steps, `steps N` limits how many a trace may take, and `Plan.Steps` is the sequence the search
chose. A retired name enters the gate only when it is a compound identifier, a module path, a
macro name, or a snake_case keyword. Bare words such as `Target`, `Behavior`, `Case`, or `Step`
are never retired, because the gate also matches their lowercase form and would ban ordinary
English; the command macros reject retired keywords instead.

### R1 glossary and vocabulary policy

`.plans/UMPIRE4_SPEC.md` is rewritten to the table above. Every capitalized term either names a
declaration (backticked, resolvable) or is marked planned with its owning spec. Terms with no code
and no owner are removed: Test, Case Artifact, Execution, Stage Status, Assurance Method, Claim
Assessment, Exact Replay. Scenario and Behavior collapse into Scenario. The twenty `Retired:` rule
tombstones move to a closing appendix, keeping their IDs under GOV-01. The module names `Temporal.Verify`,
`Umpire.Verify.Veil`, `TemporalVerify`, `TemporalVeilTests`, and `Temporal.Tool.VerifyVeil` exist
nowhere in the tree; MOD-05 and VER-01 to VER-06 are marked planned and the `ModelLint` classifiers
that reserve them are removed. New rules are drafted under new IDs for GOV-02 approval: one word per
concept, the retired-vocabulary gate as the enforcement point, and the glossary resolution check.

A Go test under `tools/umpire` reads the spec, extracts every backticked `Umpire.*`, `Testpilot.*`,
`Temporal.*`, and `Shared.*` name, and asserts it resolves in the Lean source inventory that
`Tools.LeanSourceInventory` already produces. A planned term is exempt only while its owning spec is
open.

### R2 model core

`Umpire.Target` becomes `Umpire.Model` with four modules named for what they do:

| Module | Was | Contents |
| --- | --- | --- |
| `Umpire/Model/Types.lean` | `Target/Data.lean` | `ModelSpec`, rows, `SourceRef`, `LocatedError` |
| `Umpire/Model/Canonical.lean` | `Target/Projection.lean` | ordering and canonical JSON |
| `Umpire/Model/Check.lean` | `Target/Semantics.lean` | `DraftModel`, `CheckedModel`, `Providers`, `checkModel`, `model` |
| `Umpire/Model/Elab.lean` | `Target/Frontend.lean` | `elabModel` |
| `Umpire/Model/Table.lean` | `Target/FiniteTable.lean` + `FiniteMachine.lean` | `FiniteTable`, `CheckedTable`, `FiniteMachine`, `TableModelSpec` |

`Umpire/Target/Language.lean` is deleted. `DefinitionFamily` moves to `Umpire/Id.lean`.
`Umpire/Target/Parameterized.lean` moves to `Umpire/Operation/Parameterized.lean`, where its
namespace already lives. `Umpire/Core.lean` renames `TransitionKernel` to `Machine`,
`TransitionResult` to `Step` with fields `outcome`, `state`, `facts`, `ModelTrace` to `Trace`,
`ModelCoordinate` to `TraceAddress`, the capability layer per the table, and the `canonicalBehavior`
field on five records to `behaviorVersion`, since it holds a version tag. `TargetDeclaration` merges
into `ModelSpec` with `providers` and `connectors` defaulting to empty. Exactly two checking entry
points remain per owner: `checkModel` returning `Except` and `model` requiring proof;
`composeTarget` becomes private and the four `checkTarget` variants collapse.

Deleted: `TargetBehaviorClosure`, `DefinitionKind.{experimentSpace,variationAxis,choice,fault,coverageGoal}`,
`Shared/Transition.lean`, `Shared/TraceReplay.lean`, and `Umpire/OutcomeClassification/` as a
directory. `DefinitionKind.kernel` becomes `.machine` and `.observation` becomes `.fact`, which
changes every Definition ID of those kinds and therefore every Behavior Fingerprint and golden.
`Umpire.Provenance.DefinitionKind` (was `CaseDefinitionKind`, which carries five extra kinds the
core enum lacks) loses the same five dead constructors, renames `kernel`, `observation`, and
`behavior` to `machine`, `fact`, and `scenario`, and its encoded strings follow
(`CASE_DEFINITION_KIND_FACT`); Testpilot treats those bytes as opaque, so only the Case fixtures
regenerate.

Two golden families have no regenerator today: the fingerprint and canonical-metadata goldens
under `Umpire/Target/Tests/Compatibility/Fixtures` and the six Nexus fixture JSON files, all
consumed through `include_str`. The first R2 task adds one Lake executable, `umpire-goldens`,
that writes both families from their semantic inputs, and `umpire-check-regression` renders them
into a temporary root and diffs, following the regression-views pattern. From then on every
golden in the tree has a regenerator.

### R3 authoring languages and search

Each language exposes one authored record named for its concept, with `check` and `checked` as its
only construction operations:

| Language | Authored | Checked | Module layout |
| --- | --- | --- | --- |
| Property | `Property` (was `PropertyDeclaration` + `PropertySpec`) | `CheckedProperty` | `Umpire/Property.lean` (types, fields, sugar), `Property/Check.lean` (absorbs `Trace.lean`), `Property/Evaluate.lean`, `Property/Elab.lean`, `Property/Correlated/` (was `Scoped/`) |
| Scenario | `Scenario` (was `BehaviorDeclaration`, `BehaviorSpec`, `ExactSequenceSpec`) | `CheckedScenario` | `Umpire/Scenario.lean`, `Scenario/Check.lean`, `Scenario/Elab.lean` |
| Query | `Query` (was `QueryDeclaration`, `QuerySpec`, `QueryAuthoringInput`) | `CheckedQuery` | `Umpire/Query.lean`, `Query/Check.lean`, `Query/Elab.lean` |

The `property%`, `behavior%` (now `scenario%`), and `query%` elaborators stay because fn-80 R2
routes the generalized command syntax through their located-diagnostic path. `bounded_response%`
becomes `correlated_response%`; its `closing .runtimePrefix` and `.deliberatelyClosed` endpoints
become `.partial` and `.final` in every enum that carries them. `PropertyClause` keeps its name;
its constructors lose the `guarded` prefix by taking an optional guard, `sameStepCases` becomes
`branches`, and `quiescentWithin` becomes `neverWithin`. `PropertyPredicateContext.{guard,expectation}`
become `{before,after}`. `PropertyLimit`, `PropertyLimitProfile`, `PropertyScopedClock`, and
`PropertyAuthoring.opaque` are deleted; the `opaqueDeclaration` error survives as a plain error.

`QueryForm` constructors become `verify`, `find`, `findViolation`, `pick`, with matching JSON
spellings; `verify` keeps its name because `check` is the construction method on every authored
record and the two would collide inside the `Query` namespace. `QueryQuantifier`, `QueryClaim`,
and `TieBreakPolicy` are deleted. `QueryLimits` and `BehaviorPhaseLimits` flatten to one `Limits`
record with `steps`, `actions`, `search`. `LimitUnit` becomes `steps`, `actions`, `logicalTime`,
`search`, `plans`; `observationPositions` has three consumers and is deleted rather than renamed,
so `facts` stays a step field and a model keyword only.

`Umpire.Planning` becomes `Umpire.Search`: `Search/Types.lean`, `Search/Engine.lean` becomes
`Search.lean` proper, `Planning/CaseAnalysis.lean` becomes `Search/Branches.lean` with
`analyzeBranches`, `Branch*` and `Overlap*` (was `Joint*`) types. `PlanningOutcome` constructors
become `found`, `verified`, `noneFound`, `limitReached`, `unsatisfiable`, `neverTriggered`,
`stillPending`, `invalid`. `Umpire.ExecutionHandoff` is deleted. `Umpire.Artifact.ExperimentSpec`
becomes `Plan` and `DrivePlan` becomes `Plan.Steps`; the wire format identifiers
`umpire-experiment/v2` and `umpire-drive-plan/v2` are unchanged.

### R4 Testpilot protocol, Lean authoring, and Go facade

Proto renames, all in the internal `testpilot/v1` package and regenerated into Go, Lean, and every
checked-in fixture:

- `Scoped*` becomes `Correlated*` for all 24 messages and enums; `ScopedClause` becomes
  `CorrelatedRule`; `ScopedEndpoint{RUNTIME_PREFIX,DELIBERATELY_CLOSED}` becomes
  `TraceEnding{PARTIAL,FINAL}`; `Contract.scoped` becomes `Contract.correlated`.
- `ContractHorizonDefinition` becomes `ContractDeadline`; the field `elapsed_milliseconds` stays,
  and fn-80's `rule_events` lands on the renamed message.
- `Instruction.await_outcome` becomes `await_instruction` to match its message.
- `outcome.proto` folds into `instruction.proto`; the Makefile proto list and
  `TestProtocolUsesCohesivePublicVocabulary` follow.

Lean: the provenance half of `Umpire/Case.lean` moves to `Umpire/Provenance.lean` as
`Umpire.Provenance.{Metadata,DefinitionBinding,DefinitionKind,KnownGap}` (the old names are ones
the Go protocol test already lists as retired); `Umpire/Case.lean` becomes the facade over
`Umpire.Case.{Compiler,Coverage,Correlated,Projection}`; the `Umpire.Case.{Program,Contract,Run,
Value,ProtoJSON}` alias modules and the `abbrev Case` are deleted. `Testpilot.Authoring.Monitor.*`
becomes `Testpilot.Authoring.Contract.*` and gains the Correlated constructors that
`Umpire/Case/Scoped.lean` currently builds by hand. `Testpilot.Scoped` becomes
`Testpilot.Correlated` with `Run` renamed `Monitor`; `Shared.ScopedProjection` and
`Shared.ScopedObligation` become `Shared.CorrelatedProjection` and `Shared.CorrelatedObligation`,
with `Obligation.Coordinate` renamed `Match` and `Answer` renamed `Verdict`.
`Umpire.Case.Compiler.LoweringError` becomes `Umpire.Case.Compiler.Error`. The
`umpire-gen-lean-api` generator gains a `--skip-package` flag, set to the testpilot package in the
Makefile, removing the unused duplicate mirror from `Temporal.API`. Because `Testpilot.Protocol`
loads the proto files at elaboration time and Lake does not track them as inputs, the task rebuilds
that module from a clean Lake target after the proto rename.

Go: `testpilot.Capability` becomes `Opcode` and `ProfileSpec.Capabilities` becomes `Opcodes`;
`execution.SlotBridge` becomes `CapabilityBridge` to match its public name; `execution.Policy`
becomes `Profile`; the `Context` fields and accessor on `ReservationTopology`,
`ReservationCarrierShape`, and `EntrypointPlan` become `Kind`; `recorder.terminalDisposition` and
`delivery.TriggerDisposition` become `terminalStatus` and `TriggerStatus`. Packages under
`common/testing/testpilot/internal` and the Driver interfaces are otherwise unchanged.

### R5 evidence, projection, artifacts, variations, inventory

`Umpire/Observation/` is split by generation. `Projection.lean`, `Projection/Declaration.lean`,
`Projection/Coverage.lean`, and `Evaluation/Scoped.lean` move to `Umpire/Case/Projection/` and use
`Fact` for the type parameter they currently call `Fact` in one file and `Observation` in the next.
The remaining modules become `Umpire/Evidence/` with `Umpire.Evidence.Reading` (was
`ObservationMappingDeclaration` and its checker), `Umpire.Evidence.Evaluate`, and
`Umpire.Evidence.PropertyStatus` (was `Verdict.lean`, `SemanticVerdictStatus`,
`StrictQuerySummary` becomes `QueryStatusSummary`). `EvidenceLink` and its two mirrors become
`EvidenceSupport`. `EvidenceBundle` becomes `SyntheticEvidence`.

`ImplementationLinkKnownGap` becomes `UnmappedSource`. `ForwardSimulation` becomes
`StepPreservation`, `KernelMorphism` becomes `ValueTranslation`, `notEvaluatedProjectionSentinel`
becomes `stageNotRunMarker` while its rendered string `implementation-link.not-evaluated` is
unchanged. `Umpire.OutcomeClassification.ProjectionSentinelDescriptor` becomes `NotRunMarker`.

`Umpire.Artifact.Runtime` becomes `Umpire.Artifact.RunRecord`; `ArtifactIntent` and
`ArtifactFaultIntent` become `PlanRequest` and `RequestedFault`; the twenty `Artifact*` wire
projections in `Result.lean` drop the prefix inside an `Artifact.Wire` namespace.
`Umpire.Space` becomes `Umpire.Variations` with `VariationSpace`, `PlannedVariant` (was
`LoweredSpacePoint`), and `SpaceMetadataRows`. In `Umpire.Exploration`, `CandidateUniverse` becomes
`CandidateSet`, `ExplorationSession` becomes `CandidateCursor`, `PinnedExperimentSpec` becomes
`PinnedRegression`, `ExplorationOmission` becomes `DroppedCandidate`.
`Umpire.SemanticInventory` becomes `Umpire.Inventory`; the executable becomes `umpire-inventory`,
the Make targets `umpire-gen-inventory` and `umpire-check-inventory`, and the document
`model/INVENTORY.md`. `KnownGapCarryMapping.{exact,observationAdmission}` become `{full,lossy}`.

### R6 Temporal model, command syntax, and tool names

Generation numbers leave the module tree. The existing `Temporal.Feature.Nexus.{Lifecycle,
Operations,Observation,Experimental}` modules stay where they are. Everything under
`Temporal/Feature/Nexus2/` moves to `Temporal/Feature/Nexus/Race/` keeping its file names
(`Lifecycle`, `Race`, `Cancellation`, `Authoring`, tests, `DESIGN.md`, `README.md`), since its
basic lifecycle is a different model from the established one and must not merge with it.
`Nexus3/Cancellation.lean`, the Race table with terminal conditions that the Implementation Link
consumes, becomes `Nexus/Race/Terminal.lean`. The rest of `Temporal/Feature/Nexus3/` moves to
`Temporal/Feature/Nexus/Success/`: `Nexus.lean` becomes `Model.lean`, `Testpilot.lean` becomes
`Producer.lean`, and `Syntax`, `Authoring`, `TypedUnary`, tests, `Nexus.md`, and `Integration.md`
keep their names. Identity roots `temporal.nexus2.*` become `temporal.nexus.race.*` and
`temporal.nexus3.*` become `temporal.nexus.success.*` (with the terminal table under
`temporal.nexus.race.terminal.*`). `Temporal.ImplementationLinkTests.Nexus` merges into
`Temporal.System.Nexus.ImplementationLinkTests` and its dedicated `ModelLint` class and exception
are removed. Per-feature `EVIDENCE.md` files become `COVERAGE.md`.

The generalized command syntax fn-80 R2 delivers is respelled. The five commands stay; the
keywords change where the investigation found them opaque:

```lean
model lifecycle
  role operation
  states State  actions Action  outcomes Outcome  facts Fact
  starts [scheduled]
  ends [succeeded]
  steps
    start: scheduled + awaitStart → { state := started, outcome := acknowledged, facts := [started] }
    success: started + awaitSuccess → { state := succeeded, outcome := completed, facts := [succeeded] }

property successfulResult on lifecycle for operation
  when awaitSuccess
  require successState: state succeeded
  require successOutcome: outcome completed
  require successFact: fact succeeded

scenario successfulCompletion on lifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, completion: awaitSuccess]

limits shortTrace
  steps 2
  actions 2
  search 16

query completion on lifecycle
  find successfulResult in successfulCompletion limits shortTrace

query everyCompletion on lifecycle
  verify successfulResult in successfulCompletion limits shortTrace
```

Changed keywords: `initial` to `starts`, `terminal` to `ends`, `transitions` to `steps`,
`when action X` to `when X`, `resultingState` to `state`, `behavior` to `scenario`,
`transitions N` to `steps N`, `selected_actions` to `actions`, `candidate_evaluations` to `search`,
`witness` to `find`, `all` to `verify`. Kept: `model`, `role`, `states`, `actions`, `outcomes`,
`facts`, `property`, `on`, `for`, `require`, `outcome`, `fact`, `actions exactly`, `limits`,
`query`, `in`, and the Lean-native `:=` and `→`. Each retired keyword keeps a macro arm that
throws a located error naming its replacement, asserted by one `#guard_msgs` block per keyword.
`Nexus.md` and `Integration.md` are respelled in the same task, after fn-67 closes.

Lake executables take one convention: `umpire-inspect`, `umpire-inventory` (with `-tests` and
`-make-tests`), `umpire-case` (was `temporal-testpilot`), `umpire-lint`, `umpire-lint-tests`,
`umpire-protojson-fixture`, `umpire-correlated-fixtures`. Make targets `umpire-list-nexus` and
`umpire-explain-nexus` become `umpire-list` and `umpire-explain`. `lint-model` keeps its name
because MOD-11 and the `lint` aggregate cite it.

### R7 gates, lint policy, documentation, roadmap

Gate hardening is the first task of the spec, before any rename: `addFile` in
`tools/umpire/internal/retiredvocabulary/check.go` fails closed when a listed path is missing
instead of silently dropping it from the scan, the two stale entries in `downstreamSpecs`
(a deleted fn-28 id and a misspelled fn-33 id) are corrected, and the open specs that carry
retired names (fn-46, fn-70, fn-74, fn-78, fn-79) join the list. Every later task adds the names
it retires to the gate and respells every scanned code, doc, fixture, and open spec occurrence in
the same commit, because the gate scans all of them in one pass. The proto breaking check gains
an `ignore` entry for the internal testpilot package in `proto/internal/buf.yaml`, so
`lint-protos` stays green through the R4 rename.
`model/ModelLint/ImportGraph.lean` and its tests are updated for every moved or renamed module,
including `semanticRoots`, `isTargetForbiddenDestination` (now the Model prefix), the
`semanticInventoryIsolation` rule (now `inventoryIsolation`), the Implementation Link exception,
and the removal of the Verify reservations. The Makefile package-layout check that pins
`Umpire/{Target,Property,Behavior,Query}/Language.lean` and `Planning/Engine.lean` is rewritten
for the new layout. `model/README.md`, `model/ARCHITECTURE.md`, and `model/Umpire/ARCHITECTURE.md`
are rewritten under the new vocabulary, and `.plans/UMPIRE4_ORDER.md` gains an entry for this spec.

## API Contracts
<!-- scope: technical -->

Wire changes in `proto/internal/temporal/server/api/testpilot/v1`, all renames, no field-number
changes:

```proto
// contract.proto
message Contract { string contract_id = 1; repeated ContractRuleDefinition rules = 2; ContractLimits limits = 3; CorrelatedContract correlated = 4; }
message ContractDeadline { int64 elapsed_milliseconds = 1; string violation_state_id = 2; }
message CorrelatedContract { /* every field of ScopedContract, unchanged numbers */ }
message CorrelatedRule { /* every field of ScopedClause, unchanged numbers */ }
enum TraceEnding { TRACE_ENDING_UNSPECIFIED = 0; TRACE_ENDING_PARTIAL = 1; TRACE_ENDING_FINAL = 2; }
// run.proto
message CorrelatedEvidence { /* was ScopedEvidence */ }
// instruction.proto (absorbs outcome.proto)
message Instruction { oneof instruction { ...; AwaitInstruction await_instruction = 3; ... } }
```

Go public facade after the change:

```go
// common/testing/testpilot
type Opcode uint8
const ( InvokeRPC Opcode = iota + 1; AwaitSlot; CompleteNexusOperation; StartNexusOperation; Await; Finish; RespondNexus )
type ProfileSpec struct { Identity string; Catalog *Catalog; Roles []RolePolicy; Opcodes []Opcode; EnvironmentBindings []EnvironmentBinding; ProgramLimits, ContractLimits ... }
func (p EntrypointPlan) Kind() testpilotspb.EntrypointKind
func (p InstructionPlan) Opcode() Opcode
```

Lean public facades after the change:

```lean
-- Umpire.Model
def checkModel : DraftModel → Except LocatedError CheckedModel
def model (draft : DraftModel) (ok : (checkModel draft).isOk := by native_decide) : CheckedModel
structure Machine (Setup State Action Outcome Fact : Type)
structure Step (State Outcome Fact : Type) where outcome : Outcome; state : State; facts : List Fact

-- Umpire.Property / Scenario / Query
def Property.check : Property → CheckContext → Except PropertyError CheckedProperty
def Scenario.check : Scenario → CheckContext → Except ScenarioError CheckedScenario
def Query.check : Query → CheckedModel → Except QueryError CheckedQuery
inductive Query.Form | verify (p) | find (p) | findViolation (p) | pick (ps)

-- Umpire.Search
def search : CheckedQuery → SearchView target → PlanResult

-- Testpilot.Authoring
namespace Contract  -- was Monitor
def rule / state / transition / deadline / limits / correlated / correlatedRule

-- Umpire.Case.Compiler  -- unchanged entry point, renamed error
def compile : Input → Except Compiler.Error Case
-- Umpire.Provenance  -- was the provenance half of Umpire.Case
def producerData : Metadata → ByteArray
```

Command syntax contract: the five commands `model`, `property`, `scenario`, `limits`, `query`
with the keyword set in R6.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Fingerprints and goldens regenerate.** `DefinitionKind.kernel` to `machine`, `.observation`
  to `.fact`, `canonicalBehavior` to `behaviorVersion`, `modelOutcome` to `outcome`,
  `LimitUnit` names, `QueryForm` spellings, and the Nexus identity roots all change canonical JSON,
  every Behavior Fingerprint, and the goldens under `model/Umpire/Examples`,
  `model/Umpire/Artifact/Tests/Fixtures`, `model/Temporal/Feature/Nexus/Fixtures`,
  `model/Umpire/Target/Tests/Compatibility/Fixtures`, `tests/testcore/testpilot/testdata`, and
  `common/testing/testpilot/testdata/case-runtime-conformance`. Each task regenerates through the
  owning make target (or the new `umpire-goldens` writer) and never edits a golden by hand. The
  embedded checksum in `switch_generated_view_test.go` regenerates with the views; the
  `switchIdentity` constant names a Query ID whose kind does not change and stays.
- **Wire format identifiers stay.** `umpire-experiment/v2`, `umpire-drive-plan/v2`, the
  fingerprint domain tags in `Umpire/Fingerprint.lean`, and `implementation-link.not-evaluated`
  are not human-facing vocabulary; renaming them buys nothing and rewrites every checksum twice.
- **`#guard_msgs` strings change.** Located diagnostics render role names and JSON keys; every
  expected-message block in `Umpire/*/Tests` and `Temporal/Feature/Nexus*/AuthoringTests.lean` is
  re-baselined by running the module, not by editing text.
- **The retired gate scans open specs.** `downstreamSpecs` in `retiredvocabulary/check.go` lists
  fifteen open specs; adding `CheckedTarget` and its peers to the retired list makes those documents
  fail until respelled. The sweep respells them in the same commit. The gate's `addFile` currently
  ignores missing files, so a moved file silently leaves the scan; R7 makes that an error first.
- **Lint pins names.** `ModelLint/ImportGraph.lean` names seventeen `semanticRoots`, the
  `Umpire.Target` prefix, `Umpire.OutcomeClassification`, `Umpire.SemanticInventory`,
  `Temporal.ImplementationLinkTests.Nexus`, and the Verify modules; the Makefile `lint-model`
  target asserts one exact diagnostic string. Every module move edits both in the same task.
- **Proto breaking lint.** `lint-protos` runs `develop/buf-breaking.sh`, which has no
  per-package switch; the exclusion lives in `proto/internal/buf.yaml` as a breaking `ignore`
  entry for the testpilot package, added by the gate task so the check stays green.
- **Byte conflicts with open specs.** fn-80 tasks .1 to .9 and fn-77 tasks .9 to .11 edit the
  protos, the Go facade, `Umpire/Target`, `Umpire/Property`, `Umpire/Query`, `Umpire/Space`,
  `Umpire/Case`, the Nexus3 tree, the three model documents, the Makefile, and the retired gate;
  fn-81 edits the Makefile, the gate, and the roadmap; fn-67 edits the Nexus3 documents. The spec
  depends on all four and no task starts before they close. `Umpire.Operation` and `Umpire.Value`
  renames stay limited to `ValueShape` to `Shape` and the `Parameterized.lean` move.
- **Every task leaves every gate green.** The gate scans code, docs, fixtures, and open specs in
  one pass, so a rename split across commits leaves the tree red in between. Each task is one
  atomic unit: the rename, the gate entry, the lint policy edit, the Makefile layout check, the
  regenerated goldens, and the respelled documents.
- **`Umpire.Value` already exists.** `ModelValue` keeps its name; the investigation's proposal to
  shorten it collides with the fn-77 typed value module.
- **`Umpire.Operation` already exists.** The per-operation contract prefix is `Correlated`, not
  `Operation`, to avoid colliding with the typed-operation module.
- **Renamed tokens must not be previously retired tokens.** `Declaration*`, `Bound*`,
  `Semantic*`, `Projection{Record,Manifest}`, `Refinement`, and the bare JSON keys `"bounds"`,
  `"omissions"`, `"qualification"`, `"qualified"` are already rejected; no new name reuses them.

## Quick commands

```bash
cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot
make lint-model
make umpire-check-regression
make umpire-check-case-runtime-conformance
make umpire-check-retired-vocabulary
CGO_ENABLED=0 go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...
go test -tags 'test_dep integration' ./tests -run 'TestTestpilot|TestUmpire'
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `.plans/UMPIRE4_SPEC.md` defines each term in the vocabulary table once, every
  backticked module or declaration name in it resolves against the Lean source inventory through a
  Go test under `tools/umpire`, doc-only terms are removed or marked planned with an open owning
  spec, the retired rules sit in an appendix with their IDs intact, and the new vocabulary rules
  carry new IDs awaiting GOV-02 approval. Errors: a backticked name that does not resolve fails the
  test; a planned term whose owning spec is closed fails the test.
- **R2:** `Umpire.Model` replaces `Umpire.Target` with the four-module layout, `Machine`, `Step`,
  `Fact`, `Trace`, `Vocabulary`, `DraftModel`, `CheckedModel`, and the capability names from the
  table; `checkModel` and `model` are the only construction entry points; the Switch example and
  every Nexus model build; the dead modules, `TargetBehaviorClosure`, and the five unused
  `DefinitionKind` constructors are gone from both the core and the Provenance enum; the
  `umpire-goldens` writer regenerates the compatibility and Nexus fixture goldens and the
  regression check diffs them; regenerated fingerprints and goldens pass the regression check.
  Errors: a Definition ID of kind `kernel` or `observation` is rejected as unknown kind; a stale
  golden fails `umpire-check-regression`; the retired gate rejects `CheckedTarget`,
  `TransitionKernel`, `AuthoredTarget`, `QueryTarget`, `TargetBehaviorDomain`, and `modelOutcome`.
- **R3:** `Property`, `Scenario`, and `Query` are the only authored records with `check` and
  `checked`; `Query.Form` has the `verify`, `find`, `findViolation`, and `pick` constructors with matching JSON spellings; `Limits` is one flat
  record; `LimitUnit` has the five names; `Umpire.Search` replaces `Umpire.Planning` with
  `PlanResult`, `SearchView`, `Branch*`, and `Overlap*`; `Plan` and `Plan.Steps` replace
  `ExperimentSpec` and `DrivePlan` while the wire identifiers are byte-identical;
  `ExecutionHandoff`, `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy`, `QueryAuthoringInput`,
  and `PropertyScopedClock` no longer exist. Errors: `bounded_response%`, `behavior%`, `witness`,
  `.runtimePrefix`, and `.deliberatelyClosed` are rejected by the retired gate; a Property with an
  ambiguous branch group still reports the same `Overlap*` finding under its new name.
- **R4:** the proto package has no `Scoped*` name, `ContractDeadline` replaces the horizon message,
  `await_instruction` matches its message, `outcome.proto` is gone, `Testpilot.Protocol`
  re-elaborates from a clean Lake target, `Temporal.API` no longer mirrors the testpilot package
  through the new `--skip-package` flag, the `Umpire.Case` alias family and `Umpire.Case.ProtoJSON`
  are deleted, `Umpire.Provenance` holds the producer-owned types, `Testpilot.Authoring.Contract`
  builds both rule kinds, Go exposes `Opcode` and `Opcodes`, `lint-protos` passes with the buf
  ignore entry, and the sixteen regenerated Case fixtures (three example files and thirteen
  conformance files) pass the conformance and live gates with unchanged Verdicts. Errors:
  `TestProtocolUsesCohesivePublicVocabulary` rejects every retired proto name including the
  twenty-four `Scoped*` names; a Case carrying a `scoped` field fails to decode; the retired gate
  rejects `Capability` as a Go opcode type.
- **R5:** `Umpire.Evidence` holds the offline evaluator, `Umpire.Case.Projection` holds the live
  seam with `Fact` as its type parameter, `UnmappedSource` replaces the mislabeled Known Gap,
  `Umpire.Variations` replaces `Umpire.Space`, `Umpire.Inventory` and `model/INVENTORY.md`
  replace the semantic inventory with unchanged catalog IDs, and `Umpire.Artifact.RunRecord`
  replaces `Runtime`. Errors: any production Umpire module importing `Umpire.Inventory` fails
  `lint-model`; `umpire-check-inventory` fails when the document is stale; the retired gate rejects
  `Umpire.Observation`, `SemanticVerdictStatus`, `ImplementationLinkKnownGap`, `ExperimentSpace`,
  and `SEMANTIC_INVENTORY`.
- **R6:** no module or directory named `Nexus2` or `Nexus3` remains, the Race and Success trees
  live under `Temporal.Feature.Nexus` beside the established modules with identity roots
  `temporal.nexus.race.*` and `temporal.nexus.success.*`, the Implementation Link imports
  `Nexus.Race.Terminal`, the five commands accept the R6 keyword set and reject each old spelling
  with a located error asserted by one `#guard_msgs` block per keyword, `Nexus.md` and
  `Integration.md` use the new keywords, the Lake executables and Make targets use the `umpire-`
  convention, and the live Nexus Case runs with the same satisfied Contract. Errors: `initial`,
  `terminal`, `transitions`, `witness`, `all`, `behavior`, `selected_actions`, and
  `candidate_evaluations` are macro errors naming the replacement keyword; `temporal.nexus3`,
  `temporal.nexus2`, `Temporal.Feature.Nexus2`, and `Temporal.Feature.Nexus3` in any live source
  fail the retired gate.
- **R7:** the gate fails closed on a missing scanned file and its `downstreamSpecs` list names
  only existing open specs before any rename lands; `make lint-model`,
  `make umpire-check-regression`, `make lint-protos`, the Go packages under the Testpilot facade
  and `tools/umpire`, and the tagged live selector pass after every task; the retired gate contains
  every compound name this spec retires; `ModelLint` has no Verify reservation and no
  `ImplementationLinkTests` exception; the three model documents and `UMPIRE4_ORDER.md` describe
  the tree under the new vocabulary. Errors: a listed scan path that does not exist fails
  `umpire-check-retired-vocabulary` with the path named; a doc, fixture, or open spec that
  mentions a retired token fails it; a retired rule for a bare English word is rejected by the
  gate's own test.

## Early proof point

The second task, the first rename, changes `DefinitionKind.kernel` to `machine` and
`.observation` to `.fact`, renames `TransitionKernel` and `TransitionResult`, adds the
`umpire-goldens` writer, regenerates every fingerprint, golden, and Case fixture through the
owning targets, and adds the old names to the hardened gate. If the regeneration loop cannot be
made to converge without hand-editing a golden, or if the gate cannot be extended without breaking
an open spec that no task may respell, stop and re-evaluate the "regenerate, never edit" rule
before any other rename starts.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Glossary rewrite, vocabulary rules, resolution test | fn-82-unify-the-umpire-and-testpilot.9 | — |
| R2 | Model core rename, module layout, dead code | fn-82-unify-the-umpire-and-testpilot.2, .3 | — |
| R3 | Property, Scenario, Query, Search, Plan | fn-82-unify-the-umpire-and-testpilot.4, .5 | — |
| R4 | Testpilot proto, Lean authoring, Go facade | fn-82-unify-the-umpire-and-testpilot.7 | — |
| R5 | Evidence, Projection, Artifact, Variations, Inventory | fn-82-unify-the-umpire-and-testpilot.6 | — |
| R6 | Nexus tree, command keywords, tool names | fn-82-unify-the-umpire-and-testpilot.8 | — |
| R7 | Gates, lint policy, docs, roadmap | fn-82-unify-the-umpire-and-testpilot.1, .10 | — |

## Boundaries
<!-- scope: business -->

- No deletion of the offline `Umpire.Evidence` evaluator, `Umpire.Artifact.RunRecord`,
  `Umpire.Variations`, `Umpire.Exploration`, or `Umpire.Promotion`. They are renamed and moved;
  fn-22, fn-33, fn-79, and fn-80 reserve the decision to retire them.
- No change to fn-77's in-flight terms: typed operation, parameterized Action, field-level
  Property, occurrence, capture. `Umpire.Operation` and `Umpire.Value` renames wait for fn-77 .11
  and are limited to `ValueShape` to `Shape` and moving `Parameterized.lean`.
- No new command syntax beyond respelling the five commands fn-80 R2 generalizes. No `oneOf`,
  `eventually`, `integration`, or `verify` forms from the Nexus3 design drafts.
- No proto field-number changes, no new messages, no removal of `ActivityActivation`,
  `RunRef`, `AnyType`, or `ROLE_KIND_PARTICIPANT`. fn-80 states the activity entrypoint is not
  touched; the others are renamed only if a rename applies.
- No rename of `role`, `Feature`, `System`, `DynamicConfig`, `ImplementationLink`, `Promotion`,
  `KnownGap`, `lint-model`, or the wire format identifiers.
- No edits to historical `.plans` documents other than `UMPIRE4_SPEC.md` and `UMPIRE4_ORDER.md`.
- No Go changes outside `common/testing/testpilot`, `tests/testcore/testpilot`, `tests` fixture
  helpers, and `tools/umpire`. fn-81 owns the legacy trees and their Makefile blocks.
- No new CI workflow and no generated-API drift gate.

## Decision Context
<!-- scope: both -->

One word per concept was chosen over per-layer synonyms because the reader the vision names is a
Temporal engineer with basic Lean, not an Umpire maintainer, and every synonym is a term they must
learn twice. Where two concepts really differ, the spec keeps two words and says how they differ:
Property clauses are model-level, Contract rules are run-level; Fact is a model claim, Observation
is a declared Run value; Limit is any ceiling, Deadline is the time-shaped one.

`Correlated` was chosen for the `Scoped` prefix over `PerOperation` because `Umpire.Operation`
already names the typed-operation module and `OperationRule` would read as a rule about typed
operations. `Correlated` names the mechanism the data actually carries: `ScopedCorrelation`,
`scope_fields`, and `operation_field` are all correlation keys.

`Scenario` replaces `Behavior` because the spec already defines Scenario as the concept and
Behavior as its Lean data, and because Behavior also names the Behavior Model and the Behavior
Fingerprint. The fingerprint keeps its name; it fingerprints behavior, not a scenario.

`Model` replaces `Target` because authors already write `model lifecycle` in the command syntax,
the spec's definition of Target is "a validated Behavior Model", and six types currently carry the
word Target without any of them being the model. `Machine` replaces `TransitionKernel` because
"kernel" collides with the Lean kernel in the same sentences that discuss proof trust.

`Fact` wins over `Observation` for model claims because the command syntax already says `facts`,
the spec already says Model Fact, and Observation is the one Testpilot word the Contract reads.
Keeping Observation for the runtime value and Fact for the model claim removes the worst overload
in the tree without touching the wire.

Wire format identifiers and fingerprint domain tags stay because they are never read by a human
and renaming them would rewrite every checksum a second time for no vocabulary gain. Proto message
names change because they surface in Go, Lean, and every fixture a reviewer reads.

`role` stays in both the model and the Case despite the mild overlap: "the operation role" and
"the workflow-service role" both read naturally, and the alternative (`subject` or `resource`)
touches ninety files for a collision nobody reported. `require` stays over `expect` in the command
syntax for the same reason: it reads acceptably and fn-80 R2 already generalizes it.

Deleting dead vocabulary is in scope because a name with no consumer still costs every reader who
meets it. Retiring whole generations is not, since open specs reserve those decisions; the spec
renames and relocates them so their generation is visible in the path.

The retired-vocabulary gate is the enforcement point rather than a new lint because the project
already retired one vocabulary through it (`Declaration*`, `Bound*`, `Semantic*`, the
regression-projections targets) and the mechanism is proven. Its silent skip of missing files is
fixed first because a rename sweep is exactly the change that would trigger it.

`Umpire.Provenance` was chosen over `Umpire.Producer` for the producer-owned bytes because the
spec keeps Producer as the name of the role, and a module called Producer that only encodes
provenance would put one word on two ideas again. The Nexus3 Producer module is named
`Producer` because it is one.

Two open specs with every task done, fn-75 and fn-76, state contracts this spec overrides
(`composeTarget` and `checkTarget` retained; `Umpire.SemanticInventory.Types`). Neither is in the
gate's scan list, so nothing fails, but both should be closed by their owner rather than amended.
fn-61 and fn-63 are historical records and untouched.

The declined-concept ledger entry on generated API drift verification applies: this spec adds a
`--skip-package` flag and a goldens writer to existing generators and regenerates existing
fixtures, and adds no drift gate and no CI workflow.

