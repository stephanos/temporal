# Temporal Lean model architecture

This directory contains the neutral Shared formal primitives, the reusable Umpire modeling
library, generated structural projections of Temporal APIs and dynamic configuration, and
handwritten Temporal-specific semantic models. This document is the high-level map. The reusable
package document describes the Umpire public API in detail:

- [Umpire public API](Umpire/ARCHITECTURE.md)

For generation ownership, build commands, and regression checks, see [README.md](README.md).

## Libraries and imports

The model defines three production Lean libraries:

| Import | Purpose |
| --- | --- |
| `Shared` | Neutral transition-system and trace-replay primitives for independent Lean models. |
| `Umpire` | Reusable, Temporal-independent semantic modeling and finite planning APIs. |
| `Temporal` | Generated Temporal schemas plus handwritten Temporal-specific interpretations and scenarios. |

Most consumers should start with an umbrella import:

```lean
import Umpire
import Temporal
```

Models that need only the neutral transition or replay vocabulary can import `Shared` or its
focused modules without depending on Umpire or Temporal:

```lean
import Shared.Transition
import Shared.TraceReplay
```

Use focused imports when a consumer needs a smaller surface. The package-level documents identify
the reusable facades. Temporal code is organized by semantic ownership:

```lean
import Umpire.ImplementationLink
import Temporal.Feature
import Temporal.Feature.Nexus
import Temporal.System
import Temporal.System.Configuration
import Temporal.System.Callback.Configuration
import Temporal.System.Matching.Configuration
import Temporal.System.Nexus.Core
import Temporal.System.Nexus.ImplementationLink
```

Detailed AutoClose and caller-closure material requires explicit experimental imports:

```lean
import Temporal.Feature.Nexus.Experimental.AutoClose
import Temporal.Feature.Nexus.Experimental.CallerClosure
```

`Temporal.Feature.*` owns product-visible behavior, `Temporal.System.*` owns implementation
mechanisms and interpretations, and `Temporal.Tool.*` owns developer tooling. The production
`Temporal` aggregate imports generated APIs plus the Feature and System facades. The Feature facade
consumes the ordinary `Temporal.Feature.Nexus` facade, which exports Lifecycle, Operations, and
Observation but no Experimental module. Consumers that need a narrower surface may import those
three stable child facades directly. The production aggregate deliberately does not import
experimental or executable Tool code.

The import-only `UmpireTests` library assembles reusable Umpire tests; only test modules may reach
the internal `Umpire.Shared.Test` fixture seam. Neither production umbrella exports that module.
No helper currently qualifies for `Shared.Test` or `Temporal.Shared.Test`, so those test-support
names remain reserved rather than becoming empty modules. `TemporalModelTests` is the ordinary
Temporal test root. The separate `TemporalExperimentalTests` library compiles the experimental
caller-closure tests and inspector tests, while the `temporal-model-inspect` executable is rooted at
`Temporal.Tool.Inspect`.

## Dependency map

```text
Shared
├── Shared.Transition
└── Shared.TraceReplay

Umpire.Core ── Umpire.Target ─┬── Umpire.Property ─┐
                              ├── Umpire.Observation│
                              └── Umpire.ImplementationLink
Umpire.Core ───── Umpire.Behavior ─────────────────┼── Umpire.Query
                                                    │        │
                                                    │        ▼
                                                    └─ Umpire.Artifact
                                                             │
                                                             ▼
                                                       Umpire.Planning

Temporal.API ─────────────────────────┐
Temporal.DynamicConfig ───────────────┤
Temporal.Feature ─────────────────────┼── Temporal
Temporal.System ──────────────────────┤
Temporal.Feature.Nexus ───────────────┘

Temporal.DynamicConfig ── Temporal.System.Configuration
                              ├── Temporal.System.Callback.Configuration
                              └── Temporal.System.Matching.Configuration

Temporal.Feature ── Temporal.Feature.Nexus
                       ├── Temporal.Feature.Nexus.Lifecycle
                       ├── Temporal.Feature.Nexus.Operations
                       └── Temporal.Feature.Nexus.Observation

Temporal.Feature.Nexus.Lifecycle ── Temporal.Feature.Nexus.Operations
                                   └── Temporal.Feature.Nexus.Observation

Temporal.System.Nexus.Core ────────┐
                                    ├── Temporal.System.Nexus.ImplementationLink
Temporal.Feature.Nexus.Lifecycle ──┘

Temporal.System.Nexus.ImplementationLink ──┐
                                             ├── Temporal.ImplementationLinkTests.Nexus
Temporal.Feature.Nexus.Operations ─────────┘

Temporal.Feature.Nexus.Experimental.AutoClose
                    │
                    ▼
Temporal.Feature.Nexus.Experimental.CallerClosure ── Temporal.Tool.Inspect
                                                             │
Umpire.Examples.Switch ───────────────────────────────────────┤
                                                             ▼
                                                 temporal-model-inspect
```

