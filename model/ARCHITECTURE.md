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
import Temporal.Feature
import Temporal.Feature.Nexus.Examples.BasicOperations
import Temporal.System
import Temporal.System.Configuration
import Temporal.System.Callback.Configuration
import Temporal.System.Matching.Configuration
```

`Temporal.Feature.*` owns product-visible behavior, `Temporal.System.*` owns implementation
mechanisms and interpretations, and `Temporal.Tool.*` owns developer tooling. The production
`Temporal` aggregate imports generated APIs, the Feature and System facades, and the basic Nexus
examples. It deliberately does not import executable Tool code.

The import-only `TemporalModelTests` library is the Temporal test root. The
`temporal-model-inspect` executable is rooted at `Temporal.Tool.Inspect`.

## Dependency map

```text
Shared
├── Shared.Transition
└── Shared.TraceReplay

Umpire.Core ── Umpire.Target ─┬── Umpire.Property ─┐
                              └── Umpire.Observation│
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
BasicOperations ──────────────────────┘

Temporal.DynamicConfig ── Temporal.System.Configuration
                              ├── Temporal.System.Callback.Configuration
                              └── Temporal.System.Matching.Configuration

Temporal.Feature.Nexus.AutoClose ─┬── Temporal.Feature.Nexus.CallerClosure
Umpire ────────────────────────────┤
                                  └── BasicLifecycle ── BasicOperations

Temporal.Feature.Nexus.CallerClosure ─┐
Umpire.Examples.Switch ────────────────┴── Temporal.Tool.Inspect
                                              │
                                              ▼
                                  temporal-model-inspect
