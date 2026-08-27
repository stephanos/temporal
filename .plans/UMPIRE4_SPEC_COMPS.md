# Umpire4 deep-module architecture

Status: approved architecture specification for the present and intended Umpire4 system.

This document defines the module structure, interfaces, dependency direction, artifact seams, and
development priorities for Umpire4. It refines the component inventory in
[`UMPIRE4_COMPONENTS.md`](UMPIRE4_COMPONENTS.md), the semantic design in
[`UMPIRE4_DSL.md`](UMPIRE4_DSL.md), the Lean ownership rules in
[`UMPIRE4_SPEC_MODEL_ARCH.md`](UMPIRE4_SPEC_MODEL_ARCH.md), the command contract in
[`UMPIRE4_SPEC_CLI.md`](UMPIRE4_SPEC_CLI.md), and the goals in
[`UMPIRE4_VISION.md`](UMPIRE4_VISION.md).

Implementation status in this document is descriptive, not authorization to perform a migration.
Each change still needs a bounded design and implementation plan. This specification deliberately
does not mirror task-tracker identifiers or ordering; priorities derive from module dependencies
and architectural value.

## 1. Purpose

Umpire4 should provide one Lean-owned model of software behavior that can:

- define and verify known regressions;
- compile deterministic, complete executable test intent;
- execute the same test locally, in CI, against authorized remote environments, and through a
  separately owned production canary;
- operate with white-box or black-box evidence;
- express non-linear scenarios, late-bound runtime identities, concurrency, causal ordering, and
  clock skew;
- treat faults and participant behavior as explicit authored intent;
- explore bounded semantic spaces to find unknown interactions;
- qualify evidence without overstating what an environment established; and
- replay, minimize, and propose stable regressions from evaluated violations.

The architecture must remain approachable to Temporal engineers. Sophisticated checking,
canonicalization, planning, artifact, evidence, and verification behavior belongs behind small
interfaces rather than being repeated in each model family.

## 2. Design vocabulary

This document uses the following terms consistently.

**Module** — a capability with one interface and an implementation. A module may be a Lean module,
a Go package, an executable package, or a tier-spanning slice connected by an artifact seam.

**Interface** — everything a caller must know to use a module correctly: accepted inputs, returned
outputs, invariants, ordering, Limits, errors, performance, and configuration.

**Depth** — the leverage a module provides through its interface. A deep module hides substantial
behavior behind a small interface. Removing it would force complexity to reappear across multiple
callers.

**Seam** — a location where behavior can vary without editing the caller. Checked Lean values are
in-process seams. Versioned artifacts are language and process seams.

**Adapter** — a concrete implementation at a seam, such as a Temporal SDK participant, an
ephemeral server authority, a Lean subprocess checker, or a production canary driver.

**Artifact** — immutable, versioned data exchanged across a seam. An artifact is not a module and
does not acquire semantic authority merely because it is portable.

Commands are adapters. A command package may contain a deep module, but argument parsing and
`main` are not themselves semantic modules.

## 3. Governing principles

1. **Lean is the sole behavioral authority.** Go, JSON, generated tests, runtime adapters, evidence
   collectors, and checker adapters do not independently define Temporal meaning.
2. **Deep capability modules connect through explicit seams.** Callers use checked values in Lean
   and versioned artifacts across language or process boundaries.
3. **Authored and checked values remain distinct.** Invalid declarations cannot masquerade as
   canonical Model Values.
4. **Product and implementation meanings remain distinct.** `Temporal.Feature` owns what Temporal
   means; `Temporal.System` owns how an implementation realizes or reveals it; Implementation Link relates
   them explicitly.
5. **Execution and judgment remain distinct.** A realized action is not evidence that a property
   holds. Raw evidence must be accepted before pure properties evaluate it.
6. **Operational authority belongs at adapters.** Endpoints, credentials, leases, rate limits,
   isolation, cleanup, and blast-radius controls are not semantic model concerns.
7. **Limits, Known Gaps, trust, and provenance are explicit.** Exhaustion, missing evidence,
   unsupported vocabulary, checker timeout, or cleanup failure never becomes success.
8. **Determinism is part of the interface.** Equivalent semantic inputs produce byte-identical
   canonical products regardless of incidental declaration or map order.
9. **Optional verification stays optional.** Veil is isolated from ordinary Umpire and Temporal
   imports, runtime paths, and production binaries.
10. **Production canary is standalone.** `tools/canary` consumes stable Umpire interfaces but owns
    production policy and is never imported by Umpire.

## 4. End-to-end module map

```text
Generated structure
  Descriptor Export ──▶ API Catalog
  Dynamic Config ─────▶ Config Catalog
                              │
                              ▼
Semantic authority
  Core / Target Composition
  Property / Behavior / Space / Query
  Temporal Feature / System / Implementation Link / Observation
                              │
             ┌────────────────┼────────────────┐
             ▼                ▼                ▼
          Planning        Exploration     Formal Verification
             │                │                │
             └────────── ExperimentSpec ───────┘
                              │
                ┌─────────────┴─────────────┐
                ▼                           ▼
         Go/docs Generated View          Execution + Participants
                                            │
                                   Run + Raw Evidence
                                            │
                                            ▼
                                       Run Evaluation
                                            │
                                          Result
                                            │
                              ┌─────────────┴─────────────┐
                              ▼                           ▼
                      Replay / Promotion             Claim Assessment
                                                          │
                                                          ▼
                                                standalone tools/canary
```