`Umpire.Target.FiniteMachine` is an implementation module inside the existing Target node, not a
new layer in this graph. It depends only on the Target surface below Query, Planning, Artifact,
Temporal, runtime, and optional verification modules. It is not a `Shared` helper.

The internal helper edges are one-way from the owning dependency to its consumers:

```text
Umpire.Core ── Umpire.Shared ─┬── Umpire.Examples.Switch
                              ├── Umpire.Shared.Test ── Umpire concern test fixtures
                              └── Temporal.Shared ── Temporal Feature facades
```

`Umpire.Shared` owns reusable construction over Umpire core types, `Umpire.Shared.Test` owns only
construction shared by Umpire concern fixtures, and `Temporal.Shared` owns Temporal-specific
construction over lower `Shared.*` and `Umpire.*` layers. `Temporal.Shared` cannot reach Feature,
System, API, Tool, verification, or test-support modules, and no production module may reach a
test-support namespace directly or transitively. These are internal implementation and test-support
seams, not replacements for the existing Umpire and Temporal consumer facades.

`make lint-model` checks the complete first-party module graph transitively. Its model-specific
`ModelLint.ImportGraph` policy uses the reusable pure `Tools.LeanImportGraph` traversal and
`Tools.LeanSourceInventory` discovery modules. It keeps `Shared.*` independent of `Umpire.*` and
`Temporal.*`, keeps `Umpire.*` independent of `Temporal.*`, isolates
`Temporal.Feature.*` from `Temporal.System.*`, and protects the opt-in `Temporal.Verify.*` and
`Umpire.Verify.Veil` seams. The only production cross-layer Implementation Link consumer is the exact
`Temporal.System.Nexus.ImplementationLink` module. The exact non-System composed-test class is
`Temporal.ImplementationLinkTests.Nexus`; its namespace is closed so sibling and prefix/suffix
near misses fail inventory classification. Verification consumers use the exact allowlist owned by
MOD-05. The normative import rules are MOD-01, MOD-03, MOD-05, MOD-09, MOD-10, and MOD-11.
Every `Umpire.Target.*` module, including Target tests, is additionally kept below Query, Planning,
Artifact, Temporal, runtime, and verification modules. Semantic ownership, deep interfaces, and
independent testability otherwise remain design rules rather than graph-linter claims.

The shared `Temporal.System.Configuration` facade also does not import its
`Temporal.System.Callback.Configuration` or `Temporal.System.Matching.Configuration` consumers.
`Temporal.Tool.*` may compose feature models with reusable examples for inspection.

## Modeling lifecycle

Public Umpire APIs follow an authored → checked → planned → artifact lifecycle:

1. A semantic-model maintainer defines declarations, capabilities, and laws. For the ordinary
   complete finite case, one proof-carrying `FiniteMachine` supplies the ordered domains,
   enumerators, encoders, coverage evidence, derived transition kernel, and exact finite planning.
   A maintainer whose authoritative propositions are intentionally independent of enumeration
   instead constructs a `TransitionKernel` directly. Either route feeds one `TargetDefinition`,
   explicit providers/connectors through a sealed `TargetComposition`, and one `AuthoredTarget`
   created with `AuthoredTarget.make`.
2. Call `checkTarget` for a source-located diagnostic or `checkedTarget` for a declaration that
   compiles as valid, obtaining one canonical `CheckedTarget`.
3. Call `checkProperty` and `checkBehavior` with contexts derived from the checked target to
   validate authored constraints.