```

`make lint-model` checks the complete first-party module graph transitively. It keeps `Shared.*`
independent of `Umpire.*` and `Temporal.*`, keeps `Umpire.*` independent of `Temporal.*`, isolates
`Temporal.Feature.*` from `Temporal.System.*`, and protects the opt-in `Temporal.Verify.*` and
`Umpire.Verify.Veil` seams. The only cross-layer refinement consumer is the exact
`Temporal.System.Nexus.Refinement` module; verification consumers use the exact allowlist owned by
MOD-05. The normative import rules are MOD-01, MOD-03, MOD-05, MOD-09, MOD-10, and MOD-11.
Semantic ownership, deep interfaces, and independent testability remain design rules rather than
graph-linter claims.

The shared `Temporal.System.Configuration` facade also does not import its
`Temporal.System.Callback.Configuration` or `Temporal.System.Matching.Configuration` consumers.
`Temporal.Tool.*` may compose feature models with reusable examples for inspection.

## Modeling lifecycle

Public Umpire APIs follow an authored → checked → planned → artifact lifecycle:

1. Define declarations, capabilities, laws, and a finite transition kernel.
2. Call `composeTarget` to obtain a canonical `CheckedTarget`.
3. Call `checkProperty` and `checkBehavior` to validate authored constraints.
4. Call `checkQuery` to bind a target, properties, behavior, bounds, and policy.
5. Call `plan` with an `IncrementalPlannerKernel`.
6. Inspect the resulting `PlannerRun` and optional `ExperimentSpec`.

Checked types freeze canonical metadata and semantic digests. Planning accepts checked values
rather than raw author input. See the [Umpire public API](Umpire/ARCHITECTURE.md) for the types and
function signatures at each stage.

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

- `Temporal.Feature.Nexus.AutoClose` owns the authoritative Nexus operation lifecycle and its
  proofs.
- `Temporal.Feature.Nexus.Examples.BasicLifecycle` adapts the scheduled, started, and succeeded
  lifecycle states to one small Umpire target. It encapsulates target composition, finite
  completeness, deterministic bounds and policy, and the incremental planner kernel.
- `Temporal.Feature.Nexus.Examples.BasicOperations` owns the asynchronous-start and
  successful-completion walkthroughs over that shared target.
- `Temporal.Feature.Nexus.CallerClosure` is the advanced Workflow–Nexus integration reference. It
  owns caller closure, connector composition, cancellation behavior, and its checked query modes.
- `Temporal.System.Configuration` exposes shared generated-catalog classification, validation,
  resolution, provenance, and immutable views. `Temporal.System.Callback.Configuration` and
  `Temporal.System.Matching.Configuration` add consumer-specific meanings in one direction from
  that facade.
- `Temporal.Tool.Inspect` owns canonical artifact rendering, scenario lookup, CLI diagnostics, and
  the executable entry point. It does not own feature semantics.

Property, Behavior, and Query remain distinct throughout the Temporal examples. A Property states
what a semantic trace must mean. A Behavior selects allowed controllable actions and setup without
inventing model outcomes. A Query binds checked instances of both to a target, bounds, and policy
for deterministic planning. Consequently, the target—not Behavior—produces lifecycle outcomes and
observations.

## Package boundaries

- `Shared` owns domain-neutral transition systems, finite runs, observations, and trace replay.
- `Umpire` owns reusable semantic declarations, authoring languages, checking, planning, and
  portable artifacts.
- `Temporal.Feature` owns product meaning and target compositions.
- `Temporal.System` owns configuration and execution-oriented mechanisms without defining feature
  behavior.
- `Temporal.Tool` owns inspection and other developer tools without becoming part of the
  production aggregate.
- `Temporal.API` and `Temporal.DynamicConfig` remain generated structural inputs outside the
  Feature/System semantic layers.
- `Umpire.Target.Language`, `Umpire.Property.Language`, `Umpire.Behavior.Language`,
  `Umpire.Query.Language`, `Umpire.Observation.Language`, `Umpire.Observation.Qualification`, and
  `Umpire.Planning.Engine` implement public facades and should not normally be imported directly.

Artifacts are pure model products. They do not claim that Temporal was started, actions were
executed, or runtime evidence was collected.

## Tests and inspection

`TemporalModelTests.lean` contains imports only. It assembles the focused Feature examples and
caller-closure tests, the System configuration tests, and the Tool inspector tests without
importing reusable Umpire test internals. Compile the final public and test roots with:

```sh
cd model
mise exec -- lake build Shared
mise exec -- lake build Temporal
mise exec -- lake build TemporalModelTests
mise exec -- lake build temporal-model-inspect
```

The inspector registry remains intentionally small. It exposes the canonical scenario identities
`switch.query.exact-action` and `workflow-nexus.query.exact-action-caller-closure`; the basic Nexus
walkthroughs are compile-checked examples rather than registered scenarios. Successful inspection
emits one canonical JSON `ExperimentSpec`. Unknown scenarios and invalid argument counts retain
their structured non-zero diagnostics and emit no artifact JSON on standard output.

From the repository root, `make lint-model` owns the transitive Lean import boundaries described
above. `make umpire-check-regression` builds all final targets, enforces reusable domain purity and
the `Temporal.System.Configuration` consumer direction, compares deterministic artifacts with the
canonical fixtures, and checks inspector diagnostics. `make umpire-inspect SCENARIO=<identity>`
invokes the final executable without exposing its Lake target name to callers.

## Learning path and reference models

- [`Umpire.Examples.Switch`](Umpire/Examples/Switch.lean) is the smallest complete example of the
  reusable API and the first stop for learning Property, Behavior, Query, and planning.
- [`Temporal.Feature.Nexus.Examples.BasicLifecycle`](Temporal/Feature/Nexus/Examples/BasicLifecycle.lean)
  introduces the shared one-capability, one-provider Temporal Nexus target.
- [`Temporal.Feature.Nexus.Examples.BasicOperations`](Temporal/Feature/Nexus/Examples/BasicOperations.lean)
  next demonstrates asynchronous start and successful completion. Each case exposes its Property,
  exact one-action Behavior, Query, and deterministic result separately.
- [`Temporal.Feature.Nexus.CallerClosure`](Temporal/Feature/Nexus/CallerClosure.lean) is the advanced
  integration reference for Workflow–Nexus ownership, connectors, cancellation, and multiple
  planning modes.

All examples produce pure model values. They do not start Temporal, execute Nexus operations,
collect runtime evidence, or claim that a planned action occurred.