Artifact admission, migration, complete-set validation, and atomic publication surround the
persisted seams without becoming a second semantic evaluator.

## 5. Module contract

Every main module should document:

- its one responsibility;
- its public interface;
- the checked values or artifacts it consumes and produces;
- allowed and forbidden dependencies;
- where semantic authority resides;
- adapters at its seams;
- deterministic and bounded behavior;
- typed failures and phase outcomes;
- interface-level tests; and
- current implementation status and priority.

Public interfaces and tests should align. Tests that must reach past an interface indicate that the
module or seam is probably the wrong shape.

## 6. Lean library topology

The Lean model is a hierarchy of modules rather than one undifferentiated semantic component.

```text
Shared
├── Transition
└── TraceReplay

Umpire
├── Core
├── Target
├── Property
├── Behavior
├── Space
├── Observation
├── ImplementationLink
├── Query
├── Planning
├── Exploration
├── Artifact
├── Catalog
├── Promotion
├── Claim Assessment
└── Verify
    ├── Native
    └── Veil

Temporal
├── API
├── DynamicConfig
├── Feature
├── System
├── Tool
└── Verify
```

### 6.1 Shared modules

| Module | Interface | Status and direction |
| --- | --- | --- |
| `Shared.Transition` | Neutral transition systems, reachability, finite runs, observations, and trace steps. | Present. Keep independent of Umpire and Temporal. |
| `Shared.TraceReplay` | Replay named actions through a transition function. | Present but shallow. Deepen into canonical replay as verification and promotion need it, or fold it into `Shared.Transition` if it remains trivial. |

`Shared` is neutral infrastructure. It must not become an alternate authoring language or semantic
catalog.

### 6.2 Umpire foundation

| Module | Interface | Status and direction |
| --- | --- | --- |
| `Umpire.Core` | Stable identities, kinds, metadata, sources, typed Limits, Model Values, and trace vocabulary. | Present but over-broad. Reduce it to stable shared vocabulary. |
| `Umpire.Target` | Finite kernels, capabilities, laws, providers, connectors, target checking, and canonical target identity. | Extract from `Core` and deepen before adding more authoring languages. |

The target interface is:

```text
TargetDeclaration
      │
      ▼
composeTarget
      │
      ├──▶ CheckedTarget
      └──▶ TargetError
```

Ordinary Temporal authors should not assemble metadata digests, provider lists, connector plumbing,
checked-result extraction proofs, or planner backend structures. Model maintainers may still use a
lower-level typed interface when they define a new authoritative kernel or law.

### 6.3 Semantic authoring modules

| Module | Responsibility | Status and direction |
| --- | --- | --- |
| `Umpire.Property` | Portable claims and pure evaluation over capability-limited Model Traces. | Present and deep. Retain its independent facade. |
| `Umpire.Behavior` | Setup, action, occurrence, ordering, and exact-trace constraints without assigning outcomes. | Present and deep. Retain its independent facade. |
| `Umpire.Space` | Finite variation axes, named choices, requested fault intents, and semantic coverage goals. | Planned. Lower points through Behavior, Query, and the target-owned kernel. |
| `Umpire.Observation` | Checked mappings from raw evidence to accepted Model Facts, Model Traces, and Evidence Links. | Planned and required for live Run Evaluation. |
| `Umpire.ImplementationLink` | Checked correspondence between independently authored Feature and System meanings. | Planned and required for honest implementation Run Evaluation. |

Each module follows the same authored-to-checked lifecycle:

```text
Declaration + checking context
             │
             ▼
          check
             │
       ┌─────┴─────┐
       ▼           ▼
 Checked value   Typed error
```

Property consumes semantic observations, never logs, RPC names, spans, storage rows, execution
receipts, or environment profiles. Behavior requests actions and faults but cannot claim target
outcomes or runtime realization. Observation establishes semantic facts but cannot redefine a
property. Implementation Link relates meanings but cannot silently select a provider or rewrite either side.

### 6.4 Composition, planning, and exploration

| Module | Responsibility | Status and direction |
| --- | --- | --- |
| `Umpire.Query` | Combine a checked target, properties, behavior, quantifier, Limits, completeness evidence, and policy. | Present and deep. Remains the first semantic composition point. |
| `Umpire.Planning` | Deterministic bounded selection or verification over a checked query and finite kernel. | Present and deep. Keep planning outcomes explicit. |
| `Umpire.Exploration` | Batch selection, semantic coverage, symmetry, resume state, and coverage-guided prioritization. | Planned. Own campaign selection rather than expanding Planning indefinitely. |
| `Umpire.Artifact` | Construct canonical `DrivePlan` and `ExperimentSpec` values from checked selections. | Present but partial. Deepen by controlling construction, anti-forgery, versioning, and canonical serialization. |