4. Call `checkQuery` to bind that Target to Properties, Behavior, Limits, and policy.
5. Derive the planner kernel with `IncrementalPlannerKernel.ofCheckedQuery?`, then call `plan`.
6. Inspect the resulting `PlannerRun` and optional `ExperimentSpec`.

When one independently checked Target implements another, author an inert
`ImplementationLinkDeclaration` and exact forward witness, call `checkImplementationLink`, and
apply only the resulting `CheckedImplementationLink` to an Evidence-backed source Model Trace.
Application first re-admits the source trace through its checked kernel and then returns either one
complete authoritative destination trace plus coordinate-complete Evidence Links or one typed
Implementation Link diagnostic.

Property, Behavior, and Query are the scenario and question languages; Target is their common
checked substrate. Ordinary authors consume it without constructing raw providers or connectors,
finite completeness or ordering records, or planner kernels. `FiniteMachine` is typed convenience
for a family maintainer authoring that substrate, not another language; direct `TransitionKernel`
and `composeTarget` construction remain lower-level expert seams. See the
[Umpire public API](Umpire/ARCHITECTURE.md) for exact signatures.

## Generated structural APIs

`Temporal.API` is the public facade for generated protobuf structure. Its foundational types are
`Temporal.API.Proto.Bytes`, `Temporal.API.Proto.MessageRef`, and
`Temporal.API.Proto.Method Request Response`. Messages, enums, and RPC declarations retain their
protobuf-derived namespaces. They describe protocol structure only and do not implement an RPC
client or server.

`Temporal.DynamicConfig` is the public facade for the generated configuration catalog. Its
important structural types include `Setting`, `ValueSchema`, `CanonicalValue`, `ExactConstraints`,
`SettingDefault`, and `ResolutionFixture`. `Temporal.DynamicConfig.Settings.all` contains the
catalog, and `Temporal.DynamicConfig.Settings.catalogIdentity` identifies its exact contents.

Generated modules provide structure, not product meaning, and must not be edited by hand.
Handwritten interpretations live under the Feature or System package that owns their meaning.

## Temporal semantic APIs

Temporal-specific modules are split by semantic altitude:

- `Temporal.Feature.Nexus.Lifecycle` owns the ordinary scheduled, started, canceled, and succeeded
  lifecycle states; the start, cancel, and succeed transitions; and their small checked target.
- `Temporal.Feature.Nexus.Operations` owns the start, cancellation, and successful-completion
  walkthroughs over that shared target.
- `Temporal.Feature.Nexus.Observation` owns the sole synthetic BasicLifecycle evidence profile,
  its checked mapping, and the offline `EvidenceBundle` → Observation Evaluation → Property-verdict → strict-
  summary composition over the ordinary asynchronous-start Query.
- `Temporal.System.Nexus.Core` owns the independently authored pure mechanism states and transitions
  for dispatch, cancellation recording, and completion recording.
- `Temporal.System.Nexus.ImplementationLink` is the sole production leaf that imports both Nexus
  sides and proves the checked bounded forward correspondence into the unchanged Feature lifecycle.
- `Temporal.Feature.Nexus.Experimental.AutoClose` owns the detailed AutoClose configuration,
  lifecycle, reachability, history, and proofs as explicit opt-in material.
- `Temporal.Feature.Nexus.Experimental.CallerClosure` is the opt-in Workflow–Nexus integration
  reference. It owns caller closure, connector composition, cancellation behavior, and its checked
  query modes.
- `Temporal.System.Configuration` exposes shared generated-catalog classification, validation,
  resolution, provenance, and immutable views. `Temporal.System.Callback.Configuration` and
  `Temporal.System.Matching.Configuration` add consumer-specific meanings in one direction from
  that facade.
- `Temporal.Tool.Inspect` owns canonical artifact rendering, scenario lookup, CLI diagnostics, and
  the executable entry point. It does not own feature semantics.

Property, Behavior, and Query remain distinct throughout the Temporal examples. A Property states
what a Model Trace must mean. A Behavior selects allowed controllable actions and setup without
inventing model outcomes. A Query binds checked instances of both to a Target, Limits, and policy
for deterministic planning. Consequently, the target—not Behavior—produces lifecycle outcomes and
observations.

Observation is a separate offline interpretation path over that same checked semantic substrate:

```text
checked Target + one EvidenceProfileDeclaration + ObservationMappingDeclaration
  ── checkObservation ──▶ CheckedObservationPlan
checked plan + complete synthetic EvidenceBundle
  ── evaluateEvidence ──▶ Evidence-backed Model Trace | unknown | conflict | unsupported
checked Query + checked Property + Observation Evaluation
  ── evaluateObservationProperty / summarizeQueryVerdicts ──▶ verdicts + strict summary
```

The reusable `Umpire.Observation` package owns the mapping language, deterministic compilation,
bounded Observation Evaluation, coordinate-complete Evidence Links, field dispositions, semantic verdicts, and
strict aggregation. Temporal owns only the product vocabulary and its one current synthetic
profile. `Temporal.Feature.Nexus.Observation` declares that BasicLifecycle profile, retains the
state/action/outcome/observation fields, rejects its raw-detail field, applies a two-record
`evidence-records` Limit, and demonstrates a closed scheduled-to-started bundle. The resulting
offline result contains an Evidence-backed Model Trace, the independently evaluated asynchronous-start Property
verdict, and its strict Query summary.

The future runtime seam stops at `EvidenceBundle`. A future adapter would have to translate an
external source into the declared profile and version; stable record identities; evidence kinds;
typed fields and optional digest metadata; sequence and causal-parent facts; optional binding and
fault-target facts; source-closure facts; and any compatible alternatives plus their missing
discriminator. Umpire remains responsible for every check after that handoff. No live adapter is
implemented here, and this model does not execute Temporal, collect or persist live evidence,
perform Run Evaluation, promote a result, or define a second profile.

The checked cross-altitude path adds a separate stage after accepted Observation Evaluation:

```text
accepted System Evidence-backed Model Trace + CheckedImplementationLink
  ── applyImplementationLink ──▶ authoritative Feature Model Trace + Evidence Links
                              └──▶ invalid | unknown | conflict | unsupported
authoritative Feature Model Trace + CheckedProperty
  ── evaluateProperty ──▶ satisfied | violated
```

Observation, Implementation Link, and Property outcomes keep distinct identities and diagnostics.
An Observation non-success never invokes the Implementation Link; an Implementation Link
non-success never invokes Feature Property evaluation. A later Run Evaluation may orchestrate the
three stages but cannot collapse one layer's failure into another's status.

## Package boundaries

- `Shared` owns domain-neutral transition systems, finite runs, observations, and trace replay.
- `Umpire.Shared` owns internal construction over Umpire core types, while `Umpire.Shared.Test`
  owns the corresponding internal test-fixture construction and remains reachable only from tests.
- `Umpire` owns reusable semantic declarations, authoring languages, checking, planning, portable
  Artifacts, offline Observation Evaluation and verdicts, and checked Implementation Links.
- `Umpire.Target.FiniteMachine` owns ordinary complete finite-Target assembly within the Target
  boundary; it is outside `Shared`, Temporal families, runtime, and optional verification.
- `Temporal.Shared` owns internal Temporal-specific construction over lower Shared/Umpire layers;
  the existing `Temporal.Feature` modules remain the authoritative consumer-facing facades.
- `Temporal.Feature` owns product meaning, target compositions, and the sole synthetic Nexus
  Observation profile. `Temporal.Feature.Nexus` is its stable ordinary Nexus entry facade; focused
  Lifecycle and Operations children remain implementation modules behind stable family facades.
- `Temporal.System` owns configuration and execution-oriented mechanisms without defining feature
  behavior; only its focused Nexus Implementation Link leaf imports both independently authored
  sides.
- `Temporal.Tool` owns inspection and other developer tools without becoming part of the
  production aggregate.
- `Temporal.API` and `Temporal.DynamicConfig` remain generated structures outside the
  Feature/System semantic layers.
- `Umpire.Target.Language`, `Umpire.Property.Language`, `Umpire.Behavior.Language`,
  `Umpire.Query.Language`, `Umpire.Observation.Language`, `Umpire.Observation.Evaluation`,
  `Umpire.ImplementationLink.Language`, `Umpire.ImplementationLink.Application`, and
  `Umpire.Planning.Engine` implement public facades and should not normally be imported directly.

Artifacts are pure model products. They do not claim that Temporal was started, actions were
executed, or runtime evidence was collected.

