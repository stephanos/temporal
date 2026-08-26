# Temporal Lean model architecture

This directory contains the reusable Umpire modeling library, generated structural projections of
Temporal APIs and dynamic configuration, and handwritten Temporal-specific semantic models. This
document is the high-level map. The package-level documents describe their public APIs in detail:

- [Umpire public API](Umpire/ARCHITECTURE.md)
- [Temporal Umpire public API](Temporal/Umpire/ARCHITECTURE.md)

For generation ownership, build commands, and regression checks, see [README.md](README.md).

## Libraries and imports

The model defines three production Lean libraries:

| Import | Purpose |
| --- | --- |
| `Umpire` | Reusable, Temporal-independent semantic modeling and finite planning APIs. |
| `Temporal` | Generated Temporal schemas plus handwritten Temporal-specific interpretations and scenarios. |
| `NexusAutoClose` | Standalone Nexus AutoClose state model and proofs used by the Temporal Nexus scenario. |

Most consumers should start with an umbrella import:

```lean
import Umpire
import Temporal
```

Use focused imports when a consumer needs a smaller surface. The package-level documents identify
those facades and their entry points.

## Dependency map

```text
Umpire.Core
├── Umpire.Property
├── Umpire.Behavior
└── Umpire.Search
        │
        ▼
   Umpire.Query
        │
        ▼
  Umpire.Artifact
        │
        ▼
  Umpire.Planning

Temporal.API ─────────────────────────┐
Temporal.DynamicConfig ── Config ────┤
NexusAutoClose ────────── Nexus model ├── Temporal
Umpire ───────────────────────────────┘
```

`Umpire` does not depend on Temporal. Temporal-specific semantics are adapters built on top of the
reusable Umpire APIs.

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
Handwritten interpretations of this structure live under `Temporal.Umpire`.

## Package boundaries

- `Umpire` owns reusable semantic declarations, authoring languages, checking, planning, and
  portable artifacts.
- `Temporal.Umpire` owns Temporal-specific interpretations, target compositions, scenarios, and
  inspection.
- `NexusAutoClose` owns the standalone state model and proofs consumed by the Temporal Nexus
  adapter.
- `Temporal.API` and `Temporal.DynamicConfig` are generated structural inputs.
- `Umpire.Property.Language`, `Umpire.Behavior.Language`, `Umpire.Query.Language`, and
  `Umpire.Planning.Engine` implement public facades and should not normally be imported directly.

Artifacts are pure model products. They do not claim that Temporal was started, actions were
executed, or runtime evidence was collected.

## Reference models

- [`Umpire.Examples.Switch`](Umpire/Examples/Switch.lean) is the smallest complete example of the
  reusable API.
- [`Temporal.Umpire.NexusCallerClosure`](Temporal/Umpire/NexusCallerClosure.lean) applies the same
  lifecycle to Workflow–Nexus caller closure.
- [`NexusAutoClose`](NexusAutoClose.lean) is a tutorial-style standalone proof model.