`Umpire.Search` is not a peer deep module in its current form. It mostly contains policy, Limits,
and metadata structures. Query-specific vocabulary should sit behind Query or Planning. Substantial
campaign search belongs in Exploration. Retire the standalone facade while implementing those
owners rather than through an isolated compatibility layer.

Planning selects a trace. Artifact compiles the selected trace into portable intent. Exploration
selects batches and updates semantic coverage state. These responsibilities must remain separate
even when one public operation composes them.

### 6.5 Discovery, learning, and assurance

| Module | Responsibility | Status and direction |
| --- | --- | --- |
| `Umpire.Catalog` | Checked inventory of Targets, Properties, Behaviors, Spaces, Queries, observations, and Behavior Fingerprints with deterministic list/explain Generated Views. | Planned. It discovers meaning but does not create it. |
| `Umpire.Promotion` | Convert an exact checked witness or minimized accepted failure into a reviewable regression proposal. | Planned. It never installs source automatically. |
| `Umpire.Evaluation` | Generic profile and receipt vocabulary plus bounded claim evaluation over admitted results. | Planned. Temporal owns concrete environment profiles and authority. |
| `Umpire.Verify.Native` | Lean-native bounded receipts and canonical counterexample replay. | Planned and unconditional. |
| `Umpire.Verify.Veil` | Generic optional Veil invocation, binding support, trust classes, and receipt vocabulary. | Conditional and excluded from `import Umpire`. |

Catalog and Promotion remain separate: Catalog explains existing checked declarations; Promotion
proposes a new exact regression from existing checked semantic evidence. Claim Assessment does not
execute an environment and cannot acquire authority.

### 6.6 Lean dependency direction

```text
Core ──▶ Target
  ├────▶ Property
  ├────▶ Behavior
  ├────▶ Space
  ├────▶ Observation
  └────▶ Implementation Link

Target + Property + Behavior ──▶ Query
Space + Query ─────────────────▶ Planning / Exploration
Query + selected trace ────────▶ Artifact
Observation + Property ────────▶ semantic verdict
Target + Property ─────────────▶ Verify
checked witness/result ────────▶ Promotion
all checked declarations ─────▶ Catalog
admitted results + profile ────▶ Claim Assessment
```

Checked values are the in-process interfaces. Canonical JSON is introduced only when leaving Lean
or crossing a process boundary.

### 6.7 Recommended reusable source shape

```text
model/Umpire/
├── Core.lean
├── Target.lean
├── Target/
│   └── Language.lean
├── Property.lean
├── Property/
│   └── Language.lean
├── Behavior.lean
├── Space.lean
├── Observation.lean
├── Implementation Link.lean
├── Query.lean
├── Planning.lean
├── Exploration.lean
├── Artifact.lean
├── Catalog.lean
├── Promotion.lean
├── Claim Assessment.lean
└── Verify/
    ├── Native.lean
    └── Veil.lean
```

Facade files expose the small interface. Implementation directories hide checking,
canonicalization, algorithms, and private seams. Tests mirror behavioral concerns and import the
public facade used by callers.

## 7. Temporal Lean modules

Temporal modules are classified by semantic ownership, not author expertise.

```text
Temporal
├── generated structure
│   ├── API
│   └── DynamicConfig
├── Feature
│   └── product-visible meaning
├── System
│   └── implementation meaning
├── Tool
│   └── developer adapters
└── Verify
    └── optional expert adapters
```

### 7.1 Generated structures

| Module | Responsibility | Forbidden responsibility |
| --- | --- | --- |
| `Temporal.API.Proto` | Wire primitives such as bytes, message references, and typed method shapes. | Product behavior or RPC execution. |
| `Temporal.API.Types` | Generated messages, enums, fields, presence, recursion, and references. | Semantic interpretation. |
| `Temporal.API.Catalog` | Complete mechanical API metadata, dispositions, identity, and bounded current-model selection. | Meaning inferred from names or descriptor presence. |
| `Temporal.DynamicConfig.Types` | Generated schemas, values, constraints, defaults, precedence policies, and fixtures. | Configuration effects. |
| `Temporal.DynamicConfig.Settings` | Complete initialized registry and catalog identity. | Classification or interpretation. |

`Temporal.API.Catalog` is a logical module. Its generated declarations may remain within the
existing deterministic output set rather than forcing another file solely for symmetry.

### 7.2 Feature family template

```text
Temporal.Feature.<Family>
├── Model
├── Target
├── Properties
├── Scenarios
└── Examples
```

| Module | Responsibility |
| --- | --- |
| `Model` | Canonical product states, actions, outcomes, observations, relations, and transitions. |
| `Target` | Adapt the family model into a checked `Umpire.Target`. |
| `Properties` | Portable product-visible claims. |
| `Scenarios` | Behaviors, spaces, queries, regressions, named Limits, and policies. |
| `Examples` | Teaching paths and small executable model examples. |

This is a logical template, not a requirement to create shallow files. A small family may colocate
the responsibilities. A family should split when one interface hides substantial behavior or when
independent consumers need a narrower import.