## Tests and inspection

`Temporal.Feature.NexusTests` imports only the ordinary Nexus facade and smoke-checks representative
Lifecycle, Operations, and Observation declarations. `TemporalModelTests.lean` assembles that
facade check with the focused ordinary Feature and System tests, including the exact
`Temporal.ImplementationLinkTests.Nexus` composed-test root, without importing experimental modules
or reusable Umpire test internals.
`TemporalExperimentalTests.lean` separately assembles the caller-closure and Tool inspector tests.
Compile the final public and test roots with:

```sh
cd model
mise exec -- lake build Shared
mise exec -- lake build Temporal
mise exec -- lake build TemporalModelTests
mise exec -- lake build TemporalExperimentalTests
mise exec -- lake build temporal-model-inspect
```

The inspector registry remains intentionally small. It exposes the canonical scenario identities
`switch.query.exact-action` and `workflow-nexus.query.exact-action-caller-closure`; the ordinary
Nexus walkthroughs are compile-checked rather than registered scenarios, while the inspector
explicitly opts into the experimental caller-closure model. Successful inspection emits one
canonical JSON `ExperimentSpec`. Unknown scenarios and invalid argument counts retain their
structured non-zero diagnostics and emit no artifact JSON on standard output.

From the repository root, `make lint-model` owns the transitive Lean import boundaries described
above. `make umpire-check-regression` builds all final targets, enforces reusable domain purity and
the `Temporal.System.Configuration` consumer direction, compares deterministic artifacts with the
canonical fixtures, and checks inspector diagnostics. `make umpire-inspect SCENARIO=<identity>`
invokes the final executable without exposing its Lake target name to callers.

## Learning path and reference models

- Start at [`Temporal.Feature.Nexus`](Temporal/Feature/Nexus.lean), the ordinary facade and complete
  reading map. It intentionally excludes Experimental modules.
- Read [`Lifecycle.Semantics`](Temporal/Feature/Nexus/Lifecycle/Semantics.lean) first for the
  scheduled → started → canceled/succeeded states and transitions.
- Continue through the three complete operation walkthroughs in order:
  [`AsyncStart`](Temporal/Feature/Nexus/Operations/AsyncStart.lean),
  [`Cancellation`](Temporal/Feature/Nexus/Operations/Cancellation.lean), and
  [`SuccessfulCompletion`](Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean). Each file
  keeps its Property, exact one-action Behavior, Query, and deterministic result together.
- Read [`Temporal.Feature.Nexus.Observation`](Temporal/Feature/Nexus/Observation.lean) after the
  walkthroughs for the ordinary evidence-to-verdict path.
- [`Temporal.System.Nexus.Core`](Temporal/System/Nexus/Core.lean) independently describes the pure
  dispatch, cancellation-recording, and completion-recording mechanisms.
- [`Temporal.System.Nexus.ImplementationLink`](Temporal/System/Nexus/ImplementationLink.lean) then
  checks the forward correspondence from that System meaning to the unchanged Feature lifecycle;
  [`Temporal.ImplementationLinkTests.Nexus`](Temporal/ImplementationLinkTests/Nexus.lean) shows the
  separate Observation, Implementation Link, and Property outcomes.
- Contributors who need the checked Umpire machinery can then read
  [`Lifecycle.Target`](Temporal/Feature/Nexus/Lifecycle/Target.lean) and
  [`Operations.Planning`](Temporal/Feature/Nexus/Operations/Planning.lean); they are not extra steps
  in the newcomer path. [`Umpire.Examples.Switch`](Umpire/Examples/Switch.lean) is the separate
  domain-neutral reference for direct expert `TransitionKernel` → Query → Planning authoring.
- [`Temporal.Feature.Nexus.Experimental.AutoClose`](Temporal/Feature/Nexus/Experimental/AutoClose.lean)
  and [`Temporal.Feature.Nexus.Experimental.CallerClosure`](Temporal/Feature/Nexus/Experimental/CallerClosure.lean)
  are explicit opt-in references for detailed AutoClose proofs and the inspectable Workflow–Nexus
  caller-closure regression.

All modules produce pure model values. They do not start Temporal, execute Nexus operations,
collect runtime evidence, or claim that a planned action occurred.