### 7.3 Nexus decomposition

`Temporal.Feature.Nexus.Experimental.AutoClose` contains several deep logical modules:

```text
Nexus.Experimental.AutoClose
├── Lifecycle       operation state, events, and authoritative step
├── Model           configuration, resolution, reachability, and auto-close behavior
├── Properties      honored delivery and cancellation uniqueness
└── History         emitted events, rebuild, and faithfulness
```

`Temporal.Feature.Nexus.Experimental.CallerClosure` currently combines the ownership model, target
composition, properties, behaviors, queries, planner kernel, runs, and artifact selection. Its
target shape is:

```text
CallerClosure
├── Model            ownership and caller-closure semantics
├── Target           capabilities, providers, connector, and kernel
├── Properties       honored delivery and uniqueness
└── Scenarios        behaviors, spaces, queries, Limits, and selected artifacts
```

Deepen `Umpire.Target` before physically splitting CallerClosure. The goal is to remove repeated
plumbing rather than distribute it across more files.

The root `Temporal.Feature.Nexus.Lifecycle` and `Temporal.Feature.Nexus.Operations` modules are the
ordinary Feature surface. They should demonstrate the start, cancellation, and successful-
completion authoring path without exposing the experimental AutoClose configuration or
caller-closure composition.

### 7.4 System family template

```text
Temporal.System.<Family>
├── Model
├── Configuration
├── Observation
├── Program
└── ImplementationLink
```

| Module | Responsibility |
| --- | --- |
| `Model` | Concrete tasks, handlers, state machines, routing, persistence, and mechanisms. |
| `Configuration` | Typed uses and interpretations of generated configuration settings. |
| `Observation` | Evidence mappings that establish System semantic facts. |
| `Program` | Model-owned participant or runtime intent required to realize selected actions. |
| `ImplementationLink` | Explicit correspondence between System and Feature meaning. |

Feature never imports System. Base System modules do not import Feature. Only the family Implementation Link
leaf may import both.

```text
Feature.Model ───────────────┐
                             ▼
                      System.ImplementationLink
                             ▲
System.Model/Configuration ──┘
```

`Temporal.System.Configuration.Core` is already deep: it hides classification checking, typed
uses, overrides, precedence resolution, immutable views, provenance, opaque defaults, and fixture
verification behind one interface.

`Temporal.System.Callback.Configuration` currently combines configuration interpretation with
callback routing, admission, dispatch, and trace projection. Separate these logical modules when
callback work next needs the mechanism:

```text
Callback.Configuration
  address rules, typed uses, interpretations, and configuration projection

Callback.Model
  routing, admission, dispatch, requests, and callback traces
```

`Temporal.System.Matching.Configuration` remains a small owned adapter over the shared
configuration module and does not yet justify additional decomposition.

### 7.5 Observation and Implementation Link flow

```text
Runtime RawEvidence
        │
        ▼
Temporal.System.<Family>.Observation
        │
        ▼
Checked System Model Trace
        │
        ├──▶ System properties
        │
        ▼
Temporal.System.<Family>.ImplementationLink
        │
        ▼
Feature Model Trace
        │
        ▼
Feature properties
```

Evidence rules do not leak into properties. Runtime adapters do not claim Feature meaning directly.
An Implementation Link failure and an observation failure remain distinct.

### 7.6 Tool modules

| Module | Responsibility |
| --- | --- |
| `Temporal.Tool.Inspect` | Resolve named scenarios and render canonical artifacts or diagnostics. |
| `Temporal.Tool.Catalog` | Shared list, explain, and invariant-check engine for structural and semantic catalogs. |
| `Temporal.Tool.GenerateTests` | Select named regressions or batches and emit canonical manifests and `ExperimentSpec`s. |
| `Temporal.Tool.CheckModel` | Run model-declared verification profiles and emit verification receipts. |

Tools are semantically thin. `Inspect` already demonstrates the intended shape: a pure injectable
runner plus a small IO entry point.

### 7.7 Verification modules

```text
Temporal.Verify
└── <Family>
    ├── Native
    └── Veil
```

Native verification selects existing checked targets and properties, runs bounded checking,
replays counterexamples through canonical semantics, and returns receipts. A Veil adapter may
repeat a finite checker representation only when an explicit correspondence relates it to the
canonical family model. Neither path enters `Temporal.lean`, ordinary tools, or runtime artifacts.

## 8. Artifact seams

| Artifact | Producer | Consumer and purpose |
| --- | --- | --- |
| API catalog | API importer | Temporal Feature/System authors and catalog tooling. |
| Config catalog | Config importer | System configuration interpretation. |
| Semantic catalog | Lean Catalog | Discovery, tools, exploration, and promotion. |
| Regression/space | Lean authoring | Planning and exploration. |
| `DrivePlan` | Lean Planning | Deterministic selected semantic occurrences and checkpoints. |
| `ExperimentSpec` | Lean Artifact | Generated View, execution, replay, and verification reference. |
| `RuntimeConfiguration` | Temporal-owned profile compiler | Execution runtime operational binding. |
| `ParticipantProgram` | Temporal System model | SDK participant adapters. |
| `ExperimentRun` | Execution runtime | Run Evaluation, replay, and Claim Assessment. |
| `RawEvidence` | Runtime and adapters | Observation/Run Evaluation. |
| `SemanticEvidence` | Lean checker | Result assembly and explanation. |
| `Result` | Run Evaluation | Replay, promotion, and Claim Assessment. |
| Replay bundle | Replay | Exact spec, run, evidence, Result, Limits, provenance, and reduction state needed to reproduce a accepted failure. |
| Coverage report/checkpoint | Lean Exploration | Campaign resume, reporting, and reproducibility. |
| Verification receipt | Lean Verify | CI and Claim Assessment. |
| Evaluation Receipt | Claim Assessment | Environment and downstream policy tools. |
| Artifact-set manifest | Artifact transport | Exact closure for every persisted workflow. |

### 8.1 Complete executable intent

An executable test artifact must describe its semantically relevant behavior in totality:

- required environment capabilities and initial conditions;
- resources, setup, and semantic configuration;
- participant programs and pre-programmed behavior;
- actions with typed late-bound references such as run IDs;
- requested faults and activation points;
- ordering, concurrency, and causal constraints;
- observation instructions and property-derived requirements;
- convergence and termination conditions;
- phase-specific Limits; and
- cleanup obligations.

Operational endpoints, credentials, namespaces, granted authority, and resource limits remain
runtime bindings when they do not change semantic meaning.

### 8.2 Artifact requirements

Every persisted artifact carries an exact format version. Semantic artifacts additionally carry
Behavior Fingerprints and digests, provenance, Limits, and Known Gaps. Operational artifacts carry
environment, authority, source closure, cleanup, and failure status.

Readers reject unknown major versions, meaning-bearing unknown fields, duplicate normalized keys,
stale digests, incompatible references, incomplete sets, unsafe paths, and values exceeding
declared admission Limits. Named migrations transform complete sets deterministically and never
invent new semantic meaning for an old field.

## 9. Go modules

The older Umpire implementations are evidence and test material. Their packages do not become the
target structure automatically. Capabilities move only behind Umpire4 interfaces and artifacts.

### 9.1 Present generator and Generated View modules

| Module | Responsibility | Status and direction |
| --- | --- | --- |
| `umpire-export-proto-descriptors` | Discover Go descriptor packages, select prefixes, close transitive imports, encode deterministically, and publish atomically. | Present and deep. Keep its `Run` interface; `main` remains thin. |
| `umpire-gen-lean-api` | Merge descriptor sets, validate the declaration plan, render the generated API surface, and publish safely. | Present. Deepen toward the complete API catalog without adding product meaning. |
| `umpire-gen-lean-dynamic-config-catalog` | Discover registration, snapshot production metadata, project fixtures, render Lean, validate candidates, and publish safely. | Present. Keep generated structure separate from handwritten interpretation. |
| regression Generated View | Verify a canonical regression fixture and render deterministic Go/Markdown views. | Present for a bounded catalog. Grow through semantic Catalog, not a second registry. |

Do not extract command internals into public packages merely for symmetry. One implementation and
one caller do not justify a new seam.

### 9.2 Artifact transport

`tools/umpire/artifact` is one deep module with an interface shaped like:

```text
AdmitSet(bytes, limits) → checked artifact set
MigrateSet(set, targetVersion) → migrated set
PublishSet(root, set) → atomic immutable publication
```

It owns bounded strict JSON admission, exact version dispatch, cross-document identity and digest
validation, complete-set closure, deterministic migrations, atomic publication, and crash recovery.
It does not plan, execute, evaluate properties, interpret evidence, or repair invalid meaning.

`tools/common/artifactio` remains the lower-level filesystem implementation for safe file and set
publication.

### 9.3 Runtime and learning modules

| Module | Small interface and deep responsibility |
| --- | --- |
| `runner` | Execute one admitted test through preparation, realization, observation, cleanup, and finalization while enforcing phase Limits. |
| `participant` | Interpret a closed `ParticipantProgram` through an SDK adapter and emit bounded structured observations. |
| `runevaluation` | Pass admitted run/evidence data through the bounded Lean checker and assemble validated SemanticEvidence and Result artifacts. |
| `campaign` | Coordinate Lean-selected batches, leases, parallel execution, opaque exploration state, corpus persistence, and operational time budgets. |
| `replay` | Reproduce a evaluated violation, minimize it through checked candidates, identify its evidence core, and request a reviewed promotion proposal. |
| `verification` | Invoke model-declared native or optional checker profiles and admit provenance-rich receipts. |
| `evaluation` | Apply named environment Claim Assessment policy to admitted evidence and produce an Evaluation Receipt. |
| `generatedview` | Convert admitted Artifacts into deterministic developer Generated Views without Execution or semantic interpretation. |

These target runtime and learning modules are planned. Existing implementations in older Umpire
trees are baselines to evaluate, not integrated implementations of these interfaces.

The runner interface is:

```text
ExperimentSpec
+ RuntimeConfiguration
+ Authority adapter
+ Participant adapter
          │
          ▼
       runner.Run
          │
          ▼
ExperimentRun + RawEvidence
```

The Run Evaluation interface is:

```text
ExperimentSpec + ExperimentRun + RawEvidence
                         │
                         ▼
                 runevaluation.Check
                         │
                 Lean checker adapter
                         │
                         ▼
               SemanticEvidence + Result
```

Go validates transport, process behavior, and artifact closure. Lean performs observation
interpretation, Implementation Link, and property evaluation.

Lean `Umpire.Exploration` and Go `campaign` remain distinct. Lean decides which semantic experiment
is useful and updates semantic coverage. Go leases and executes batches concurrently and persists
opaque state. Go must not reproduce coverage scoring, mutation meaning, or selection policy.

### 9.4 Adapters

```text
tools/umpire/adapter/
├── leanchecker
├── temporaltest
├── temporalsdk
├── kitchensink
├── grpc
├── otel
├── internalhistory
├── ci
└── staging
```

- `leanchecker` invokes focused Lean executables with bounded input and output.
- `temporaltest` owns isolated ephemeral Temporal lifecycle.
- `temporalsdk` realizes the participant protocol through the Go SDK.
- `kitchensink` realizes closed pre-programmed participant behavior through a Kitchensink-style
  environment.
- `grpc` supplies public black-box actions and observations without internal server access.
- `otel` supplies authorized metrics, logs, and spans as typed raw evidence.
- `internalhistory` supplies explicitly authorized white-box history or state-machine evidence.
- `ci` supplies disposable runner provenance and isolation.
- `staging` supplies authorized remote bindings when that operational ownership is justified.

Adapters own transport and environment behavior, not semantic interpretation. A production adapter
and a controlled test adapter make each external seam real. Evidence adapters emit typed raw facts;
Temporal System Observation modules decide what those facts mean. Source-local and causal ordering
must remain valid under clock skew rather than relying on a global wall-clock order.

Canary is intentionally absent from this directory.

### 9.5 Commands

Commands remain thin compositions over modules. The intended user operations include:

- checking model-declared verification profiles;
- listing and explaining declarations;
- generating canonical test manifests and specifications;
- generating deterministic Go Generated Views;
- running bounded campaigns against selected environments;
- checking Run Evaluation;
- replaying and minimizing accepted failures; and
- assessing admitted results.

Test generation remains Lean-owned. Generated Go tests call the reusable runner directly; there is
no separate public run-tests command. Commands may eventually become subcommands of one `umpire`
executable without changing module ownership.

### 9.6 Recommended Go structure

```text
tools/umpire/
├── artifact/
├── generatedview/
├── runner/
├── participant/
├── runevaluation/
├── campaign/
├── replay/
├── verification/
├── evaluation/
├── adapter/
│   ├── leanchecker/
│   ├── temporaltest/
│   ├── temporalsdk/
│   ├── kitchensink/
│   ├── grpc/
│   ├── otel/
│   ├── internalhistory/
│   ├── ci/
│   └── staging/
└── cmd/
```

### 9.7 Public command ownership

| Command | Owner | Responsibility |
| --- | --- | --- |
| `umpire-check-model` | Go verification adapter plus Lean `Temporal.Tool.CheckModel` | Run model-declared per-commit, nightly, or named checks and assemble an honest verification receipt. |
| `umpire-gen-tests` | Lean `Temporal.Tool.GenerateTests` executable | List, explain, and compile named regressions, test sets, or selected batches into canonical JSON manifests and complete traces. |
| `umpire-gen-tests-go` | Go Generated View module | Convert admitted manifests into readable deterministic Go tests. |
| `umpire-fuzz` | Go campaign coordinator plus Lean Exploration | Run time-bounded parallel exploration with opaque resumable semantic state. |

There is no public `umpire-run-tests` command. Generated Go tests call `runner` directly. Catalog,
Run Evaluation, replay, and Claim Assessment operations may be focused commands or subcommands of a
single `umpire` executable; that packaging choice does not change module ownership. Production
canary commands belong under `tools/canary`, not under an Umpire command tree.

## 10. Standalone production canary

Production canary policy is substantial and operationally distinct from Umpire. It owns signed
approval, authorization, environment selection, leases, fencing, isolation, killable workers,
recovery, cleanup, rate and concurrency limits, audit, trusted artifact channels, and blast-radius
control. It therefore belongs in a standalone deep module.

```text
tools/umpire
├── artifact
├── runner
├── participant
├── runevaluation
└── evaluation
          ▲
          │ consumes stable interfaces
          │
tools/canary
├── approval and authorization
├── production safety policy
├── controller and killable worker
├── leases, fencing, and recovery
├── cleanup enforcement
├── rate, concurrency, and blast-radius limits
├── Temporal production adapter
├── audit and provenance
└── canary command/workflow
```

Recommended physical shape:

```text
tools/canary/
├── canary.go
├── adapter/
│   └── temporal/
├── internal/
│   ├── approval/
│   ├── controller/
│   ├── recovery/
│   └── worker/
└── cmd/
    └── canary/
```

Its external interface is:

```text
Umpire ArtifactSet
+ production profile
+ signed approval
+ production authority
          │
          ▼
      canary.Run
          │
          ▼
Canary audit/recovery record
+ Umpire Evaluation Receipt
```

Canary is independently owned and executable while consuming stable Umpire artifacts and
libraries. Umpire never imports `tools/canary` and contains no canary-specific approval, policy,
credential, recovery, or release concepts.

## 11. Dependency rules

The normative Lean import rules are owned by MOD-01, MOD-03, MOD-05, MOD-09, MOD-10, and MOD-11;
this component view applies those IDs without defining a second policy. `make lint-model` checks
their direct and transitive reachability constraints over the complete first-party module graph.

- `Shared.*` remains independent of `Umpire.*` and `Temporal.*` (MOD-09).
- `Umpire.*` remains independent of `Temporal.*` (MOD-01).
- `Temporal.Feature.*` remains isolated from `Temporal.System.*`, `Temporal.Verify.*`, and
  `Umpire.Verify.Veil` (MOD-03).
- `Temporal.System.*` remains isolated from `Temporal.Feature.*` except for the exact
  `Temporal.System.Nexus.ImplementationLink` consumer (MOD-10).
- `Temporal.Verify.*` and `Umpire.Verify.Veil` remain opt-in. Their exact aggregate, tool, and test
  consumers are `TemporalVerify`, `TemporalVeilTests`, `Temporal.Tool.VerifyVeil`, and
  `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests` (MOD-05).
- `Temporal.Tool.*` composes modules but owns no semantic authority.
- Umpire Go packages never import `tools/canary`; Canary may import stable Umpire Go packages.
- Commands remain thin adapters.

Generated `Temporal.API.*` and `Temporal.DynamicConfig.*` modules are never edited by hand.
`Temporal.Feature.*` and base `Temporal.System.*` modules remain independently understandable and
testable under MOD-04. Verification isolation is enforced by the exact MOD-05 policy above rather
than a broad module-name exception.

Old Umpire implementations remain reference material until functionality is deliberately moved
behind the new interfaces. There is no wholesale package migration and no compatibility facade by
default.

## 12. Failure model

Failures and outcomes remain phase-specific:

| Phase | Outcomes |
| --- | --- |
| Authoring | valid, invalid, unsatisfiable |
| Planning | selected, verified within complete Limits, absent, exhausted |
| Execution | realized, diverged, unsupported, infrastructure failed |
| Observation | established, missing, ambiguous, conflicting, unsupported |
| Property | satisfied, violated, unknown, conflict, unsupported |
| Verification | established, violated, unknown, unsupported, invalid |
| Claim Assessment | accepted, rejected, incomplete, unauthorized |

A successful execution never implies property satisfaction. Missing evidence never proves absence.
Budget exhaustion never proves verification. A requested action or fault never proves realization.
Checker timeout, stale binding, replay disagreement, incomplete source closure, or cleanup failure
never becomes success.

Go errors should distinguish malformed input, unsupported capability, operational failure,
infrastructure failure, and semantic Result. A property violation is an inspectable semantic result,
not an artifact decoder or subprocess error.

## 13. Testing strategy

Tests use the same interfaces as callers.

- Test authored-to-checked transitions through public facades.
- Keep internal seams private and use them only for focused implementation tests.
- Use canonical positive and negative fixtures at every artifact seam.
- Require byte-identical deterministic generation.
- Use independent semantic, property, observation, Implementation Link, and runtime mutations.
- Test missing, duplicate, stale, ambiguous, conflicting, unsupported, truncated, and oversized
  inputs.
- Test execution cancellation, source closure, evidence Limits, cleanup, and crash recovery.
- Test adapters against shared interface conformance suites.
- Run `make lint-model` for the MOD-01, MOD-03, MOD-05, MOD-09, and MOD-10 import boundaries; keep
  domain-purity and generated-ownership checks in their existing focused gates.
- Test counterexamples through canonical replay before accepting violation or promotion.
- Test canary approval, authorization, isolation, fencing, killability, recovery, cleanup, redaction,
  and audit independently of Umpire semantic tests.
- Prefer `require` over `assert` in Go tests and compare complete values rather than field-by-field
  checks where practical.

When a deep interface test supersedes tests coupled to a shallow implementation, replace rather
than layer the redundant tests. Tests should survive internal refactoring of the module.

## 14. Development priorities

Priorities follow module dependencies and proof-of-value rather than task-tracker order.

### Priority 1: deepen Lean authoring

- Extract and deepen `Umpire.Target`.
- Simplify Temporal family target declarations.
- Hide routine identity, source, provider, connector, digest, checked-result, and planner plumbing.
- Preserve semantic behavior and canonical artifacts.

Do this before adding more authoring languages. Otherwise Space, Observation, Implementation Link, and new
families will copy the current low-level interface.

### Priority 2: complete semantic foundations

- Add `Umpire.Observation` and one Temporal-owned mapping.
- Add `Umpire.ImplementationLink` and one Feature/System correspondence.
- Add `Umpire.Space` with finite choices, fault intent, and coverage goals.
- Complete structural and semantic catalogs with list/explain interfaces.
- Deepen `Umpire.Artifact` construction and versioning.

Retire the shallow Search facade as Query and Exploration take ownership. Split Callback
configuration from its mechanism only when the new observation or execution work benefits from the
seam. Split CallerClosure after the target interface removes its boilerplate.

### Priority 3: establish artifact transport

- Implement strict bounded artifact admission.
- Freeze the minimal RuntimeConfiguration, ExperimentRun, evidence, Result, coverage, verification,
  and Claim Assessment schemas.
- Add complete-set validation, deterministic migrations, atomic publication, and crash recovery.

Runtime must not grow around ad hoc structs or an unversioned JSON contract.

### Priority 4: prove the local execution and Run Evaluation loop

- Build the domain-neutral runner.
- Bind an isolated ephemeral Temporal adapter.
- Bind one Go SDK participant program.
- Capture bounded raw evidence and explicit cleanup.
- Interpret evidence through the Lean Observation and Implementation Link modules.
- Produce SemanticEvidence and a evaluated Result.
- Prove a deterministic negative control so the path demonstrates violation as well as success.

This is the highest-value vertical proof: the model checks real Temporal behavior.

### Priority 5: build the learning loop

- Add `Umpire.Exploration` and the Go campaign coordinator.
- Add semantic coverage, resume state, symmetry, and guided prioritization.
- Add deterministic replay and semantic minimization.
- Identify a non-destructive diagnostic evidence core.
- Produce reviewed promotion proposals and broader deterministic Generated Views.

Exploration success is measured by meaningful semantic coverage and retained regressions, not raw
case count.

### Priority 6: add Claim Assessment primitives

- Define generic Evaluation Profiles and receipts.
- Add local and CI-owned profiles.
- Preserve execution, evidence, property, verification, cleanup, environment, and authority status
  independently.
- Add authorized remote staging only when its operational owner and authority model are explicit.

Claim Assessment consumes admitted artifacts and acquires no authority by itself.

### Priority 7: build standalone production canary

- Create `tools/canary` as an independent module and executable.
- Consume stable Umpire artifacts, runner, Run Evaluation, and Claim Assessment interfaces.
- Own signed approval, production policy, trusted artifact acquisition, isolation, leases, fencing,
  recovery, cleanup, audit, rate limits, concurrency, and blast radius.
- Keep canary-specific types and claims out of Umpire.

### Parallel assurance track

- Add Lean-native verification receipts and canonical counterexample replay first.
- Evaluate optional Veil toolchain compatibility independently.
- Add a family-specific Veil binding only when correspondence, trust, cost, and replay requirements
  are met.

Formal assurance can proceed in parallel but must not delay the local execution and Run Evaluation
proof or enter production runtime paths.

## 15. Non-goals

- A second semantic authority in Go, JSON, generated tests, runtime adapters, or formal checker
  declarations.
- A monolithic model of all Temporal behavior.
- A universal instruction tree combining Property, Behavior, Query, Observation, and execution.
- Exact modeling of goroutine, network, storage, or distributed scheduler internals.
- Unbounded liveness claims derived from finite execution.
- Arbitrary opaque callbacks in portable declarations.
- Treating execution success, fault receipts, coverage, or metadata as correctness evidence.
- Hiding Limits, Known Gaps, truncation, evidence gaps, trust, or cleanup failure.
- Wholesale migration of Umpire2 or Umpire3 package trees.
- Compatibility facades without multiple active consumers.
- Importing optional Veil machinery into ordinary Umpire, Temporal, tools, or runtime paths.
- Putting production canary policy, credentials, authorization, recovery, or release decisions under
  `tools/umpire`.

## 16. Architectural completion criteria

The architecture is realized when:

1. A Temporal engineer with Lean basics can author a Feature regression without assembling target,
   proof, metadata, digest, checked-result, or planner plumbing.
2. The same engineer can author a System mechanism, configuration interpretation, observation
   mapping, and Feature Implementation Link through cohesive interfaces.
3. Property, Behavior, Space, Query, Observation, Implementation Link, Planning, Exploration, Artifact,
   Verification, and Claim Assessment remain independently testable modules.
4. Feature and base System are independently understandable and meet only through explicit
   Implementation Link.
5. One canonical ExperimentSpec drives model inspection, generated tests, local execution,
   campaigns, replay, Claim Assessment, and standalone canary.
6. Live evidence is accepted through Lean-owned observation and property meaning without semantic
   duplication in Go.
7. A known negative control produces a evaluated violation, deterministic replay, minimization, and
   a reviewable regression proposal.
8. Local, CI, staging, and canary results retain distinct environment authority, evidence,
   Known Gaps, and cleanup status.
9. `tools/canary` is independently owned, safety-bounded, recoverable, and downstream of stable
   Umpire interfaces.
10. Import-direction, domain-purity, artifact-closure, deterministic-output, mutation, and adapter
    Run Evaluation checks enforce the architecture mechanically.
